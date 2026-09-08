from __future__ import annotations

import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ptsip.governance import AuthorityCatalog


ROOT = Path(__file__).resolve().parents[3]


def _json(path: str) -> dict[str, object]:
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def _yaml(path: str) -> dict[str, object]:
    value = yaml.safe_load((ROOT / path).read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def test_governance_runtime_schemas_are_valid_draft_2020_12() -> None:
    for path in (
        "schemas/ptsip-adr-index.schema.json",
        "schemas/ptsip-adr.schema.json",
        "schemas/ptsip-governance-authority-registry.schema.json",
        "schemas/ptsip-governance-authority-role.schema.json",
        "schemas/ptsip-governance-authority-role-registry.schema.json",
        "schemas/ptsip-governance-authority-semantics.schema.json",
        "schemas/ptsip-governance-authority-subject-registry.schema.json",
        "schemas/ptsip-governance-subject-binding.schema.json",
        "schemas/ptsip-project-authority-record.schema.json",
        "schemas/ptsip-authority-eligibility-result.schema.json",
        "schemas/ptsip-authorization-transition.schema.json",
    ):
        Draft202012Validator.check_schema(_json(path))


def test_current_governance_corpus_is_canonical_and_machine_valid() -> None:
    catalog = AuthorityCatalog(ROOT)
    assert catalog.validate_current_corpus() == tuple(
        f"ADR-{index:04d}" for index in range(1, 24)
    )


def test_frozen_registry_provenance_uses_immutable_git_commits_not_planning_files() -> None:
    role = _yaml("decisions/AUTHORITY-ROLE-REGISTRY.yaml")
    subject = _yaml("decisions/AUTHORITY-SUBJECT-REGISTRY.yaml")
    assert role["frozen_by"] == {
        "type": "GIT_COMMIT",
        "repository": "Kinirin/PTSIP",
        "revision": "54a4349144affc62135a2716017ca1f943cb555e",
    }
    assert subject["frozen_by"] == {
        "type": "GIT_COMMIT",
        "repository": "Kinirin/PTSIP",
        "revision": "584c3d3d59b2b9028c545cd700f4eb7c0c118f7b",
    }
    assert "planning/" not in yaml.safe_dump(role)
    assert "planning/" not in yaml.safe_dump(subject)


def test_authority_lifecycle_mapping_is_closed_and_separate_from_index_selection() -> None:
    registry = _yaml("decisions/AUTHORITY-SCHEMA-REGISTRY.yaml")
    assert registry["lifecycle_policy"] == {
        "decision_status_mapping": {
            "ACCEPTED": "ACTIVE",
            "ACCEPTED_FOR_DRAFT_SPECIFICATION": "ACTIVE",
            "PROPOSED_FOR_NEXT_NORMATIVE_SNAPSHOT": "DRAFT",
        },
        "project_authority_eligible_states": ["ACTIVE"],
    }


def test_subject_registry_keeps_relaxed_matching_machine_registered_only() -> None:
    matching = _yaml("decisions/AUTHORITY-SUBJECT-REGISTRY.yaml")["matching"]
    assert matching["order"] == ["EXACT", "REGISTERED_MACHINE_RELATIONSHIP", "NO_MATCH"]
    assert matching["registered_machine_relationships"] == []
    assert matching["relationship_must_be_registered_before_relaxed_match"] is True
    assert matching["fuzzy_match"] == "FORBIDDEN"
    assert matching["ai_semantic_match"] == "FORBIDDEN"
