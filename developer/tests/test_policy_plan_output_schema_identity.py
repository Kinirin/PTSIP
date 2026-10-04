from __future__ import annotations

import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator


ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = (
    ROOT
    / "developer"
    / "policy"
    / "schemas"
    / "management"
    / "verification"
    / "policy-plan-consistency-report.schema.json"
)
POLICY_PATH = ROOT / "developer" / "policy" / "VERI" / "MPD-VERI-0006.yaml"

SCHEMA_VERSION = "ptsip-management-verification-policy-plan-consistency-report/v1"
SCHEMA_ID = "urn:ptsip:management-verification:policy-plan-consistency-report:v1"


def test_output_schema_identity_is_exact_and_identity_only() -> None:
    schema = json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))

    assert schema["$id"] == SCHEMA_ID
    assert schema["required"] == ["schema_version"]
    assert schema["properties"] == {
        "schema_version": {"const": SCHEMA_VERSION},
    }
    assert schema["additionalProperties"] is True

    validator = Draft202012Validator(schema)
    assert tuple(validator.iter_errors({"schema_version": SCHEMA_VERSION})) == ()
    assert tuple(validator.iter_errors({"schema_version": "wrong/v1"}))


def test_output_policy_owns_output_semantics_not_verification_semantics() -> None:
    payload = yaml.safe_load(POLICY_PATH.read_text(encoding="utf-8"))
    policy = payload["policy"]
    rules = payload["rules"]["policy_plan_consistency_verification_output_contract"]

    assert policy == {
        "id": "MPD-VERI-0006",
        "version": "0.1",
        "title": "Policy-Plan Consistency Verification Output Contract",
        "status": "DRAFT",
    }
    assert rules["verification_semantics_ref"] == "MPD-VERI-0001"
    assert rules["schema_artifact"]["path"] == (
        "developer/policy/schemas/management/verification/"
        "policy-plan-consistency-report.schema.json"
    )
    assert rules["schema_artifact"]["instance_schema_version"] == SCHEMA_VERSION
    assert rules["schema_artifact"]["schema_document_id"] == SCHEMA_ID
    assert rules["schema_artifact"]["schema_document_is_management_policy"] is False
    assert (
        rules["authority_boundary"]["verification_semantics_owner"]
        == "MPD-VERI-0001"
    )
    assert (
        rules["authority_boundary"]["verification_output_semantics_owner"]
        == "MPD-VERI-0006"
    )
