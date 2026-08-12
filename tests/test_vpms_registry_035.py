from __future__ import annotations

from vpms.model import VerificationPurpose
from vpms.registry import (
    RegistryDiagnosticCode,
    RegistryReferenceIndex,
    load_registry,
)


def _refs() -> RegistryReferenceIndex:
    return RegistryReferenceIndex(
        targets=("api-sdk", "repo-tool"),
        formulas=("structure.required-fields",),
        variables=("product.variables", "toolchain.variables"),
        policies=("product.policy", "toolchain.policy"),
        runners=("command",),
    )


def _case(
    *,
    case_id: str = "product.required-fields",
    purpose: str = "PRODUCT",
    target: str = "api-sdk",
    formula: str = "structure.required-fields",
    variables: str = "product.variables",
    policy: str = "product.policy",
    runner: str = "command",
) -> dict[str, str]:
    return {
        "id": case_id,
        "purpose": purpose,
        "target": target,
        "formula": formula,
        "variables": variables,
        "policy": policy,
        "runner": runner,
    }


def test_valid_definitions_load_into_deterministic_registry() -> None:
    definitions = [
        _case(
            case_id="toolchain.required-fields",
            purpose="TOOLCHAIN",
            target="repo-tool",
            variables="toolchain.variables",
            policy="toolchain.policy",
        ),
        _case(),
    ]

    result = load_registry(definitions, references=_refs())

    assert result.ok is True
    assert result.diagnostics == ()
    assert result.registry is not None
    assert [case.id for case in result.registry.cases] == [
        "product.required-fields",
        "toolchain.required-fields",
    ]
    case = result.registry.get_case("product.required-fields")
    assert case is not None
    assert case.purpose is VerificationPurpose.PRODUCT


def test_unknown_purpose_is_explicit_and_not_inferred() -> None:
    definition = _case(case_id="tests/product/path", purpose="UNKNOWN")

    result = load_registry([definition], references=_refs())

    assert result.ok is False
    assert result.registry is None
    assert [item.as_dict() for item in result.diagnostics] == [
        {
            "format": "vpms-registry-diagnostic/v1",
            "code": "UNKNOWN_PURPOSE",
            "location": "$[0].purpose",
            "message": "Unknown verification purpose: UNKNOWN.",
            "case_id": "tests/product/path",
        }
    ]


def test_missing_and_unknown_fields_are_machine_readable() -> None:
    definition = _case()
    del definition["policy"]
    definition["directory"] = "tests/product"

    result = load_registry([definition], references=_refs())

    assert [item.code for item in result.diagnostics] == [
        RegistryDiagnosticCode.UNKNOWN_FIELD,
        RegistryDiagnosticCode.MISSING_FIELD,
    ]
    payload = result.as_dict()
    assert payload["ok"] is False
    assert payload["registry"] is None
    assert payload["diagnostics"][0]["location"] == "$[0].directory"
    assert payload["diagnostics"][1]["location"] == "$[0].policy"


def test_unresolved_references_report_kind_and_identity_deterministically() -> None:
    definition = _case(
        formula="missing.formula",
        policy="missing.policy",
        target="missing.component",
    )

    result = load_registry([definition], references=_refs())

    assert [item.as_dict() for item in result.diagnostics] == [
        {
            "format": "vpms-registry-diagnostic/v1",
            "code": "UNRESOLVED_REFERENCE",
            "location": "$[0].formula",
            "message": "Unresolved formula reference: missing.formula.",
            "case_id": "product.required-fields",
            "reference_kind": "formula",
            "reference": "missing.formula",
        },
        {
            "format": "vpms-registry-diagnostic/v1",
            "code": "UNRESOLVED_REFERENCE",
            "location": "$[0].policy",
            "message": "Unresolved policy reference: missing.policy.",
            "case_id": "product.required-fields",
            "reference_kind": "policy",
            "reference": "missing.policy",
        },
        {
            "format": "vpms-registry-diagnostic/v1",
            "code": "UNRESOLVED_REFERENCE",
            "location": "$[0].target",
            "message": "Unresolved target reference: missing.component.",
            "case_id": "product.required-fields",
            "reference_kind": "target",
            "reference": "missing.component",
        },
    ]


def test_duplicate_case_ids_fail_without_partial_registry() -> None:
    definitions = [_case(), _case()]

    result = load_registry(definitions, references=_refs())

    assert result.ok is False
    assert result.registry is None
    assert any(item.code is RegistryDiagnosticCode.DUPLICATE_CASE_ID for item in result.diagnostics)


def test_malformed_inputs_fail_explicitly() -> None:
    root = load_registry({"cases": []}, references=_refs())
    assert root.diagnostics[0].code is RegistryDiagnosticCode.MALFORMED_DEFINITIONS

    case = load_registry(["not-a-mapping"], references=_refs())
    assert case.diagnostics[0].code is RegistryDiagnosticCode.MALFORMED_CASE


def test_reference_index_normalization_is_stable() -> None:
    refs = RegistryReferenceIndex(
        formulas=("z", "a", "z"),
        targets=("b", "a"),
    )

    assert refs.formulas == ("a", "z")
    assert refs.targets == ("a", "b")


def test_success_payload_is_machine_readable() -> None:
    result = load_registry([_case()], references=_refs())

    assert result.as_dict() == {
        "ok": True,
        "registry": {
            "references": {
                "targets": ("api-sdk", "repo-tool"),
                "formulas": ("structure.required-fields",),
                "variables": ("product.variables", "toolchain.variables"),
                "policies": ("product.policy", "toolchain.policy"),
                "runners": ("command",),
            },
            "cases": [
                {
                    "id": "product.required-fields",
                    "purpose": "PRODUCT",
                    "target": {"component_id": "api-sdk"},
                    "formula": {"ref": "structure.required-fields"},
                    "variables": {"ref": "product.variables"},
                    "policy": {"ref": "product.policy"},
                    "runner": {"ref": "command"},
                }
            ],
        },
        "diagnostics": [],
    }
