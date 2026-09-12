from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Mapping
import re

from ...evidence.contract import stable_digest
from ...governance import (
    EligibilityResult,
    EligibilityStatus,
    ProjectAuthorityRecord,
    SubjectMatchKind,
)


class CapabilityRecoveryContractError(ValueError):
    """Stable fail-closed error for S3 capability-recovery contracts."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class CapabilityRecoveryPath(StrEnum):
    LOSSLESS_CANONICAL_NORMALIZATION = "LOSSLESS_CANONICAL_NORMALIZATION"
    PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER = (
        "PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER"
    )
    EXISTING_AUTHORITY_COMPATIBLE_MATERIALIZATION = (
        "EXISTING_AUTHORITY_COMPATIBLE_MATERIALIZATION"
    )


class CapabilityRecoveryAttemptStatus(StrEnum):
    RECOVERED = "RECOVERED"
    REJECTED = "REJECTED"


_RECOVERY_ORDER = (
    CapabilityRecoveryPath.LOSSLESS_CANONICAL_NORMALIZATION,
    CapabilityRecoveryPath.PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER,
    CapabilityRecoveryPath.EXISTING_AUTHORITY_COMPATIBLE_MATERIALIZATION,
)
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_SUBJECT_MATCHES = {
    SubjectMatchKind.EXACT,
    SubjectMatchKind.REGISTERED_MACHINE_RELATIONSHIP,
}


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _require_text_tuple(
    name: str,
    values: tuple[str, ...],
    *,
    allow_empty: bool = True,
) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_TUPLE_REQUIRED",
            f"{name} must be a tuple of canonical strings.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if not allow_empty and not normalized:
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_TUPLE_EMPTY",
            f"{name} must contain at least one value.",
            values,
        )
    if len(set(normalized)) != len(normalized):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_DUPLICATE_VALUE",
            f"{name} must not contain duplicate values.",
            values,
        )
    return normalized


def _require_sha256(name: str, value: object) -> str:
    value = _require_text(name, value)
    if not _SHA256.fullmatch(value):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_DIGEST_INVALID",
            f"{name} must be a lowercase SHA-256 digest.",
            value,
        )
    return value


def _looks_like_physical_authority_identity(value: str) -> bool:
    if "/" in value or "\\" in value:
        return True
    if value in {"HEAD", "FETCH_HEAD", "ORIG_HEAD"}:
        return True
    return bool(re.fullmatch(r"[0-9a-fA-F]{7,64}", value))


def _require_logical_authority_ids(values: tuple[str, ...]) -> tuple[str, ...]:
    normalized = _require_text_tuple("consumed_project_authority_ids", values)
    invalid = tuple(
        value for value in normalized if _looks_like_physical_authority_identity(value)
    )
    if invalid:
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_AUTHORITY_IDENTITY_INVALID",
            "Canonical authority identity must not be a filesystem path or Git revision.",
            invalid,
        )
    return normalized


@dataclass(frozen=True)
class AuthorityRequirementView:
    """Read-only S3 view over S1 CandidateProvenance.authority_requirements.

    This is deliberately not a canonical AuthorityRequirement definition. The
    canonical SSOT remains the upstream CandidateProvenance entry.
    """

    requirement_id: str
    subject_binding: Mapping[str, object]
    required_effects: tuple[str, ...]

    @classmethod
    def from_mapping(cls, value: Mapping[str, object]) -> "AuthorityRequirementView":
        if not isinstance(value, Mapping):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AUTHORITY_REQUIREMENT_INVALID",
                "Authority requirement must be a machine-readable mapping from CandidateProvenance.",
                value,
            )
        requirement_id = _require_text("requirement_id", value.get("requirement_id"))
        subject_binding = value.get("subject_binding")
        if not isinstance(subject_binding, Mapping) or not subject_binding:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_SUBJECT_BINDING_INVALID",
                "Authority requirement subject_binding must be a non-empty mapping.",
                subject_binding,
            )
        raw_effects = value.get("required_effects")
        if isinstance(raw_effects, list):
            raw_effects = tuple(raw_effects)
        effects = _require_text_tuple(
            "required_effects",
            raw_effects,
            allow_empty=False,
        )
        return cls(
            requirement_id=requirement_id,
            subject_binding=dict(subject_binding),
            required_effects=tuple(sorted(effects)),
        )

    @property
    def subject_binding_digest(self) -> str:
        return stable_digest(dict(self.subject_binding))

    def as_reference_dict(self) -> dict[str, object]:
        return {
            "requirement_id": self.requirement_id,
            "subject_binding_digest": self.subject_binding_digest,
            "required_effects": list(self.required_effects),
        }


def authority_requirement_view(value: Mapping[str, object]) -> AuthorityRequirementView:
    """Parse without taking canonical ownership of an S1 authority requirement."""
    return AuthorityRequirementView.from_mapping(value)


@dataclass(frozen=True)
class AuthorityRequirementEligibilityBinding:
    """Non-authoritative transport proof binding eligibility to one requirement subject."""

    authority: ProjectAuthorityRecord
    eligibility: EligibilityResult
    evaluated_subject_binding_digest: str

    def __post_init__(self) -> None:
        if not isinstance(self.authority, ProjectAuthorityRecord):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_PROJECT_AUTHORITY_REQUIRED",
                "Eligibility binding requires a canonical ProjectAuthorityRecord.",
                self.authority,
            )
        if not isinstance(self.eligibility, EligibilityResult):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ELIGIBILITY_REQUIRED",
                "Eligibility binding requires a governance EligibilityResult.",
                self.eligibility,
            )
        _require_sha256(
            "evaluated_subject_binding_digest",
            self.evaluated_subject_binding_digest,
        )
        if self.eligibility.authority_id != self.authority.authority_id:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AUTHORITY_ELIGIBILITY_MISMATCH",
                "EligibilityResult must refer to the same Project Authority.",
                (
                    self.eligibility.authority_id,
                    self.authority.authority_id,
                ),
            )
        source_ref = self.authority.authority_provenance.get("source_ref")
        if self.eligibility.source_ref != source_ref:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AUTHORITY_SOURCE_MISMATCH",
                "EligibilityResult source_ref must match Project Authority provenance.",
                (self.eligibility.source_ref, source_ref),
            )


def authority_requirement_ids_from_candidate_provenance(
    candidate_provenance: Mapping[str, object],
) -> tuple[str, ...]:
    """Deterministic non-authoritative projection of canonical requirement IDs."""
    if not isinstance(candidate_provenance, Mapping):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_CANDIDATE_PROVENANCE_INVALID",
            "CandidateProvenance must be a machine-readable mapping.",
            candidate_provenance,
        )
    raw = candidate_provenance.get("authority_requirements", ())
    if isinstance(raw, list):
        raw = tuple(raw)
    if not isinstance(raw, tuple):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_AUTHORITY_REQUIREMENTS_INVALID",
            "CandidateProvenance.authority_requirements must be an ordered collection.",
            raw,
        )
    views = tuple(authority_requirement_view(item) for item in raw)
    ids = tuple(view.requirement_id for view in views)
    if len(set(ids)) != len(ids):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_AUTHORITY_REQUIREMENT_DUPLICATE",
            "CandidateProvenance must not contain duplicate authority requirement identities.",
            ids,
        )
    return tuple(sorted(ids))


@dataclass(frozen=True)
class RecoveredCapability:
    """Recovered semantics plus recovery trace; never a CandidateProvenance SSOT."""

    required_capability: str
    subject: str
    canonical_key: str
    canonical_value: str
    closed_vocabulary: tuple[str, ...]
    source_path: CapabilityRecoveryPath
    authority_requirement_id: str | None = None
    consumed_project_authority_ids: tuple[str, ...] = ()
    materialization_ref: str | None = None

    def __post_init__(self) -> None:
        _require_text("required_capability", self.required_capability)
        _require_text("subject", self.subject)
        _require_text("canonical_key", self.canonical_key)
        _require_text("canonical_value", self.canonical_value)
        vocabulary = _require_text_tuple(
            "closed_vocabulary",
            self.closed_vocabulary,
            allow_empty=False,
        )
        if self.canonical_value not in vocabulary:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_VALUE_OUTSIDE_VOCABULARY",
                "canonical_value must belong to the declared closed vocabulary.",
                self.canonical_value,
            )
        if not isinstance(self.source_path, CapabilityRecoveryPath):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_PATH_INVALID",
                "source_path must be an explicit CapabilityRecoveryPath.",
                self.source_path,
            )
        authority_ids = _require_logical_authority_ids(
            self.consumed_project_authority_ids
        )
        if (
            self.source_path
            is CapabilityRecoveryPath.PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER
        ):
            if self.authority_requirement_id is None:
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_AUTHORITY_REQUIREMENT_REQUIRED",
                    "Authority compatibility recovery must bind the upstream requirement identity.",
                    self.authority_requirement_id,
                )
            _require_text("authority_requirement_id", self.authority_requirement_id)
            if len(authority_ids) != 1:
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_AUTHORITY_CARDINALITY_INVALID",
                    "One exact compatibility recovery must record exactly one actually consumed Project Authority.",
                    authority_ids,
                )
        elif self.authority_requirement_id is not None:
            _require_text("authority_requirement_id", self.authority_requirement_id)
        if self.materialization_ref is not None:
            _require_text("materialization_ref", self.materialization_ref)

    def as_dict(self) -> dict[str, object]:
        return {
            "required_capability": self.required_capability,
            "subject": self.subject,
            "canonical_key": self.canonical_key,
            "canonical_value": self.canonical_value,
            "closed_vocabulary": list(self.closed_vocabulary),
            "source_path": self.source_path.value,
            "authority_requirement_id": self.authority_requirement_id,
            "consumed_project_authority_ids": list(
                self.consumed_project_authority_ids
            ),
            "materialization_ref": self.materialization_ref,
        }


@dataclass(frozen=True)
class CapabilityRecoveryAttempt:
    path: CapabilityRecoveryPath
    status: CapabilityRecoveryAttemptStatus
    reason_code: str
    reason: str
    recovered_capability: RecoveredCapability | None = None

    def __post_init__(self) -> None:
        if not isinstance(self.path, CapabilityRecoveryPath):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_PATH_INVALID",
                "path must be an explicit CapabilityRecoveryPath.",
                self.path,
            )
        if not isinstance(self.status, CapabilityRecoveryAttemptStatus):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_STATUS_INVALID",
                "status must be an explicit CapabilityRecoveryAttemptStatus.",
                self.status,
            )
        _require_text("reason_code", self.reason_code)
        _require_text("reason", self.reason)
        if self.status is CapabilityRecoveryAttemptStatus.RECOVERED:
            if not isinstance(self.recovered_capability, RecoveredCapability):
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_RESULT_REQUIRED",
                    "A RECOVERED attempt must contain RecoveredCapability.",
                    self.recovered_capability,
                )
            if self.recovered_capability.source_path is not self.path:
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_RESULT_PATH_MISMATCH",
                    "RecoveredCapability.source_path must match the attempt path.",
                    self.recovered_capability.source_path,
                )
        elif self.recovered_capability is not None:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_REJECTED_RESULT_CONFLICT",
                "A REJECTED attempt cannot contain recovered semantics.",
                self.recovered_capability,
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "path": self.path.value,
            "status": self.status.value,
            "reason_code": self.reason_code,
            "reason": self.reason,
            "recovered_capability": (
                self.recovered_capability.as_dict()
                if self.recovered_capability
                else None
            ),
        }


@dataclass(frozen=True)
class CapabilityRecoveryAssessment:
    """Ordered proof that safe recovery succeeded or was exhausted."""

    required_capability: str
    subject: str
    authority_requirement_id: str | None = None
    authority_requirement_subject_binding_digest: str | None = None
    attempts: tuple[CapabilityRecoveryAttempt, ...] = ()

    def __post_init__(self) -> None:
        _require_text("required_capability", self.required_capability)
        _require_text("subject", self.subject)
        if self.authority_requirement_id is not None:
            _require_text("authority_requirement_id", self.authority_requirement_id)
            _require_sha256(
                "authority_requirement_subject_binding_digest",
                self.authority_requirement_subject_binding_digest,
            )
        elif self.authority_requirement_subject_binding_digest is not None:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_REQUIREMENT_BINDING_CONFLICT",
                "A requirement subject-binding digest cannot exist without requirement identity.",
                self.authority_requirement_subject_binding_digest,
            )
        if not isinstance(self.attempts, tuple) or any(
            not isinstance(item, CapabilityRecoveryAttempt) for item in self.attempts
        ):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ATTEMPTS_INVALID",
                "attempts must be a tuple of CapabilityRecoveryAttempt values.",
                self.attempts,
            )
        paths = tuple(item.path for item in self.attempts)
        if paths != _RECOVERY_ORDER[: len(paths)]:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ORDER_INVALID",
                "Recovery must follow normalization, adapter, materialization order.",
                paths,
            )
        successes = tuple(
            item
            for item in self.attempts
            if item.status is CapabilityRecoveryAttemptStatus.RECOVERED
        )
        if len(successes) > 1 or (
            successes and successes[0] is not self.attempts[-1]
        ):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AFTER_SUCCESS",
                "Recovery must stop immediately after the first successful path.",
                self.attempts,
            )
        for success in successes:
            recovered = success.recovered_capability
            assert recovered is not None
            if recovered.required_capability != self.required_capability:
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_CAPABILITY_MISMATCH",
                    "Recovered semantics must match the required capability.",
                    recovered.required_capability,
                )
            if recovered.subject != self.subject:
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_SUBJECT_MISMATCH",
                    "Recovered semantics must preserve the unresolved subject binding.",
                    recovered.subject,
                )
            if (
                recovered.authority_requirement_id is not None
                and recovered.authority_requirement_id
                != self.authority_requirement_id
            ):
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_REQUIREMENT_MISMATCH",
                    "Recovered semantics must preserve the bound authority requirement.",
                    recovered.authority_requirement_id,
                )

    @property
    def recovered(self) -> bool:
        return bool(self.attempts) and (
            self.attempts[-1].status is CapabilityRecoveryAttemptStatus.RECOVERED
        )

    @property
    def exhausted(self) -> bool:
        return len(self.attempts) == len(_RECOVERY_ORDER) and all(
            item.status is CapabilityRecoveryAttemptStatus.REJECTED
            for item in self.attempts
        )

    @property
    def next_path(self) -> CapabilityRecoveryPath | None:
        if self.recovered or self.exhausted:
            return None
        return _RECOVERY_ORDER[len(self.attempts)]

    @property
    def recovered_capability(self) -> RecoveredCapability | None:
        if not self.recovered:
            return None
        return self.attempts[-1].recovered_capability

    def append(
        self,
        attempt: CapabilityRecoveryAttempt,
    ) -> "CapabilityRecoveryAssessment":
        if self.next_path is None:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ALREADY_COMPLETE",
                "No recovery path remains after success or exhaustion.",
                self.as_dict(),
            )
        if attempt.path is not self.next_path:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ORDER_INVALID",
                "Attempt does not match the required next recovery path.",
                attempt.path,
            )
        return CapabilityRecoveryAssessment(
            required_capability=self.required_capability,
            subject=self.subject,
            authority_requirement_id=self.authority_requirement_id,
            authority_requirement_subject_binding_digest=(
                self.authority_requirement_subject_binding_digest
            ),
            attempts=self.attempts + (attempt,),
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "required_capability": self.required_capability,
            "subject": self.subject,
            "authority_requirement_id": self.authority_requirement_id,
            "authority_requirement_subject_binding_digest": (
                self.authority_requirement_subject_binding_digest
            ),
            "attempts": [item.as_dict() for item in self.attempts],
            "recovered": self.recovered,
            "exhausted": self.exhausted,
            "next_path": self.next_path.value if self.next_path else None,
        }


def start_capability_recovery(
    *,
    required_capability: str,
    subject: str,
    authority_requirement: Mapping[str, object] | None = None,
) -> CapabilityRecoveryAssessment:
    requirement = (
        authority_requirement_view(authority_requirement)
        if authority_requirement is not None
        else None
    )
    return CapabilityRecoveryAssessment(
        required_capability=required_capability,
        subject=subject,
        authority_requirement_id=(
            requirement.requirement_id if requirement is not None else None
        ),
        authority_requirement_subject_binding_digest=(
            requirement.subject_binding_digest if requirement is not None else None
        ),
    )


def reject_next_recovery_path(
    assessment: CapabilityRecoveryAssessment,
    *,
    reason_code: str,
    reason: str,
) -> CapabilityRecoveryAssessment:
    if not isinstance(assessment, CapabilityRecoveryAssessment):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_ASSESSMENT_REQUIRED",
            "Recovery rejection requires CapabilityRecoveryAssessment.",
            assessment,
        )
    if assessment.next_path is None:
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_ALREADY_COMPLETE",
            "No recovery path remains to reject.",
            assessment.as_dict(),
        )
    return assessment.append(
        CapabilityRecoveryAttempt(
            path=assessment.next_path,
            status=CapabilityRecoveryAttemptStatus.REJECTED,
            reason_code=reason_code,
            reason=reason,
        )
    )


def recover_on_next_path(
    assessment: CapabilityRecoveryAssessment,
    *,
    recovered_capability: RecoveredCapability,
    reason_code: str,
    reason: str,
) -> CapabilityRecoveryAssessment:
    if not isinstance(assessment, CapabilityRecoveryAssessment):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_ASSESSMENT_REQUIRED",
            "Recovery success requires CapabilityRecoveryAssessment.",
            assessment,
        )
    if assessment.next_path is None:
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_ALREADY_COMPLETE",
            "No recovery path remains to recover.",
            assessment.as_dict(),
        )
    if recovered_capability.source_path is not assessment.next_path:
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_ORDER_INVALID",
            "Recovered semantics must come from the currently required recovery path.",
            recovered_capability.source_path,
        )
    return assessment.append(
        CapabilityRecoveryAttempt(
            path=assessment.next_path,
            status=CapabilityRecoveryAttemptStatus.RECOVERED,
            reason_code=reason_code,
            reason=reason,
            recovered_capability=recovered_capability,
        )
    )


def _authority_effects(authority: ProjectAuthorityRecord) -> tuple[str, ...]:
    raw = authority.authority_role.get("effects")
    if isinstance(raw, list):
        raw = tuple(raw)
    return tuple(
        sorted(
            _require_text_tuple(
                "authority_role.effects",
                raw,
                allow_empty=False,
            )
        )
    )


def _eligible_for_requirement(
    *,
    binding: AuthorityRequirementEligibilityBinding,
    requirement: AuthorityRequirementView,
) -> bool:
    if (
        binding.evaluated_subject_binding_digest
        != requirement.subject_binding_digest
    ):
        raise CapabilityRecoveryContractError(
            "CAPABILITY_RECOVERY_ELIGIBILITY_SUBJECT_BINDING_MISMATCH",
            "EligibilityResult must be explicitly bound to the exact canonical authority requirement subject.",
            {
                "evaluated": binding.evaluated_subject_binding_digest,
                "required": requirement.subject_binding_digest,
            },
        )
    if binding.eligibility.status is not EligibilityStatus.CURRENTLY_ELIGIBLE:
        return False
    if binding.eligibility.subject_match not in _SUBJECT_MATCHES:
        return False
    authority_effects = set(_authority_effects(binding.authority))
    return set(requirement.required_effects).issubset(authority_effects)


@dataclass(frozen=True)
class AuthorityCompatibilityMapping:
    """Exact machine mapping from eligible Project Authority semantics to PTSIP semantics."""

    id: str
    requirement_id: str
    requirement_subject_binding_digest: str
    required_effects: tuple[str, ...]
    authority_id: str
    authority_semantics_digest: str
    canonical_key: str
    canonical_value: str
    closed_vocabulary: tuple[str, ...]

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        _require_text("requirement_id", self.requirement_id)
        _require_sha256(
            "requirement_subject_binding_digest",
            self.requirement_subject_binding_digest,
        )
        effects = _require_text_tuple(
            "required_effects",
            self.required_effects,
            allow_empty=False,
        )
        if tuple(sorted(effects)) != effects:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_REQUIRED_EFFECTS_NOT_CANONICAL",
                "required_effects must be stored in deterministic sorted order.",
                effects,
            )
        authority_id = _require_text("authority_id", self.authority_id)
        if _looks_like_physical_authority_identity(authority_id):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AUTHORITY_IDENTITY_INVALID",
                "Compatibility mapping authority_id must be a stable logical identity.",
                authority_id,
            )
        _require_sha256(
            "authority_semantics_digest",
            self.authority_semantics_digest,
        )
        _require_text("canonical_key", self.canonical_key)
        _require_text("canonical_value", self.canonical_value)
        vocabulary = _require_text_tuple(
            "closed_vocabulary",
            self.closed_vocabulary,
            allow_empty=False,
        )
        if self.canonical_value not in vocabulary:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_VALUE_OUTSIDE_VOCABULARY",
                "Compatibility mapping value must belong to its closed vocabulary.",
                self.canonical_value,
            )

    @property
    def lookup_key(self) -> tuple[object, ...]:
        return (
            self.requirement_id,
            self.requirement_subject_binding_digest,
            self.required_effects,
            self.authority_id,
            self.authority_semantics_digest,
        )


@dataclass(frozen=True)
class AuthorityCompatibilityCatalog:
    """Injected machine-readable catalog; S3 performs no repository discovery."""

    id: str
    mappings: tuple[AuthorityCompatibilityMapping, ...]

    def __post_init__(self) -> None:
        catalog_id = _require_text("id", self.id)
        if _looks_like_physical_authority_identity(catalog_id):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_CATALOG_IDENTITY_INVALID",
                "Catalog identity must be logical and must not encode a path or Git revision.",
                catalog_id,
            )
        if not isinstance(self.mappings, tuple) or any(
            not isinstance(item, AuthorityCompatibilityMapping)
            for item in self.mappings
        ):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_MAPPINGS_INVALID",
                "mappings must be a tuple of AuthorityCompatibilityMapping values.",
                self.mappings,
            )
        ids = tuple(item.id for item in self.mappings)
        if len(set(ids)) != len(ids):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_MAPPING_ID_DUPLICATE",
                "Compatibility mapping identifiers must be unique.",
                ids,
            )
        keys = tuple(item.lookup_key for item in self.mappings)
        if len(set(keys)) != len(keys):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_MAPPING_AMBIGUOUS",
                "Multiple mappings must not claim the same exact authority requirement and semantics.",
                keys,
            )

    def resolve(
        self,
        *,
        requirement: AuthorityRequirementView,
        authority: ProjectAuthorityRecord,
    ) -> AuthorityCompatibilityMapping | None:
        key = (
            requirement.requirement_id,
            requirement.subject_binding_digest,
            requirement.required_effects,
            authority.authority_id,
            stable_digest(dict(authority.authority_semantics)),
        )
        matches = tuple(item for item in self.mappings if item.lookup_key == key)
        if len(matches) > 1:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_MAPPING_AMBIGUOUS",
                "Authority compatibility lookup produced multiple exact mappings.",
                key,
            )
        return matches[0] if matches else None


class ProjectLocalAuthorityCompatibilityAdapter:
    """Pure adapter over injected canonical inputs; it never scans or infers."""

    def __init__(self, catalog: AuthorityCompatibilityCatalog) -> None:
        if not isinstance(catalog, AuthorityCompatibilityCatalog):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_CATALOG_REQUIRED",
                "Adapter requires an explicit AuthorityCompatibilityCatalog.",
                catalog,
            )
        self._catalog = catalog

    @property
    def catalog_id(self) -> str:
        return self._catalog.id

    def adapt(
        self,
        assessment: CapabilityRecoveryAssessment,
        *,
        authority_requirement: Mapping[str, object],
        authority_inputs: tuple[
            AuthorityRequirementEligibilityBinding,
            ...,
        ],
    ) -> CapabilityRecoveryAssessment:
        if not isinstance(assessment, CapabilityRecoveryAssessment):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ASSESSMENT_REQUIRED",
                "Authority compatibility recovery requires CapabilityRecoveryAssessment.",
                assessment,
            )
        if (
            assessment.next_path
            is not CapabilityRecoveryPath.PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER
        ):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_ORDER_INVALID",
                "Compatibility adapter may run only after lossless normalization was rejected.",
                assessment.next_path,
            )
        if not isinstance(authority_inputs, tuple):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AUTHORITY_INPUTS_INVALID",
                "authority_inputs must be a tuple of ProjectAuthorityRecord and EligibilityResult pairs.",
                authority_inputs,
            )

        requirement = authority_requirement_view(authority_requirement)
        if assessment.authority_requirement_id != requirement.requirement_id:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_REQUIREMENT_MISMATCH",
                "Recovery assessment must bind the same canonical authority requirement.",
                (
                    assessment.authority_requirement_id,
                    requirement.requirement_id,
                ),
            )
        if (
            assessment.authority_requirement_subject_binding_digest
            != requirement.subject_binding_digest
        ):
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_REQUIREMENT_SUBJECT_MISMATCH",
                "Recovery assessment must preserve the authority requirement subject binding.",
                requirement.subject_binding,
            )

        recoverable: list[
            tuple[ProjectAuthorityRecord, AuthorityCompatibilityMapping]
        ] = []
        seen_authority_ids: set[str] = set()
        for binding in authority_inputs:
            if not isinstance(binding, AuthorityRequirementEligibilityBinding):
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_AUTHORITY_INPUTS_INVALID",
                    "Each authority input must be an AuthorityRequirementEligibilityBinding.",
                    binding,
                )
            authority = binding.authority
            if authority.authority_id in seen_authority_ids:
                raise CapabilityRecoveryContractError(
                    "CAPABILITY_RECOVERY_AUTHORITY_INPUT_DUPLICATE",
                    "The same Project Authority must not be supplied more than once.",
                    authority.authority_id,
                )
            seen_authority_ids.add(authority.authority_id)
            if not _eligible_for_requirement(
                binding=binding,
                requirement=requirement,
            ):
                continue
            mapping = self._catalog.resolve(
                requirement=requirement,
                authority=authority,
            )
            if mapping is not None:
                recoverable.append((authority, mapping))

        if not recoverable:
            return reject_next_recovery_path(
                assessment,
                reason_code="NO_EXPLICIT_ELIGIBLE_AUTHORITY_COMPATIBILITY_MAPPING",
                reason=(
                    "No currently eligible subject-applicable Project Authority "
                    "with sufficient registered effects matched an exact injected mapping."
                ),
            )
        if len(recoverable) > 1:
            raise CapabilityRecoveryContractError(
                "CAPABILITY_RECOVERY_AUTHORITY_MAPPING_AMBIGUOUS",
                "More than one eligible Project Authority can recover the same requirement; no implicit preference is allowed.",
                tuple(authority.authority_id for authority, _ in recoverable),
            )

        authority, mapping = recoverable[0]
        return recover_on_next_path(
            assessment,
            recovered_capability=RecoveredCapability(
                required_capability=assessment.required_capability,
                subject=assessment.subject,
                canonical_key=mapping.canonical_key,
                canonical_value=mapping.canonical_value,
                closed_vocabulary=mapping.closed_vocabulary,
                source_path=(
                    CapabilityRecoveryPath.PROJECT_LOCAL_AUTHORITY_COMPATIBILITY_ADAPTER
                ),
                authority_requirement_id=requirement.requirement_id,
                consumed_project_authority_ids=(authority.authority_id,),
                materialization_ref=mapping.id,
            ),
            reason_code="EXPLICIT_ELIGIBLE_AUTHORITY_COMPATIBILITY_MAPPING",
            reason=(
                "One eligible subject-applicable Project Authority with sufficient "
                "registered effects matched an exact injected compatibility mapping."
            ),
        )
