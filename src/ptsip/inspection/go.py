from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

from ..model import (
    DependencyEdge,
    DependencyPhase,
    EdgeType,
    EvidenceNodeScope,
    EvidenceProvenance,
    ResolutionStatus,
)
from .lexing import code_positions, comments_removed, keyword_is_code


_MODULE_RE = re.compile(r"^\s*module\s+(\S+)\s*$")
_SINGLE_IMPORT_RE = re.compile(r"^\s*import\s+(?:[._A-Za-z][\w.]*\s+)?([\"`])([^\"`]+)\1")
_BLOCK_IMPORT_RE = re.compile(r"^\s*(?:[._A-Za-z][\w.]*\s+)?([\"`])([^\"`]+)\1")


@dataclass(frozen=True)
class GoModule:
    manifest_path: str
    directory: str
    module_path: str


def discover_go_modules(root: Path, paths: list[str]) -> tuple[list[GoModule], list[tuple[str, str]]]:
    modules: list[GoModule] = []
    issues: list[tuple[str, str]] = []
    seen_names: set[str] = set()
    for rel in paths:
        if Path(rel).name != "go.mod":
            continue
        try:
            lines = (root / rel).read_text(encoding="utf-8-sig").splitlines()
        except (OSError, UnicodeError) as exc:
            issues.append((rel, str(exc)))
            continue
        module_path: str | None = None
        for line in lines:
            match = _MODULE_RE.match(line)
            if match:
                module_path = match.group(1)
                break
        if not module_path:
            issues.append((rel, "go.mod does not declare a module path"))
            continue
        if module_path in seen_names:
            issues.append((rel, f"Duplicate Go module path {module_path!r}; local target identity is ambiguous"))
            continue
        seen_names.add(module_path)
        directory = Path(rel).parent.as_posix()
        if directory == ".":
            directory = ""
        modules.append(GoModule(rel, directory, module_path))
    modules.sort(key=lambda item: (-len(item.module_path), item.manifest_path))
    return modules, issues


def _imports(source: str) -> list[tuple[int, str]]:
    found: list[tuple[int, str]] = []
    in_block = False
    mask = code_positions(source, backtick_strings=True)
    sanitized = comments_removed(source, backtick_strings=True)
    offset = 0
    for line_no, raw in enumerate(sanitized.splitlines(keepends=True), start=1):
        line = raw.strip()
        if not line:
            offset += len(raw)
            continue
        if not in_block:
            match = _SINGLE_IMPORT_RE.match(line)
            leading = len(raw) - len(raw.lstrip())
            if match and keyword_is_code(source, mask, offset + leading, offset + len(raw), "import"):
                found.append((line_no, match.group(2)))
                offset += len(raw)
                continue
            if re.match(r"^import\s*\(\s*$", line) and keyword_is_code(source, mask, offset + leading, offset + len(raw), "import"):
                in_block = True
                offset += len(raw)
                continue
        else:
            if line.startswith(")"):
                in_block = False
                offset += len(raw)
                continue
            match = _BLOCK_IMPORT_RE.match(line)
            if match:
                found.append((line_no, match.group(2)))
        offset += len(raw)
    return found


def _representative_go_file(root: Path, directory: Path) -> str | None:
    try:
        candidates = sorted(path for path in directory.glob("*.go") if path.is_file())
    except OSError:
        return None
    for candidate in candidates:
        try:
            return candidate.relative_to(root).as_posix()
        except ValueError:
            continue
    return None


def _target_state(root: Path, import_path: str, modules: list[GoModule]) -> tuple[ResolutionStatus, EvidenceNodeScope, str | None, str | None]:
    for module in modules:
        if import_path != module.module_path and not import_path.startswith(module.module_path + "/"):
            continue
        suffix = import_path[len(module.module_path) :].lstrip("/")
        base = root / module.directory if module.directory else root
        target_dir = (base / suffix).resolve() if suffix else base.resolve()
        try:
            inside = target_dir.is_relative_to(root)
        except (OSError, ValueError):
            inside = False
        resolved = _representative_go_file(root, target_dir) if inside and target_dir.is_dir() else None
        if resolved:
            return ResolutionStatus.RESOLVED, EvidenceNodeScope.PROJECT_COMPONENT, resolved, "Import resolves within a repository Go module"
        return ResolutionStatus.UNRESOLVED, EvidenceNodeScope.UNRESOLVED_TARGET, None, "Import matches a repository Go module but its package directory could not be resolved"

    first = import_path.split("/", 1)[0]
    if "." not in first:
        return ResolutionStatus.EXTERNAL, EvidenceNodeScope.PLATFORM, None, "Go standard-library import path"
    return ResolutionStatus.EXTERNAL, EvidenceNodeScope.EXTERNAL_DEPENDENCY, None, "Import is outside repository Go module paths"


def source_edges(root: Path, rel: str, modules: list[GoModule]) -> tuple[list[DependencyEdge], list[str]]:
    try:
        source = (root / rel).read_text(encoding="utf-8-sig")
    except (OSError, UnicodeError) as exc:
        return [], [str(exc)]
    edges: list[DependencyEdge] = []
    phase = DependencyPhase.TEST if rel.endswith("_test.go") else DependencyPhase.RUNTIME
    for line, import_path in _imports(source):
        resolution, scope, resolved, note = _target_state(root, import_path, modules)
        edges.append(
            DependencyEdge(
                evidence_id=f"go:{rel}:{line}:{import_path}",
                source=rel,
                target=import_path,
                edge_type=EdgeType.IMPORTS,
                phase=phase,
                resolution=resolution,
                target_scope=scope,
                provenance=EvidenceProvenance.OBSERVED,
                line=line,
                resolved_path=resolved,
                adapter="go-source",
                note=note,
            )
        )
    return edges, []
