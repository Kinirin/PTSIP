from __future__ import annotations

import json
import re
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator


ROOT = Path(__file__).resolve().parents[2]


def _json(path: str) -> dict[str, object]:
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def _yaml(path: str) -> dict[str, object]:
    return yaml.safe_load((ROOT / path).read_text(encoding="utf-8"))


def test_canonical_json_schemas_are_valid_draft_2020_12() -> None:
    for path in (
        "schemas/ptsip-profile.schema.json",
        "schemas/ptsip-profile-pp-1.01.schema.json",
        "schemas/ptsip-agent-classification.schema.json",
        "schemas/ptsip-diagnostic.schema.json",
        "schemas/ptsip-artifact-evidence.schema.json",
    ):
        Draft202012Validator.check_schema(_json(path))


def test_registry_preserves_exactly_five_canonical_lifecycle_classifications() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    assert registry["specification"]["version"] == "0.3.7-draft"
    assert [item["id"] for item in registry["classifications"]] == [
        "PRODUCT",
        "DEVELOPMENT_TOOLING",
        "DELIVERY",
        "OPERATIONS",
        "NEUTRAL_CONTRACT",
    ]
    retired = {item["id"] for item in registry.get("retired_classifications", [])}
    assert "TOOLCHAIN" in retired
    node_scopes = {item["id"] for item in registry["evidence_node_scopes"]}
    assert {"EXTERNAL_DEPENDENCY", "PLATFORM", "UNRESOLVED_TARGET"} <= node_scopes
    assert not (node_scopes & {"PRODUCT", "DEVELOPMENT_TOOLING", "DELIVERY", "OPERATIONS", "NEUTRAL_CONTRACT"})


def test_registry_contains_lifecycle_map_migration_and_authority_rule_ids() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    rules = {item["id"]: item for item in registry["rules"]}
    assert {
        "PTSIP-CLS-011",
        "PTSIP-ART-001",
        "PTSIP-RMAP-004",
        "PTSIP-RMAP-012",
        "PTSIP-MIG-001",
        "PTSIP-MIG-003",
        "PTSIP-MIG-004",
        "PTSIP-MIG-015",
        "PTSIP-EVD-005",
        "PTSIP-DIA-001",
        "PTSIP-POL-001",
        "PTSIP-ADP-001",
        "PTSIP-AUT-001",
        "PTSIP-AUT-007",
    } <= set(rules)
    assert rules["PTSIP-ADP-001"]["applies_to"] == "implementation"
    assert rules["PTSIP-MIG-001"]["applies_to"] == "migration_implementation"
    assert rules["PTSIP-AUT-001"]["applies_to"] == "distributed_coordination_implementation"


def test_every_registry_rule_id_has_a_normative_spec_heading() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    spec_text = "\n".join(
        (ROOT / path).read_text(encoding="utf-8")
        for path in (
            "spec/PTSIP-SPEC.md",
            "spec/PTSIP-RESPONSIBILITY-MAP.md",
            "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
        )
    )
    headings = set(re.findall(r"^###\s+(PTSIP-[A-Z]+-\d{3})\b", spec_text, flags=re.MULTILINE))
    registered = {item["id"] for item in registry["rules"]}
    assert registered <= headings


def _minimal_profile() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "product",
                "classification": "PRODUCT",
                "roles": ["IMPLEMENTATION"],
                "include": ["product/**"],
                "purpose": "product_runtime",
            },
            {
                "id": "dev-tools",
                "classification": "DEVELOPMENT_TOOLING",
                "roles": ["VERIFICATION", "AUTOMATION"],
                "include": ["tools/**"],
                "purpose": "development_tooling",
            },
        ],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }


def test_example_profile_validates_against_canonical_schema() -> None:
    schema = _json("schemas/ptsip-profile-pp-1.01.schema.json")
    profile = _yaml("profiles/example.ptsip.yaml")
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_accepts_explicit_mode_and_optional_component_policy() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["component_dependency_policy"] = {
        "default": "deny",
        "allow": [{"from": "dev-tools", "to": "product"}],
    }
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_accepts_roles_relationships_and_associated_artifacts() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["associated_artifacts"] = [
        {
            "id": "dev-tools-docs",
            "anchor": "dev-tools",
            "include": ["docs/tools/**"],
            "purpose": "tool_owned_documentation",
        }
    ]
    profile["relationships"] = [
        {
            "id": "docs-document-tools",
            "from": "dev-tools-docs",
            "to": "dev-tools",
            "type": "DOCUMENTS",
        },
        {
            "id": "tools-verify-product",
            "from": "dev-tools",
            "to": "product",
            "type": "VERIFIES",
        },
    ]
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_rejects_legacy_toolchain_classification() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["components"][1]["classification"] = "TOOLCHAIN"
    with pytest.raises(Exception):
        Draft202012Validator(schema).validate(profile)


def test_profile_schema_rejects_nonproduct_product_runtime_implementation() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    for classification in ("DEVELOPMENT_TOOLING", "DELIVERY", "OPERATIONS"):
        profile = _minimal_profile()
        profile["components"][1].update(
            {
                "classification": classification,
                "runtime_required": True,
                "executable": True,
            }
        )
        with pytest.raises(Exception):
            Draft202012Validator(schema).validate(profile)


def test_profile_schema_rejects_legacy_boundaries() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["boundaries"] = {
        "product": {"roots": ["product"]},
        "toolchain": {"roots": ["tools"]},
    }
    with pytest.raises(Exception):
        Draft202012Validator(schema).validate(profile)


def test_template_profile_requires_explicit_version_bound_template_reference() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "responsibility_map": {
            "mode": "template",
            "template": {"id": "python-package", "revision": "template-v1"},
        },
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    Draft202012Validator(schema).validate(profile)
    profile["components"] = _minimal_profile()["components"]
    with pytest.raises(Exception):
        Draft202012Validator(schema).validate(profile)


def test_hybrid_profile_uses_id_addressed_overrides() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "responsibility_map": {
            "mode": "hybrid",
            "template": {"id": "python-package", "revision": "template-v1"},
            "overrides": {
                "components": [
                    {
                        "id": "release",
                        "classification": "DELIVERY",
                        "roles": ["AUTOMATION"],
                        "include": [".github/workflows/release.yml"],
                        "purpose": "release_delivery",
                        "shipped": False,
                        "runtime_required": False,
                    }
                ]
            },
        },
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_rejects_legacy_exception_waiver() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["exceptions"] = []
    with pytest.raises(Exception):
        Draft202012Validator(schema).validate(profile)


def test_retired_exception_rule_is_not_active() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    active = {item["id"] for item in registry["rules"]}
    retired = {item["id"] for item in registry.get("retired_rules", [])}
    assert "PTSIP-EXC-001" not in active
    assert "PTSIP-EXC-001" in retired


def test_diagnostic_schema_distinguishes_rule_and_finding_identity() -> None:
    schema = _json("schemas/ptsip-diagnostic.schema.json")
    payload = {
        "format": "ptsip-diagnostic/v1",
        "diagnostic_id": "finding-0001",
        "rule_id": "PTSIP-DEP-001",
        "outcome_effect": "NON_CONFORMANT",
        "severity": "ERROR",
        "source_component": "product",
        "target_component": "delivery",
        "evidence_ids": ["edge:1"],
        "message": "Product runtime depends on Delivery implementation.",
        "evaluator": {"id": "reference", "provenance": "OBSERVED"},
    }
    Draft202012Validator(schema).validate(payload)


def test_artifact_evidence_schema_separates_product_owner_from_delivery_producer() -> None:
    schema = _json("schemas/ptsip-artifact-evidence.schema.json")
    payload = {
        "format": "ptsip-artifact-evidence/v1",
        "artifact_id": "installer",
        "classification": "PRODUCT",
        "producer_component": "installer-builder",
        "artifact_type": "installer",
        "shipping_scope": "customer-distribution",
        "contents": {
            "paths": ["app/runtime.bin"],
            "components": ["product-runtime"],
            "complete": True,
        },
        "derivation": [
            {"relation": "BUILDS", "source": "installer-builder"},
            {"relation": "PACKAGES", "source": "installer-builder"},
        ],
        "provenance": "OBSERVED",
        "evidence_ids": ["artifact:installer:contents"],
    }
    Draft202012Validator(schema).validate(payload)
