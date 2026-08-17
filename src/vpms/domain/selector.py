from __future__ import annotations

from enum import StrEnum

from .model import VerificationCase, VerificationPurpose
from .registry import Registry


class SelectionScope(StrEnum):
    """Explicit selection scope. FULL is not a VerificationPurpose."""

    PRODUCT = "PRODUCT"
    TOOLCHAIN = "TOOLCHAIN"
    FULL = "FULL"


_SCOPE_PURPOSE = {
    SelectionScope.PRODUCT: VerificationPurpose.PRODUCT,
    SelectionScope.TOOLCHAIN: VerificationPurpose.TOOLCHAIN,
}


def select_cases(
    registry: Registry,
    *,
    scope: SelectionScope,
) -> tuple[VerificationCase, ...]:
    """Select validated Verification Cases by explicit VPMS purpose scope."""

    if not isinstance(registry, Registry):
        raise TypeError("registry must be a VPMS Registry.")
    if not isinstance(scope, SelectionScope):
        raise TypeError("scope must be a SelectionScope.")

    if scope is SelectionScope.FULL:
        selected = registry.cases
    else:
        purpose = _SCOPE_PURPOSE[scope]
        selected = tuple(case for case in registry.cases if case.purpose is purpose)

    return tuple(sorted(selected, key=lambda case: case.id))
