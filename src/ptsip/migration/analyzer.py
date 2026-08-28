from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

from ..evidence.contract import (
    EvidenceChannelStatus,
    EvidenceRecordStatus,
    NormalizedEvidenceSet,
)
from ..model import Classification, ResponsibilityRelationshipType
from ..repository.profile_path import normalize_profile_path, profile_path_on_disk
from ..repository.snapshot import capture_snapshot, compare_snapshots, repository_files
from ..source_compat.model import (
    CompatibilitySourceProfile,
    SourceFamily,
    V034SourceSemantics,
    V036SourceSemantics,
    thaw_json,
)
from ..validation.components import normalize_selector, selector_matches_path, selector_specificity
from ..validation.templates import TemplateMaterializationError, materialize_profile
from .model import (
    AmbiguousSourceElement,
    ArchitectureFinding,
    ArchitectureFindingKind,
    AsynchronousWorkTarget,
    EvidenceCorrelation,
    ExistingSourceElement,
    LifecycleFinding,
    LifecycleFindingKind,
    MigrationAnalysis,
    MigrationAnalysisIssue,
    RemovedSourceElement,
    RemovalMigrationElement,
    RequiredWorkElement,
    SourceCoverageProjection,
    SourceMigrationCompletion,
    SourceProjectionKind,
    TargetArchitectureState,
    TargetAssociatedArtifact,
    TargetCompatibility,
    TargetComponent,
    TargetRelationship,
    TargetSemantics,
    UncoveredRepositoryElement,
)


@dataclass(frozen=True)
class _ProjectedSourceArchitecture:
    coverages: tuple[SourceCoverageProjection, ...]
    relationship_semantics: tuple[tuple[str, str, str], ...]


@dataclass(frozen=True)
class _RepositoryResolution:
    existing: tuple[ExistingSourceElement, ...]
    removed: tuple[RemovedSourceElement, ...]
    uncovered: tuple[UncoveredRepositoryElement, ...]
    ambiguous: tuple[AmbiguousSourceElement, ...]


@dataclass(frozen=True)
class _TargetOwner:
    kind: SourceProjectionKind
    id: str
    classification: str | None
    selector: str


def default_target_semantics() -> TargetSemantics:
    return TargetSemantics(
        draft_version="0.3.7-draft",
        classifications=tuple(item.value for item in Classification),
        relationship_types=tuple(item.value for item in ResponsibilityRelationshipType),
    )


def target_state_from_mapping(payload: dict[str, object]) -> TargetArchitectureState:
    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        raise ValueError("Target state requires ptsip metadata.")
    version = ptsip.get("version")
    specification = ptsip.get("specification")
    revision = specification.get("revision") if isinstance(specification, dict) else None
    if not isinstance(version, str) or not isinstance(revision, str) or not revision:
        raise ValueError("Target state requires draft version and immutable specification revision.")

    responsibility_map = payload.get("responsibility_map")
    if not isinstance(responsibility_map, dict) or responsibility_map.get("mode") != "explicit":
        raise ValueError(
            "WU-05 target-state mapping accepts explicit target state only; "
            "template/hybrid target materialization belongs to the target-draft runtime boundary."
        )

    components = tuple(
        sorted(
            (
                TargetComponent(
                    id=str(item.get("id", "")),
                    classification=str(item.get("classification", "")),
                    include=tuple(str(value) for value in item.get("include", [])),
                    exclude=tuple(str(value) for value in item.get("exclude", [])),
                )
                for item in payload.get("components", [])
                if isinstance(item, dict)
            ),
            key=lambda item: item.id,
        )
    )
    artifacts = tuple(
        sorted(
            (
                TargetAssociatedArtifact(
                    id=str(item.get("id", "")),
                    anchor=str(item.get("anchor", "")),
                    include=tuple(str(value) for value in item.get("include", [])),
                    exclude=tuple(str(value) for value in item.get("exclude", [])),
                )
                for item in payload.get("associated_artifacts", [])
                if isinstance(item, dict)
            ),
            key=lambda item: item.id,
        )
    )
    relationships = tuple(
        sorted(
            (
                TargetRelationship(
                    id=str(item.get("id", "")),
                    source=str(item.get("from", "")),
                    target=str(item.get("to", "")),
                    relationship_type=str(item.get("type", "")),
                )
                for item in payload.get("relationships", [])
                if isinstance(item, dict)
            ),
            key=lambda item: item.id,
        )
    )
    return TargetArchitectureState(version, revision, components, artifacts, relationships)


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


def _boundary_matches(path: str, root: str) -> bool:
    path = normalize_selector(path)
    root = normalize_selector(root)
    return path == root or path.startswith(root + "/")


def _coverage_match(path: str, coverage: SourceCoverageProjection) -> tuple[tuple[int, int, int, int], str] | None:
    if coverage.kind == SourceProjectionKind.BOUNDARY:
        root = coverage.include[0] if coverage.include else ""
        if not root or not _boundary_matches(path, root):
            return None
        selector = normalize_selector(root)
        return selector_specificity(selector + "/**"), selector
    if any(selector_matches_path(path, selector) for selector in coverage.exclude):
        return None
    matching = [selector for selector in coverage.include if selector_matches_path(path, selector)]
    if not matching:
        return None
    best = max(matching, key=selector_specificity)
    return selector_specificity(best), normalize_selector(best)


def _resolve_repository(paths: Iterable[str], source: _ProjectedSourceArchitecture) -> _RepositoryResolution:
    existing: list[ExistingSourceElement] = []
    uncovered: list[UncoveredRepositoryElement] = []
    ambiguous: list[AmbiguousSourceElement] = []
    selector_hits: dict[tuple[str, str], int] = {}

    for path in sorted(set(paths)):
        matches: list[tuple[tuple[int, int, int, int], SourceCoverageProjection, str]] = []
        for coverage in source.coverages:
            matched = _coverage_match(path, coverage)
            if matched is None:
                continue
            score, selector = matched
            selector_hits[(coverage.declaration_id, selector)] = selector_hits.get((coverage.declaration_id, selector), 0) + 1
            matches.append((score, coverage, selector))
        if not matches:
            uncovered.append(UncoveredRepositoryElement(path))
            continue

        kinds = {item[1].kind for item in matches}
        if SourceProjectionKind.COMPONENT in kinds and SourceProjectionKind.ASSOCIATED_ARTIFACT in kinds:
            selected = matches
        else:
            best_score = max(item[0] for item in matches)
            selected = [item for item in matches if item[0] == best_score]

        if len({item[1].declaration_id for item in selected}) != 1:
            ambiguous.append(
                AmbiguousSourceElement(
                    path,
                    tuple(sorted({item[1].declaration_id: item[1] for item in selected}.values(), key=lambda coverage: (coverage.kind.value, coverage.declaration_id))),
                    tuple(sorted({item[2] for item in selected})),
                )
            )
            continue
        chosen = sorted(selected, key=lambda item: (item[1].kind.value, item[1].declaration_id, item[2]))[0]
        existing.append(ExistingSourceElement(path, chosen[1], chosen[2]))

    removed: list[RemovedSourceElement] = []
    for coverage in source.coverages:
        for selector in coverage.include:
            normalized = normalize_selector(selector)
            if selector_hits.get((coverage.declaration_id, normalized), 0):
                continue
            removed.append(RemovedSourceElement(f"{coverage.declaration_id}:{normalized}", coverage, normalized))

    return _RepositoryResolution(
        tuple(sorted(existing, key=lambda item: item.path)),
        tuple(sorted(removed, key=lambda item: item.element_id)),
        tuple(sorted(uncovered, key=lambda item: item.path)),
        tuple(sorted(ambiguous, key=lambda item: item.path)),
    )


def _evidence_for_path(evidence: NormalizedEvidenceSet, path: str) -> EvidenceCorrelation:
    semantic_ids: set[str] = set()
    conflict_ids: set[str] = set()
    normalized_path = normalize_selector(path)
    for record in evidence.records:
        matched = record.subject in {normalized_path, f"path:{normalized_path}"}
        qualifier_path = record.qualifiers.get("path")
        if isinstance(qualifier_path, str) and normalize_selector(qualifier_path) == normalized_path:
            matched = True
        for assertion in record.assertions:
            for origin in assertion.origins:
                if origin.source_path and normalize_selector(origin.source_path) == normalized_path:
                    matched = True
        if matched:
            semantic_ids.add(record.semantic_id)
            if record.status == EvidenceRecordStatus.CONFLICT:
                conflict_ids.add(record.semantic_id)
    incomplete = tuple(sorted(channel.id for channel in evidence.channels if channel.status in {EvidenceChannelStatus.FAILED, EvidenceChannelStatus.NOT_ANALYZED}))
    return EvidenceCorrelation(tuple(sorted(semantic_ids)), tuple(sorted(conflict_ids)), incomplete)


def _validate_source_binding(repository_root: Path, profile: CompatibilitySourceProfile) -> tuple[MigrationAnalysisIssue, ...]:
    binding = profile.generation
    try:
        path = profile_path_on_disk(repository_root, normalize_profile_path(binding.profile_path))
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
    except (OSError, ValueError) as exc:
        return (MigrationAnalysisIssue("SOURCE_BINDING_INVALID", f"Unable to validate WU-04 source binding: {exc}"),)
    if digest != binding.content_sha256:
        return (MigrationAnalysisIssue("SOURCE_CONTENT_STALE", "Source profile bytes changed after WU-04 compatibility read."),)
    return ()


def _validate_evidence_context(profile: CompatibilitySourceProfile, evidence: NormalizedEvidenceSet, snapshot) -> tuple[MigrationAnalysisIssue, ...]:
    issues: list[MigrationAnalysisIssue] = []
    binding = evidence.context.source_generation
    source = profile.generation
    if binding is None:
        issues.append(MigrationAnalysisIssue("EVIDENCE_SOURCE_CONTEXT_MISSING", "Normalized evidence is not bound to a source profile generation."))
    elif (
        binding.profile_path != source.profile_path
        or binding.version != source.declared_version
        or binding.specification_revision != source.specification_revision
        or binding.content_sha256 != source.content_sha256
    ):
        issues.append(MigrationAnalysisIssue("EVIDENCE_SOURCE_CONTEXT_MISMATCH", "Normalized evidence source generation does not match the WU-04 source profile."))
    evidence_snapshot = evidence.context.snapshot
    if (
        evidence_snapshot.revision != snapshot.head
        or evidence_snapshot.status_fingerprint != snapshot.status_fingerprint
        or evidence_snapshot.tracked_content_fingerprint != snapshot.tracked_content_fingerprint
    ):
        issues.append(MigrationAnalysisIssue("EVIDENCE_SNAPSHOT_STALE", "Normalized evidence is not bound to the repository snapshot being analyzed."))
    return tuple(issues)


def _target_match(path: str, target: TargetArchitectureState) -> tuple[_TargetOwner | None, bool]:
    component_matches: list[tuple[tuple[int, int, int, int], _TargetOwner]] = []
    for item in target.components:
        if any(selector_matches_path(path, selector) for selector in item.exclude):
            continue
        matching = [selector for selector in item.include if selector_matches_path(path, selector)]
        if matching:
            best = max(matching, key=selector_specificity)
            component_matches.append((selector_specificity(best), _TargetOwner(SourceProjectionKind.COMPONENT, item.id, item.classification, best)))
    artifact_matches: list[tuple[tuple[int, int, int, int], _TargetOwner]] = []
    for item in target.associated_artifacts:
        if any(selector_matches_path(path, selector) for selector in item.exclude):
            continue
        matching = [selector for selector in item.include if selector_matches_path(path, selector)]
        if matching:
            best = max(matching, key=selector_specificity)
            artifact_matches.append((selector_specificity(best), _TargetOwner(SourceProjectionKind.ASSOCIATED_ARTIFACT, item.id, None, best)))
    if component_matches and artifact_matches:
        return None, True
    selected = component_matches or artifact_matches
    if not selected:
        return None, False
    best_score = max(item[0] for item in selected)
    winners = [item[1] for item in selected if item[0] == best_score]
    if len({(item.kind.value, item.id) for item in winners}) != 1:
        return None, True
    return sorted(winners, key=lambda item: (item.kind.value, item.id))[0], False


def _target_compatibility(
    element: ExistingSourceElement,
    target: TargetArchitectureState | None,
    target_semantics: TargetSemantics,
) -> tuple[TargetCompatibility, LifecycleFinding | None]:
    source_classification = element.coverage.source_classification
    if target is None:
        if source_classification == "TOOLCHAIN":
            return TargetCompatibility.NOT_EVALUATED, LifecycleFinding(
                element.path,
                LifecycleFindingKind.HISTORICAL_TOOLCHAIN_AMBIGUITY,
                source_classification,
                None,
                "Historical TOOLCHAIN is source vocabulary and cannot select one target lifecycle without project-owned resolution.",
            )
        return TargetCompatibility.NOT_EVALUATED, None

    owner, ambiguous = _target_match(element.path, target)
    if ambiguous:
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
            source_classification,
            None,
            "Accepted target state has ambiguous coverage for this source obligation.",
        )
    if owner is None:
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
            source_classification,
            None,
            "No accepted target declaration provably covers this required source element.",
        )

    if element.coverage.kind == SourceProjectionKind.ASSOCIATED_ARTIFACT:
        if owner.kind != SourceProjectionKind.ASSOCIATED_ARTIFACT:
            return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
                element.path,
                LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
                None,
                owner.classification,
                "A source associated-artifact obligation is covered by a different target declaration kind.",
            )
        return (TargetCompatibility.ALREADY_SATISFIED if owner.id == element.coverage.declaration_id else TargetCompatibility.COMPATIBLE_TARGET_STATE), None

    if owner.kind != SourceProjectionKind.COMPONENT:
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
            source_classification,
            None,
            "A source component/boundary obligation is covered by a target associated-artifact scope.",
        )
    target_classification = owner.classification
    if source_classification == "TOOLCHAIN":
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.HISTORICAL_TOOLCHAIN_AMBIGUITY,
            source_classification,
            target_classification,
            "Historical TOOLCHAIN can correspond to multiple target lifecycles; no automatic conversion is authoritative.",
        )
    if source_classification == target_classification and source_classification in target_semantics.classifications:
        return (
            TargetCompatibility.ALREADY_SATISFIED if owner.id == element.coverage.declaration_id else TargetCompatibility.COMPATIBLE_TARGET_STATE,
            LifecycleFinding(
                element.path,
                LifecycleFindingKind.EXACT_SEMANTIC_PRESERVATION,
                source_classification,
                target_classification,
                "Source lifecycle classification is preserved by accepted target coverage.",
            ),
        )
    if source_classification in target_semantics.classifications and target_classification in target_semantics.classifications:
        return TargetCompatibility.CONFLICTING_TARGET_STATE, LifecycleFinding(
            element.path,
            LifecycleFindingKind.POSSIBLE_LIFECYCLE_SEPARATION,
            source_classification,
            target_classification,
            "Source and accepted target lifecycle classifications differ; owner review is required before treating the obligation as resolved.",
        )
    return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
        element.path,
        LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
        source_classification,
        target_classification,
        "Lifecycle compatibility cannot be proven under the target draft vocabulary.",
    )


def _relationship_findings(source: _ProjectedSourceArchitecture, target: TargetArchitectureState | None) -> tuple[ArchitectureFinding, ...]:
    if target is None or not source.relationship_semantics:
        return ()
    target_semantics = {(item.source, item.target, item.relationship_type) for item in target.relationships}
    return tuple(
        ArchitectureFinding(
            f"relationship:{relation[0]}:{relation[1]}:{relation[2]}",
            ArchitectureFindingKind.MISSING_RELATIONSHIP,
            "A source typed relationship has no exact semantic counterpart in the accepted target state; this is reviewable analysis, not an automatic target delta.",
        )
        for relation in source.relationship_semantics
        if relation not in target_semantics
    )


def analyze_source_migration(
    repository_root: str | Path,
    source_profile: CompatibilitySourceProfile,
    evidence: NormalizedEvidenceSet,
    *,
    target_semantics: TargetSemantics | None = None,
    target_state: TargetArchitectureState | None = None,
) -> MigrationAnalysis:
    root = Path(repository_root).expanduser().resolve()
    semantics = target_semantics or default_target_semantics()
    issues: list[MigrationAnalysisIssue] = []
    before = capture_snapshot(root)
    if before.observation_errors:
        issues.append(MigrationAnalysisIssue("REPOSITORY_SNAPSHOT_INCOMPLETE", "Repository snapshot observation was incomplete: " + "; ".join(before.observation_errors)))

    if target_state is not None:
        if target_state.draft_version != semantics.draft_version:
            issues.append(MigrationAnalysisIssue("TARGET_DRAFT_MISMATCH", f"Target state {target_state.draft_version!r} does not match target semantics {semantics.draft_version!r}."))
        invalid_target_classes = sorted({item.classification for item in target_state.components if item.classification not in semantics.classifications})
        if invalid_target_classes:
            issues.append(MigrationAnalysisIssue("TARGET_CLASSIFICATION_UNSUPPORTED", "Target state uses unsupported lifecycle classification(s): " + ", ".join(invalid_target_classes)))

    issues.extend(_validate_source_binding(root, source_profile))
    issues.extend(_validate_evidence_context(source_profile, evidence, before))
    projected, projection_issues = _project_source(source_profile)
    issues.extend(projection_issues)
    if projected is None:
        return MigrationAnalysis(
            source_generation=source_profile.generation,
            repository_head=before.head,
            repository_status_fingerprint=before.status_fingerprint,
            repository_content_fingerprint=before.tracked_content_fingerprint,
            required=(),
            removals=(),
            async_targets=(),
            ambiguous=(),
            lifecycle_findings=(),
            architecture_findings=(),
            issues=tuple(sorted(issues, key=lambda item: (item.code, item.subject_id or "", item.message))),
            completion=SourceMigrationCompletion(0, 0, 0, 0, 0),
        )

    _mode, paths, path_errors = repository_files(root)
    issues.extend(MigrationAnalysisIssue("REPOSITORY_FILE_SCAN_ERROR", message) for message in path_errors)
    resolution = _resolve_repository(paths, projected)
    required: list[RequiredWorkElement] = []
    removals: list[RemovalMigrationElement] = []
    async_targets: list[AsynchronousWorkTarget] = []
    lifecycle_findings: list[LifecycleFinding] = []
    architecture_findings: list[ArchitectureFinding] = []

    for item in resolution.existing:
        correlation = _evidence_for_path(evidence, item.path)
        target_status, lifecycle = _target_compatibility(item, target_state, semantics)
        if lifecycle is not None:
            lifecycle_findings.append(lifecycle)
        resolved = target_status in {TargetCompatibility.ALREADY_SATISFIED, TargetCompatibility.COMPATIBLE_TARGET_STATE}
        required.append(
            RequiredWorkElement(
                f"required:{item.path}",
                item.path,
                item.coverage.declaration_id,
                item.coverage.source_classification,
                item.selector,
                correlation,
                target_status,
                resolved,
            )
        )
        if correlation.conflict_ids:
            architecture_findings.append(ArchitectureFinding(item.path, ArchitectureFindingKind.EVIDENCE_CONFLICT, "Normalized evidence contains incompatible assertions for this repository element.", correlation.conflict_ids))
        if correlation.incomplete_channels:
            architecture_findings.append(ArchitectureFinding(item.path, ArchitectureFindingKind.EVIDENCE_INCOMPLETE, "One or more evidence channels were not analyzed successfully.", correlation.semantic_ids))
        if item.coverage.kind == SourceProjectionKind.ASSOCIATED_ARTIFACT and target_state is not None and target_status == TargetCompatibility.TARGET_REVIEW_REQUIRED:
            architecture_findings.append(ArchitectureFinding(item.path, ArchitectureFindingKind.MISSING_ASSOCIATED_ARTIFACT, "Required source associated-artifact scope is not provably preserved as an associated artifact in accepted target state.", correlation.semantic_ids))

    for item in resolution.removed:
        removals.append(
            RemovalMigrationElement(
                f"removal:{item.element_id}",
                item.coverage.declaration_id,
                item.coverage.source_classification,
                item.selector,
                "Source declaration selector currently resolves to no repository element; it is not copied forward solely for historical preservation.",
            )
        )
        architecture_findings.append(ArchitectureFinding(item.element_id, ArchitectureFindingKind.STALE_SOURCE_DECLARATION, "Source selector has no current repository match."))

    for item in resolution.uncovered:
        correlation = _evidence_for_path(evidence, item.path)
        async_targets.append(AsynchronousWorkTarget(f"async:{item.path}", item.path, correlation))
        architecture_findings.append(ArchitectureFinding(item.path, ArchitectureFindingKind.NEW_REPOSITORY_CANDIDATE, "Current repository element is outside the source profile's active coverage and is non-blocking asynchronous work.", correlation.semantic_ids))

    for item in resolution.ambiguous:
        architecture_findings.append(ArchitectureFinding(item.path, ArchitectureFindingKind.AMBIGUOUS_SOURCE_COVERAGE, "Repository element matches multiple source declarations at the controlling specificity; obligation taxonomy is fail-closed."))

    architecture_findings.extend(_relationship_findings(projected, target_state))
    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    if not comparison.stable:
        issues.append(MigrationAnalysisIssue("ANALYSIS_SNAPSHOT_INVALIDATED", "; ".join(comparison.reasons)))

    required.sort(key=lambda item: item.id)
    removals.sort(key=lambda item: item.id)
    async_targets.sort(key=lambda item: item.id)
    lifecycle_findings.sort(key=lambda item: (item.subject_id, item.kind.value, item.target_classification or ""))
    architecture_findings.sort(key=lambda item: (item.subject_id, item.kind.value, item.rationale))
    issues.sort(key=lambda item: (item.code, item.subject_id or "", item.message))
    resolved_count = sum(item.resolved for item in required)
    completion = SourceMigrationCompletion(
        len(required),
        resolved_count,
        len(required) - resolved_count,
        len(removals),
        len(async_targets),
    )
    return MigrationAnalysis(
        source_profile.generation,
        before.head,
        before.status_fingerprint,
        before.tracked_content_fingerprint,
        tuple(required),
        tuple(removals),
        tuple(async_targets),
        resolution.ambiguous,
        tuple(lifecycle_findings),
        tuple(architecture_findings),
        tuple(issues),
        completion,
    )
