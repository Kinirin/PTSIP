from __future__ import annotations

import json
import runpy
import shutil
from importlib.resources import files
from pathlib import Path

import pytest
import yaml
from jsonschema import ValidationError

from agent_contracts.candidate import (
    CandidateValidationError,
    read_json,
    validate_candidate_assets,
    validate_projection_bindings,
)


ROOT = Path(__file__).resolve().parents[4]
FOUR_PATHS = (
    "src/agent_contracts/candidate.py",
    "src/agent_contracts/machine_resolver.py",
    "src/agent_contracts/digests/policy-v1.json",
    "src/agent_contracts/digests/projection-v1.json",
)


@pytest.fixture
def contract_root(tmp_path: Path) -> Path:
    target = tmp_path / "contracts"
    shutil.copytree(Path(str(files("agent_contracts"))), target,
                    ignore=shutil.ignore_patterns("__pycache__", "*.pyc"))
    return target


def _write(root: Path, ref: str, value: dict) -> None:
    (root / ref).write_text(json.dumps(value, ensure_ascii=False) + "\n", encoding="utf-8")


def _write_projection(root: Path, value: dict) -> None:
    _write(root, "digests/projection-v1.json", value)
    policy = read_json(root, "digests/policy-v1.json")
    policy["projection_policy"] = {
        "projection_id": value["id"], "status": value["binding_scope"]["status"],
        "field_governance": value["field_governance"],
        "collection_semantics": value["collection_semantics"],
        "binding_scope": value["binding_scope"],
    }
    _write(root, "digests/policy-v1.json", policy)


def test_exact_binding_inventory_validates_without_digest_activation() -> None:
    assert validate_projection_bindings() == {
        "included": 576, "excluded": 23, "collections": 254, "assets": 11,
    }
    assert validate_candidate_assets()["operations"] == 5
    index = read_json(files("agent_contracts"), "index.json")
    policy = read_json(files("agent_contracts"), "digests/policy-v1.json")
    assert index["canonical_digest"] == policy["active_scheme"] == "UNRESOLVED"
    assert index["normative_authority"] is policy["normative_authority"] is False


def test_domain_status_is_included_not_excluded_by_its_field_name() -> None:
    projection = read_json(files("agent_contracts"), "digests/projection-v1.json")
    fields = projection["field_governance"]
    included = fields["exact_inclusion_list"]["exact_role_bindings"]
    excluded = fields["exact_exclusion_list"]["exact_role_bindings"]
    assert {b["semantic_role"] for b in included} == set(fields["exact_inclusion_list"]["exact_semantic_roles"])
    assert any(b["json_pointer"].endswith("/status") and b["semantic_role"] == "STATE_MEANING"
               for b in included)
    assert not any(b["json_pointer"].startswith("/payloads/") for b in excluded)


@pytest.mark.parametrize("change", [
    "missing", "unknown_role", "unknown_asset", "unknown_pointer", "cardinality",
    "condition", "duplicate", "overlap", "missing_collection", "unknown_collection_type",
    "identity_pointer", "exception", "authority",
])
def test_invalid_bindings_fail_closed_without_fallback(contract_root: Path, change: str) -> None:
    projection = read_json(contract_root, "digests/projection-v1.json")
    fields = projection["field_governance"]
    included = fields["exact_inclusion_list"]["exact_role_bindings"]
    excluded = fields["exact_exclusion_list"]["exact_role_bindings"]
    collections = projection["collection_semantics"]["exact_collection_bindings"]
    if change == "missing":
        included.pop(0)
    elif change == "unknown_role":
        included[0]["semantic_role"] = "UNREGISTERED_ROLE"
    elif change == "unknown_asset":
        included[0]["asset_ref"] = "../outside.json"
    elif change == "unknown_pointer":
        included[0]["json_pointer"] = "/payloads/specs/MISSING/rules"
    elif change == "cardinality":
        included[0]["required_cardinality"]["minimum"] = 0
    elif change == "condition":
        included[0]["extraction_condition"] = "INFER_IF_MISSING"
    elif change == "duplicate":
        included.append(dict(included[0]))
    elif change == "overlap":
        conflict = dict(included[0])
        conflict["json_pointer"] = conflict["json_pointer"].rsplit("/", 1)[0]
        conflict["semantic_role"] = "NON_NORMATIVE_HUMAN_EXPLANATION"
        excluded.append(conflict)
    elif change == "missing_collection":
        collections.pop()
    elif change == "unknown_collection_type":
        collections[0]["collection_type"] = "AI_INFERRED"
    elif change == "identity_pointer":
        next(b for b in collections if b["collection_type"] == "SET_LIKE")["identity"]["logical_id_pointer"] = "/scope"
    elif change == "exception":
        projection["collection_semantics"]["set_like"]["exception"]["exact_exception_bindings"] = ["UNDECLARED"]
    elif change == "authority":
        projection["normative_authority"] = True
    _write_projection(contract_root, projection)
    with pytest.raises(CandidateValidationError):
        validate_projection_bindings(contract_root)


def test_unknown_semantic_field_cannot_enter_an_existing_asset(contract_root: Path) -> None:
    group = read_json(contract_root, "contracts/current.json")
    group["undeclared_semantics"] = True
    _write(contract_root, "contracts/current.json", group)
    with pytest.raises((CandidateValidationError, ValidationError)):
        validate_projection_bindings(contract_root)


def test_new_collection_inside_a_bound_subtree_needs_an_exact_binding(contract_root: Path) -> None:
    group = read_json(contract_root, "contracts/current.json")
    schema = next(iter(group["payloads"]["io_schemas"].values()))
    schema["properties"]["new_field"] = {"type": "string", "enum": ["A", "B"]}
    _write(contract_root, "contracts/current.json", group)
    with pytest.raises(CandidateValidationError, match="unclassified or stale collection"):
        validate_projection_bindings(contract_root)


def test_payload_registry_key_must_match_record_identity(contract_root: Path) -> None:
    group = read_json(contract_root, "contracts/current.json")
    record = next(iter(group["payloads"]["specs"].values()))
    record["id"] = "DIFFERENT_IDENTITY"
    _write(contract_root, "contracts/current.json", group)
    with pytest.raises(CandidateValidationError, match="registry/record identity mismatch"):
        validate_projection_bindings(contract_root)


def test_duplicate_set_identity_is_rejected(contract_root: Path) -> None:
    group = read_json(contract_root, "contracts/current.json")
    entries = next(iter(group["payloads"]["vocabularies"].values()))["entries"]
    entries[1]["id"] = entries[0]["id"]
    _write(contract_root, "contracts/current.json", group)
    with pytest.raises(CandidateValidationError, match="duplicate set-like logical identity"):
        validate_projection_bindings(contract_root)


def test_set_member_collection_bindings_follow_identity_not_array_position(contract_root: Path) -> None:
    group = read_json(contract_root, "contracts/current.json")
    for record in group["payloads"]["specs"].values():
        record["rules"].reverse()
    for record in group["payloads"]["vocabularies"].values():
        record["entries"].reverse()
    _write(contract_root, "contracts/current.json", group)
    assert validate_projection_bindings(contract_root)["collections"] == 254


def test_excluded_vector_expectations_still_require_exact_parity(contract_root: Path) -> None:
    vectors = read_json(contract_root, "conformance/current.json")
    next(iter(vectors["operations"].values()))[0]["expected"] = "ALTERED_EXPECTATION"
    _write(contract_root, "conformance/current.json", vectors)
    with pytest.raises(CandidateValidationError, match="excluded migration expectation"):
        validate_projection_bindings(contract_root)


def test_independent_digest_and_projection_records_must_match(contract_root: Path) -> None:
    policy = read_json(contract_root, "digests/policy-v1.json")
    policy["projection_policy"]["status"] = "OTHER_BINDING_SET"
    _write(contract_root, "digests/policy-v1.json", policy)
    with pytest.raises(CandidateValidationError, match="digest/projection binding mismatch"):
        validate_projection_bindings(contract_root)


def test_binding_completion_does_not_authorize_an_active_digest_scheme(contract_root: Path) -> None:
    policy = read_json(contract_root, "digests/policy-v1.json")
    policy["active_scheme"] = "UNVERIFIED-SCHEME"
    _write(contract_root, "digests/policy-v1.json", policy)
    with pytest.raises(CandidateValidationError, match="unverified digest scheme activation"):
        validate_projection_bindings(contract_root)


def test_four_approved_paths_have_exact_owners_and_select_only_the_new_mode() -> None:
    profile = yaml.safe_load((ROOT / ".ptsip/profiles/main.ptsip.yaml").read_text(encoding="utf-8"))
    registry = yaml.safe_load((ROOT / ".github/test_modes.yaml").read_text(encoding="utf-8"))
    resolver = runpy.run_path(str(ROOT / ".github/scripts/resolve_test_modes.py"))
    components = {c["id"]: c for c in profile["components"]}
    verification = components["agent-contract-candidate-verification"]
    assert verification["analysis_inputs"][:4] == list(FOUR_PATHS)
    assert "src/agent_contracts/**" not in components["ptsip-contract-verification"]["analysis_inputs"]
    for path in FOUR_PATHS:
        selected, not_required = resolver["resolve_automatic_selection"](registry, profile, [path])
        assert [item["id"] for item in selected] == ["agent-contract-candidate"]
        assert not_required == []
        assert sum(path in c["include"] for c in components.values()) == 1
    for path in verification["include"]:
        assert sum(resolver["matches_pattern"](path, pattern)
                   for c in components.values() for pattern in c["include"]) == 1
    relations = {(r["from"], r["to"], r["type"]) for r in profile["relationships"]}
    for source in (
        "agent-contract-candidate-compiler",
        "agent-contract-candidate-resolver",
        "agent-contract-candidate-contracts",
        "agent-contract-candidate-projections",
    ):
        assert (verification["id"], source, "VERIFIES") in relations
    with pytest.raises(resolver["TestModeSelectionError"], match="unmapped changed paths"):
        resolver["resolve_automatic_selection"](registry, profile, ["src/agent_contracts/not-approved.py"])
