from __future__ import annotations

from dataclasses import fields

import pytest

from ptsip.remediation.solution.candidate import (
    AuthorityRequirement,
    CandidateGenerationContractError,
    CandidateProvenance,
    CandidateSupportingReferences,
    SemanticCandidate,
    SemanticCandidateProposal,
    materialize_candidate,
    materialize_candidates,
)


def _subject_binding() -> dict[str, object]:
    return {
        "authority_domain": "PROJECT_GOVERNANCE",
        "repository_binding": {
            "scheme": "GIT_REPOSITORY",
            "host": "github.com",
            "repository_id": "Kinirin/PTSIP",
        },
        "subject_type": "COMPONENT",
        "subject_identity": {
            "scheme": "COMPONENT_ID",
            "value": "product-runtime",
        },
    }


def _proposal(
    *,
    local_id: str = "non-shipped-colocation",
    references: CandidateSupportingReferences | None = None,
    requirements: tuple[AuthorityRequirement, ...] = (),
) -> SemanticCandidateProposal:
    return SemanticCandidateProposal(
        interpretation_id=f"interpretation:PTSIP-PKG-001:{local_id}",
        rule_id="PTSIP-PKG-001",
        remediation_family="NON_REMEDIATION_INTERPRETATION",
        target_state={
            "shipping_role": "NON_SHIPPED",
            "repository_colocation": True,
        },
        supporting_references=references
        or CandidateSupportingReferences(
            rules=("PTSIP-PKG-001",),
            facts=("fact:artifact-not-shipped",),
        ),
        authority_requirements=requirements,
    )


def test_non_remediation_interpretation_materializes_without_violation_assumption() -> None:
    candidate = materialize_candidate(_proposal())

    assert candidate.id == "interpretation:PTSIP-PKG-001:non-shipped-colocation"
    assert candidate.remediation_family == "NON_REMEDIATION_INTERPRETATION"
    assert candidate.target_state == {
        "repository_colocation": True,
        "shipping_role": "NON_SHIPPED",
    }
    assert "evaluation_status" not in candidate.as_dict()
    assert "violation" not in candidate.as_dict()


def test_candidate_has_exact_canonical_shape_without_legacy_ssot_fields() -> None:
    candidate = materialize_candidate(_proposal())

    assert [item.name for item in fields(SemanticCandidate)] == [
        "id",
        "rule_id",
        "remediation_family",
        "target_state",
        "provenance",
    ]
    assert set(candidate.as_dict()) == {
        "id",
        "rule_id",
        "remediation_family",
        "target_state",
        "provenance",
    }
    assert "fact_ids" not in candidate.as_dict()
    assert "required_authority_ids" not in candidate.as_dict()


def test_producer_rule_must_be_explicitly_consumed_by_provenance() -> None:
    with pytest.raises(CandidateGenerationContractError) as exc_info:
        _proposal(
            references=CandidateSupportingReferences(
                rules=("PTSIP-DEP-001",),
                facts=("fact:artifact-not-shipped",),
            )
        )

    assert exc_info.value.code == "CANDIDATE_PRODUCER_RULE_NOT_CONSUMED"


def test_consumed_authority_and_unsatisfied_authority_requirement_are_separate_channels() -> None:
    requirement = AuthorityRequirement(
        requirement_id="requirement:component-owner",
        subject_binding=_subject_binding(),
        required_effects=("declare_component_owner",),
    )
    candidate = materialize_candidate(
        _proposal(
            references=CandidateSupportingReferences(
                rules=("PTSIP-PKG-001",),
                facts=("fact:artifact-hosted-tooling",),
                authorities=("authority:current-project-architecture",),
            ),
            requirements=(requirement,),
        )
    )

    provenance = candidate.as_dict()["provenance"]
    assert provenance["supporting_references"]["AUTHORITY"] == [
        "authority:current-project-architecture"
    ]
    assert provenance["authority_requirements"] == [
        {
            "requirement_id": "requirement:component-owner",
            "subject_binding": _subject_binding(),
            "required_effects": ["declare_component_owner"],
        }
    ]


def test_authority_requirement_reuses_established_governance_subject_binding_shape() -> None:
    with pytest.raises(CandidateGenerationContractError) as exc_info:
        AuthorityRequirement(
            requirement_id="requirement:bad-subject",
            subject_binding={"subject": "free-text"},
            required_effects=("declare_component_owner",),
        )

    assert exc_info.value.code == "AUTHORITY_REQUIREMENT_SUBJECT_BINDING_INVALID"


def test_authority_requirement_requires_machine_readable_effects() -> None:
    with pytest.raises(CandidateGenerationContractError) as exc_info:
        AuthorityRequirement(
            requirement_id="requirement:no-effects",
            subject_binding=_subject_binding(),
            required_effects=(),
        )

    assert exc_info.value.code == "CANDIDATE_REFERENCE_COLLECTION_EMPTY"


def test_reference_collections_are_unique_and_canonical() -> None:
    refs = CandidateSupportingReferences(
        rules=("PTSIP-PKG-001", "PTSIP-DEP-001"),
        facts=("fact:b", "fact:a"),
        authorities=("authority:b", "authority:a"),
    )

    assert refs.rules == ("PTSIP-DEP-001", "PTSIP-PKG-001")
    assert refs.facts == ("fact:a", "fact:b")
    assert refs.authorities == ("authority:a", "authority:b")

    with pytest.raises(CandidateGenerationContractError) as exc_info:
        CandidateSupportingReferences(
            rules=("PTSIP-PKG-001", "PTSIP-PKG-001"),
        )
    assert exc_info.value.code == "CANDIDATE_REFERENCE_DUPLICATE"


def test_cross_role_reference_reuse_is_not_implicitly_forbidden() -> None:
    refs = CandidateSupportingReferences(
        rules=("PTSIP-PKG-001",),
        facts=("shared:identity",),
        authorities=("shared:identity",),
    )

    assert refs.facts == refs.authorities == ("shared:identity",)


def test_interpretation_identity_is_bound_to_producer_rule() -> None:
    with pytest.raises(CandidateGenerationContractError) as exc_info:
        SemanticCandidateProposal(
            interpretation_id="interpretation:PTSIP-DEP-001:wrong-rule",
            rule_id="PTSIP-PKG-001",
            remediation_family="PACKAGE_ISOLATION",
            target_state={"shipping_role": "EXCLUDED"},
            supporting_references=CandidateSupportingReferences(
                rules=("PTSIP-PKG-001",),
            ),
        )

    assert exc_info.value.code == "CANDIDATE_INTERPRETATION_ID_SCOPE_MISMATCH"


def test_materialization_preserves_proposal_order_but_rejects_duplicate_identity() -> None:
    first = _proposal(local_id="a")
    second = _proposal(local_id="b")

    result = materialize_candidates((second, first))
    assert tuple(item.id for item in result) == (
        "interpretation:PTSIP-PKG-001:b",
        "interpretation:PTSIP-PKG-001:a",
    )

    with pytest.raises(CandidateGenerationContractError) as exc_info:
        materialize_candidates((first, first))
    assert exc_info.value.code == "CANDIDATE_ID_DUPLICATE"


def test_candidate_provenance_is_the_only_support_and_requirement_ssot() -> None:
    requirement = AuthorityRequirement(
        requirement_id="requirement:owner-intent",
        subject_binding=_subject_binding(),
        required_effects=("declare_component_owner", "declare_shipping_role"),
    )
    provenance = CandidateProvenance(
        supporting_references=CandidateSupportingReferences(
            rules=("PTSIP-PKG-001",),
            facts=("fact:1",),
        ),
        authority_requirements=(requirement,),
    )

    assert set(provenance.as_dict()) == {
        "supporting_references",
        "authority_requirements",
    }
    assert "required_authority_ids" not in provenance.as_dict()
    assert "mutation_authorization" not in provenance.as_dict()
