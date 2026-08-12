from __future__ import annotations

import json
import re
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator


ROOT = Path(__file__).resolve().parents[1]


def _json(path: str) -> dict[str, object]:
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def _yaml(path: str) -> dict[str, object]:
    return yaml.safe_load((ROOT / path).read_text(encoding="utf-8"))


def test_canonical_json_schemas_are_valid_draft_2020_12() -> None:
    for path in (
        "schemas/ptsip-profile.schema.json",
        "schemas/ptsip-agent-classification.schema.json",
        "schemas/ptsip-diagnostic.schema.json",
        "schemas/ptsip-artifact-evidence.schema.json",
    ):
        Draft202012Validator.check_schema(_json(path))


def test_registry_preserves_exactly_three_architecture_classifications() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    assert registry["specification"]["version"] == "0.3.4-draft"
    assert [item["id"] for item in registry["classifications"]] == [
        "PRODUCT",
        "TOOLCHAIN",
        "NEUTRAL_CONTRACT",
    ]
    node_scopes = {item["id"] for item in registry["evidence_node_scopes"]}
    assert {"EXTERNAL_DEPENDENCY", "PLATFORM", "UNRESOLVED_TARGET"} <= node_scopes
    assert not (node_scopes & {"PRODUCT", "TOOLCHAIN", "NEUTRAL_CONTRACT"})


def test_registry_contains_adoption_and_distributed_authority_rule_ids() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    rules = {item["id"]: item for item in registry["rules"]}
    assert {
        "PTSIP-CLS-002",
        "PTSIP-ART-001",
        "PTSIP-EVD-003",
        "PTSIP-EVD-004",
        "PTSIP-DIA-001",
        "PTSIP-POL-001",
        "PTSIP-ADP-001",
        "PTSIP-AUT-001",
        "PTSIP-AUT-002",
        "PTSIP-AUT-003",
        "PTSIP-AUT-004",
        "PTSIP-AUT-005",
        "PTSIP-AUT-006",
        "PTSIP-AUT-007",
    } <= set(rules)
    assert rules["PTSIP-ADP-001"]["applies_to"] == "implementation"
    assert rules["PTSIP-AUT-001"]["applies_to"] == "distributed_coordination_implementation"


def test_every_registry_rule_id_has_a_normative_spec_heading() -> None:
    registry = _yaml("registry/ptsip-registry.yaml")["ptsip_registry"]
    spec_text = (ROOT / "spec/PTSIP-SPEC.md").read_text(encoding="utf-8")
    headings = set(re.findall(r"^###\s+(PTSIP-[A-Z]+-\d{3})\b", spec_text, flags=re.MULTILINE))
    registered = {item["id"] for item in registry["rules"]}
    assert registered <= headings


def _minimal_profile() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "components": [
            {
                "id": "product",
                "classification": "PRODUCT",
                "include": ["product/**"],
                "purpose": "product_runtime",
            },
            {
                "id": "toolchain",
                "classification": "TOOLCHAIN",
                "include": ["tools/**"],
                "purpose": "development_tooling",
            },
        ],
        "policies": {
            "product_to_toolchain_runtime_dependency": "deny",
            "toolchain_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }


def test_example_profile_validates_against_canonical_schema() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _yaml("profiles/example.ptsip.yaml")
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_accepts_component_mode_and_optional_component_policy() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["component_dependency_policy"] = {
        "default": "deny",
        "allow": [{"from": "toolchain", "to": "product"}],
    }
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_accepts_lossless_structured_adoption_facts() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    tool = profile["components"][1]
    tool.update(
        {
            "shipped": False,
            "runtime_required": False,
            "lifecycle_owner": "DEVELOPMENT_TOOLING",
            "executable": True,
        }
    )
    Draft202012Validator(schema).validate(profile)


def test_profile_schema_rejects_toolchain_runtime_required_true() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["components"][1].update(
        {
            "runtime_required": True,
            "lifecycle_owner": "DEVELOPMENT_TOOLING",
            "executable": True,
        }
    )
    with pytest.raises(Exception):
        Draft202012Validator(schema).validate(profile)


def test_profile_schema_rejects_boundaries_and_components_together() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = _minimal_profile()
    profile["boundaries"] = {
        "product": {"roots": ["product"]},
        "toolchain": {"roots": ["tools"]},
    }
    with pytest.raises(Exception):
        Draft202012Validator(schema).validate(profile)


def test_component_dependency_policy_requires_component_mode() -> None:
    schema = _json("schemas/ptsip-profile.schema.json")
    profile = {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "boundaries": {
            "product": {"roots": ["product"]},
            "toolchain": {"roots": ["tools"]},
        },
        "component_dependency_policy": {"default": "deny"},
        "policies": {
            "product_to_toolchain_runtime_dependency": "deny",
            "toolchain_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    with pytest.raises(Exception):
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
        "target_component": "toolchain",
        "evidence_ids": ["edge:1"],
        "message": "Product runtime depends on Toolchain implementation.",
        "evaluator": {"id": "reference", "provenance": "OBSERVED"},
    }
    Draft202012Validator(schema).validate(payload)


def test_artifact_evidence_schema_separates_product_owner_from_toolchain_producer() -> None:
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
        "derivation": [{"relation": "PACKAGES", "source": "installer-builder"}],
        "provenance": "OBSERVED",
        "evidence_ids": ["artifact:installer:contents"],
    }
    Draft202012Validator(schema).validate(payload)
