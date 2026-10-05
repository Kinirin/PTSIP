from __future__ import annotations

from pathlib import Path
import runpy

import pytest
from jsonschema import ValidationError

ROOT = Path(__file__).resolve().parents[3]
TOOLS = runpy.run_path(str(ROOT / "developer/automation/support_contract_registration.py"))
VALIDATE = TOOLS["validate_selection_document"]
RegistrationError = TOOLS["RegistrationError"]


@pytest.mark.parametrize("kind,payload", [
    ("request", {"kind": "CASE_IDS", "case_ids": ["b", "a"]}),
    ("request", {"kind": "RULE_REF", "rule_ref": "explicit.selection.release"}),
    ("rule", {"id": "explicit.selection.release", "kind": "CASE_ID_SET", "case_ids": ["b", "a"]}),
    ("result", {"state": "RESOLVED", "case_ids": ["a", "b"], "diagnostics": []}),
    ("result", {"state": "REJECTED", "case_ids": [], "diagnostics": [
        {"code": "UNKNOWN_CASE", "location": "$.case_ids[0]", "reference": "missing"}
    ]}),
])
def test_registered_selection_io_is_valid_but_never_executes(kind, payload):
    assert VALIDATE(kind, payload, ROOT) == {
        "status": "PASS", "kind": kind, "execution_performed": False
    }


@pytest.mark.parametrize("payload", [
    {}, {"kind": "CASE_IDS", "case_ids": []},
    {"kind": "CASE_IDS", "case_ids": ["a", "a"]},
    {"kind": "CASE_IDS", "case_ids": [" a"]},
    {"kind": "CASE_IDS", "case_ids": [""]},
    {"kind": "RULE_REF", "rule_ref": ""},
    {"kind": "RULE_REF", "rule_ref": "release", "case_ids": ["a"]},
    {"kind": "CASE_IDS", "case_ids": ["a"], "unknown": True},
    {"kind": "AUTO", "case_ids": ["a"]},
])
def test_invalid_unknown_duplicate_empty_or_mixed_request_is_rejected(payload):
    with pytest.raises(ValidationError):
        VALIDATE("request", payload, ROOT)


@pytest.mark.parametrize("payload", [
    {"state": "RESOLVED", "case_ids": [], "diagnostics": []},
    {"state": "RESOLVED", "case_ids": ["a", "a"], "diagnostics": []},
    {"state": "REJECTED", "case_ids": ["a"], "diagnostics": [{"code": "UNKNOWN_CASE", "location": "$"}]},
    {"state": "REJECTED", "case_ids": [], "diagnostics": []},
    {"state": "REJECTED", "case_ids": [], "diagnostics": [{"code": "UNKNOWN_ERROR", "location": "$"}]},
])
def test_rejected_selection_cannot_have_a_usable_partial_set(payload):
    with pytest.raises(ValidationError):
        VALIDATE("result", payload, ROOT)


def test_selection_result_order_is_explicit_and_deterministic():
    with pytest.raises(RegistrationError, match="NONDETERMINISTIC_SELECTION_ORDER"):
        VALIDATE("result", {"state": "RESOLVED", "case_ids": ["b", "a"], "diagnostics": []}, ROOT)
    with pytest.raises(RegistrationError, match="NONDETERMINISTIC_DIAGNOSTIC_ORDER"):
        VALIDATE("result", {"state": "REJECTED", "case_ids": [], "diagnostics": [
            {"code": "UNKNOWN_CASE", "location": "$.z"},
            {"code": "UNKNOWN_CASE", "location": "$.a"}
        ]}, ROOT)


def test_unknown_document_kind_does_not_fall_back():
    with pytest.raises(RegistrationError, match="UNKNOWN_SELECTION_DOCUMENT"):
        VALIDATE("other", {}, ROOT)


def test_rule_rejects_duplicate_cases_and_unknown_rule_kinds():
    for payload in (
        {"id": "release", "kind": "CASE_ID_SET", "case_ids": ["a", "a"]},
        {"id": "release", "kind": "PATH_GLOB", "case_ids": ["a"]},
    ):
        with pytest.raises(ValidationError):
            VALIDATE("rule", payload, ROOT)
