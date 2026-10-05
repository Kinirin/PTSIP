from __future__ import annotations

import ast
import subprocess
import sys
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
from vpms.execution.adapters.command import (
    COMMAND_EXECUTION_ERROR,
    COMMAND_NONZERO_EXIT,
    CommandExecutor,
)
from vpms.execution.runner import RunnerExecution, run_case


_REPO_ROOT = Path(__file__).resolve().parents[3]


def _case() -> VerificationCase:
    return VerificationCase(
        id="product.command-check",
        purpose=VerificationPurpose.PRODUCT,
        target=TargetRef(component_id="api-sdk"),
        formula=FormulaRef(ref="command.exit-status"),
        variables=VariablesRef(ref="product.command.variables"),
        policy=PolicyRef(ref="product.command.policy"),
        runner=RunnerRef(ref="command"),
    )


def test_zero_exit_maps_to_pass_through_runner_contract() -> None:
    case = _case()
    executor = CommandExecutor(argv=(sys.executable, "-c", "raise SystemExit(0)"))

    result = run_case(case, executor)

    assert result.case_id == case.id
    assert result.purpose is case.purpose
    assert result.target is case.target
    assert result.outcome is VerificationOutcome.PASS
    assert result.diagnostics == ()
    assert result.failure_detail is None


def test_nonzero_exit_maps_to_fail_with_deterministic_detail() -> None:
    execution = CommandExecutor(
        argv=(sys.executable, "-c", "raise SystemExit(7)")
    ).execute(_case())

    assert execution == RunnerExecution(
        outcome=VerificationOutcome.FAIL,
        diagnostics=(COMMAND_NONZERO_EXIT,),
        failure_detail="Command exited with status 7.",
    )


def test_os_error_maps_to_error_without_raising() -> None:
    execution = CommandExecutor(
        argv=("__vpms_missing_executable_035__",)
    ).execute(_case())

    assert execution.outcome is VerificationOutcome.ERROR
    assert execution.diagnostics == (COMMAND_EXECUTION_ERROR,)
    assert execution.failure_detail is not None
    assert execution.failure_detail.startswith("FileNotFoundError:")


def test_subprocess_error_maps_to_error(monkeypatch: pytest.MonkeyPatch) -> None:
    def _raise(*args: object, **kwargs: object) -> subprocess.CompletedProcess[str]:
        raise subprocess.TimeoutExpired(cmd=("tool",), timeout=1)

    monkeypatch.setattr(subprocess, "run", _raise)
    execution = CommandExecutor(argv=("tool",), timeout_seconds=1).execute(_case())

    assert execution.outcome is VerificationOutcome.ERROR
    assert execution.diagnostics == (COMMAND_EXECUTION_ERROR,)
    assert execution.failure_detail is not None
    assert execution.failure_detail.startswith("TimeoutExpired:")


def test_argv_is_passed_without_shell_interpretation() -> None:
    literal = "value;echo should-not-run"
    script = "import sys; raise SystemExit(0 if sys.argv[1] == %r else 9)" % literal
    execution = CommandExecutor(
        argv=(sys.executable, "-c", script, literal)
    ).execute(_case())

    assert execution.outcome is VerificationOutcome.PASS


@pytest.mark.parametrize(
    "kwargs",
    [
        {"argv": ()},
        {"argv": ["tool"]},
        {"argv": ("",)},
        {"argv": ("tool", 1)},
        {"argv": ("tool",), "cwd": ""},
        {"argv": ("tool",), "timeout_seconds": 0},
        {"argv": ("tool",), "timeout_seconds": True},
    ],
)
def test_invalid_command_configuration_fails_explicitly(kwargs: dict[str, object]) -> None:
    with pytest.raises(ValueError):
        CommandExecutor(**kwargs)  # type: ignore[arg-type]


def test_adapter_does_not_mutate_case_identity_or_state() -> None:
    case = _case()
    before = case.as_dict()

    execution = CommandExecutor(
        argv=(sys.executable, "-c", "raise SystemExit(0)")
    ).execute(case)

    assert execution.outcome is VerificationOutcome.PASS
    assert case.as_dict() == before


def test_adapter_has_no_ptsip_selector_or_pytest_runtime_dependency() -> None:
    source = _REPO_ROOT / "src" / "vpms" / "execution" / "adapters" / "command.py"
    tree = ast.parse(source.read_text(encoding="utf-8"))

    imports: set[str] = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module is not None:
            imports.add(node.module)

    assert all(not name.startswith("ptsip") for name in imports)
    assert all("selector" not in name for name in imports)
    assert "pytest" not in imports


def test_adapter_never_uses_shell_true() -> None:
    source = _REPO_ROOT / "src" / "vpms" / "execution" / "adapters" / "command.py"
    tree = ast.parse(source.read_text(encoding="utf-8"))

    shell_values: list[object] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            for keyword in node.keywords:
                if keyword.arg == "shell" and isinstance(keyword.value, ast.Constant):
                    shell_values.append(keyword.value.value)

    assert shell_values == [False]
