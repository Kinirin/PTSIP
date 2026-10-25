"""GOV-only policy shape migration verification.

Existing non-GOV families retain the public Status:v1 route until individually migrated.
"""
from __future__ import annotations

import copy
import json
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator
from referencing import Registry, Resource


REPO = Path(__file__).resolve().parents[4]
GOV_FILES = {
    "MPD-GOV-0001": (6, 3, "2"),
    "MPD-GOV-0004": (6, 3, "3"),
    "MPD-GOV-0005": (8, 5, "3"),
    "MPD-GOV-0006": (12, 4, "1"),
}
SCHEMA_ROOT = ".ptsip/contracts/schemas"
GOV_SCHEMA_REF = f"{SCHEMA_ROOT}/Policy_Rules/root-family/GOV_rules.schema.json"
GOV_STATUS_REF = f"{SCHEMA_ROOT}/Policy_Status/root-family/GOV_status.schema.json"
GOV_PUBLIC_REF = f"{SCHEMA_ROOT}/root-family/GOV_public.schema.json"
SHARED_RULE_REF = f"{SCHEMA_ROOT}/Policy_Rules/Public_Structure_Rules.schema.json"


def _json(path: str) -> dict:
    return json.loads((REPO / path).read_text(encoding="utf-8"))


@pytest.fixture(scope="module")
def rule_validator() -> Draft202012Validator:
    rule_schema = _json(GOV_SCHEMA_REF)
    shared = _json(SHARED_RULE_REF)
    Draft202012Validator.check_schema(rule_schema)
    resources = Registry().with_resource(shared["$id"], Resource.from_contents(shared))
    return Draft202012Validator(rule_schema, registry=resources)


@pytest.mark.parametrize("policy_id", GOV_FILES)
def test_gov_policy_document_migration(policy_id: str, rule_validator: Draft202012Validator) -> None:
    document = yaml.safe_load((REPO / f"developer/policy/GOV/{policy_id}.yaml").read_text(encoding="utf-8"))
    obligation_count, prohibition_count, schema_version = GOV_FILES[policy_id]
    assert set(document) == {"Policy_Identity", "Policy_Status", "Authority", "Policy_Rules"}
    assert document["Policy_Identity"]["id"] == policy_id
    assert document["Policy_Identity"]["root_family"] == "GOV"
    status = document["Policy_Status"]
    assert status == {
        "Policy_Status_Schema_ver": schema_version,
        "status": "APPROVED.provisional",
        "contract": status["contract"],
    }
    assert status["contract"].startswith("urn:ptsip:contract:policy:gov:")
    assert not list(Draft202012Validator(_json(GOV_STATUS_REF)).iter_errors(status))
    rules = document["Policy_Rules"]
    identity = rules["Identity"]
    assert set(identity) == {"contract_id", "Rule_ver"}
    assert identity["contract_id"].startswith("urn:ptsip:policy:rule:gov:")
    assert ":v1" not in identity["contract_id"]
    assert identity["Rule_ver"] == "1"
    assert not list(rule_validator.iter_errors(rules))
    constraints = rules["Constraints"]
    assert "prohibitions" not in constraints
    obligations = constraints["obligations"]
    positive = [row for row in obligations if row["id"].startswith("OBLIGATION/")]
    negative = [row for row in obligations if row["id"].startswith("PROHIBITION/")]
    assert (len(positive), len(negative)) == (obligation_count, prohibition_count)
    assert len({row["id"] for row in obligations}) == len(obligations)
    assert all(set(row) == {"id", "applies_when", "rule", "Rule_state"} for row in positive)
    assert all(set(row) == {"id", "rule"} for row in negative)
    assert all(row["Rule_state"] == "APPROVED.provisional" for row in positive)
    assert all(row["applies_when"] for row in positive)
    assert all(row["rule"] for row in obligations)


def test_gov_route_only_uses_family_specific_public_schema() -> None:
    catalog = _json(".ptsip/contracts/policy/routing/policy_routes.json")
    paths = catalog["routes"]["PTSIP_DEVELOPER_POLICY"]
    assert len(paths) == 14
    assert paths["GOV"]["schema_refs"]["public"] == GOV_PUBLIC_REF
    assert paths["GOV"]["schema_refs"]["rules"] == GOV_SCHEMA_REF
    for family, route in paths.items():
        if family != "GOV":
            assert route["schema_refs"]["public"] == (
                ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json"
            )
    gov_public = _json(GOV_PUBLIC_REF)
    assert gov_public["properties"]["Policy_Status"]["$ref"] == _json(GOV_STATUS_REF)["$id"]
    assert gov_public["properties"]["Policy_Rules"]["$ref"] == _json(GOV_SCHEMA_REF)["$id"]
    shared = _json(".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json")
    assert shared["$id"] == "urn:ptsip:policy:Status:v1"
    assert shared["required"] == ["Schema", "constant"]


def test_all_flat_status_names_follow_approved_lifecycle_vocabulary() -> None:
    schema = _json(GOV_STATUS_REF)
    allowed = schema["properties"]["status"]["enum"]
    derived = [
        major + "." + name
        for major in ("APPROVED", "DRAFT", "RETIRED")
        for name in _json(
            f".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_{major}.json"
        )["properties"]["substatus"]["enum"]
    ]
    assert sorted(allowed) == sorted(derived)
    assert len(allowed) == len(set(allowed))
    assert not list(Draft202012Validator(schema).iter_errors({
        "Policy_Status_Schema_ver": "3", "status": "APPROVED.provisional",
        "contract": "urn:ptsip:contract:policy:gov:identity:integrity",
    }))
    for changed in (
        {"status": "APPROVED.nonexistent"},
        {"constant": {"status": "APPROVED", "substatus": "provisional"}},
        {"Policy_Status_Schema_ver": ""},
    ):
        instance = {"Policy_Status_Schema_ver": "3", "status": "APPROVED.provisional",
                    "contract": "urn:ptsip:contract:policy:gov:identity:integrity"}
        instance.update(changed)
        assert list(Draft202012Validator(schema).iter_errors(instance))


def test_obligation_and_prohibition_negative_cases(rule_validator: Draft202012Validator) -> None:
    document = yaml.safe_load(
        (REPO / "developer/policy/GOV/MPD-GOV-0006.yaml").read_text(encoding="utf-8")
    )
    original = document["Policy_Rules"]

    missing_state = copy.deepcopy(original)
    del missing_state["Constraints"]["obligations"][0]["Rule_state"]
    assert list(rule_validator.iter_errors(missing_state))

    invalid_prohibition = copy.deepcopy(original)
    prohibited = next(
        x for x in invalid_prohibition["Constraints"]["obligations"]
        if x["id"].startswith("PROHIBITION/")
    )
    prohibited["operation"] = prohibited["rule"]
    assert list(rule_validator.iter_errors(invalid_prohibition))

    legacy_id = copy.deepcopy(original)
    legacy_id["Identity"]["contract_id"] += ":v1"
    assert list(rule_validator.iter_errors(legacy_id))

    legacy_prohibitions = copy.deepcopy(original)
    legacy_prohibitions["Constraints"]["prohibitions"] = []
    assert list(rule_validator.iter_errors(legacy_prohibitions))


def test_full_gov_document_schema_validates_four_policies() -> None:
    resource_paths = (
        ".ptsip/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json",
        ".ptsip/contracts/policy/identity/policy_kind.json",
        ".ptsip/contracts/policy/identity/root_family.json",
        ".ptsip/contracts/schemas/Authority/PTSIP_Policy_Authority.schema.json",
        ".ptsip/contracts/schemas/Authority/Authority_Category/Authority_Major_Category.schema.json",
        ".ptsip/contracts/policy/authority/Authority_Major_Category.json",
        GOV_STATUS_REF,
        GOV_SCHEMA_REF,
        SHARED_RULE_REF,
    )
    resources = [_json(path) for path in resource_paths]
    registry = Registry().with_resources(
        (resource["$id"], Resource.from_contents(resource)) for resource in resources
    )
    schema = _json(GOV_PUBLIC_REF)
    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema, registry=registry)
    for policy_id in GOV_FILES:
        policy = yaml.safe_load(
            (REPO / f"developer/policy/GOV/{policy_id}.yaml").read_text(encoding="utf-8")
        )
        errors = sorted(
            validator.iter_errors(policy),
            key=lambda error: tuple(map(str, error.absolute_path)),
        )
        assert not errors, f"{policy_id}: {errors}"
