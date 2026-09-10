from __future__ import annotations

from dataclasses import fields

import pytest

from ptsip.evidence.contract import (
    EvidenceEvaluationContext,
    NormalizedEvidenceSet,
    SnapshotBinding,
)
from ptsip.remediation.domain import RemediationContext, ResolutionOutcome, SemanticCandidate
from ptsip.remediation.escalation import (
    EscalationContractError,
    FreshSolveBinding,
    FreshSolveRestartPolicy,
    InputMaterializationTarget,
    ResolutionInputKind,
    build_resolution_input_request,
    compare_fresh_solve_bindings,
    prepare_fresh_solve_handoff,
    require_current_resolution_input_request,
)
from ptsip.specification_binding import SPECIFICATION_037


def _context(
    *,
    status: str = "status:clean",
    tracked: str = "tracked:abc",
    revision: str | None = "a" * 40,
) -> RemediationContext:
    evidence = NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="wu03-escalation-test",
            snapshot=SnapshotBinding(
                repository_root=".",
                revision=revision,
                status_fingerprint=status,
                tracked_content_fingerprint=tracked,
            ),
        ),
        records=(),
        channels=(),
    )
    return RemediationContext(evidence=evidence, specification=SPECIFICATION_037)


def _binding(
    *,
    authority: str = "authority:snapshot:1",
    status: str = "status:clean",
    tracked: str = "tracked:abc",
) -> FreshSolveBinding:
    return FreshSolveBinding.from_context(
        _context(status=status, tracked=tracked),
        authority_snapshot_id=authority,
    )


def _candidate(candidate_id: str, target: str) -> SemanticCandidate:
    return SemanticCandidate(
        id=candidate_id,
        rule_id="PTSIP-CLS-001",
        remediation_family="COMPONENT_BINDING",
        target_state={"component": target},
        fact_ids=("fact:coverage:1",),
    )


def test_owner_intent_request_contains_only_remaining_semantic_choice_boundary() -> None:
    request = build_resolution_input_request(
        outcome=ResolutionOutcome.OWNER_INTENT_REQUIRED,
        solve_binding=_binding(),
        unresolved_dimension="component ownership for src/new.py",
        candidates=(
            _candidate("candidate:z", "runtime-b"),
            _candidate("candidate:a", "runtime-a"),
        ),
        eliminated_candidate_ids=("candidate:invalid",),
        fact_ids=("fact:coverage:1",),
        constraint_rule_ids=("PTSIP-CLS-001",),
        available_authority_ids=("authority:profile:1",),
        required_input="project-owned component selection",
    )

    assert request.kind is ResolutionInputKind.PROJECT_OWNER_INTENT
    assert tuple(choice.candidate_id for choice in request.choices) == (
        "candidate:a",
        "candidate:z",
    )
    assert request.owner_escalation_proof is not None
    assert request.owner_escalation_proof.surviving_candidate_ids == (
        "candidate:a",
        "candidate:z",
    )
    assert request.external_fact_requirement is None
    assert request.question == "Select the project-authoritative target for component ownership for src/new.py."
    assert request.as_dict()["authoritative"] is False


def test_owner_question_is_forbidden_when_only_one_semantic_target_remains() -> None:
    with pytest.raises(EscalationContractError) as exc_info:
        build_resolution_input_request(
            outcome=ResolutionOutcome.OWNER_INTENT_REQUIRED,
            solve_binding=_binding(),
            unresolved_dimension="component ownership",
            candidates=(_candidate("candidate:only", "runtime-a"),),
            required_input="owner choice",
        )

    assert exc_info.value.code == "ESCALATION_OWNER_CARDINALITY_INVALID"


@pytest.mark.parametrize(
    "outcome",
    [
        ResolutionOutcome.DETERMINISTIC,
        ResolutionOutcome.UNSATISFIABLE,
        ResolutionOutcome.TOOL_CAPABILITY_GAP,
    ],
)
def test_non_input_outcomes_cannot_be_converted_into_human_questions(outcome: ResolutionOutcome) -> None:
    with pytest.raises(EscalationContractError) as exc_info:
        build_resolution_input_request(
            outcome=outcome,
            solve_binding=_binding(),
            unresolved_dimension="unsupported dimension",
            candidates=(),
            required_input="do not ask a human",
        )

    assert exc_info.value.code == "ESCALATION_OUTCOME_NOT_INPUT_ACTIONABLE"


def test_external_fact_request_can_exist_before_semantic_candidates_are_available() -> None:
    request = build_resolution_input_request(
        outcome=ResolutionOutcome.EXTERNAL_FACT_REQUIRED,
        solve_binding=_binding(),
        unresolved_dimension="published artifact membership",
        candidates=(),
        fact_ids=("fact:artifact:insufficient",),
        constraint_rule_ids=("PTSIP-PKG-001",),
        required_input="exact published wheel content manifest",
    )

    assert request.kind is ResolutionInputKind.EXTERNAL_FACT
    assert request.choices == ()
    assert request.external_fact_requirement is not None
    assert request.external_fact_requirement.candidate_ids == ()
    assert request.external_fact_requirement.as_dict()["authoritative"] is False
    assert request.owner_escalation_proof is None


def test_owner_input_routes_to_project_authority_then_requires_fresh_solve() -> None:
    binding = _binding()
    request = build_resolution_input_request(
        outcome=ResolutionOutcome.OWNER_INTENT_REQUIRED,
        solve_binding=binding,
        unresolved_dimension="component ownership",
        candidates=(
            _candidate("candidate:a", "runtime-a"),
            _candidate("candidate:b", "runtime-b"),
        ),
        required_input="project-owned component selection",
    )

    handoff = prepare_fresh_solve_handoff(
        request,
        current_binding=binding,
        supplied_input_ref="owner-response:42",
    )

    assert handoff.materialization_target is InputMaterializationTarget.PROJECT_AUTHORITY
    assert handoff.restart_policy is FreshSolveRestartPolicy.REQUIRED
    assert "semantic_plan" not in {item.name for item in fields(type(handoff))}
    assert "target_candidate_id" not in {item.name for item in fields(type(handoff))}


def test_external_fact_routes_to_evidence_then_requires_fresh_solve() -> None:
    binding = _binding()
    request = build_resolution_input_request(
        outcome=ResolutionOutcome.EXTERNAL_FACT_REQUIRED,
        solve_binding=binding,
        unresolved_dimension="package assembly truth",
        candidates=(),
        required_input="build-system output",
    )

    handoff = prepare_fresh_solve_handoff(
        request,
        current_binding=binding,
        supplied_input_ref="external-evidence:manifest:1",
    )

    assert handoff.materialization_target is InputMaterializationTarget.EVIDENCE
    assert handoff.restart_policy is FreshSolveRestartPolicy.REQUIRED


def test_stale_resolution_request_is_rejected_when_authority_changes() -> None:
    previous = _binding(authority="authority:snapshot:1")
    request = build_resolution_input_request(
        outcome=ResolutionOutcome.OWNER_INTENT_REQUIRED,
        solve_binding=previous,
        unresolved_dimension="component ownership",
        candidates=(
            _candidate("candidate:a", "runtime-a"),
            _candidate("candidate:b", "runtime-b"),
        ),
        required_input="project-owned component selection",
    )
    current = _binding(authority="authority:snapshot:2")

    with pytest.raises(EscalationContractError) as exc_info:
        require_current_resolution_input_request(request, current)

    assert exc_info.value.code == "ESCALATION_REQUEST_STALE"
    assert "authority_snapshot_id" in exc_info.value.value["changed_dimensions"]


def test_fresh_solve_comparison_identifies_repository_state_changes() -> None:
    previous = _binding(status="status:clean", tracked="tracked:abc")
    current = _binding(status="status:dirty", tracked="tracked:def")

    comparison = compare_fresh_solve_bindings(previous, current)

    assert comparison.changed is True
    assert comparison.changed_dimensions == (
        "evidence_digest",
        "repository_status_fingerprint",
        "repository_tracked_content_fingerprint",
    )


def test_fresh_solve_binding_preserves_exact_spec_repository_and_authority_identity() -> None:
    context = _context()
    binding = FreshSolveBinding.from_context(
        context,
        authority_snapshot_id="authority:snapshot:exact",
    )

    payload = binding.as_dict()
    assert payload["evidence_digest"] == context.evidence.deterministic_digest
    assert payload["specification"] == SPECIFICATION_037.as_dict()
    assert payload["repository_revision"] == "a" * 40
    assert payload["repository_status_fingerprint"] == "status:clean"
    assert payload["repository_tracked_content_fingerprint"] == "tracked:abc"
    assert payload["authority_snapshot_id"] == "authority:snapshot:exact"
