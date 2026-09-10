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
    assert "ACTIVE_REFERENCE_COUNT_NONZERO" not in result.blockers
    assert "MIGRATION_ONLY_REFERENCE_RETIREMENT_PENDING" in result.blockers
    assert "HISTORICAL_PROVENANCE_REVISION_ANCHOR_NOT_MATERIALIZED" in result.blockers


def test_legacy_reference_inventory_is_machine_valid_and_fail_closed() -> None:
    import yaml
    from developer.automation.legacy_reference_scanner import (
        scan_summary,
        validate_legacy_reference_inventory,
    )

    assert validate_legacy_reference_inventory(ROOT) == ()
    summary = scan_summary(ROOT)
    inventory = yaml.safe_load(
        (ROOT / "developer" / "policy" / "legacy-reference-inventory.yaml").read_text(
            encoding="utf-8"
        )
    )
    expected = inventory["active_dependencies"]

    assert summary["counts"].get("UNCLASSIFIED", 0) == 0
    assert summary["counts"].get("ACTIVE_DEPENDENCY", 0) == expected[
        "expected_reference_count"
    ]
    assert summary["files"].get("ACTIVE_DEPENDENCY", []) == sorted(
        item["path"] for item in expected["entries"]
    )


def test_legacy_corpus_provenance_anchor_matches_current_decisions_tree() -> None:
    import subprocess
    import yaml

    inventory = yaml.safe_load(
        (ROOT / "developer" / "policy" / "legacy-reference-inventory.yaml").read_text(
            encoding="utf-8"
        )
    )
    basis = inventory["scan_basis"]
    anchor = subprocess.run(
        ["git", "rev-parse", f"{basis['legacy_corpus_revision']}:decisions"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    current = subprocess.run(
        ["git", "rev-parse", "HEAD:decisions"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    assert anchor == basis["legacy_decisions_tree_sha"]
    assert current == anchor


def test_split_decision_routes_remain_available_for_remaining_reference_retirement() -> None:
    from developer.automation.decision_reference_migrator import (
        canonical_targets,
        project_relation,
    )

    assert canonical_targets("ADR-0003", root=ROOT) == ("SFP-0003", "MPD-0002")
    assert canonical_targets("ADR-0005", root=ROOT) == ("SFP-0005", "MPD-0003")
    assert canonical_targets("ADR-0011", root=ROOT) == ("SFP-0011", "MPD-0004")
    assert canonical_targets("ADR-0017", root=ROOT) == ("SFP-0017", "MPD-0005")
    assert canonical_targets("ADR-0021", root=ROOT) == ("SFP-0019", "MPD-0006")
    assert canonical_targets("ADR-0023", root=ROOT) == ("SFP-0021", "MPD-0007")
    assert canonical_targets("ADR-0018", root=ROOT) == ("MPD-0008",)
    assert canonical_targets("ADR-0020", root=ROOT) == ("MPD-0009",)

    assert project_relation(
        "ADR-0017",
        "amends",
        "ADR-0011",
        scope="REPOSITORY_SELF_ADOPTION_ASSUMPTION",
        root=ROOT,
    )[0].source == "MPD-0005"
    assert project_relation(
        "ADR-0017",
        "amends",
        "ADR-0011",
        scope="REPOSITORY_SELF_ADOPTION_ASSUMPTION",
        root=ROOT,
    )[0].target == "MPD-0004"


def test_unregistered_split_relation_fails_closed() -> None:
    import pytest
    from developer.automation.decision_reference_migrator import (
        UnroutedDecisionReferenceError,
        project_relation,
    )

    with pytest.raises(UnroutedDecisionReferenceError):
        project_relation("ADR-0017", "depends_on", "ADR-0021", root=ROOT)


def test_current_policy_relations_are_machine_valid_without_legacy_derivation() -> None:
    assert validate_developer_policy(ROOT) == ()


def test_machine_reference_scan_never_rewrites_by_textual_lineage_rule() -> None:
    from developer.automation.decision_reference_migrator import scan_machine_references

    refs = scan_machine_references(ROOT)
    assert isinstance(refs, dict)
    for path in refs:
        assert not path.endswith((".md", ".txt", ".rst"))


def test_split_textual_references_are_never_auto_rewritten() -> None:
    from developer.automation.decision_reference_migrator import rewrite_textual_reference

    text = "Authority: ADR-0017 and ADR-0019"
    rewritten = rewrite_textual_reference(text, root=ROOT)
    assert "ADR-0017" in rewritten
    assert "SFP-0017 + MPD-0005" not in rewritten
    assert "SFP-0018" in rewritten


def _current_relation_edges() -> list[tuple[str, str, str, str | None]]:
    import yaml

    paths = [
        ROOT / "src" / "ptsip" / "specdata" / f"SFP-{number:04d}.yaml"
        for number in range(1, 22)
    ] + [
        ROOT / "developer" / "policy" / f"MPD-{number:04d}.yaml"
        for number in range(1, 10)
    ]
    edges: list[tuple[str, str, str, str | None]] = []
    for path in paths:
        payload = yaml.safe_load(path.read_text(encoding="utf-8"))
        source = payload["policy"]["id"]
        for relation_kind in ("supersedes", "amends", "extends", "depends_on"):
            for edge in payload.get("relations", {}).get(relation_kind, []):
                edges.append(
                    (source, relation_kind, edge["policy"], edge.get("scope"))
                )
    return edges


def test_current_policy_relations_preserve_materialized_relation_set() -> None:
    assert _current_relation_edges() == [
        ("SFP-0008", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        ("SFP-0009", "depends_on", "SFP-0007", "PRIMARY_LIFECYCLE_ONTOLOGY"),
        (
            "SFP-0009",
            "depends_on",
            "SFP-0008",
            "RESPONSIBILITY_MAP_SEMANTIC_AXES",
        ),
        ("SFP-0011", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        ("MPD-0004", "depends_on", "SFP-0010", "PROFILE_TRANSITION_SEMANTICS"),
        (
            "MPD-0005",
            "amends",
            "MPD-0004",
            "REPOSITORY_SELF_ADOPTION_ASSUMPTION",
        ),
    ]


def test_support_policy_never_depends_on_developer_policy() -> None:
    for source, _, target, _ in _current_relation_edges():
        assert not (source.startswith("SFP-") and target.startswith("MPD-"))


def test_current_policy_indexes_cover_self_contained_corpus() -> None:
    import yaml

    mpd_index = yaml.safe_load(
        (ROOT / "developer" / "policy" / "index.yaml").read_text(encoding="utf-8")
    )
    sfp_index = yaml.safe_load(
        (
            ROOT / "src" / "ptsip" / "specdata" / "support-policy-index.yaml"
        ).read_text(encoding="utf-8")
    )

    assert [item["id"] for item in mpd_index["policies"]] == [
        f"MPD-{number:04d}" for number in range(1, 10)
    ]
    assert [item["id"] for item in sfp_index["policies"]] == [
        f"SFP-{number:04d}" for number in range(1, 22)
    ]
    assert "legacy_decisions_migration" not in mpd_index

    for entry in [*mpd_index["policies"], *sfp_index["policies"]]:
        payload = yaml.safe_load((ROOT / entry["path"]).read_text(encoding="utf-8"))
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


def test_split_textual_reference_migration_preserves_frozen_spec_revision() -> None:
    import subprocess
    import yaml

    review_path = ROOT / "developer" / "policy" / "split-textual-reference-review.yaml"
    review = yaml.safe_load(review_path.read_text(encoding="utf-8"))
    assert review["generation"]["status"] == "COMPLETE"
    exclusion = review["excluded_revision_bound_documents"][0]
    assert exclusion["path"] == "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md"
    assert exclusion["revision"] == "3c47816770d194ae42f98faedc911d980db0e62a"
    assert exclusion["classification"] == "FROZEN_REVISION_LINEAGE"
    assert exclusion["active_reference"] is False
    assert exclusion["rewrite_allowed"] is False

    current = subprocess.run(
        ["git", "rev-parse", "HEAD:spec/PTSIP-DRAFT-PROFILE-TRANSITION.md"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    frozen = subprocess.run(
        [
            "git",
            "rev-parse",
            "3c47816770d194ae42f98faedc911d980db0e62a:spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
        ],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    assert current == frozen


def test_mutable_split_markdown_references_are_migrated() -> None:
    split_ids = {
        "ADR-0003",
        "ADR-0005",
        "ADR-0011",
        "ADR-0017",
        "ADR-0021",
        "ADR-0023",
    }
    mutable_migrated_files = (
        ROOT / "releasenote" / "README.md",
        ROOT / "releasenote" / "project-profile" / "pp.1.01.md",
        ROOT / "releasenote" / "specification" / "spec-0.2.0-draft.md",
        ROOT / "releasenote" / "specification" / "spec-0.3.4-draft.md",
        ROOT / "releasenote" / "specification" / "spec-0.3.7-draft.md",
    )
    for path in mutable_migrated_files:
        text = path.read_text(encoding="utf-8")
        assert not any(split_id in text for split_id in split_ids)

    release_note = mutable_migrated_files[0].read_text(encoding="utf-8")
    assert "originally recorded with MPD-0006" in release_note
    assert "SFP-0019's substantive identity separation" in release_note


def test_four_legacy_governance_registries_are_split_and_materialized() -> None:
    import yaml
    from developer.automation.registry_split_validator import validate_registry_split

    assert validate_registry_split(ROOT) == ()
    inventory = yaml.safe_load(
        (
            ROOT / "developer" / "policy" / "registry-split-inventory.yaml"
        ).read_text(encoding="utf-8")
    )
    assert inventory["classification"] == "SPLIT"
    assert inventory["source_registry_count"] == 4
    assert {item["id"] for item in inventory["source_registries"]} == {
        "AUTHORITY_SCHEMA_REGISTRY",
        "AUTHORITY_ROLE_REGISTRY",
        "AUTHORITY_SUBJECT_REGISTRY",
        "AUTHORIZATION_TRANSITION_REGISTRY",
    }


def test_support_registry_projection_contains_no_ptsip_repository_binding() -> None:
    import yaml

    subject = yaml.safe_load(
        (
            ROOT
            / "src"
            / "ptsip"
            / "specdata"
            / "ptsip-support-authority-subject-registry.yaml"
        ).read_text(encoding="utf-8")
    )
    assert "current_repository_bindings" not in subject
    assert set(subject["subject_identity_schemes"]) == {"SUPPORT_POLICY_ID"}


def test_owner_authorization_grants_remain_developer_policy_only() -> None:
    import yaml

    support = yaml.safe_load(
        (
            ROOT
            / "src"
            / "ptsip"
            / "specdata"
            / "ptsip-support-authorization-registry.yaml"
        ).read_text(encoding="utf-8")
    )
    developer = yaml.safe_load(
        (
            ROOT
            / "developer"
            / "policy"
            / "registries"
            / "authorization-transition-registry.yaml"
        ).read_text(encoding="utf-8")
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
    assert readiness == {
        "AUTHORITY_SCHEMA_REGISTRY_VALID": True,
        "AUTHORITY_ROLE_REGISTRY_VALID": True,
        "AUTHORITY_SUBJECT_REGISTRY_VALID": True,
        "CURRENT_SUPPORT_POLICY_CORPUS_VALID": True,
        "ROLE_EFFECT_VOCABULARY_VALID": True,
        "SUPPORT_POLICY_SUBJECT_CONTRACT_VALID": True,
        "PROJECT_AUTHORITY_RUNTIME_OWNER_PREAUTHORIZED": True,
    }
    results = evaluator.evaluate_current_project_authority_runtime()
    assert {item.state for item in results} == {AuthorizationState.AUTHORIZED}


def test_product_governance_runtime_has_zero_legacy_decisions_path_dependency() -> None:
    root = ROOT / "src" / "ptsip" / "governance"
    offenders = []
    for path in root.glob("*.py"):
        if "decisions/" in path.read_text(encoding="utf-8"):
            offenders.append(path.name)
    assert offenders == []
