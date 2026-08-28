from __future__ import annotations

from dataclasses import dataclass

from .direct_convergence import DirectConvergenceAnalysis
from .planner import (
    DeletionGate,
    ExecutionPreview,
    FinalPointConvergencePlan,
    FinalPointKind,
    FinalPointReference as LegacyFinalPointReference,
    FinalPointStateSnapshot,
    PlanningIssue,
    ReconciliationResult,
    ReconciliationStatus,
    SourceConvergencePlan,
    _state_with_delta,
    reconcile_delta,
)
from .proposal import SourceProposalSet, semantic_digest
from ..repository.profile_convergence import DirectConvergenceMode, DirectConvergenceState


@dataclass(frozen=True)
class DirectFinalPointReference(LegacyFinalPointReference):
    """PP-native view of the shared Final Point reference.

    The WU-01~07 sequential implementation historically named this identity
    ``draft_version``.  Direct latest-target convergence uses an independent
    Project Profile contract such as ``pp.1.01``; its serialized/current API
    therefore exposes ``profile_contract`` instead of leaking the historical
    Tool-numbered draft vocabulary.
    """

    @property
    def profile_contract(self) -> str:
        return self.draft_version

    def as_dict(self) -> dict[str, object]:
        payload = super().as_dict()
        payload["profile_contract"] = payload.pop("draft_version")
        return payload


def _source_matches(state: DirectConvergenceState, analysis) -> bool:
    source = analysis.source_generation
    return (
        source.profile_path == state.source.path
        and source.declared_version == state.source.declared_version
        and source.specification_revision == state.source.specification_revision
        and source.specification_source == state.source.specification_source
        and source.content_sha256 == state.source.content_sha256
        and source.temporary == state.source.temporary
    )


def _logical_final_point(
    state: DirectConvergenceState,
    *,
    target_specification_revision: str,
    final_point_state: FinalPointStateSnapshot | None,
) -> tuple[DirectFinalPointReference, FinalPointStateSnapshot, list[PlanningIssue]]:
    issues: list[PlanningIssue] = []
    revision = target_specification_revision.strip()
    if not revision:
        revision = "<invalid>"
        issues.append(
            PlanningIssue(
                "PP_DIRECT_TARGET_REVISION_REQUIRED",
                "Direct PP target requires an immutable Specification revision.",
                state.target_path,
            )
        )

    if state.target is None:
        final_ref = DirectFinalPointReference(
            FinalPointKind.PLANNED,
            state.target_path,
            state.target_contract.canonical,
            revision,
            None,
        )
        if final_point_state is not None:
            issues.append(
                PlanningIssue(
                    "PP_DIRECT_PLANNED_TARGET_ALREADY_HAS_STATE",
                    "A planned direct-convergence target must not be represented as pre-existing state.",
                    state.target_path,
                )
            )
        working = FinalPointStateSnapshot(
            state.target_path,
            state.target_contract.canonical,
            revision,
            None,
            (),
        )
        return final_ref, working, issues

    final_ref = DirectFinalPointReference(
        FinalPointKind.EXISTING,
        state.target.path,
        state.target_contract.canonical,
        revision,
        state.target.content_sha256,
    )
    if final_point_state is None:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_FINAL_POINT_STATE_REQUIRED",
                "Existing direct-convergence target requires an exact semantic state snapshot.",
                state.target.path,
            )
        )
        working = FinalPointStateSnapshot(
            state.target.path,
            state.target_contract.canonical,
            revision,
            state.target.content_sha256,
            (),
        )
        return final_ref, working, issues

    if final_point_state.path != state.target.path:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_FINAL_POINT_PATH_MISMATCH",
                "Final Point state path does not match direct-convergence discovery.",
                state.target.path,
            )
        )
    if final_point_state.specification_revision != revision:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_FINAL_POINT_REVISION_MISMATCH",
                "Final Point Specification revision does not match the selected direct target revision.",
                state.target.path,
            )
        )
    if (
        final_point_state.content_sha256 is not None
        and final_point_state.content_sha256 != state.target.content_sha256
    ):
        issues.append(
            PlanningIssue(
                "PP_DIRECT_FINAL_POINT_CONTENT_STALE",
                "Final Point content SHA does not match direct-convergence discovery.",
                state.target.path,
            )
        )

    # The physical file may still carry a legacy alias identity.  Reconciliation
    # operates on its accepted entities under the logical current PP target.
    working = FinalPointStateSnapshot(
        state.target.path,
        state.target_contract.canonical,
        revision,
        state.target.content_sha256,
        final_point_state.entities,
    )
    return final_ref, working, issues


def build_direct_final_point_convergence_plan(
    state: DirectConvergenceState,
    direct_analysis: DirectConvergenceAnalysis,
    proposal_set: SourceProposalSet,
    *,
    target_specification_revision: str,
    final_point_state: FinalPointStateSnapshot | None = None,
) -> FinalPointConvergencePlan:
    final_ref, working_state, issues = _logical_final_point(
        state,
        target_specification_revision=target_specification_revision,
        final_point_state=final_point_state,
    )

    if state.mode is not DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_SEMANTIC_PLAN_NOT_APPLICABLE",
                "Semantic direct planner is only valid for DIRECT_SEMANTIC_MIGRATION; identity-only uses its own executor.",
                state.source.path,
            )
        )

    if direct_analysis.target_contract != state.target_contract.canonical:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_ANALYSIS_TARGET_MISMATCH",
                "Direct analysis is bound to a different logical PP target.",
                state.source.path,
            )
        )
    if direct_analysis.target_path != state.target_path:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_ANALYSIS_PATH_MISMATCH",
                "Direct analysis target path differs from discovery.",
                state.source.path,
            )
        )

    analysis = direct_analysis.semantic_analysis
    if analysis is None:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_SEMANTIC_ANALYSIS_REQUIRED",
                "Direct semantic planning requires a WU-05 semantic analysis.",
                state.source.path,
            )
        )
        analysis_digest = "<missing>"
        required_total = 0
        required_unresolved = 0
    else:
        analysis_digest = analysis.deterministic_digest
        required_total = analysis.completion.required_total
        required_unresolved = analysis.completion.required_unresolved
        if not _source_matches(state, analysis):
            issues.append(
                PlanningIssue(
                    "PP_DIRECT_SOURCE_ANALYSIS_IDENTITY_MISMATCH",
                    "WU-05 analysis is not bound to the exact direct-convergence source.",
                    state.source.path,
                )
            )
        if not analysis.valid:
            issues.append(
                PlanningIssue(
                    "PP_DIRECT_SOURCE_ANALYSIS_INVALID",
                    "WU-05 semantic analysis is not valid for direct planning.",
                    state.source.path,
                )
            )

    if proposal_set.source_generation.profile_path != state.source.path:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_PROPOSAL_SOURCE_MISMATCH",
                "Proposal set belongs to a different source path.",
                state.source.path,
            )
        )
    if proposal_set.analysis_digest != analysis_digest:
        issues.append(
            PlanningIssue(
                "PP_DIRECT_PROPOSAL_ANALYSIS_MISMATCH",
                "Proposal set is not bound to the current semantic analysis digest.",
                state.source.path,
            )
        )

    accepted_refs: set[str] = set()
    reconciliations: list[ReconciliationResult] = []
    required_delta_ids: list[str] = []
    async_delta_ids: list[str] = []
    accepted_conflicts: list[str] = []

    for bundle in sorted(proposal_set.accepted, key=lambda item: item.proposal_id):
        for delta in bundle.deltas:
            accepted_refs.update(delta.obligation_ids)
            reconciliation = reconcile_delta(
                delta,
                working_state,
                accepted=True,
                bundle_id=bundle.proposal_id,
            )
            reconciliations.append(reconciliation)
            if reconciliation.status in {
                ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
                ReconciliationStatus.UNRESOLVED,
            }:
                accepted_conflicts.append(delta.id)
            else:
                working_state = _state_with_delta(working_state, delta, reconciliation)
            if bundle.async_only:
                async_delta_ids.append(delta.id)
            else:
                required_delta_ids.append(delta.id)

    for bundle in sorted(proposal_set.suggested, key=lambda item: item.id):
        for delta in bundle.deltas:
            reconciliations.append(
                reconcile_delta(
                    delta,
                    working_state,
                    accepted=False,
                    bundle_id=bundle.id,
                )
            )

    unplanned_required = (
        tuple(
            sorted(
                item.id
                for item in analysis.required
                if not item.resolved and item.id not in accepted_refs
            )
        )
        if analysis is not None
        else ()
    )
    blocking_suggested = tuple(
        sorted(item.id for item in proposal_set.suggested if item.blocking)
    )
    blocking_unresolved = tuple(
        sorted(item.id for item in proposal_set.unresolved if item.blocking)
    )
    proposal_issue_ids = tuple(
        f"proposal-issue:{semantic_digest(item)[:16]}" for item in proposal_set.issues
    )
    source_blocking = tuple(
        sorted(
            set(unplanned_required)
            | set(blocking_suggested)
            | set(blocking_unresolved)
            | set(accepted_conflicts)
            | set(proposal_issue_ids)
        )
    )

    if source_blocking or issues:
        deletion_gate = DeletionGate.BLOCKED
    elif analysis is not None and analysis.completion.complete and not required_delta_ids:
        deletion_gate = DeletionGate.ALREADY_ELIGIBLE
    else:
        deletion_gate = DeletionGate.REQUIRES_POST_APPLY_VERIFICATION

    step = SourceConvergencePlan(
        source_path=state.source.path,
        source_content_sha256=state.source.content_sha256,
        analysis_digest=analysis_digest,
        required_total=required_total,
        required_unresolved_before=required_unresolved,
        accepted_bundle_ids=tuple(
            sorted(item.proposal_id for item in proposal_set.accepted)
        ),
        suggested_bundle_ids=tuple(
            sorted(item.id for item in proposal_set.suggested)
        ),
        unresolved_bundle_ids=tuple(
            sorted(item.id for item in proposal_set.unresolved)
        ),
        unplanned_required_ids=unplanned_required,
        execution_delta_ids=tuple(sorted(required_delta_ids) + sorted(async_delta_ids)),
        reconciliations=tuple(
            sorted(reconciliations, key=lambda item: (item.bundle_id, item.delta_id))
        ),
        deletion_gate=deletion_gate,
        next_source_path=None,
        projected_final_state_digest=working_state.semantic_digest,
    )

    blocking = [f"{state.source.path}:{item}" for item in source_blocking]
    blocking.extend(
        f"issue:{item.code}:{item.source_path or ''}"
        for item in issues
    )
    ready = not blocking and deletion_gate is not DeletionGate.BLOCKED
    preview = ExecutionPreview(
        final_point=final_ref,
        ordered_sources=(state.source.path,),
        source_steps=(step,),
        blocking_ids=tuple(sorted(set(blocking))),
        ready_for_wu07=ready,
    )
    return FinalPointConvergencePlan(
        final_point=final_ref,
        source_steps=(step,),
        issues=tuple(sorted(issues, key=lambda item: (item.code, item.source_path or "", item.message))),
        preview=preview,
        projected_final_state_digest=working_state.semantic_digest,
    )
