from __future__ import annotations

import copy
import hashlib
import json
from pathlib import Path
import runpy
import shutil

import pytest
from jsonschema import ValidationError

from ptsip.governance import AuthorityCatalog
from developer.automation.policy_identity_lifecycle import inspect_policy

ROOT = Path(__file__).resolve().parents[2]
TOOLS = runpy.run_path(str(ROOT / "developer/automation/support_contract_registration.py"))
RegistrationError = TOOLS["RegistrationError"]


def _json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def _contract_fixture(tmp_path):
    shutil.copytree(ROOT / "src/vpms/contracts", tmp_path / "src/vpms/contracts")
    return tmp_path


def test_explicit_scope_provenance_and_allocations_are_registered_non_active():
    record = TOOLS["scope_record"](ROOT)
    assert record["approval"]["decision"] == "APPROVED"
    assert record["approval"]["decision_source"] == "USER_EXPLICIT"
    assert not record["approval"]["runtime_activation_authorized"]
    assert not record["approval"]["selector_retirement_authorized"]
    assert not record["approval"]["sfp_0006_retirement_authorized"]
    assert not record["approval"]["commit_push_authorized"]
    result = TOOLS["verify_registration"](ROOT)
    assert result["status"] == "REGISTERED_NON_ACTIVE"
    assert result["support_policy_id"] == "SFP-0023"
    assert result["product_contract_count"] == 3
    assert result["runtime_enabled"] is False
    assert TOOLS["preflight"](ROOT)["product_contract_ids"] == record["allocation"]["product_contract_ids"]


def test_reusing_allocated_identity_does_not_allocate_again(monkeypatch):
    def fail():
        raise AssertionError("must reuse the exact allocated identity")
    monkeypatch.setattr(TOOLS["preflight"].__globals__["uuid"], "uuid4", fail)
    assert TOOLS["preflight"](ROOT)["status"] == "ALLOCATED"


def test_draft_bridge_cannot_be_current_project_authority():
    catalog = AuthorityCatalog(ROOT)
    _, route, record = catalog.load_current_record("SFP-0023")
    assert route["status"] == record["policy"]["status"] == "DRAFT"
    assert not catalog.lifecycle_is_eligible(catalog.lifecycle_state(record))
    assert record["feature_contract"]["runtime_surface"] == ["src/vpms/integration/ptsip_bridge.py"]
    assert "expanded_identity" not in record["authority_semantics"]
    _, _, legacy = catalog.load_current_record("SFP-0006")
    assert legacy["policy"]["status"] == "ACTIVE"
    assert legacy["authority_semantics"]["expanded_identity"] == "VERIFICATION_PURPOSE_MANAGEMENT_SYSTEM"


def test_mpd_activation_and_excluded_sources_are_preserved():
    record = TOOLS["scope_record"](ROOT)
    for file in record["preserved_files"]:
        assert TOOLS["preserved_text_matches"]((ROOT / file["path"]).read_bytes(), file)
    policy = inspect_policy("MPD-VERI-0007", root=ROOT)
    assert policy["policy_status"] == policy["index_status"] == "ACTIVE"
    assert policy["policy_version"] == "2.0"
    assert all(item["binding_state"] != "ACTIVE" for item in record["deferred_ownership"])


def test_unknown_or_non_active_identity_fails_closed(tmp_path):
    root = _contract_fixture(tmp_path)
    ids = _json(root / "src/vpms/contracts/index.json")["entrypoints"]
    with pytest.raises(RegistrationError, match="UNKNOWN_CONTRACT_ID"):
        TOOLS["inspect_contract"]("protocol", root)
    for identity in ids.values():
        with pytest.raises(RegistrationError, match="CONTRACT_NOT_ACTIVE"):
            TOOLS["inspect_contract"](identity, root, require_active=True)


def test_status_or_identity_mismatch_does_not_resolve(tmp_path):
    root = _contract_fixture(tmp_path)
    path = root / "src/vpms/contracts/protocol.json"
    payload = _json(path)
    payload["status"] = "ACTIVE"
    path.write_text(json.dumps(payload), encoding="utf-8")
    identity = _json(root / "src/vpms/contracts/index.json")["entrypoints"]["protocol"]
    with pytest.raises(RegistrationError, match="CONTRACT_INDEX_MISMATCH"):
        TOOLS["inspect_contract"](identity, root)


def test_even_matching_active_flags_do_not_bypass_materialization_only_gate(tmp_path):
    root = _contract_fixture(tmp_path)
    path = root / "src/vpms/contracts/protocol.json"
    payload = _json(path)
    payload.update(status="ACTIVE", runtime_enabled=True)
    path.write_text(json.dumps(payload), encoding="utf-8")
    index_path = root / "src/vpms/contracts/index.json"
    index = _json(index_path)
    identity = index["entrypoints"]["protocol"]
    index["contracts"][identity].update(status="ACTIVE", runtime_enabled=True)
    index_path.write_text(json.dumps(index), encoding="utf-8")
    with pytest.raises(RegistrationError, match="CONTRACT_NOT_ACTIVE"):
        TOOLS["inspect_contract"](identity, root, require_active=True)


@pytest.mark.parametrize("path", ["../policy/SFP-0006.yaml", "/absolute", "nested/../../escape"])
def test_registered_path_cannot_escape_contract_root(path):
    with pytest.raises(RegistrationError, match="UNSAFE_REGISTERED_PATH"):
        TOOLS["_bounded"](ROOT / "src/vpms/contracts", path)


def test_duplicate_json_keys_are_rejected(tmp_path):
    path = tmp_path / "duplicate.json"
    path.write_text('{"id":"one","id":"two"}', encoding="utf-8")
    with pytest.raises(RegistrationError, match="DUPLICATE_JSON_KEY"):
        TOOLS["_json"](path)


def test_invalid_approval_flags_are_rejected_by_scope_schema():
    record = TOOLS["scope_record"](ROOT)
    record = copy.deepcopy(record)
    record["approval"]["runtime_activation_authorized"] = True
    schema = _json(ROOT / TOOLS["SCOPE_SCHEMA"])
    with pytest.raises(ValidationError):
        TOOLS["Draft202012Validator"](schema).validate(record)


@pytest.mark.parametrize("item", TOOLS["scope_record"](ROOT)["preserved_files"], ids=lambda item: item["path"])
def test_preservation_allows_only_git_text_line_ending_conversion(item):
    raw = (ROOT / item["path"]).read_bytes().replace(b"\r\n", b"\n")
    assert TOOLS["preserved_text_matches"](raw, item)
    assert TOOLS["preserved_text_matches"](raw.replace(b"\n", b"\r\n"), item)
    assert not TOOLS["preserved_text_matches"](raw + b"# semantic/source change\n", item)


def test_preservation_never_treats_lone_cr_or_unicode_as_equivalent():
    raw = "A\nB\n".encode("utf-8")
    record = {"lf_sha256": hashlib.sha256(raw).hexdigest()}
    assert not TOOLS["preserved_text_matches"](b"A\rB\r", record)
    assert not TOOLS["preserved_text_matches"]("Ａ\nB\n".encode("utf-8"), record)
