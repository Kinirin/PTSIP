from __future__ import annotations

from .model import (
    CLASSIFICATIONS,
    LEGACY_LIFECYCLE_OWNERS,
    DecisionAnswer,
    LegacyDecisionAnswerV1,
    ResolutionValidation,
)

_EXPECTED_LEGACY_OWNER = {
    "PRODUCT": "PRODUCT",
    "DEVELOPMENT_TOOLING": "DEVELOPMENT_TOOLING",
    "DELIVERY": "DELIVERY",
    "OPERATIONS": "OPERATIONS",
    "NEUTRAL_CONTRACT": "INDEPENDENT",
}


def validate_answer(answer: DecisionAnswer) -> ResolutionValidation:
    """Validate only canonical ptsip-clarification-answer/v2 semantics."""

    errors: list[str] = []
    if answer.classification not in CLASSIFICATIONS:
        errors.append(
            "classification must be PRODUCT, DEVELOPMENT_TOOLING, DELIVERY, OPERATIONS, or NEUTRAL_CONTRACT"
        )
    if not answer.purpose:
        errors.append("purpose must be non-empty")

    if answer.classification in {"DEVELOPMENT_TOOLING", "DELIVERY", "OPERATIONS"}:
        if answer.shipped:
            errors.append(f"{answer.classification} component cannot be declared as shipped Product implementation")
        if answer.runtime_required:
            errors.append(f"{answer.classification} component cannot be required as Product runtime implementation")

    if answer.classification == "NEUTRAL_CONTRACT" and answer.executable:
        errors.append("NEUTRAL_CONTRACT must be non-executable")

    return ResolutionValidation(not errors, "RESOLVED" if not errors else "CONFLICT", tuple(errors))


def validate_legacy_answer(answer: LegacyDecisionAnswerV1) -> ResolutionValidation:
    """Validate v1-only compatibility facts before canonicalization.

    The compatibility lifecycle_owner can reject an inconsistent historical
    record, but it never supplies or rewrites canonical classification.
    """

    errors: list[str] = []
    if answer.lifecycle_owner not in LEGACY_LIFECYCLE_OWNERS:
        errors.append(
            "legacy lifecycle_owner must be PRODUCT, DEVELOPMENT_TOOLING, DELIVERY, OPERATIONS, or INDEPENDENT"
        )
    expected_owner = _EXPECTED_LEGACY_OWNER.get(answer.classification)
    if expected_owner is not None and answer.lifecycle_owner != expected_owner:
        errors.append(
            f"legacy classification {answer.classification!r} conflicts with lifecycle_owner "
            f"{answer.lifecycle_owner!r}; expected {expected_owner!r}"
        )

    canonical = answer.to_canonical()
    canonical_validation = validate_answer(canonical)
    errors.extend(canonical_validation.errors)
    return ResolutionValidation(not errors, "RESOLVED" if not errors else "CONFLICT", tuple(errors))


def canonicalize_legacy_answer(answer: LegacyDecisionAnswerV1) -> DecisionAnswer:
    validation = validate_legacy_answer(answer)
    if not validation.valid:
        raise ValueError("Legacy v1 clarification answer is invalid: " + "; ".join(validation.errors))
    return answer.to_canonical()
