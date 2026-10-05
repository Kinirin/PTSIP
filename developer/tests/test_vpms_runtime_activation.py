from __future__ import annotations

import copy
import json
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator, ValidationError

from developer.tests.policy_migration_helpers import source_file

from developer.automation.vpms_runtime_activation import (
    RECORD, SCHEMA, activation_record, verify, verify_preserved,
)
from developer.automation.support_contract_registration import scope_record

ROOT = Path(__file__).resolve().parents[2]


def test_activation_is_an_explicit_successor_not_a_historical_approval_rewrite():
    record = activation_record(ROOT)
    assert record["approval"]["runtime_activation_authorized"]
    assert not scope_record(ROOT)["approval"]["runtime_activation_authorized"]
    result = verify(ROOT)
    assert result["status"] == "ACTIVE_VERIFIED"
    assert result["product_contract_count"] == 3
    assert result["runtime_enabled"]
    assert result["support_targets"] == {"SFP-0006": "RETIRED", "SFP-0023": "ACTIVE"}


@pytest.mark.parametrize("field", ["runtime_activation_authorized", "selector_retirement_authorized",
                                  "sfp_0006_retirement_authorized", "sfp_0023_activation_authorized"])
def test_missing_activation_approval_is_not_inferred(field):
    record = copy.deepcopy(activation_record(ROOT))
    record["approval"][field] = False
    schema = json.loads((ROOT / SCHEMA).read_text(encoding="utf-8"))
    with pytest.raises(ValidationError):
        Draft202012Validator(schema).validate(record)


def test_missing_successor_approval_fails_closed(monkeypatch):
    original = Path.is_file
    monkeypatch.setattr(Path, "is_file", lambda path: False if path == ROOT / RECORD else original(path))
    with pytest.raises(ValueError, match="EXPLICIT_ACTIVATION_APPROVAL_REQUIRED"):
        verify(ROOT)


def test_retired_selector_cannot_be_reintroduced(monkeypatch):
    original = Path.exists
    selector = ROOT / "src/vpms/domain/selector.py"
    monkeypatch.setattr(Path, "exists", lambda path: True if path == selector else original(path))
    with pytest.raises(ValueError, match="RETIRED_SOURCE_PRESENT"):
        verify_preserved(ROOT, scope_record(ROOT)["preserved_files"])


def test_sfp_0006_retirement_does_not_authorize_semantic_redefinition(monkeypatch):
    original = Path.read_bytes
    legacy = source_file(ROOT / "src/policy/SFP-0006.yaml")
    monkeypatch.setattr(Path, "read_bytes", lambda path: original(path) + b"# unauthorized\n" if path == legacy else original(path))
    with pytest.raises(ValueError, match="PRESERVED_SOURCE_CHANGED: src/policy/SFP-0006.yaml"):
        verify_preserved(ROOT, scope_record(ROOT)["preserved_files"])
