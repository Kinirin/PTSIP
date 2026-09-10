from __future__ import annotations

from pathlib import Path

from developer.automation.planning_validator import validate_planning
from developer.automation.policy_validator import validate_developer_policy
from developer.automation.transition_evaluator import evaluate_legacy_decisions_removal


ROOT = Path(__file__).resolve().parents[2]


def test_developer_policy_control_plane_is_machine_valid() -> None:
    assert validate_developer_policy(ROOT) == ()


def test_developer_planning_control_plane_is_machine_valid() -> None:
    assert validate_planning(ROOT) == ()


def test_legacy_decisions_removal_is_preauthorized_but_currently_held() -> None:
    result = evaluate_legacy_decisions_removal(ROOT)
    assert result.state == "HOLD_NOT_AUTHORIZED"
    assert result.action is None
    assert result.confirmation_required is False
    assert "SRC_PTSIP_GOVERNANCE_READS_DECISIONS" in result.blockers


def test_legacy_decision_inventory_is_complete_and_boundary_classified() -> None:
    import yaml

    inventory_path = ROOT / "developer" / "policy" / "legacy-decisions-inventory.yaml"
    inventory = yaml.safe_load(inventory_path.read_text(encoding="utf-8"))
    assert inventory["summary"] == {
        "MPD": 2,
        "SFP": 15,
        "SPLIT": 6,
        "RETIRE": 0,
        "total": 23,
        "planned_sfp_targets": 21,
        "planned_mpd_targets": 8,
        "retired_fragments": 0,
    }
    assert [item["source_id"] for item in inventory["entries"]] == [
        f"ADR-{number:04d}" for number in range(1, 24)
    ]
    assert all(
        output.get("target_id") is None or not output["target_id"].startswith("ADR-")
        for item in inventory["entries"]
        for output in item["outputs"]
    )


def test_split_decision_routes_are_fixed_before_materialization() -> None:
    from developer.automation.decision_reference_migrator import canonical_targets, project_relation

    assert canonical_targets("ADR-0003", root=ROOT) == ("SFP-0003", "MPD-0002")
    assert canonical_targets("ADR-0005", root=ROOT) == ("SFP-0005", "MPD-0003")
    assert canonical_targets("ADR-0011", root=ROOT) == ("SFP-0011", "MPD-0004")
    assert canonical_targets("ADR-0017", root=ROOT) == ("SFP-0017", "MPD-0005")
    assert canonical_targets("ADR-0021", root=ROOT) == ("SFP-0019", "MPD-0006")
    assert canonical_targets("ADR-0023", root=ROOT) == ("SFP-0021", "MPD-0007")
    assert canonical_targets("ADR-0018", root=ROOT) == ("MPD-0008",)
    assert canonical_targets("ADR-0020", root=ROOT) == ("MPD-0009",)

    assert project_relation(
        "ADR-0017", "amends", "ADR-0011",
        scope="REPOSITORY_SELF_ADOPTION_ASSUMPTION", root=ROOT
    )[0].source == "MPD-0005"
    assert project_relation(
        "ADR-0017", "amends", "ADR-0011",
        scope="REPOSITORY_SELF_ADOPTION_ASSUMPTION", root=ROOT
    )[0].target == "MPD-0004"


def test_unregistered_split_relation_fails_closed() -> None:
    import pytest
    from developer.automation.decision_reference_migrator import (
        UnroutedDecisionReferenceError,
        project_relation,
    )

    with pytest.raises(UnroutedDecisionReferenceError):
        project_relation("ADR-0017", "depends_on", "ADR-0021", root=ROOT)


def test_all_legacy_machine_relations_have_deterministic_projection_routes() -> None:
    # validate_developer_policy walks every legacy ADR relation and fails closed
    # if a SPLIT source/target lacks an explicit route.
    assert validate_developer_policy(ROOT) == ()


def test_machine_reference_scan_never_rewrites_by_textual_lineage_rule() -> None:
    from developer.automation.decision_reference_migrator import scan_machine_references

    refs = scan_machine_references(ROOT)
    assert isinstance(refs, dict)
    # Existing machine references are allowed during migration, but must be
    # surfaced separately rather than silently rewritten.
    for path in refs:
        assert not path.endswith((".md", ".txt", ".rst"))


def test_split_textual_references_are_never_auto_rewritten() -> None:
    from developer.automation.decision_reference_migrator import rewrite_textual_reference

    text = "Authority: ADR-0017 and ADR-0019"
    rewritten = rewrite_textual_reference(text, root=ROOT)
    assert "ADR-0017" in rewritten
    assert "SFP-0017 + MPD-0005" not in rewritten
    assert "SFP-0018" in rewritten


def test_split_textual_review_blocks_legacy_removal_until_manually_complete() -> None:
    result = evaluate_legacy_decisions_removal(ROOT)
    assert "SPLIT_TEXTUAL_REFERENCE_REVIEW_INCOMPLETE" in result.blockers


def test_temporary_split_mapping_dry_run_is_exact_and_apply_is_blocked() -> None:
    from developer.automation.decision_reference_migrator import (
        simulate_temporary_split_map,
        temporary_split_map_apply_blockers,
    )

    previews = simulate_temporary_split_map(ROOT)
    assert len(previews) == 16
    assert {item.file_path for item in previews} == {
        "releasenote/README.md",
        "releasenote/project-profile/pp.1.01.md",
        "releasenote/specification/spec-0.2.0-draft.md",
        "releasenote/specification/spec-0.3.4-draft.md",
        "releasenote/specification/spec-0.3.7-draft.md",
        "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
    }
    blockers = temporary_split_map_apply_blockers(ROOT)
    assert "SPLIT_TARGET_POLICIES_NOT_MATERIALIZED" in blockers
    assert "TEXTUAL_POLICY_BOUNDARY_FINDINGS_OPEN" in blockers


def test_split_map_preserves_occurrence_level_choice_on_same_line() -> None:
    from developer.automation.decision_reference_migrator import simulate_temporary_split_map

    previews = {
        item.mapping_id: item
        for item in simulate_temporary_split_map(ROOT)
    }
    assert previews["TMP-SPLIT-001"].selected_target == "MPD-0006"
    assert previews["TMP-SPLIT-002"].selected_target == "SFP-0019"


def test_relation_migration_manifest_covers_all_five_legacy_edges() -> None:
    import yaml

    path = ROOT / "developer" / "policy" / "policy-relation-migration.yaml"
    payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert payload["source_relation_count"] == 5
    assert len(payload["relations"]) == 5
    projected = [
        (edge["source_policy"], edge["relation"], edge["target_policy"], edge["scope"])
        for item in payload["relations"]
        for edge in item["projected_edges"]
    ]
    assert projected == [
        ("SFP-0008", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0008", "RESPONSIBILITY_MAP_SEMANTIC_AXES"),
        ("SFP-0011", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-0004", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-0005", "amends", "MPD-0004", "REPOSITORY_SELF_ADOPTION_ASSUMPTION"),
    ]


def test_support_policy_never_depends_on_developer_policy() -> None:
    import yaml

    path = ROOT / "developer" / "policy" / "policy-relation-migration.yaml"
    payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    for item in payload["relations"]:
        for edge in item["projected_edges"]:
            assert not (
                edge["source_policy"].startswith("SFP-")
                and edge["target_policy"].startswith("MPD-")
            )


def test_all_29_migrated_policy_files_match_deterministic_materializer() -> None:
    from developer.automation.policy_loader import load_yaml
    from developer.automation.policy_materializer import expected_materialized_policies

    expected = expected_materialized_policies(ROOT)
    assert len(expected) == 29
    assert sum(path.startswith("src/ptsip/specdata/SFP-") for path in expected) == 21
    assert sum(path.startswith("developer/policy/MPD-") for path in expected) == 8
    for path, payload in expected.items():
        assert load_yaml(path, root=ROOT) == payload


def test_support_feature_corpus_has_no_repository_specific_authority_wrapper() -> None:
    for number in range(1, 22):
        path = ROOT / "src" / "ptsip" / "specdata" / f"SFP-{number:04d}.yaml"
        text = path.read_text(encoding="utf-8")
        assert "subject_binding:" not in text
        assert "authority_role:" not in text
        assert "repository_binding:" not in text


def test_support_policy_index_has_exact_21_targets() -> None:
    import yaml

    path = ROOT / "src" / "ptsip" / "specdata" / "support-policy-index.yaml"
    payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert [item["id"] for item in payload["policies"]] == [
        f"SFP-{number:04d}" for number in range(1, 22)
    ]
    assert payload["policies"][3]["status"] == "DRAFT"
    assert all(
        item["status"] == "ACTIVE"
        for index, item in enumerate(payload["policies"])
        if index != 3
    )


def test_product_normative_profile_transition_spec_has_no_mpd_0009_lineage() -> None:
    path = ROOT / "spec" / "PTSIP-DRAFT-PROFILE-TRANSITION.md"
    text = path.read_text(encoding="utf-8")
    assert "ADR-0020" not in text
    assert "MPD-0009" not in text


def test_boundary_findings_are_resolved_before_split_mapping_apply() -> None:
    import yaml

    path = ROOT / "developer" / "policy" / "tmp-split-textual-reference-map.yaml"
    payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert payload["application_gate"]["boundary_findings_resolved"] is True
    assert payload["boundary_findings"]
    assert all(item["status"] == "RESOLVED" for item in payload["boundary_findings"])
