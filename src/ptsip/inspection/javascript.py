from __future__ import annotations

import json
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
from .lexing import code_positions, keyword_is_code
from .source_adapters import JAVASCRIPT_TYPESCRIPT_SOURCE_SUFFIXES


_JS_EXTENSIONS = tuple(sorted(JAVASCRIPT_TYPESCRIPT_SOURCE_SUFFIXES))
_STATIC_IMPORT_RE = re.compile(
    r"(?:^|[;\n])\s*(?:import\s+(?:[^;\n]*?\s+from\s+)?|export\s+[^;\n]*?\s+from\s+)[\"']([^\"']+)[\"']",
    re.MULTILINE,
)
_SIDE_EFFECT_IMPORT_RE = re.compile(r"(?:^|[;\n])\s*import\s*[\"']([^\"']+)[\"']", re.MULTILINE)
_REQUIRE_LITERAL_RE = re.compile(r"\brequire\s*\(\s*[\"']([^\"']+)[\"']\s*\)")
_DYNAMIC_IMPORT_LITERAL_RE = re.compile(r"\bimport\s*\(\s*[\"']([^\"']+)[\"']\s*\)")
_REQUIRE_CALL_RE = re.compile(r"\brequire\s*\(")
_DYNAMIC_IMPORT_CALL_RE = re.compile(r"\bimport\s*\(")


@dataclass(frozen=True)
class NpmPackage:
    path: str
    directory: str
    name: str | None
    sections: dict[str, dict[str, str]]


def _line_number(source: str, offset: int) -> int:
    return source.count("\n", 0, offset) + 1


def _skip_quoted(source: str, index: int, quote: str) -> int:
    index += 1
    while index < len(source):
        char = source[index]
        if char == "\\":
            index += 2
            continue
        index += 1
        if char == quote:
            break
    return index


def _scan_template_expression(source: str, start: int, ranges: list[tuple[int, int]]) -> int:
    depth = 1
    index = start
    while index < len(source):
        char = source[index]
        following = source[index + 1] if index + 1 < len(source) else ""
        if char == "/" and following == "/":
            newline = source.find("\n", index + 2)
            index = len(source) if newline < 0 else newline + 1
            continue
        if char == "/" and following == "*":
            close = source.find("*/", index + 2)
            index = len(source) if close < 0 else close + 2
            continue
        if char in {"'", '"'}:
            index = _skip_quoted(source, index, char)
            continue
        if char == "`":
            index = _scan_template(source, index, ranges)
            continue
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return index
        index += 1
    return len(source)


def _scan_template(source: str, start: int, ranges: list[tuple[int, int]]) -> int:
    index = start + 1
    while index < len(source):
        char = source[index]
        following = source[index + 1] if index + 1 < len(source) else ""
        if char == "\\":
            index += 2
            continue
        if char == "`":
            return index + 1
        if char == "$" and following == "{":
            expression_start = index + 2
            expression_end = _scan_template_expression(source, expression_start, ranges)
            ranges.append((expression_start, expression_end))
            index = expression_end + 1 if expression_end < len(source) else len(source)
            continue
        index += 1
    return len(source)


def _template_expression_ranges(source: str) -> tuple[tuple[int, int], ...]:
    ranges: list[tuple[int, int]] = []
    base_mask = code_positions(source, backtick_strings=False)
    index = 0
    while index < len(source):
        if source[index] == "`" and base_mask[index]:
            index = _scan_template(source, index, ranges)
        else:
            index += 1
    return tuple(sorted(set(ranges)))


def _javascript_code_positions(source: str) -> tuple[bool, ...]:
    # Backtick template text is string data, but `${ ... }` is executable
    # JavaScript/TypeScript. Start with template literals masked and restore only
    # their interpolation expression code so dependency calls cannot disappear.
    mask = list(code_positions(source, backtick_strings=True))
    for start, end in _template_expression_ranges(source):
        local = code_positions(source[start:end], backtick_strings=True)
        for offset, is_code in enumerate(local):
            if is_code:
                mask[start + offset] = True
    return tuple(mask)


def _read_package(path: Path, rel: str) -> tuple[NpmPackage | None, str | None]:
    try:
        payload = json.loads(path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        return None, str(exc)
    if not isinstance(payload, dict):
        return None, "package.json root is not an object"
    sections: dict[str, dict[str, str]] = {}
    for key in ("dependencies", "devDependencies", "peerDependencies", "optionalDependencies"):
        raw = payload.get(key, {})
        if isinstance(raw, dict):
            sections[key] = {str(name): str(value) for name, value in raw.items() if isinstance(name, str)}
        else:
            sections[key] = {}
    name = payload.get("name") if isinstance(payload.get("name"), str) else None
    return NpmPackage(rel, Path(rel).parent.as_posix(), name, sections), None


def discover_npm_packages(root: Path, paths: list[str]) -> tuple[dict[str, NpmPackage], dict[str, NpmPackage], list[tuple[str, str]]]:
    by_path: dict[str, NpmPackage] = {}
    by_name: dict[str, NpmPackage] = {}
    issues: list[tuple[str, str]] = []
    for rel in paths:
        if Path(rel).name != "package.json":
            continue
        package, issue = _read_package(root / rel, rel)
        if issue is not None or package is None:
            issues.append((rel, issue or "Unable to parse package.json"))
            continue
        by_path[rel] = package
        if package.name:
            if package.name in by_name:
                issues.append((rel, f"Duplicate local npm package name {package.name!r}; package identity is ambiguous"))
            else:
                by_name[package.name] = package
    return by_path, by_name, issues


def _nearest_package(rel: str, by_path: dict[str, NpmPackage]) -> NpmPackage | None:
    parent = Path(rel).parent
    while True:
        candidate = (parent / "package.json").as_posix()
        if candidate == "./package.json":
            candidate = "package.json"
        if candidate in by_path:
            return by_path[candidate]
        if parent == Path(".") or str(parent) in {"", "."}:
            break
        parent = parent.parent
    return by_path.get("package.json")


def _package_section(package: NpmPackage | None, name: str) -> str | None:
    if package is None:
        return None
    for section, values in package.sections.items():
        if name in values:
            return section
    return None


def _bare_package_name(specifier: str) -> str:
    if specifier.startswith("@"):
        parts = specifier.split("/")
        return "/".join(parts[:2]) if len(parts) >= 2 else specifier
    return specifier.split("/", 1)[0]


def _resolve_relative(root: Path, source_rel: str, specifier: str) -> str | None:
    base = (root / source_rel).parent
    target = (base / specifier).resolve()
    try:
        if not target.is_relative_to(root):
            return None
    except (OSError, ValueError):
        return None

    candidates: list[Path] = []
    if target.suffix.lower() in _JS_EXTENSIONS and target.is_file():
        candidates.append(target)
    elif target.is_file():
        candidates.append(target)
    else:
        candidates.extend(Path(str(target) + extension) for extension in _JS_EXTENSIONS)
        candidates.extend(target / f"index{extension}" for extension in _JS_EXTENSIONS)
    for candidate in candidates:
        if candidate.is_file():
            try:
                return candidate.relative_to(root).as_posix()
            except ValueError:
                return None
    return None


def _source_phase(rel: str) -> DependencyPhase:
    text = rel.lower()
    if any(token in text for token in ("/test/", "/tests/", "/__tests__/", ".test.", ".spec.")):
        return DependencyPhase.TEST
    return DependencyPhase.UNKNOWN


def _target_state(
    root: Path,
    source_rel: str,
    specifier: str,
    by_name: dict[str, NpmPackage],
    nearest: NpmPackage | None,
) -> tuple[ResolutionStatus, EvidenceNodeScope, str | None, str | None, DependencyPhase]:
    phase = _source_phase(source_rel)
    if specifier.startswith(("./", "../")):
        resolved = _resolve_relative(root, source_rel, specifier)
        if resolved:
            return ResolutionStatus.RESOLVED, EvidenceNodeScope.PROJECT_COMPONENT, resolved, None, phase
        return ResolutionStatus.UNRESOLVED, EvidenceNodeScope.UNRESOLVED_TARGET, None, "Relative JavaScript/TypeScript import target could not be resolved", phase

    package_name = _bare_package_name(specifier)
    local = by_name.get(package_name)
    if local is not None:
        return ResolutionStatus.RESOLVED, EvidenceNodeScope.PROJECT_COMPONENT, local.path, "Target resolves to a local npm package manifest", phase

    section = _package_section(nearest, package_name)
    if section is not None:
        if section in {"dependencies", "peerDependencies", "optionalDependencies"} and phase == DependencyPhase.UNKNOWN:
            phase = DependencyPhase.RUNTIME
        elif section == "devDependencies" and phase == DependencyPhase.UNKNOWN:
            phase = DependencyPhase.BUILD
        return ResolutionStatus.EXTERNAL, EvidenceNodeScope.EXTERNAL_DEPENDENCY, None, f"Target matches nearest npm {section} declaration", phase
    return ResolutionStatus.UNRESOLVED, EvidenceNodeScope.UNRESOLVED_TARGET, None, "Bare package import is not declared in the nearest package.json", phase


def source_edges(
    root: Path,
    rel: str,
    by_path: dict[str, NpmPackage],
    by_name: dict[str, NpmPackage],
) -> tuple[list[DependencyEdge], list[str]]:
    try:
        source = (root / rel).read_text(encoding="utf-8-sig")
    except (OSError, UnicodeError) as exc:
        return [], [str(exc)]
    nearest = _nearest_package(rel, by_path)
    mask = _javascript_code_positions(source)
    edges: list[DependencyEdge] = []
    seen: set[tuple[int, str, EdgeType]] = set()

    patterns = [
        (_STATIC_IMPORT_RE, EdgeType.IMPORTS),
        (_SIDE_EFFECT_IMPORT_RE, EdgeType.IMPORTS),
        (_REQUIRE_LITERAL_RE, EdgeType.IMPORTS),
        (_DYNAMIC_IMPORT_LITERAL_RE, EdgeType.LOADS),
    ]
    literal_require_offsets: set[int] = set()
    literal_dynamic_offsets: set[int] = set()
    for pattern, edge_type in patterns:
        for match in pattern.finditer(source):
            if not keyword_is_code(source, mask, match.start(), match.end(), "import", "export", "require"):
                continue
            specifier = match.group(1)
            line = _line_number(source, match.start())
            key = (line, specifier, edge_type)
            if key in seen:
                continue
            seen.add(key)
            if pattern is _REQUIRE_LITERAL_RE:
                literal_require_offsets.add(source.find("require", match.start(), match.end()))
            if pattern is _DYNAMIC_IMPORT_LITERAL_RE:
                literal_dynamic_offsets.add(source.find("import", match.start(), match.end()))
            resolution, scope, resolved, note, phase = _target_state(root, rel, specifier, by_name, nearest)
            edges.append(
                DependencyEdge(
                    evidence_id=f"javascript:{rel}:{line}:{edge_type.value.lower()}:{specifier}",
                    source=rel,
                    target=specifier,
                    edge_type=edge_type,
                    phase=phase,
                    resolution=resolution,
                    target_scope=scope,
                    provenance=EvidenceProvenance.OBSERVED,
                    line=line,
                    resolved_path=resolved,
                    adapter="javascript-typescript",
                    note=note,
                )
            )

    for pattern, literal_offsets, label in (
        (_REQUIRE_CALL_RE, literal_require_offsets, "require"),
        (_DYNAMIC_IMPORT_CALL_RE, literal_dynamic_offsets, "import"),
    ):
        for match in pattern.finditer(source):
            if not mask[match.start()]:
                continue
            if match.start() in literal_offsets:
                continue
            line = _line_number(source, match.start())
            edges.append(
                DependencyEdge(
                    evidence_id=f"javascript-dynamic:{rel}:{line}:{label}",
                    source=rel,
                    target="<dynamic-import>",
                    edge_type=EdgeType.LOADS,
                    phase=_source_phase(rel),
                    resolution=ResolutionStatus.DYNAMIC,
                    target_scope=EvidenceNodeScope.UNRESOLVED_TARGET,
                    provenance=EvidenceProvenance.OBSERVED,
                    line=line,
                    adapter="javascript-typescript",
                    note="Dynamic JavaScript/TypeScript dependency target is not a string literal",
                )
            )
    return edges, []


def manifest_edges(package: NpmPackage, by_name: dict[str, NpmPackage]) -> list[DependencyEdge]:
    edges: list[DependencyEdge] = []
    for section, values in package.sections.items():
        phase = DependencyPhase.RUNTIME if section in {"dependencies", "peerDependencies", "optionalDependencies"} else DependencyPhase.BUILD
        for name in sorted(values):
            local = by_name.get(name)
            if local is not None and local.path != package.path:
                resolution = ResolutionStatus.RESOLVED
                scope = EvidenceNodeScope.PROJECT_COMPONENT
                resolved = local.path
                note = f"Local npm package dependency declared in {section}"
            else:
                resolution = ResolutionStatus.EXTERNAL
                scope = EvidenceNodeScope.EXTERNAL_DEPENDENCY
                resolved = None
                note = f"External npm package dependency declared in {section}"
            edges.append(
                DependencyEdge(
                    evidence_id=f"npm-manifest:{package.path}:{section}:{name}",
                    source=package.path,
                    target=name,
                    edge_type=EdgeType.IMPORTS,
                    phase=phase,
                    resolution=resolution,
                    target_scope=scope,
                    provenance=EvidenceProvenance.DECLARED,
                    resolved_path=resolved,
                    adapter="npm-manifest",
                    note=note,
                )
            )
    return edges
