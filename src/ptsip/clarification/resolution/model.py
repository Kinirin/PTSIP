from __future__ import annotations

from dataclasses import asdict, dataclass

CLASSIFICATIONS = (
    "PRODUCT",
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
    "NEUTRAL_CONTRACT",
)

CANONICAL_ANSWER_FIELDS = (
    "classification",
    "purpose",
    "shipped",
    "runtime_required",
    "executable",
)

LEGACY_V1_ANSWER_FIELDS = (
    "classification",
    "purpose",
    "shipped",
    "runtime_required",
    "lifecycle_owner",
    "executable",
)

LEGACY_LIFECYCLE_OWNERS = (
    "PRODUCT",
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
    "INDEPENDENT",
)


@dataclass(frozen=True)
class DecisionAnswer:
    """Canonical ptsip-clarification-answer/v2 decision facts."""

    classification: str
    purpose: str
    shipped: bool
    runtime_required: bool
    executable: bool

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class LegacyDecisionAnswerV1:
    """Explicit compatibility shape for persisted/received v1 decisions only."""

    classification: str
    purpose: str
    shipped: bool
    runtime_required: bool
    lifecycle_owner: str
    executable: bool

    def as_dict(self) -> dict[str, object]:
        return asdict(self)

    def to_canonical(self) -> DecisionAnswer:
        # Preserve classification exactly. In particular, legacy TOOLCHAIN is
        # never translated to DEVELOPMENT_TOOLING by this compatibility step.
        return DecisionAnswer(
            classification=self.classification,
            purpose=self.purpose,
            shipped=self.shipped,
            runtime_required=self.runtime_required,
            executable=self.executable,
        )


@dataclass(frozen=True)
class ResolutionValidation:
    valid: bool
    status: str
    errors: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "valid": self.valid,
            "status": self.status,
            "errors": list(self.errors),
        }
