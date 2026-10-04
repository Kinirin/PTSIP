from __future__ import annotations

from dataclasses import dataclass

from ptsip.migration.execution.state.phase import ExecutionPhase

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

