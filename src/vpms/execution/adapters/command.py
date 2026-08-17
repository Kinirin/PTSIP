from __future__ import annotations

import subprocess
from dataclasses import dataclass

from ...domain.model import VerificationCase, VerificationOutcome
from ..runner import RunnerExecution


COMMAND_NONZERO_EXIT = "VPMS-COMMAND-NONZERO-EXIT"
COMMAND_EXECUTION_ERROR = "VPMS-COMMAND-EXECUTION-ERROR"


@dataclass(frozen=True)
class CommandExecutor:
    """Execute one configured argv command through the framework-neutral runner contract."""

    argv: tuple[str, ...]
    cwd: str | None = None
    timeout_seconds: float | None = None

    def __post_init__(self) -> None:
        if not isinstance(self.argv, tuple) or not self.argv:
            raise ValueError("command argv must be a non-empty tuple of strings.")
        if not all(isinstance(argument, str) for argument in self.argv):
            raise ValueError("command argv must be a non-empty tuple of strings.")
        if not self.argv[0]:
            raise ValueError("command executable must be a non-empty string.")
        if self.cwd is not None and (not isinstance(self.cwd, str) or not self.cwd):
            raise ValueError("command cwd must be a non-empty string when provided.")
        if self.timeout_seconds is not None:
            if (
                isinstance(self.timeout_seconds, bool)
                or not isinstance(self.timeout_seconds, (int, float))
                or self.timeout_seconds <= 0
            ):
                raise ValueError("command timeout_seconds must be positive when provided.")

    def execute(self, case: VerificationCase) -> RunnerExecution:
        """Run the configured command without inferring or mutating Verification Case state."""

        _ = case
        try:
            completed = subprocess.run(
                self.argv,
                cwd=self.cwd,
                timeout=self.timeout_seconds,
                check=False,
                shell=False,
                capture_output=True,
                text=True,
            )
        except (OSError, subprocess.SubprocessError) as exc:
            return RunnerExecution(
                outcome=VerificationOutcome.ERROR,
                diagnostics=(COMMAND_EXECUTION_ERROR,),
                failure_detail=f"{type(exc).__name__}: {exc}",
            )

        if completed.returncode == 0:
            return RunnerExecution(outcome=VerificationOutcome.PASS)

        return RunnerExecution(
            outcome=VerificationOutcome.FAIL,
            diagnostics=(COMMAND_NONZERO_EXIT,),
            failure_detail=f"Command exited with status {completed.returncode}.",
        )
