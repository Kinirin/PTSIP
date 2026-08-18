from __future__ import annotations

import json
import re
import tomllib
import xml.etree.ElementTree as ET
from dataclasses import asdict, dataclass
from pathlib import Path

from .validation.components import ComponentPartition, partition_components


_REQUIREMENT_NAME_RE = re.compile(r"^\s*([A-Za-z0-9][A-Za-z0-9_.-]*)")
_EXECUTABLE_LIFECYCLE_CLASSES = {
    "PRODUCT",
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
}


@dataclass(frozen=True)
class ManifestEvidence:
    component_id: str
    classification: str
    path: str
    kind: str
    direct_dependencies: tuple[str, ...]
    complete: bool
    issue: str | None = None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class BuildResolutionResult:
    status: str
    reason: str | None
    manifests: tuple[ManifestEvidence, ...]
    blocking_gaps: tuple[dict[str, object], ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "status": self.status,
            "reason": self.reason,
            "manifests": [item.as_dict() for item in self.manifests],
            "blocking_gaps": list(self.blocking_gaps),
        }


def _normalize_requirement(value: str) -> str | None:
    match = _REQUIREMENT_NAME_RE.match(value)
    return match.group(1) if match else None


def _parse_pyproject(path: Path) -> tuple[tuple[str, ...], str | None]:
    try:
        payload = tomllib.loads(path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, tomllib.TOMLDecodeError) as exc:
        return (), str(exc)
    project = payload.get("project", {}) if isinstance(payload, dict) else {}
    if not isinstance(project, dict):
        return (), None
    values: list[str] = []
    groups: list[object] = [project.get("dependencies", [])]
    optional = project.get("optional-dependencies", {})
    if isinstance(optional, dict):
        groups.extend(optional.values())
    for group in groups:
        if not isinstance(group, list):
            continue
        for item in group:
            if isinstance(item, str):
                name = _normalize_requirement(item)
                if name:
                    values.append(name)
    return tuple(sorted(set(values))), None


def _parse_requirements(path: Path) -> tuple[tuple[str, ...], str | None]:
    try:
        lines = path.read_text(encoding="utf-8-sig").splitlines()
    except (OSError, UnicodeError) as exc:
        return (), str(exc)
    values: list[str] = []
    unresolved_includes: list[str] = []
    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if stripped.startswith(("-r", "--requirement", "-c", "--constraint")):
            unresolved_includes.append(stripped)
            continue
        if stripped.startswith(("-e", "--editable")):
            unresolved_includes.append(stripped)
            continue
        name = _normalize_requirement(stripped)
        if name:
            values.append(name)
    issue = None
    if unresolved_includes:
        issue = "Nested/editable requirement declarations require additional resolution: " + ", ".join(unresolved_includes[:10])
    return tuple(sorted(set(values))), issue


def _parse_package_json(path: Path) -> tuple[tuple[str, ...], str | None]:
    try:
        payload = json.loads(path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        return (), str(exc)
    if not isinstance(payload, dict):
        return (), "package.json root is not an object"
    values: set[str] = set()
    for section in ("dependencies", "devDependencies", "peerDependencies", "optionalDependencies"):
        group = payload.get(section, {})
        if isinstance(group, dict):
            values.update(str(name) for name in group if isinstance(name, str))
    return tuple(sorted(values)), None


def _parse_csproj(path: Path) -> tuple[tuple[str, ...], str | None]:
    try:
        tree = ET.parse(path)
    except (OSError, ET.ParseError) as exc:
        return (), str(exc)
    values: set[str] = set()
    for element in tree.iter():
        if element.tag.endswith("PackageReference"):
            include = element.attrib.get("Include") or element.attrib.get("Update")
            if include:
                values.add(include)
        elif element.tag.endswith("ProjectReference"):
            include = element.attrib.get("Include")
            if include:
                values.add(include.replace("\\", "/"))
    return tuple(sorted(values)), None


def _parse_go_mod(path: Path) -> tuple[tuple[str, ...], str | None]:
    try:
        lines = path.read_text(encoding="utf-8-sig").splitlines()
    except (OSError, UnicodeError) as exc:
        return (), str(exc)
    values: set[str] = set()
    in_require = False
    for line in lines:
        stripped = line.split("//", 1)[0].strip()
        if not stripped:
            continue
        if stripped == "require (":
            in_require = True
            continue
        if in_require and stripped == ")":
            in_require = False
            continue
        if stripped.startswith("require "):
            fields = stripped[len("require ") :].split()
            if fields:
                values.add(fields[0])
        elif in_require:
            fields = stripped.split()
            if fields:
                values.add(fields[0])
    return tuple(sorted(values)), None


def _parse_manifest(path: Path) -> tuple[str, tuple[str, ...], str | None]:
    name = path.name.lower()
    suffix = path.suffix.lower()
    if name == "pyproject.toml":
        deps, issue = _parse_pyproject(path)
        return "python-pyproject", deps, issue
    if name.startswith("requirements") and suffix in {".txt", ".in"}:
        deps, issue = _parse_requirements(path)
        return "python-requirements", deps, issue
    if name == "package.json":
        deps, issue = _parse_package_json(path)
        return "npm-package", deps, issue
    if suffix == ".csproj":
        deps, issue = _parse_csproj(path)
        return "dotnet-project", deps, issue
    if name == "go.mod":
        deps, issue = _parse_go_mod(path)
        return "go-module", deps, issue
    return "unsupported", (), "Unsupported build/dependency manifest type"


def _gap(component_id: str, message: str, evidence_id: str) -> dict[str, object]:
    return {
        "id": f"build-resolution:{component_id}:{evidence_id}",
        "blocking": True,
        "rule_ids": ["PTSIP-BLD-001", "PTSIP-EVD-003"],
        "evidence_ids": [f"build-resolution:{component_id}:{evidence_id}"],
        "message": message,
    }


def _normalize_manifest_path(raw: str) -> str:
    normalized = raw.replace("\\", "/")
    while normalized.startswith("./"):
        normalized = normalized[2:]
    return normalized


def evaluate_independent_build_resolution(
    repository_root: str | Path,
    components: list[dict[str, object]],
    partition: ComponentPartition | None = None,
) -> BuildResolutionResult:
    root = Path(repository_root).resolve()
    partition = partition or partition_components(root, components)
    owners = {assignment.path: assignment.component_id for assignment in partition.assignments}
    classifications = {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }
    conflicted_paths = {conflict.path for conflict in partition.conflicts}
    manifests: list[ManifestEvidence] = []
    gaps: list[dict[str, object]] = []
    manifest_users: dict[str, list[tuple[str, str]]] = {}

    relevant_components = [
        item
        for item in components
        if str(item.get("classification")) in _EXECUTABLE_LIFECYCLE_CLASSES
        and item.get("executable") is not False
    ]
    if not relevant_components:
        return BuildResolutionResult("RAN", None, (), ())

    for component in relevant_components:
        component_id = str(component.get("id", ""))
        classification = str(component.get("classification", ""))
        declared = component.get("manifests", [])
        manifest_paths = [str(item) for item in declared] if isinstance(declared, list) else []
        if not manifest_paths:
            gaps.append(
                _gap(
                    component_id,
                    (
                        f"Executable {classification} component does not declare a dependency/build manifest, "
                        "so its direct dependency set cannot be determined independently by this evaluator."
                    ),
                    "manifest-missing",
                )
            )
            continue

        usable = 0
        for raw in manifest_paths:
            normalized = _normalize_manifest_path(raw)
            candidate = (root / normalized).resolve()
            try:
                inside = candidate.is_relative_to(root)
            except (OSError, ValueError):
                inside = False
            if not inside or not candidate.is_file():
                gaps.append(_gap(component_id, f"Declared manifest {raw!r} does not resolve to a repository file.", f"manifest-unresolved:{normalized}"))
                continue
            relative_path = candidate.relative_to(root).as_posix()
            manifest_owner = owners.get(relative_path)
            ownership_issue: str | None = None
            if relative_path in conflicted_paths:
                ownership_issue = "manifest ownership is conflicted across component selectors"
            elif manifest_owner is None:
                ownership_issue = "manifest is outside declared component ownership"
            elif manifest_owner != component_id:
                owner_plane = classifications.get(manifest_owner, "UNKNOWN")
                ownership_issue = (
                    f"manifest is owned by component {manifest_owner!r} ({owner_plane}), not declaring component "
                    f"{component_id!r} ({classification})"
                )
            kind, dependencies, issue = _parse_manifest(candidate)
            combined_issue = "; ".join(item for item in (ownership_issue, issue) if item) or None
            evidence = ManifestEvidence(
                component_id=component_id,
                classification=classification,
                path=relative_path,
                kind=kind,
                direct_dependencies=dependencies,
                complete=combined_issue is None,
                issue=combined_issue,
            )
            manifests.append(evidence)
            manifest_users.setdefault(evidence.path, []).append((component_id, classification))
            if combined_issue is not None:
                gaps.append(_gap(component_id, f"Manifest {evidence.path!r} is not usable as independent build evidence: {combined_issue}", f"manifest-incomplete:{evidence.path}"))
            else:
                usable += 1
        if usable == 0 and manifest_paths:
            gaps.append(_gap(component_id, "No declared manifest produced a complete direct-dependency set.", "no-complete-manifest"))

    for path, users in sorted(manifest_users.items()):
        lifecycles = {classification for _, classification in users}
        if len(lifecycles) <= 1:
            continue
        components_text = ", ".join(sorted(component_id for component_id, _ in users))
        lifecycle_text = ", ".join(sorted(lifecycles))
        gaps.append(
            _gap(
                components_text.replace(", ", "+"),
                (
                    f"The same manifest {path!r} is declared by components across multiple primary lifecycles "
                    f"({lifecycle_text}): {components_text}. This evidence does not by itself prove independently "
                    "determinable direct dependency sets."
                ),
                f"shared-cross-lifecycle-manifest:{path}",
            )
        )

    status = "RAN" if not gaps else "BLOCKED"
    reason = None if not gaps else "BUILD_RESOLUTION_EVIDENCE_INCOMPLETE"
    return BuildResolutionResult(status, reason, tuple(manifests), tuple(gaps))
