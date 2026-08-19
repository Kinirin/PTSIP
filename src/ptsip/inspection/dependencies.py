from __future__ import annotations

import ast
import re
import shlex
import sys
import tokenize
import tomllib
import xml.etree.ElementTree as ET
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml

from ..model import (
    DependencyEdge,
    DependencyPhase,
    EdgeType,
    EvidenceNodeScope,
    EvidenceProvenance,
    ResolutionStatus,
)
from ..repository.snapshot import repository_files

_REQUIREMENT_NAME_RE = re.compile(r"^\s*([A-Za-z0-9][A-Za-z0-9_.-]*)")
_SCRIPT_SUFFIXES = {".py", ".ps1", ".sh", ".bash", ".bat", ".cmd"}
_PYTHON_RUNNERS = {"python", "python3", "python.exe", "python3.exe", "py", "py.exe"}
_POWERSHELL_RUNNERS = {"pwsh", "pwsh.exe", "powershell", "powershell.exe"}
_SHELL_RUNNERS = {"bash", "sh"}
_CMD_RUNNERS = {"cmd", "cmd.exe"}


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


def _normalize_dependency_name(value: str) -> str:
    return re.sub(r"[-.]", "_", value.strip().lower())


def _manifest_directory(rel: str) -> str:
    parent = Path(rel).parent.as_posix()
    return "." if parent in {"", "."} else parent


def _declared_python_dependencies(
    root: Path,
    paths: list[str],
) -> tuple[dict[str, set[str]], list[DependencyScanIssue]]:
    by_directory: dict[str, set[str]] = {}
    issues: list[DependencyScanIssue] = []

    for rel in paths:
        candidate = Path(rel)
        name = candidate.name.lower()
        is_pyproject = name == "pyproject.toml"
        is_requirements = name.startswith("requirements") and candidate.suffix.lower() in {".txt", ".in"}
        if not is_pyproject and not is_requirements:
            continue

        names = by_directory.setdefault(_manifest_directory(rel), set())
        path = root / rel
        if is_pyproject:
            try:
                payload = tomllib.loads(path.read_text(encoding="utf-8-sig"))
                project = payload.get("project", {}) if isinstance(payload, dict) else {}
                if isinstance(project, dict):
                    groups: list[object] = [project.get("dependencies", [])]
                    optional = project.get("optional-dependencies", {})
                    if isinstance(optional, dict):
                        groups.extend(optional.values())
                    for group in groups:
                        if not isinstance(group, list):
                            continue
                        for item in group:
                            if not isinstance(item, str):
                                continue
                            match = _REQUIREMENT_NAME_RE.match(item)
                            if match:
                                names.add(_normalize_dependency_name(match.group(1)))
            except (OSError, UnicodeError, tomllib.TOMLDecodeError) as exc:
                issues.append(DependencyScanIssue("python-declarations", rel, str(exc)))
            continue

        try:
            lines = path.read_text(encoding="utf-8-sig").splitlines()
        except (OSError, UnicodeError) as exc:
            issues.append(DependencyScanIssue("python-declarations", rel, str(exc)))
            continue
        for line in lines:
            stripped = line.strip()
            if not stripped or stripped.startswith(("#", "-r", "--requirement", "-c", "--constraint", "-e", "--editable")):
                continue
            match = _REQUIREMENT_NAME_RE.match(stripped)
            if match:
                names.add(_normalize_dependency_name(match.group(1)))
    return by_directory, issues


def _python_dependencies_for_source(
    rel: str,
    declarations: dict[str, set[str]],
) -> set[str]:
    names: set[str] = set()
    current = Path(rel).parent
    while True:
        key = current.as_posix()
        if key in {"", "."}:
            key = "."
        names.update(declarations.get(key, set()))
        if key == ".":
            break
        current = current.parent
    return names


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


def _source_package_parts(root: Path, rel: str) -> list[str]:
    path = root / rel
    parent = path.parent
    parts: list[str] = []
    while parent != root and (parent / "__init__.py").is_file():
        parts.insert(0, parent.name)
        parent = parent.parent
    return parts


def _resolve_relative_import(root: Path, rel: str, node: ast.ImportFrom) -> tuple[str, str | None]:
    package = _source_package_parts(root, rel)
    if not package or node.level <= 0 or node.level > len(package):
        target = ("." * node.level) + (node.module or "")
        return target or "<relative>", None

    keep = len(package) - node.level + 1
    base_parts = package[:keep]
    if node.module:
        base_parts.extend(node.module.split("."))
        target = ".".join(base_parts)
        return target, _resolve_module(root, target)

    for alias in node.names:
        if alias.name == "*":
            continue
        target = ".".join([*base_parts, *alias.name.split(".")])
        resolved = _resolve_module(root, target)
        if resolved:
            return target, resolved

    target = ".".join(base_parts)
    return target, _resolve_module(root, target)


def _python_target_state(
    module: str,
    resolved: str | None,
    declared_dependencies: set[str],
) -> tuple[ResolutionStatus, EvidenceNodeScope, str | None]:
    if resolved:
        return ResolutionStatus.RESOLVED, EvidenceNodeScope.PROJECT_COMPONENT, None
    root_name = module.lstrip(".").split(".", 1)[0]
    if root_name and root_name in sys.stdlib_module_names:
        return ResolutionStatus.EXTERNAL, EvidenceNodeScope.PLATFORM, "Python standard-library target"
    if root_name and _normalize_dependency_name(root_name) in declared_dependencies:
        return ResolutionStatus.EXTERNAL, EvidenceNodeScope.EXTERNAL_DEPENDENCY, "Target matches a declared Python dependency"
    return ResolutionStatus.UNRESOLVED, EvidenceNodeScope.UNRESOLVED_TARGET, None


def _python_edges(
    root: Path,
    rel: str,
    declared_dependencies: set[str],
) -> tuple[list[DependencyEdge], list[DependencyScanIssue]]:
    path = root / rel
    try:
        with tokenize.open(path) as handle:
            source = handle.read()
        tree = ast.parse(source, filename=rel)
    except (OSError, UnicodeError, SyntaxError) as exc:
        return [], [DependencyScanIssue("python", rel, str(exc))]

    edges: list[DependencyEdge] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                resolved = _resolve_module(root, alias.name)
                resolution, scope, note = _python_target_state(alias.name, resolved, declared_dependencies)
                edges.append(
                    DependencyEdge(
                        evidence_id=f"python:{rel}:{node.lineno}:{alias.name}",
                        source=rel,
                        target=alias.name,
                        edge_type=EdgeType.IMPORTS,
                        phase=DependencyPhase.UNKNOWN,
                        resolution=resolution,
                        target_scope=scope,
                        provenance=EvidenceProvenance.OBSERVED,
                        line=node.lineno,
                        resolved_path=resolved,
                        adapter="python",
                        note=note,
                    )
                )
        elif isinstance(node, ast.ImportFrom):
            if node.level:
                target, resolved = _resolve_relative_import(root, rel, node)
            else:
                target = node.module or "<unknown-import>"
                resolved = _resolve_module(root, node.module or "")
            resolution, scope, note = _python_target_state(target, resolved, declared_dependencies)
            if node.level and not resolved and note is None:
                note = "Relative import target could not be resolved from repository package evidence"
            edges.append(
                DependencyEdge(
                    evidence_id=f"python:{rel}:{node.lineno}:{target}",
                    source=rel,
                    target=target,
                    edge_type=EdgeType.IMPORTS,
                    phase=DependencyPhase.UNKNOWN,
                    resolution=resolution,
                    target_scope=scope,
                    provenance=EvidenceProvenance.OBSERVED,
                    line=node.lineno,
                    resolved_path=resolved,
                    adapter="python",
                    note=note,
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
                target = module_value.value
                resolved = _resolve_module(root, target)
                resolution, scope, note = _python_target_state(target, resolved, declared_dependencies)
            else:
                resolved = None
                resolution = ResolutionStatus.DYNAMIC
                scope = EvidenceNodeScope.UNRESOLVED_TARGET
                target = "<dynamic-import>"
                note = "Dynamic import target is not statically known"
            edges.append(
                DependencyEdge(
                    evidence_id=f"python-dynamic:{rel}:{node.lineno}",
                    source=rel,
                    target=target,
                    edge_type=EdgeType.LOADS,
                    phase=DependencyPhase.UNKNOWN,
                    resolution=resolution,
                    target_scope=scope,
                    provenance=EvidenceProvenance.OBSERVED,
                    line=node.lineno,
                    resolved_path=resolved,
                    adapter="python",
                    note=note,
                )
            )
    return edges, []


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
                target_scope=EvidenceNodeScope.PROJECT_COMPONENT if resolved else EvidenceNodeScope.UNRESOLVED_TARGET,
                provenance=EvidenceProvenance.DECLARED,
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


def _working_directory(payload: dict[str, object], job: dict[str, object], step: dict[str, object]) -> str:
    step_value = step.get("working-directory")
    if isinstance(step_value, str) and step_value.strip():
        return step_value.strip()
    for owner in (job, payload):
        defaults = owner.get("defaults")
        if not isinstance(defaults, dict):
            continue
        run_defaults = defaults.get("run")
        if not isinstance(run_defaults, dict):
            continue
        value = run_defaults.get("working-directory")
        if isinstance(value, str) and value.strip():
            return value.strip()
    return "."


def _clean_shell_token(value: str) -> str:
    return value.strip().strip("'\"")


def _is_script_token(value: str) -> bool:
    return Path(_clean_shell_token(value)).suffix.lower() in _SCRIPT_SUFFIXES


def _script_targets_from_command(command: str) -> list[str]:
    targets: list[str] = []
    for line in command.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        for segment in re.split(r"\s*(?:&&|\|\||;)\s*", stripped):
            if not segment:
                continue
            try:
                tokens = [_clean_shell_token(item) for item in shlex.split(segment, posix=False)]
            except ValueError:
                tokens = [_clean_shell_token(item) for item in segment.split()]
            tokens = [item for item in tokens if item]
            while tokens and "=" in tokens[0] and not _is_script_token(tokens[0]):
                tokens.pop(0)
            if not tokens:
                continue

            first = tokens[0]
            executable = Path(first).name.lower()
            candidate: str | None = None

            if _is_script_token(first):
                candidate = first
            elif executable in _PYTHON_RUNNERS:
                for token in tokens[1:]:
                    if token in {"-c", "-m"}:
                        break
                    if token.startswith("-"):
                        continue
                    if Path(token).suffix.lower() == ".py":
                        candidate = token
                    break
            elif executable in _POWERSHELL_RUNNERS:
                for index, token in enumerate(tokens[1:], start=1):
                    if token.lower() in {"-file", "/file"} and index + 1 < len(tokens):
                        value = tokens[index + 1]
                        if Path(value).suffix.lower() == ".ps1":
                            candidate = value
                        break
                    if Path(token).suffix.lower() == ".ps1":
                        candidate = token
                        break
            elif executable in _SHELL_RUNNERS:
                for token in tokens[1:]:
                    if token.startswith("-"):
                        continue
                    if Path(token).suffix.lower() in {".sh", ".bash"}:
                        candidate = token
                    break
            elif executable in _CMD_RUNNERS:
                for token in tokens[1:]:
                    if token.lower() in {"/c", "/k"}:
                        continue
                    if Path(token).suffix.lower() in {".bat", ".cmd"}:
                        candidate = token
                    break

            if candidate:
                normalized = candidate.replace("\\", "/")
                if normalized not in targets:
                    targets.append(normalized)
    return targets


def _resolve_workflow_script(root: Path, working_directory: str, raw_target: str) -> str | None:
    target_path = Path(raw_target)
    if target_path.is_absolute():
        return None
    cwd = (root / working_directory).resolve()
    try:
        if not cwd.is_relative_to(root):
            return None
    except (OSError, ValueError):
        return None
    candidate = (cwd / target_path).resolve()
    try:
        return candidate.relative_to(root).as_posix() if candidate.is_file() and candidate.is_relative_to(root) else None
    except (OSError, ValueError):
        return None


def _workflow_edges(root: Path, rel: str) -> tuple[list[DependencyEdge], list[DependencyScanIssue]]:
    path = root / rel
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig")) or {}
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
            working_directory = _working_directory(payload, job, step)
            for raw_target in _script_targets_from_command(command):
                if raw_target.startswith("/"):
                    continue
                resolved = _resolve_workflow_script(root, working_directory, raw_target)
                looks_explicitly_local = raw_target.startswith(("./", "../")) or "/" in raw_target
                if not resolved and not looks_explicitly_local:
                    continue
                edges.append(
                    DependencyEdge(
                        evidence_id=f"github-actions:{rel}:{job_name}:{index}:{raw_target}",
                        source=rel,
                        target=raw_target,
                        edge_type=EdgeType.INVOKES,
                        phase=_workflow_phase(rel, str(job_name)),
                        resolution=ResolutionStatus.RESOLVED if resolved else ResolutionStatus.UNRESOLVED,
                        target_scope=EvidenceNodeScope.PROJECT_COMPONENT if resolved else EvidenceNodeScope.UNRESOLVED_TARGET,
                        provenance=EvidenceProvenance.DECLARED,
                        resolved_path=resolved,
                        adapter="github-actions",
                        working_directory=working_directory,
                        note=None if resolved else "Explicit local-script invocation could not be resolved from its effective working-directory",
                    )
                )
    return edges, []


def scan_dependency_edges(root: str | Path) -> DependencyScan:
    root = Path(root).resolve()
    _mode, paths, discovery_errors = repository_files(root)
    edges: list[DependencyEdge] = []
    issues = [DependencyScanIssue("repository", "<repository>", item) for item in discovery_errors]
    adapters: set[str] = set()
    declared_python_dependencies, declaration_issues = _declared_python_dependencies(root, paths)
    issues.extend(declaration_issues)
    for rel in paths:
        suffix = Path(rel).suffix.lower()
        if suffix == ".py":
            found, found_issues = _python_edges(
                root,
                rel,
                _python_dependencies_for_source(rel, declared_python_dependencies),
            )
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
