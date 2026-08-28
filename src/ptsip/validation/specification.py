from __future__ import annotations

from ..specification_binding import (
    SPECIFICATION_036_FAMILY,
    SpecificationBindingError,
    SpecificationOperation,
    require_current_specification_support,
)


def specification_binding_errors(
    payload: dict[str, object],
    *,
    details: dict[str, object],
) -> list[str]:
    """Validate declared Specification capability without migration inference."""

    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        return []
    binding = ptsip.get("specification")
    if not isinstance(binding, dict):
        return []

    declared_profile = ptsip.get("version")
    candidate: dict[str, object]
    if declared_profile == SPECIFICATION_036_FAMILY:
        # Historical Tool-numbered profiles did not carry a separate family field.
        # This compatibility projection is ordinary Specification capability lookup,
        # not historical migration-bridge authority.
        candidate = {
            "family": SPECIFICATION_036_FAMILY,
            "source": binding.get("source"),
            "revision": binding.get("revision"),
        }
    else:
        candidate = dict(binding)

    try:
        support = require_current_specification_support(
            candidate,
            SpecificationOperation.VALIDATE,
        )
    except SpecificationBindingError as exc:
        return [f"ptsip.specification [{exc.code}]: {exc}"]

    details["specification_binding"] = {
        **support.binding.as_dict(),
        "compatibility_tool_target": support.tool_version,
        "operation": SpecificationOperation.VALIDATE.value,
    }
    return []
