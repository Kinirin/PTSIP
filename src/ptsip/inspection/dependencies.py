from __future__ import annotations

import ast
import re
import xml.etree.ElementTree as ET
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml

from ..model import DependencyEdge, DependencyPhase, EdgeType, ResolutionStatus
from ..repository.snapshot import repository_files

_SCRIPT_RE = re.compile(r"(?P<path>[A-Za-z0-9_./\\-]+\.(?:py|ps1|sh|bash|bat|cmd))")


@dataclass(frozen=True)
class DependencyScanIssue:
    adapter: str
    path: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class DependencyScan:
    edges: tuple[DependencyEdge, ...]
    issues: tuple[DependencyScanIssue, ...]
    adapters: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "edges": [edge.as_dict() for edge in self.edges],
            "issues": [issue.as_dict() for issue in self.issues],
            "adapters": list(self.adapters),
            "edge_count": len(self.edges),
            "coverage_complete": not self.issues,
        }


def _resolve_module(root: Path, module: str) -> str | None:
    if not module:
        return None
    rel = Path(*module.split("."))
    candidates = [
        root / rel.with_suffix(".py"),
        root / rel / "__init__.py",
        root / "src" / rel.with_suffix(".py"),
        root / "src" / rel / "__init__.py",
    ]
    for candidate in candidates:
        if candidate.is_file():
            return candidate.relative_to(root).as_posix()
    return None


def _python_edges(root: Path, rel: str) -> tuple[list[DependencyEdge], list[DependencyScanIssue]]:
    path = root / rel
    issues: list[DependencyScanIssue] = []
    try:
        source = path.read_text(encoding="utf-8", errors="strict")
        tree = ast.parse(source, filename=rel)
    except (OSError, UnicodeError, SyntaxError) as exc:
        return [], [DependencyScanIssue("python", rel, str(exc))]

    edges: list[DependencyEdge] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                resolved = _resolve_module(root, alias.name)
                edges.append(
                    DependencyEdge(
                        evidence_id=f"python:{rel}:{node.lineno}:{alias.name}",
                        source=rel,
                        target=alias.name,
                        edge_type=EdgeType.IMPORTS,
                        phase=DependencyPhase.UNKNOWN,
                        resolution=ResolutionStatus.RESOLVED if resolved else ResolutionStatus.UNRESOLVED,
                        line=node.lineno,
                        resolved_path=resolved,
                        adapter="python",
                    )
                )
        elif isinstance(node, ast.ImportFrom):
            module = node.module or ""
            target = ("." * node.level) + module
            resolved = _resolve_module(root, module) if node.level == 0 else None
            edges.append(
                DependencyEdge(
                    evidence_id=f"python:{rel}:{node.lineno}:{target or '<relative>'}",
                    source=rel,
                    target=target or "<relative>",
                    edge_type=EdgeType.IMPORTS,
                    phase=DependencyPhase.UNKNOWN,
                    resolution=ResolutionStatus.RESOLVED if resolved else ResolutionStatus.UNRESOLVED,
                    line=node.lineno,
                    resolved_path=resolved,
                    adapter="python",
                    note="Relative imports are preserved as unresolved evidence" if node.level else None,
                )
            )
        elif isinstance(node, ast.Call):
            is_importlib = (
                isinstance(node.func, ast.Attribute)
                and isinstance(node.func.value, ast.Name)
                and node.func.value.id == "importlib"
                and node.func.attr == "import_module"
            )
            is_builtin = isinstance(node.func, ast.Name) and node.func.id == "__import__"
            if not (is_importlib or is_builtin):
                continue
            module_value = node.args[0] if node.args else None
            if isinstance(module_value, ast.Constant) and isinstance(module_value.value, str):
                module = module_value.value
                resolved = _resolve_module(root, module)
                resolution = ResolutionStatus.RESOLVED if resolved else ResolutionStatus.UNRESOLVED
                target = module
            else:
                resolved = None
                resolution = ResolutionStatus.DYNAMIC
                target = "<dynamic-import>"
            edges.append(
                DependencyEdge(
                    evidence_id=f"python-dynamic:{rel}:{node.lineno}",
                    source=rel,
                    target=target,
                    edge_type=EdgeType.IMPORTS,
                    phase=DependencyPhase.UNKNOWN,
                    resolution=resolution,
                    line=node.lineno,
                    resolved_path=resolved,
                    adapter="python",
                )
            )
    return edges, issues


def _csproj_edges(root: Path, rel: str) -> tuple[list[DependencyEdge], list[DependencyScanIssue]]:
    path = root / rel
    try:
        tree = ET.parse(path)
    except (OSError, ET.ParseError) as exc:
        return [], [DependencyScanIssue("dotnet-csproj", rel, str(exc))]
    edges: list[DependencyEdge] = []
    for element in tree.iter():
        if not element.tag.endswith("ProjectReference"):
            continue
        include = element.attrib.get("Include")
        if not include:
            continue
        candidate = (path.parent / include).resolve()
        try:
            resolved = candidate.relative_to(root).as_posix() if candidate.is_file() and candidate.is_relative_to(root) else None
        except (OSError, ValueError):
            resolved = None
        edges.append(
            DependencyEdge(
                evidence_id=f"dotnet-project-reference:{rel}:{include}",
                source=rel,
                target=include.replace("\\", "/"),
                edge_type=EdgeType.LINKS,
                phase=DependencyPhase.UNKNOWN,
                resolution=ResolutionStatus.RESOLVED if resolved else ResolutionStatus.UNRESOLVED,
                resolved_path=resolved,
                adapter="dotnet-csproj",
            )
        )
    return edges, []


def _workflow_phase(rel: str, job_name: str) -> DependencyPhase:
    text = f"{rel} {job_name}".lower()
    if "release" in text or "publish" in text or "deploy" in text:
        return DependencyPhase.RELEASE
    if "test" in text:
        return DependencyPhase.TEST
    return DependencyPhase.BUILD


def _workflow_edges(root: Path, rel: str) -> tuple[list[DependencyEdge], list[DependencyScanIssue]]:
    path = root / rel
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    except (OSError, UnicodeError, yaml.YAMLError) as exc:
        return [], [DependencyScanIssue("github-actions", rel, str(exc))]
    if not isinstance(payload, dict):
        return [], [DependencyScanIssue("github-actions", rel, "Workflow root is not a mapping")]
    jobs = payload.get("jobs", {})
    if not isinstance(jobs, dict):
        return [], [DependencyScanIssue("github-actions", rel, "Workflow jobs is not a mapping")]
    edges: list[DependencyEdge] = []
    for job_name, job in jobs.items():
        if not isinstance(job, dict):
            continue
        steps = job.get("steps", [])
        if not isinstance(steps, list):
            continue
        for index, step in enumerate(steps):
            if not isinstance(step, dict) or not isinstance(step.get("run"), str):
                continue
            command = step["run"]
            matches = list(_SCRIPT_RE.finditer(command))
            if not matches:
                edges.append(
                    DependencyEdge(
                        evidence_id=f"github-actions:{rel}:{job_name}:{index}",
                        source=rel,
                        target=command.strip().splitlines()[0][:240],
                        edge_type=EdgeType.INVOKES,
                        phase=_workflow_phase(rel, str(job_name)),
                        resolution=ResolutionStatus.UNRESOLVED,
                        adapter="github-actions",
                        note="Raw run command retained because no local script path was resolved",
                    )
                )
                continue
            for match in matches:
                raw_target = match.group("path").replace("\\", "/")
                candidate = (root / raw_target).resolve()
                try:
                    resolved = candidate.relative_to(root).as_posix() if candidate.is_file() and candidate.is_relative_to(root) else None
                except (OSError, ValueError):
                    resolved = None
                edges.append(
                    DependencyEdge(
                        evidence_id=f"github-actions:{rel}:{job_name}:{index}:{raw_target}",
                        source=rel,
                        target=raw_target,
                        edge_type=EdgeType.INVOKES,
                        phase=_workflow_phase(rel, str(job_name)),
                        resolution=ResolutionStatus.RESOLVED if resolved else ResolutionStatus.UNRESOLVED,
                        resolved_path=resolved,
                        adapter="github-actions",
                    )
                )
    return edges, []


def scan_dependency_edges(root: str | Path) -> DependencyScan:
    root = Path(root).resolve()
    _mode, paths, discovery_errors = repository_files(root)
    edges: list[DependencyEdge] = []
    issues = [DependencyScanIssue("repository", "<repository>", item) for item in discovery_errors]
    adapters: set[str] = set()
    for rel in paths:
        suffix = Path(rel).suffix.lower()
        if suffix == ".py":
            found, found_issues = _python_edges(root, rel)
            adapters.add("python")
        elif suffix == ".csproj":
            found, found_issues = _csproj_edges(root, rel)
            adapters.add("dotnet-csproj")
        elif rel.startswith(".github/workflows/") and suffix in {".yaml", ".yml"}:
            found, found_issues = _workflow_edges(root, rel)
            adapters.add("github-actions")
        else:
            continue
        edges.extend(found)
        issues.extend(found_issues)
    edges.sort(key=lambda item: (item.source, item.line or 0, item.target, item.evidence_id))
    return DependencyScan(tuple(edges), tuple(issues), tuple(sorted(adapters)))
