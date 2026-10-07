"""Compose a validated selection with fully preflighted runner adapters."""
from __future__ import annotations

from dataclasses import dataclass
from typing import Callable, Mapping

from .. import contract_runtime as contracts
from ..domain.model import VerificationCase, VerificationOutcome, VerificationResult
from ..selection.resolver import _selected_cases
from .runner import CaseExecutor, RunnerExecution, RUNNER_CONTRACT_ERROR, run_case


@dataclass(frozen=True)
class _BoundExecutor:
    callback: Callable[[VerificationCase], RunnerExecution]

    def execute(self, case):
        execution = self.callback(case)
        if isinstance(execution, RunnerExecution) and not isinstance(execution.outcome, VerificationOutcome):
            # A typed outer wrapper alone is not proof of a valid protocol outcome.
            return RunnerExecution(
                VerificationOutcome.ERROR, diagnostics=(RUNNER_CONTRACT_ERROR,),
                failure_detail="Case executor returned an invalid VerificationOutcome.",
            )
        return execution


def run_cases(snapshot, selection, *, executors: Mapping[str, CaseExecutor]) -> tuple[VerificationResult, ...]:
    """Never select, mutate the registry, or execute before all adapter checks."""
    composition = contracts.require_active_contract("execution_composition")
    protocol = contracts.require_active_contract("protocol")
    selection_contract = contracts.require_active_contract("selection")
    semantics = composition["semantics"]
    if (semantics["single_case_execution_contract"] != protocol["id"]
            or semantics["selection_contract"] != selection_contract["id"]
            or semantics["adapter_registration_check"] != "ALL_SELECTED_CASES_BEFORE_EXECUTION"
            or semantics["missing_adapter"] != "REJECT_BEFORE_FIRST_CASE_EXECUTION"):
        raise contracts.ContractUnavailable("UNSUPPORTED_EXECUTION_COMPOSITION")
    cases = _selected_cases(snapshot, selection)
    if not isinstance(executors, Mapping):
        raise ValueError("Executor registrations must be an explicit runner-id mapping.")
    bound, missing, invalid = {}, [], []
    for identity in sorted({case.runner.ref for case in cases}):
        try:
            executor = executors[identity]
        except KeyError:
            missing.append(identity)
            continue
        callback = getattr(executor, "execute", None)
        if not callable(callback):
            invalid.append(identity)
        else:
            # Capture once: later mapping mutations cannot swap a checked adapter.
            bound[identity] = _BoundExecutor(callback)
    if missing or invalid:
        raise ValueError("Executor preflight rejected; missing=" + ",".join(missing)
                         + "; invalid=" + ",".join(invalid))
    return tuple(run_case(case, bound[case.runner.ref]) for case in cases)
