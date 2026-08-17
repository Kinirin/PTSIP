from __future__ import annotations

import ast
from dataclasses import FrozenInstanceError, fields
from pathlib import Path

import pytest

from vpms.domain.model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationOutcome,
    VerificationPurpose,
)
from vpms.execution.runner import (
    RUNNER_CONTRACT_ERROR,
    RUNNER_EXECUTION_ERROR,
    RunnerExecution,
    run_case,
)


_REPO_ROOT = Path(__file__).resolve().parents[3]


def _case() -> VerificationCase:
    return VerificationCase(
        id="product.api.required-fields",
        purpose=VerificationPurpose.PRODUCT,
        target=TargetRef(component_id="api-sdk"),
        formula=FormulaRef(ref="structure.required-fields"),
        variables=VariablesRef(ref="product.variables"),
        policy=PolicyRef(ref="product.policy"),
        runner=RunnerRef(ref="command"),
    )


class _Executor:
    def __init__(self, execution: RunnerExecution) -> None:
        self.execution = execution
        self.received: VerificationCase | None = None

    def execute(self, case: VerificationCase) -> RunnerExecution:
        self.received = case
        return self.execution


def test_runner_preserves_case_identity_and_normalizes_execution() -> None:
    case = _case()
    executor = _Executor(
        RunnerExecution(
            outcome=VerificationOutcome.FAIL,
            diagnostics=("missing field: runtime_required",),
            failure_detail="required fields did not match policy",
        )
    )

    result = run_case(case, executor)

    assert executor.received is case
    assert result.case_id == case.id
    assert result.purpose is case.purpose
    assert result.target is case.target
    assert result.outcome is VerificationOutcome.FAIL
    assert result.diagnostics == ("missing field: runtime_required",)
    assert result.failure_detail == "required fields did not match policy"


def test_executor_result_contract_cannot_supply_case_identity() -> None:
    assert [field.name for field in fields(RunnerExecution)] == [
        "outcome",
        "diagnostics",
        "failure_detail",
    ]


def test_runner_execution_is_immutable_and_machine_readable() -> None:
    execution = RunnerExecution(
        outcome=VerificationOutcome.PASS,
        diagnostics=("checked",),
    )

    assert execution.as_dict() == {
        "outcome": "PASS",
        "diagnostics": ("checked",),
        "failure_detail": None,
    }
    with pytest.raises(FrozenInstanceError):
        execution.outcome = VerificationOutcome.FAIL


def test_executor_exception_becomes_normalized_error_result() -> None:
    class ExplodingExecutor:
        def execute(self, case: VerificationCase) -> RunnerExecution:
            raise RuntimeError("backend failed")

    case = _case()
    result = run_case(case, ExplodingExecutor())

    assert result.case_id == case.id
    assert result.purpose is case.purpose
    assert result.target is case.target
    assert result.outcome is VerificationOutcome.ERROR
    assert result.diagnostics == (RUNNER_EXECUTION_ERROR,)
    assert result.failure_detail == "RuntimeError: backend failed"


def test_invalid_executor_return_becomes_contract_error() -> None:
    class InvalidExecutor:
        def execute(self, case: VerificationCase) -> RunnerExecution:
            return "PASS"  # type: ignore[return-value]

    result = run_case(_case(), InvalidExecutor())

    assert result.outcome is VerificationOutcome.ERROR
    assert result.diagnostics == (RUNNER_CONTRACT_ERROR,)
    assert result.failure_detail == "Case executor returned an invalid result type: str."


def test_runner_has_no_ptsip_or_framework_dependency() -> None:
    source = _REPO_ROOT / "src" / "vpms" / "execution" / "runner.py"
    tree = ast.parse(source.read_text(encoding="utf-8"))

    imports: set[str] = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module is not None:
            imports.add(node.module)

    assert all(not name.startswith("ptsip") for name in imports)
    assert "pytest" not in imports
    assert "unittest" not in imports
    assert "subprocess" not in imports


def test_runner_does_not_accept_or_mutate_ptsip_state() -> None:
    classification_state = {"api-sdk": "PRODUCT"}
    before = dict(classification_state)

    result = run_case(
        _case(),
        _Executor(RunnerExecution(outcome=VerificationOutcome.PASS)),
    )

    assert result.outcome is VerificationOutcome.PASS
    assert classification_state == before
