from __future__ import annotations

import ast
from pathlib import Path

import pytest

from vpms.domain.model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationPurpose,
)
from vpms.domain.registry import Registry, RegistryReferenceIndex
from vpms.domain.selector import SelectionScope, select_cases

_REPO_ROOT = Path(__file__).resolve().parents[3]
_SHARED_FORMULA = FormulaRef(ref="structure.required-fields")


def _case(case_id: str, purpose: VerificationPurpose, target: str, variables: str, policy: str) -> VerificationCase:
    return VerificationCase(
        id=case_id,
        purpose=purpose,
        target=TargetRef(component_id=target),
        formula=_SHARED_FORMULA,
        variables=VariablesRef(ref=variables),
        policy=PolicyRef(ref=policy),
        runner=RunnerRef(ref="command"),
    )


def _registry() -> Registry:
    return Registry(
        references=RegistryReferenceIndex(),
        cases=(
            _case("toolchain.required-fields", VerificationPurpose.TOOLCHAIN, "toolchain-component", "toolchain.variables", "toolchain.policy"),
            _case("product.required-fields", VerificationPurpose.PRODUCT, "product-component", "product.variables", "product.policy"),
        ),
    )


def test_scope_is_separate_from_verification_purpose() -> None:
    assert [scope.value for scope in SelectionScope] == ["PRODUCT", "TOOLCHAIN", "FULL"]
    assert {purpose.value for purpose in VerificationPurpose} == {"PRODUCT", "TOOLCHAIN"}
    assert SelectionScope.FULL.value not in {purpose.value for purpose in VerificationPurpose}


def test_product_scope_selects_only_product_cases() -> None:
    selected = select_cases(_registry(), scope=SelectionScope.PRODUCT)
    assert [case.id for case in selected] == ["product.required-fields"]
    assert all(case.purpose is VerificationPurpose.PRODUCT for case in selected)


def test_toolchain_scope_selects_only_toolchain_cases() -> None:
    selected = select_cases(_registry(), scope=SelectionScope.TOOLCHAIN)
    assert [case.id for case in selected] == ["toolchain.required-fields"]
    assert all(case.purpose is VerificationPurpose.TOOLCHAIN for case in selected)


def test_full_scope_selects_explicit_union() -> None:
    selected = select_cases(_registry(), scope=SelectionScope.FULL)
    assert [case.id for case in selected] == ["product.required-fields", "toolchain.required-fields"]
    assert {case.purpose for case in selected} == {VerificationPurpose.PRODUCT, VerificationPurpose.TOOLCHAIN}


def test_selection_order_is_deterministic_by_case_id() -> None:
    registry = _registry()
    assert [case.id for case in registry.cases] == ["toolchain.required-fields", "product.required-fields"]
    assert [case.id for case in select_cases(registry, scope=SelectionScope.FULL)] == ["product.required-fields", "toolchain.required-fields"]


def test_path_like_case_id_does_not_override_purpose() -> None:
    registry = Registry(
        references=RegistryReferenceIndex(),
        cases=(
            _case("tests/toolchain/product-case", VerificationPurpose.PRODUCT, "p", "pv", "pp"),
            _case("tests/product/toolchain-case", VerificationPurpose.TOOLCHAIN, "t", "tv", "tp"),
        ),
    )
    assert [case.id for case in select_cases(registry, scope=SelectionScope.PRODUCT)] == ["tests/toolchain/product-case"]
    assert [case.id for case in select_cases(registry, scope=SelectionScope.TOOLCHAIN)] == ["tests/product/toolchain-case"]


def test_shared_formula_does_not_couple_selection() -> None:
    registry = _registry()
    product = select_cases(registry, scope=SelectionScope.PRODUCT)[0]
    toolchain = select_cases(registry, scope=SelectionScope.TOOLCHAIN)[0]
    assert product.formula is toolchain.formula
    assert product.purpose is VerificationPurpose.PRODUCT
    assert toolchain.purpose is VerificationPurpose.TOOLCHAIN
    assert product.variables != toolchain.variables
    assert product.policy != toolchain.policy


def test_selection_preserves_existing_case_objects() -> None:
    registry = _registry()
    originals = {case.id: case for case in registry.cases}
    for case in select_cases(registry, scope=SelectionScope.FULL):
        assert case is originals[case.id]


def test_selection_returns_tuple_without_mutating_registry() -> None:
    registry = _registry()
    before = registry.cases
    selected = select_cases(registry, scope=SelectionScope.PRODUCT)
    assert isinstance(selected, tuple)
    assert registry.cases is before


@pytest.mark.parametrize("scope", ["PRODUCT", "TOOLCHAIN", "FULL", None])
def test_raw_scope_values_are_not_coerced(scope: object) -> None:
    with pytest.raises(TypeError, match="scope must be a SelectionScope"):
        select_cases(_registry(), scope=scope)  # type: ignore[arg-type]


def test_non_registry_input_is_rejected() -> None:
    with pytest.raises(TypeError, match="registry must be a VPMS Registry"):
        select_cases((), scope=SelectionScope.FULL)  # type: ignore[arg-type]


def test_selector_has_no_execution_or_path_discovery_coupling() -> None:
    source = _REPO_ROOT / "src" / "vpms" / "domain" / "selector.py"
    tree = ast.parse(source.read_text(encoding="utf-8"))
    imports = set()
    calls = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module is not None:
            imports.add(node.module)
        elif isinstance(node, ast.Call) and isinstance(node.func, ast.Name):
            calls.add(node.func.id)
    assert "subprocess" not in imports
    assert all("execution" not in name for name in imports)
    assert all("pathlib" not in name for name in imports)
    assert not {"glob", "rglob", "walk", "run", "Popen"} & calls
