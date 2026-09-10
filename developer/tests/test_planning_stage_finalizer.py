from __future__ import annotations

import pytest

from developer.automation.planning_stage_finalizer import promote_stage_text


def test_pending_stage_is_promoted_and_next_stage_is_unblocked() -> None:
    text = """execution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n      validation:\n        status: PENDING\n      automatic_completion:\n        from_status: IMPLEMENTED_VALIDATION_PENDING\n        to_status: COMPLETE\n    - id: STAGE_B\n      status: BLOCKED_BY_STAGE_A_VALIDATION\n"""
    automatic = {
        "next_stage": {
            "id": "STAGE_B",
            "from_status": "BLOCKED_BY_STAGE_A_VALIDATION",
            "to_status": "READY",
        }
    }

    promoted = promote_stage_text(text, "STAGE_A", automatic)

    assert "- id: STAGE_A\n      status: COMPLETE" in promoted
    assert "validation:\n        status: PASS" in promoted
    assert "- id: STAGE_B\n      status: READY" in promoted


def test_non_pending_stage_cannot_be_promoted_by_text_helper() -> None:
    text = """execution_order:\n    - id: STAGE_A\n      status: READY\n"""
    with pytest.raises(ValueError):
        promote_stage_text(text, "STAGE_A", {})
