from __future__ import annotations
from developer.tests.policy_migration_helpers import resolve_source_bindings as resolve_policies

from pathlib import Path
from developer.tests.policy_migration_helpers import source_file

import yaml


ROOT = Path(__file__).resolve().parents[2]


def _load(path: str) -> dict[str, object]:
    payload = yaml.safe_load(source_file(ROOT / path).read_text(encoding="utf-8"))
    assert isinstance(payload, dict)
    return payload


def test_policy_plan_identity_contract_is_exactly_aligned() -> None:
    spec = _load("developer/policy/SPEC/MPD-SPEC-0023.yaml")
    automation = _load("developer/policy/MPD-0013.yaml")
    verification = _load("developer/policy/VERI/MPD-VERI-0001.yaml")

    identity = spec["rules"]["binding_identity"]
    assert "plan_id" not in identity

    resolved = identity["resolved_plan_id"]
    assert resolved["role"] == "LOGICAL_PLAN_IDENTITY"
    assert resolved["current_identity_pattern"] == (
        r"^PLN\.[A-Z]{4}\.[A-Z]{4}\.[A-Za-z0-9]{8}$"
    )
    assert resolved["stable_across_version_change"] is True
    assert resolved["stable_across_revision_change"] is True
    assert resolved["repository_unique"] is True

    plan_file = identity["plan_file_id"]
    assert plan_file["role"] == "STABLE_PLAN_FILE_IDENTITY"
    assert plan_file["current_identity_pattern"] == (
        r"^PLANFILE\.[A-Z]{4}\.[A-Za-z0-9]{12}$"
    )
    assert plan_file["stable_across_path_move"] is True
    assert plan_file["stable_across_file_rename"] is True
    assert plan_file["stable_across_content_change"] is True
    assert plan_file["repository_unique"] is True

    registered = identity["registered_plan_id"]
    assert registered == {
        "separate_identity_supported": False,
        "responsibility_merged_into": "plan_file_id",
    }

    created = spec["rules"]["planning_existence_state"]["CREATED"]
    assert created["resolved_plan_id"] == "REQUIRED"
    assert created["plan_file_id"] == "REQUIRED"
    assert created["version"] == "REQUIRED"
    assert created["revision"] == "REQUIRED"
    assert created["plan_ref"] == "REQUIRED"

    binding = automation["rules"]["deterministic_binding_management"]
    assert binding["relationship_resolution"]["exact_lookup_keys"] == [
        "binding_id",
        "policy_ref",
        "resolved_plan_id",
        "plan_file_id",
        "plan_ref",
    ]
    assert binding["plan_materialization"]["separate_registered_plan_id_allocation"] == "FORBIDDEN"

    movement = automation["rules"]["plan_movement_and_replacement"]
    assert movement["same_plan_path_move"]["classification"] == {
        "resolved_plan_id": "SAME",
        "plan_file_id": "SAME",
        "plan_ref": "DIFFERENT",
        "outcome": "MOVE",
    }
    assert movement["plan_file_replacement"]["classification"] == {
        "resolved_plan_id": "SAME",
        "plan_file_id": "DIFFERENT",
        "outcome": "FILE_REPLACEMENT",
    }
    assert movement["plan_file_identity_conflict"]["classification"] == {
        "resolved_plan_id": "DIFFERENT",
        "plan_file_id": "SAME",
        "outcome": "PLAN_FILE_ID_CONFLICT",
    }

    tracker = automation["rules"]["plan_ref_tracking"]
    mutation = tracker["movement_mutation_contract"]
    assert mutation["required_inputs"] == [
        "binding_id",
        "policy_ref",
        "resolved_plan_id",
        "plan_file_id",
        "from_plan_ref",
        "to_plan_ref",
    ]
    assert mutation["forbidden_inputs"] == ["version", "revision"]
    assert mutation["update_scope"] == ["plan_ref"]
    assert mutation["stale_state_protection"] == "EXPECTED_REGISTRY_DIGEST_CAS"

    scope = verification["rules"]["policy_plan_consistency_verification"]["verification_scope"]
    assert "PLAN_ID_BINDING" not in scope
    assert "RESOLVED_PLAN_ID_BINDING" in scope
    assert "PLAN_FILE_ID_BINDING" in scope
    assert "PLAN_REF_BINDING" in scope
