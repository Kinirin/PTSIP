from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import TypeAlias

from ..repository.snapshot import RepositorySnapshot
from .planner import FinalPointConvergencePlan
from .proposal import AcceptedDeltaBundle, TargetDelta


class ExecutionPhase(StrEnum):
    PLAN_BOUND = "PLAN_BOUND"
    AUTHORIZED = "AUTHORIZED"
    PRECONDITIONS_VERIFIED = "PRECONDITIONS_VERIFIED"
    FINAL_POINT_APPLIED = "FINAL_POINT_APPLIED"
    SOURCE_REANALYZED = "SOURCE_REANALYZED"
    SOURCE_COMPLETE = "SOURCE_COMPLETE"
    ASYNC_APPLIED = "ASYNC_APPLIED"
    SOURCE_REMOVED = "SOURCE_REMOVED"
    CANONICAL_SOURCE_COMPLETE = "CANONICAL_SOURCE_COMPLETE"
    GLOBAL_VALIDATION = "GLOBAL_VALIDATION"
    PROMOTION_READY = "PROMOTION_READY"
    PROMOTED = "PROMOTED"
    POST_PROMOTION_VERIFIED = "POST_PROMOTION_VERIFIED"
    RECOVERY_REQUIRED = "RECOVERY_REQUIRED"


@dataclass(frozen=True)
class RepositorySnapshotExpectation:
    head: str | None
    status_fingerprint: str
    tracked_content_fingerprint: str

    @classmethod
    def from_snapshot(cls, snapshot: RepositorySnapshot) -> "RepositorySnapshotExpectation":
        return cls(snapshot.head, snapshot.status_fingerprint, snapshot.tracked_content_fingerprint)

    def matches(self, snapshot: RepositorySnapshot) -> bool:
        return (
            self.head == snapshot.head
            and self.status_fingerprint == snapshot.status_fingerprint
            and self.tracked_content_fingerprint == snapshot.tracked_content_fingerprint
            and not snapshot.observation_errors
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "head": self.head,
            "status_fingerprint": self.status_fingerprint,
            "tracked_content_fingerprint": self.tracked_content_fingerprint,
        }


@dataclass(frozen=True)
class MutationGuardExpectation:
    head: str | None
    status_fingerprint: str
    content_fingerprint: str

    def as_dict(self) -> dict[str, object]:
        return {
            "head": self.head,
            "status_fingerprint": self.status_fingerprint,
            "content_fingerprint": self.content_fingerprint,
        }


@dataclass(frozen=True)
class SourceExecutionBinding:
    source_path: str
    source_content_sha256: str
    analysis_digest: str
    snapshot: RepositorySnapshotExpectation
    accepted_bundles: tuple[AcceptedDeltaBundle, ...]
    required_deltas: tuple[TargetDelta, ...]
    async_deltas: tuple[TargetDelta, ...]
    next_source_path: str | None

    @property
    def decision_ids(self) -> tuple[str, ...]:
        return tuple(sorted({item.decision_id for item in self.accepted_bundles}))

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "source_content_sha256": self.source_content_sha256,
            "analysis_digest": self.analysis_digest,
            "snapshot": self.snapshot.as_dict(),
            "accepted_bundle_ids": [item.proposal_id for item in self.accepted_bundles],
            "required_delta_ids": [item.id for item in self.required_deltas],
            "async_delta_ids": [item.id for item in self.async_deltas],
            "decision_ids": list(self.decision_ids),
            "next_source_path": self.next_source_path,
        }


@dataclass(frozen=True)
class BoundExecutionPlan:
    plan: FinalPointConvergencePlan
    plan_digest: str
    sources: tuple[SourceExecutionBinding, ...]
    mutation_guard: MutationGuardExpectation
    phase: ExecutionPhase = ExecutionPhase.PLAN_BOUND

    def as_dict(self) -> dict[str, object]:
        return {
            "phase": self.phase.value,
            "plan_digest": self.plan_digest,
            "sources": [item.as_dict() for item in self.sources],
            "mutation_guard": self.mutation_guard.as_dict(),
            "final_point": self.plan.final_point.as_dict(),
            "execution_preview": self.plan.preview.as_dict(),
            "projected_final_state_digest": self.plan.projected_final_state_digest,
        }


@dataclass(frozen=True)
class AuthorizationProof:
    plan_digest: str
    decision_ids: tuple[str, ...]
    authority_revision: str
    proof_id: str

    def as_dict(self) -> dict[str, object]:
        return {
            "plan_digest": self.plan_digest,
            "decision_ids": list(self.decision_ids),
            "authority_revision": self.authority_revision,
            "proof_id": self.proof_id,
        }


@dataclass(frozen=True)
class AuthorizedExecutionPlan:
    bound: BoundExecutionPlan
    authorization: AuthorizationProof
    phase: ExecutionPhase = ExecutionPhase.AUTHORIZED


@dataclass(frozen=True)
class VerifiedSourceStep:
    authorized: AuthorizedExecutionPlan
    source_index: int
    source: SourceExecutionBinding
    observed_snapshot: RepositorySnapshotExpectation
    final_point_before_sha256: str | None
    phase: ExecutionPhase = ExecutionPhase.PRECONDITIONS_VERIFIED


@dataclass(frozen=True)
class AppliedSourceStep:
    verified: VerifiedSourceStep
    final_point_after_sha256: str
    applied_delta_ids: tuple[str, ...]
    phase: ExecutionPhase = ExecutionPhase.FINAL_POINT_APPLIED


@dataclass(frozen=True)
class SourceCompletionProof:
    source_path: str
    analysis_digest: str
    required_total: int
    required_unresolved: int
    target_valid: bool
    evidence: tuple[str, ...] = ()

    @property
    def complete(self) -> bool:
        return self.required_unresolved == 0 and self.target_valid

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "analysis_digest": self.analysis_digest,
            "required_total": self.required_total,
            "required_unresolved": self.required_unresolved,
            "target_valid": self.target_valid,
            "complete": self.complete,
            "evidence": list(self.evidence),
        }


@dataclass(frozen=True)
class ReanalyzedSourceStep:
    applied: AppliedSourceStep
    proof: SourceCompletionProof
    phase: ExecutionPhase = ExecutionPhase.SOURCE_REANALYZED


@dataclass(frozen=True)
class CompletedSourceStep:
    reanalyzed: ReanalyzedSourceStep
    phase: ExecutionPhase = ExecutionPhase.SOURCE_COMPLETE


@dataclass(frozen=True)
class AsyncAppliedSourceStep:
    completed: CompletedSourceStep
    final_point_after_sha256: str
    applied_delta_ids: tuple[str, ...]
    phase: ExecutionPhase = ExecutionPhase.ASYNC_APPLIED


@dataclass(frozen=True)
class RemovedTemporarySourceStep:
    completed: CompletedSourceStep | AsyncAppliedSourceStep
    removed_source_path: str
    phase: ExecutionPhase = ExecutionPhase.SOURCE_REMOVED


@dataclass(frozen=True)
class CanonicalSourceComplete:
    completed: CompletedSourceStep | AsyncAppliedSourceStep
    phase: ExecutionPhase = ExecutionPhase.CANONICAL_SOURCE_COMPLETE


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


@dataclass(frozen=True)
class RecoveryInspection:
    plan_digest: str
    last_phase: ExecutionPhase | None
    safe_to_resume: bool
    next_source_index: int
    reasons: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "plan_digest": self.plan_digest,
            "last_phase": self.last_phase.value if self.last_phase else None,
            "safe_to_resume": self.safe_to_resume,
            "next_source_index": self.next_source_index,
            "reasons": list(self.reasons),
        }


@dataclass(frozen=True)
class RecoveryRequiredState:
    plan_digest: str
    last_phase: ExecutionPhase | None
    reason: str
    source_path: str | None = None
    phase: ExecutionPhase = ExecutionPhase.RECOVERY_REQUIRED


ExecutionState: TypeAlias = (
    BoundExecutionPlan
    | AuthorizedExecutionPlan
    | VerifiedSourceStep
    | AppliedSourceStep
    | ReanalyzedSourceStep
    | CompletedSourceStep
    | AsyncAppliedSourceStep
    | RemovedTemporarySourceStep
    | CanonicalSourceComplete
    | PromotionReadyState
    | PromotedState
    | PostPromotionVerifiedState
    | RecoveryRequiredState
)
