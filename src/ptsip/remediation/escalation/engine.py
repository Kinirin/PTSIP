from __future__ import annotations

from ..domain import EscalationProof, ResolutionOutcome, SemanticCandidate
from ...evidence.contract import stable_digest
from .contract import (
    EscalationChoice,
    EscalationContractError,
    ExternalFactRequirement,
    FreshSolveBinding,
    FreshSolveBindingComparison,
    FreshSolveHandoff,
    FreshSolveRestartPolicy,
    InputMaterializationTarget,
    ResolutionInputKind,
    ResolutionInputRequest,
)


def _stable_id(prefix: str, payload: object) -> str:
    return f"{prefix}:{stable_digest(payload)[:24]}"


def _sorted_choices(candidates: tuple[SemanticCandidate, ...]) -> tuple[EscalationChoice, ...]:
    if not isinstance(candidates, tuple) or any(
        not isinstance(candidate, SemanticCandidate) for candidate in candidates
    ):
        raise EscalationContractError(
            "ESCALATION_CANDIDATES_INVALID",
            "candidates must be a tuple of already-modeled SemanticCandidate values.",
            candidates,
        )
    candidate_ids = tuple(candidate.id for candidate in candidates)
    if len(set(candidate_ids)) != len(candidate_ids):
        raise EscalationContractError(
            "ESCALATION_CANDIDATE_IDS_DUPLICATE",
            "Semantic candidate identifiers must be unique before escalation.",
            candidate_ids,
        )
    return tuple(
        EscalationChoice.from_candidate(candidate)
        for candidate in sorted(candidates, key=lambda item: item.id)
    )


def compare_fresh_solve_bindings(
    previous: FreshSolveBinding,
    current: FreshSolveBinding,
) -> FreshSolveBindingComparison:
    if not isinstance(previous, FreshSolveBinding) or not isinstance(current, FreshSolveBinding):
        raise EscalationContractError(
            "ESCALATION_FRESH_SOLVE_BINDING_REQUIRED",
            "Fresh Solve comparison requires two FreshSolveBinding values.",
            (previous, current),
        )
    dimensions = (
        "evidence_digest",
        "specification",
        "repository_revision",
        "repository_status_fingerprint",
        "repository_tracked_content_fingerprint",
        "authority_snapshot_id",
    )
    changed = tuple(
        dimension
        for dimension in dimensions
        if getattr(previous, dimension) != getattr(current, dimension)
    )
    return FreshSolveBindingComparison(
        previous_digest=previous.identity_digest,
        current_digest=current.identity_digest,
        changed_dimensions=changed,
    )


def require_current_resolution_input_request(
    request: ResolutionInputRequest,
    current_binding: FreshSolveBinding,
) -> FreshSolveBindingComparison:
    if not isinstance(request, ResolutionInputRequest):
        raise EscalationContractError(
            "ESCALATION_REQUEST_REQUIRED",
            "Freshness validation requires a ResolutionInputRequest.",
            request,
        )
    comparison = compare_fresh_solve_bindings(request.solve_binding, current_binding)
    if comparison.changed:
        raise EscalationContractError(
            "ESCALATION_REQUEST_STALE",
            "The resolution request is stale because current Specification, authority, evidence, or repository state changed.",
            comparison.as_dict(),
        )
    return comparison


def build_resolution_input_request(
    *,
    outcome: ResolutionOutcome,
    solve_binding: FreshSolveBinding,
    unresolved_dimension: str,
    candidates: tuple[SemanticCandidate, ...],
    eliminated_candidate_ids: tuple[str, ...] = (),
    fact_ids: tuple[str, ...] = (),
    constraint_rule_ids: tuple[str, ...] = (),
    available_authority_ids: tuple[str, ...] = (),
    required_input: str,
) -> ResolutionInputRequest:
    if outcome not in (
        ResolutionOutcome.OWNER_INTENT_REQUIRED,
        ResolutionOutcome.EXTERNAL_FACT_REQUIRED,
    ):
        raise EscalationContractError(
            "ESCALATION_OUTCOME_NOT_INPUT_ACTIONABLE",
            "Only OWNER_INTENT_REQUIRED or EXTERNAL_FACT_REQUIRED may produce a resolution input request.",
            outcome,
        )
    choices = _sorted_choices(candidates)
    candidate_ids = tuple(choice.candidate_id for choice in choices)

    if outcome is ResolutionOutcome.OWNER_INTENT_REQUIRED:
        if len(choices) < 2:
            raise EscalationContractError(
                "ESCALATION_OWNER_CARDINALITY_INVALID",
                "Owner intent may be requested only after deterministic reduction leaves at least two legal targets.",
                candidate_ids,
            )
        proof_payload = {
            "unresolved_dimension": unresolved_dimension,
            "surviving_candidate_ids": candidate_ids,
            "eliminated_candidate_ids": eliminated_candidate_ids,
            "fact_ids": fact_ids,
            "constraint_rule_ids": constraint_rule_ids,
            "available_authority_ids": available_authority_ids,
            "required_input": required_input,
            "solve_binding": solve_binding.identity_digest,
        }
        proof = EscalationProof(
            id=_stable_id("escalation-proof", proof_payload),
            unresolved_dimension=unresolved_dimension,
            surviving_candidate_ids=candidate_ids,
            eliminated_candidate_ids=eliminated_candidate_ids,
            fact_ids=fact_ids,
            constraint_rule_ids=constraint_rule_ids,
            available_authority_ids=available_authority_ids,
            required_input=required_input,
        )
        request_payload = {
            "kind": ResolutionInputKind.PROJECT_OWNER_INTENT.value,
            "solve_binding": solve_binding.identity_digest,
            "proof_id": proof.id,
        }
        return ResolutionInputRequest(
            id=_stable_id("resolution-input", request_payload),
            kind=ResolutionInputKind.PROJECT_OWNER_INTENT,
            solve_binding=solve_binding,
            question=f"Select the project-authoritative target for {unresolved_dimension}.",
            choices=choices,
            owner_escalation_proof=proof,
        )

    requirement_payload = {
        "unresolved_dimension": unresolved_dimension,
        "required_fact": required_input,
        "candidate_ids": candidate_ids,
        "basis_fact_ids": fact_ids,
        "constraint_rule_ids": constraint_rule_ids,
        "solve_binding": solve_binding.identity_digest,
    }
    requirement = ExternalFactRequirement(
        id=_stable_id("external-fact", requirement_payload),
        unresolved_dimension=unresolved_dimension,
        required_fact=required_input,
        candidate_ids=candidate_ids,
        basis_fact_ids=fact_ids,
        constraint_rule_ids=constraint_rule_ids,
    )
    request_payload = {
        "kind": ResolutionInputKind.EXTERNAL_FACT.value,
        "solve_binding": solve_binding.identity_digest,
        "requirement_id": requirement.id,
    }
    return ResolutionInputRequest(
        id=_stable_id("resolution-input", request_payload),
        kind=ResolutionInputKind.EXTERNAL_FACT,
        solve_binding=solve_binding,
        question=(
            f"Provide the external fact required for {unresolved_dimension}: {required_input}."
        ),
        choices=choices,
        external_fact_requirement=requirement,
    )


def prepare_fresh_solve_handoff(
    request: ResolutionInputRequest,
    *,
    current_binding: FreshSolveBinding,
    supplied_input_ref: str,
) -> FreshSolveHandoff:
    require_current_resolution_input_request(request, current_binding)
    target = (
        InputMaterializationTarget.PROJECT_AUTHORITY
        if request.kind is ResolutionInputKind.PROJECT_OWNER_INTENT
        else InputMaterializationTarget.EVIDENCE
    )
    return FreshSolveHandoff(
        request_id=request.id,
        input_kind=request.kind,
        materialization_target=target,
        prior_solve_binding=request.solve_binding,
        supplied_input_ref=supplied_input_ref,
        restart_policy=FreshSolveRestartPolicy.REQUIRED,
    )
