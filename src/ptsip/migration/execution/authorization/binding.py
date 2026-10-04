from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

from ptsip.repository.snapshot import capture_snapshot
from ptsip.migration.analysis.result import MigrationAnalysis
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import MutationGuardExpectation, capture_mutation_guard
from ptsip.migration.execution.guard.repository_snapshot import RepositorySnapshotExpectation, _analysis_snapshot, _current_snapshot_matches_analysis
from ptsip.migration.execution.state.source_steps import ExecutionPhase
from ptsip.migration.planning.result import FinalPointConvergencePlan
from ptsip.migration.proposal.source_set import SourceProposalSet
from ptsip.migration.proposal.target_delta import TargetDelta

@dataclass(frozen=True)
class SourceExecutionBinding:
    source_path: str
    source_content_sha256: str
    analysis_digest: str
    snapshot: RepositorySnapshotExpectation
    accepted_bundles: tuple[AcceptedDeltaBundle, ...]
    required_deltas: tuple[TargetDelta, ...]
    async_deltas: tuple[TargetDelta, ...]
    next_source_path: str | None

    @property
    def decision_ids(self) -> tuple[str, ...]:
        return tuple(sorted({item.decision_id for item in self.accepted_bundles}))

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "source_content_sha256": self.source_content_sha256,
            "analysis_digest": self.analysis_digest,
            "snapshot": self.snapshot.as_dict(),
            "accepted_bundle_ids": [item.proposal_id for item in self.accepted_bundles],
            "required_delta_ids": [item.id for item in self.required_deltas],
            "async_delta_ids": [item.id for item in self.async_deltas],
            "decision_ids": list(self.decision_ids),
            "next_source_path": self.next_source_path,
        }

@dataclass(frozen=True)
class BoundExecutionPlan:
    plan: FinalPointConvergencePlan
    plan_digest: str
    sources: tuple[SourceExecutionBinding, ...]
    mutation_guard: MutationGuardExpectation
    phase: ExecutionPhase = ExecutionPhase.PLAN_BOUND

    def as_dict(self) -> dict[str, object]:
        return {
            "phase": self.phase.value,
            "plan_digest": self.plan_digest,
            "sources": [item.as_dict() for item in self.sources],
            "mutation_guard": self.mutation_guard.as_dict(),
            "final_point": self.plan.final_point.as_dict(),
            "execution_preview": self.plan.preview.as_dict(),
            "projected_final_state_digest": self.plan.projected_final_state_digest,
        }

def _generation_matches(left, right) -> bool:
    return (
        left.profile_path == right.profile_path
        and left.declared_version == right.declared_version
        and left.specification_revision == right.specification_revision
        and left.specification_source == right.specification_source
        and left.content_sha256 == right.content_sha256
        and left.temporary == right.temporary
    )

def bind_execution_plan(
    repository_root: str | Path,
    plan: FinalPointConvergencePlan,
    analyses_by_source: Mapping[str, MigrationAnalysis],
    proposals_by_source: Mapping[str, SourceProposalSet],
) -> BoundExecutionPlan:
    if plan.issues:
        raise ExecutionStateError("WU-06 convergence plan contains planning issues.")
    if not plan.preview.ready_for_wu07 or plan.preview.blocking_ids:
        raise ExecutionStateError("WU-06 execution preview is not ready for WU-07.")

    ordered = tuple(item.source_path for item in plan.source_steps)
    if ordered != plan.preview.ordered_sources:
        raise ExecutionStateError("WU-06 source order and execution preview disagree.")

    root = Path(repository_root).expanduser().resolve()
    initial_snapshot = capture_snapshot(root)
    if initial_snapshot.observation_errors:
        raise ExecutionStateError("Repository snapshot is incomplete at WU-07 bind boundary.")

    bindings: list[SourceExecutionBinding] = []
    for step in plan.source_steps:
        analysis = analyses_by_source.get(step.source_path)
        proposal_set = proposals_by_source.get(step.source_path)
        if analysis is None or proposal_set is None:
            raise ExecutionStateError(f"Missing WU-05/WU-06 input for source {step.source_path}.")
        if not analysis.valid or analysis.deterministic_digest != step.analysis_digest:
            raise ExecutionStateError(f"Source analysis for {step.source_path} is invalid or digest-stale.")
        if analysis.source_generation.profile_path != step.source_path:
            raise ExecutionStateError(f"Source analysis path mismatch for {step.source_path}.")
        if analysis.source_generation.content_sha256 != step.source_content_sha256:
            raise ExecutionStateError(f"Source content identity mismatch for {step.source_path}.")
        if not _current_snapshot_matches_analysis(analysis, initial_snapshot):
            raise ExecutionStateError(
                f"Repository changed after accepted analysis for {step.source_path}; WU-07 bind is fail-closed."
            )
        if proposal_set.analysis_digest != step.analysis_digest:
            raise ExecutionStateError(f"Proposal analysis binding mismatch for {step.source_path}.")
        if not _generation_matches(proposal_set.source_generation, analysis.source_generation):
            raise ExecutionStateError(f"Proposal source generation mismatch for {step.source_path}.")
        for bundle in proposal_set.accepted:
            if bundle.analysis_digest != step.analysis_digest or not _generation_matches(
                bundle.source_generation, analysis.source_generation
            ):
                raise ExecutionStateError(
                    f"Accepted bundle {bundle.proposal_id} lost its exact source/analysis binding."
                )
        accepted_ids = tuple(sorted(item.proposal_id for item in proposal_set.accepted))
        if accepted_ids != tuple(sorted(step.accepted_bundle_ids)):
            raise ExecutionStateError(f"Accepted bundle set changed after WU-06 planning for {step.source_path}.")
        if proposal_set.blocking_proposal_ids or proposal_set.blocking_unresolved_ids or proposal_set.issues:
            raise ExecutionStateError(f"Source {step.source_path} still contains blocking proposal state.")

        delta_owner: dict[str, tuple[TargetDelta, bool]] = {}
        for bundle in proposal_set.accepted:
            for delta in bundle.deltas:
                owner = delta_owner.get(delta.id)
                current = (delta, bundle.async_only)
                if owner is not None and owner[1] != current[1]:
                    raise ExecutionStateError(f"Delta {delta.id} is both required and async in accepted bundles.")
                delta_owner[delta.id] = current
        execution_ids = tuple(step.execution_delta_ids)
        missing = [item for item in execution_ids if item not in delta_owner]
        if missing:
            raise ExecutionStateError(
                f"WU-06 execution delta(s) no longer resolve to accepted bundles: {', '.join(missing)}"
            )
        required = tuple(delta_owner[item][0] for item in execution_ids if not delta_owner[item][1])
        async_deltas = tuple(delta_owner[item][0] for item in execution_ids if delta_owner[item][1])
        bindings.append(
            SourceExecutionBinding(
                source_path=step.source_path,
                source_content_sha256=step.source_content_sha256,
                analysis_digest=step.analysis_digest,
                snapshot=_analysis_snapshot(analysis),
                accepted_bundles=tuple(sorted(proposal_set.accepted, key=lambda item: item.proposal_id)),
                required_deltas=required,
                async_deltas=async_deltas,
                next_source_path=step.next_source_path,
            )
        )

    controlled = tuple(sorted(set(ordered + (plan.final_point.path,))))
    guard = capture_mutation_guard(root, controlled)
    return BoundExecutionPlan(plan, plan.deterministic_digest, tuple(bindings), guard)

