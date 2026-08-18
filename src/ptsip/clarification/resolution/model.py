from __future__ import annotations

from dataclasses import asdict, dataclass

CLASSIFICATIONS = (
    "PRODUCT",
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
    "NEUTRAL_CONTRACT",
)

# DecisionAnswer keeps lifecycle_owner as an input/migration compatibility fact
# until WU-07/WU-10 retire legacy adoption surfaces. Canonical 0.3.6 Project
# Profiles do not persist lifecycle_owner; classification is the ownership authority.
LIFECYCLE_OWNERS = (
    "PRODUCT",
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
    "INDEPENDENT",
)


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
