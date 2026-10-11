from __future__ import annotations

import json
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[4]
POLICY = "src/policy/CNTR/SFP-CNTR-0007.yaml"


def _catalog(relative: str) -> dict[str, dict]:
    data = yaml.safe_load((ROOT / relative).read_text(encoding="utf-8"))
    return {item["id"]: item for item in data["policies"]}


def _references(value: object) -> set[str]:
    if isinstance(value, str):
        return {value}
    if isinstance(value, dict):
        return set().union(*(_references(item) for item in value.values()))
    if isinstance(value, list):
        return set().union(*(_references(item) for item in value))
    return set()


def _invalid_refs(value: object, forbidden: set[str]) -> set[str]:
    return _references(value) & forbidden


def test_extension_contract_uses_only_current_support_authority() -> None:
    current = _catalog("src/policy/index.yaml")
    developer = _catalog("developer/policy/index.yaml")
    historic = {
        identity for index in (current, developer)
        for identity, entry in index.items()
        if entry["authority_role"] == "MIGRATION_SOURCE"
    }
    policy = json.loads((ROOT / POLICY).read_text(encoding="utf-8"))
    assert policy["policy"]["status"] == "ACTIVE"
    assert current["SFP-CNTR-0007"]["status"] == "ACTIVE"
    unit = policy["authority_semantics"]["unit_registered_cli_extension_activation"]
    reconciliation = unit["reconciliation"]

    # Migration provenance must not become a live policy dependency.
    assert not _invalid_refs(unit, historic)
    assert not _invalid_refs(policy["relations"], historic)
    assert not _invalid_refs(unit, set(developer))
    assert reconciliation["authority_source"] == "CURRENT_SUPPORT_ROOT_FAMILY"
    assert reconciliation["legacy_migration_sources_are_runtime_authority"] is False
    assert reconciliation["developer_policy_authority_inheritance"] == "FORBIDDEN"

    refs = reconciliation["current_authority_policy_refs"]
    assert refs == ["SFP-CNTR-0001", "SFP-GOV-0001", "SFP-CTRL-0001"]
    for identity in refs:
        assert current[identity]["authority_role"] == "CANONICAL_AUTHORITY"
        assert current[identity]["status"] == "ACTIVE"
    assert {entry["policy"] for entry in policy["relations"]["depends_on"]} == set(refs)
    assert reconciliation["external_resolution_environment_in_ptsip_support_scope"] is False
    assert unit["initial_execution"]["mechanism"] == "REGISTERED_COMMAND_INVOCATION"
    assert unit["lifecycle"] == "ACTIVE_NORMATIVE_CONTRACT_ONLY_RUNTIME_IMPLEMENTATION_NOT_AUTHORIZED"
    assert "RUNTIME_IMPLEMENTATION_AUTHORIZATION" in unit["explicit_exclusions"]


def test_live_reference_gate_rejects_historical_policy_identity() -> None:
    developer = _catalog("developer/policy/index.yaml")
    historic = {
        identity for identity, entry in developer.items()
        if entry["authority_role"] == "MIGRATION_SOURCE"
    }
    # Guard regression: an archived policy may exist but must never be a live owner.
    assert "MPD-0012" in historic
    assert _invalid_refs({"authority_ref": "MPD-0012"}, historic) == {"MPD-0012"}
    assert _invalid_refs({"authority_ref": "SFP-GOV-0001"}, historic) == set()
