from __future__ import annotations

from ptsip.migration.execution.state.phase import ExecutionPhase
from pathlib import Path

from ptsip.repository.profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from ptsip.repository.profile_transition import discover_profile_transition
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.mutation.delta import _load_yaml
from ptsip.migration.execution.state.source_steps import (
    AsyncAppliedSourceStep,
    CanonicalSourceComplete,
    CompletedSourceStep,
    RemovedTemporarySourceStep,
    VerifiedSourceStep,
)
from ptsip.migration.planning.convergence.final_state import final_point_state_from_mapping

def _completed_verified(state: CompletedSourceStep | AsyncAppliedSourceStep) -> VerifiedSourceStep:
    if isinstance(state, AsyncAppliedSourceStep):
        return state.completed.reanalyzed.applied.verified
    return state.reanalyzed.applied.verified

def _current_final_sha(state: CompletedSourceStep | AsyncAppliedSourceStep) -> str:
    if isinstance(state, AsyncAppliedSourceStep):
        return state.final_point_after_sha256
    return state.reanalyzed.applied.final_point_after_sha256

def _validate_projected_final_state(root: Path, state: CompletedSourceStep | AsyncAppliedSourceStep) -> str:
    verified = _completed_verified(state)
    final_path = profile_path_on_disk(root, verified.authorized.bound.plan.final_point.path)
    current_sha = _current_final_sha(state)
    if not final_path.is_file() or _sha256_file(final_path) != current_sha:
        raise ExecutionStateError("Final Point changed after source completion.")
    payload = _load_yaml(final_path)
    snapshot = final_point_state_from_mapping(
        payload,
        path=verified.authorized.bound.plan.final_point.path,
        content_sha256=current_sha,
    )
    step_plan = verified.authorized.bound.plan.source_steps[verified.source_index]
    if snapshot.semantic_digest != step_plan.projected_final_state_digest:
        raise ExecutionStateError("Actual Final Point semantics do not match WU-06 projected state for this source.")
    return current_sha

def finalize_source(
    repository_root: str | Path,
    completed: CompletedSourceStep | AsyncAppliedSourceStep,
    ledger: CheckpointLedger,
) -> RemovedTemporarySourceStep | CanonicalSourceComplete:
    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(completed)
    source = verified.source
    current_final_sha = _validate_projected_final_state(root, completed)
    source_path = profile_path_on_disk(root, source.source_path)
    if not source_path.is_file() or _sha256_file(source_path) != source.source_content_sha256:
        raise ExecutionStateError("Source changed before completion finalization.")
    if normalize_profile_path(source.source_path) == DEFAULT_PROFILE_PATH:
        result = CanonicalSourceComplete(completed)
        ledger.append(
            phase=ExecutionPhase.CANONICAL_SOURCE_COMPLETE,
            source_path=source.source_path,
            source_sha256=source.source_content_sha256,
            final_point_after_sha256=current_final_sha,
            analysis_digest=verified.source.analysis_digest,
            decision_ids=source.decision_ids,
            payload={"canonical_deleted": False},
        )
        return result

    source_path.unlink()
    if source_path.exists():
        raise ExecutionStateError("Temporary source deletion did not complete.")
    discovery = discover_profile_transition(root)
    if not discovery.valid:
        raise ExecutionStateError("Transition discovery became invalid after temporary source deletion.")
    result = RemovedTemporarySourceStep(completed, source.source_path)
    ledger.append(
        phase=ExecutionPhase.SOURCE_REMOVED,
        source_path=source.source_path,
        source_sha256=source.source_content_sha256,
        final_point_after_sha256=current_final_sha,
        analysis_digest=verified.source.analysis_digest,
        decision_ids=source.decision_ids,
        payload={"next_source_path": source.next_source_path},
    )
    return result

