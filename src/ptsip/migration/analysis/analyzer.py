from __future__ import annotations

from pathlib import Path

from ptsip.evidence.contract import NormalizedEvidenceSet
from ptsip.ptsip_specification.spec_identity import capture_snapshot, compare_snapshots, native_profile_operation, repository_files
from ptsip.source_compat.model import CompatibilitySourceProfile
from ptsip.migration.analysis.findings.architecture import ArchitectureFinding, ArchitectureFindingKind
from ptsip.migration.analysis.findings.lifecycle import LifecycleFinding, LifecycleFindingKind
from ptsip.migration.analysis.findings.work_requirements import (
    AsynchronousWorkTarget,
    RemovalMigrationElement,
    RequiredWorkElement,
    SourceMigrationCompletion,
    TargetCompatibility,
)
from ptsip.migration.analysis.issue import MigrationAnalysisIssue
from ptsip.migration.analysis.result import AmbiguousSourceElement, MigrationAnalysis, SourceProjectionKind
from ptsip.migration.analysis.source.binding_validation import _validate_evidence_context, _validate_source_binding
from ptsip.migration.analysis.target.relationship_findings import _relationship_findings
from ptsip.migration.analysis.target.semantics import (
    TargetArchitectureState,
    TargetSemantics,
    default_target_semantics,
)

def _project_source(profile: CompatibilitySourceProfile) -> dict[str, object]:
    def declaration(item) -> dict[str, object]:
        return {"id": item.id, "include": list(item.include), "exclude": list(item.exclude),
                "purpose": item.purpose, "source_pointer": item.location.pointer}

    semantics = profile.family_semantics.as_dict()
    if semantics["kind"] == "V034":
        semantics["boundaries"] = [
            {"source_classification": item.source_classification, "roots": list(item.roots),
             "source_pointer": item.location.pointer} for item in profile.family_semantics.boundaries
        ]
    return native_profile_operation("projection.migration.source", {
        "family": profile.family.value, "family_semantics": semantics,
        "components": [{**declaration(item), "source_classification": item.source_classification} for item in profile.components],
        "associated_artifacts": [declaration(item) for item in profile.associated_artifacts],
        "relationships": [{"id": item.id, "from": item.source, "to": item.target, "type": item.relationship_type} for item in profile.relationships],
    })

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
    projected = _project_source(source_profile)
    projection_issues = [MigrationAnalysisIssue(**item) for item in projected["issues"]]
    issues.extend(projection_issues)
    if projection_issues:
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
    resolution = native_profile_operation("projection.migration.coverage", {"paths": paths, "coverages": projected["coverages"]})
    compatibility = native_profile_operation("projection.migration.compatibility", {
        "existing": resolution["existing"], "target": target_state.as_dict() if target_state else None,
        "classifications": list(semantics.classifications),
    })
    by_path = {item["path"]: item for item in compatibility["results"]}
    required: list[RequiredWorkElement] = []
    removals: list[RemovalMigrationElement] = []
    async_targets: list[AsynchronousWorkTarget] = []
    lifecycle_findings: list[LifecycleFinding] = []
    architecture_findings: list[ArchitectureFinding] = []

    for item in resolution["existing"]:
        path = item["path"]
        coverage = item["coverage"]
        result = by_path[path]
        target_status = TargetCompatibility(result["target_status"])
        lifecycle = result["lifecycle"]
        if lifecycle is not None:
            lifecycle_findings.append(LifecycleFinding(**{**lifecycle, "kind": LifecycleFindingKind(lifecycle["kind"])}))
        resolved = target_status in {TargetCompatibility.ALREADY_SATISFIED, TargetCompatibility.COMPATIBLE_TARGET_STATE}
        required.append(
            RequiredWorkElement(
                f"required:{path}", path, coverage["declaration_id"],
                coverage["source_classification"], item["selector"],
                target_status,
                resolved,
            )
        )
        if coverage["kind"] == SourceProjectionKind.ASSOCIATED_ARTIFACT and target_state is not None and target_status == TargetCompatibility.TARGET_REVIEW_REQUIRED:
            architecture_findings.append(ArchitectureFinding(path, ArchitectureFindingKind.MISSING_ASSOCIATED_ARTIFACT, "Required source associated-artifact scope is not provably preserved as an associated artifact in accepted target state."))

    for item in resolution["removed"]:
        element_id = item["element_id"]
        coverage = item["coverage"]
        removals.append(
            RemovalMigrationElement(
                f"removal:{element_id}", coverage["declaration_id"],
                coverage["source_classification"], item["selector"],
                "Source declaration selector currently resolves to no repository element; it is not copied forward solely for historical preservation.",
            )
        )
        architecture_findings.append(ArchitectureFinding(element_id, ArchitectureFindingKind.STALE_SOURCE_DECLARATION, "Source selector has no current repository match."))

    for item in resolution["uncovered"]:
        path = item["path"]
        async_targets.append(AsynchronousWorkTarget(f"async:{path}", path))
        architecture_findings.append(ArchitectureFinding(path, ArchitectureFindingKind.NEW_REPOSITORY_CANDIDATE, "Current repository element is outside the source profile's active coverage and is non-blocking asynchronous work."))

    ambiguous = tuple(AmbiguousSourceElement.from_native(item) for item in resolution["ambiguous"])
    for item in ambiguous:
        architecture_findings.append(ArchitectureFinding(item.path, ArchitectureFindingKind.AMBIGUOUS_SOURCE_COVERAGE, "Repository element matches multiple source declarations at the controlling specificity; obligation taxonomy is fail-closed."))

    architecture_findings.extend(_relationship_findings(tuple(tuple(item) for item in projected["relationship_semantics"]), target_state))
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
        ambiguous,
        tuple(lifecycle_findings),
        tuple(architecture_findings),
        tuple(issues),
        completion,
    )

