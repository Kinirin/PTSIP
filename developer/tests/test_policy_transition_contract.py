from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator

from developer.automation.policy_validator import _validate_policy_transition_semantics


ROOT = Path(__file__).resolve().parents[2]


def _schema(name: str) -> dict[str, object]:
    return json.loads(
        (ROOT / "developer" / "policy" / "schemas" / name).read_text(encoding="utf-8")
    )


def _approved_policy(
    *,
    transition_state: str = "PENDING",
    requirement_state: str = "OPEN",
) -> dict[str, object]:
    return {
        "schema_version": "ptsip-developer-policy/v1",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "policy": {
            "id": "MPD-VERI-0001",
            "version": "1.0",
            "title": "Fixture",
            "status": "APPROVED",
        },
        "rules": {"fixture": {"enabled": True}},
        "transition": {
            "target_status": "ACTIVE",
            "state": transition_state,
            "requirements": [
                {
                    "id": "TR-001",
                    "type": "STRUCTURAL_COMPATIBILITY",
                    "state": requirement_state,
                    "refs": ["MPD-SPEC-0023"],
                    "next_action": {
                        "action": "ALIGN_PLAN_IDENTITY_CONTRACT",
                        "execution": "PARALLEL_ALLOWED",
                        "target_refs": ["MPD-SPEC-0023"],
                        "completion_check": "PLAN_IDENTITY_CONTRACT_ALIGNED",
                    },
                }
            ],
        },
        "relations": {
            "supersedes": [],
            "amends": [],
            "extends": [],
            "depends_on": [],
        },
    }


def test_management_policy_schema_accepts_approved_transition_contract() -> None:
    payload = _approved_policy()
    errors = tuple(
        Draft202012Validator(_schema("management-policy.schema.json")).iter_errors(payload)
    )
    assert errors == ()


def test_requirement_state_vocabulary_is_closed_and_has_no_blocking_field() -> None:
    schema = _schema("management-policy.schema.json")

    deferred = _approved_policy()
    deferred["transition"]["requirements"][0]["state"] = "DEFERRED"
    assert tuple(Draft202012Validator(schema).iter_errors(deferred))

    blocking = _approved_policy()
    blocking["transition"]["requirements"][0]["blocking"] = True
    assert tuple(Draft202012Validator(schema).iter_errors(blocking))


def test_approved_ready_state_requires_all_requirements_satisfied_semantically() -> None:
    payload = _approved_policy(transition_state="READY", requirement_state="OPEN")
    errors = _validate_policy_transition_semantics("MPD-VERI-0001", payload)
    assert any("must be PENDING" in item for item in errors)

    payload = _approved_policy(
        transition_state="READY",
        requirement_state="SATISFIED",
    )
    assert _validate_policy_transition_semantics("MPD-VERI-0001", payload) == []


def test_transition_after_references_are_exact_and_acyclic() -> None:
    payload = _approved_policy()
    requirement = payload["transition"]["requirements"][0]
    requirement["next_action"]["after"] = ["TR-999"]
    errors = _validate_policy_transition_semantics("MPD-VERI-0001", payload)
    assert any("unknown transition requirement TR-999" in item for item in errors)

    payload = _approved_policy()
    first = payload["transition"]["requirements"][0]
    first["next_action"]["after"] = ["TR-002"]
    payload["transition"]["requirements"].append(
        {
            "id": "TR-002",
            "type": "VERIFICATION",
            "state": "OPEN",
            "next_action": {
                "action": "RUN_VERIFICATION",
                "execution": "SERIAL_REQUIRED",
                "after": ["TR-001"],
                "completion_check": "VERIFICATION_PASS",
            },
        }
    )
    errors = _validate_policy_transition_semantics("MPD-VERI-0001", payload)
    assert any("transition dependency cycle" in item for item in errors)


def test_active_transition_history_requires_complete_and_satisfied() -> None:
    payload = _approved_policy(
        transition_state="READY",
        requirement_state="SATISFIED",
    )
    payload["policy"]["status"] = "ACTIVE"
    payload["policy"]["version"] = "2.0"
    payload["transition"]["state"] = "COMPLETE"

    schema_errors = tuple(
        Draft202012Validator(_schema("management-policy.schema.json")).iter_errors(payload)
    )
    assert schema_errors == ()
    assert _validate_policy_transition_semantics("MPD-VERI-0001", payload) == []


def test_related_status_schemas_accept_approved() -> None:
    index_schema = _schema("developer-policy-index.schema.json")
    approval_schema = _schema("policy-approval-provenance.schema.json")
    review_schema = _schema("source-application-review.schema.json")

    index_status = index_schema["properties"]["policies"]["items"]["properties"]["status"]["enum"]
    approval_status = approval_schema["properties"]["approval"]["properties"]["target_status"]["enum"]
    review_status = review_schema["properties"]["policy_query_list"]["items"]["properties"]["expected_status"]["enum"]

    assert "APPROVED" in index_status
    assert "APPROVED" in approval_status
    assert "APPROVED" in review_status
