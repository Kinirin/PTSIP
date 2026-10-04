from __future__ import annotations

from pathlib import Path

from ptsip.evidence.contract import NormalizedEvidenceSet
from ptsip.repository.profile_convergence import DirectConvergenceMode, DirectConvergenceState, validate_direct_convergence_snapshot
from ptsip.migration.analysis.analyzer import analyze_source_migration
from ptsip.migration.analysis.direct.result import DirectConvergenceAnalysis, DirectConvergenceAnalysisIssue
from ptsip.migration.analysis.direct.source_projection import _read_historical_source
from ptsip.migration.analysis.direct.target_projection import current_pp_target_semantics, _project_existing_target_state

def analyze_direct_profile_convergence(
    repository_root: str | Path,
    state: DirectConvergenceState,
    *,
    evidence: NormalizedEvidenceSet | None = None,
) -> DirectConvergenceAnalysis:
    root = Path(repository_root).expanduser().resolve()
    stale = validate_direct_convergence_snapshot(root, state)
    if stale:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=state.mode is DirectConvergenceMode.IDENTITY_ONLY,
            semantic_analysis=None,
            issues=tuple(
                DirectConvergenceAnalysisIssue(item.code, item.message)
                for item in stale
            ),
        )

    if state.mode is DirectConvergenceMode.CURRENT:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=False,
            semantic_analysis=None,
            issues=(),
        )

    source_profile, source_issues = _read_historical_source(root, state)
    if source_profile is None:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=state.mode is DirectConvergenceMode.IDENTITY_ONLY,
            semantic_analysis=None,
            issues=source_issues,
        )

    if state.mode is DirectConvergenceMode.IDENTITY_ONLY:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=True,
            semantic_analysis=None,
            issues=(),
        )

    if evidence is None:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=False,
            semantic_analysis=None,
            issues=(
                DirectConvergenceAnalysisIssue(
                    "PP_DIRECT_EVIDENCE_REQUIRED",
                    "Semantic direct convergence requires normalized evidence bound to the actual historical source.",
                ),
            ),
        )

    target_state, target_issues = _project_existing_target_state(root, state)
    if target_issues:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=False,
            semantic_analysis=None,
            issues=target_issues,
        )

    analysis = analyze_source_migration(
        root,
        source_profile,
        evidence,
        target_semantics=current_pp_target_semantics(state),
        target_state=target_state,
    )
    return DirectConvergenceAnalysis(
        mode=state.mode,
        source_path=state.source.path,
        source_declared_version=state.source.declared_version,
        source_compatibility_contract=state.source_compatibility_contract.canonical,
        target_contract=state.target_contract.canonical,
        target_path=state.target_path,
        target_is_legacy_alias=state.target_is_legacy_alias,
        identity_rewrite_required=False,
        semantic_analysis=analysis,
        issues=(),
    )

