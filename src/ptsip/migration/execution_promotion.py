from __future__ import annotations

import os
from pathlib import Path

from ..repository.profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from ..repository.profile_transition import discover_profile_transition
from .execution_apply import _completed_verified, _load_yaml
from .execution_binding import ExecutionStateError, _guard_matches, _sha256_file
from .execution_ledger import CheckpointLedger
from .execution_model import (
    CanonicalSourceComplete,
    ExecutionPhase,
    PostPromotionVerifiedState,
    PromotedState,
    PromotionReadyState,
    RecoveryRequiredState,
)
from .planner import final_point_state_from_mapping


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


def verify_post_promotion(
    repository_root: str | Path,
    promoted: PromotedState,
    ledger: CheckpointLedger,
) -> PostPromotionVerifiedState | RecoveryRequiredState:
    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(promoted.ready.canonical.completed)
    bound = verified.authorized.bound
    canonical_path = profile_path_on_disk(root, DEFAULT_PROFILE_PATH)
    final_path = profile_path_on_disk(root, bound.plan.final_point.path)
    reasons: list[str] = []
    if not canonical_path.is_file() or _sha256_file(canonical_path) != promoted.canonical_after_sha256:
        reasons.append("promoted canonical content changed or disappeared")
    if final_path.exists():
        reasons.append("Final Point path still exists after promotion")
    discovery = discover_profile_transition(root)
    if not discovery.valid or discovery.state is None:
        reasons.append("transition rediscovery is invalid after promotion")
    elif discovery.state.final_point is not None:
        reasons.append("post-promotion transition still reports a Final Point")
    else:
        canonical = discovery.state.canonical_source
        if canonical.declared_version != bound.plan.final_point.draft_version:
            reasons.append("canonical target draft does not match promoted Final Point")
        if canonical.specification_revision != bound.plan.final_point.specification_revision:
            reasons.append("canonical specification revision does not match promoted Final Point")
    if reasons:
        ledger.append(
            phase=ExecutionPhase.RECOVERY_REQUIRED,
            source_path=DEFAULT_PROFILE_PATH,
            final_point_after_sha256=promoted.canonical_after_sha256,
            payload={"reasons": reasons},
        )
        return RecoveryRequiredState(
            bound.plan_digest,
            ExecutionPhase.PROMOTED,
            "; ".join(reasons),
            DEFAULT_PROFILE_PATH,
        )
    result = PostPromotionVerifiedState(promoted)
    ledger.append(
        phase=ExecutionPhase.POST_PROMOTION_VERIFIED,
        source_path=DEFAULT_PROFILE_PATH,
        final_point_after_sha256=promoted.canonical_after_sha256,
        payload={"transition_valid": True},
    )
    return result
