from __future__ import annotations

import subprocess

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
from vpms.domain.registry import Registry, RegistryReferenceIndex
from vpms.domain.selector import SelectionScope
from vpms.execution.adapters.command import CommandExecutor
from vpms.execution.runner import run_selected_cases


_SHARED_FORMULA = FormulaRef(ref="structure.required-fields")


def _case(
    case_id: str,
    purpose: VerificationPurpose,
    target: str,
    variables: str,
    policy: str,
    runner: str,
) -> VerificationCase:
    return VerificationCase(
        id=case_id,
        purpose=purpose,
        target=TargetRef(component_id=target),
        formula=_SHARED_FORMULA,
        variables=VariablesRef(ref=variables),
        policy=PolicyRef(ref=policy),
        runner=RunnerRef(ref=runner),
    )


def _registry() -> Registry:
    return Registry(
        references=RegistryReferenceIndex(
            targets=("product-component", "toolchain-component"),
            formulas=(_SHARED_FORMULA.ref,),
            variables=("product.variables", "toolchain.variables"),
            policies=("product.policy", "toolchain.policy"),
            runners=("command.product", "command.toolchain"),
        ),
        cases=(
            _case(
                "toolchain.integration",
                VerificationPurpose.TOOLCHAIN,
                "toolchain-component",
                "toolchain.variables",
                "toolchain.policy",
                "command.toolchain",
            ),
            _case(
                "product.integration",
                VerificationPurpose.PRODUCT,
                "product-component",
                "product.variables",
                "product.policy",
                "command.product",
            ),
        ),
    )


def _executors() -> dict[str, CommandExecutor]:
    return {
        "command.product": CommandExecutor(argv=("product-command",)),
        "command.toolchain": CommandExecutor(argv=("toolchain-command",)),
    }


def _patch_commands(
    monkeypatch: pytest.MonkeyPatch,
    *,
    returncodes: dict[str, int] | None = None,
) -> list[tuple[str, ...]]:
    calls: list[tuple[str, ...]] = []
    codes = returncodes or {}

    def _run(
        argv: tuple[str, ...],
        **kwargs: object,
    ) -> subprocess.CompletedProcess[str]:
        calls.append(tuple(argv))
        return subprocess.CompletedProcess(
            argv,
            codes.get(argv[0], 0),
            stdout="",
            stderr="",
        )

    monkeypatch.setattr(subprocess, "run", _run)
    return calls


def test_product_selection_executes_only_product_case(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _patch_commands(monkeypatch)

    results = run_selected_cases(
        _registry(),
        scope=SelectionScope.PRODUCT,
        executors=_executors(),
    )

    assert [result.case_id for result in results] == ["product.integration"]
    assert results[0].purpose is VerificationPurpose.PRODUCT
    assert results[0].outcome is VerificationOutcome.PASS
    assert calls == [("product-command",)]


def test_toolchain_selection_executes_only_toolchain_case(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _patch_commands(monkeypatch)

    results = run_selected_cases(
        _registry(),
        scope=SelectionScope.TOOLCHAIN,
        executors=_executors(),
    )

    assert [result.case_id for result in results] == ["toolchain.integration"]
    assert results[0].purpose is VerificationPurpose.TOOLCHAIN
    assert results[0].outcome is VerificationOutcome.PASS
    assert calls == [("toolchain-command",)]


def test_full_selection_executes_both_in_deterministic_case_order(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _patch_commands(monkeypatch)

    results = run_selected_cases(
        _registry(),
        scope=SelectionScope.FULL,
        executors=_executors(),
    )

    assert [result.case_id for result in results] == [
        "product.integration",
        "toolchain.integration",
    ]
    assert calls == [("product-command",), ("toolchain-command",)]


def test_shared_formula_does_not_couple_scope_or_case_owned_state(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _patch_commands(monkeypatch)
    registry = _registry()
    product = registry.get_case("product.integration")
    toolchain = registry.get_case("toolchain.integration")
    assert product is not None and toolchain is not None
    assert product.formula is toolchain.formula
    assert product.purpose is not toolchain.purpose
    assert product.variables != toolchain.variables
    assert product.policy != toolchain.policy
    assert product.target != toolchain.target

    results = run_selected_cases(
        registry,
        scope=SelectionScope.PRODUCT,
        executors={"command.product": _executors()["command.product"]},
    )

    assert [result.case_id for result in results] == ["product.integration"]
    assert results[0].purpose is product.purpose
    assert results[0].target is product.target
    assert calls == [("product-command",)]


def test_missing_selected_runner_is_rejected_before_partial_execution(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = _patch_commands(monkeypatch)

    with pytest.raises(
        ValueError,
        match=r"Missing executor registration for runner\(s\): command\.toolchain\.",
    ):
        run_selected_cases(
            _registry(),
            scope=SelectionScope.FULL,
            executors={"command.product": _executors()["command.product"]},
        )

    assert calls == []


def test_full_execution_preserves_independent_outcomes_and_identity(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    _patch_commands(
        monkeypatch,
        returncodes={"product-command": 0, "toolchain-command": 9},
    )

    results = run_selected_cases(
        _registry(),
        scope=SelectionScope.FULL,
        executors=_executors(),
    )

    by_id = {result.case_id: result for result in results}
    assert by_id["product.integration"].outcome is VerificationOutcome.PASS
    assert by_id["toolchain.integration"].outcome is VerificationOutcome.FAIL
    assert by_id["product.integration"].purpose is VerificationPurpose.PRODUCT
    assert by_id["toolchain.integration"].purpose is VerificationPurpose.TOOLCHAIN
    assert by_id["product.integration"].target.component_id == "product-component"
    assert by_id["toolchain.integration"].target.component_id == "toolchain-component"


def test_integrated_execution_does_not_mutate_registry_or_cases(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    _patch_commands(monkeypatch)
    registry = _registry()
    before = registry.as_dict()

    run_selected_cases(
        registry,
        scope=SelectionScope.FULL,
        executors=_executors(),
    )

    assert registry.as_dict() == before
