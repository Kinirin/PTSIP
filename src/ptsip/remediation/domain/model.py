from __future__ import annotations

from dataclasses import asdict, dataclass
from enum import StrEnum

from ...evidence.contract import NormalizedEvidenceSet, canonical_value
from ...specification_binding import SpecificationBinding


class RemediationContractError(ValueError):
    """Stable fail-closed error for malformed remediation domain contracts."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class ResolutionOutcome(StrEnum):
    DETERMINISTIC = "DETERMINISTIC"
    OWNER_INTENT_REQUIRED = "OWNER_INTENT_REQUIRED"
    EXTERNAL_FACT_REQUIRED = "EXTERNAL_FACT_REQUIRED"
    UNSATISFIABLE = "UNSATISFIABLE"
    TOOL_CAPABILITY_GAP = "TOOL_CAPABILITY_GAP"


class MutationClass(StrEnum):
    MECHANICAL_REVERSIBLE = "MECHANICAL_REVERSIBLE"
    STRUCTURAL_SEMANTIC_PRESERVING = "STRUCTURAL_SEMANTIC_PRESERVING"
    ARCHITECTURE_SEMANTIC = "ARCHITECTURE_SEMANTIC"
    DESTRUCTIVE = "DESTRUCTIVE"


class AuthorizationDecision(StrEnum):
    AUTHORIZED = "AUTHORIZED"
    OWNER_CONFIRMATION_REQUIRED = "OWNER_CONFIRMATION_REQUIRED"
    NOT_AUTHORIZED = "NOT_AUTHORIZED"


class RemediationDisposition(StrEnum):
    EXISTING_COMPONENT_CANDIDATE = "EXISTING_COMPONENT_CANDIDATE"
    ASSOCIATED_ARTIFACT_CANDIDATE = "ASSOCIATED_ARTIFACT_CANDIDATE"
    RELOCATION_CANDIDATE = "RELOCATION_CANDIDATE"
    REMOVAL_CANDIDATE = "REMOVAL_CANDIDATE"
    NEW_COMPONENT_CANDIDATE = "NEW_COMPONENT_CANDIDATE"
    DECISION_REQUIRED = "DECISION_REQUIRED"


class ProjectIntentSource(StrEnum):
    PROJECT_PROFILE = "PROJECT_PROFILE"
    OWNER_DECISION = "OWNER_DECISION"
    GOVERNANCE_RECORD = "GOVERNANCE_RECORD"


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise RemediationContractError(
            "REMEDIATION_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _require_ids(name: str, values: tuple[str, ...], *, allow_empty: bool = True) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise RemediationContractError(
            "REMEDIATION_IDS_TYPE",
            f"{name} must be a tuple of canonical identifiers.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if not allow_empty and not normalized:
        raise RemediationContractError(
            "REMEDIATION_IDS_EMPTY",
            f"{name} must contain at least one identifier.",
            values,
        )
    if len(set(normalized)) != len(normalized):
        raise RemediationContractError(
            "REMEDIATION_IDS_DUPLICATE",
            f"{name} must not contain duplicate identifiers.",
            values,
        )
    return normalized


@dataclass(frozen=True)
class RemediationContext:
    evidence: NormalizedEvidenceSet
    specification: SpecificationBinding

    def __post_init__(self) -> None:
        if not isinstance(self.evidence, NormalizedEvidenceSet):
            raise RemediationContractError(
                "REMEDIATION_EVIDENCE_TYPE",
                "RemediationContext must consume the established NormalizedEvidenceSet contract.",
                self.evidence,
            )
        if not isinstance(self.specification, SpecificationBinding):
            raise RemediationContractError(
                "REMEDIATION_SPECIFICATION_TYPE",
                "RemediationContext must consume an explicit SpecificationBinding.",
                self.specification,
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "evidence_digest": self.evidence.deterministic_digest,
            "specification": self.specification.as_dict(),
        }


@dataclass(frozen=True)
class CoverageGap:
    id: str
    code: str
    subject: str
    evidence_ids: tuple[str, ...] = ()
    detail: str | None = None

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("code", self.code)
        _require_text("subject", self.subject)
        _require_ids("evidence_ids", self.evidence_ids)
        if self.detail is not None:
            _require_text("detail", self.detail)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "code": self.code,
            "subject": self.subject,
            "evidence_ids": list(self.evidence_ids),
            "detail": self.detail,
        }


@dataclass(frozen=True)
class DerivedFact:
    id: str
    subject: str
    predicate: str
    value: object
    evidence_ids: tuple[str, ...]

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("subject", self.subject)
        _require_text("predicate", self.predicate)
        if self.value is None:
            raise RemediationContractError(
                "REMEDIATION_FACT_VALUE_INVALID",
                "DerivedFact.value must be an explicit deterministic value.",
                self.value,
            )
        _require_ids("evidence_ids", self.evidence_ids, allow_empty=False)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "subject": self.subject,
            "predicate": self.predicate,
            "value": canonical_value(self.value),
            "evidence_ids": list(self.evidence_ids),
        }


@dataclass(frozen=True)
class NormativeConstraint:
    rule_id: str
    specification: SpecificationBinding
    subject: str
    requirement: str

    def __post_init__(self) -> None:
        _require_text("rule_id", self.rule_id)
        if not isinstance(self.specification, SpecificationBinding):
            raise RemediationContractError(
                "REMEDIATION_SPECIFICATION_TYPE",
                "NormativeConstraint must be bound to an explicit SpecificationBinding.",
                self.specification,
            )
        _require_text("subject", self.subject)
        _require_text("requirement", self.requirement)

    def as_dict(self) -> dict[str, object]:
        return {
            "rule_id": self.rule_id,
            "specification": self.specification.as_dict(),
            "subject": self.subject,
            "requirement": self.requirement,
        }


@dataclass(frozen=True)
class ProjectIntentAuthority:
    id: str
    subject: str
    intent: str
    source: ProjectIntentSource
    source_ref: str

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("subject", self.subject)
        _require_text("intent", self.intent)
        if not isinstance(self.source, ProjectIntentSource):
            raise RemediationContractError(
                "REMEDIATION_INTENT_SOURCE_INVALID",
                "ProjectIntentAuthority requires an explicit project-owned authority source.",
                self.source,
            )
        _require_text("source_ref", self.source_ref)

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["source"] = self.source.value
        return payload


@dataclass(frozen=True)
class RemediationDispositionCandidate:
    id: str
    gap_id: str
    disposition: RemediationDisposition
    rationale: str
    evidence_ids: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("gap_id", self.gap_id)
        if not isinstance(self.disposition, RemediationDisposition):
            raise RemediationContractError(
                "REMEDIATION_DISPOSITION_INVALID",
                "RemediationDispositionCandidate requires a non-authoritative disposition value.",
                self.disposition,
            )
        _require_text("rationale", self.rationale)
        _require_ids("evidence_ids", self.evidence_ids)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "gap_id": self.gap_id,
            "disposition": self.disposition.value,
            "rationale": self.rationale,
            "evidence_ids": list(self.evidence_ids),
        }


@dataclass(frozen=True)
class SemanticCandidate:
    id: str
    rule_id: str
    remediation_family: str
    target_state: object
    fact_ids: tuple[str, ...] = ()
    required_authority_ids: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("rule_id", self.rule_id)
        _require_text("remediation_family", self.remediation_family)
        if self.target_state is None:
            raise RemediationContractError(
                "REMEDIATION_TARGET_STATE_INVALID",
                "SemanticCandidate.target_state must be an explicit modeled semantic state.",
                self.target_state,
            )
        _require_ids("fact_ids", self.fact_ids)
        _require_ids("required_authority_ids", self.required_authority_ids)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "rule_id": self.rule_id,
            "remediation_family": self.remediation_family,
            "target_state": canonical_value(self.target_state),
            "fact_ids": list(self.fact_ids),
            "required_authority_ids": list(self.required_authority_ids),
        }


@dataclass(frozen=True)
class SemanticRemediationPlan:
    id: str
    target_candidate_id: str
    rule_ids: tuple[str, ...]
    fact_ids: tuple[str, ...]
    authority_ids: tuple[str, ...] = ()
    rationale: str | None = None

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("target_candidate_id", self.target_candidate_id)
        _require_ids("rule_ids", self.rule_ids, allow_empty=False)
        _require_ids("fact_ids", self.fact_ids)
        _require_ids("authority_ids", self.authority_ids)
        if self.rationale is not None:
            _require_text("rationale", self.rationale)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "target_candidate_id": self.target_candidate_id,
            "rule_ids": list(self.rule_ids),
            "fact_ids": list(self.fact_ids),
            "authority_ids": list(self.authority_ids),
            "rationale": self.rationale,
        }


@dataclass(frozen=True)
class EscalationProof:
    id: str
    unresolved_dimension: str
    surviving_candidate_ids: tuple[str, ...]
    eliminated_candidate_ids: tuple[str, ...]
    fact_ids: tuple[str, ...]
    constraint_rule_ids: tuple[str, ...]
    available_authority_ids: tuple[str, ...]
    required_input: str

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("unresolved_dimension", self.unresolved_dimension)
        _require_ids("surviving_candidate_ids", self.surviving_candidate_ids, allow_empty=False)
        _require_ids("eliminated_candidate_ids", self.eliminated_candidate_ids)
        _require_ids("fact_ids", self.fact_ids)
        _require_ids("constraint_rule_ids", self.constraint_rule_ids)
        _require_ids("available_authority_ids", self.available_authority_ids)
        _require_text("required_input", self.required_input)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "unresolved_dimension": self.unresolved_dimension,
            "surviving_candidate_ids": list(self.surviving_candidate_ids),
            "eliminated_candidate_ids": list(self.eliminated_candidate_ids),
            "fact_ids": list(self.fact_ids),
            "constraint_rule_ids": list(self.constraint_rule_ids),
            "available_authority_ids": list(self.available_authority_ids),
            "required_input": self.required_input,
        }


@dataclass(frozen=True)
class RepositoryChangePlan:
    id: str
    semantic_plan_id: str
    mutation_class: MutationClass
    prestate_fingerprint: str
    operation_ids: tuple[str, ...]

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("semantic_plan_id", self.semantic_plan_id)
        if not isinstance(self.mutation_class, MutationClass):
            raise RemediationContractError(
                "REMEDIATION_MUTATION_CLASS_INVALID",
                "RepositoryChangePlan requires an explicit MutationClass.",
                self.mutation_class,
            )
        _require_text("prestate_fingerprint", self.prestate_fingerprint)
        _require_ids("operation_ids", self.operation_ids, allow_empty=False)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "semantic_plan_id": self.semantic_plan_id,
            "mutation_class": self.mutation_class.value,
            "prestate_fingerprint": self.prestate_fingerprint,
            "operation_ids": list(self.operation_ids),
        }


@dataclass(frozen=True)
class PostconditionResult:
    semantic_plan_id: str
    satisfied: bool
    verified_rule_ids: tuple[str, ...]
    failed_check_ids: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        _require_text("semantic_plan_id", self.semantic_plan_id)
        if not isinstance(self.satisfied, bool):
            raise RemediationContractError(
                "REMEDIATION_POSTCONDITION_STATUS_INVALID",
                "PostconditionResult.satisfied must be boolean.",
                self.satisfied,
            )
        _require_ids("verified_rule_ids", self.verified_rule_ids)
        _require_ids("failed_check_ids", self.failed_check_ids)
        if self.satisfied and self.failed_check_ids:
            raise RemediationContractError(
                "REMEDIATION_POSTCONDITION_CONFLICT",
                "A satisfied postcondition result cannot contain failed checks.",
                self.failed_check_ids,
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "semantic_plan_id": self.semantic_plan_id,
            "satisfied": self.satisfied,
            "verified_rule_ids": list(self.verified_rule_ids),
            "failed_check_ids": list(self.failed_check_ids),
        }
