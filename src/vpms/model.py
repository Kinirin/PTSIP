from __future__ import annotations

from dataclasses import asdict, dataclass
from enum import StrEnum


class VerificationPurpose(StrEnum):
    PRODUCT = "PRODUCT"
    TOOLCHAIN = "TOOLCHAIN"


class VerificationOutcome(StrEnum):
    PASS = "PASS"
    FAIL = "FAIL"
    ERROR = "ERROR"
    SKIPPED = "SKIPPED"


@dataclass(frozen=True)
class TargetRef:
    component_id: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class FormulaRef:
    ref: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class VariablesRef:
    ref: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class PolicyRef:
    ref: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class RunnerRef:
    ref: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class VerificationCase:
    id: str
    purpose: VerificationPurpose
    target: TargetRef
    formula: FormulaRef
    variables: VariablesRef
    policy: PolicyRef
    runner: RunnerRef

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["purpose"] = self.purpose.value
        return payload


@dataclass(frozen=True)
class VerificationResult:
    case_id: str
    purpose: VerificationPurpose
    target: TargetRef
    outcome: VerificationOutcome
    diagnostics: tuple[str, ...] = ()
    failure_detail: str | None = None

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["purpose"] = self.purpose.value
        payload["outcome"] = self.outcome.value
        return payload
