from __future__ import annotations

from dataclasses import asdict, dataclass

CLASSIFICATIONS = ("PRODUCT", "TOOLCHAIN", "NEUTRAL_CONTRACT")
LIFECYCLE_OWNERS = ("PRODUCT", "DEVELOPMENT_TOOLING", "INDEPENDENT")


@dataclass(frozen=True)
class DecisionAnswer:
    classification: str
    purpose: str
    shipped: bool
    runtime_required: bool
    lifecycle_owner: str
    executable: bool

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


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
