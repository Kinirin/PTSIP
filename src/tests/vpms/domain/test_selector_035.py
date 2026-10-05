"""Successor explicit selection preserves the former independence guarantees."""
from __future__ import annotations

import ast
import importlib.util
from pathlib import Path

import pytest

from vpms import load_registry_snapshot, resolve_selection

ROOT = Path(__file__).resolve().parents[3]


def _snapshot(ids=("toolchain.required-fields", "product.required-fields")):
    raw = [dict(id=identity, purpose=purpose, target=target, formula="shared",
                variables=variables, policy=policy, runner="command")
           for identity, purpose, target, variables, policy in (
               (ids[0], "TOOLCHAIN", "t", "tv", "tp"),
               (ids[1], "PRODUCT", "p", "pv", "pp"))]
    loaded = load_registry_snapshot(raw, references={
        "targets": ["t", "p"], "formulas": ["shared"],
        "variables": ["tv", "pv"], "policies": ["tp", "pp"], "runners": ["command"],
    })
    assert loaded.ok, loaded.diagnostics
    return loaded.snapshot


def _select(snapshot, ids):
    result = resolve_selection(snapshot, {"kind": "CASE_IDS", "case_ids": list(ids)})
    assert result.ok, result.diagnostics
    return result


def test_legacy_selector_is_really_retired():
    assert importlib.util.find_spec("vpms.domain.selector") is None
    assert not (ROOT / "src/vpms/domain/selector.py").exists()


@pytest.mark.parametrize("identity", ["product.required-fields", "toolchain.required-fields"])
def test_explicit_ids_select_only_requested_case_without_purpose_inference(identity):
    snapshot = _snapshot()
    result = _select(snapshot, [identity])
    assert result.case_ids == (identity,)


def test_selection_order_is_deterministic_by_case_id():
    snapshot = _snapshot()
    result = _select(snapshot, [case.id for case in snapshot.cases])
    assert result.case_ids == ("product.required-fields", "toolchain.required-fields")


def test_path_like_case_id_is_not_selection_authority():
    snapshot = _snapshot(("tests/product/toolchain-case", "tests/toolchain/product-case"))
    assert _select(snapshot, ["tests/product/toolchain-case"]).case_ids == ("tests/product/toolchain-case",)
    assert _select(snapshot, ["tests/toolchain/product-case"]).case_ids == ("tests/toolchain/product-case",)


def test_shared_formula_does_not_couple_selection_or_case_owned_state():
    snapshot = _snapshot()
    product = snapshot.get_case(_select(snapshot, ["product.required-fields"]).case_ids[0])
    toolchain = snapshot.get_case(_select(snapshot, ["toolchain.required-fields"]).case_ids[0])
    assert product.formula == toolchain.formula
    assert product.purpose != toolchain.purpose
    assert product.variables != toolchain.variables
    assert product.policy != toolchain.policy


def test_selection_preserves_validated_snapshot_and_existing_case_objects():
    snapshot = _snapshot()
    cases = snapshot.cases
    before = snapshot.as_dict()
    result = _select(snapshot, [case.id for case in cases])
    assert isinstance(result.case_ids, tuple)
    assert snapshot.cases is cases
    assert snapshot.as_dict() == before
    assert all(snapshot.get_case(case.id) is case for case in cases)


@pytest.mark.parametrize("raw_request", ["PRODUCT", "TOOLCHAIN", "FULL", None])
def test_raw_legacy_scope_values_are_not_coerced(raw_request):
    result = resolve_selection(_snapshot(), raw_request)
    assert result.state == "REJECTED" and not result.case_ids


def test_non_validated_registry_input_is_rejected():
    result = resolve_selection((), {"kind": "CASE_IDS", "case_ids": ["a"]})
    assert result.state == "REJECTED" and not result.case_ids


def test_selection_has_no_execution_or_path_discovery_coupling():
    tree = ast.parse((ROOT / "src/vpms/selection/resolver.py").read_text(encoding="utf-8"))
    imports, calls = set(), set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import): imports.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module: imports.add(node.module)
        elif isinstance(node, ast.Call) and isinstance(node.func, ast.Name): calls.add(node.func.id)
    assert "subprocess" not in imports
    assert all("execution" not in name and "pathlib" not in name for name in imports)
    assert not {"glob", "rglob", "walk", "run", "Popen"} & calls
