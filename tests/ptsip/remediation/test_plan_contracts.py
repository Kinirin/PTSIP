from __future__ import annotations

import pytest

from ptsip.remediation.domain import (
    EscalationProof,
    MutationClass,
    PostconditionResult,
    RemediationContractError,
    RepositoryChangePlan,
    SemanticCandidate,
    SemanticRemediationPlan,
)


def test_semantic_candidate_and_plan_remain_separate_from_physical_plan() -> None:
    candidate = SemanticCandidate(
        id="candidate:pkg:1",
        rule_id="PTSIP-PKG-001",
        remediation_family="PACKAGE_ISOLATION",
        target_state="product artifact excludes tooling implementation",
        fact_ids=("fact:pkg:1",),
    )
    semantic = SemanticRemediationPlan(
        id="semantic-plan:pkg:1",
        target_candidate_id=candidate.id,
        rule_ids=(candidate.rule_id,),
        fact_ids=candidate.fact_ids,
    )
    physical = RepositoryChangePlan(
        id="change-plan:pkg:1",
        semantic_plan_id=semantic.id,
        mutation_class=MutationClass.MECHANICAL_REVERSIBLE,
        prestate_fingerprint="snapshot:abc",
        operation_ids=("operation:1",),
    )

    assert semantic.id == physical.semantic_plan_id
    assert type(semantic) is not type(physical)
    assert "mutation_class" not in semantic.as_dict()
    assert physical.as_dict()["mutation_class"] == "MECHANICAL_REVERSIBLE"


def test_escalation_proof_is_a_record_not_an_authority_object() -> None:
    proof = EscalationProof(
        id="escalation:1",
        unresolved_dimension="governing lifecycle intent",
        surviving_candidate_ids=("candidate:1", "candidate:2"),
        eliminated_candidate_ids=("candidate:3",),
        fact_ids=("fact:1",),
        constraint_rule_ids=("PTSIP-CLS-001",),
        available_authority_ids=(),
        required_input="project owner lifecycle intent",
    )

    payload = proof.as_dict()
    assert payload["surviving_candidate_ids"] == ["candidate:1", "candidate:2"]
    assert "authorization" not in payload
    assert "project_intent" not in payload


def test_postcondition_result_rejects_satisfied_state_with_failed_checks() -> None:
    with pytest.raises(RemediationContractError) as exc_info:
        PostconditionResult(
            semantic_plan_id="semantic-plan:1",
            satisfied=True,
            verified_rule_ids=("PTSIP-PKG-001",),
            failed_check_ids=("check:unexpected-delta",),
        )
    assert exc_info.value.code == "REMEDIATION_POSTCONDITION_CONFLICT"
