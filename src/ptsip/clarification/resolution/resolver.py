from __future__ import annotations

from .model import CLASSIFICATIONS, LIFECYCLE_OWNERS, DecisionAnswer, ResolutionValidation


def validate_answer(answer: DecisionAnswer) -> ResolutionValidation:
    errors: list[str] = []
    if answer.classification not in CLASSIFICATIONS:
        errors.append("classification must be PRODUCT, TOOLCHAIN, or NEUTRAL_CONTRACT")
    if not answer.purpose:
        errors.append("purpose must be non-empty")
    if answer.lifecycle_owner not in LIFECYCLE_OWNERS:
        errors.append("lifecycle_owner must be PRODUCT, DEVELOPMENT_TOOLING, or INDEPENDENT")

    if answer.classification == "PRODUCT" and answer.lifecycle_owner != "PRODUCT":
        errors.append("PRODUCT classification requires PRODUCT lifecycle ownership")
    if answer.classification == "TOOLCHAIN":
        if answer.lifecycle_owner != "DEVELOPMENT_TOOLING":
            errors.append("TOOLCHAIN classification requires DEVELOPMENT_TOOLING lifecycle ownership")
        if answer.shipped:
            errors.append("TOOLCHAIN component cannot be declared as shipped with the Product")
        if answer.runtime_required:
            errors.append("TOOLCHAIN component cannot be required by the Product at runtime")
    if answer.classification == "NEUTRAL_CONTRACT":
        if answer.executable:
            errors.append("NEUTRAL_CONTRACT must be non-executable")
        if answer.lifecycle_owner != "INDEPENDENT":
            errors.append("NEUTRAL_CONTRACT requires INDEPENDENT lifecycle ownership")

    return ResolutionValidation(not errors, "RESOLVED" if not errors else "CONFLICT", tuple(errors))
