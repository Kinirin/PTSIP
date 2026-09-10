from __future__ import annotations

import json
import subprocess
from pathlib import Path

import yaml

from developer.automation.current_dependency_gate import (
    validate_current_legacy_dependency_gate,
)
from developer.automation.planning_validator import validate_planning
from developer.automation.policy_validator import validate_developer_policy
from developer.automation.transition_evaluator import evaluate_legacy_decisions_removal


ROOT = Path(__file__).resolve().parents[2]
P01_PLAN = ROOT / "docs" / "planning" / "0.4.0" / "WU-02" / "WU-02-P01.yaml"


def _yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def test_developer_policy_control_plane_is_machine_valid() -> None:
    assert validate_developer_policy(ROOT) == ()


def test_developer_planning_control_plane_is_machine_valid() -> None:
    assert validate_planning(ROOT) == ()


def test_current_control_planes_have_zero_retired_local_policy_dependencies() -> None:
    assert validate_current_legacy_dependency_gate(ROOT) == ()


def test_removal_gate_tracks_e4_machine_completion() -> None:
    plan = _yaml(P01_PLAN)
    execution = plan["p01_e_execution_plan"]["execution_order"]
    e4 = next(
        item
        for item in execution
        if item["id"] == "P01_E4_MIGRATION_ONLY_RETIREMENT_AND_GATE_SIMPLIFICATION"
    )
    result = evaluate_legacy_decisions_removal(ROOT)

    assert "HISTORICAL_PROVENANCE_REVISION_ANCHOR_NOT_MATERIALIZED" not in result.blockers
    assert "MIGRATION_ONLY_REFERENCE_RETIREMENT_PENDING" not in result.blockers
    assert "CURRENT_LEGACY_DEPENDENCY_NONZERO" not in result.blockers

    if e4["status"] == "COMPLETE":
        assert result.state == "AUTHORIZED"
        assert result.action == "REMOVE_DECISIONS_DIRECTORY_FROM_ACTIVE_TREE"
        assert result.blockers == ()
        assert result.confirmation_required is False
    else:
        assert result.state == "HOLD_NOT_AUTHORIZED"
        assert result.action is None
        assert result.blockers == ("P01_E4_VALIDATION_NOT_COMPLETE",)
        assert result.confirmation_required is False


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
    paths = [
        ROOT / "src" / "ptsip" / "specdata" / f"SFP-{number:04d}.yaml"
        for number in range(1, 22)
    ] + [
        ROOT / "developer" / "policy" / f"MPD-{number:04d}.yaml"
        for number in range(1, 10)
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
    assert _current_relation_edges() == [
        ("SFP-0008", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0008", "RESPONSIBILITY_MAP_SEMANTIC_AXES"),
        ("SFP-0011", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-0004", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-0005", "amends", "MPD-0004", "REPOSITORY_SELF_ADOPTION_ASSUMPTION"),
    ]


def test_support_policy_never_depends_on_developer_policy() -> None:
    for source, _, target, _ in _current_relation_edges():
        assert not (source.startswith("SFP-") and target.startswith("MPD-"))


def test_current_policy_indexes_cover_self_contained_corpus() -> None:
    mpd_index = _yaml(ROOT / "developer" / "policy" / "index.yaml")
    sfp_index = _yaml(ROOT / "src" / "ptsip" / "specdata" / "support-policy-index.yaml")

    assert [item["id"] for item in mpd_index["policies"]] == [
        f"MPD-{number:04d}" for number in range(1, 10)
    ]
    assert [item["id"] for item in sfp_index["policies"]] == [
        f"SFP-{number:04d}" for number in range(1, 22)
    ]
    assert "legacy_decisions_migration" not in mpd_index

    for entry in [*mpd_index["policies"], *sfp_index["policies"]]:
        payload = _yaml(ROOT / entry["path"])
        assert payload["policy"]["id"] == entry["id"]
        assert payload["policy"]["status"] == entry["status"]


def test_support_feature_corpus_has_no_repository_specific_authority_wrapper() -> None:
    for number in range(1, 22):
        path = ROOT / "src" / "ptsip" / "specdata" / f"SFP-{number:04d}.yaml"
        text = path.read_text(encoding="utf-8")
        assert "subject_binding:" not in text
        assert "authority_role:" not in text
        assert "repository_binding:" not in text


def test_support_policy_index_has_exact_21_targets() -> None:
    payload = _yaml(ROOT / "src" / "ptsip" / "specdata" / "support-policy-index.yaml")
    assert [item["id"] for item in payload["policies"]] == [
        f"SFP-{number:04d}" for number in range(1, 22)
    ]
    assert payload["policies"][3]["status"] == "DRAFT"
    assert all(
        item["status"] == "ACTIVE"
        for index, item in enumerate(payload["policies"])
        if index != 3
    )


def test_frozen_specification_asset_remains_byte_identical() -> None:
    path = "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md"
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


def test_support_registry_projection_contains_no_repository_binding() -> None:
    subject = _yaml(
        ROOT / "src" / "ptsip" / "specdata" / "ptsip-support-authority-subject-registry.yaml"
    )
    assert "current_repository_bindings" not in subject
    assert set(subject["subject_identity_schemes"]) == {"SUPPORT_POLICY_ID"}


def test_owner_authorization_grants_remain_developer_policy_only() -> None:
    support = _yaml(
        ROOT / "src" / "ptsip" / "specdata" / "ptsip-support-authorization-registry.yaml"
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


def test_product_governance_runtime_has_zero_local_policy_tree_dependency() -> None:
    root = ROOT / "src" / "ptsip" / "governance"
    forbidden = "decisions" + "/"
    offenders = [
        path.name
        for path in root.glob("*.py")
        if forbidden in path.read_text(encoding="utf-8")
    ]
    assert offenders == []
