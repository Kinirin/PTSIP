from __future__ import annotations

import ast
from dataclasses import fields
from pathlib import Path

from vpms.domain.model import VerificationPurpose
from vpms.domain.registry import (
    RegisteredFormula,
    RegistryDiagnosticCode,
    RegistryReferenceIndex,
    load_registry,
    register_formulas,
)


_REPO_ROOT = Path(__file__).resolve().parents[3]
_FORMULA = "structure.required-fields"


def _references(*, include_formula: bool = True) -> RegistryReferenceIndex:
    base = RegistryReferenceIndex(
        targets=("api-sdk", "repo-tool"),
        variables=("product.variables", "toolchain.variables", "product.alt.variables"),
        policies=("product.policy", "toolchain.policy", "product.alt.policy"),
        runners=("command",),
    )
    if not include_formula:
        return base
    return base.with_registered_formulas(register_formulas([_FORMULA]))


def _case(
    *,
    case_id: str,
    purpose: str,
    target: str,
    variables: str,
    policy: str,
    formula: str = _FORMULA,
) -> dict[str, str]:
    return {
        "id": case_id,
        "purpose": purpose,
        "target": target,
        "formula": formula,
        "variables": variables,
        "policy": policy,
        "runner": "command",
    }


def _cross_purpose_registry():
    definitions = [
        _case(
            case_id="product.required-fields",
            purpose="PRODUCT",
            target="api-sdk",
            variables="product.variables",
            policy="product.policy",
        ),
        _case(
            case_id="toolchain.required-fields",
            purpose="TOOLCHAIN",
            target="repo-tool",
            variables="toolchain.variables",
            policy="toolchain.policy",
        ),
    ]
    result = load_registry(definitions, references=_references())
    assert result.ok
    assert result.registry is not None
    return result.registry


def test_formula_identity_is_explicitly_registered() -> None:
    formulas = register_formulas(["z.formula", _FORMULA])

    assert formulas.identities == (_FORMULA, "z.formula")
    registered = formulas.get_formula(_FORMULA)
    assert registered == RegisteredFormula(id=_FORMULA)
    assert formulas.as_dict() == {
        "formulas": [
            {"id": _FORMULA},
            {"id": "z.formula"},
        ]
    }


def test_unregistered_formula_reference_remains_deterministic_registry_error() -> None:
    result = load_registry(
        [
            _case(
                case_id="product.required-fields",
                purpose="PRODUCT",
                target="api-sdk",
                variables="product.variables",
                policy="product.policy",
            )
        ],
        references=_references(include_formula=False),
    )

    assert result.registry is None
    assert [item.code for item in result.diagnostics] == [
        RegistryDiagnosticCode.UNRESOLVED_REFERENCE
    ]
    assert result.diagnostics[0].as_dict() == {
        "format": "vpms-registry-diagnostic/v1",
        "code": "UNRESOLVED_REFERENCE",
        "location": "$[0].formula",
        "message": f"Unresolved formula reference: {_FORMULA}.",
        "case_id": "product.required-fields",
        "reference_kind": "formula",
        "reference": _FORMULA,
    }


def test_one_registered_formula_can_be_reused_by_multiple_cases() -> None:
    definitions = [
        _case(
            case_id="product.required-fields",
            purpose="PRODUCT",
            target="api-sdk",
            variables="product.variables",
            policy="product.policy",
        ),
        _case(
            case_id="product.required-fields-alt",
            purpose="PRODUCT",
            target="api-sdk",
            variables="product.alt.variables",
            policy="product.alt.policy",
        ),
    ]

    result = load_registry(definitions, references=_references())

    assert result.ok
    assert result.registry is not None
    assert {case.formula.ref for case in result.registry.cases} == {_FORMULA}
    assert len(result.registry.cases) == 2


def test_registered_formula_can_cross_product_and_toolchain_purposes() -> None:
    registry = _cross_purpose_registry()

    purposes = {case.purpose for case in registry.cases}
    assert purposes == {VerificationPurpose.PRODUCT, VerificationPurpose.TOOLCHAIN}
    assert {case.formula.ref for case in registry.cases} == {_FORMULA}


def test_formula_reuse_does_not_change_case_purpose() -> None:
    registry = _cross_purpose_registry()

    product = registry.get_case("product.required-fields")
    toolchain = registry.get_case("toolchain.required-fields")
    assert product is not None and toolchain is not None
    assert product.purpose is VerificationPurpose.PRODUCT
    assert toolchain.purpose is VerificationPurpose.TOOLCHAIN


def test_formula_reuse_does_not_share_variables() -> None:
    registry = _cross_purpose_registry()

    product = registry.get_case("product.required-fields")
    toolchain = registry.get_case("toolchain.required-fields")
    assert product is not None and toolchain is not None
    assert product.variables.ref == "product.variables"
    assert toolchain.variables.ref == "toolchain.variables"
    assert product.variables != toolchain.variables


def test_formula_reuse_does_not_share_policy() -> None:
    registry = _cross_purpose_registry()

    product = registry.get_case("product.required-fields")
    toolchain = registry.get_case("toolchain.required-fields")
    assert product is not None and toolchain is not None
    assert product.policy.ref == "product.policy"
    assert toolchain.policy.ref == "toolchain.policy"
    assert product.policy != toolchain.policy


def test_formula_reuse_does_not_merge_target_or_case_identity() -> None:
    registry = _cross_purpose_registry()

    product = registry.get_case("product.required-fields")
    toolchain = registry.get_case("toolchain.required-fields")
    assert product is not None and toolchain is not None
    assert product.id != toolchain.id
    assert product.target.component_id == "api-sdk"
    assert toolchain.target.component_id == "repo-tool"
    assert product.target != toolchain.target


def test_registered_formula_has_no_purpose_or_case_owned_fields() -> None:
    assert [field.name for field in fields(RegisteredFormula)] == ["id"]
    assert not {
        "purpose",
        "target",
        "variables",
        "policy",
        "runner",
    } & {field.name for field in fields(RegisteredFormula)}


def test_registry_serialization_preserves_cross_purpose_case_identity() -> None:
    payload = _cross_purpose_registry().as_dict()
    cases = {case["id"]: case for case in payload["cases"]}

    assert payload["references"]["formulas"] == (_FORMULA,)
    assert cases["product.required-fields"]["purpose"] == "PRODUCT"
    assert cases["toolchain.required-fields"]["purpose"] == "TOOLCHAIN"
    assert cases["product.required-fields"]["formula"] == {"ref": _FORMULA}
    assert cases["toolchain.required-fields"]["formula"] == {"ref": _FORMULA}
    assert cases["product.required-fields"]["variables"] != cases["toolchain.required-fields"]["variables"]
    assert cases["product.required-fields"]["policy"] != cases["toolchain.required-fields"]["policy"]
    assert cases["product.required-fields"]["target"] != cases["toolchain.required-fields"]["target"]


def test_registration_preserves_wu03_reference_index_contract() -> None:
    refs = RegistryReferenceIndex(
        targets=("repo-tool", "api-sdk"),
        formulas=("legacy.formula",),
        variables=("toolchain.variables", "product.variables"),
        policies=("toolchain.policy", "product.policy"),
        runners=("command",),
    ).with_registered_formulas(register_formulas([_FORMULA]))

    assert refs.targets == ("api-sdk", "repo-tool")
    assert refs.formulas == (_FORMULA,)
    assert refs.variables == ("product.variables", "toolchain.variables")
    assert refs.policies == ("product.policy", "toolchain.policy")
    assert refs.runners == ("command",)


def test_wu07_registry_scope_has_no_selection_or_execution_coupling() -> None:
    source = _REPO_ROOT / "src" / "vpms" / "domain" / "registry.py"
    tree = ast.parse(source.read_text(encoding="utf-8"))

    imports: set[str] = set()
    function_names: set[str] = set()
    for node in tree.body:
        if isinstance(node, ast.Import):
            imports.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module is not None:
            imports.add(node.module)
        elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            function_names.add(node.name)

    assert "subprocess" not in imports
    assert all("execution" not in name for name in imports)
    assert "select_cases" not in function_names
    assert "run_case" not in function_names
    assert "execute" not in function_names
