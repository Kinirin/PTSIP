from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from ptsip.repository.profile_transition import discover_profile_transition
from ptsip.migration.execution.authorization.preconditions import _guard_matches
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.mutation.delta import _load_yaml
from ptsip.migration.execution.mutation.finalization import _completed_verified
from ptsip.migration.execution.state.promotion import PromotionReadyState
from ptsip.migration.execution.state.source_steps import CanonicalSourceComplete, ExecutionPhase
from ptsip.migration.planning.convergence.final_state import final_point_state_from_mapping

def prepare_promotion(
    repository_root: str | Path,
    canonical: CanonicalSourceComplete,
    ledger: CheckpointLedger,
) -> PromotionReadyState:
    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(canonical.completed)
    bound = verified.authorized.bound
    if normalize_profile_path(verified.source.source_path) != DEFAULT_PROFILE_PATH:
        raise ExecutionStateError("Promotion requires completed canonical source state.")
    if verified.source_index != len(bound.sources) - 1:
        raise ExecutionStateError("Canonical source is not the final WU-06 source.")
    for source in bound.sources[:-1]:
        if profile_path_on_disk(root, source.source_path).exists():
            raise ExecutionStateError(f"Participating temporary source still exists: {source.source_path}")
    final_path = profile_path_on_disk(root, bound.plan.final_point.path)
    canonical_path = profile_path_on_disk(root, DEFAULT_PROFILE_PATH)
    if not final_path.is_file() or not canonical_path.is_file():
        raise ExecutionStateError("Canonical promotion requires both source and Final Point files.")
    canonical_sha = _sha256_file(canonical_path)
    if canonical_sha != verified.source.source_content_sha256:
        raise ExecutionStateError("Canonical source changed after its completion proof.")
    final_sha = _sha256_file(final_path)
    payload = _load_yaml(final_path)
    final_state = final_point_state_from_mapping(payload, path=bound.plan.final_point.path, content_sha256=final_sha)
    if final_state.semantic_digest != bound.plan.projected_final_state_digest:
        raise ExecutionStateError("Final Point does not match WU-06 global projected target state.")
    if (
        final_state.draft_version != bound.plan.final_point.draft_version
        or final_state.specification_revision != bound.plan.final_point.specification_revision
    ):
        raise ExecutionStateError("Final Point target identity changed before promotion.")
    if not _guard_matches(root, bound):
        raise ExecutionStateError("Repository changed outside controlled profile paths before promotion.")
    discovery = discover_profile_transition(root)
    if not discovery.valid or discovery.state is None:
        raise ExecutionStateError("Transition discovery is invalid before promotion.")
    if discovery.state.final_point is None or discovery.state.final_point.path != bound.plan.final_point.path:
        raise ExecutionStateError("WU-01 Final Point selection no longer matches WU-06 plan.")
    remaining = tuple(item.path for item in discovery.state.ordered_sources)
    if remaining != (DEFAULT_PROFILE_PATH,):
        raise ExecutionStateError("Canonical source is not the only remaining migration source.")
    ledger.append(
        phase=ExecutionPhase.GLOBAL_VALIDATION,
        source_path=DEFAULT_PROFILE_PATH,
        source_sha256=canonical_sha,
        final_point_after_sha256=final_sha,
        analysis_digest=verified.source.analysis_digest,
        decision_ids=verified.source.decision_ids,
        payload={"projected_final_state_digest": final_state.semantic_digest},
    )
    ready = PromotionReadyState(canonical, final_sha, canonical_sha)
    ledger.append(
        phase=ExecutionPhase.PROMOTION_READY,
        source_path=DEFAULT_PROFILE_PATH,
        source_sha256=canonical_sha,
        final_point_after_sha256=final_sha,
        decision_ids=verified.source.decision_ids,
        payload={"atomic_replace": True},
    )
    return ready

