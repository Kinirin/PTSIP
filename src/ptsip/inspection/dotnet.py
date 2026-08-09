from __future__ import annotations

import re
import xml.etree.ElementTree as ET
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
from .lexing import code_positions


_USING_RE = re.compile(
    r"^\s*(?:global\s+)?using\s+(?:static\s+)?(?:[A-Za-z_][\w]*\s*=\s*)?([A-Za-z_][\w.]*)\s*;"
)


@dataclass(frozen=True)
class DotNetProject:
    project_path: str
    directory: str
    root_namespace: str
    package_references: tuple[str, ...]
    project_references: tuple[str, ...]


def _element_text(root: ET.Element, name: str) -> str | None:
    for element in root.iter():
        if element.tag.endswith(name) and element.text and element.text.strip():
            return element.text.strip()
    return None


def discover_dotnet_projects(root: Path, paths: list[str]) -> tuple[list[DotNetProject], list[tuple[str, str]]]:
    projects: list[DotNetProject] = []
    issues: list[tuple[str, str]] = []
    for rel in paths:
        if Path(rel).suffix.lower() != ".csproj":
            continue
        try:
            tree = ET.parse(root / rel)
        except (OSError, ET.ParseError) as exc:
            issues.append((rel, str(exc)))
            continue
        document = tree.getroot()
        root_namespace = _element_text(document, "RootNamespace") or _element_text(document, "AssemblyName") or Path(rel).stem
        packages: set[str] = set()
        project_references: set[str] = set()
        for element in document.iter():
            if element.tag.endswith("PackageReference"):
                name = element.attrib.get("Include") or element.attrib.get("Update")
                if name:
                    packages.add(name)
            elif element.tag.endswith("ProjectReference"):
                reference = element.attrib.get("Include")
                if reference:
                    candidate = ((root / rel).parent / reference.replace("\\", "/")).resolve()
                    try:
                        if candidate.is_relative_to(root):
                            project_references.add(candidate.relative_to(root).as_posix())
                    except (OSError, ValueError):
                        pass
        directory = Path(rel).parent.as_posix()
        if directory == ".":
            directory = ""
        projects.append(DotNetProject(rel, directory, root_namespace, tuple(sorted(packages)), tuple(sorted(project_references))))
    projects.sort(key=lambda item: (-len(item.root_namespace), item.project_path))
    return projects, issues


def _nearest_project(rel: str, projects: list[DotNetProject]) -> DotNetProject | None:
    source = Path(rel)
    candidates: list[tuple[int, DotNetProject]] = []
    for project in projects:
        directory = Path(project.directory) if project.directory else Path(".")
        try:
            source.relative_to(directory)
        except ValueError:
            continue
        candidates.append((len(directory.parts), project))
    return max(candidates, default=(0, None), key=lambda item: item[0])[1]


def _target_project(namespace: str, projects: list[DotNetProject], source_project: DotNetProject | None) -> DotNetProject | None:
    references = set(source_project.project_references) if source_project is not None else set()
    matches = [
        project
        for project in projects
        if project is not source_project
        and project.project_path in references
        and (namespace == project.root_namespace or namespace.startswith(project.root_namespace + "."))
    ]
    return max(matches, default=None, key=lambda item: len(item.root_namespace))


def _namespace_project(namespace: str, projects: list[DotNetProject], source_project: DotNetProject | None) -> DotNetProject | None:
    matches = [
        project
        for project in projects
        if project is not source_project
        and (namespace == project.root_namespace or namespace.startswith(project.root_namespace + "."))
    ]
    return max(matches, default=None, key=lambda item: len(item.root_namespace))


def _package_match(namespace: str, project: DotNetProject | None) -> str | None:
    if project is None:
        return None
    normalized_namespace = namespace.lower().replace("_", ".")
    matches = [
        package
        for package in project.package_references
        if normalized_namespace == package.lower().replace("-", ".")
        or normalized_namespace.startswith(package.lower().replace("-", ".") + ".")
    ]
    return max(matches, default=None, key=len)


def source_edges(root: Path, rel: str, projects: list[DotNetProject]) -> tuple[list[DependencyEdge], list[str]]:
    try:
        source = (root / rel).read_text(encoding="utf-8-sig")
    except (OSError, UnicodeError) as exc:
        return [], [str(exc)]
    source_project = _nearest_project(rel, projects)
    phase = DependencyPhase.TEST if any(token in rel.lower() for token in ("/test/", "/tests/", ".tests/", ".test/")) else DependencyPhase.RUNTIME
    edges: list[DependencyEdge] = []
    mask = code_positions(source)
    offset = 0
    for line_no, line in enumerate(source.splitlines(keepends=True), start=1):
        match = _USING_RE.match(line)
        keyword = line.find("using")
        if not match or keyword < 0 or not mask[offset + keyword]:
            offset += len(line)
            continue
        namespace = match.group(1)
        local = _target_project(namespace, projects, source_project)
        namespace_only = _namespace_project(namespace, projects, source_project)
        if local is not None:
            resolution = ResolutionStatus.RESOLVED
            scope = EvidenceNodeScope.PROJECT_COMPONENT
            resolved_path = local.project_path
            note = f"Namespace matches local .NET project root namespace {local.root_namespace!r}"
        elif namespace == "System" or namespace.startswith("System."):
            resolution = ResolutionStatus.EXTERNAL
            scope = EvidenceNodeScope.PLATFORM
            resolved_path = None
            note = ".NET platform namespace"
        else:
            package = _package_match(namespace, source_project)
            if package and namespace_only is None:
                resolution = ResolutionStatus.EXTERNAL
                scope = EvidenceNodeScope.EXTERNAL_DEPENDENCY
                resolved_path = None
                note = f"Namespace deterministically matches PackageReference {package!r}"
            else:
                resolution = ResolutionStatus.UNRESOLVED
                scope = EvidenceNodeScope.UNRESOLVED_TARGET
                resolved_path = None
                if package and namespace_only is not None:
                    note = "Namespace matches both an unreferenced local project and PackageReference; source usage is not attributable without assembly/project identity"
                elif namespace_only is not None:
                    note = "Namespace matches a local project, but no ProjectReference or equivalent build relationship establishes local project identity"
                else:
                    note = "Namespace is not attributable to a local project, platform namespace, or deterministically matching PackageReference"
        edges.append(
            DependencyEdge(
                evidence_id=f"dotnet-source:{rel}:{line_no}:{namespace}",
                source=rel,
                target=namespace,
                edge_type=EdgeType.IMPORTS,
                phase=phase,
                resolution=resolution,
                target_scope=scope,
                provenance=EvidenceProvenance.OBSERVED,
                line=line_no,
                resolved_path=resolved_path,
                adapter="dotnet-source",
                note=note,
            )
        )
        offset += len(line)
    return edges, []
