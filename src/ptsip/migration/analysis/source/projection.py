from __future__ import annotations

from dataclasses import dataclass

from ptsip.source_compat.model import CompatibilitySourceProfile, SourceFamily, V034SourceSemantics, V036SourceSemantics, thaw_json
from ptsip.validation.components import normalize_selector
from ptsip.validation.templates import TemplateMaterializationError, materialize_profile
from ptsip.migration.analysis.result import MigrationAnalysisIssue

class SourceProjectionKind(StrEnum):
    COMPONENT = "COMPONENT"
    ASSOCIATED_ARTIFACT = "ASSOCIATED_ARTIFACT"
    BOUNDARY = "BOUNDARY"

class RepositoryResolutionKind(StrEnum):
    EXISTING_SOURCE_ELEMENT = "EXISTING_SOURCE_ELEMENT"
    REMOVED_SOURCE_ELEMENT = "REMOVED_SOURCE_ELEMENT"
    UNCOVERED_REPOSITORY_ELEMENT = "UNCOVERED_REPOSITORY_ELEMENT"
    AMBIGUOUS_SOURCE_ELEMENT = "AMBIGUOUS_SOURCE_ELEMENT"

@dataclass(frozen=True)
class SourceCoverageProjection:
    declaration_id: str
    kind: SourceProjectionKind
    source_classification: str | None
    include: tuple[str, ...]
    exclude: tuple[str, ...]
    purpose: str
    source_pointer: str
    source_family: SourceFamily
    origin: str

    def as_dict(self) -> dict[str, object]:
        return {
            "declaration_id": self.declaration_id,
            "kind": self.kind.value,
            "source_classification": self.source_classification,
            "include": list(self.include),
            "exclude": list(self.exclude),
            "purpose": self.purpose,
            "source_pointer": self.source_pointer,
            "source_family": self.source_family.value,
            "origin": self.origin,
        }

@dataclass(frozen=True)
class ExistingSourceElement:
    path: str
    coverage: SourceCoverageProjection
    selector: str
    kind: RepositoryResolutionKind = RepositoryResolutionKind.EXISTING_SOURCE_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "coverage": self.coverage.as_dict(),
            "selector": self.selector,
        }

@dataclass(frozen=True)
class RemovedSourceElement:
    element_id: str
    coverage: SourceCoverageProjection
    selector: str
    kind: RepositoryResolutionKind = RepositoryResolutionKind.REMOVED_SOURCE_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "element_id": self.element_id,
            "coverage": self.coverage.as_dict(),
            "selector": self.selector,
        }

@dataclass(frozen=True)
class UncoveredRepositoryElement:
    path: str
    kind: RepositoryResolutionKind = RepositoryResolutionKind.UNCOVERED_REPOSITORY_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {"kind": self.kind.value, "path": self.path}

@dataclass(frozen=True)
class AmbiguousSourceElement:
    path: str
    coverages: tuple[SourceCoverageProjection, ...]
    selectors: tuple[str, ...]
    kind: RepositoryResolutionKind = RepositoryResolutionKind.AMBIGUOUS_SOURCE_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "coverages": [item.as_dict() for item in self.coverages],
            "selectors": list(self.selectors),
        }

@dataclass(frozen=True)
class _ProjectedSourceArchitecture:
    coverages: tuple[SourceCoverageProjection, ...]
    relationship_semantics: tuple[tuple[str, str, str], ...]

def _source_component_projection(profile: CompatibilitySourceProfile, item, *, origin: str) -> SourceCoverageProjection:
    return SourceCoverageProjection(
        declaration_id=item.id,
        kind=SourceProjectionKind.COMPONENT,
        source_classification=item.source_classification,
        include=tuple(item.include),
        exclude=tuple(item.exclude),
        purpose=item.purpose,
        source_pointer=item.location.pointer,
        source_family=profile.family,
        origin=origin,
    )

def _source_artifact_projection(profile: CompatibilitySourceProfile, item, *, origin: str) -> SourceCoverageProjection:
    return SourceCoverageProjection(
        declaration_id=item.id,
        kind=SourceProjectionKind.ASSOCIATED_ARTIFACT,
        source_classification=None,
        include=tuple(item.include),
        exclude=tuple(item.exclude),
        purpose=item.purpose,
        source_pointer=item.location.pointer,
        source_family=profile.family,
        origin=origin,
    )

def _mapping_projection(
    profile: CompatibilitySourceProfile,
    item: dict[str, object],
    *,
    kind: SourceProjectionKind,
    origin: str,
    pointer: str,
) -> SourceCoverageProjection:
    return SourceCoverageProjection(
        declaration_id=str(item["id"]),
        kind=kind,
        source_classification=str(item["classification"]) if kind == SourceProjectionKind.COMPONENT else None,
        include=tuple(str(value) for value in item.get("include", [])),
        exclude=tuple(str(value) for value in item.get("exclude", [])),
        purpose=str(item.get("purpose", "")),
        source_pointer=pointer,
        source_family=profile.family,
        origin=origin,
    )

def _project_source(profile: CompatibilitySourceProfile) -> tuple[_ProjectedSourceArchitecture | None, tuple[MigrationAnalysisIssue, ...]]:
    if profile.family == SourceFamily.TOOL_035_PROFILE:
        semantics = profile.family_semantics
        if not isinstance(semantics, V034SourceSemantics):
            return None, (MigrationAnalysisIssue("SOURCE_SEMANTICS_MISMATCH", "Tool 0.3.5 source family is not backed by V034 source semantics."),)
        coverages = [
            _source_component_projection(profile, item, origin="SOURCE_COMPONENT")
            for item in profile.components
        ]
        if semantics.declaration_form == "BOUNDARIES":
            for boundary in semantics.boundaries:
                for index, root in enumerate(boundary.roots):
                    normalized = normalize_selector(root)
                    coverages.append(
                        SourceCoverageProjection(
                            declaration_id=f"boundary:{boundary.source_classification}:{index}:{normalized}",
                            kind=SourceProjectionKind.BOUNDARY,
                            source_classification=boundary.source_classification,
                            include=(normalized,),
                            exclude=(),
                            purpose="historical_boundary_root",
                            source_pointer=boundary.location.pointer,
                            source_family=profile.family,
                            origin="SOURCE_BOUNDARY",
                        )
                    )
        return _ProjectedSourceArchitecture(
            tuple(sorted(coverages, key=lambda item: (item.kind.value, item.declaration_id))),
            (),
        ), ()

    semantics = profile.family_semantics
    if not isinstance(semantics, V036SourceSemantics):
        return None, (MigrationAnalysisIssue("SOURCE_SEMANTICS_MISMATCH", "Tool 0.3.6 source family is not backed by V036 source semantics."),)

    if semantics.responsibility_map_mode == "explicit":
        coverages = [
            *(_source_component_projection(profile, item, origin="PROJECT_EXPLICIT") for item in profile.components),
            *(_source_artifact_projection(profile, item, origin="PROJECT_EXPLICIT") for item in profile.associated_artifacts),
        ]
        relationships = tuple(sorted((item.source, item.target, item.relationship_type) for item in profile.relationships))
        return _ProjectedSourceArchitecture(
            tuple(sorted(coverages, key=lambda item: (item.kind.value, item.declaration_id))),
            relationships,
        ), ()

    payload = thaw_json(profile.raw_payload)
    if not isinstance(payload, dict):
        return None, (MigrationAnalysisIssue("SOURCE_PAYLOAD_INVALID", "Compatibility source raw payload is not a mapping."),)
    try:
        resolved = materialize_profile(payload)
    except TemplateMaterializationError as exc:
        return None, (MigrationAnalysisIssue("SOURCE_TEMPLATE_RESOLUTION_FAILED", str(exc)),)

    effective = resolved.effective_payload
    override_components = {item.id: item for item in profile.components}
    override_artifacts = {item.id: item for item in profile.associated_artifacts}
    coverages: list[SourceCoverageProjection] = []
    for item in effective.get("components", []):
        if not isinstance(item, dict):
            continue
        item_id = str(item.get("id", ""))
        override = override_components.get(item_id)
        coverages.append(
            _source_component_projection(profile, override, origin="PROJECT_OVERRIDE")
            if override is not None
            else _mapping_projection(
                profile,
                item,
                kind=SourceProjectionKind.COMPONENT,
                origin="TEMPLATE_EFFECTIVE",
                pointer="/responsibility_map/template",
            )
        )
    for item in effective.get("associated_artifacts", []):
        if not isinstance(item, dict):
            continue
        item_id = str(item.get("id", ""))
        override = override_artifacts.get(item_id)
        coverages.append(
            _source_artifact_projection(profile, override, origin="PROJECT_OVERRIDE")
            if override is not None
            else _mapping_projection(
                profile,
                item,
                kind=SourceProjectionKind.ASSOCIATED_ARTIFACT,
                origin="TEMPLATE_EFFECTIVE",
                pointer="/responsibility_map/template",
            )
        )
    relationships = tuple(
        sorted(
            (str(item.get("from", "")), str(item.get("to", "")), str(item.get("type", "")))
            for item in effective.get("relationships", [])
            if isinstance(item, dict)
        )
    )
    return _ProjectedSourceArchitecture(
        tuple(sorted(coverages, key=lambda item: (item.kind.value, item.declaration_id))),
        relationships,
    ), ()

