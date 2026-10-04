from __future__ import annotations

from pathlib import Path

import pytest

from agent_contracts.candidate import read_json
from developer.automation.policy_identity_lifecycle import inspect_policy, status_preflight
from developer.automation.policy_loader import load_yaml
from developer.automation.policy_plan_binding.reconciler import reconcile_registry
from developer.automation.policy_plan_binding.resolver import resolve_bindings
from developer.automation.policy_resolver import get_policy, resolve_policies
from developer.automation.verification.policy_plan_consistency import verify_policy_plan_consistency


ROOT = Path(__file__).resolve().parents[2]
PLAN = "developer/planning/0.3.8/0.3.8a3/index.yaml"
WU08 = "developer/planning/0.3.8/0.3.8a3/WU-08/WU-08.yaml"
POLICIES = ("MPD-0015", "MPD-SPEC-0022", "MPD-VERI-0004", "MPD-WORK-0004")


def test_mutation_gate_activation_has_explicit_approval_and_consistent_identity() -> None:
    inspected = inspect_policy("MPD-WORK-0004", root=ROOT)
    assert inspected["policy_status"] == inspected["index_status"] == "ACTIVE"
    assert inspected["policy_version"] == "2.0"
    assert inspected["operationally_resolvable"] is True
    for target in ("APPROVED", "ACTIVE"):
        ref = f"developer/policy/approvals/MPA-20261004-WU08-MUTATION-GATE-{target}.yaml"
        approval = load_yaml(ref, root=ROOT)["approval"]
        assert approval["target_status"] == target
        assert approval["decision_source"] == "USER_EXPLICIT"
        assert status_preflight("MPD-WORK-0004", ref, root=ROOT)["status"] == "READY"


@pytest.mark.parametrize("policy_id", POLICIES)
def test_formal_relation_resolves_exactly_to_the_materialized_plan(policy_id: str) -> None:
    identity = load_yaml(PLAN, root=ROOT)["plan_identity"]
    result = resolve_bindings(policy_ref=policy_id, plan_ref=PLAN, root=ROOT)
    assert result.status == "BOUND"
    assert len(result.bindings) == 1
    binding = result.bindings[0]
    assert binding["planning_state"] == "CREATED"
    for field in ("resolved_plan_id", "plan_file_id", "version", "revision"):
        assert binding[field] == identity[field]


def test_formal_registry_and_policy_plan_consistency_are_valid_without_repair() -> None:
    registry = reconcile_registry(root=ROOT)
    assert registry.status == "CURRENT"
    assert registry.changed is registry.applied is False
    assert registry.failures == ()
    report = verify_policy_plan_consistency(root=ROOT)
    assert report.status == "PASS"
    assert report.checked_binding_count == report.binding_count


@pytest.mark.parametrize("operation", ["READ", "PLAN", "MODIFY", "VERIFY"])
def test_wu08_owner_policies_enter_through_exact_scope_binding(operation: str) -> None:
    result = resolve_policies(ROOT, scope=WU08, operation=operation)
    assert result["binding_scope"] == WU08
    selected = {item["policy_id"]: item for item in result["policies"]}
    for policy_id in POLICIES[1:]:
        assert selected[policy_id]["status"] == "ACTIVE"
    assert "binding_identity" in selected["MPD-SPEC-0023"]["sections"]


def test_digest_candidates_reflect_approved_policy_without_activating_unknown_bindings() -> None:
    canonical = get_policy(ROOT, policy_id="MPD-SPEC-0022", section="contract_artifact_representation")["record"]
    contract_root = ROOT / "src/agent_contracts"
    projection = read_json(contract_root, "digests/projection-v1.json")
    digest = read_json(contract_root, "digests/policy-v1.json")
    expected = canonical["normative_semantic_projection"]
    for candidate in (projection, digest["projection_policy"]):
        assert candidate["field_governance"] == expected["field_governance"]
        assert candidate["collection_semantics"] == expected["collection_semantics"]
        assert candidate["binding_scope"] == expected["binding_scope"]
        for kind in ("exact_inclusion_list", "exact_exclusion_list"):
            assert candidate["field_governance"][kind]["exact_role_bindings"]
    assert digest["activation_requirements"] == canonical["canonical_digest_scheme_selection"]["activation_requirements"]
    assert digest["active_scheme"] == "UNRESOLVED"
    assert projection["normative_authority"] is digest["normative_authority"] is False
    assert read_json(contract_root, "index.json")["canonical_digest"] == "UNRESOLVED"


def test_verification_entry_consumes_profile_authority_without_inventing_ownership() -> None:
    required = get_policy(ROOT, policy_id="MPD-VERI-0004", section="required_verification")["record"]
    entry = required["repository_entry"]
    assert entry["source_ownership_required_before_mode_selection"] is True
    assert entry["adding_analysis_inputs_alone_creates_source_ownership"] is False
    assert entry["agent_may_infer_source_classification_or_verification_relationship"] is False
    assert entry["unmapped_path_may_use_full_verification_fallback"] is False
    result = resolve_policies(ROOT, scope="src/agent_contracts/candidate.py", operation="VERIFY")
    selected = {item["policy_id"]: item for item in result["policies"]}
    assert selected["MPD-VERI-0004"]["sections"] == ["required_verification"]
