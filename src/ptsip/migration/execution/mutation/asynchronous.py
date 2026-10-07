from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_path import profile_path_on_disk
from ptsip.migration.execution.authorization.preconditions import _guard_matches
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.mutation.delta import _apply_delta_batch
from ptsip.migration.execution.state.phase import ExecutionPhase
from ptsip.migration.execution.state.source_steps import AsyncAppliedSourceStep, CompletedSourceStep

def apply_optional_async_deltas(
    repository_root: str | Path,
    completed: CompletedSourceStep,
    ledger: CheckpointLedger,
) -> CompletedSourceStep | AsyncAppliedSourceStep:
    verified = completed.reanalyzed.applied.verified
    if not verified.source.async_deltas:
        return completed
    root = Path(repository_root).expanduser().resolve()
    if not _guard_matches(root, verified.authorized.bound):
        raise ExecutionStateError("Repository changed outside controlled paths before optional Async apply.")
    final_path = profile_path_on_disk(root, verified.authorized.bound.plan.final_point.path)
    if _sha256_file(final_path) != completed.reanalyzed.applied.final_point_after_sha256:
        raise ExecutionStateError("Final Point changed after Required completion and before Async apply.")
    sha, _payload = _apply_delta_batch(final_path, verified.source.async_deltas, verified)
    result = AsyncAppliedSourceStep(completed, sha, tuple(item.id for item in verified.source.async_deltas))
    ledger.append(
        phase=ExecutionPhase.ASYNC_APPLIED,
        source_path=verified.source.source_path,
        source_sha256=verified.source.source_content_sha256,
        final_point_before_sha256=completed.reanalyzed.applied.final_point_after_sha256,
        final_point_after_sha256=sha,
        analysis_digest=completed.reanalyzed.proof.analysis_digest,
        decision_ids=verified.source.decision_ids,
        payload={"applied_delta_ids": list(result.applied_delta_ids), "completion_contribution": False},
    )
    return result

