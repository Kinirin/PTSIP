from __future__ import annotations

from dataclasses import FrozenInstanceError
import pytest

from vpms import contract_runtime as contracts
from vpms.domain.snapshot import ValidatedRegistrySnapshot, _snapshot_data, load_registry_snapshot


@pytest.fixture
def candidate():
    # Exercise the actual activated product guard, without a lifecycle bypass.
    assert contracts.require_active_contract("protocol")["runtime_enabled"]


def definitions():
    return [{"id": "case-b", "purpose": "TOOLCHAIN", "target": "t", "formula": "f",
             "variables": "v", "policy": "p", "runner": "r"},
            {"id": "case-a", "purpose": "PRODUCT", "target": "t", "formula": "f",
             "variables": "v", "policy": "p", "runner": "r"}]


def references():
    return {"targets": ["t"], "formulas": ["f"], "variables": ["v"],
            "policies": ["p"], "runners": ["r"]}


def test_factory_copies_explicit_input_and_preserves_formula_reuse(candidate):
    raw, refs = definitions(), references()
    loaded = load_registry_snapshot(raw, references=refs)
    assert loaded.ok and not loaded.diagnostics
    snapshot = loaded.snapshot
    raw[0]["id"] = "changed"
    refs["targets"].append("new")
    assert [case.id for case in snapshot.cases] == ["case-a", "case-b"]
    assert snapshot.references.targets == ("t",)
    assert all(case.formula.ref == "f" for case in snapshot.cases)
    with pytest.raises(FrozenInstanceError):
        snapshot.get_case("case-a").id = "changed"
    with pytest.raises(AttributeError):
        snapshot.cases = ()


def test_direct_or_forged_construction_is_not_validation_proof(candidate):
    with pytest.raises(TypeError, match="direct construction"):
        ValidatedRegistrySnapshot()
    with pytest.raises(ValueError, match="UNVALIDATED_REGISTRY"):
        _snapshot_data(object.__new__(ValidatedRegistrySnapshot))


@pytest.mark.parametrize("change", ["unknown", "duplicate", "invalid", "wrong_shape"])
def test_invalid_reference_registrations_never_produce_a_usable_snapshot(candidate, change):
    refs = references()
    if change == "unknown": refs["extra"] = ["x"]
    if change == "duplicate": refs["runners"] = ["r", "r"]
    if change == "invalid": refs["targets"] = [""]
    if change == "wrong_shape": refs = True
    result = load_registry_snapshot(definitions(), references=refs)
    assert not result.ok and result.snapshot is None and result.diagnostics


@pytest.mark.parametrize("change", ["unresolved", "duplicate_case", "malformed", "purpose"])
def test_legacy_loader_diagnostics_are_all_or_nothing(candidate, change):
    raw = definitions()
    if change == "unresolved": raw[1]["formula"] = "unknown"
    if change == "duplicate_case": raw[1]["id"] = raw[0]["id"]
    if change == "malformed": raw = {"cases": raw}
    if change == "purpose": raw[1]["purpose"] = "UNREGISTERED"
    result = load_registry_snapshot(raw, references=references())
    assert result.snapshot is None and result.diagnostics
