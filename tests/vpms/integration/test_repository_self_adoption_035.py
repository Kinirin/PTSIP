from __future__ import annotations

import ast
import subprocess
import sys
from pathlib import Path

import pytest

from vpms.domain.model import VerificationOutcome, VerificationPurpose
from vpms.domain.registry import (
    Registry,
    RegistryReferenceIndex,
    load_registry,
    register_formulas,
)
from vpms.domain.selector import SelectionScope
from vpms.execution.adapters.command import CommandExecutor
from vpms.execution.runner import run_selected_cases


REPO_ROOT = Path(__file__).resolve().parents[3]
SHARED_FORMULA_ID = "command.exit-zero"
FULL_REPOSITORY_PYTEST_ARGV = (sys.executable, "-m", "pytest", "-q", "tests")

_CASES = [
    {
        "id": "ptsip.product.canonical-contracts",
        "purpose": "PRODUCT",
        "target": "ptsip-distribution",
        "formula": SHARED_FORMULA_ID,
        "variables": "nodeid.product.canonical-contracts",
        "policy": "distribution.contract-integrity",
        "runner": "pytest.product.canonical-contracts",
    },
    {
        "id": "ptsip.product.package-contracts",
        "purpose": "PRODUCT",
        "target": "ptsip-distribution",
        "formula": SHARED_FORMULA_ID,
        "variables": "nodeid.product.package-contracts",
        "policy": "distribution.package-integrity",
        "runner": "pytest.product.package-contracts",
    },
    {
        "id": "ptsip.toolchain.release-workflow",
        "purpose": "TOOLCHAIN",
        "target": "repository-release-automation",
        "formula": SHARED_FORMULA_ID,
        "variables": "nodeid.toolchain.release-workflow",
        "policy": "release.workflow-integrity",
        "runner": "pytest.toolchain.release-workflow",
    },
    {
        "id": "ptsip.toolchain.routine-ci",
        "purpose": "TOOLCHAIN",
        "target": "repository-release-automation",
        "formula": SHARED_FORMULA_ID,
        "variables": "nodeid.toolchain.routine-ci",
        "policy": "ci.verification-boundary",
        "runner": "pytest.toolchain.routine-ci",
    },
]

RUNNER_NODEIDS = {
    "pytest.product.canonical-contracts": (
        "tests/ptsip/test_release_readiness_030.py::"
        "test_canonical_and_embedded_machine_readable_contracts_are_identical"
    ),
    "pytest.product.package-contracts": (
        "tests/ptsip/test_release_readiness_030.py::"
        "test_release_package_contains_bound_machine_readable_contracts"
    ),
    "pytest.toolchain.release-workflow": (
        "tests/ptsip/test_release_readiness_030.py::"
        "test_release_workflow_derives_tool_tag_from_package_version"
    ),
    "pytest.toolchain.routine-ci": (
        "tests/ptsip/test_release_readiness_030.py::"
        "test_routine_ci_supports_selective_modes_and_preserves_full_exact_sha"
    ),
}


def repository_registry() -> Registry:
    formulas = register_formulas((SHARED_FORMULA_ID,))
    references = RegistryReferenceIndex(
        targets=("ptsip-distribution", "repository-release-automation"),
        variables=tuple(case["variables"] for case in _CASES),
        policies=tuple(case["policy"] for case in _CASES),
        runners=tuple(RUNNER_NODEIDS),
    ).with_registered_formulas(formulas)
    loaded = load_registry(_CASES, references=references)
    if not loaded.ok or loaded.registry is None:
        raise RuntimeError(
            f"Repository VPMS adoption registry is invalid: {loaded.diagnostics!r}"
        )
    return loaded.registry


def repository_executors() -> dict[str, CommandExecutor]:
    return {
        runner_ref: CommandExecutor(
            argv=(sys.executable, "-m", "pytest", "-q", nodeid),
            cwd=str(REPO_ROOT),
        )
        for runner_ref, nodeid in RUNNER_NODEIDS.items()
    }


def _release_readiness_test_names() -> set[str]:
    path = REPO_ROOT / "tests" / "ptsip" / "test_release_readiness_030.py"
    tree = ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
    return {
        node.name
        for node in tree.body
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
        and node.name.startswith("test_")
    }


def _record_commands(monkeypatch: pytest.MonkeyPatch) -> list[tuple[str, ...]]:
    calls: list[tuple[str, ...]] = []

    def _run(
        argv: tuple[str, ...],
        **kwargs: object,
    ) -> subprocess.CompletedProcess[str]:
        calls.append(tuple(argv))
        assert kwargs["cwd"] == str(REPO_ROOT)
        assert kwargs["check"] is False
        assert kwargs["shell"] is False
        return subprocess.CompletedProcess(argv, 0, stdout="", stderr="")

    monkeypatch.setattr(subprocess, "run", _run)
    return calls


def _nodeids_from_calls(calls: list[tuple[str, ...]]) -> list[str]:
    return [argv[-1] for argv in calls]


def test_repository_registry_registers_real_mixed_module_cases_by_purpose() -> None:
    registry = repository_registry()

    assert [case.id for case in registry.cases] == [
        "ptsip.product.canonical-contracts",
        "ptsip.product.package-contracts",
        "ptsip.toolchain.release-workflow",
        "ptsip.toolchain.routine-ci",
    ]
    assert {case.purpose for case in registry.cases} == {
        VerificationPurpose.PRODUCT,
        VerificationPurpose.TOOLCHAIN,
    }
    assert {case.formula.ref for case in registry.cases} == {SHARED_FORMULA_ID}
    assert {case.id: case.policy.ref for case in registry.cases} == {
        "ptsip.product.canonical-contracts": "distribution.contract-integrity",
        "ptsip.product.package-contracts": "distribution.package-integrity",
        "ptsip.toolchain.release-workflow": "release.workflow-integrity",
        "ptsip.toolchain.routine-ci": "ci.verification-boundary",
    }
    assert all(
        not case.policy.ref.startswith(("product.", "toolchain."))
        and "pytest" not in case.policy.ref
        for case in registry.cases
    )
    assert {nodeid.split("::", 1)[0] for nodeid in RUNNER_NODEIDS.values()} == {
        "tests/ptsip/test_release_readiness_030.py"
    }


def test_registered_pytest_nodeids_exist_in_real_repository_module() -> None:
    available = _release_readiness_test_names()
    registered = {nodeid.split("::", 1)[1] for nodeid in RUNNER_NODEIDS.values()}

    assert registered <= available


def test_product_scope_executes_only_real_product_cases(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _record_commands(monkeypatch)

    results = run_selected_cases(
        repository_registry(),
        scope=SelectionScope.PRODUCT,
        executors=repository_executors(),
    )

    assert [result.case_id for result in results] == [
        "ptsip.product.canonical-contracts",
        "ptsip.product.package-contracts",
    ]
    assert all(result.purpose is VerificationPurpose.PRODUCT for result in results)
    assert all(result.outcome is VerificationOutcome.PASS for result in results)
    assert _nodeids_from_calls(calls) == [
        RUNNER_NODEIDS["pytest.product.canonical-contracts"],
        RUNNER_NODEIDS["pytest.product.package-contracts"],
    ]


def test_toolchain_scope_executes_only_real_toolchain_cases(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _record_commands(monkeypatch)

    results = run_selected_cases(
        repository_registry(),
        scope=SelectionScope.TOOLCHAIN,
        executors=repository_executors(),
    )

    assert [result.case_id for result in results] == [
        "ptsip.toolchain.release-workflow",
        "ptsip.toolchain.routine-ci",
    ]
    assert all(result.purpose is VerificationPurpose.TOOLCHAIN for result in results)
    assert all(result.outcome is VerificationOutcome.PASS for result in results)
    assert _nodeids_from_calls(calls) == [
        RUNNER_NODEIDS["pytest.toolchain.release-workflow"],
        RUNNER_NODEIDS["pytest.toolchain.routine-ci"],
    ]


def test_full_scope_executes_all_registered_cases_in_case_id_order(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _record_commands(monkeypatch)

    results = run_selected_cases(
        repository_registry(),
        scope=SelectionScope.FULL,
        executors=repository_executors(),
    )

    assert [result.case_id for result in results] == [
        "ptsip.product.canonical-contracts",
        "ptsip.product.package-contracts",
        "ptsip.toolchain.release-workflow",
        "ptsip.toolchain.routine-ci",
    ]
    assert _nodeids_from_calls(calls) == [
        RUNNER_NODEIDS["pytest.product.canonical-contracts"],
        RUNNER_NODEIDS["pytest.product.package-contracts"],
        RUNNER_NODEIDS["pytest.toolchain.release-workflow"],
        RUNNER_NODEIDS["pytest.toolchain.routine-ci"],
    ]


def test_full_repository_regression_path_remains_explicit_and_separate() -> None:
    assert FULL_REPOSITORY_PYTEST_ARGV[-2:] == ("-q", "tests")
    assert all(nodeid != "tests" for nodeid in RUNNER_NODEIDS.values())
