from __future__ import annotations

from collections.abc import Mapping
import pytest

from vpms import contract_runtime as contracts
from vpms.domain.model import VerificationOutcome
from vpms.domain.snapshot import load_registry_snapshot
from vpms.execution.composition import run_cases
from vpms.execution.runner import RunnerExecution, RUNNER_CONTRACT_ERROR, RUNNER_EXECUTION_ERROR
from vpms.selection import SelectionResult, resolve_selection


@pytest.fixture
def candidate(monkeypatch):
    monkeypatch.setattr(contracts, "require_active_contract", contracts.load_registered_contract)
    refs = {"targets": ["t"], "formulas": ["f"], "variables": ["v"],
            "policies": ["p"], "runners": ["ra", "rb"]}
    raw = [{"id": name, "purpose": purpose, "target": "t", "formula": "f",
            "variables": "v", "policy": "p", "runner": runner}
           for name, purpose, runner in [("b", "TOOLCHAIN", "rb"), ("a", "PRODUCT", "ra")]]
    def make(): return load_registry_snapshot(raw, references=refs).snapshot
    snapshot = make()
    selection = resolve_selection(snapshot, {"kind": "CASE_IDS", "case_ids": ["b", "a"]})
    return snapshot, selection, make


class Executor:
    def __init__(self, calls, outcome=VerificationOutcome.PASS):
        self.calls, self.outcome = calls, outcome
    def execute(self, case):
        self.calls.append(case.id)
        return RunnerExecution(self.outcome)


def test_execution_preserves_order_identity_and_compatibility_without_reselection(candidate, monkeypatch):
    snapshot, selection, _ = candidate
    import vpms.execution.runner as legacy
    monkeypatch.setattr(legacy, "select_cases", lambda *a, **k: pytest.fail("legacy selection was invoked"))
    calls = []
    before = snapshot.as_dict()
    results = run_cases(snapshot, selection, executors={"ra": Executor(calls), "rb": Executor(calls)})
    assert calls == ["a", "b"]
    assert [(r.case_id, r.target.component_id, r.purpose.value) for r in results] == [("a", "t", "PRODUCT"), ("b", "t", "TOOLCHAIN")]
    assert snapshot.as_dict() == before


@pytest.mark.parametrize("bad", ["missing", "invalid", "none"])
def test_all_adapters_are_preflighted_before_the_first_execution(candidate, bad):
    snapshot, selection, _ = candidate
    calls = []
    registrations = {"ra": Executor(calls)}
    if bad == "invalid": registrations["rb"] = object()
    if bad == "none": registrations["rb"] = None
    with pytest.raises(ValueError, match="preflight rejected"):
        run_cases(snapshot, selection, executors=registrations)
    assert calls == []


def test_mapping_mutations_cannot_replace_preflighted_adapters(candidate):
    snapshot, selection, _ = candidate
    calls, registrations = [], {}
    class Mutator:
        def execute(self, case):
            calls.append(case.id)
            registrations["rb"] = object()
            return RunnerExecution(VerificationOutcome.PASS)
    registrations.update(ra=Mutator(), rb=Executor(calls))
    assert all(r.outcome == VerificationOutcome.PASS for r in run_cases(snapshot, selection, executors=registrations))
    assert calls == ["a", "b"]


def test_only_selected_runner_registrations_are_read(candidate):
    snapshot, selection, _ = candidate
    calls, lookups = [], []
    class ExactAdapters(Mapping):
        def __getitem__(self, identity):
            assert identity in {"ra", "rb"}
            lookups.append(identity)
            return Executor(calls)
        def __iter__(self): raise AssertionError("unrelated adapter scan")
        def __len__(self): raise AssertionError("unrelated adapter scan")
    run_cases(snapshot, selection, executors=ExactAdapters())
    assert lookups == ["ra", "rb"] and calls == ["a", "b"]


@pytest.mark.parametrize("bad", ["different_snapshot", "rejected", "forged", "boolean", "dict"])
def test_invalid_or_cross_snapshot_handoffs_never_execute(candidate, bad):
    snapshot, selection, make = candidate
    if bad == "different_snapshot": snapshot = make()
    if bad == "rejected": selection = resolve_selection(snapshot, {"kind": "CASE_IDS", "case_ids": ["missing"]})
    if bad == "forged": selection = object.__new__(SelectionResult)
    if bad == "boolean": selection = True
    if bad == "dict": selection = selection.as_dict()
    calls = []
    with pytest.raises(ValueError):
        run_cases(snapshot, selection, executors={"ra": Executor(calls), "rb": Executor(calls)})
    assert calls == []


@pytest.mark.parametrize("outcome", list(VerificationOutcome))
def test_registered_outcomes_are_preserved(candidate, outcome):
    snapshot, selection, _ = candidate
    results = run_cases(snapshot, selection, executors={"ra": Executor([], outcome), "rb": Executor([], outcome)})
    assert all(result.outcome == outcome for result in results)


@pytest.mark.parametrize("mode,diagnostic", [("exception", RUNNER_EXECUTION_ERROR), ("wrong_type", RUNNER_CONTRACT_ERROR)])
def test_executor_errors_use_the_existing_single_case_protocol(candidate, mode, diagnostic):
    snapshot, selection, _ = candidate
    class Bad:
        def execute(self, case):
            if mode == "exception": raise RuntimeError("adapter error")
            return object()
    results = run_cases(snapshot, selection, executors={"ra": Bad(), "rb": Bad()})
    assert all(result.outcome == VerificationOutcome.ERROR and result.diagnostics == (diagnostic,) for result in results)


@pytest.mark.parametrize("invalid_outcome", ["UNKNOWN", None, True, "PASS"])
def test_a_typed_outer_result_does_not_validate_an_invalid_protocol_outcome(candidate, invalid_outcome):
    snapshot, selection, _ = candidate
    class BadOutcome:
        def execute(self, case):
            return RunnerExecution(invalid_outcome)
    results = run_cases(snapshot, selection, executors={"ra": BadOutcome(), "rb": BadOutcome()})
    assert all(result.outcome == VerificationOutcome.ERROR and result.diagnostics == (RUNNER_CONTRACT_ERROR,)
               for result in results)
    assert all(result.as_dict()["outcome"] == "ERROR" for result in results)
