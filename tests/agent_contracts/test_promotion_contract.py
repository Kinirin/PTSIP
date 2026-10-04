from __future__ import annotations

import copy
import json
from importlib.resources import files

import pytest
from jsonschema import Draft202012Validator, ValidationError


ROOT = files("agent_contracts")


def _json(ref: str) -> dict:
    return json.loads(ROOT.joinpath(*ref.split("/")).read_text(encoding="utf-8"))


def test_product_promotion_resolves_one_exact_specification_candidate() -> None:
    index = _json("promotion/index.json")
    matches = [item for item in index["contracts"] if item["id"] == index["current_candidate"]]
    assert len(matches) == 1
    entry = matches[0]
    contract = _json(entry["ref"])
    schema = _json(entry["schema_ref"])
    Draft202012Validator.check_schema(schema)
    Draft202012Validator(schema).validate(contract)
    assert contract["contract_id"] == entry["id"]
    assert entry["status"] == contract["status"] == "CANDIDATE"
    assert index["normative_authority"] is False


def test_promotion_projection_preserves_registered_candidate_payloads() -> None:
    contract = _json("promotion/specification-contract.json")
    projection = _json("promotion/current.json")
    source = projection["source"]
    assert source == {
        "authority_class": "SPECIFICATION_MACHINE_CONTRACT",
        "registry_ref": "promotion/index.json",
        "contract_id": contract["contract_id"],
        "contract_ref": "promotion/specification-contract.json",
    }
    for field in ("binding_discovery_and_promotion", "required_verification"):
        assert projection[field] == contract[field]


def test_promotion_candidate_does_not_claim_specification_activation() -> None:
    contract = _json("promotion/specification-contract.json")
    projection = _json("promotion/current.json")
    assert contract["specification_binding"] == {"status": "UNBOUND", "revision": None}
    for asset in (contract, projection):
        assert asset["status"] == "CANDIDATE"
        assert asset["normative_authority"] is False
        assert asset["activation"] == "REQUIRES_VERIFIED_SPECIFICATION_CUTOVER"


@pytest.mark.parametrize("change", ["developer_source", "activation", "payload"])
def test_promotion_contract_rejects_unapproved_authority_or_semantic_drift(change: str) -> None:
    contract = copy.deepcopy(_json("promotion/specification-contract.json"))
    if change == "developer_source":
        contract["source"] = {"policy_id": "MPD-0012"}
    elif change == "activation":
        contract["normative_authority"] = True
    else:
        contract["required_verification"]["aggregate"]["unknown_without_fail"] = "AUTO_PROMOTE"
    with pytest.raises(ValidationError):
        Draft202012Validator(_json("schemas/contracts/promotion.schema.json")).validate(contract)
