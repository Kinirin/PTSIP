from __future__ import annotations

from pathlib import Path

from ptsip.evidence.contract import NormalizedEvidenceSet
from ptsip.repository.snapshot import capture_snapshot, compare_snapshots, repository_files
from ptsip.source_compat.model import CompatibilitySourceProfile
from ptsip.migration.analysis.findings.architecture import ArchitectureFinding, ArchitectureFindingKind
from ptsip.migration.analysis.findings.lifecycle import LifecycleFinding
from ptsip.migration.analysis.findings.work_requirements import (
    AsynchronousWorkTarget,
    RemovalMigrationElement,
    RequiredWorkElement,
    SourceMigrationCompletion,
)
from ptsip.migration.analysis.issue import MigrationAnalysisIssue
from ptsip.migration.analysis.result import MigrationAnalysis
from ptsip.migration.analysis.source.binding_validation import _validate_evidence_context, _validate_source_binding
from ptsip.migration.analysis.source.evidence_correlation import _evidence_for_path
from ptsip.migration.analysis.source.projection import SourceProjectionKind, _project_source
from ptsip.migration.analysis.source.repository_resolution import _resolve_repository
from ptsip.migration.analysis.target.compatibility import TargetCompatibility, _target_compatibility
from ptsip.migration.analysis.target.relationship_findings import _relationship_findings
from ptsip.migration.analysis.target.semantics import (
    TargetArchitectureState,
    TargetSemantics,
    default_target_semantics,
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

