from typing import TypeAlias

from .promotion import PostPromotionVerifiedState, PromotedState, PromotionReadyState
from .recovery import RecoveryInspection, RecoveryRequiredState
from .source_steps import (
    AppliedSourceStep, AsyncAppliedSourceStep, CanonicalSourceComplete, CompletedSourceStep,
    ExecutionPhase, ReanalyzedSourceStep, RemovedTemporarySourceStep, SourceCompletionProof, VerifiedSourceStep,
)
from ptsip.migration.execution.authorization.binding import BoundExecutionPlan
from ptsip.migration.execution.authorization.proof import AuthorizedExecutionPlan

ExecutionState: TypeAlias = (
    BoundExecutionPlan | AuthorizedExecutionPlan | VerifiedSourceStep | AppliedSourceStep |
    ReanalyzedSourceStep | CompletedSourceStep | AsyncAppliedSourceStep | RemovedTemporarySourceStep |
    CanonicalSourceComplete | PromotionReadyState | PromotedState | PostPromotionVerifiedState | RecoveryRequiredState
)
