from __future__ import annotations

import pytest

from developer.automation.planning_stage_finalizer import promote_stage_text


def test_pending_stage_is_promoted_and_document_state_is_synchronized() -> None:
    text = """migration_stages:\n  P01_E:\n    status: IMPLEMENTED_VALIDATION_PENDING\n    migration_only_artifacts: RETIRED_PENDING_MACHINE_VALIDATION\ncurrent_known_blockers:\n  - STAGE_A_MACHINE_VALIDATION_PENDING\nexecution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n      validation:\n        status: PENDING\n    - id: STAGE_B\n      status: BLOCKED_BY_STAGE_A_VALIDATION\n"""
    automatic = {
        "next_stage": {
            "id": "STAGE_B",
            "from_status": "BLOCKED_BY_STAGE_A_VALIDATION",
            "to_status": "READY",
        },
        "document_updates": {
            "mapping_scalars": [
                {
                    "section": "migration_stages",
                    "key": "P01_E",
                    "field": "status",
                    "from_value": "IMPLEMENTED_VALIDATION_PENDING",
                    "to_value": "COMPLETE_READY_FOR_NEXT_GATE",
                },
                {
                    "section": "migration_stages",
                    "key": "P01_E",
                    "field": "migration_only_artifacts",
                    "from_value": "RETIRED_PENDING_MACHINE_VALIDATION",
                    "to_value": "RETIRED",
                },
            ],
            "list_removals": [
                {
                    "section": "current_known_blockers",
                    "value": "STAGE_A_MACHINE_VALIDATION_PENDING",
                }
            ],
        },
    }

    promoted = promote_stage_text(text, "STAGE_A", automatic)

    assert "P01_E:\n    status: COMPLETE_READY_FOR_NEXT_GATE" in promoted
    assert "migration_only_artifacts: RETIRED" in promoted
    assert "STAGE_A_MACHINE_VALIDATION_PENDING" not in promoted
    assert "- id: STAGE_A\n      status: COMPLETE" in promoted
    assert "validation:\n        status: PASS" in promoted
    assert "- id: STAGE_B\n      status: READY" in promoted


def test_document_update_fails_closed_when_scalar_does_not_match() -> None:
    text = """migration_stages:\n  P01_E:\n    status: READY\nexecution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n"""
    automatic = {
        "document_updates": {
            "mapping_scalars": [
                {
                    "section": "migration_stages",
                    "key": "P01_E",
                    "field": "status",
                    "from_value": "IMPLEMENTED_VALIDATION_PENDING",
                    "to_value": "COMPLETE_READY_FOR_NEXT_GATE",
                }
            ],
            "list_removals": [],
        }
    }
    with pytest.raises(ValueError):
        promote_stage_text(text, "STAGE_A", automatic)


def test_non_pending_stage_cannot_be_promoted_by_text_helper() -> None:
    text = """execution_order:\n    - id: STAGE_A\n      status: READY\n"""
    with pytest.raises(ValueError):
        promote_stage_text(text, "STAGE_A", {})
