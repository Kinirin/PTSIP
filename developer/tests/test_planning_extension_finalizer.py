from __future__ import annotations

from developer.automation.planning_extension_finalizer import (
    _replace_authorization_status,
    _replace_extension_lifecycle,
    _replace_parent_extension_status,
    extension_is_machine_ready,
    extension_parent_consistency_errors,
)


def _ready_extension_payload() -> dict[str, object]:
    return {
        "schema_version": "ptsip-plan-extension/v1",
        "plan_version": "0.4.0",
        "extension": {
            "id": "WU-02-P01",
            "parent": "WU-02",
            "lifecycle": {"status": "ACTIVE"},
        },
        "implementation_authorization": {"status": "AUTHORIZED"},
        "current_known_blockers": [],
        "migration_stages": {
            "P01_A": {"status": "COMPLETE"},
            "P01_B": {"status": "COMPLETE"},
        },
        "p01_f_execution_plan": {
            "execution_order": [
                {
                    "id": "P01_F_FINAL",
                    "status": "COMPLETE",
                    "validation": {"status": "PASS"},
                }
            ]
        },
    }


def test_machine_ready_extension_requires_complete_machine_state() -> None:
    payload = _ready_extension_payload()
    assert extension_is_machine_ready(payload)

    blocked = _ready_extension_payload()
    blocked["current_known_blockers"] = ["WAITING_FOR_VALIDATION"]
    assert not extension_is_machine_ready(blocked)

    pending = _ready_extension_payload()
    pending["p01_f_execution_plan"]["execution_order"][0]["validation"]["status"] = "PENDING"  # type: ignore[index]
    assert not extension_is_machine_ready(pending)

    incomplete = _ready_extension_payload()
    incomplete["migration_stages"]["P01_B"]["status"] = "ACTIVE"  # type: ignore[index]
    assert not extension_is_machine_ready(incomplete)


def test_extension_and_parent_status_text_updates_preserve_structure() -> None:
    extension_text = """schema_version: ptsip-plan-extension/v1
extension:
  id: WU-02-P01
  parent: WU-02
  lifecycle:
    status: ACTIVE
approval:
  status: APPROVED
implementation_authorization:
  status: AUTHORIZED
depends_on:
  - WU-02
"""
    parent_text = """extensions:
  - id: WU-02-P01
    path: docs/planning/0.4.0/WU-02/WU-02-P01.yaml
    status: ACTIVE
scope_contract:
  planning_contract: ptsip-planning/v1
"""

    extension_text = _replace_extension_lifecycle(extension_text, "ACTIVE", "COMPLETE")
    extension_text = _replace_authorization_status(extension_text, "AUTHORIZED", "COMPLETE")
    parent_text = _replace_parent_extension_status(
        parent_text,
        extension_id="WU-02-P01",
        old="ACTIVE",
        new="COMPLETE",
    )

    assert "lifecycle:\n    status: COMPLETE" in extension_text
    assert "implementation_authorization:\n  status: COMPLETE" in extension_text
    assert "- id: WU-02-P01\n    path:" in parent_text
    assert "    status: COMPLETE\nscope_contract:" in parent_text


def test_parent_extension_status_mismatch_is_machine_detectable() -> None:
    extension_payload = _ready_extension_payload()
    parent_payload = {
        "work_unit": {"id": "WU-02"},
        "extensions": [
            {
                "id": "WU-02-P01",
                "path": "docs/planning/0.4.0/WU-02/WU-02-P01.yaml",
                "status": "COMPLETE",
            }
        ],
    }

    errors = extension_parent_consistency_errors(
        parent_payload,
        extension_payload,
        extension_id="WU-02-P01",
        extension_path="docs/planning/0.4.0/WU-02/WU-02-P01.yaml",
    )

    assert len(errors) == 1
    assert "does not match extension lifecycle 'ACTIVE'" in errors[0]
