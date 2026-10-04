from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_convergence import DirectConvergenceMode, discover_direct_profile_convergence
from ptsip.repository.profile_path import DEFAULT_PROFILE_PATH, profile_path_on_disk
from ptsip.migration.execution.authorization.preconditions import _guard_matches
from ptsip.migration.execution.direct.promotion import _direct_plan_profile_contract, _payload_profile_contract
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.mutation import _sha256_file
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.mutation.delta import _load_yaml
from ptsip.migration.execution.mutation.finalization import _completed_verified
from ptsip.migration.execution.state.promotion import PostPromotionVerifiedState, PromotedState
from ptsip.migration.execution.state.recovery import RecoveryRequiredState
from ptsip.migration.execution.state.phase import ExecutionPhase
from ptsip.migration.planning.convergence.final_state import final_point_state_from_mapping

def verify_direct_post_promotion(
    repository_root: str | Path,
    promoted: PromotedState,
    ledger: CheckpointLedger,
) -> PostPromotionVerifiedState | RecoveryRequiredState:
    """Verify canonical promotion through PP-aware convergence discovery."""

    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(promoted.ready.canonical.completed)
    bound = verified.authorized.bound
    profile_contract = _direct_plan_profile_contract(bound)
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
                    reasons.append(
                        "promoted canonical semantics differ from WU-06 projected direct target state"
                    )
                if _payload_profile_contract(canonical_payload) != profile_contract:
                    reasons.append("promoted canonical PP identity differs from bound direct target")
                if canonical_state.specification_revision != bound.plan.final_point.specification_revision:
                    reasons.append("promoted canonical Specification revision differs from bound direct target")
            except (ExecutionStateError, ValueError) as exc:
                reasons.append(f"unable to validate promoted canonical semantics: {exc}")

    if final_path.exists():
        reasons.append("Direct Final Point path still exists after promotion")
    try:
        if not _guard_matches(root, bound):
            reasons.append("repository changed outside controlled profile paths during direct promotion")
    except ExecutionStateError as exc:
        reasons.append(str(exc))

    discovery = discover_direct_profile_convergence(root)
    if not discovery.valid or discovery.state is None:
        detail = "; ".join(
            f"{item.code}: {item.message}" for item in discovery.diagnostics
        )
        reasons.append(
            "direct convergence rediscovery is invalid after promotion"
            + (f": {detail}" if detail else "")
        )
    else:
        state = discovery.state
        if state.mode is not DirectConvergenceMode.CURRENT:
            reasons.append("post-promotion repository is not at CURRENT canonical PP state")
        if state.source.declared_version != profile_contract:
            reasons.append("canonical PP contract does not match promoted Final Point")
        if state.source.specification_revision != bound.plan.final_point.specification_revision:
            reasons.append("canonical Specification revision does not match promoted Final Point")
        if state.target_path != DEFAULT_PROFILE_PATH or state.intermediate_profiles:
            reasons.append("post-promotion direct convergence still exposes a temporary/intermediate target")

    if reasons:
        ledger.append(
            phase=ExecutionPhase.RECOVERY_REQUIRED,
            source_path=DEFAULT_PROFILE_PATH,
            final_point_after_sha256=promoted.canonical_after_sha256,
            payload={
                "reasons": reasons,
                "transition_model": "DIRECT_LATEST_TARGET_CONVERGENCE",
                "profile_contract": profile_contract,
            },
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
        payload={
            "transition_valid": True,
            "transition_model": "DIRECT_LATEST_TARGET_CONVERGENCE",
            "profile_contract": profile_contract,
        },
    )
    return result

