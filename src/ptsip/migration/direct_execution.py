from __future__ import annotations

from pathlib import Path

from ..profile_compatibility import require_direct_historical_transition
from ..repository.profile_convergence import (
    DirectConvergenceMode,
    discover_direct_profile_convergence,
)
from ..repository.profile_path import DEFAULT_PROFILE_PATH, profile_path_on_disk
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
from .identity_rewrite import IdentityRewriteError, IdentityRewritePlan
from .planner import final_point_state_from_mapping
from ..repository.profile_convergence import DirectConvergenceState


def build_legacy_target_identity_rewrite_plan(
    state: DirectConvergenceState,
) -> IdentityRewritePlan:
    """Bind an existing historical target alias to its logical current PP identity.

    This is continuity normalization, not an intermediate migration hop.  The
    physical target path is preserved while only its historical contract identity
    is rewritten before the WU-07 semantic execution plan is bound.
    """

    if state.mode is not DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION:
        raise IdentityRewriteError(
            "PP_LEGACY_TARGET_REWRITE_NOT_APPLICABLE",
            "Legacy target normalization requires a direct semantic convergence state.",
        )
    if not state.target_is_legacy_alias or state.target is None:
        raise IdentityRewriteError(
            "PP_LEGACY_TARGET_ALIAS_REQUIRED",
            "Legacy target normalization requires one discovered historical target alias.",
        )

    target = state.target
    bridge = require_direct_historical_transition(
        target.declared_version,
        target.specification_revision,
        state.target_contract,
    )
    if bridge.legacy_target_filename != target.path:
        raise IdentityRewriteError(
            "PP_LEGACY_TARGET_ALIAS_MISMATCH",
            (
                f"Historical target {target.path!r} is not the registered legacy alias "
                f"for {state.target_contract.canonical!r}."
            ),
        )

    return IdentityRewritePlan(
        source_path=target.path,
        source_sha256=target.content_sha256,
        source_declared_version=target.declared_version,
        target_contract=state.target_contract.canonical,
        specification_revision=target.specification_revision,
        repository_snapshot=state.snapshot,
    )


def _require_direct_pre_promotion_state(root: Path, canonical: CanonicalSourceComplete) -> None:
    verified = _completed_verified(canonical.completed)
    bound = verified.authorized.bound
    discovery = discover_direct_profile_convergence(root)
    if not discovery.valid or discovery.state is None:
        detail = "; ".join(
            f"{item.code}: {item.message}" for item in discovery.diagnostics
        )
        raise ExecutionStateError(
            "Direct PP convergence discovery is invalid before promotion"
            + (f": {detail}" if detail else ".")
        )
    state = discovery.state
    if state.mode is not DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION:
        raise ExecutionStateError(
            "Direct PP promotion requires a DIRECT_SEMANTIC_MIGRATION repository state."
        )
    if state.source.path != DEFAULT_PROFILE_PATH:
        raise ExecutionStateError("Direct PP promotion source is not the canonical ptsip.yaml.")
    if state.target_contract.canonical != bound.plan.final_point.draft_version:
        raise ExecutionStateError("Direct PP target contract no longer matches the bound WU-06 plan.")
    if state.target_path != bound.plan.final_point.path or state.target is None:
        raise ExecutionStateError("Direct PP Final Point selection no longer matches the bound WU-06 plan.")
    if state.intermediate_profiles:
        raise ExecutionStateError("Direct PP promotion must not materialize intermediate Project Profiles.")


def prepare_direct_promotion(
    repository_root: str | Path,
    canonical: CanonicalSourceComplete,
    ledger: CheckpointLedger,
) -> PromotionReadyState:
    """WU-07 promotion gate using WU-10 PP-aware direct convergence authority."""

    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(canonical.completed)
    bound = verified.authorized.bound
    if verified.source.source_path != DEFAULT_PROFILE_PATH:
        raise ExecutionStateError("Direct PP promotion requires completed canonical source state.")
    if verified.source_index != len(bound.sources) - 1:
        raise ExecutionStateError("Canonical source is not the final direct-convergence source.")
    if len(bound.sources) != 1:
        raise ExecutionStateError(
            "Direct latest-target convergence must bind exactly one actual source, not an intermediate chain."
        )

    final_path = profile_path_on_disk(root, bound.plan.final_point.path)
    canonical_path = profile_path_on_disk(root, DEFAULT_PROFILE_PATH)
    if not final_path.is_file() or not canonical_path.is_file():
        raise ExecutionStateError("Direct canonical promotion requires both source and Final Point files.")

    canonical_sha = _sha256_file(canonical_path)
    if canonical_sha != verified.source.source_content_sha256:
        raise ExecutionStateError("Canonical source changed after its completion proof.")
    final_sha = _sha256_file(final_path)
    payload = _load_yaml(final_path)
    final_state = final_point_state_from_mapping(
        payload,
        path=bound.plan.final_point.path,
        content_sha256=final_sha,
    )
    if final_state.semantic_digest != bound.plan.projected_final_state_digest:
        raise ExecutionStateError("Direct Final Point does not match the WU-06 projected target state.")
    if (
        final_state.draft_version != bound.plan.final_point.draft_version
        or final_state.specification_revision != bound.plan.final_point.specification_revision
    ):
        raise ExecutionStateError("Direct Final Point target identity changed before promotion.")
    if not _guard_matches(root, bound):
        raise ExecutionStateError("Repository changed outside controlled profile paths before direct promotion.")

    _require_direct_pre_promotion_state(root, canonical)

    ledger.append(
        phase=ExecutionPhase.GLOBAL_VALIDATION,
        source_path=DEFAULT_PROFILE_PATH,
        source_sha256=canonical_sha,
        final_point_after_sha256=final_sha,
        analysis_digest=verified.source.analysis_digest,
        decision_ids=verified.source.decision_ids,
        payload={
            "projected_final_state_digest": final_state.semantic_digest,
            "transition_model": "DIRECT_LATEST_TARGET_CONVERGENCE",
        },
    )
    ready = PromotionReadyState(canonical, final_sha, canonical_sha)
    ledger.append(
        phase=ExecutionPhase.PROMOTION_READY,
        source_path=DEFAULT_PROFILE_PATH,
        source_sha256=canonical_sha,
        final_point_after_sha256=final_sha,
        decision_ids=verified.source.decision_ids,
        payload={"atomic_replace": True, "intermediate_profiles": []},
    )
    return ready


def verify_direct_post_promotion(
    repository_root: str | Path,
    promoted: PromotedState,
    ledger: CheckpointLedger,
) -> PostPromotionVerifiedState | RecoveryRequiredState:
    """Verify canonical promotion through PP-aware convergence discovery."""

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
                    reasons.append(
                        "promoted canonical semantics differ from WU-06 projected direct target state"
                    )
                if canonical_state.draft_version != bound.plan.final_point.draft_version:
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
        if state.source.declared_version != bound.plan.final_point.draft_version:
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
            payload={"reasons": reasons, "transition_model": "DIRECT_LATEST_TARGET_CONVERGENCE"},
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
        payload={"transition_valid": True, "transition_model": "DIRECT_LATEST_TARGET_CONVERGENCE"},
    )
    return result
