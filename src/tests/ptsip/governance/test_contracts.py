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


def test_consumer_analysis_context_evidence_references_and_cleanup_boundaries() -> None:
    info = _yaml("src/policy/INFO/SFP-INFO-0005.yaml")
    decisions = info["authority_semantics"]["unit_consumer_analysis_approved_decisions"]
    assert decisions["context_analysis_reference_granularity"] == "INDIVIDUAL_OBSERVATION_AND_EVIDENCE"
    assert decisions["context_first_analysis_discovery"] == "AUTOMATIC_DERIVED_REVERSE_INDEX"
    assert info["authority_semantics"]["unit_consumer_analysis_invariants"]["context_analysis_references"]["whole_context_snapshot_as_substitute"] == "FORBIDDEN"
    cntr = _json("src/policy/CNTR/SFP-CNTR-0006.yaml")
    assert cntr["authority_semantics"]["unit_consumer_analysis_invariants"]["reverse_index_is_non_authoritative_projection"] is True
    record = _json("src/policy/RECORD/SFP-RECORD-0004.yaml")
    assert record["authority_semantics"]["unit_consumer_analysis_approved_decisions"]["reusable_long_term_retention_scope"] == "CONSUMER_DELEGATED_LONG_TERM_RETENTION"
    ctrl = _json("src/policy/CTRL/SFP-CTRL-0003.yaml")
    assert ctrl["authority_semantics"]["unit_consumer_analysis_approved_decisions"]["cleanup_completion_consistency"] == "CONTRACT_PROVEN_LOGICAL_COMPLETION"
    assert ctrl["authority_semantics"]["unit_consumer_analysis_invariants"]["cleanup_execution_precondition_recheck_required"] is True
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    for policy in (info, cntr, record, ctrl):
        Draft202012Validator(schema).validate(policy)


def test_context_observation_evidence_identity_binding_draft() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    assert policy["responsibility_family"] == "INFO"
    assert policy["policy"]["id"] == "SFP-INFO-0007"
    decisions = policy["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert decisions["policy_direction"] == "PTSIP_COMMON_ID_WITH_VERIFIED_PROVIDER_ID_BINDING"
    assert decisions["common_reference_identity_issuer"] == "PTSIP_AUTOMATED"
    assert decisions["analysis_reference_identity"] == "PTSIP_COMMON_CONTEXT_EVIDENCE_ID"
    assert decisions["analysis_reference_owner"] == "ANALYSIS"
    assert decisions["analysis_discovery_direction"] == "CONTEXT_TO_ANALYSIS_DERIVED_REVERSE_INDEX"
    invariants = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]
    assert invariants["registration"]["unverified_or_ambiguous_binding"] == "FAIL_CLOSED"
    assert invariants["registration"]["provider_identity_collision_may_not_merge_observations"] is True
    assert invariants["authority"]["provider_id_alone_confers_ptsip_reference_authority"] is False
    assert invariants["authority"]["record_owns_source_bound_provenance_and_history"] is True
    assert invariants["reference"]["reverse_lookup_is_non_authoritative_projection"] is True
    assert invariants["ownership"]["independent_ai_identity_creation"] == "FORBIDDEN"
    assert invariants["ownership"]["existing_record_retention_overridden"] is False
    assert "PROVIDER_TYPE_EVENT_IDENTITY_PROOF_INTERFACE" in policy["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]
    index = _yaml("src/policy/index.yaml")
    assert any(
        item["id"] == "SFP-INFO-0007"
        and item["path"] == "INFO/SFP-INFO-0007.yaml"
        and item["status"] == "DRAFT"
        for item in index["policies"]
    )
    analysis = _yaml("src/policy/INFO/SFP-INFO-0005.yaml")
    assert analysis["authority_semantics"]["unit_consumer_analysis_policy_family_refs"]["INFO_CONTEXT_EVIDENCE_IDENTITY"] == "SFP-INFO-0007"
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)


def test_context_reobservation_reuses_id_only_for_proven_same_event() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    approved = policy["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert approved["reobservation_identity_policy"] == "CONTRACT_VERIFIED_SAME_EVENT_ID_REUSE"
    assert approved["verified_same_event"] == "REUSE_EXISTING_COMMON_ID"
    assert approved["verified_distinct_event"] == "ISSUE_NEW_COMMON_ID"
    assert approved["unresolved_event_identity"] == "NO_AUTOMATIC_MERGE_OR_REUSE"

    invariants = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]
    assert invariants["registration"]["identical_payload_alone_proves_same_event"] is False
    assert invariants["registration"]["identical_provider_id_alone_proves_same_event"] is False
    event = invariants["reobservation"]
    assert event["event_identity_requires_registered_machine_verifiable_proof"] is True
    assert event["same_event_proven"] == "REUSE_EXISTING_COMMON_ID"
    assert event["distinct_event_proven"] == "ISSUE_NEW_COMMON_ID"
    assert event["identity_unresolved"] == "FAIL_CLOSED_NO_MERGE_OR_REUSE"
    assert event["provider_identity_only_as_equivalence_proof"] == "FORBIDDEN"
    assert event["payload_similarity_as_equivalence_proof"] == "FORBIDDEN"
    assert event["previous_common_identity_must_remain_stable"] is True
    assert event["reobservation_must_not_rewrite_original_source_receipts"] is True
    assert event["subsequent_collection_provenance_must_remain_traceable"] is True

    assert approved["analysis_reference_owner"] == "ANALYSIS"
    assert approved["analysis_discovery_direction"] == "CONTEXT_TO_ANALYSIS_DERIVED_REVERSE_INDEX"
    assert invariants["ownership"]["existing_record_retention_overridden"] is False
    assert "PROVIDER_TYPE_EVENT_IDENTITY_PROOF_INTERFACE" in policy["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]

    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)


def test_context_event_identity_proof_common_plus_provider_type_extension() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    decisions = policy["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert decisions["event_identity_proof_authority"] == "PTSIP_COMMON_PROOF_WITH_PROVIDER_TYPE_EXTENSION"
    assert decisions["common_proof_obligations_owner"] == "PTSIP"
    assert decisions["provider_type_proof_extension"] == "REGISTERED_MACHINE_VERIFIABLE_CONTRACT"
    assert decisions["duplicate_event_manual_mapping"] == "NOT_REQUIRED"
    proof = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]["proof_authority"]
    assert proof["ptsip_common_proof_obligations_required"] is True
    assert proof["common_proof_obligations_cannot_be_weakened_by_provider_type"] is True
    assert proof["provider_type_specific_rules_must_be_registered"] is True
    assert proof["provider_type_specific_proof_must_be_machine_verifiable"] is True
    assert proof["provider_type_extension_alone_confers_identity_authority"] is False
    assert proof["source_and_observation_context_binding_required"] is True
    assert proof["unregistered_or_ambiguous_proof_contract"] == "FAIL_CLOSED"
    assert proof["proof_scope_or_source_revision_unverified"] == "FAIL_CLOSED"
    assert proof["ai_ad_hoc_event_equivalence_inference"] == "FORBIDDEN"
    assert proof["manual_duplicate_event_linkage_required"] is False
    assert decisions["verified_same_event"] == "REUSE_EXISTING_COMMON_ID"
    assert decisions["verified_distinct_event"] == "ISSUE_NEW_COMMON_ID"
    assert decisions["analysis_reference_owner"] == "ANALYSIS"
    assert decisions["analysis_discovery_direction"] == "CONTEXT_TO_ANALYSIS_DERIVED_REVERSE_INDEX"
    assert policy["authority_semantics"]["unit_context_evidence_identity_invariants"]["ownership"]["existing_record_retention_overridden"] is False
    unresolved = policy["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]
    assert "PROVIDER_TYPE_EVENT_IDENTITY_PROOF_INTERFACE" in unresolved
    assert "REOBSERVATION_EVENT_IDENTITY_PROOF_AUTHORITY_AND_SCOPE" not in unresolved
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)

def test_context_missing_provider_event_id_uses_verified_source_observation() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    approved = policy["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert approved["missing_provider_event_id_registration"] == "CONTRACT_VERIFIED_SOURCE_AND_INDIVIDUAL_OBSERVATION"
    assert approved["provider_event_id_requirement"] == "OPTIONAL_WITH_VERIFIED_ALTERNATIVE"
    assert approved["missing_provider_event_id_reuse"] == "SEPARATE_SAME_EVENT_PROOF_REQUIRED"

    invariants = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]
    assert invariants["registration"]["missing_provider_event_id_alone_forbids_registration"] is False
    assert invariants["registration"]["missing_event_id_cannot_waive_provider_and_source_verification"] is True
    alternative = invariants["missing_provider_event_id"]
    assert alternative["provider_identity_and_source_must_remain_verifiable"] is True
    assert alternative["individual_observation_must_be_machine_verified"] is True
    assert alternative["registered_source_and_observation_contract_required"] is True
    assert alternative["verified_alternative_may_authorize_new_common_id"] is True
    assert alternative["provider_event_id_may_not_be_fabricated"] is True
    assert alternative["common_id_issuance_does_not_prove_event_equivalence"] is True
    assert alternative["same_event_id_reuse_requires_separate_contract_proof"] is True
    assert alternative["ambiguous_observation_or_source"] == "FAIL_CLOSED"
    assert alternative["unproven_identity_collision"] == "FAIL_CLOSED_NO_MERGE_OR_REUSE"
    assert alternative["mandatory_record_provenance_unaffected"] is True

    assert approved["verified_same_event"] == "REUSE_EXISTING_COMMON_ID"
    assert approved["verified_distinct_event"] == "ISSUE_NEW_COMMON_ID"
    assert approved["analysis_reference_owner"] == "ANALYSIS"
    assert approved["analysis_discovery_direction"] == "CONTEXT_TO_ANALYSIS_DERIVED_REVERSE_INDEX"
    assert invariants["ownership"]["existing_record_retention_overridden"] is False
    unresolved = policy["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]
    assert "MISSING_PROVIDER_EVENT_ID_SOURCE_OBSERVATION_PROOF_INTERFACE" in unresolved
    assert "PROVIDER_ID_BINDING_INTERFACE_AND_MISSING_PROVIDER_ID_HANDLING" not in unresolved
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)


def test_context_evidence_ids_are_consumer_repository_scoped() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    decisions = policy["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert decisions["common_reference_identity_uniqueness_scope"] == "CONSUMER_REPOSITORY"
    assert decisions["common_reference_identity_independence"] == "REPOSITORY_LOCAL_AUTOMATED_ISSUANCE"
    assert decisions["cross_repository_identity_interpretation"] == "EXPLICIT_REPOSITORY_SCOPE_REQUIRED"
    invariants = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]
    scope = invariants["identity_uniqueness"]
    assert scope["consumer_repository_scoped_uniqueness_required"] is True
    assert scope["no_global_issuer_or_central_id_registry_required"] is True
    assert scope["independent_ptsip_issuance_required"] is True
    assert scope["newly_issued_id_must_not_collide_with_registered_local_identity"] is True
    assert scope["ambiguous_or_colliding_local_identity"] == "FAIL_CLOSED"
    assert scope["unqualified_id_does_not_identify_cross_repository_event"] is True
    assert scope["foreign_repository_id_string_equality_proves_same_event"] is False
    assert scope["cross_repository_reference_requires_explicit_verified_repository_scope"] is True
    assert scope["repository_fork_semantics"] == "CONTRACT_VERIFIED_LINEAGE"
    assert scope["repository_relocation_semantics"] == "RESET_IDENTITY_SCOPE"
    assert decisions["reobservation_identity_policy"] == "CONTRACT_VERIFIED_SAME_EVENT_ID_REUSE"
    assert decisions["analysis_reference_owner"] == "ANALYSIS"
    assert decisions["analysis_discovery_direction"] == "CONTEXT_TO_ANALYSIS_DERIVED_REVERSE_INDEX"
    assert decisions["mandatory_record_provenance"] == "PRESERVED"
    assert invariants["ownership"]["existing_record_retention_overridden"] is False
    unresolved = policy["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]
    assert "CONTEXT_COMMON_ID_PHYSICAL_FORMAT_AND_LOCAL_COLLISION_CHECK_CONTRACT" in unresolved
    assert "CONTEXT_FORK_LINEAGE_PROOF_INTERFACE" in unresolved
    assert "CONTEXT_REPOSITORY_NAME_LOCATION_CHANGE_SCOPE_RESET_INTERFACE" in unresolved
    assert "CONTEXT_COMMON_ID_FORMAT_NAMESPACE_AND_ALLOCATION_CONTRACT" not in unresolved
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)


def test_verified_fork_lineage_preserves_identity_and_requires_current_reuse_validity() -> None:
    info = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert info["policy"]["status"] == "DRAFT"
    approved = info["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert approved["repository_fork_lineage_policy"] == "CONTRACT_VERIFIED_FORK_LINEAGE"
    assert approved["fork_lineage_reuse_role"] == "ELIGIBILITY_INPUT_NOT_REUSE_GRANT"
    assert approved["pre_fork_source_identity"] == "ORIGINAL_REPOSITORY_SCOPE_AND_ID_PRESERVED"
    assert approved["post_fork_observation_identity"] == "FORK_LOCAL_INDEPENDENT_ID"
    fork = info["authority_semantics"]["unit_context_evidence_identity_invariants"]["fork_lineage"]
    assert fork["distinct_consumer_repositories_keep_independent_identity_authority"] is True
    assert fork["verified_registered_origin_relationship_required"] is True
    assert fork["provenance_of_inherited_observations_preserved"] is True
    assert fork["copied_pre_fork_observation_ids_keep_origin_repository_scope"] is True
    assert fork["newly_observed_fork_events_use_fork_repository_scoped_ids"] is True
    assert fork["lineage_alone_does_not_prove_same_event"] is True
    assert fork["lineage_alone_does_not_authorize_reuse"] is True
    assert fork["current_dependency_and_change_impact_validation_required"] is True
    assert fork["unverified_or_ambiguous_lineage"] == "FAIL_CLOSED"
    assert fork["ai_inferred_or_fuzzy_lineage"] == "FORBIDDEN"
    assert fork["manual_per_record_lineage_mapping_required"] is False

    assure = _json("src/policy/ASSURE/SFP-ASSURE-0005.yaml")
    assert assure["policy"]["status"] == "DRAFT"
    decisions = assure["authority_semantics"]["unit_consumer_analysis_approved_decisions"]
    assert decisions["fork_lineage_reuse"] == "VERIFIED_LINEAGE_PLUS_CURRENT_DEPENDENCY_AND_EVIDENCE_VALIDITY"
    ai = assure["authority_semantics"]["unit_consumer_analysis_invariants"]
    assert ai["verified_fork_lineage_may_provide_reuse_candidate"] is True
    assert ai["lineage_alone_cannot_prove_current_reuse_validity"] is True
    assert ai["lineage_alone_cannot_prove_observation_event_equivalence"] is True
    assert ai["fork_change_impact_must_be_machine_verified"] is True
    assert ai["unverified_fork_lineage_or_current_validity"] == "FAIL_CLOSED"
    assert ai["fork_lineage_cannot_override_record_provenance_retention"] is True

    record = _yaml("src/policy/RECORD/SFP-RECORD-0001.yaml")
    assert record["policy"]["status"] == "ACTIVE"
    analysis = _yaml("src/policy/INFO/SFP-INFO-0005.yaml")
    assert analysis["authority_semantics"]["unit_consumer_analysis_approved_decisions"]["context_first_analysis_discovery"] == "AUTOMATIC_DERIVED_REVERSE_INDEX"
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    for policy in (info, assure):
        Draft202012Validator(schema).validate(policy)


def test_repository_relocation_resets_context_identity_scope_without_history_rewrite() -> None:
    policy = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert policy["policy"]["status"] == "DRAFT"
    approved = policy["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert approved["repository_name_location_change_policy"] == "RESET_IDENTITY_SCOPE"
    assert approved["relocation_historical_identity"] == "PRESERVE_ORIGINAL_REPOSITORY_SCOPE_AND_ID"
    assert approved["relocation_new_observation_identity"] == "NEW_REPOSITORY_SCOPE_LOCAL_ID"
    assert approved["relocation_id_continuity"] == "NO_AUTOMATIC_SCOPE_INHERITANCE"
    scope = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]["identity_uniqueness"]
    assert scope["repository_fork_semantics"] == "CONTRACT_VERIFIED_LINEAGE"
    assert scope["repository_relocation_semantics"] == "RESET_IDENTITY_SCOPE"
    relocation = policy["authority_semantics"]["unit_context_evidence_identity_invariants"]["repository_relocation"]
    assert relocation["name_or_location_change_requires_identity_scope_reset"] is True
    assert relocation["prior_observation_ids_keep_original_repository_scope"] is True
    assert relocation["existing_historical_source_receipts_must_be_preserved"] is True
    assert relocation["historical_identity_or_provenance_rewrite"] == "FORBIDDEN"
    assert relocation["prior_ids_are_not_automatically_rebound_to_new_scope"] is True
    assert relocation["new_observations_use_new_repository_scoped_identity"] is True
    assert relocation["location_change_alone_grants_same_event_identity"] is False
    assert relocation["location_change_alone_grants_analysis_reuse"] is False
    assert relocation["registered_verified_cross_scope_reference_required"] is True
    assert relocation["unverified_cross_scope_reference"] == "FAIL_CLOSED"
    assert relocation["fork_lineage_rules_remain_independent"] is True
    assert approved["analysis_reference_owner"] == "ANALYSIS"
    assert approved["analysis_discovery_direction"] == "CONTEXT_TO_ANALYSIS_DERIVED_REVERSE_INDEX"
    assert approved["reobservation_identity_policy"] == "CONTRACT_VERIFIED_SAME_EVENT_ID_REUSE"
    assert policy["authority_semantics"]["unit_context_evidence_identity_invariants"]["ownership"]["existing_record_retention_overridden"] is False
    record = _yaml("src/policy/RECORD/SFP-RECORD-0001.yaml")
    assert record["policy"]["status"] == "ACTIVE"
    unresolved = policy["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]
    assert "CONTEXT_REPOSITORY_NAME_LOCATION_CHANGE_SCOPE_RESET_INTERFACE" in unresolved
    assert "CONTEXT_REPOSITORY_RELOCATION_IDENTITY_CONTINUITY" not in unresolved
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    Draft202012Validator(schema).validate(policy)


def test_relocation_historical_analysis_candidates_require_proven_scope_and_current_validity() -> None:
    info = _yaml("src/policy/INFO/SFP-INFO-0007.yaml")
    assert info["policy"]["status"] == "DRAFT"
    decisions = info["authority_semantics"]["unit_context_evidence_identity_approved_decisions"]
    assert decisions["repository_name_location_change_policy"] == "RESET_IDENTITY_SCOPE"
    assert decisions["relocation_historical_analysis_access"] == "VERIFIED_CROSS_SCOPE_RELATION_CANDIDATE_ONLY"
    assert decisions["relocation_historical_reference_authority"] == "REGISTERED_VERIFIED_SCOPE_RELATION"
    scope = info["authority_semantics"]["unit_context_evidence_identity_invariants"]["relocation_historical_analysis"]
    assert scope["original_repository_scope_and_analysis_ids_preserved"] is True
    assert scope["candidate_discovery_requires_registered_verified_scope_relation"] is True
    assert scope["cross_scope_relation_does_not_rebind_original_ids"] is True
    assert scope["relocation_alone_does_not_create_candidate_authority"] is True
    assert scope["candidate_discovery_does_not_authorize_reuse"] is True
    assert scope["historical_provenance_and_receipts_preserved"] is True
    assert scope["unverified_or_ambiguous_scope_relation"] == "FAIL_CLOSED"
    assert scope["ai_inferred_scope_relationship"] == "FORBIDDEN"
    assert "RELOCATION_CROSS_SCOPE_RELATION_PROOF_INTERFACE" in info["authority_semantics"]["unit_context_evidence_identity_unresolved_decisions"]

    cntr = _json("src/policy/CNTR/SFP-CNTR-0006.yaml")
    assert cntr["policy"]["status"] == "DRAFT"
    selected = cntr["authority_semantics"]["unit_consumer_analysis_approved_decisions"]
    assert selected["context_first_lookup"] == "AUTOMATIC_REGISTERED_EVIDENCE_REVERSE_INDEX"
    assert selected["reference_source_authority"] == "ANALYSIS_OWNED"
    assert selected["relocation_historical_candidate_discovery"] == "REGISTERED_VERIFIED_CROSS_SCOPE_RELATION_EXPANSION"
    lookup = cntr["authority_semantics"]["unit_consumer_analysis_invariants"]
    assert lookup["relocation_cross_scope_expansion_requires_verified_relation"] is True
    assert lookup["relocation_candidates_must_respect_original_repository_scoped_identities"] is True
    assert lookup["relocation_cross_scope_lookup_is_candidate_discovery_only"] is True
    assert lookup["relocation_without_proof_not_candidate"] is True
    assert lookup["relocation_candidate_lookup_requires_complete_registered_closure"] is True
    assert lookup["relocation_cross_scope_unverified_or_ambiguous"] == "FAIL_CLOSED"
    assert lookup["relocation_cross_scope_ai_candidate_inference"] == "FORBIDDEN"

    assure = _json("src/policy/ASSURE/SFP-ASSURE-0005.yaml")
    assert assure["policy"]["status"] == "DRAFT"
    approved = assure["authority_semantics"]["unit_consumer_analysis_approved_decisions"]
    assert approved["relocation_historical_analysis_reuse"] == "VERIFIED_SCOPE_LINK_FOR_CANDIDACY_WITH_CURRENT_ASSURE_VALIDITY_GATE"
    validity = assure["authority_semantics"]["unit_consumer_analysis_invariants"]
    assert validity["relocation_verified_scope_link_alone_does_not_grant_reuse"] is True
    assert validity["relocation_analysis_must_pass_current_evidence_dependency_change_impact_validation"] is True
    assert validity["relocation_scope_link_does_not_prove_same_observation_event"] is True
    assert validity["relocation_reuse_cannot_promote_historical_analysis_authority"] is True
    assert validity["relocation_invalid_or_incomplete_current_validity"] == "FAIL_CLOSED"
    assert validity["relocation_reuse_preserves_original_record_history"] is True

    record = _yaml("src/policy/RECORD/SFP-RECORD-0001.yaml")
    assert record["policy"]["status"] == "ACTIVE"
    schema = _json("src/policy/schemas/ptsip-support-root-family-policy.schema.json")
    for policy in (info, cntr, assure):
        Draft202012Validator(schema).validate(policy)
