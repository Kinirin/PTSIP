from __future__ import annotations

from collections.abc import Mapping
import pytest

from vpms import contract_runtime as contracts
from vpms.domain.registry import Registry, RegistryReferenceIndex
from vpms.domain.snapshot import load_registry_snapshot
from vpms.selection import SelectionResult, resolve_selection


@pytest.fixture
def snapshot():
    refs = {"targets": ["t"], "formulas": ["f"], "variables": ["v"],
            "policies": ["p"], "runners": ["r"]}
    raw = [{"id": name, "purpose": purpose, "target": "t", "formula": "f",
            "variables": "v", "policy": "p", "runner": "r"}
           for name, purpose in [("b", "PRODUCT"), ("a", "TOOLCHAIN")]]
    return load_registry_snapshot(raw, references=refs).snapshot


def assert_schema(result):
    assert not contracts.document_errors("result", result.as_dict())


def test_explicit_ids_are_sorted_without_purpose_inference_or_mutation(snapshot):
    before = snapshot.as_dict()
    request = {"kind": "CASE_IDS", "case_ids": ["b", "a"]}
    result = resolve_selection(snapshot, request)
    request["case_ids"].clear()
    assert result.ok and result.case_ids == ("a", "b")
    assert not result.diagnostics and snapshot.as_dict() == before
    assert_schema(result)
    exported = result.as_dict()
    exported["case_ids"].clear()
    assert result.case_ids == ("a", "b")


@pytest.mark.parametrize("payload", [None, True, {}, {"kind": "FULL"},
    {"kind": "CASE_IDS", "case_ids": []}, {"kind": "CASE_IDS", "case_ids": [""]},
    {"kind": "CASE_IDS", "case_ids": ["a a"]}, {"kind": "CASE_IDS", "case_ids": [1]},
    {"kind": "CASE_IDS", "case_ids": ["a"], "rule_ref": "p"}])
def test_malformed_or_empty_requests_are_rejected(snapshot, payload):
    result = resolve_selection(snapshot, payload)
    assert not result.ok and result.case_ids == ()
    assert result.diagnostics[0].code == "INVALID_REQUEST"
    assert_schema(result)


def test_duplicates_and_unknown_cases_never_return_partial_usable_cases(snapshot):
    duplicate = resolve_selection(snapshot, {"kind": "CASE_IDS", "case_ids": ["a", "a"]})
    assert duplicate.diagnostics[0].code == "DUPLICATE_CASE_ID"
    unknown = resolve_selection(snapshot, {"kind": "CASE_IDS", "case_ids": ["a", "missing-z", "missing-a"]})
    assert unknown.case_ids == ()
    keys = [(d.location, d.code, d.reference) for d in unknown.diagnostics]
    assert keys == sorted(keys)
    assert all(d.code == "UNKNOWN_CASE" for d in unknown.diagnostics)
    assert_schema(duplicate)
    assert_schema(unknown)


def test_rule_resolution_reads_only_the_exact_registered_rule(snapshot):
    class ExactRules(Mapping):
        def __getitem__(self, key):
            assert key == "chosen"
            return {"id": key, "kind": "CASE_ID_SET", "case_ids": ["b", "a"]}
        def __iter__(self): raise AssertionError("unrelated rule scan")
        def __len__(self): raise AssertionError("unrelated rule scan")
    result = resolve_selection(snapshot, {"kind": "RULE_REF", "rule_ref": "chosen"}, rules=ExactRules())
    assert result.case_ids == ("a", "b")
    assert_schema(result)


@pytest.mark.parametrize("raw,code", [(None, "UNRESOLVED_SELECTION_RULE"),
    ({"id": "other", "kind": "CASE_ID_SET", "case_ids": ["a"]}, "UNRESOLVED_SELECTION_RULE"),
    ({"id": "p", "kind": "FILTER", "case_ids": ["a"]}, "UNSUPPORTED_SELECTION_RULE"),
    ({"id": "p", "kind": "CASE_ID_SET", "case_ids": []}, "UNSUPPORTED_SELECTION_RULE"),
    ({"id": "p", "kind": "CASE_ID_SET", "case_ids": ["a", "a"]}, "UNSUPPORTED_SELECTION_RULE")])
def test_missing_mismatched_or_unsupported_rules_fail_closed(snapshot, raw, code):
    result = resolve_selection(snapshot, {"kind": "RULE_REF", "rule_ref": "p"}, rules={"p": raw})
    assert not result.ok and result.case_ids == () and result.diagnostics[0].code == code
    assert_schema(result)


def test_case_policy_does_not_silently_become_a_selection_rule(snapshot):
    result = resolve_selection(snapshot, {"kind": "RULE_REF", "rule_ref": "p"})
    assert result.diagnostics[0].code == "UNRESOLVED_SELECTION_RULE"


@pytest.mark.parametrize("raw", [True, Registry(RegistryReferenceIndex(), ())])
def test_boolean_or_direct_registry_is_not_a_validated_snapshot(snapshot, raw):
    result = resolve_selection(raw, {"kind": "CASE_IDS", "case_ids": ["a"]})
    assert result.diagnostics[0].code == "UNVALIDATED_REGISTRY"
    assert result.case_ids == ()
    assert_schema(result)


def test_selection_results_cannot_be_directly_constructed(snapshot):
    with pytest.raises(TypeError, match="resolve_selection"):
        SelectionResult()
