from __future__ import annotations

import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ptsip.governance import AuthorityCatalog


ROOT = Path(__file__).resolve().parents[4]


def _json(path: str) -> dict[str, object]:
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def _yaml(path: str) -> dict[str, object]:
    value = yaml.safe_load((ROOT / path).read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def test_governance_runtime_schemas_are_valid_draft_2020_12() -> None:
    for path in (
        "src/policy/schemas/ptsip-support-authority-role.schema.json",
        "src/policy/schemas/ptsip-support-subject-binding.schema.json",
        "src/policy/schemas/ptsip-support-project-authority-record.schema.json",
        "src/policy/schemas/ptsip-support-authority-eligibility-result.schema.json",
        "src/policy/schemas/ptsip-support-authority-semantics.schema.json",
        "src/policy/schemas/ptsip-support-governance-registry.schema.json",
        "src/policy/schemas/ptsip-support-root-family-policy.schema.json",
    ):
        Draft202012Validator.check_schema(_json(path))


def test_current_governance_corpus_is_shipped_support_policy_corpus() -> None:
    catalog = AuthorityCatalog(ROOT)
    expected = tuple(
        entry["id"]
        for entry in _yaml("src/policy/index.yaml")["policies"]
        if entry.get("authority_role") != "MIGRATION_SOURCE"
    )
    assert catalog.validate_current_corpus() == expected


def test_support_registry_has_no_builtin_repository_binding_or_owner_grant() -> None:
    subject = _yaml("src/policy/registries/ptsip-support-authority-subject-registry.yaml")
    authorization = _yaml("src/policy/registries/ptsip-support-authorization-registry.yaml")
    assert "current_repository_bindings" not in subject
    assert subject["repository_binding_policy"] == "SOLVE_SUBJECT_PROVIDED_NO_BUILTIN_CURRENT_REPOSITORY"
    assert "authorization_provenance" not in authorization
    assert "rules" not in authorization


def test_support_authority_lifecycle_mapping_is_policy_status_based() -> None:
    registry = _yaml("src/policy/registries/ptsip-support-authority-schema-registry.yaml")
    assert registry["lifecycle_policy"]["project_authority_eligible_states"] == ["ACTIVE"]
    assert "decision_status_mapping" not in registry["lifecycle_policy"]


def test_subject_registry_keeps_relaxed_matching_machine_registered_only() -> None:
    matching = _yaml("src/policy/registries/ptsip-support-authority-subject-registry.yaml")["matching"]
    assert matching["order"] == ["EXACT", "REGISTERED_MACHINE_RELATIONSHIP", "NO_MATCH"]
    assert matching["registered_machine_relationships"] == []
    assert matching["relationship_must_be_registered_before_relaxed_match"] is True
    assert matching["fuzzy_match"] == "FORBIDDEN"
    assert matching["ai_semantic_match"] == "FORBIDDEN"


def test_support_policy_canonical_layout_has_no_legacy_specdata_authority() -> None:
    canonical = ROOT / "src" / "policy"
    assert (canonical / "index.yaml").is_file()
    index = _yaml("src/policy/index.yaml")
    discovered = sorted(
        path.relative_to(canonical).as_posix()
        for path in canonical.rglob("SFP-*.yaml")
    )
    assert discovered == sorted(item["path"] for item in index["policies"])
    assert (canonical / "schemas").is_dir()
    assert (canonical / "registries").is_dir()
    assert not (ROOT / "docs" / "Support_policy" / "policy").exists()
    assert not (ROOT / "src" / "ptsip" / "specdata" / "support-policy-index.yaml").exists()
    assert not list((ROOT / "src" / "ptsip" / "specdata").glob("SFP-*.yaml"))

def test_support_legacy_policy_catalog_preserves_history_and_retires_all_sources() -> None:
    index = _yaml("src/policy/index.yaml")
    registry = _json("src/policy/registries/root-family-migration.json")
    entries = {entry["id"]: entry for entry in index["policies"]}
    for source in registry["sources"]:
        entry = entries[source["source_policy_id"]]
        assert entry["authority_role"] == "MIGRATION_SOURCE"
        assert entry["status"] == "RETIRED"
    schema = _json("src/policy/schemas/ptsip-support-feature-policy-index.schema.json")
    Draft202012Validator(schema).validate(index)

    # Neither the historical ACTIVE nor DRAFT lifecycle is a current authority.
    for forbidden_status in ("ACTIVE", "DRAFT"):
        altered = json.loads(json.dumps(index))
        next(item for item in altered["policies"] if item["id"] == "SFP-0004")["status"] = forbidden_status
        assert not Draft202012Validator(schema).is_valid(altered)

def test_consumer_analysis_draft_records_only_approved_registry_and_id_direction() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0005.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    assert policy["responsibility_family"] == "INFO"
    approved = policy["authority_semantics"]["unit_consumer_analysis_approved_decisions"]
    assert approved["repository_root"] == ".ptsip/analysis/"
    assert approved["registry_ref"] == ".ptsip/analysis/registry.yaml"
    assert approved["schema_root"] == ".ptsip/analysis/schemas/"
    assert approved["record_root"] == ".ptsip/analysis/records/"
    assert approved["record_identity"] == "OPAQUE_STABLE_ID"
    assert approved["record_id_issuance_authority"] == "PTSIP_AUTOMATED"
    assert approved["record_serialization_format"] == "JSON_ONLY"
    assert approved["resolution"] == "EXACT_ID_TO_REGISTERED_RECORD_REF"
    invariants = policy["authority_semantics"]["unit_consumer_analysis_invariants"]
    assert invariants["record_identity"]["id_to_path_derivation"] == "FORBIDDEN"
    assert invariants["record_identity"]["issuer"] == "PTSIP"
    assert invariants["record_identity"]["independent_ai_id_issuance"] == "FORBIDDEN"
    assert invariants["record_serialization"]["record_payload_format"] == "JSON_ONLY"
    pending = policy["authority_semantics"]["unit_consumer_analysis_unresolved_decisions"]
    assert "RECORD_ID_PHYSICAL_FORMAT_AND_COLLISION_CHECK_CONTRACT" in pending
    assert approved["record_id_uniqueness_scope"] == "CONSUMER_REPOSITORY"
    assert "RECORD_SERIALIZATION_FORMAT" not in pending
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)


def test_consumer_analysis_cleanup_delegation_is_operation_specific_and_nonactivating() -> None:
    policy = _json("src/policy/CTRL/SFP-CTRL-0003.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    assert policy["responsibility_family"] == "CTRL"
    semantics = policy["authority_semantics"]
    approved = semantics["unit_consumer_analysis_approved_decisions"]
    invariants = semantics["unit_consumer_analysis_invariants"]
    assert approved["working_cleanup_execution_authority"] == "CONSUMER_PROJECT_DELEGATED_AUTOMATION"
    assert approved["cleanup_operation_kind"] == "CONSUMER_DELEGATED_PER_OPERATION_KIND"
    assert approved["physical_deletion_authority"] == "EXPLICIT_OPERATION_SPECIFIC_DELEGATION_REQUIRED"
    assert approved["general_cleanup_delegation_grants_physical_deletion"] is False
    assert approved["mandatory_record_retention"] == "PROTECTED"
    assert invariants["cleanup_requires_operation_kind_specific_delegation"] is True
    assert invariants["delegation_scope_must_cover_requested_operation_kind"] is True
    assert invariants["generic_cleanup_delegation_grants_physical_deletion"] is False
    assert invariants["physical_deletion_requires_explicit_kind_and_scope"] is True
    assert invariants["operation_delegation_cannot_override_record_retention"] is True
    assert invariants["unknown_or_undelegated_operation_kind"] == "FAIL_CLOSED"
    assert invariants["cleanup_execution_requires_fresh_valid_delegation"] is True
    assert "AUTHORIZED_CLEANUP_OPERATION_CLASSES" in semantics["unit_consumer_analysis_unresolved_decisions"]
    record = _json("src/policy/RECORD/SFP-RECORD-0004.yaml")
    assert record["authority_semantics"]["unit_consumer_analysis_approved_decisions"]["mandatory_record_retention"] == "PRESERVED_WITH_PRECEDENCE"
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)
