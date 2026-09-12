from __future__ import annotations

import pytest

from ptsip.evidence.contract import stable_digest
from ptsip.governance import (
    EligibilityResult,
    EligibilityStatus,
    ProjectAuthorityRecord,
    SubjectMatchKind,
)
from ptsip.remediation.solution import recovery as recovery_module
from ptsip.remediation.solution.recovery import (
    AuthorityCompatibilityCatalog,
    AuthorityCompatibilityMapping,
    AuthorityRequirementEligibilityBinding,
    CapabilityRecoveryAttemptStatus,
    CapabilityRecoveryContractError,
    CapabilityRecoveryPath,
    ProjectLocalAuthorityCompatibilityAdapter,
    RecoveredCapability,
    authority_requirement_view,
    recover_on_next_path,
    reject_next_recovery_path,
    start_capability_recovery,
)


def _subject_binding(policy_id: str = "SFP-0007") -> dict[str, object]:
    return {
        "authority_domain": "PROJECT_GOVERNANCE",
        "repository_binding": {
            "scheme": "GITHUB_REPOSITORY_ID",
            "host": "github.com",
            "repository_id": "1327447827",
        },
        "subject_type": "GOVERNANCE_TOPIC",
        "subject_identity": {
            "scheme": "SUPPORT_POLICY_ID",
            "value": policy_id,
        },
    }


def _requirement(
    *,
    requirement_id: str = "authority-requirement:classification",
    effects: tuple[str, ...] = ("govern_classification_model",),
) -> dict[str, object]:
    return {
        "requirement_id": requirement_id,
        "subject_binding": _subject_binding(),
        "required_effects": list(effects),
    }


def _authority(
    authority_id: str = "SFP-0007",
    *,
    effects: tuple[str, ...] = (
        "govern_classification_model",
        "require_project_owner_decision",
    ),
    semantics: dict[str, object] | None = None,
) -> ProjectAuthorityRecord:
    return ProjectAuthorityRecord(
        authority_id=authority_id,
        authority_contract={
            "authority_type": "PRIMARY_LIFECYCLE_AUTHORITY",
            "schema_id": "ptsip.governance/primary-lifecycle-authority",
            "schema_version": 1,
        },
        authority_semantics=semantics or {
            "classification_model": "CANONICAL_LIFECYCLE",
        },
        authority_role={
            "projection_role": "PROJECT_ARCHITECTURE_AUTHORITY",
            "resolution_state": "DECLARED",
            "effects": list(effects),
        },
        authority_provenance={
            "source_type": "SUPPORT_FEATURE_POLICY",
            "source_ref": f"src/ptsip/specdata/{authority_id}.yaml",
            "source_revision": "1" * 40,
            "source_digest": "2" * 64,
        },
        subject_binding=_subject_binding(authority_id),
    )


def _eligibility(
    authority: ProjectAuthorityRecord,
    *,
    status: EligibilityStatus = EligibilityStatus.CURRENTLY_ELIGIBLE,
    subject_match: SubjectMatchKind = SubjectMatchKind.EXACT,
) -> EligibilityResult:
    return EligibilityResult(
        authority_id=authority.authority_id,
        source_ref=str(authority.authority_provenance["source_ref"]),
        status=status,
        checks=(),
        subject_match=subject_match,
    )


def _authority_input(
    authority: ProjectAuthorityRecord,
    requirement: dict[str, object],
    *,
    eligibility: EligibilityResult | None = None,
    evaluated_requirement: dict[str, object] | None = None,
) -> AuthorityRequirementEligibilityBinding:
    evaluated = authority_requirement_view(evaluated_requirement or requirement)
    return AuthorityRequirementEligibilityBinding(
        authority=authority,
        eligibility=eligibility or _eligibility(authority),
        evaluated_subject_binding_digest=evaluated.subject_binding_digest,
    )


def _mapping(
    authority: ProjectAuthorityRecord,
    requirement: dict[str, object],
    *,
    mapping_id: str = "authority-compatibility:classification",
) -> AuthorityCompatibilityMapping:
    view = authority_requirement_view(requirement)
    return AuthorityCompatibilityMapping(
        id=mapping_id,
        requirement_id=view.requirement_id,
        requirement_subject_binding_digest=view.subject_binding_digest,
        required_effects=view.required_effects,
        authority_id=authority.authority_id,
        authority_semantics_digest=stable_digest(dict(authority.authority_semantics)),
        canonical_key="component.classification",
        canonical_value="DEVELOPMENT_TOOLING",
        closed_vocabulary=(
            "PRODUCT",
            "DEVELOPMENT_TOOLING",
            "DELIVERY",
            "OPERATIONS",
            "NEUTRAL_CONTRACT",
        ),
    )


def _after_normalization_rejection(requirement: dict[str, object]):
    return reject_next_recovery_path(
        start_capability_recovery(
            required_capability="component.classification",
            subject="component:runtime-a",
            authority_requirement=requirement,
        ),
        reason_code="NORMALIZATION_NOT_LOSSLESS",
        reason="Normalization would require semantic interpretation.",
    )


def test_recovery_paths_are_checked_in_required_order() -> None:
    assessment = start_capability_recovery(
        required_capability="component.classification",
        subject="component:runtime-a",
    )
    assert assessment.next_path is CapabilityRecoveryPath.LOSSLESS_CANONICAL_NORMALIZATION

    assessment = reject_next_recovery_path(
        assessment,
        reason_code="NORMALIZATION_NOT_LOSSLESS",
        reason="Normalization would require semantic interpretation.",
    )
    assert assessment.next_path is CapabilityRecoveryPath.PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER

    assessment = reject_next_recovery_path(
        assessment,
        reason_code="NO_EXPLICIT_ADAPTER",
        reason="No exact compatibility mapping exists.",
    )
    assert assessment.next_path is CapabilityRecoveryPath.EXISTING_AUTHORITY_COMPATIBLE_MATERIALIZATION

    assessment = reject_next_recovery_path(
        assessment,
        reason_code="NO_COMPATIBLE_MATERIALIZATION",
        reason="No existing compatible materialization exists.",
    )
    assert assessment.exhausted is True
    assert [item.status for item in assessment.attempts] == [
        CapabilityRecoveryAttemptStatus.REJECTED,
        CapabilityRecoveryAttemptStatus.REJECTED,
        CapabilityRecoveryAttemptStatus.REJECTED,
    ]


def test_recovery_cannot_skip_required_order() -> None:
    requirement = _requirement()
    authority = _authority()
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        adapter.adapt(
            start_capability_recovery(
                required_capability="component.classification",
                subject="component:runtime-a",
                authority_requirement=requirement,
            ),
            authority_requirement=requirement,
            authority_inputs=(_authority_input(authority, requirement),),
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_ORDER_INVALID"


def test_exact_eligible_project_authority_recovers_existing_semantics() -> None:
    requirement = _requirement()
    authority = _authority()
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    result = adapter.adapt(
        _after_normalization_rejection(requirement),
        authority_requirement=requirement,
        authority_inputs=(_authority_input(authority, requirement),),
    )

    assert result.recovered is True
    recovered = result.recovered_capability
    assert recovered is not None
    assert recovered.authority_requirement_id == requirement["requirement_id"]
    assert recovered.consumed_project_authority_ids == ("SFP-0007",)
    assert recovered.canonical_value == "DEVELOPMENT_TOOLING"


def test_available_but_unused_authority_is_not_recorded_as_consumed() -> None:
    requirement = _requirement()
    used = _authority("SFP-0007")
    unused = _authority(
        "SFP-0008",
        effects=("govern_relationship_contract",),
    )
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(used, requirement),),
        )
    )

    result = adapter.adapt(
        _after_normalization_rejection(requirement),
        authority_requirement=requirement,
        authority_inputs=(
            _authority_input(unused, requirement),
            _authority_input(used, requirement),
        ),
    )

    assert result.recovered_capability is not None
    assert result.recovered_capability.consumed_project_authority_ids == ("SFP-0007",)


def test_ineligible_or_effect_incomplete_authority_does_not_satisfy_requirement() -> None:
    requirement = _requirement()
    authority = _authority(effects=("require_project_owner_decision",))
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    result = adapter.adapt(
        _after_normalization_rejection(requirement),
        authority_requirement=requirement,
        authority_inputs=(_authority_input(authority, requirement),),
    )

    assert result.recovered is False
    assert result.attempts[-1].reason_code == (
        "NO_EXPLICIT_ELIGIBLE_AUTHORITY_COMPATIBILITY_MAPPING"
    )


def test_subject_non_applicable_authority_does_not_satisfy_requirement() -> None:
    requirement = _requirement()
    authority = _authority()
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    result = adapter.adapt(
        _after_normalization_rejection(requirement),
        authority_requirement=requirement,
        authority_inputs=(
            _authority_input(
                authority,
                requirement,
                eligibility=_eligibility(
                    authority,
                    status=EligibilityStatus.CURRENTLY_INELIGIBLE,
                    subject_match=SubjectMatchKind.NO_MATCH,
                ),
            ),
        ),
    )

    assert result.recovered is False


def test_eligibility_must_be_bound_to_exact_requirement_subject() -> None:
    requirement = _requirement()
    other_requirement = {
        **requirement,
        "subject_binding": _subject_binding("SFP-0008"),
    }
    authority = _authority()
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        adapter.adapt(
            _after_normalization_rejection(requirement),
            authority_requirement=requirement,
            authority_inputs=(
                _authority_input(
                    authority,
                    requirement,
                    evaluated_requirement=other_requirement,
                ),
            ),
        )

    assert exc_info.value.code == (
        "CAPABILITY_RECOVERY_ELIGIBILITY_SUBJECT_BINDING_MISMATCH"
    )


def test_authority_semantics_change_does_not_reuse_old_mapping() -> None:
    requirement = _requirement()
    mapped = _authority()
    changed = _authority(
        semantics={"classification_model": "DIFFERENT_MACHINE_SEMANTICS"},
    )
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(mapped, requirement),),
        )
    )

    result = adapter.adapt(
        _after_normalization_rejection(requirement),
        authority_requirement=requirement,
        authority_inputs=(_authority_input(changed, requirement),),
    )

    assert result.recovered is False


def test_multiple_recoverable_authorities_fail_closed_as_ambiguous() -> None:
    requirement = _requirement()
    first = _authority("SFP-0007")
    second = _authority("SFP-0008")
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(
                _mapping(first, requirement, mapping_id="mapping:first"),
                _mapping(second, requirement, mapping_id="mapping:second"),
            ),
        )
    )

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        adapter.adapt(
            _after_normalization_rejection(requirement),
            authority_requirement=requirement,
            authority_inputs=(
                _authority_input(first, requirement),
                _authority_input(second, requirement),
            ),
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_AUTHORITY_MAPPING_AMBIGUOUS"


def test_requirement_identity_and_subject_binding_must_match_assessment() -> None:
    requirement = _requirement()
    changed_requirement = {
        **requirement,
        "subject_binding": _subject_binding("SFP-0008"),
    }
    authority = _authority()
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        adapter.adapt(
            _after_normalization_rejection(requirement),
            authority_requirement=changed_requirement,
            authority_inputs=(_authority_input(authority, requirement),),
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_REQUIREMENT_SUBJECT_MISMATCH"


def test_catalog_duplicate_exact_mapping_fails_closed() -> None:
    requirement = _requirement()
    authority = _authority()
    mapping = _mapping(authority, requirement, mapping_id="mapping:a")
    duplicate = AuthorityCompatibilityMapping(
        id="mapping:b",
        requirement_id=mapping.requirement_id,
        requirement_subject_binding_digest=mapping.requirement_subject_binding_digest,
        required_effects=mapping.required_effects,
        authority_id=mapping.authority_id,
        authority_semantics_digest=mapping.authority_semantics_digest,
        canonical_key=mapping.canonical_key,
        canonical_value=mapping.canonical_value,
        closed_vocabulary=mapping.closed_vocabulary,
    )

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(mapping, duplicate),
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_MAPPING_AMBIGUOUS"


@pytest.mark.parametrize(
    "invalid_identity",
    ["decisions/ADR-0001.md", "a" * 40, "HEAD"],
)
def test_physical_locator_cannot_be_consumed_authority_identity(
    invalid_identity: str,
) -> None:
    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        RecoveredCapability(
            required_capability="component.classification",
            subject="component:runtime-a",
            canonical_key="component.classification",
            canonical_value="DEVELOPMENT_TOOLING",
            closed_vocabulary=("DEVELOPMENT_TOOLING",),
            source_path=CapabilityRecoveryPath.PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER,
            authority_requirement_id="authority-requirement:classification",
            consumed_project_authority_ids=(invalid_identity,),
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_AUTHORITY_IDENTITY_INVALID"


def test_closed_vocabulary_is_enforced() -> None:
    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        RecoveredCapability(
            required_capability="component.classification",
            subject="component:runtime-a",
            canonical_key="component.classification",
            canonical_value="DEVELOPMENT_TOOLING",
            closed_vocabulary=("PRODUCT",),
            source_path=CapabilityRecoveryPath.LOSSLESS_CANONICAL_NORMALIZATION,
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_VALUE_OUTSIDE_VOCABULARY"


def test_success_stops_later_recovery_paths() -> None:
    assessment = recover_on_next_path(
        start_capability_recovery(
            required_capability="component.classification",
            subject="component:runtime-a",
        ),
        recovered_capability=RecoveredCapability(
            required_capability="component.classification",
            subject="component:runtime-a",
            canonical_key="component.classification",
            canonical_value="DEVELOPMENT_TOOLING",
            closed_vocabulary=("DEVELOPMENT_TOOLING",),
            source_path=CapabilityRecoveryPath.LOSSLESS_CANONICAL_NORMALIZATION,
        ),
        reason_code="ALREADY_CANONICAL",
        reason="The existing value is already canonical.",
    )

    assert assessment.recovered is True
    assert assessment.next_path is None

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        reject_next_recovery_path(
            assessment,
            reason_code="MUST_NOT_RUN",
            reason="No later path may run after successful recovery.",
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_ALREADY_COMPLETE"


def test_malformed_authority_input_fails_closed_with_contract_error() -> None:
    requirement = _requirement()
    authority = _authority()
    adapter = ProjectLocalAuthorityCompatibilityAdapter(
        AuthorityCompatibilityCatalog(
            id="authority-compatibility:project",
            mappings=(_mapping(authority, requirement),),
        )
    )

    with pytest.raises(CapabilityRecoveryContractError) as exc_info:
        adapter.adapt(
            _after_normalization_rejection(requirement),
            authority_requirement=requirement,
            authority_inputs=("not-an-eligibility-binding",),
        )

    assert exc_info.value.code == "CAPABILITY_RECOVERY_AUTHORITY_INPUTS_INVALID"


def test_adapter_has_no_project_intent_authority_or_repository_discovery_surface() -> None:
    catalog = AuthorityCompatibilityCatalog(
        id="authority-compatibility:project",
        mappings=(),
    )
    adapter = ProjectLocalAuthorityCompatibilityAdapter(catalog)

    assert not hasattr(recovery_module, "ProjectIntentAuthority")
    assert not hasattr(adapter, "repository_root")
    assert not hasattr(adapter, "discover")
