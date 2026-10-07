from __future__ import annotations

import os
from pathlib import Path

from ptsip.repository.profile_path import DEFAULT_PROFILE_PATH, profile_path_on_disk
from ptsip.migration.execution.authorization.preconditions import _guard_matches
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.mutation.finalization import _completed_verified
from ptsip.migration.execution.state.promotion import PromotedState, PromotionReadyState
from ptsip.migration.execution.state.phase import ExecutionPhase

def promote_canonical(
    repository_root: str | Path,
    ready: PromotionReadyState,
    ledger: CheckpointLedger,
) -> PromotedState:
    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(ready.canonical.completed)
    bound = verified.authorized.bound
    final_path = profile_path_on_disk(root, bound.plan.final_point.path)
    canonical_path = profile_path_on_disk(root, DEFAULT_PROFILE_PATH)
    if _sha256_file(final_path) != ready.final_point_sha256:
        raise ExecutionStateError("Final Point changed after PROMOTION_READY.")
    if _sha256_file(canonical_path) != ready.canonical_before_sha256:
        raise ExecutionStateError("Canonical source changed after PROMOTION_READY.")
    if not _guard_matches(root, bound):
        raise ExecutionStateError("Repository changed outside controlled paths after PROMOTION_READY.")
    os.replace(final_path, canonical_path)
    canonical_after = _sha256_file(canonical_path)
    if canonical_after != ready.final_point_sha256:
        ledger.append(
            phase=ExecutionPhase.RECOVERY_REQUIRED,
            source_path=DEFAULT_PROFILE_PATH,
            source_sha256=ready.canonical_before_sha256,
            final_point_before_sha256=ready.final_point_sha256,
            final_point_after_sha256=canonical_after,
            decision_ids=verified.source.decision_ids,
            payload={"reason": "promoted canonical bytes differ from PROMOTION_READY Final Point"},
        )
        raise ExecutionStateError("Promoted canonical bytes differ from the PROMOTION_READY Final Point.")
    promoted = PromotedState(ready, canonical_after)
    ledger.append(
        phase=ExecutionPhase.PROMOTED,
        source_path=DEFAULT_PROFILE_PATH,
        source_sha256=ready.canonical_before_sha256,
        final_point_before_sha256=ready.final_point_sha256,
        final_point_after_sha256=canonical_after,
        decision_ids=verified.source.decision_ids,
        payload={"final_point_path_removed": not final_path.exists()},
    )
    return promoted

