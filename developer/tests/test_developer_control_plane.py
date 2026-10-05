from __future__ import annotations
from developer.tests.policy_migration_helpers import source_file

import json
import re
import subprocess
from collections import Counter
from pathlib import Path

import pytest
import yaml

from developer.automation.current_dependency_gate import (
    validate_current_legacy_dependency_gate,
)
from developer.automation.planning.planning_validator import (
    _governance_source_errors,
    validate_planning,
)
from developer.automation.policy_validator import validate_developer_policy
from developer.automation.project_profile_registry import (
    validate_project_profile_registry_plane,
)
from developer.automation.transition_evaluator import evaluate_legacy_decisions_removal
import developer.automation.transition_evaluator as transition_evaluator


ROOT = Path(__file__).resolve().parents[2]


def _yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def _expected_policy_path(policy_id: str) -> str:
    family_match = re.fullmatch(
        rf"MPD-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD|SPEC|PLAN|WORK|VERI|MIGR|RELS)-[0-9]{{4}}", policy_id
    )
    if family_match is not None:
        return f"developer/policy/{family_match.group(1)}/{policy_id}.yaml"
    assert re.fullmatch(r"MPD-(?:[0-9]{4}|BOUND-[0-9]{4})", policy_id)
    return f"developer/policy/{policy_id}.yaml"


def test_developer_policy_control_plane_is_machine_valid() -> None:
    assert validate_developer_policy(ROOT) == ()


def test_project_profile_registry_plane_is_machine_valid() -> None:
    assert validate_project_profile_registry_plane(ROOT) == ()


def test_developer_planning_control_plane_is_machine_valid() -> None:
    assert validate_planning(ROOT) == ()


@pytest.mark.parametrize("source", (["projection.py"], {"module": "projection.py"}))
def test_layout_source_containers_are_not_governance_constants(source: object) -> None:
    registry = _yaml(ROOT / "developer/policy/registries/governance-source-registry.yaml")
    payload = {"target_layout": {"analysis": {"source": source}}}
    assert _governance_source_errors(payload, registry, "responsibility-map.yaml") == []


def test_layout_source_containers_preserve_nested_governance_validation() -> None:
    registry = _yaml(ROOT / "developer/policy/registries/governance-source-registry.yaml")
    payload = {"source": [
        {"approval_source": "DIRECT_PROJECT_OWNER_TEMPORARY_APPROVAL"},
        {"source": "DIRECT_PROJECT_OWNER_INSTRUCTION"},
    ]}
    errors = _governance_source_errors(payload, registry, "fixture.yaml")
    assert len(errors) == 2
    assert "source[0].approval_source" in errors[0]
    assert "source[1].source uses legacy governance source" in errors[1]


def test_current_control_planes_have_zero_retired_local_policy_dependencies() -> None:
    assert validate_current_legacy_dependency_gate(ROOT) == ()


def test_removal_gate_tracks_e4_machine_completion(monkeypatch, tmp_path: Path) -> None:
    # This unit fixture must not reactivate or preload a retired version plan.
    monkeypatch.setattr(transition_evaluator, "repository_root", lambda root: Path(root))
    monkeypatch.setattr(transition_evaluator, "validate_current_legacy_dependency_gate", lambda _root: ())
    for status in ("COMPLETE", "VALIDATION_PENDING"):
        plan = {
            "migration_stages": {"P01_E_LEGACY_REMOVAL": {
                "preauthorized_action": "REMOVE_DECISIONS_DIRECTORY_FROM_ACTIVE_TREE",
                "confirmation_required": False,
            }},
            "p01_e_execution_plan": {"execution_order": [{
                "id": transition_evaluator.E4_STAGE, "status": status,
            }]},
        }
        monkeypatch.setattr(transition_evaluator, "load_yaml", lambda _path, *, root: plan)
        result = evaluate_legacy_decisions_removal(tmp_path)
        assert result.confirmation_required is False
        if status == "COMPLETE":
            assert result.state == "AUTHORIZED"
            assert result.action == "REMOVE_DECISIONS_DIRECTORY_FROM_ACTIVE_TREE"
            assert result.blockers == ()
        else:
            assert result.state == "HOLD_NOT_AUTHORIZED"
            assert result.action is None
            assert result.blockers == ("P01_E4_VALIDATION_NOT_COMPLETE",)


def test_migration_only_tooling_and_evidence_are_retired() -> None:
    retired_paths = (
        "developer/automation/decision_reference_migrator.py",
        "developer/automation/legacy_reference_scanner.py",
        "developer/automation/registry_split_validator.py",
        "developer/policy/legacy-reference-inventory.yaml",
        "developer/policy/legacy-decisions-inventory.yaml",
        "developer/policy/legacy-decision-reference-routing.yaml",
        "developer/policy/registry-split-inventory.yaml",
        "developer/policy/split-textual-reference-review.yaml",
        "developer/policy/schemas/legacy-reference-inventory.schema.json",
        "developer/policy/schemas/legacy-decision-inventory.schema.json",
        "developer/policy/schemas/legacy-decision-reference-routing.schema.json",
        "developer/policy/schemas/registry-split-inventory.schema.json",
        "developer/policy/schemas/split-textual-reference-review.schema.json",
        "developer/policy/schemas/split-textual-reference-map.schema.json",
        "schemas/ptsip-adr-index.schema.json",
        "schemas/ptsip-adr.schema.json",
        "schemas/ptsip-adr-template.schema.json",
        "schemas/ptsip-governance-authority-registry.schema.json",
        "schemas/ptsip-governance-authority-role.schema.json",
        "schemas/ptsip-governance-authority-role-registry.schema.json",
        "schemas/ptsip-governance-authority-semantics.schema.json",
        "schemas/ptsip-governance-authority-subject-registry.schema.json",
        "schemas/ptsip-governance-subject-binding.schema.json",
    )
    assert [path for path in retired_paths if (ROOT / path).exists()] == []


def _current_relation_edges() -> list[tuple[str, str, str, str | None]]:
    mpd_index = _yaml(ROOT / "developer" / "policy" / "index.yaml")
    support_root = ROOT / "src" / "policy"
    support_index = _yaml(support_root / "index.yaml")
    paths = [
        support_root / entry["path"]
        for entry in support_index["policies"]
    ] + [
        ROOT / entry["path"]
        for entry in mpd_index["policies"]
    ]
    edges: list[tuple[str, str, str, str | None]] = []
    for path in paths:
        payload = _yaml(path)
        source = payload["policy"]["id"]
        for relation_kind in ("supersedes", "amends", "extends", "depends_on"):
            for edge in payload.get("relations", {}).get(relation_kind, []):
                edges.append((source, relation_kind, edge["policy"], edge.get("scope")))
    return edges


def test_current_policy_relations_preserve_materialized_relation_set() -> None:
    assert Counter(_current_relation_edges()) == Counter([
        ("SFP-0008", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0008", "RESPONSIBILITY_MAP_SEMANTIC_AXES"),
        ("SFP-0011", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-0001", "depends_on", "MPD-MIGR-0001", "PROFILE_PLANNING_AND_RETIRED_POLICY_CORPUS_MIGRATION"),
        ("MPD-0001", "depends_on", "MPD-PLAN-0001", "PLANNING_AUTHORITY_AND_LIFECYCLE"),
        ("MPD-0001", "depends_on", "MPD-WORK-0001", "WORK_UNIT_LIFECYCLE"),
        ("MPD-SPEC-0011", "amends", "MPD-SPEC-0004", "REPOSITORY_SELF_ADOPTION_ASSUMPTION"),
        ("MPD-0011", "extends", "MPD-SPEC-0006", "identity_and_resolution"),
        ("MPD-0011", "depends_on", "MPD-VERI-0003", "PROJECT_PROFILE_TRANSITION_VERIFICATION"),
        ("MPD-0011", "depends_on", "MPD-MIGR-0003", "PROJECT_PROFILE_VERSION_MIGRATION_AUTHORIZATION"),
        ("MPD-0011", "depends_on", "MPD-MIGR-0004", "PROJECT_PROFILE_TRANSITION_MECHANICS"),
        ("MPD-0011", "depends_on", "MPD-SPEC-0001", "developer_distribution_boundary"),
        ("MPD-0011", "depends_on", "MPD-SPEC-0026", "T2_AUTHORITY_DELTA"),
        ("MPD-0011", "depends_on", "MPD-SPEC-0027", "USER_REVISION_LINEAGE"),
        ("MPD-0012", "extends", "MPD-0010", "AGENT_CONTRACT_SEMANTIC_VERIFICATION_AND_FEATURE_EXTENSION_REFINEMENT"),
        ("MPD-0012", "extends", "MPD-SPEC-0007", "AGENT_CONTRACT_SEMANTIC_VERIFICATION_AND_FEATURE_EXTENSION_REFINEMENT"),
        ("MPD-0012", "depends_on", "MPD-VERI-0004", "AGENT_CONTRACT_SEMANTIC_VERIFICATION"),
        ("MPD-0012", "depends_on", "MPD-0010", "INFERENCE_COST_EXTENSION_AND_IDENTITY_BASELINE"),
        ("MPD-0012", "depends_on", "MPD-SPEC-0006", "INFERENCE_COST_EXTENSION_AND_IDENTITY_BASELINE"),
        ("MPD-0012", "depends_on", "MPD-SPEC-0007", "INFERENCE_COST_EXTENSION_AND_IDENTITY_BASELINE"),
        ("MPD-0012", "depends_on", "MPD-SPEC-0024", "AGENT_CONTRACT_NORMATIVE_AUTHORITY"),
        ("MPD-0012", "depends_on", "MPD-SPEC-0025", "SOURCE_AND_FEATURE_CONTRACT_MODEL"),
        ("MPD-0012", "depends_on", "MPD-PLAN-0004", "AGENT_CONTRACT_DEFERRED_WORK_AND_PENDING_MATERIALIZATION"),
        ("MPD-0013", "depends_on", "MPD-SPEC-0023", "POLICY_PLAN_BINDING_CONTRACT"),
        ("MPD-0013", "depends_on", "MPD-PLAN-0001", "PLANNING_AUTHORITY_AND_LIFECYCLE"),
        ("MPD-0015", "depends_on", "MPD-VERI-0005", "EXECUTION_EVIDENCE_AND_EXPECTED_OBSERVED_VERIFICATION"),
        ("MPD-0015", "depends_on", "MPD-MIGR-0005", "IMPLEMENTATION_MIGRATION_REFERENCE_AND_AUTHORIZATION"),
        ("MPD-0015", "depends_on", "MPD-SPEC-0021", "LANGUAGE_NEUTRAL_NORMATIVE_AUTHORITY"),
        ("MPD-0015", "depends_on", "MPD-SPEC-0022", "CONTRACT_ARTIFACT_REPRESENTATION"),
        ("MPD-0015", "depends_on", "MPD-WORK-0002", "EXECUTION_LIFECYCLE_AND_RESULT_HANDOFF"),
        ("MPD-0017", "depends_on", "MPD-0010", "EXACT_MACHINE_RESOLUTION_AND_INFERENCE_COST_BASELINE"),
        ("MPD-0018", "extends", "MPD-BOUND-0001", "POLICY_CLASS_AND_RESPONSIBILITY_FAMILY_COMPOSITE_IDENTITY"),
        ("MPD-0018", "depends_on", "MPD-BOUND-0001", "AUTHORITY_LOOKUP_ISOLATION"),
        ("MPD-BOUND-0001", "depends_on", "MPD-WORK-0003", "RESPONSIBILITY_FAMILY_DECOMPOSITION_AND_AUTHORITY_RECONCILIATION_BASELINE"),
        ("MPD-MIGR-0001", "depends_on", "MPD-PLAN-0001", "PLANNING_TARGET_AUTHORITY"),
        ("MPD-MIGR-0002", "depends_on", "MPD-0016", "README_TRANSLATION_GOVERNANCE"),
        ("MPD-MIGR-0002", "depends_on", "MPD-MIGR-0001", "DEVELOPER_REPOSITORY_MIGRATION_BOUNDARY"),
        ("MPD-MIGR-0003", "depends_on", "MPD-0011", "PROJECT_PROFILE_TRANSITION_APPROVAL_BASELINE"),
        ("MPD-MIGR-0004", "depends_on", "MPD-MIGR-0003", "PROJECT_PROFILE_VERSION_MIGRATION_AUTHORIZATION"),
        ("MPD-MIGR-0004", "depends_on", "MPD-SPEC-0006", "IDENTITY_AND_RESOLUTION"),
        ("MPD-MIGR-0004", "depends_on", "MPD-SPEC-0026", "T2_AUTHORITY_DELTA"),
        ("MPD-MIGR-0004", "depends_on", "MPD-SPEC-0027", "USER_REVISION_LINEAGE"),
        ("MPD-MIGR-0005", "depends_on", "MPD-SPEC-0021", "LANGUAGE_NEUTRAL_NORMATIVE_AUTHORITY"),
        ("MPD-MIGR-0005", "depends_on", "MPD-SPEC-0022", "CONTRACT_ARTIFACT_REPRESENTATION"),
        ("MPD-PLAN-0001", "depends_on", "MPD-SPEC-0023", "POLICY_PLAN_BINDING_CONTRACT"),
        ("MPD-PLAN-0002", "depends_on", "MPD-PLAN-0001", "PLANNING_AUTHORITY_AND_LIFECYCLE"),
        ("MPD-PLAN-0003", "depends_on", "MPD-PLAN-0001", "PLANNING_AUTHORITY_AND_LIFECYCLE"),
        ("MPD-PLAN-0004", "depends_on", "MPD-PLAN-0001", "PLANNING_AUTHORITY_AND_LIFECYCLE"),
        ("MPD-RELS-0002", "depends_on", "MPD-VERI-0003", "PROJECT_PROFILE_TRANSITION_VERIFICATION"),
        ("MPD-SPEC-0004", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-SPEC-0006", "extends", "MPD-SPEC-0001", "DEVELOPER_CANONICAL_SEMANTIC_CONTRACT_AND_EXTENSION_BOUNDARY"),
        ("MPD-SPEC-0022", "extends", "MPD-SPEC-0021", "LANGUAGE_NEUTRAL_NORMATIVE_AUTHORITY"),
        ("MPD-SPEC-0025", "extends", "MPD-SPEC-0007", "AGENT_CONTRACT_FEATURE_EXTENSION_MODEL"),
        ("MPD-SPEC-0025", "depends_on", "MPD-SPEC-0024", "AGENT_CONTRACT_NORMATIVE_AUTHORITY"),
        ("MPD-SPEC-0026", "depends_on", "MPD-SPEC-0006", "identity_and_resolution"),
        ("MPD-VERI-0001", "depends_on", "MPD-SPEC-0023", "POLICY_PLAN_BINDING_CONTRACT"),
        ("MPD-VERI-0001", "depends_on", "MPD-PLAN-0001", "PLANNING_AUTHORITY_AND_LIFECYCLE"),
        ("MPD-VERI-0003", "depends_on", "MPD-MIGR-0004", "PROJECT_PROFILE_TRANSITION_MECHANICS"),
        ("MPD-VERI-0004", "depends_on", "MPD-SPEC-0006", "IDENTITY_AND_RESOLUTION"),
        ("MPD-VERI-0004", "depends_on", "MPD-SPEC-0024", "AGENT_CONTRACT_NORMATIVE_AUTHORITY"),
        ("MPD-VERI-0004", "depends_on", "MPD-SPEC-0025", "SOURCE_AND_FEATURE_CONTRACT_MODEL"),
        ("MPD-VERI-0005", "depends_on", "MPD-WORK-0002", "EXECUTION_LIFECYCLE_AND_RESULT_HANDOFF"),
        ("MPD-VERI-0006", "depends_on", "MPD-0017", "MANAGEMENT_NEXT_ACTION_IDENTITY_AND_EXECUTION_CONTROL_PLANE"),
        ("MPD-VERI-0006", "depends_on", "MPD-VERI-0001", "POLICY_PLAN_CONSISTENCY_VERIFICATION_SEMANTICS"),
        ("MPD-VERI-0007", "depends_on", "MPD-BOUND-0001", "SHARED_DEVELOPER_POLICY_CLASS_AND_RESPONSIBILITY_FAMILY_ISOLATION"),
        ("MPD-WORK-0002", "depends_on", "MPD-WORK-0001", "WORK_UNIT_LIFECYCLE"),
        ("MPD-WORK-0003", "depends_on", "MPD-WORK-0001", "WORK_UNIT_LIFECYCLE"),
        ("MPD-WORK-0003", "depends_on", "MPD-WORK-0002", "EXECUTION_LIFECYCLE_AND_RESULT_HANDOFF"),
    ])

def test_support_policy_never_depends_on_developer_policy() -> None:
    for source, _, target, _ in _current_relation_edges():
        assert not (source.startswith("SFP-") and target.startswith("MPD-"))


def test_current_policy_indexes_cover_self_contained_corpus() -> None:
    mpd_index = _yaml(ROOT / "developer" / "policy" / "index.yaml")
    support_root = ROOT / "src" / "policy"
    sfp_index = _yaml(support_root / "index.yaml")

    discovered_mpd_ids = [
        path.stem
        for path in sorted((ROOT / "developer" / "policy").rglob("MPD-*.yaml"))
    ]
    indexed_mpd_ids = [item["id"] for item in mpd_index["policies"]]
    assert len(indexed_mpd_ids) == len(set(indexed_mpd_ids))
    assert Counter(indexed_mpd_ids) == Counter(discovered_mpd_ids)
    for item in mpd_index["policies"]:
        policy_id = item["id"]
        assert item["path"] == source_file(ROOT / _expected_policy_path(policy_id)).relative_to(ROOT).as_posix()
    sfp_ids = [item["id"] for item in sfp_index["policies"]]
    assert sfp_ids[:24] == [
        f"SFP-{number:04d}" for number in range(1, 25)
    ]
    assert len(sfp_ids) == len(set(sfp_ids))
    assert "legacy_decisions_migration" not in mpd_index

    for entry in mpd_index["policies"]:
        payload = _yaml(ROOT / entry["path"])
        assert payload["policy"]["id"] == entry["id"]
        assert payload["policy"]["status"] == entry["status"]
    for entry in sfp_index["policies"]:
        payload = _yaml(support_root / entry["path"])
        assert payload["policy"]["id"] == entry["id"]
        assert payload["policy"]["status"] == entry["status"]


@pytest.mark.parametrize(
    ("policy_id", "expected_path"),
    [
        ("MPD-0001", "developer/policy/MPD-0001.yaml"),
        ("MPD-BOUND-0001", "developer/policy/MPD-BOUND-0001.yaml"),
        ("MPD-SPEC-0001", "developer/policy/SPEC/MPD-SPEC-0001.yaml"),
        ("MPD-PLAN-0001", "developer/policy/PLAN/MPD-PLAN-0001.yaml"),
        ("MPD-WORK-0001", "developer/policy/WORK/MPD-WORK-0001.yaml"),
        ("MPD-VERI-0001", "developer/policy/VERI/MPD-VERI-0001.yaml"),
        ("MPD-MIGR-0001", "developer/policy/MIGR/MPD-MIGR-0001.yaml"),
        ("MPD-RELS-0001", "developer/policy/RELS/MPD-RELS-0001.yaml"),
        ("MPD-ARCH-0001", "developer/policy/ARCH/MPD-ARCH-0001.yaml"),
    ],
)
def test_policy_path_fixture_distinguishes_boundary_from_families(
    policy_id: str, expected_path: str
) -> None:
    assert _expected_policy_path(policy_id) == expected_path
    assert _expected_policy_path("MPD-BOUND-0001") != (
        "developer/policy/BOUND/MPD-BOUND-0001.yaml"
    )


def test_policy_path_fixture_rejects_unregistered_family() -> None:
    with pytest.raises(AssertionError):
        _expected_policy_path("MPD-UNKNOWN-0001")



def test_mpd_0011_declares_incremental_implementation_state() -> None:
    payload = _yaml(source_file(ROOT / "developer" / "policy" / "MPD-0011.yaml"))
    state = payload["rules"]["implementation_state"]

    assert state["operationalization_level"] == "L4_LOCAL_AUTO_REMEDIATION_L3_REMOTE_AND_RELEASE_VERIFY"
    assert state["policy_record"] == "MATERIALIZED"
    assert state["policy_index_registration"] == "MATERIALIZED"
    assert state["policy_resolver_routing"] == "MATERIALIZED"
    assert state["authority_plane_registration"] == "MATERIALIZED"
    assert state["transition_reconciler"] == "MATERIALIZED_AND_HOOK_INVOKED"
    assert state["h3_hook_activation"] == "MATERIALIZED_SHARED_INSTALLER"
    assert state["remote_commit_verifier"] == "MATERIALIZED_PUSH_VERIFY_ONLY"
    assert state["release_transition_verifier"] == "MATERIALIZED_EXACT_SHA_VERIFY_ONLY"
    assert state["claim"] == "LOCAL_REMOTE_RELEASE_PP_AUTOMATION_ACTIVE"

def test_support_feature_corpus_has_no_repository_specific_authority_wrapper() -> None:
    support_root = ROOT / "src" / "policy"
    index = _yaml(support_root / "index.yaml")
    for entry in index["policies"]:
        path = support_root / entry["path"]
        text = path.read_text(encoding="utf-8")
        assert "subject_binding:" not in text
        assert "authority_role:" not in text
        assert "repository_binding:" not in text


def test_support_policy_index_has_exact_targets_and_active_vpms_boundary() -> None:
    payload = _yaml(ROOT / "src" / "policy" / "index.yaml")
    policy_ids = [item["id"] for item in payload["policies"]]
    assert policy_ids[:24] == [
        f"SFP-{number:04d}" for number in range(1, 25)
    ]
    assert len(policy_ids) == len(set(policy_ids))
    assert payload["policies"][3]["status"] == "DRAFT"
    assert payload["policies"][22] == {
        "id": "SFP-0023", "path": "legacy/SFP-0023.yaml", "status": "ACTIVE", "authority_role": "MIGRATION_SOURCE",
    }
    assert payload["policies"][23] == {
        "id": "SFP-0024", "path": "legacy/SFP-0024.yaml", "status": "ACTIVE", "authority_role": "MIGRATION_SOURCE",
    }
    assert payload["policies"][5] == {
        "id": "SFP-0006", "path": "legacy/SFP-0006.yaml", "status": "RETIRED", "authority_role": "MIGRATION_SOURCE",
    }
    assert all(
        item["status"] == "ACTIVE"
        for index, item in enumerate(payload["policies"][:24])
        if index not in {3, 5}
    )


def test_policy_namespace_distinguishes_canonical_source_and_installed_projection() -> None:
    payload = _yaml(source_file(ROOT / "developer/policy/SPEC/MPD-SPEC-0001.yaml"))
    support = payload["rules"]["namespace"]["support_feature"]
    assert support["machine_policy_path"] == "src/policy/"
    assert support["machine_policy_index"] == "src/policy/index.yaml"
    assert support["machine_policy_pattern"] == "src/policy/SFP-*.yaml"
    assert support["canonical_schema_path"] == "src/policy/schemas/ptsip-support-feature-policy.schema.json"
    assert (ROOT / support["canonical_schema_path"]).is_file()
    assert support["embedded_schema_path"] == "ptsip/support/schemas/ptsip-support-feature-policy.schema.json"
    assert support["embedded_schema_path_role"] == "INSTALLED_DISTRIBUTION_PROJECTION"
    assert support["distribution"] == "REQUIRED"


def test_frozen_machine_specification_registry_remains_byte_identical() -> None:
    path = "registry/ptsip-registry.yaml"
    revision = "3c47816770d194ae42f98faedc911d980db0e62a"
    current = subprocess.run(
        ["git", "rev-parse", f"HEAD:{path}"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    frozen = subprocess.run(
        ["git", "rev-parse", f"{revision}:{path}"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    assert current == frozen


def test_current_generic_authority_schema_names_resolve_to_support_contracts() -> None:
    aliases = {
        "schemas/ptsip-project-authority-record.schema.json": "ptsip-support-project-authority-record.schema.json",
        "schemas/ptsip-authority-eligibility-result.schema.json": "ptsip-support-authority-eligibility-result.schema.json",
    }
    for relative, target in aliases.items():
        payload = json.loads((ROOT / relative).read_text(encoding="utf-8"))
        assert payload["$ref"] == target



def test_developer_authority_subject_registry_tracks_policy_index() -> None:
    index = _yaml(ROOT / "developer" / "policy" / "index.yaml")
    subject = _yaml(
        ROOT / "developer" / "policy" / "registries" / "authority-subject-registry.yaml"
    )

    registered = subject["subject_identity_schemes"]["MANAGEMENT_POLICY_ID"]["registered_values"]
    assert registered == [item["id"] for item in index["policies"]]

def test_support_registry_projection_contains_no_repository_binding() -> None:
    subject = _yaml(
        ROOT / "src" / "policy" / "registries" / "ptsip-support-authority-subject-registry.yaml"
    )
    assert "current_repository_bindings" not in subject
    assert set(subject["subject_identity_schemes"]) == {"SUPPORT_POLICY_ID"}


def test_owner_authorization_grants_remain_developer_policy_only() -> None:
    support = _yaml(
        ROOT / "src" / "policy" / "registries" / "ptsip-support-authorization-registry.yaml"
    )
    developer = _yaml(
        ROOT / "developer" / "policy" / "registries" / "authorization-transition-registry.yaml"
    )
    assert "authorization_provenance" not in support
    assert "rules" not in support
    assert developer["authorization_provenance"]["authority"] == "PROJECT_OWNER"
    assert "P03G_PROJECT_AUTHORITY_RUNTIME" in developer["rules"]


def test_developer_owner_authorization_uses_mpd_registry() -> None:
    from developer.automation.authorization_transition import (
        DeveloperAuthorizationTransitionEvaluator,
    )
    from ptsip.governance import AuthorizationState

    evaluator = DeveloperAuthorizationTransitionEvaluator(ROOT)
    readiness = evaluator.derive_project_authority_runtime_readiness()
    assert all(readiness.values())
    results = evaluator.evaluate_current_project_authority_runtime()
    assert {item.state for item in results} == {AuthorizationState.AUTHORIZED}


@pytest.mark.parametrize("defect", ["missing", "duplicate", "unknown", "empty"])
def test_owner_authorization_requires_exact_validated_support_corpus(monkeypatch, defect: str) -> None:
    import developer.automation.authorization_transition as transitions
    from ptsip.governance import AuthorizationState
    from ptsip.governance.authority import AuthorityCatalog

    catalog = AuthorityCatalog(ROOT)
    ids = tuple(catalog.current_routes)
    invalid = {
        "missing": ids[:-1],
        "duplicate": (*ids[:-1], ids[0]),
        "unknown": (*ids[:-1], "SFP-9999"),
        "empty": (),
    }[defect]
    monkeypatch.setattr(catalog, "validate_current_corpus", lambda: invalid)
    monkeypatch.setattr(transitions, "AuthorityCatalog", lambda _root: catalog)
    evaluator = transitions.DeveloperAuthorizationTransitionEvaluator(ROOT)
    readiness = evaluator.derive_project_authority_runtime_readiness()
    assert readiness["CURRENT_SUPPORT_POLICY_CORPUS_VALID"] is False
    assert {item.state for item in evaluator.evaluate_current_project_authority_runtime()} == {
        AuthorizationState.HOLD_NOT_AUTHORIZED
    }


def test_product_governance_runtime_has_zero_local_policy_tree_dependency() -> None:
    root = ROOT / "src" / "ptsip" / "governance"
    forbidden = "decisions" + "/"
    offenders = [
        path.name
        for path in root.glob("*.py")
        if forbidden in path.read_text(encoding="utf-8")
    ]
    assert offenders == []
