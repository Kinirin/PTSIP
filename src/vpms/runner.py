from __future__ import annotations

from dataclasses import asdict, dataclass
from typing import Protocol

from .model import VerificationCase, VerificationOutcome, VerificationResult


RUNNER_CONTRACT_ERROR = "VPMS-RUN-CONTRACT-ERROR"
RUNNER_EXECUTION_ERROR = "VPMS-RUN-EXECUTION-ERROR"


@dataclass(frozen=True)
class RunnerExecution:
    """Framework-neutral execution data returned by a case executor."""

    outcome: VerificationOutcome
    diagnostics: tuple[str, ...] = ()
    failure_detail: str | None = None

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["outcome"] = self.outcome.value
        return payload


class CaseExecutor(Protocol):
    """Minimal adapter contract consumed by the VPMS runner."""

    def execute(self, case: VerificationCase) -> RunnerExecution:
        ...


def _error_execution(*, diagnostic: str, failure_detail: str) -> RunnerExecution:
    return RunnerExecution(
        outcome=VerificationOutcome.ERROR,
        diagnostics=(diagnostic,),
        failure_detail=failure_detail,
    )


def run_case(case: VerificationCase, executor: CaseExecutor) -> VerificationResult:
    """Execute one Verification Case and preserve its canonical identity."""

    try:
        execution = executor.execute(case)
    except Exception as exc:
        execution = _error_execution(
            diagnostic=RUNNER_EXECUTION_ERROR,
            failure_detail=f"{type(exc).__name__}: {exc}",
        )

    if not isinstance(execution, RunnerExecution):
        execution = _error_execution(
            diagnostic=RUNNER_CONTRACT_ERROR,
            failure_detail=(
                "Case executor returned an invalid result type: "
                f"{type(execution).__name__}."
            ),
        )

    return VerificationResult(
        case_id=case.id,
        purpose=case.purpose,
        target=case.target,
        outcome=execution.outcome,
        diagnostics=execution.diagnostics,
        failure_detail=execution.failure_detail,
    )
