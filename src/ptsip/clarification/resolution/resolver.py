from __future__ import annotations

from .model import CLASSIFICATIONS, LIFECYCLE_OWNERS, DecisionAnswer, ResolutionValidation


_EXPECTED_LEGACY_OWNER = {
    "PRODUCT": "PRODUCT",
    "DEVELOPMENT_TOOLING": "DEVELOPMENT_TOOLING",
    "DELIVERY": "DELIVERY",
    "OPERATIONS": "OPERATIONS",
    "NEUTRAL_CONTRACT": "INDEPENDENT",
}


def validate_answer(answer: DecisionAnswer) -> ResolutionValidation:
    errors: list[str] = []
    if answer.classification not in CLASSIFICATIONS:
        errors.append(
            "classification must be PRODUCT, DEVELOPMENT_TOOLING, DELIVERY, OPERATIONS, or NEUTRAL_CONTRACT"
        )
    if not answer.purpose:
        errors.append("purpose must be non-empty")
    if answer.lifecycle_owner not in LIFECYCLE_OWNERS:
        errors.append(
            "lifecycle_owner compatibility fact must be PRODUCT, DEVELOPMENT_TOOLING, DELIVERY, OPERATIONS, or INDEPENDENT"
        )

    expected_owner = _EXPECTED_LEGACY_OWNER.get(answer.classification)
    if expected_owner is not None and answer.lifecycle_owner != expected_owner:
        errors.append(
            f"{answer.classification} classification conflicts with lifecycle_owner compatibility fact "
            f"{answer.lifecycle_owner!r}; expected {expected_owner!r}"
        )

    if answer.classification in {"DEVELOPMENT_TOOLING", "DELIVERY", "OPERATIONS"}:
        if answer.shipped:
            errors.append(f"{answer.classification} component cannot be declared as shipped Product implementation")
        if answer.runtime_required:
            errors.append(f"{answer.classification} component cannot be required as Product runtime implementation")

    if answer.classification == "NEUTRAL_CONTRACT" and answer.executable:
        errors.append("NEUTRAL_CONTRACT must be non-executable")

    return ResolutionValidation(not errors, "RESOLVED" if not errors else "CONFLICT", tuple(errors))
