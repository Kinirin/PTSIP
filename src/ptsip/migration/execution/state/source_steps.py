from __future__ import annotations

from dataclasses import dataclass

from ptsip.migration.execution.authorization.binding import SourceExecutionBinding
from ptsip.migration.execution.authorization.proof import AuthorizedExecutionPlan
from ptsip.migration.execution.state.phase import ExecutionPhase

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

