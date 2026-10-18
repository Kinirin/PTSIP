"""Control-plane implementation contracts must not materialize an unapproved MPD policy."""
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
CONTRACT = ROOT / "developer/contracts/test-mode-control-plane/independence.v1.yaml"
PROFILE = ROOT / ".ptsip/profiles/main.ptsip.yaml"
REGISTRY = ROOT / ".github/test_modes.yaml"
EVIDENCE = {
    "no_legacy_code_import_or_invocation",
    "no_unapproved_semantic_inheritance",
    "normative_source_status_and_authority_proof",
    "registry_target_and_profile_responsibility_coverage",
    "no_indirect_workflow_or_wrapper_calls",
    "new_validator_passes_after_legacy_removal",
}


def read_yaml(path):
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def test_unapproved_arch_policy_is_not_materialized_or_registered():
    policy_path = ROOT / "developer/policy/ARCH/MPD-ARCH-0004.yaml"
    assert not policy_path.exists()
    catalog = read_yaml(ROOT / "developer/policy/index.yaml")
    assert not any(
        p["id"] == "MPD-ARCH-0004" or p["path"] == "developer/policy/ARCH/MPD-ARCH-0004.yaml"
        for p in catalog["policies"]
    )
    contract = read_yaml(CONTRACT)
    assert contract["artifact_role"] == "NON_POLICY_IMPLEMENTATION_CONTRACT"
    assert contract["policy_authority_claim"] == "NONE"
    assert contract["root_family_policy_materialization"] == "PENDING_SCHEMA_AND_ID_DECISION"
    assert "policy_ref" not in contract


def test_six_approved_independence_evidence_obligations_and_open_selection():
    contract = read_yaml(CONTRACT)
    assert set(contract["verification"]["required_evidence"]) == EVIDENCE
    assert contract["selection"]["status"] == "UNRESOLVED"
    assert contract["selection"]["unknown_changed_file_behavior"] == "FAIL_CLOSED"
    assert contract["selection"]["legacy_algorithm_import"] == "FORBIDDEN"
    assert contract["authority"]["pinned_package"] == "PTSIP==0.3.8b5"
    assert contract["authority"]["structural_operation"] == "validation.schema"


def test_contract_refuses_to_claim_unexecuted_independence():
    contract = read_yaml(CONTRACT)
    assert contract["component_ref"] == "repository-test-mode-control-plane"
    assert contract["verification"]["conformance_claim"] == "NOT_ASSERTED"
    assert contract["verification"]["missing_evidence"] == "FAIL_CLOSED"
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
    assert "developer/policy/ARCH/MPD-ARCH-0004.yaml" not in components[0]["analysis_inputs"]
