from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_path import DEFAULT_PROFILE_PATH, profile_path_on_disk
from ptsip.repository.profile_transition import discover_profile_transition
from ptsip.migration.execution.authorization.preconditions import _guard_matches
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.mutation.delta import _load_yaml
from ptsip.migration.execution.mutation.finalization import _completed_verified
from ptsip.migration.execution.state.promotion import PostPromotionVerifiedState, PromotedState
from ptsip.migration.execution.state.recovery import RecoveryRequiredState
from ptsip.migration.execution.state.phase import ExecutionPhase
from ptsip.migration.planning.convergence.final_state import final_point_state_from_mapping

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
    canonical_sha: str | None = None
    if not canonical_path.is_file():
        reasons.append("promoted canonical content changed or disappeared")
    else:
        canonical_sha = _sha256_file(canonical_path)
        if canonical_sha != promoted.canonical_after_sha256:
            reasons.append("promoted canonical content changed or disappeared")
        else:
            try:
                canonical_payload = _load_yaml(canonical_path)
                canonical_state = final_point_state_from_mapping(
                    canonical_payload,
                    path=DEFAULT_PROFILE_PATH,
                    content_sha256=canonical_sha,
                )
                if canonical_state.semantic_digest != bound.plan.projected_final_state_digest:
                    reasons.append("promoted canonical semantics differ from WU-06 global projected target state")
            except (ExecutionStateError, ValueError) as exc:
                reasons.append(f"unable to validate promoted canonical semantics: {exc}")
    if final_path.exists():
        reasons.append("Final Point path still exists after promotion")
    try:
        if not _guard_matches(root, bound):
            reasons.append("repository changed outside controlled profile paths during promotion")
    except ExecutionStateError as exc:
        reasons.append(str(exc))
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

