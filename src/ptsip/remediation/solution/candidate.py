from __future__ import annotations

from copy import deepcopy
from dataclasses import dataclass
from typing import Mapping

from ...evidence.contract import canonical_json, canonical_value
from ...governance.model import GovernanceAuthorityError, SolveSubject


class CandidateGenerationContractError(ValueError):
    """Stable fail-closed error for canonical S1 candidate materialization."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise CandidateGenerationContractError(
            "CANDIDATE_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _canonical_ids(
    name: str,
    values: tuple[str, ...],
    *,
    allow_empty: bool = True,
) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise CandidateGenerationContractError(
            "CANDIDATE_REFERENCE_COLLECTION_TYPE",
            f"{name} must be a tuple of canonical identifiers.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if not allow_empty and not normalized:
        raise CandidateGenerationContractError(
            "CANDIDATE_REFERENCE_COLLECTION_EMPTY",
            f"{name} must contain at least one identifier.",
            values,
        )
    if len(set(normalized)) != len(normalized):
        raise CandidateGenerationContractError(
            "CANDIDATE_REFERENCE_DUPLICATE",
            f"{name} must not contain duplicate identifiers.",
            values,
        )
    return tuple(sorted(normalized))


def _canonical_target_state(value: object) -> object:
    if value is None:
        raise CandidateGenerationContractError(
            "CANDIDATE_TARGET_STATE_INVALID",
            "target_state must be an explicit machine-readable semantic state.",
            value,
        )
    normalized = canonical_value(value)
    try:
        canonical_json(normalized)
    except (TypeError, ValueError) as exc:
        raise CandidateGenerationContractError(
            "CANDIDATE_TARGET_STATE_UNSERIALIZABLE",
            "target_state must be canonically serializable.",
            value,
        ) from exc
    return normalized


@dataclass(frozen=True)
class CandidateSupportingReferences:
    """Sparse canonical support actually consumed by one candidate materialization."""

    rules: tuple[str, ...]
    facts: tuple[str, ...] = ()
    authorities: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        object.__setattr__(
            self,
            "rules",
            _canonical_ids("rules", self.rules, allow_empty=False),
        )
        object.__setattr__(self, "facts", _canonical_ids("facts", self.facts))
        object.__setattr__(
            self,
            "authorities",
            _canonical_ids("authorities", self.authorities),
        )

    def as_dict(self) -> dict[str, list[str]]:
        return {
            "RULE": list(self.rules),
            "FACT": list(self.facts),
            "AUTHORITY": list(self.authorities),
        }


@dataclass(frozen=True)
class AuthorityRequirement:
    """Currently unsatisfied Project Authority requirement; never a guessed authority id."""

    requirement_id: str
    subject_binding: Mapping[str, object]
    required_effects: tuple[str, ...]

    def __post_init__(self) -> None:
        _require_text("requirement_id", self.requirement_id)
        if not isinstance(self.subject_binding, Mapping):
            raise CandidateGenerationContractError(
                "AUTHORITY_REQUIREMENT_SUBJECT_BINDING_INVALID",
                "subject_binding must use the established machine-readable solve-subject contract.",
                self.subject_binding,
            )
        try:
            canonical_subject = SolveSubject.from_mapping(self.subject_binding).as_dict()
        except GovernanceAuthorityError as exc:
            raise CandidateGenerationContractError(
                "AUTHORITY_REQUIREMENT_SUBJECT_BINDING_INVALID",
                "subject_binding must conform to the established governance subject contract.",
                self.subject_binding,
            ) from exc
        object.__setattr__(self, "subject_binding", canonical_subject)
        object.__setattr__(
            self,
            "required_effects",
            _canonical_ids("required_effects", self.required_effects, allow_empty=False),
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "requirement_id": self.requirement_id,
            "subject_binding": deepcopy(dict(self.subject_binding)),
            "required_effects": list(self.required_effects),
        }


@dataclass(frozen=True)
class CandidateProvenance:
    """Canonical SSOT for candidate support and unsatisfied authority requirements."""

    supporting_references: CandidateSupportingReferences
    authority_requirements: tuple[AuthorityRequirement, ...] = ()

    def __post_init__(self) -> None:
        if not isinstance(self.supporting_references, CandidateSupportingReferences):
            raise CandidateGenerationContractError(
                "CANDIDATE_SUPPORTING_REFERENCES_INVALID",
                "supporting_references must use CandidateSupportingReferences.",
                self.supporting_references,
            )
        if not isinstance(self.authority_requirements, tuple) or any(
            not isinstance(item, AuthorityRequirement)
            for item in self.authority_requirements
        ):
            raise CandidateGenerationContractError(
                "AUTHORITY_REQUIREMENT_COLLECTION_INVALID",
                "authority_requirements must contain only AuthorityRequirement values.",
                self.authority_requirements,
            )
        ids = tuple(item.requirement_id for item in self.authority_requirements)
        if len(set(ids)) != len(ids):
            raise CandidateGenerationContractError(
                "AUTHORITY_REQUIREMENT_DUPLICATE",
                "authority_requirements must not duplicate requirement identities.",
                ids,
            )
        object.__setattr__(
            self,
            "authority_requirements",
            tuple(sorted(self.authority_requirements, key=lambda item: item.requirement_id)),
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "supporting_references": self.supporting_references.as_dict(),
            "authority_requirements": [
                item.as_dict() for item in self.authority_requirements
            ],
        }


@dataclass(frozen=True)
class SemanticCandidateProposal:
    """Source-neutral declared meaning before canonical candidate materialization."""

    interpretation_id: str
    rule_id: str
    remediation_family: str
    target_state: object
    supporting_references: CandidateSupportingReferences
    authority_requirements: tuple[AuthorityRequirement, ...] = ()

    def __post_init__(self) -> None:
        interpretation_id = _require_text("interpretation_id", self.interpretation_id)
        rule_id = _require_text("rule_id", self.rule_id)
        _require_text("remediation_family", self.remediation_family)
        prefix = f"interpretation:{rule_id}:"
        if not interpretation_id.startswith(prefix) or interpretation_id == prefix:
            raise CandidateGenerationContractError(
                "CANDIDATE_INTERPRETATION_ID_SCOPE_MISMATCH",
                "interpretation_id must be rule-scoped as interpretation:<RULE_ID>:<LOCAL_INTERPRETATION_ID>.",
                interpretation_id,
            )
        if not isinstance(self.supporting_references, CandidateSupportingReferences):
            raise CandidateGenerationContractError(
                "CANDIDATE_SUPPORTING_REFERENCES_INVALID",
                "proposal supporting_references must use CandidateSupportingReferences.",
                self.supporting_references,
            )
        if rule_id not in self.supporting_references.rules:
            raise CandidateGenerationContractError(
                "CANDIDATE_PRODUCER_RULE_NOT_CONSUMED",
                "The producing rule must appear in CandidateProvenance supporting RULE references.",
                rule_id,
            )
        object.__setattr__(self, "target_state", _canonical_target_state(self.target_state))
        CandidateProvenance(
            supporting_references=self.supporting_references,
            authority_requirements=self.authority_requirements,
        )


@dataclass(frozen=True)
class SemanticCandidate:
    """Canonical S1 candidate. Candidate existence is not authority or elimination."""

    id: str
    rule_id: str
    remediation_family: str
    target_state: object
    provenance: CandidateProvenance

    def __post_init__(self) -> None:
        candidate_id = _require_text("id", self.id)
        rule_id = _require_text("rule_id", self.rule_id)
        _require_text("remediation_family", self.remediation_family)
        prefix = f"interpretation:{rule_id}:"
        if not candidate_id.startswith(prefix) or candidate_id == prefix:
            raise CandidateGenerationContractError(
                "CANDIDATE_ID_SCOPE_MISMATCH",
                "candidate id must preserve the approved rule-scoped interpretation identity.",
                candidate_id,
            )
        if not isinstance(self.provenance, CandidateProvenance):
            raise CandidateGenerationContractError(
                "CANDIDATE_PROVENANCE_INVALID",
                "SemanticCandidate must contain canonical CandidateProvenance.",
                self.provenance,
            )
        if rule_id not in self.provenance.supporting_references.rules:
            raise CandidateGenerationContractError(
                "CANDIDATE_PRODUCER_RULE_NOT_CONSUMED",
                "The producing rule must appear in CandidateProvenance supporting RULE references.",
                rule_id,
            )
        object.__setattr__(self, "target_state", _canonical_target_state(self.target_state))

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "rule_id": self.rule_id,
            "remediation_family": self.remediation_family,
            "target_state": canonical_value(self.target_state),
            "provenance": self.provenance.as_dict(),
        }


def materialize_candidate(proposal: SemanticCandidateProposal) -> SemanticCandidate:
    """Materialize declared meaning without ranking, elimination, authority creation, or mutation."""

    if not isinstance(proposal, SemanticCandidateProposal):
        raise CandidateGenerationContractError(
            "CANDIDATE_PROPOSAL_INVALID",
            "materialize_candidate requires a SemanticCandidateProposal.",
            proposal,
        )
    provenance = CandidateProvenance(
        supporting_references=proposal.supporting_references,
        authority_requirements=proposal.authority_requirements,
    )
    return SemanticCandidate(
        id=proposal.interpretation_id,
        rule_id=proposal.rule_id,
        remediation_family=proposal.remediation_family,
        target_state=proposal.target_state,
        provenance=provenance,
    )


def materialize_candidates(
    proposals: tuple[SemanticCandidateProposal, ...],
) -> tuple[SemanticCandidate, ...]:
    """Materialize a declared candidate set while preserving source order and identity uniqueness."""

    if not isinstance(proposals, tuple):
        raise CandidateGenerationContractError(
            "CANDIDATE_PROPOSAL_COLLECTION_TYPE",
            "proposals must be a tuple.",
            proposals,
        )
    candidates = tuple(materialize_candidate(item) for item in proposals)
    ids = tuple(item.id for item in candidates)
    if len(set(ids)) != len(ids):
        raise CandidateGenerationContractError(
            "CANDIDATE_ID_DUPLICATE",
            "A candidate set must contain unique interpretation identities.",
            ids,
        )
    return candidates
