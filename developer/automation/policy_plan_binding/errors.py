from __future__ import annotations


class PolicyPlanBindingError(RuntimeError):
    """Fail-closed error raised by Policy ↔ Planning binding automation."""

    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code
