from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_path import profile_path_on_disk
from ptsip.repository.snapshot import capture_snapshot
from ptsip.migration.execution.authorization.binding import BoundExecutionPlan
from ptsip.migration.execution.authorization.proof import AuthorizedExecutionPlan
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file, capture_mutation_guard
from ptsip.migration.execution.guard.repository_snapshot import RepositorySnapshotExpectation
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.state.phase import ExecutionPhase
from ptsip.migration.execution.state.source_steps import VerifiedSourceStep
from ptsip.migration.planning.convergence.final_state import FinalPointKind

def _guard_matches(root: Path, bound: BoundExecutionPlan) -> bool:
    controlled = tuple(sorted(set(tuple(item.source_path for item in bound.sources) + (bound.plan.final_point.path,))))
    return capture_mutation_guard(root, controlled) == bound.mutation_guard

def _expected_final_point_sha(authorized: AuthorizedExecutionPlan, ledger: CheckpointLedger) -> str | None:
    rows = ledger.read_all()
    for row in reversed(rows):
        if row.final_point_after_sha256:
            return row.final_point_after_sha256
    return authorized.bound.plan.final_point.content_sha256

def verify_source_preconditions(
    repository_root: str | Path,
    authorized: AuthorizedExecutionPlan,
    source_index: int,
    ledger: CheckpointLedger,
) -> VerifiedSourceStep:
    root = Path(repository_root).expanduser().resolve()
    if not 0 <= source_index < len(authorized.bound.sources):
        raise ExecutionStateError("Source index is outside the bound execution plan.")
    source = authorized.bound.sources[source_index]
    removed_count = sum(1 for row in ledger.read_all() if row.phase == ExecutionPhase.SOURCE_REMOVED)
    if source_index != removed_count:
        raise ExecutionStateError("Requested source is not the next source allowed by the checkpoint ledger.")
    if not _guard_matches(root, authorized.bound):
        raise ExecutionStateError("Repository changed outside the WU-07 controlled profile mutation set.")

    source_path = profile_path_on_disk(root, source.source_path)
    if not source_path.is_file() or _sha256_file(source_path) != source.source_content_sha256:
        raise ExecutionStateError("Source profile changed or disappeared after WU-06 planning.")

    final_path = profile_path_on_disk(root, authorized.bound.plan.final_point.path)
    expected_final_sha = _expected_final_point_sha(authorized, ledger)
    if expected_final_sha is None:
        if authorized.bound.plan.final_point.kind == FinalPointKind.EXISTING:
            raise ExecutionStateError("Existing Final Point lacks an expected content SHA.")
        if final_path.exists():
            raise ExecutionStateError("Planned Final Point unexpectedly exists before its guarded creation.")
    else:
        if not final_path.is_file() or _sha256_file(final_path) != expected_final_sha:
            raise ExecutionStateError("Final Point content changed from the exact expected checkpoint state.")

    snapshot = RepositorySnapshotExpectation.from_snapshot(capture_snapshot(root))
    ledger.append(
        phase=ExecutionPhase.PRECONDITIONS_VERIFIED,
        source_path=source.source_path,
        source_sha256=source.source_content_sha256,
        final_point_before_sha256=expected_final_sha,
        analysis_digest=source.analysis_digest,
        decision_ids=source.decision_ids,
        repository_snapshot=snapshot,
        payload={"source_index": source_index},
    )
    return VerifiedSourceStep(authorized, source_index, source, snapshot, expected_final_sha)

