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
        "schemas/ptsip-support-authority-role.schema.json",
        "schemas/ptsip-support-subject-binding.schema.json",
        "schemas/ptsip-support-project-authority-record.schema.json",
        "schemas/ptsip-support-authority-eligibility-result.schema.json",
        "schemas/ptsip-support-authority-semantics.schema.json",
        "schemas/ptsip-support-governance-registry.schema.json",
    ):
        Draft202012Validator.check_schema(_json(path))


def test_current_governance_corpus_is_shipped_support_policy_corpus() -> None:
    catalog = AuthorityCatalog(ROOT)
    assert catalog.validate_current_corpus() == tuple(
        f"SFP-{index:04d}" for index in range(1, 22)
    )


def test_support_registry_has_no_builtin_repository_binding_or_owner_grant() -> None:
    subject = _yaml("src/ptsip/specdata/ptsip-support-authority-subject-registry.yaml")
    authorization = _yaml("src/ptsip/specdata/ptsip-support-authorization-registry.yaml")
    assert "current_repository_bindings" not in subject
    assert subject["repository_binding_policy"] == "SOLVE_SUBJECT_PROVIDED_NO_BUILTIN_CURRENT_REPOSITORY"
    assert "authorization_provenance" not in authorization
    assert "rules" not in authorization


def test_support_authority_lifecycle_mapping_is_policy_status_based() -> None:
    registry = _yaml("src/ptsip/specdata/ptsip-support-authority-schema-registry.yaml")
    assert registry["lifecycle_policy"]["project_authority_eligible_states"] == ["ACTIVE"]
    assert "decision_status_mapping" not in registry["lifecycle_policy"]


def test_subject_registry_keeps_relaxed_matching_machine_registered_only() -> None:
    matching = _yaml("src/ptsip/specdata/ptsip-support-authority-subject-registry.yaml")["matching"]
    assert matching["order"] == ["EXACT", "REGISTERED_MACHINE_RELATIONSHIP", "NO_MATCH"]
    assert matching["registered_machine_relationships"] == []
    assert matching["relationship_must_be_registered_before_relaxed_match"] is True
    assert matching["fuzzy_match"] == "FORBIDDEN"
    assert matching["ai_semantic_match"] == "FORBIDDEN"
