from __future__ import annotations

import copy
from pathlib import Path

from jsonschema import Draft202012Validator
import pytest

from developer.automation.policy_loader import load_json, load_yaml
from developer.automation.policy_validator import (
    NEUTRAL_CATALOG_CONTRACTS,
    resolve_neutral_catalog_contract,
    validate_neutral_catalog_contract_registration,
    validate_neutral_catalog_snapshot,
)


ROOT = Path(__file__).resolve().parents[2]


@pytest.fixture
def snapshot():
    registry = load_json(NEUTRAL_CATALOG_CONTRACTS, root=ROOT)
    source_index = load_yaml("developer/policy/index.yaml", root=ROOT)
    source_subject = load_yaml("developer/policy/registries/authority-subject-registry.yaml", root=ROOT)
    records = {entry["id"]: load_yaml(entry["path"], root=ROOT) for entry in source_index["policies"]}
    index_contract = registry["contracts"][registry["entrypoints"]["index"]]
    catalog = {"schema_version": registry["entrypoints"]["index"], "artifact_class": index_contract["artifact_class"], "policies": copy.deepcopy(source_index["policies"])}
    for entry in catalog["policies"]:
        entry["policy_class"] = records[entry["id"]]["policy_class"]
    subject = copy.deepcopy(source_subject)
    subject.pop("policy_class")
    subject["schema_version"] = registry["entrypoints"]["subject"]
    subject["artifact_class"] = registry["contracts"][registry["entrypoints"]["subject"]]["artifact_class"]
    return catalog, subject, records, source_index, source_subject


def _check(snapshot):
    catalog, subject, records, source_index, source_subject = snapshot
    return validate_neutral_catalog_snapshot(catalog, subject, records, source_index=source_index, source_subject=source_subject, root=ROOT)


def test_neutral_contract_registration_and_exact_scope_are_valid() -> None:
    assert validate_neutral_catalog_contract_registration(ROOT) == ()
    registry = load_json(NEUTRAL_CATALOG_CONTRACTS, root=ROOT)
    assert registry["catalog_application_status"] == "NOT_APPLIED"
    assert len(registry["change_scope"]["materialization_targets"]) == 7
    assert all(not target["execution_authorized"] for target in registry["change_scope"]["deferred_application_targets"])


def test_neutral_contract_resolution_is_exact_and_fail_closed() -> None:
    assert resolve_neutral_catalog_contract("developer-policy-catalog/v1", ROOT)["canonical_path"] == "developer/policy/index.yaml"
    with pytest.raises(ValueError, match="UNKNOWN_NEUTRAL_CATALOG_CONTRACT"):
        resolve_neutral_catalog_contract("developer-policy-catalog", ROOT)


def test_legacy_catalog_payloads_have_not_been_migrated() -> None:
    for path in ("developer/policy/index.yaml", "developer/policy/registries/authority-subject-registry.yaml"):
        assert load_yaml(path, root=ROOT)["policy_class"] == "PTSIP_DEVELOPER_POLICY"


def test_in_memory_neutral_snapshot_preserves_registered_corpus(snapshot) -> None:
    assert _check(snapshot) == ()


@pytest.mark.parametrize("policy_class", [None, "UNKNOWN_POLICY_CLASS"])
def test_missing_or_unknown_policy_class_is_rejected(snapshot, policy_class) -> None:
    entry = snapshot[0]["policies"][0]
    if policy_class is None:
        entry.pop("policy_class")
    else:
        entry["policy_class"] = policy_class
    assert _check(snapshot)


@pytest.mark.parametrize("field,value", [("policy_class", "VPMS_DEVELOPER_POLICY"), ("status", "DRAFT"), ("path", "developer/policy/MPD-0010.yaml")])
def test_metadata_projection_mismatch_is_rejected(snapshot, field, value) -> None:
    entry = next(item for item in snapshot[0]["policies"] if item["id"] == "MPD-WORK-0003")
    entry[field] = value
    assert _check(snapshot)


def test_boundary_class_cannot_be_a_family_policy(snapshot) -> None:
    entry = next(item for item in snapshot[0]["policies"] if item["id"] == "MPD-VERI-0001")
    entry["policy_class"] = "PTSIP_BOUND_POLICY"
    assert any("boundary class/namespace mismatch" in error for error in _check(snapshot))


def test_duplicate_policy_ids_are_rejected(snapshot) -> None:
    snapshot[0]["policies"].append(copy.deepcopy(snapshot[0]["policies"][0]))
    assert _check(snapshot)


def test_duplicate_policy_ids_across_classes_are_rejected(snapshot) -> None:
    duplicate = copy.deepcopy(snapshot[0]["policies"][0])
    duplicate["policy_class"] = "VPMS_DEVELOPER_POLICY"
    snapshot[0]["policies"].append(duplicate)
    assert any("globally unique" in error for error in _check(snapshot))


def test_catalog_cannot_define_policy_normative_rules(snapshot) -> None:
    snapshot[0]["policies"][0]["rules"] = {"unregistered_rule": True}
    assert _check(snapshot)


def test_subject_membership_projection_drift_is_rejected(snapshot) -> None:
    snapshot[1]["subject_identity_schemes"]["MANAGEMENT_POLICY_ID"]["registered_values"].pop()
    assert any("exactly project catalog membership" in error for error in _check(snapshot))


@pytest.mark.parametrize("field", ["repository_identity_schemes", "current_repository_bindings", "matching"])
def test_subject_semantic_changes_are_rejected(snapshot, field) -> None:
    if field == "matching":
        snapshot[1][field]["order"].reverse()
    elif field == "current_repository_bindings":
        snapshot[1][field][0]["repository_id"] = 1
    else:
        snapshot[1][field]["GITHUB_REPOSITORY_ID"]["host_required"] = False
    assert any("source semantics changed" in error for error in _check(snapshot))


@pytest.mark.parametrize("field", ["fuzzy_match", "ai_semantic_match"])
def test_subject_matching_prohibitions_are_preserved(snapshot, field) -> None:
    snapshot[1]["matching"][field] = True
    assert any("FORBIDDEN" in error for error in _check(snapshot))


@pytest.mark.parametrize("artifact_index", [0, 1])
def test_root_policy_class_is_not_a_neutral_artifact_field(snapshot, artifact_index) -> None:
    snapshot[artifact_index]["policy_class"] = "PTSIP_DEVELOPER_POLICY"
    assert _check(snapshot)


def test_neutral_type_recognition_does_not_open_vpms_policy_materialization(snapshot) -> None:
    registry = load_json(NEUTRAL_CATALOG_CONTRACTS, root=ROOT)
    assert "VPMS_DEVELOPER_POLICY" in registry["$defs"]["developer_policy_class"]["enum"]
    assert not registry["application_gate"]["vpms_policy_materialization_authorized"]
    candidate = copy.deepcopy(snapshot[2]["MPD-VERI-0001"])
    candidate["policy_class"] = "VPMS_DEVELOPER_POLICY"
    schema = load_json(registry["application_gate"]["existing_policy_materialization_schema_ref"], root=ROOT)
    assert tuple(Draft202012Validator(schema).iter_errors(candidate))
