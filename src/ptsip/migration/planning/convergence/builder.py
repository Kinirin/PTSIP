from __future__ import annotations

from typing import Iterable

from ptsip.repository.profile_transition import DraftVersion, ProfileTransitionState
from ptsip.migration.analysis.result import MigrationAnalysis
from ptsip.migration.planning.convergence.final_state import (
    FinalPointKind,
    FinalPointReference,
    FinalPointStateSnapshot,
)
from ptsip.migration.planning.convergence.reconciliation import (
    DeletionGate,
    ReconciliationResult,
    ReconciliationStatus,
    _state_with_delta,
    reconcile_delta,
)
from ptsip.migration.planning.result import ExecutionPreview, FinalPointConvergencePlan, PlanningIssue, SourceConvergencePlan
from ptsip.migration.proposal.semantic_identity import semantic_digest
from ptsip.migration.proposal.source_set import SourceProposalSet

def _generation_matches(identity, binding) -> bool:
    return (
        identity.path == binding.profile_path
        and identity.declared_version == binding.declared_version
        and identity.specification_revision == binding.specification_revision
        and identity.content_sha256 == binding.content_sha256
    )

def _final_point_reference(
    transition: ProfileTransitionState,
    target_draft_version: str,
    target_specification_revision: str,
) -> tuple[FinalPointReference, list[PlanningIssue]]:
    issues: list[PlanningIssue] = []
    parsed = DraftVersion.from_draft_label(target_draft_version)
    if parsed is None:
        return (
            FinalPointReference(
                FinalPointKind.PLANNED,
                "<invalid>",
                target_draft_version,
                target_specification_revision,
                None,
            ),
            [PlanningIssue("INVALID_TARGET_DRAFT", "Target draft must be a <major>.<minor>.<micro>-draft label.")],
        )

    if transition.final_point is None:
        return (
            FinalPointReference(
                FinalPointKind.PLANNED,
                f"ptsip_{parsed.semantic}.yaml",
                target_draft_version,
                target_specification_revision,
                None,
            ),
            issues,
        )

    final_point = transition.final_point
    if final_point.declared_version != target_draft_version:
        issues.append(
            PlanningIssue(
                "FINAL_POINT_DRAFT_MISMATCH",
                f"Existing Final Point {final_point.declared_version!r} does not match requested target {target_draft_version!r}.",
                final_point.path,
            )
        )
    if final_point.specification_revision != target_specification_revision:
        issues.append(
            PlanningIssue(
                "FINAL_POINT_REVISION_MISMATCH",
                "Existing Final Point revision does not match requested target revision.",
                final_point.path,
            )
        )
    return (
        FinalPointReference(
            FinalPointKind.EXISTING,
            final_point.path,
            final_point.declared_version,
            final_point.specification_revision,
            final_point.content_sha256,
        ),
        issues,
    )

def build_final_point_convergence_plan(
    transition: ProfileTransitionState,
    analyses: Iterable[MigrationAnalysis],
    proposal_sets: Iterable[SourceProposalSet],
    *,
    target_draft_version: str,
    target_specification_revision: str,
    final_point_state: FinalPointStateSnapshot | None = None,
) -> FinalPointConvergencePlan:
    final_ref, issues = _final_point_reference(
        transition,
        target_draft_version,
        target_specification_revision,
    )

    if final_ref.kind == FinalPointKind.EXISTING:
        if final_point_state is None:
            issues.append(
                PlanningIssue(
                    "FINAL_POINT_STATE_REQUIRED",
                    "Existing Final Point requires an exact semantic state snapshot for reconciliation.",
                    final_ref.path,
                )
            )
        else:
            if (
                final_point_state.path != final_ref.path
                or final_point_state.draft_version != final_ref.draft_version
                or final_point_state.specification_revision != final_ref.specification_revision
            ):
                issues.append(
                    PlanningIssue(
                        "FINAL_POINT_STATE_MISMATCH",
                        "Final Point semantic snapshot identity does not match WU-01 Final Point identity.",
                        final_ref.path,
                    )
                )
            if final_ref.content_sha256 and final_point_state.content_sha256 != final_ref.content_sha256:
                issues.append(
                    PlanningIssue(
                        "FINAL_POINT_CONTENT_STALE",
                        "Final Point content SHA does not match WU-01 identity.",
                        final_ref.path,
                    )
                )
    elif final_point_state is not None:
        issues.append(
            PlanningIssue(
                "PLANNED_FINAL_POINT_ALREADY_HAS_STATE",
                "A planned Final Point must not be represented as pre-existing accepted state.",
                final_ref.path,
            )
        )

    working_state = (
        final_point_state
        if final_point_state is not None
        else FinalPointStateSnapshot(
            final_ref.path,
            final_ref.draft_version,
            final_ref.specification_revision,
            None,
            (),
        )
    )

    analysis_map = {item.source_generation.profile_path: item for item in analyses}
    proposal_map = {item.source_generation.profile_path: item for item in proposal_sets}
    ordered = list(transition.ordered_sources)
    steps: list[SourceConvergencePlan] = []
    blocking: list[str] = []

    for index, identity in enumerate(ordered):
        analysis = analysis_map.get(identity.path)
        proposal_set = proposal_map.get(identity.path)
        if analysis is None:
            issues.append(
                PlanningIssue(
                    "SOURCE_ANALYSIS_MISSING",
                    "WU-05 analysis is missing for ordered source.",
                    identity.path,
                )
            )
            continue
        if not _generation_matches(identity, analysis.source_generation):
            issues.append(
                PlanningIssue(
                    "SOURCE_ANALYSIS_IDENTITY_MISMATCH",
                    "WU-05 source identity does not match WU-01 ordered source.",
                    identity.path,
                )
            )
        if proposal_set is None:
            issues.append(
                PlanningIssue(
                    "SOURCE_PROPOSAL_SET_MISSING",
                    "WU-06 proposal set is missing for ordered source.",
                    identity.path,
                )
            )
            continue
        if proposal_set.analysis_digest != analysis.deterministic_digest:
            issues.append(
                PlanningIssue(
                    "PROPOSAL_ANALYSIS_DIGEST_MISMATCH",
                    "Proposal set is not bound to current WU-05 deterministic analysis.",
                    identity.path,
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

        unplanned_required = tuple(
            sorted(
                item.id
                for item in analysis.required
                if not item.resolved and item.id not in accepted_refs
            )
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
        blocking.extend(f"{identity.path}:{item}" for item in source_blocking)

        if source_blocking:
            deletion_gate = DeletionGate.BLOCKED
        elif analysis.completion.complete and not required_delta_ids:
            deletion_gate = DeletionGate.ALREADY_ELIGIBLE
        else:
            deletion_gate = DeletionGate.REQUIRES_POST_APPLY_VERIFICATION

        steps.append(
            SourceConvergencePlan(
                source_path=identity.path,
                source_content_sha256=identity.content_sha256,
                analysis_digest=analysis.deterministic_digest,
                required_total=analysis.completion.required_total,
                required_unresolved_before=analysis.completion.required_unresolved,
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
                next_source_path=ordered[index + 1].path if index + 1 < len(ordered) else None,
                projected_final_state_digest=working_state.semantic_digest,
            )
        )

    blocking.extend(
        f"issue:{item.code}:{item.source_path or ''}"
        for item in issues
    )
    step_rows = tuple(steps)
    ready = (
        not blocking
        and len(step_rows) == len(ordered)
        and all(item.deletion_gate != DeletionGate.BLOCKED for item in step_rows)
    )
    preview = ExecutionPreview(
        final_point=final_ref,
        ordered_sources=tuple(item.path for item in ordered),
        source_steps=step_rows,
        blocking_ids=tuple(sorted(set(blocking))),
        ready_for_wu07=ready,
    )
    return FinalPointConvergencePlan(
        final_point=final_ref,
        source_steps=step_rows,
        issues=tuple(sorted(issues, key=lambda item: (item.code, item.source_path or "", item.message))),
        preview=preview,
        projected_final_state_digest=working_state.semantic_digest,
    )

