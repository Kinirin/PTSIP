from __future__ import annotations

from copy import deepcopy
import hashlib
import importlib.util
import json
from pathlib import Path

import pytest

from ptsip.policy_contracts.unified import (
    AUTHORING_BASIS,
    EVALUATION_INTERFACE,
    EVALUATION_REGISTRY,
    INDEX,
    SCHEMA,
    VERSION,
    assess_registration,
    load_document,
    restore_source,
    source_rule_units,
    validate_document,
)


ROOT = Path(__file__).resolve().parents[4]


def _digest(source):
    raw = json.dumps(source, ensure_ascii=True, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(raw).hexdigest()


@pytest.fixture
def routed_root(tmp_path):
    paths = [
        ".ptsip/index.json", ".ptsip/contracts/index.json",
        ".ptsip/contracts/policy/boundary.json", INDEX, SCHEMA,
        AUTHORING_BASIS, EVALUATION_REGISTRY, EVALUATION_INTERFACE,
    ]
    for path in paths:
        target = tmp_path / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes((ROOT / path).read_bytes())
    return tmp_path


@pytest.fixture
def source():
    return {
        "Policy_Identity": {
            "id": "MPD-ARCH-0999", "policy_kind": "PTSIP_DEVELOPER_POLICY",
            "root_family": "ARCH", "title": "Source preservation fixture",
        },
        "Policy_Lifecycle": {"constant": {"status": "APPROVED", "substatus": "provisional"}},
        "Policy_Rules": {
            "Definition": [{"id": "source-rule", "rule": "A-CONSTANT-LONGER-THAN-TWENTY-CHARACTERS"}],
            "Boundary": {"subjects": ["Preserve the exact source requirement."]},
            "Archive": {"value": None, "empty": [], "numeric_object": {"0": "object value"}},
        },
    }


@pytest.fixture
def candidate(source):
    # The public migration entry provides the ordinary serializer under test.
    path = ROOT / ".ptsip/engine/developer/policy/migrate_unified.py"
    spec = importlib.util.spec_from_file_location("policy_migration_test", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.normalize(
        ROOT, "developer/policy/ARCH/MPD-ARCH-0999.yaml", source,
        "UNINDEXED", "1" * 40, VERSION,
    )


def test_common_candidate_restores_exact_tree_and_nested_lifecycle(routed_root, source, candidate):
    assert restore_source(candidate) == source
    assert candidate["Policy_Content"]["source_document"]["sha256"] == _digest(source)
    assert candidate["Policy_Lifecycle"]["status"] == "APPROVED"
    report = validate_document(routed_root, candidate)
    assert report["status"] == "VALID", report
    assert report["rule_count"] == len(source_rule_units(source)) == 2
    assert report["evaluation_registration"]["status"] == "UNREGISTERED"
    assert report["execution_authorized"] is False
    assert report["evaluation_authorized"] is False


def test_rule_projection_cannot_drop_duplicate_or_reassign_source(routed_root, candidate):
    for mutation in ("drop", "duplicate", "id", "scheme", "kind", "source"):
        changed = deepcopy(candidate)
        rules = changed["Policy_Rules"]["rules"]
        if mutation == "drop":
            rules.clear()
        elif mutation == "duplicate":
            rules.append(deepcopy(rules[0]))
        elif mutation == "id":
            rules[0]["id"] = "unrelated-rule"
        elif mutation == "scheme":
            rules[0]["identity_scheme"] = "SOURCE_COORDINATE"
        elif mutation == "kind":
            rules[0]["kind"] = "SEMANTIC_UNIT"
        else:
            rules[0]["source_ref"] = changed["Policy_Lifecycle"]["source_ref"]
        result = validate_document(routed_root, changed)
        assert result["status"] == "FAIL", (mutation, result)


def test_source_partition_rejects_overlap_missing_content_and_wrong_hash(candidate):
    changed = deepcopy(candidate)
    changed["Policy_Content"]["items"].append({"id": "overlap", "source_pointer": "", "value": {}})
    with pytest.raises(ValueError, match="OVERLAPPING"):
        restore_source(changed)
    changed = deepcopy(candidate)
    changed["Policy_Content"]["items"].pop()
    with pytest.raises(ValueError, match="DIGEST|ARRAY"):
        restore_source(changed)
    changed = deepcopy(candidate)
    changed["Policy_Content"]["source_document"]["sha256"] = "0" * 64
    with pytest.raises(ValueError, match="DIGEST_MISMATCH"):
        restore_source(changed)


def test_duplicate_mapping_keys_and_invalid_pointer_are_rejected(candidate):
    with pytest.raises(ValueError, match="DUPLICATE_MAPPING_KEY"):
        load_document(b'{"policy": 1, "policy": 2}')
    with pytest.raises(ValueError, match="INVALID_POLICY_YAML"):
        load_document(b"policy: [unterminated")
    changed = deepcopy(candidate)
    changed["Policy_Content"]["items"][0]["source_pointer"] = "/bad~2escape"
    with pytest.raises(ValueError, match="INVALID_SOURCE_POINTER_ESCAPE"):
        restore_source(changed)


def test_identity_status_and_registration_cannot_be_promoted(routed_root, candidate):
    for section, key in (("Policy_Identity", "title"), ("Policy_Lifecycle", "status")):
        changed = deepcopy(candidate)
        changed[section][key] = "changed"
        assert validate_document(routed_root, changed)["status"] == "FAIL"
    registry_path = routed_root / EVALUATION_REGISTRY
    registry = json.loads(registry_path.read_bytes())
    registry["execution_authorized"] = True
    registry_path.write_text(json.dumps(registry), encoding="utf-8")
    result = validate_document(routed_root, candidate)
    assert result["status"] == "FAIL"
    assert result["problem"] == "CANDIDATE_EVALUATION_REGISTRATION_CHANGED"


def test_unresolved_binding_has_no_runtime_grant_and_forbidden_refs_fail(routed_root, candidate):
    rule = candidate["Policy_Rules"]["rules"][0]
    result = assess_registration(routed_root, rule, "RAC")
    assert result["state"] == "UNRESOLVED"
    assert result["execution_authorized"] is False
    cases = [
        ("developer/Temporary_Record/temporary-policy.json", "TEMPORARY_POLICY_DEPENDENCY_FORBIDDEN"),
        (".ptsip/contracts/compile-sealed-rbac/Responsibility/Responsibilities_List.schema.json", "RETIRED_CONTRACT_REFERENCE_FORBIDDEN"),
    ]
    for path, problem in cases:
        changed = deepcopy(candidate)
        binding = changed["Policy_Rules"]["rules"][0]["contract_bindings"]["RAC"]
        binding["contract_ref"] = {"source_ref": path, "item_ref": "#", "revision_ref": "git:" + "1" * 40}
        result = validate_document(routed_root, changed)
        assert result["status"] == "FAIL"
        assert result["problem"] == problem


def test_binding_basis_refs_must_resolve_and_candidate_cannot_grant_execution(routed_root, candidate):
    changed = deepcopy(candidate)
    binding = changed["Policy_Rules"]["rules"][0]["contract_bindings"]["RA"]
    binding["basis_refs"] = ["missing-source"]
    assert validate_document(routed_root, changed)["problem"] == "UNRESOLVED_POLICY_SOURCE_ITEM"
    changed = deepcopy(candidate)
    changed["Policy_Evaluation"]["execution_authorized"] = True
    result = validate_document(routed_root, changed)
    assert result["status"] == "FAIL"
    assert result["execution_authorized"] is False
