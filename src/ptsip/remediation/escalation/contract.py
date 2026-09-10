from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

from ..domain import EscalationProof, RemediationContext, SemanticCandidate
from ...evidence.contract import canonical_value, stable_digest
from ...specification_binding import SpecificationBinding


class EscalationContractError(ValueError):
    """Stable fail-closed error for WU-03 escalation and Fresh Solve contracts."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class ResolutionInputKind(StrEnum):
    PROJECT_OWNER_INTENT = "PROJECT_OWNER_INTENT"
    EXTERNAL_FACT = "EXTERNAL_FACT"


class InputMaterializationTarget(StrEnum):
    PROJECT_AUTHORITY = "PROJECT_AUTHORITY"
    EVIDENCE = "EVIDENCE"


class FreshSolveRestartPolicy(StrEnum):
    REQUIRED = "REQUIRED"


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise EscalationContractError(
            "ESCALATION_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _require_ids(name: str, values: tuple[str, ...], *, allow_empty: bool = True) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise EscalationContractError(
            "ESCALATION_IDS_TYPE",
            f"{name} must be a tuple of canonical identifiers.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if not allow_empty and not normalized:
        raise EscalationContractError(
            "ESCALATION_IDS_EMPTY",
            f"{name} must contain at least one identifier.",
            values,
        )
    if len(set(normalized)) != len(normalized):
        raise EscalationContractError(
            "ESCALATION_IDS_DUPLICATE",
            f"{name} must not contain duplicate identifiers.",
            values,
        )
    return normalized


@dataclass(frozen=True)
class FreshSolveBinding:
    """Exact current-state identity that invalidates stale resolution requests."""

    evidence_digest: str
    specification: SpecificationBinding
    repository_revision: str | None
    repository_status_fingerprint: str
    repository_tracked_content_fingerprint: str
    authority_snapshot_id: str

    def __post_init__(self) -> None:
        _require_text("evidence_digest", self.evidence_digest)
        if not isinstance(self.specification, SpecificationBinding):
            raise EscalationContractError(
                "ESCALATION_SPECIFICATION_BINDING_REQUIRED",
                "FreshSolveBinding requires an explicit SpecificationBinding.",
                self.specification,
            )
        if self.repository_revision is not None:
            _require_text("repository_revision", self.repository_revision)
        _require_text("repository_status_fingerprint", self.repository_status_fingerprint)
        _require_text(
            "repository_tracked_content_fingerprint",
            self.repository_tracked_content_fingerprint,
        )
        _require_text("authority_snapshot_id", self.authority_snapshot_id)

    @classmethod
    def from_context(
        cls,
        context: RemediationContext,
        *,
        authority_snapshot_id: str,
    ) -> "FreshSolveBinding":
        if not isinstance(context, RemediationContext):
            raise EscalationContractError(
                "ESCALATION_REMEDIATION_CONTEXT_REQUIRED",
                "FreshSolveBinding must be constructed from a RemediationContext.",
                context,
            )
        snapshot = context.evidence.context.snapshot
        return cls(
            evidence_digest=context.evidence.deterministic_digest,
            specification=context.specification,
            repository_revision=snapshot.revision,
            repository_status_fingerprint=snapshot.status_fingerprint,
            repository_tracked_content_fingerprint=snapshot.tracked_content_fingerprint,
            authority_snapshot_id=authority_snapshot_id,
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "evidence_digest": self.evidence_digest,
            "specification": self.specification.as_dict(),
            "repository_revision": self.repository_revision,
            "repository_status_fingerprint": self.repository_status_fingerprint,
            "repository_tracked_content_fingerprint": self.repository_tracked_content_fingerprint,
            "authority_snapshot_id": self.authority_snapshot_id,
        }

    @property
    def identity_digest(self) -> str:
        return stable_digest(self.as_dict())


@dataclass(frozen=True)
class FreshSolveBindingComparison:
    previous_digest: str
    current_digest: str
    changed_dimensions: tuple[str, ...]

    def __post_init__(self) -> None:
        _require_text("previous_digest", self.previous_digest)
        _require_text("current_digest", self.current_digest)
        _require_ids("changed_dimensions", self.changed_dimensions)

    @property
    def changed(self) -> bool:
        return bool(self.changed_dimensions)

    def as_dict(self) -> dict[str, object]:
        return {
            "previous_digest": self.previous_digest,
            "current_digest": self.current_digest,
            "changed_dimensions": list(self.changed_dimensions),
            "changed": self.changed,
        }


@dataclass(frozen=True)
class EscalationChoice:
    candidate_id: str
    rule_id: str
    remediation_family: str
    target_state: object

    def __post_init__(self) -> None:
        _require_text("candidate_id", self.candidate_id)
        _require_text("rule_id", self.rule_id)
        _require_text("remediation_family", self.remediation_family)
        if self.target_state is None:
            raise EscalationContractError(
                "ESCALATION_TARGET_STATE_REQUIRED",
                "EscalationChoice requires an explicit semantic target state.",
                self.target_state,
            )

    @classmethod
    def from_candidate(cls, candidate: SemanticCandidate) -> "EscalationChoice":
        if not isinstance(candidate, SemanticCandidate):
            raise EscalationContractError(
                "ESCALATION_SEMANTIC_CANDIDATE_REQUIRED",
                "Escalation choices must come from already-modeled SemanticCandidate values.",
                candidate,
            )
        return cls(
            candidate_id=candidate.id,
            rule_id=candidate.rule_id,
            remediation_family=candidate.remediation_family,
            target_state=candidate.target_state,
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "candidate_id": self.candidate_id,
            "rule_id": self.rule_id,
            "remediation_family": self.remediation_family,
            "target_state": canonical_value(self.target_state),
        }


@dataclass(frozen=True)
class ExternalFactRequirement:
    """Non-authoritative statement of the missing externally knowable fact."""

    id: str
    unresolved_dimension: str
    required_fact: str
    candidate_ids: tuple[str, ...] = ()
    basis_fact_ids: tuple[str, ...] = ()
    constraint_rule_ids: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("unresolved_dimension", self.unresolved_dimension)
        _require_text("required_fact", self.required_fact)
        _require_ids("candidate_ids", self.candidate_ids)
        _require_ids("basis_fact_ids", self.basis_fact_ids)
        _require_ids("constraint_rule_ids", self.constraint_rule_ids)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "unresolved_dimension": self.unresolved_dimension,
            "required_fact": self.required_fact,
            "candidate_ids": list(self.candidate_ids),
            "basis_fact_ids": list(self.basis_fact_ids),
            "constraint_rule_ids": list(self.constraint_rule_ids),
            "authoritative": False,
        }


@dataclass(frozen=True)
class ResolutionInputRequest:
    """Input request produced only after deterministic reduction reaches a real input boundary."""

    id: str
    kind: ResolutionInputKind
    solve_binding: FreshSolveBinding
    question: str
    choices: tuple[EscalationChoice, ...] = ()
    owner_escalation_proof: EscalationProof | None = None
    external_fact_requirement: ExternalFactRequirement | None = None

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        if not isinstance(self.kind, ResolutionInputKind):
            raise EscalationContractError(
                "ESCALATION_INPUT_KIND_INVALID",
                "ResolutionInputRequest requires an explicit ResolutionInputKind.",
                self.kind,
            )
        if not isinstance(self.solve_binding, FreshSolveBinding):
            raise EscalationContractError(
                "ESCALATION_FRESH_SOLVE_BINDING_REQUIRED",
                "ResolutionInputRequest requires an exact FreshSolveBinding.",
                self.solve_binding,
            )
        _require_text("question", self.question)
        if not isinstance(self.choices, tuple) or any(
            not isinstance(choice, EscalationChoice) for choice in self.choices
        ):
            raise EscalationContractError(
                "ESCALATION_CHOICES_INVALID",
                "choices must be a tuple of EscalationChoice values.",
                self.choices,
            )
        choice_ids = tuple(choice.candidate_id for choice in self.choices)
        _require_ids("choice_candidate_ids", choice_ids)

        if self.kind is ResolutionInputKind.PROJECT_OWNER_INTENT:
            if not isinstance(self.owner_escalation_proof, EscalationProof):
                raise EscalationContractError(
                    "ESCALATION_OWNER_PROOF_REQUIRED",
                    "Owner-intent requests require an EscalationProof.",
                    self.owner_escalation_proof,
                )
            if self.external_fact_requirement is not None:
                raise EscalationContractError(
                    "ESCALATION_INPUT_KIND_CONFLICT",
                    "Owner-intent requests cannot also contain an external-fact requirement.",
                    self.external_fact_requirement,
                )
            if len(self.choices) < 2:
                raise EscalationContractError(
                    "ESCALATION_OWNER_CARDINALITY_INVALID",
                    "Owner intent may be requested only when at least two legal semantic targets remain.",
                    choice_ids,
                )
            if self.owner_escalation_proof.surviving_candidate_ids != choice_ids:
                raise EscalationContractError(
                    "ESCALATION_OWNER_PROOF_CHOICE_MISMATCH",
                    "EscalationProof survivors must exactly match the presented owner choices.",
                    choice_ids,
                )
        else:
            if not isinstance(self.external_fact_requirement, ExternalFactRequirement):
                raise EscalationContractError(
                    "ESCALATION_EXTERNAL_FACT_REQUIREMENT_REQUIRED",
                    "External-fact requests require an ExternalFactRequirement.",
                    self.external_fact_requirement,
                )
            if self.owner_escalation_proof is not None:
                raise EscalationContractError(
                    "ESCALATION_INPUT_KIND_CONFLICT",
                    "External-fact requests cannot also contain an owner escalation proof.",
                    self.owner_escalation_proof,
                )
            if self.external_fact_requirement.candidate_ids != choice_ids:
                raise EscalationContractError(
                    "ESCALATION_EXTERNAL_FACT_CHOICE_MISMATCH",
                    "ExternalFactRequirement candidate_ids must exactly match any presented choices.",
                    choice_ids,
                )

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "kind": self.kind.value,
            "solve_binding": self.solve_binding.as_dict(),
            "question": self.question,
            "choices": [choice.as_dict() for choice in self.choices],
            "owner_escalation_proof": (
                self.owner_escalation_proof.as_dict() if self.owner_escalation_proof else None
            ),
            "external_fact_requirement": (
                self.external_fact_requirement.as_dict() if self.external_fact_requirement else None
            ),
            "authoritative": False,
        }


@dataclass(frozen=True)
class FreshSolveHandoff:
    """Routes supplied input to its proper materialization boundary, then requires a new solve."""

    request_id: str
    input_kind: ResolutionInputKind
    materialization_target: InputMaterializationTarget
    prior_solve_binding: FreshSolveBinding
    supplied_input_ref: str
    restart_policy: FreshSolveRestartPolicy = FreshSolveRestartPolicy.REQUIRED

    def __post_init__(self) -> None:
        _require_text("request_id", self.request_id)
        if not isinstance(self.input_kind, ResolutionInputKind):
            raise EscalationContractError(
                "ESCALATION_INPUT_KIND_INVALID",
                "FreshSolveHandoff requires an explicit ResolutionInputKind.",
                self.input_kind,
            )
        if not isinstance(self.materialization_target, InputMaterializationTarget):
            raise EscalationContractError(
                "ESCALATION_MATERIALIZATION_TARGET_INVALID",
                "FreshSolveHandoff requires an explicit materialization target.",
                self.materialization_target,
            )
        if not isinstance(self.prior_solve_binding, FreshSolveBinding):
            raise EscalationContractError(
                "ESCALATION_FRESH_SOLVE_BINDING_REQUIRED",
                "FreshSolveHandoff requires the exact prior FreshSolveBinding.",
                self.prior_solve_binding,
            )
        _require_text("supplied_input_ref", self.supplied_input_ref)
        if self.restart_policy is not FreshSolveRestartPolicy.REQUIRED:
            raise EscalationContractError(
                "ESCALATION_RESTART_POLICY_INVALID",
                "Every accepted resolution input must require a new Fresh Solve.",
                self.restart_policy,
            )
        expected_target = (
            InputMaterializationTarget.PROJECT_AUTHORITY
            if self.input_kind is ResolutionInputKind.PROJECT_OWNER_INTENT
            else InputMaterializationTarget.EVIDENCE
        )
        if self.materialization_target is not expected_target:
            raise EscalationContractError(
                "ESCALATION_MATERIALIZATION_TARGET_CONFLICT",
                "Resolution input kind must route to its governed materialization boundary.",
                self.materialization_target,
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "request_id": self.request_id,
            "input_kind": self.input_kind.value,
            "materialization_target": self.materialization_target.value,
            "prior_solve_binding": self.prior_solve_binding.as_dict(),
            "supplied_input_ref": self.supplied_input_ref,
            "restart_policy": self.restart_policy.value,
        }
