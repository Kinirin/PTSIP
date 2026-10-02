from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator


ROOT = Path(__file__).resolve().parents[2]
SCHEMA = ROOT / "developer/bindings/schemas/policy-plan-bindings.schema.json"


def _validator() -> Draft202012Validator:
    schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def _registry(binding: dict[str, object] | None = None) -> dict[str, object]:
    return {
        "schema_version": "ptsip-policy-plan-bindings/v2",
        "registry_role": "POLICY_PLAN_BINDING",
        "schema_ref": "developer/bindings/schemas/policy-plan-bindings.schema.json",
        "bindings": [] if binding is None else [binding],
    }


def _created(*, policy_ref: str = "MPD-0013") -> dict[str, object]:
    return {
        "binding_id": "PPB-0001",
        "policy_ref": policy_ref,
        "planning_state": "CREATED",
        "resolved_plan_id": "PLN.MIGR.READ.A7k2Q9mX",
        "plan_file_id": "PLANFILE.MAIN.H7sP2kQ9mXa4",
        "version": "1.0",
        "revision": "Rev.0001",
        "plan_ref": "developer/planning/0.4.0/example.yaml",
    }


def test_empty_v2_registry_is_valid() -> None:
    assert tuple(_validator().iter_errors(_registry())) == ()


def test_not_created_forbids_all_plan_identity_and_location_fields() -> None:
    binding = {
        "binding_id": "PPB-0001",
        "policy_ref": "MPD-0013",
        "planning_state": "NOT_CREATED",
    }
    assert tuple(_validator().iter_errors(_registry(binding))) == ()

    for field, value in (
        ("resolved_plan_id", "PLN.MIGR.READ.A7k2Q9mX"),
        ("plan_file_id", "PLANFILE.MAIN.H7sP2kQ9mXa4"),
        ("version", "1.0"),
        ("revision", "Rev.0001"),
        ("plan_ref", "developer/planning/0.4.0/example.yaml"),
    ):
        candidate = dict(binding)
        candidate[field] = value
        assert tuple(_validator().iter_errors(_registry(candidate))), field


def test_created_requires_new_identity_fields_and_rejects_legacy_plan_id() -> None:
    validator = _validator()
    binding = _created()
    assert tuple(validator.iter_errors(_registry(binding))) == ()

    for field in ("resolved_plan_id", "plan_file_id", "version", "revision", "plan_ref"):
        candidate = dict(binding)
        del candidate[field]
        assert tuple(validator.iter_errors(_registry(candidate))), field

    legacy = dict(binding)
    legacy["plan_id"] = "WU-07"
    assert tuple(validator.iter_errors(_registry(legacy)))

    registered = dict(binding)
    registered["registered_plan_id"] = "REGPLAN.example"
    assert tuple(validator.iter_errors(_registry(registered)))


def test_identity_patterns_are_exact() -> None:
    validator = _validator()

    bad_resolved = _created()
    bad_resolved["resolved_plan_id"] = "PLN.MIGRATION.READ.A7k2Q9mX"
    assert tuple(validator.iter_errors(_registry(bad_resolved)))

    bad_file = _created()
    bad_file["plan_file_id"] = "PLANFILE.MAIN.short"
    assert tuple(validator.iter_errors(_registry(bad_file)))


def test_version_and_revision_are_required_but_format_is_not_invented() -> None:
    validator = _validator()

    empty_version = _created()
    empty_version["version"] = ""
    assert tuple(validator.iter_errors(_registry(empty_version)))

    empty_revision = _created()
    empty_revision["revision"] = ""
    assert tuple(validator.iter_errors(_registry(empty_revision)))

    nonempty = _created()
    nonempty["version"] = "custom-version"
    nonempty["revision"] = "custom-revision"
    assert tuple(validator.iter_errors(_registry(nonempty))) == ()


def test_schema_preserves_many_to_many_plan_identity_reuse() -> None:
    registry = _registry()
    first = _created(policy_ref="MPD-0013")
    second = _created(policy_ref="MPD-VERI-0001")
    second["binding_id"] = "PPB-0002"
    registry["bindings"] = [first, second]

    assert tuple(_validator().iter_errors(registry)) == ()
