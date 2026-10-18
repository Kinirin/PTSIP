"""Fresh control-plane contract assertions; does not implement a Test Mode selector."""
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[3]
POLICY = ROOT / "developer/policy/ARCH/MPD-ARCH-0004.yaml"
CONTRACT = ROOT / "developer/contracts/test-mode-control-plane/independence.v1.yaml"
PROFILE = ROOT / ".ptsip/profiles/main.ptsip.yaml"
REGISTRY = ROOT / ".github/test_modes.yaml"
GATES = {
    "NO_LEGACY_VALIDATOR_IMPORT_OR_INVOCATION",
    "NO_UNAPPROVED_HISTORICAL_RULE_INHERITANCE",
    "ONLY_EFFECTIVE_POLICY_PROFILE_REGISTRY_AS_AUTHORITY",
    "ALL_REGISTERED_MODE_TARGETS_EXIST_AND_HAVE_VERIFICATION_OWNERS",
    "NO_INDIRECT_LEGACY_EXECUTION_FROM_WORKFLOWS_OR_WRAPPERS",
    "NEW_VALIDATOR_PASSES_WITH_LEGACY_VALIDATORS_REMOVED",
}


def read_yaml(path):
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def test_new_canonical_policy_is_registered_active():
    catalog = read_yaml(ROOT / "developer/policy/index.yaml")
    matches = [item for item in catalog["policies"] if item["id"] == "MPD-ARCH-0004"]
    assert len(matches) == 1
    assert matches[0]["status"] == "ACTIVE"
    assert matches[0]["authority_role"] == "CANONICAL_AUTHORITY"
    policy = read_yaml(POLICY)
    assert policy["policy"]["status"] == "ACTIVE"
    assert policy["rules"]["control_plane"]["definition"] == "FRESH_AUTHORITY_CONTRACT"


def test_six_owner_approved_conditions_and_fail_closed_open_decision():
    policy = read_yaml(POLICY)["rules"]["control_plane"]
    assert set(policy["independence_acceptance"]) == GATES
    assert policy["selection"]["changed_file_trigger_semantics"] == "UNRESOLVED"
    assert policy["selection"]["unresolved_selection"] == "FAIL_CLOSED"
    assert policy["selection"]["automatic_algorithm_legacy_fallback"] == "FORBIDDEN"
    assert policy["structure"]["pinned_version"] == "0.3.8b5"
    assert policy["structure"]["operation"] == "validation.schema"
    assert all(policy["prohibitions"].values())


def test_contract_refuses_to_claim_unexecuted_independence():
    contract = read_yaml(CONTRACT)
    assert contract["component_ref"] == "repository-test-mode-control-plane"
    assert contract["verification"]["conformance_claim"] == "NOT_ASSERTED"
    assert contract["verification"]["missing_evidence"] == "FAIL_CLOSED"
    assert contract["selection"]["status"] == "UNRESOLVED"
    assert contract["retirement"]["remove_legacy_validator_after_new_validator_verified"] is True


def test_component_mapping_is_new_but_identity_is_stable():
    registry = read_yaml(REGISTRY)
    modes = [m for m in registry["modes"] if m["id"] == "test-mode-control-plane"]
    assert len(modes) == 1
    assert modes[0]["component_ref"] == "repository-test-mode-control-plane"
    assert modes[0]["execution"] == {"pytest": ["developer/tests/testmode"]}
    profile = read_yaml(PROFILE)
    components = [c for c in profile["components"] if c["id"] == "repository-test-mode-control-plane"]
    assert len(components) == 1
    assert "VERIFICATION" in components[0]["roles"]
    assert "developer/tests/testmode/**" in components[0]["include"]
    assert "developer/contracts/test-mode-control-plane/**" in components[0]["include"]
    assert "src/tests/ptsip/test_modes/**" not in components[0]["include"]
