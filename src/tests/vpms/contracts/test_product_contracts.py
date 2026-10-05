from __future__ import annotations

import json
import unicodedata
from pathlib import Path

from jsonschema import Draft202012Validator
from referencing import Registry, Resource
import pytest

ROOT = Path(__file__).resolve().parents[4]
BASE = ROOT / "src/vpms/contracts"


def load(relative):
    return json.loads((BASE / relative).read_text(encoding="utf-8"))


def resources():
    identity = load("identity-registry.json")
    registry = Registry().with_resource(identity["$id"], Resource.from_contents(identity))
    for relative in identity["schema_resources"]:
        schema = load(relative)
        Draft202012Validator.check_schema(schema)
        registry = registry.with_resource(schema["$id"], Resource.from_contents(schema))
    return registry


def test_catalog_and_contracts_are_exact_independent_and_active():
    catalog = load("index.json")
    schema = load("schemas/catalog.schema.json")
    registry = resources()
    Draft202012Validator(schema, registry=registry).validate(catalog)
    assert catalog["capability"] == "ACTIVE"
    assert len(set(catalog["entrypoints"].values())) == 3
    assert set(catalog["entrypoints"].values()) == set(catalog["contracts"])
    payload_schema = load("schemas/product-contract.schema.json")
    payloads = {}
    for identity, route in catalog["contracts"].items():
        payload = load(route["path"])
        Draft202012Validator(payload_schema, registry=registry).validate(payload)
        assert payload["id"] == identity
        assert payload["status"] == route["status"] == "ACTIVE"
        assert payload["responsibility"] == route["responsibility"]
        assert payload["runtime_enabled"] is route["runtime_enabled"] is True
        assert payload["owner_approval"] == "USER_EXPLICIT"
        assert not set(payload["owns"]) & set(payload["excludes"])
        payloads[payload["responsibility"]] = payload
    assert set(payloads) == {
        "VPMS_PRODUCT_PROTOCOL", "VPMS_CASE_SELECTION", "VPMS_EXECUTION_COMPOSITION"
    }


def test_product_contracts_do_not_depend_on_developer_policy():
    for relative in ("protocol.json", "selection.json", "execution-composition.json"):
        assert load(relative)["semantics"]["developer_policy_runtime_dependency"] is False
    for path in BASE.rglob("*.json"):
        assert "MPD-" not in path.read_text(encoding="utf-8")
        assert "developer/" not in path.read_text(encoding="utf-8")


def test_protocol_identity_purpose_and_reference_boundaries():
    semantics = load("protocol.json")["semantics"]
    assert semantics["identity"] == {
        "canonical_name": "VPMS",
        "expanded_identity": "VERIFICATION_PROTOCOL_MANAGEMENT_SYSTEM",
        "governing_question": "HOW_EXPLICIT_VERIFICATION_CASES_ARE_BOUND_EXECUTED_AND_REPORTED",
    }
    assert semantics["case"]["canonical_fields"] == [
        "id", "target", "formula", "variables", "policy", "runner"
    ]
    purpose = semantics["case"]["compatibility_fields"]["purpose"]
    assert purpose["role"] == "COMPATIBILITY_ONLY"
    assert purpose["canonical_selection_authority"] is False
    assert purpose["removal_requires_separate_owner_approval"] is True
    assert semantics["registry_handoff"]["direct_construction_is_validation_proof"] is False
    assert semantics["result"]["ptsip_conformance_authority"] is False


def test_selection_and_execution_composition_do_not_share_authority():
    catalog = load("index.json")
    selected = load("selection.json")
    execution = load("execution-composition.json")
    assert selected["semantics"]["execution_side_effects"] is False
    assert selected["semantics"]["case_policy_is_selection_rule"] is False
    assert "EXECUTION_COMPOSITION" in selected["excludes"]
    assert "SELECTION_REQUEST_INTERPRETATION" in execution["excludes"]
    assert execution["semantics"]["missing_adapter"] == "REJECT_BEFORE_FIRST_CASE_EXECUTION"
    assert execution["semantics"]["single_case_execution_contract"] == catalog["entrypoints"]["protocol"]
    assert execution["semantics"]["selection_contract"] == catalog["entrypoints"]["selection"]
    assert selected["planned_bindings"][0]["availability"] == "IMPLEMENTED_ACTIVE"
    assert execution["planned_bindings"][0]["availability"] == "IMPLEMENTED_ACTIVE"


@pytest.mark.parametrize("relative", ["protocol.json", "selection.json", "execution-composition.json"])
def test_unknown_semantic_field_is_not_silently_accepted(relative):
    payload = load(relative)
    payload["semantics"]["unregistered_extension"] = True
    validator = Draft202012Validator(load("schemas/product-contract.schema.json"), registry=resources())
    assert list(validator.iter_errors(payload))


def test_neutral_catalog_does_not_define_runtime_case_shape():
    identity = load("identity-registry.json")
    assert all(value.get("type") != "object" for value in identity["$defs"].values())
    assert load("invariants.json")["schema_is_shape_authority"] is True
    assert load("invariants.json")["non_active_registry_may_execute"] is False


@pytest.mark.parametrize("relative", ["protocol.json", "selection.json", "execution-composition.json", "index.json"])
def test_active_artifacts_have_canonical_json_bytes_without_activating_a_digest(relative):
    def encode(value):
        if value is None or isinstance(value, bool): return json.dumps(value)
        if isinstance(value, str): return json.dumps(unicodedata.normalize("NFC", value), ensure_ascii=False)
        if isinstance(value, (int, float)):
            # These exact artifacts contain only revision 2 and zero counts.
            assert value in (0, 2)
            return "0" if value == 0 else "2e0"
        if isinstance(value, list): return "[" + ",".join(encode(item) for item in value) + "]"
        return "{" + ",".join(encode(key) + ":" + encode(value[key]) for key in sorted(value)) + "}"
    assert (BASE / relative).read_bytes() == encode(load(relative)).encode("utf-8")
    assert load("invariants.json")["canonical_semantic_digest_authority"] == "NOT_ACTIVATED"
