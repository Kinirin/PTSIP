from __future__ import annotations

import re
from dataclasses import dataclass
from enum import StrEnum

from ...evidence.contract import stable_digest
from ...specification_binding import SpecificationBinding
from ..domain import RemediationContext, SemanticCandidate


_SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")


class EliminationContractError(ValueError):
    """Stable fail-closed error for malformed or stale elimination contracts."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class EliminationReasonFamily(StrEnum):
    VIOLATES_NORMATIVE_CONSTRAINT = "VIOLATES_NORMATIVE_CONSTRAINT"
    CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY = "CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY"
    FAILS_REQUIRED_PRECONDITION = "FAILS_REQUIRED_PRECONDITION"
    BREAKS_REQUIRED_ARCHITECTURE_INVARIANT = "BREAKS_REQUIRED_ARCHITECTURE_INVARIANT"
    PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE = (
        "PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE"
    )
    PROVABLY_DOMINATED_UNDER_MODELED_SEMANTIC_OBJECTIVE = (
        "PROVABLY_DOMINATED_UNDER_MODELED_SEMANTIC_OBJECTIVE"
    )


class EliminationReferenceKind(StrEnum):
    RULE = "RULE"
    FACT = "FACT"
    AUTHORITY = "AUTHORITY"
    PRECONDITION = "PRECONDITION"
    ARCHITECTURE_INVARIANT = "ARCHITECTURE_INVARIANT"
    CANDIDATE = "CANDIDATE"
    SEMANTIC_OBJECTIVE = "SEMANTIC_OBJECTIVE"


_REQUIRED_REFERENCE_KINDS = {
    EliminationReasonFamily.VIOLATES_NORMATIVE_CONSTRAINT: frozenset(
        {EliminationReferenceKind.RULE}
    ),
    EliminationReasonFamily.CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY: frozenset(
        {EliminationReferenceKind.AUTHORITY}
    ),
    EliminationReasonFamily.FAILS_REQUIRED_PRECONDITION: frozenset(
        {EliminationReferenceKind.PRECONDITION}
    ),
    EliminationReasonFamily.BREAKS_REQUIRED_ARCHITECTURE_INVARIANT: frozenset(
        {EliminationReferenceKind.ARCHITECTURE_INVARIANT}
    ),
    EliminationReasonFamily.PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE: frozenset(
        {EliminationReferenceKind.CANDIDATE}
    ),
    EliminationReasonFamily.PROVABLY_DOMINATED_UNDER_MODELED_SEMANTIC_OBJECTIVE: frozenset(
        {
            EliminationReferenceKind.CANDIDATE,
            EliminationReferenceKind.SEMANTIC_OBJECTIVE,
        }
    ),
}


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise EliminationContractError(
            "ELIMINATION_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _canonical_text_ids(
    name: str,
    values: tuple[str, ...],
    *,
    allow_empty: bool,
) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise EliminationContractError(
            "ELIMINATION_IDS_TYPE",
            f"{name} must be a tuple of canonical identifiers.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if not allow_empty and not normalized:
        raise EliminationContractError(
            "ELIMINATION_IDS_EMPTY",
            f"{name} must contain at least one identifier.",
            values,
        )
    if len(set(normalized)) != len(normalized):
        raise EliminationContractError(
            "ELIMINATION_IDS_DUPLICATE",
            f"{name} must not contain duplicate identifiers.",
            values,
        )
    return tuple(sorted(normalized))


@dataclass(frozen=True, order=True)
class EliminationReference:
    """Typed reference proving why a candidate is no longer legal."""

    kind: EliminationReferenceKind
    identity: str

    def __post_init__(self) -> None:
        if not isinstance(self.kind, EliminationReferenceKind):
            raise EliminationContractError(
                "ELIMINATION_REFERENCE_KIND_INVALID",
                "EliminationReference.kind must use the closed reference vocabulary.",
                self.kind,
            )
        _require_text("identity", self.identity)

    def as_dict(self) -> dict[str, str]:
        return {"kind": self.kind.value, "identity": self.identity}


@dataclass(frozen=True)
class EliminationSolveBinding:
    """Exact evidence/specification/consumed-authority identity for one solve."""

    evidence_identity: str
    specification_identity: SpecificationBinding
    consumed_authority_identities: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        evidence_identity = _require_text("evidence_identity", self.evidence_identity)
        if _SHA256_PATTERN.fullmatch(evidence_identity) is None:
            raise EliminationContractError(
                "ELIMINATION_EVIDENCE_IDENTITY_INVALID",
                "evidence_identity must be the NormalizedEvidenceSet deterministic SHA-256 digest.",
                evidence_identity,
            )
        if not isinstance(self.specification_identity, SpecificationBinding):
            raise EliminationContractError(
                "ELIMINATION_SPECIFICATION_IDENTITY_INVALID",
                "specification_identity must be an explicit SpecificationBinding.",
                self.specification_identity,
            )
        authority_ids = _canonical_text_ids(
            "consumed_authority_identities",
            self.consumed_authority_identities,
            allow_empty=True,
        )
        object.__setattr__(self, "consumed_authority_identities", authority_ids)

    @classmethod
    def from_context(
        cls,
        context: RemediationContext,
        *,
        consumed_authority_identities: tuple[str, ...] = (),
    ) -> "EliminationSolveBinding":
        if not isinstance(context, RemediationContext):
            raise EliminationContractError(
                "ELIMINATION_REMEDIATION_CONTEXT_REQUIRED",
                "Fresh solve binding must reuse the existing RemediationContext.",
                context,
            )
        return cls(
            evidence_identity=context.evidence.deterministic_digest,
            specification_identity=context.specification,
            consumed_authority_identities=consumed_authority_identities,
        )

    @property
    def id(self) -> str:
        return f"solve-binding:{stable_digest(self.as_dict())[:24]}"

    def as_dict(self) -> dict[str, object]:
        return {
            "evidence_identity": self.evidence_identity,
            "specification_identity": self.specification_identity.as_dict(),
            "consumed_authority_identities": list(self.consumed_authority_identities),
        }


@dataclass(frozen=True)
class EliminationRecord:
    """Canonical non-authoritative elimination proof bound to one current solve."""

    candidate_id: str
    reason_family: EliminationReasonFamily
    solve_binding: EliminationSolveBinding
    supporting_references: tuple[EliminationReference, ...]
    retained_candidate_id: str | None = None
    detail: str | None = None

    def __post_init__(self) -> None:
        _require_text("candidate_id", self.candidate_id)
        if not isinstance(self.reason_family, EliminationReasonFamily):
            raise EliminationContractError(
                "ELIMINATION_REASON_FAMILY_INVALID",
                "reason_family must use the closed elimination reason vocabulary.",
                self.reason_family,
            )
        if not isinstance(self.solve_binding, EliminationSolveBinding):
            raise EliminationContractError(
                "ELIMINATION_SOLVE_BINDING_REQUIRED",
                "EliminationRecord must carry an exact fresh-solve binding.",
                self.solve_binding,
            )
        if not isinstance(self.supporting_references, tuple) or not self.supporting_references:
            raise EliminationContractError(
                "ELIMINATION_REFERENCES_REQUIRED",
                "Every elimination must carry machine-inspectable supporting references.",
                self.supporting_references,
            )
        if any(not isinstance(item, EliminationReference) for item in self.supporting_references):
            raise EliminationContractError(
                "ELIMINATION_REFERENCE_TYPE_INVALID",
                "supporting_references must contain only EliminationReference values.",
                self.supporting_references,
            )
        if len(set(self.supporting_references)) != len(self.supporting_references):
            raise EliminationContractError(
                "ELIMINATION_REFERENCE_DUPLICATE",
                "supporting_references must not contain duplicate references.",
                self.supporting_references,
            )
        object.__setattr__(
            self,
            "supporting_references",
            tuple(sorted(self.supporting_references)),
        )

        kinds = {item.kind for item in self.supporting_references}
        required = _REQUIRED_REFERENCE_KINDS[self.reason_family]
        missing = required - kinds
        if missing:
            raise EliminationContractError(
                "ELIMINATION_REQUIRED_REFERENCE_KIND_MISSING",
                "Elimination reason is missing required machine-inspectable reference kinds.",
                tuple(sorted(item.value for item in missing)),
            )

        authority_identities = set(self.solve_binding.consumed_authority_identities)
        for reference in self.supporting_references:
            if (
                reference.kind is EliminationReferenceKind.AUTHORITY
                and reference.identity not in authority_identities
            ):
                raise EliminationContractError(
                    "ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE",
                    "Authority references must identify authority actually consumed by this solve.",
                    reference.identity,
                )

        comparative_reason = self.reason_family in {
            EliminationReasonFamily.PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE,
            EliminationReasonFamily.PROVABLY_DOMINATED_UNDER_MODELED_SEMANTIC_OBJECTIVE,
        }
        if comparative_reason:
            if self.retained_candidate_id is None:
                raise EliminationContractError(
                    "ELIMINATION_RETAINED_CANDIDATE_REQUIRED",
                    "Equivalence or dominance elimination must identify the retained candidate.",
                    self,
                )
            retained = _require_text("retained_candidate_id", self.retained_candidate_id)
            if retained == self.candidate_id:
                raise EliminationContractError(
                    "ELIMINATION_SELF_COMPARISON_INVALID",
                    "A candidate cannot eliminate itself by equivalence or dominance.",
                    retained,
                )
            candidate_references = {
                item.identity
                for item in self.supporting_references
                if item.kind is EliminationReferenceKind.CANDIDATE
            }
            if retained not in candidate_references:
                raise EliminationContractError(
                    "ELIMINATION_RETAINED_CANDIDATE_REFERENCE_REQUIRED",
                    "Comparative elimination must reference the retained candidate explicitly.",
                    retained,
                )
        elif self.retained_candidate_id is not None:
            raise EliminationContractError(
                "ELIMINATION_RETAINED_CANDIDATE_UNEXPECTED",
                "retained_candidate_id is valid only for equivalence or dominance elimination.",
                self.retained_candidate_id,
            )

        if self.detail is not None:
            _require_text("detail", self.detail)

    @property
    def id(self) -> str:
        identity_payload = {
            "candidate_id": self.candidate_id,
            "reason_family": self.reason_family.value,
            "solve_binding_id": self.solve_binding.id,
            "supporting_references": [
                item.as_dict() for item in self.supporting_references
            ],
            "retained_candidate_id": self.retained_candidate_id,
        }
        return f"elimination:{stable_digest(identity_payload)[:24]}"

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "candidate_id": self.candidate_id,
            "reason_family": self.reason_family.value,
            "solve_binding": self.solve_binding.as_dict(),
            "solve_binding_id": self.solve_binding.id,
            "supporting_references": [
                item.as_dict() for item in self.supporting_references
            ],
            "retained_candidate_id": self.retained_candidate_id,
            "detail": self.detail,
            "project_authority": False,
        }


@dataclass(frozen=True)
class SurvivorReductionResult:
    """S2 handoff: generated candidates minus valid bound eliminations."""

    solve_binding: EliminationSolveBinding
    survivors: tuple[SemanticCandidate, ...]
    elimination_records: tuple[EliminationRecord, ...]

    @property
    def survivor_candidate_ids(self) -> tuple[str, ...]:
        return tuple(candidate.id for candidate in self.survivors)

    @property
    def eliminated_candidate_ids(self) -> tuple[str, ...]:
        return tuple(
            sorted({record.candidate_id for record in self.elimination_records})
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "solve_binding": self.solve_binding.as_dict(),
            "solve_binding_id": self.solve_binding.id,
            "survivor_candidate_ids": list(self.survivor_candidate_ids),
            "eliminated_candidate_ids": list(self.eliminated_candidate_ids),
            "elimination_records": [
                record.as_dict() for record in self.elimination_records
            ],
        }


def reduce_survivors(
    candidates: tuple[SemanticCandidate, ...],
    elimination_records: tuple[EliminationRecord, ...],
    *,
    solve_binding: EliminationSolveBinding,
) -> SurvivorReductionResult:
    """Reduce only by canonical records valid for the exact current solve."""

    if not isinstance(candidates, tuple) or any(
        not isinstance(candidate, SemanticCandidate) for candidate in candidates
    ):
        raise EliminationContractError(
            "ELIMINATION_CANDIDATES_TYPE",
            "candidates must be a tuple of SemanticCandidate values.",
            candidates,
        )
    if not isinstance(elimination_records, tuple) or any(
        not isinstance(record, EliminationRecord) for record in elimination_records
    ):
        raise EliminationContractError(
            "ELIMINATION_RECORDS_TYPE",
            "elimination_records must be a tuple of EliminationRecord values.",
            elimination_records,
        )
    if not isinstance(solve_binding, EliminationSolveBinding):
        raise EliminationContractError(
            "ELIMINATION_SOLVE_BINDING_REQUIRED",
            "Survivor reduction requires the exact current solve binding.",
            solve_binding,
        )

    candidate_by_id: dict[str, SemanticCandidate] = {}
    for candidate in candidates:
        if candidate.id in candidate_by_id:
            raise EliminationContractError(
                "ELIMINATION_CANDIDATE_ID_DUPLICATE",
                "Candidate IDs must be unique within one solve.",
                candidate.id,
            )
        candidate_by_id[candidate.id] = candidate

    record_ids: set[str] = set()
    eliminated_ids: set[str] = set()
    for record in elimination_records:
        if record.id in record_ids:
            raise EliminationContractError(
                "ELIMINATION_RECORD_DUPLICATE",
                "The same canonical EliminationRecord cannot be supplied twice.",
                record.id,
            )
        record_ids.add(record.id)

        if record.solve_binding != solve_binding:
            raise EliminationContractError(
                "ELIMINATION_SOLVE_BINDING_MISMATCH",
                "Changed evidence, specification, or consumed authority identity requires a fresh solve.",
                {
                    "record_id": record.id,
                    "record_solve_binding_id": record.solve_binding.id,
                    "current_solve_binding_id": solve_binding.id,
                },
            )
        if record.candidate_id not in candidate_by_id:
            raise EliminationContractError(
                "ELIMINATION_CANDIDATE_UNKNOWN",
                "An EliminationRecord may only remove a candidate generated in the bound current solve.",
                record.candidate_id,
            )
        if (
            record.retained_candidate_id is not None
            and record.retained_candidate_id not in candidate_by_id
        ):
            raise EliminationContractError(
                "ELIMINATION_RETAINED_CANDIDATE_UNKNOWN",
                "Comparative elimination must retain a candidate generated in the same solve.",
                record.retained_candidate_id,
            )
        eliminated_ids.add(record.candidate_id)

    for record in elimination_records:
        if (
            record.retained_candidate_id is not None
            and record.retained_candidate_id in eliminated_ids
        ):
            raise EliminationContractError(
                "ELIMINATION_RETAINED_CANDIDATE_NOT_SURVIVOR",
                "Equivalence or dominance proof cannot claim a retained candidate that is also eliminated.",
                {
                    "record_id": record.id,
                    "retained_candidate_id": record.retained_candidate_id,
                },
            )

    survivors = tuple(
        sorted(
            (
                candidate
                for candidate in candidates
                if candidate.id not in eliminated_ids
            ),
            key=lambda candidate: candidate.id,
        )
    )
    records = tuple(sorted(elimination_records, key=lambda record: record.id))

    return SurvivorReductionResult(
        solve_binding=solve_binding,
        survivors=survivors,
        elimination_records=records,
    )
