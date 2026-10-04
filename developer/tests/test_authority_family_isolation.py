from __future__ import annotations

import copy
import ast
import json
from pathlib import Path

import pytest

from developer.automation.authority_family_migration import legacy_analysis_projection, semantic_digest
from developer.automation.policy_identity_lifecycle import (
    PolicyIdentityLifecycleError, _next_policy_id, _next_family_policy_id,
    _require_family_class_materialization, _validated_analysis_group, inspect_policy,
    preflight_family_policy, register_family_policy,
)
from developer.automation.policy_loader import load_json, load_yaml
from developer.automation.policy_responsibility_gate import (
    ResponsibilityGateError, _active_family_ids, validate_analysis_semantics,
    validate_responsibility_analysis,
)
from developer.tests.test_policy_responsibility_gate import repo, _write_yaml, _responsibility

ROOT = Path(__file__).resolve().parents[2]
PTSIP = "PTSIP_DEVELOPER_POLICY"
VPMS = "VPMS_DEVELOPER_POLICY"


@pytest.fixture
def domains(repo: Path) -> Path:
    index = load_yaml("developer/policy/index.yaml", root=repo)
    for policy_id, policy_class in (("MPD-VERI-0001", PTSIP), ("MPD-VERI-0002", VPMS)):
        path = f"developer/policy/VERI/{policy_id}.yaml"
        _write_yaml(repo / path, {"schema_version": "developer-policy/v2", "policy_class": policy_class,
            "policy": {"id": policy_id, "status": "ACTIVE", "version": "2.0", "title": "fixture"}, "rules": {"fixture": {"enabled": True}}})
        index["policies"].append({"id": policy_id, "policy_class": policy_class, "path": path, "status": "ACTIVE"})
    index["policies"].sort(key=lambda item: item["id"])
    _write_yaml(repo / "developer/policy/index.yaml", index)
    subject = load_yaml("developer/policy/registries/authority-subject-registry.yaml", root=repo)
    subject["subject_identity_schemes"]["MANAGEMENT_POLICY_ID"]["registered_values"] = [entry["id"] for entry in index["policies"]]
    _write_yaml(repo / "developer/policy/registries/authority-subject-registry.yaml", subject)
    return repo


def analysis(policy_class: str = PTSIP, searched: list[str] | None = None) -> dict[str, object]:
    responsibility = _responsibility("R01", "VERI", "G01", searched_policy_ids=searched)
    responsibility["policy_class"] = policy_class
    responsibility["existing_authority_lookup"]["searched_policy_class"] = policy_class
    return {"schema_version": "developer-policy-responsibility-analysis/v2", "artifact_class": "PTSIP_POLICY_RESPONSIBILITY_ANALYSIS", "analysis": {
        "analysis_id": "PRA-isolation", "source_ref": "test", "responsibilities": [responsibility], "decision": {
            "owned_authority_family_set": [{"policy_class": policy_class, "family": "VERI"}], "split_required": False, "materialization_allowed": True,
            "materialization_groups": [{"group_id": "G01", "policy_class": policy_class, "family": "VERI", "cohesion_key": "VERI_COHESION", "responsibility_ids": ["R01"], "cohesion_rationale": "fixture"}]
        }}}


@pytest.mark.parametrize("policy_class,expected", [(PTSIP, ("MPD-VERI-0001",)), (VPMS, ("MPD-VERI-0002",))])
def test_same_family_lookup_is_isolated_by_policy_class(domains: Path, policy_class, expected) -> None:
    assert _active_family_ids(domains, policy_class, "VERI") == expected


@pytest.mark.parametrize("policy_class,family", [(None, "VERI"), ("UNKNOWN", "VERI"), (PTSIP, "BOUND")])
def test_unknown_or_incomplete_composite_key_fails_closed(domains: Path, policy_class, family) -> None:
    with pytest.raises(ResponsibilityGateError, match="explicit registered"):
        _active_family_ids(domains, policy_class, family)


def test_family_only_lookup_is_not_a_callable_interface(domains: Path) -> None:
    with pytest.raises(TypeError):
        _active_family_ids(domains, "VERI")


@pytest.mark.parametrize("field,value", [("policy_class", VPMS), ("status", "DRAFT")])
def test_projection_metadata_mismatch_fails_closed(domains: Path, field, value) -> None:
    index = load_yaml("developer/policy/index.yaml", root=domains)
    next(entry for entry in index["policies"] if entry["id"] == "MPD-VERI-0001")[field] = value
    _write_yaml(domains / "developer/policy/index.yaml", index)
    with pytest.raises(ResponsibilityGateError, match="projection mismatch"):
        _active_family_ids(domains, PTSIP, "VERI")


def test_cross_class_comparison_cannot_enter_collision_domain(domains: Path) -> None:
    path = domains / "developer/policy/analysis/PRA-isolation.yaml"
    _write_yaml(path, analysis(VPMS, ["MPD-VERI-0001"]))
    with pytest.raises(ResponsibilityGateError, match="must exactly cover"):
        validate_responsibility_analysis(path, root=domains)


@pytest.mark.parametrize("policy_class,policy_id", [(PTSIP, "MPD-VERI-0001"), (VPMS, "MPD-VERI-0002")])
def test_same_class_same_family_collision_is_analyzed(domains: Path, policy_class, policy_id) -> None:
    payload = analysis(policy_class, [policy_id])
    responsibility = payload["analysis"]["responsibilities"][0]
    responsibility["existing_authority_lookup"].update({"lookup_outcome": "MATCHES_FOUND", "candidate_comparisons": [{"policy_id": policy_id, "section": "fixture", "scope_relation": "SAME_SCOPE", "collision_class": "EXACT_DUPLICATE", "resolution_action": "REFERENCE_EXISTING"}]})
    responsibility.update({"materialization_action": "USE_EXISTING_AUTHORITY", "target_group_id": None})
    payload["analysis"]["decision"]["materialization_groups"] = []
    path = domains / "developer/policy/analysis/PRA-isolation.yaml"
    _write_yaml(path, payload)
    assert validate_responsibility_analysis(path, root=domains)["status"] == "PASS"


def test_missing_owner_class_is_not_inferred_from_family() -> None:
    payload = analysis()
    payload["analysis"]["responsibilities"][0].pop("policy_class")
    assert any("explicit policy_class" in error for error in validate_analysis_semantics(payload))


def test_lookup_class_must_match_candidate_class() -> None:
    payload = analysis()
    payload["analysis"]["responsibilities"][0]["existing_authority_lookup"]["searched_policy_class"] = VPMS
    assert any("searched_policy_class" in error for error in validate_analysis_semantics(payload))


def test_materialization_group_cannot_mix_policy_classes() -> None:
    payload = analysis()
    other = copy.deepcopy(payload["analysis"]["responsibilities"][0])
    other["responsibility_id"] = "R02"
    other["policy_class"] = VPMS
    other["existing_authority_lookup"]["searched_policy_class"] = VPMS
    payload["analysis"]["responsibilities"].append(other)
    payload["analysis"]["decision"]["owned_authority_family_set"].append({"policy_class": VPMS, "family": "VERI"})
    payload["analysis"]["decision"]["split_required"] = True
    payload["analysis"]["decision"]["materialization_groups"][0]["responsibility_ids"].append("R02")
    assert any("policy_class mismatch" in error for error in validate_analysis_semantics(payload))


def test_requested_materialization_class_must_match_group(domains: Path) -> None:
    path = domains / "developer/policy/analysis/PRA-isolation.yaml"
    _write_yaml(path, analysis(PTSIP, ["MPD-VERI-0001"]))
    with pytest.raises(PolicyIdentityLifecycleError) as exc:
        _validated_analysis_group(policy_class=VPMS, family="VERI", analysis_ref=path, group_id="G01", base=domains)
    assert exc.value.code == "MATERIALIZATION_GROUP_CLASS_MISMATCH"


def test_generic_allocator_ignores_boundary_and_family_ids() -> None:
    assert _next_policy_id(("MPD-0001", "MPD-BOUND-9999", "MPD-VERI-9999")) == "MPD-0002"


def test_family_allocator_keeps_one_shared_cross_class_sequence() -> None:
    assert _next_family_policy_id(("MPD-VERI-0001", "MPD-VERI-0002", "MPD-BOUND-9999"), "VERI") == "MPD-VERI-0003"


def test_vpms_opening_requires_completed_isolation_verification(domains: Path) -> None:
    path = domains / "developer/policy/registries/developer-policy-catalog-contracts.json"
    record = json.loads(path.read_text(encoding="utf-8"))
    record["application_execution"]["m1_m7_verified"] = False
    path.write_text(json.dumps(record), encoding="utf-8")
    with pytest.raises(PolicyIdentityLifecycleError) as exc:
        _require_family_class_materialization(VPMS, domains)
    assert exc.value.code == "VPMS_CLASS_MATERIALIZATION_NOT_ENABLED"


def test_all_eight_legacy_analyses_preserve_collision_and_materialization_meaning() -> None:
    contract = load_json("developer/policy/registries/developer-policy-catalog-contracts.json", root=ROOT)
    digests = contract["application_execution"]["preserved_analysis_semantic_digests"]
    assert len(digests) == 8
    for ref, expected in digests.items():
        assert semantic_digest(legacy_analysis_projection(load_yaml(ref, root=ROOT))) == expected


def _first_vpms_policy() -> dict[str, object]:
    contract = load_json("developer/policy/registries/developer-policy-catalog-contracts.json", root=ROOT)
    policy_id = contract["application_execution"]["registered_vpms_policy_id"]
    return load_yaml(f"developer/policy/VERI/{policy_id}.yaml", root=ROOT)


def test_first_vpms_policy_is_registered_approved_but_not_active() -> None:
    payload = _first_vpms_policy()
    inspected = inspect_policy(payload["policy"]["id"], root=ROOT)
    assert payload["schema_version"] == "developer-policy/v2"
    assert payload["policy_class"] == VPMS
    assert inspected["policy_status"] == inspected["index_status"] == "APPROVED"
    assert inspected["policy_version"] == "1.0"
    assert inspected["subject_identity_registered"]
    assert not inspected["operationally_resolvable"]
    assert payload["transition"]["state"] == "PENDING"
    assert payload["transition"]["requirements"][0]["next_action"]["execution"] == "OWNER_DECISION_REQUIRED"
    assert _active_family_ids(ROOT, VPMS, "VERI") == ()
    assert _active_family_ids(ROOT, PTSIP, "VERI") == tuple(f"MPD-VERI-{number:04}" for number in range(2, 6))


def test_first_vpms_policy_owns_only_the_three_explicit_source_responsibilities() -> None:
    rules = _first_vpms_policy()["rules"]
    bindings = rules["source_bindings"]
    assert {item["path"]: item["responsibility"] for item in bindings["own_sources"]} == {
        "src/vpms/domain/model.py": "VPMS_CASE_AND_OUTCOME_PROTOCOL",
        "src/vpms/domain/registry.py": "VPMS_EXPLICIT_REFERENCE_BINDING",
        "src/vpms/execution/runner.py": "VPMS_RUNNER_EXECUTION_AND_NORMALIZATION",
    }
    runner = next(item for item in bindings["own_sources"] if item["path"] == "src/vpms/execution/runner.py")
    assert runner["symbols"] == ["RunnerExecution", "CaseExecutor", "run_case"]
    assert runner["excluded_symbols"] == ["run_selected_cases"]
    evidence = bindings["implementation_evidence"][0]
    assert evidence["path"] == "src/vpms/execution/adapters/command.py"
    assert not evidence["normative_ownership_created_by_evidence"]
    legacy = bindings["legacy_boundary_references"][0]
    assert legacy["policy_id"] == "SFP-0006"
    assert not legacy["normative_semantics_changed"]
    assert not legacy["consumer_policy_authority_replaced"]
    retire = bindings["retirement_evidence"][0]
    assert retire["path"] == "src/vpms/domain/selector.py"
    assert not retire["owned_by_this_policy"]
    assert not retire["physical_retirement_authorized"]
    assert retire["current_runtime_dependency_present"]
    assert set(rules["responsibility_boundary"]["owns"]) == {item["responsibility"] for item in bindings["own_sources"]}
    assert "CASE_SELECTION" in rules["responsibility_boundary"]["does_not_own"]
    assert not rules["case_and_outcome_protocol"]["purpose_vocabulary_owned"]
    assert rules["verification_protocol_scope"]["consumer_runtime_policy_dependency"] == "FORBIDDEN"


def test_first_vpms_materialization_analysis_is_class_aware_and_registered() -> None:
    registry = load_yaml("developer/policy/analysis/registry.yaml", root=ROOT)
    policy_id = _first_vpms_policy()["policy"]["id"]
    binding = next(item for item in registry["bindings"] if item["policy_id"] == policy_id)
    assert (binding["policy_class"], binding["family"]) == (VPMS, "VERI")
    result = validate_responsibility_analysis(binding["analysis_ref"], root=ROOT)
    assert result["status"] == "PASS"
    assert result["owned_authority_family_set"] == [{"policy_class": VPMS, "family": "VERI"}]
    assert result["materialization_groups"][0]["responsibility_ids"] == ["R01", "R02", "R03"]


def test_m8_opening_retains_pre_m8_verification_provenance() -> None:
    contract = load_json("developer/policy/registries/developer-policy-catalog-contracts.json", root=ROOT)
    execution = contract["application_execution"]
    assert execution["status"] == "COMPLETE"
    assert execution["m1_m7_verified"] and execution["vpms_class_materialization_enabled"]
    evidence = execution["m1_m7_verification"]
    assert evidence["snapshot_kind"] == "WORKING_TREE_ON_BASE_HEAD"
    assert evidence["base_head"] == execution["base_head"]
    assert evidence["return_code"] == evidence["failed"] == 0
    assert evidence["passed"] == 518 and evidence["skipped"] == 2
    assert evidence["test_modes"] == ["repository-architecture", "ptsip-contract"]


def test_owned_source_bindings_resolve_exact_declared_symbols() -> None:
    bindings = _first_vpms_policy()["rules"]["source_bindings"]
    for binding in bindings["own_sources"]:
        tree = ast.parse((ROOT / binding["path"]).read_text(encoding="utf-8"))
        names = {node.name for node in tree.body if isinstance(node, (ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef))}
        assert set(binding["symbols"]).issubset(names)
        assert set(binding.get("excluded_symbols", [])).isdisjoint(binding["symbols"])


@pytest.mark.parametrize("candidate_class", [VPMS, PTSIP])
def test_class_aware_preflight_allocates_then_register_checks_exact_source_class(domains: Path, candidate_class: str) -> None:
    approval_path = domains / "developer/policy/approvals/MPA-vpms-allocation.yaml"
    approval = {"schema_version": "ptsip-policy-approval-provenance/v1", "policy_class": PTSIP, "approval": {
        "approval_id": "MPA-vpms-allocation", "decision": "APPROVED", "source_kind": "PROJECT_OWNER_DIRECT_INSTRUCTION",
        "decision_source": "USER_EXPLICIT", "source_reference": "fixture explicit approval", "approval_scope": "DRAFT_CREATION",
        "target_status": "DRAFT", "implementation_authorized": True, "policy_content_review_scope": "FULL", "recorded_at": "2026-10-05",
    }}
    _write_yaml(approval_path, approval)
    analysis_path = domains / "developer/policy/analysis/PRA-isolation.yaml"
    _write_yaml(analysis_path, analysis(VPMS, ["MPD-VERI-0002"]))
    result = preflight_family_policy("VERI", approval_path, analysis_path, "G01", policy_class=VPMS, root=domains)
    assert result["allocated_policy_id"] == "MPD-VERI-0003"
    assert result["policy_class"] == VPMS
    approval["approval"]["requested_policy_id"] = result["allocated_policy_id"]
    _write_yaml(approval_path, approval)
    policy_path = domains / "developer/policy/VERI/MPD-VERI-0003.yaml"
    _write_yaml(policy_path, {"schema_version": "developer-policy/v2", "policy_class": candidate_class,
        "policy": {"id": "MPD-VERI-0003", "version": "0.0", "title": "VPMS fixture", "status": "DRAFT"}, "rules": {"fixture": {"enabled": True}}})
    if candidate_class == PTSIP:
        with pytest.raises(PolicyIdentityLifecycleError) as exc:
            register_family_policy("VERI", approval_path, analysis_path, "G01", policy_path, policy_class=VPMS, root=domains)
        assert exc.value.code == "POLICY_CLASS_MISMATCH"
        assert all(entry["id"] != "MPD-VERI-0003" for entry in load_yaml("developer/policy/index.yaml", root=domains)["policies"])
    else:
        registered = register_family_policy("VERI", approval_path, analysis_path, "G01", policy_path, policy_class=VPMS, root=domains)
        assert registered["status"] == "REGISTERED"
        entry = next(entry for entry in load_yaml("developer/policy/index.yaml", root=domains)["policies"] if entry["id"] == "MPD-VERI-0003")
        assert entry["policy_class"] == VPMS and entry["status"] == "DRAFT"
