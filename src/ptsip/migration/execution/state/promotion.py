from __future__ import annotations

from dataclasses import dataclass

from ptsip.migration.execution.state.source_steps import CanonicalSourceComplete, ExecutionPhase

@dataclass(frozen=True)
class PromotionReadyState:
    canonical: CanonicalSourceComplete
    final_point_sha256: str
    canonical_before_sha256: str
    phase: ExecutionPhase = ExecutionPhase.PROMOTION_READY

@dataclass(frozen=True)
class PromotedState:
    ready: PromotionReadyState
    canonical_after_sha256: str
    phase: ExecutionPhase = ExecutionPhase.PROMOTED

@dataclass(frozen=True)
class PostPromotionVerifiedState:
    promoted: PromotedState
    phase: ExecutionPhase = ExecutionPhase.POST_PROMOTION_VERIFIED

