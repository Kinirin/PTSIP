from __future__ import annotations

import inspect
from dataclasses import FrozenInstanceError

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
    VerificationResult,
)


def _case(*, case_id: str, purpose: VerificationPurpose, formula: FormulaRef) -> VerificationCase:
    suffix = purpose.value.lower()
    return VerificationCase(
        id=case_id,
        purpose=purpose,
        target=TargetRef(component_id=f"{suffix}.component"),
        formula=formula,
        variables=VariablesRef(ref=f"{suffix}.variables"),
        policy=PolicyRef(ref=f"{suffix}.policy"),
        runner=RunnerRef(ref=f"{suffix}.runner"),
    )


def test_verification_purpose_is_explicit_and_closed() -> None:
    assert {item.value for item in VerificationPurpose} == {"PRODUCT", "TOOLCHAIN"}
    signature = inspect.signature(VerificationCase)
    assert signature.parameters["purpose"].default is inspect.Parameter.empty

    with pytest.raises(TypeError):
        VerificationCase(
            id="tests/product-like-path",
            target=TargetRef(component_id="component"),
            formula=FormulaRef(ref="formula"),
            variables=VariablesRef(ref="variables"),
            policy=PolicyRef(ref="policy"),
            runner=RunnerRef(ref="runner"),
        )


def test_reference_types_preserve_distinct_semantic_identities() -> None:
    shared_text = "shared.identity"

    formula = FormulaRef(ref=shared_text)
    variables = VariablesRef(ref=shared_text)
    policy = PolicyRef(ref=shared_text)
    runner = RunnerRef(ref=shared_text)

    assert formula.ref == variables.ref == policy.ref == runner.ref
    assert type(formula) is FormulaRef
    assert type(variables) is VariablesRef
    assert type(policy) is PolicyRef
    assert type(runner) is RunnerRef
    assert formula != variables
    assert variables != policy
    assert policy != runner


def test_cases_can_share_formula_without_collapsing_purpose_owned_state() -> None:
    formula = FormulaRef(ref="structure.required-fields")

    product = _case(
        case_id="product.required-fields",
        purpose=VerificationPurpose.PRODUCT,
        formula=formula,
    )
    toolchain = _case(
        case_id="toolchain.required-fields",
        purpose=VerificationPurpose.TOOLCHAIN,
        formula=formula,
    )

    assert product.formula is toolchain.formula
    assert product.purpose is VerificationPurpose.PRODUCT
    assert toolchain.purpose is VerificationPurpose.TOOLCHAIN
    assert product.target != toolchain.target
    assert product.variables != toolchain.variables
    assert product.policy != toolchain.policy
    assert product.runner != toolchain.runner


def test_case_purpose_is_not_inferred_from_path_like_identity() -> None:
    case = VerificationCase(
        id="tests/toolchain/path-shaped-name",
        purpose=VerificationPurpose.PRODUCT,
        target=TargetRef(component_id="product.component"),
        formula=FormulaRef(ref="formula"),
        variables=VariablesRef(ref="variables"),
        policy=PolicyRef(ref="policy"),
        runner=RunnerRef(ref="runner"),
    )

    assert case.purpose is VerificationPurpose.PRODUCT


def test_verification_result_preserves_case_purpose_target_and_outcome() -> None:
    result = VerificationResult(
        case_id="product.required-fields",
        purpose=VerificationPurpose.PRODUCT,
        target=TargetRef(component_id="api-sdk"),
        outcome=VerificationOutcome.FAIL,
        diagnostics=("missing field: runtime_required",),
        failure_detail="required field set did not match policy",
    )

    assert result.as_dict() == {
        "case_id": "product.required-fields",
        "purpose": "PRODUCT",
        "target": {"component_id": "api-sdk"},
        "outcome": "FAIL",
        "diagnostics": ("missing field: runtime_required",),
        "failure_detail": "required field set did not match policy",
    }


def test_model_values_are_immutable_semantic_records() -> None:
    formula = FormulaRef(ref="formula")

    with pytest.raises(FrozenInstanceError):
        formula.ref = "changed"
