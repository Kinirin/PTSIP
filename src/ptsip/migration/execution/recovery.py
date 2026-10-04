from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from ptsip.migration.execution.authorization.binding import BoundExecutionPlan
from ptsip.migration.execution.authorization.preconditions import _guard_matches
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.record import LedgerIntegrityError
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.state.recovery import RecoveryInspection
from ptsip.migration.execution.state.phase import ExecutionPhase

def inspect_recovery(
    repository_root: str | Path,
    bound: BoundExecutionPlan,
    ledger: CheckpointLedger,
) -> RecoveryInspection:
    root = Path(repository_root).expanduser().resolve()
    try:
        rows = ledger.read_all()
    except LedgerIntegrityError as exc:
        return RecoveryInspection(bound.plan_digest, None, False, 0, (str(exc),))
    if not rows:
        return RecoveryInspection(bound.plan_digest, None, True, 0, ())
    latest = rows[-1]
    reasons: list[str] = []
    if latest.plan_digest != bound.plan_digest:
        reasons.append("ledger plan digest differs from bound WU-06 plan")
    try:
        if not _guard_matches(root, bound):
            reasons.append("repository changed outside controlled profile paths")
    except ExecutionStateError as exc:
        reasons.append(str(exc))

    final_path = profile_path_on_disk(root, bound.plan.final_point.path)
    promoted_phases = {ExecutionPhase.PROMOTED, ExecutionPhase.POST_PROMOTION_VERIFIED}
    if latest.phase in promoted_phases:
        canonical = profile_path_on_disk(root, DEFAULT_PROFILE_PATH)
        if latest.final_point_after_sha256 and (
            not canonical.is_file() or _sha256_file(canonical) != latest.final_point_after_sha256
        ):
            reasons.append("canonical content does not match promoted checkpoint")
        if final_path.exists():
            reasons.append("Final Point unexpectedly exists after promoted checkpoint")
    else:
        expected_final_sha: str | None = None
        for row in reversed(rows):
            if row.final_point_after_sha256:
                expected_final_sha = row.final_point_after_sha256
                break
        if expected_final_sha is None:
            expected_final_sha = bound.plan.final_point.content_sha256
        if expected_final_sha is None:
            if final_path.exists():
                reasons.append("planned Final Point exists without a persisted mutation checkpoint")
        elif not final_path.is_file() or _sha256_file(final_path) != expected_final_sha:
            reasons.append("Final Point content does not match latest persisted checkpoint state")

    removed_sources = [row.source_path for row in rows if row.phase == ExecutionPhase.SOURCE_REMOVED and row.source_path]
    removed_source_set = set(removed_sources)
    for source in bound.sources:
        source_path = profile_path_on_disk(root, source.source_path)
        if source.source_path in removed_source_set:
            if source_path.exists():
                reasons.append(f"source reappeared after removal checkpoint: {source.source_path}")
            continue
        if normalize_profile_path(source.source_path) == DEFAULT_PROFILE_PATH and latest.phase in promoted_phases:
            continue
        if not source_path.is_file():
            reasons.append(f"source disappeared without removal checkpoint: {source.source_path}")
            continue
        if _sha256_file(source_path) != source.source_content_sha256:
            reasons.append(f"source content does not match bound identity: {source.source_path}")

    next_index = len(removed_sources)
    if latest.phase in {
        ExecutionPhase.CANONICAL_SOURCE_COMPLETE,
        ExecutionPhase.GLOBAL_VALIDATION,
        ExecutionPhase.PROMOTION_READY,
        ExecutionPhase.PROMOTED,
        ExecutionPhase.POST_PROMOTION_VERIFIED,
    }:
        next_index = len(bound.sources)
    return RecoveryInspection(
        bound.plan_digest,
        latest.phase,
        not reasons and latest.phase != ExecutionPhase.RECOVERY_REQUIRED,
        next_index,
        tuple(reasons),
    )

