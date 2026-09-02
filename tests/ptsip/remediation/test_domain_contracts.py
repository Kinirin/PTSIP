from __future__ import annotations

from dataclasses import FrozenInstanceError, fields

import pytest

from ptsip.evidence.contract import (
    EvidenceEvaluationContext,
    NormalizedEvidenceSet,
    SnapshotBinding,
)
from ptsip.remediation.domain import (
    AuthorizationDecision,
    CoverageGap,
    DerivedFact,
    MutationClass,
    NormativeConstraint,
    ProjectIntentAuthority,
    ProjectIntentSource,
    RemediationContext,
    RemediationContractError,
    RemediationDisposition,
    RemediationDispositionCandidate,
    ResolutionOutcome,
)
from ptsip.specification_binding import SPECIFICATION_037


def _evidence() -> NormalizedEvidenceSet:
    return NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="remediation-domain-test",
            snapshot=SnapshotBinding(
                repository_root=".",
                revision=None,
                status_fingerprint="status",
                tracked_content_fingerprint="content",
            ),
        ),
        records=(),
        channels=(),
    )


def test_remediation_context_reuses_existing_evidence_and_specification_contracts() -> None:
    evidence = _evidence()
    context = RemediationContext(evidence=evidence, specification=SPECIFICATION_037)

    assert context.evidence is evidence
    assert context.specification is SPECIFICATION_037
    assert context.as_dict()["evidence_digest"] == evidence.deterministic_digest
    assert context.as_dict()["specification"] == SPECIFICATION_037.as_dict()


def test_resolution_mutation_and_authorization_axes_are_distinct() -> None:
    assert [item.value for item in ResolutionOutcome] == [
        "DETERMINISTIC",
        "OWNER_INTENT_REQUIRED",
        "EXTERNAL_FACT_REQUIRED",
        "UNSATISFIABLE",
        "TOOL_CAPABILITY_GAP",
    ]
    assert [item.value for item in MutationClass] == [
        "MECHANICAL_REVERSIBLE",
        "STRUCTURAL_SEMANTIC_PRESERVING",
        "ARCHITECTURE_SEMANTIC",
        "DESTRUCTIVE",
    ]
    assert [item.value for item in AuthorizationDecision] == [
        "AUTHORIZED",
        "OWNER_CONFIRMATION_REQUIRED",
        "NOT_AUTHORIZED",
    ]
    assert ResolutionOutcome.DETERMINISTIC != AuthorizationDecision.AUTHORIZED
    assert MutationClass.MECHANICAL_REVERSIBLE != AuthorizationDecision.AUTHORIZED


def test_derived_fact_requires_evidence_and_does_not_gain_authority() -> None:
    fact = DerivedFact(
        id="fact:1",
        subject="src/pkg.py",
        predicate="artifact-membership",
        value={"present": True},
        evidence_ids=("evidence:1",),
    )

    assert fact.evidence_ids == ("evidence:1",)
    assert fact.as_dict()["value"] == {"present": True}
    assert "authority" not in {item.name for item in fields(DerivedFact)}

    with pytest.raises(RemediationContractError) as exc_info:
        DerivedFact(
            id="fact:2",
            subject="src/pkg.py",
            predicate="artifact-membership",
            value=True,
            evidence_ids=(),
        )
    assert exc_info.value.code == "REMEDIATION_IDS_EMPTY"


def test_normative_constraint_requires_explicit_specification_binding() -> None:
    constraint = NormativeConstraint(
        rule_id="PTSIP-PKG-001",
        specification=SPECIFICATION_037,
        subject="product-artifact",
        requirement="exclude explicitly non-product implementation",
    )
    assert constraint.specification is SPECIFICATION_037

    with pytest.raises(RemediationContractError) as exc_info:
        NormativeConstraint(
            rule_id="PTSIP-PKG-001",
            specification="0.3.7-draft",  # type: ignore[arg-type]
            subject="product-artifact",
            requirement="exclude explicitly non-product implementation",
        )
    assert exc_info.value.code == "REMEDIATION_SPECIFICATION_TYPE"


def test_coverage_gap_candidate_and_project_intent_are_separate_types() -> None:
    gap = CoverageGap(
        id="gap:unassigned:src/new.py",
        code="component-ownership:unassigned-relevant-files",
        subject="src/new.py",
        evidence_ids=("evidence:coverage:1",),
    )
    candidate = RemediationDispositionCandidate(
        id="candidate:existing-component:src/new.py",
        gap_id=gap.id,
        disposition=RemediationDisposition.EXISTING_COMPONENT_CANDIDATE,
        rationale="Existing component coverage is a modeled candidate only.",
        evidence_ids=gap.evidence_ids,
    )
    authority = ProjectIntentAuthority(
        id="intent:component-owner:src/new.py",
        subject="src/new.py",
        intent="owned by product-runtime",
        source=ProjectIntentSource.OWNER_DECISION,
        source_ref="decision:42",
    )

    assert type(gap) is not type(candidate)
    assert type(candidate) is not type(authority)
    assert candidate.as_dict()["disposition"] == "EXISTING_COMPONENT_CANDIDATE"
    assert "authority" not in {item.name for item in fields(CoverageGap)}
    assert "authority" not in {item.name for item in fields(RemediationDispositionCandidate)}


def test_project_intent_authority_cannot_be_constructed_from_evidence_source() -> None:
    with pytest.raises(RemediationContractError) as exc_info:
        ProjectIntentAuthority(
            id="intent:invalid",
            subject="src/new.py",
            intent="owned by product-runtime",
            source="EVIDENCE",  # type: ignore[arg-type]
            source_ref="evidence:1",
        )
    assert exc_info.value.code == "REMEDIATION_INTENT_SOURCE_INVALID"


def test_domain_records_are_immutable() -> None:
    gap = CoverageGap(
        id="gap:1",
        code="coverage",
        subject="src/new.py",
    )
    with pytest.raises(FrozenInstanceError):
        gap.subject = "src/other.py"  # type: ignore[misc]
