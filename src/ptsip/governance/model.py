from __future__ import annotations

from copy import deepcopy
from dataclasses import dataclass
from enum import StrEnum
from typing import Mapping


class GovernanceAuthorityError(ValueError):
    """Fail-closed error for malformed or unsupported governance authority inputs."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class EligibilityStatus(StrEnum):
    CURRENTLY_ELIGIBLE = "CURRENTLY_ELIGIBLE_FOR_PROJECT_AUTHORITY_PROJECTION"
    CURRENTLY_INELIGIBLE = "CURRENTLY_INELIGIBLE_FOR_PROJECT_AUTHORITY_PROJECTION"
    MALFORMED_AUTHORITY_RECORD = "MALFORMED_AUTHORITY_RECORD"
    TOOL_CAPABILITY_GAP = "TOOL_CAPABILITY_GAP"


class LifecycleState(StrEnum):
    ACTIVE = "ACTIVE"
    DEPRECATED = "DEPRECATED"
    SUPERSEDED = "SUPERSEDED"
    RETIRED = "RETIRED"
    DRAFT = "DRAFT"


class SubjectMatchKind(StrEnum):
    EXACT = "EXACT"
    REGISTERED_MACHINE_RELATIONSHIP = "REGISTERED_MACHINE_RELATIONSHIP"
    NO_MATCH = "NO_MATCH"
    INVALID_BINDING = "INVALID_BINDING"
    UNSUPPORTED_BINDING = "UNSUPPORTED_BINDING"


class CheckStatus(StrEnum):
    PASS = "PASS"
    FAIL = "FAIL"
    NOT_EVALUATED = "NOT_EVALUATED"


class AuthorizationState(StrEnum):
    NOT_AUTHORIZED = "NOT_AUTHORIZED"
    HOLD_NOT_AUTHORIZED = "HOLD_NOT_AUTHORIZED"
    READY_FOR_AUTO_AUTHORIZATION = "READY_FOR_AUTO_AUTHORIZATION"
    AUTHORIZED = "AUTHORIZED"
    REVOKED_OR_INVALIDATED = "REVOKED_OR_INVALIDATED"


@dataclass(frozen=True)
class RepositoryBinding:
    scheme: str
    host: str
    repository_id: str

    @classmethod
    def from_mapping(cls, value: Mapping[str, object]) -> "RepositoryBinding":
        try:
            scheme = value["scheme"]
            host = value["host"]
            repository_id = value["repository_id"]
        except KeyError as exc:
            raise GovernanceAuthorityError(
                "AUTHORITY_BINDING_INVALID",
                f"repository binding is missing {exc.args[0]!r}.",
                value,
            ) from exc
        if not all(isinstance(item, str) and item for item in (scheme, host, repository_id)):
            raise GovernanceAuthorityError(
                "AUTHORITY_BINDING_INVALID",
                "repository binding fields must be non-empty strings.",
                value,
            )
        return cls(scheme=scheme, host=host, repository_id=repository_id)

    def as_dict(self) -> dict[str, str]:
        return {"scheme": self.scheme, "host": self.host, "repository_id": self.repository_id}


@dataclass(frozen=True)
class SubjectIdentity:
    scheme: str
    value: str

    @classmethod
    def from_mapping(cls, value: Mapping[str, object]) -> "SubjectIdentity":
        try:
            scheme = value["scheme"]
            identity = value["value"]
        except KeyError as exc:
            raise GovernanceAuthorityError(
                "AUTHORITY_BINDING_INVALID",
                f"subject identity is missing {exc.args[0]!r}.",
                value,
            ) from exc
        if not isinstance(scheme, str) or not scheme or not isinstance(identity, str) or not identity:
            raise GovernanceAuthorityError(
                "AUTHORITY_BINDING_INVALID",
                "subject identity fields must be non-empty strings.",
                value,
            )
        return cls(scheme=scheme, value=identity)

    def as_dict(self) -> dict[str, str]:
        return {"scheme": self.scheme, "value": self.value}


@dataclass(frozen=True)
class SolveSubject:
    authority_domain: str
    repository_binding: RepositoryBinding
    subject_type: str
    subject_identity: SubjectIdentity

    @classmethod
    def from_mapping(cls, value: Mapping[str, object]) -> "SolveSubject":
        try:
            repository = value["repository_binding"]
            identity = value["subject_identity"]
            authority_domain = value["authority_domain"]
            subject_type = value["subject_type"]
        except KeyError as exc:
            raise GovernanceAuthorityError(
                "AUTHORITY_BINDING_INVALID",
                f"solve subject is missing {exc.args[0]!r}.",
                value,
            ) from exc
        if not isinstance(repository, Mapping) or not isinstance(identity, Mapping):
            raise GovernanceAuthorityError(
                "AUTHORITY_BINDING_INVALID",
                "solve subject requires structured repository_binding and subject_identity.",
                value,
            )
        if not isinstance(authority_domain, str) or not authority_domain:
            raise GovernanceAuthorityError("AUTHORITY_BINDING_INVALID", "authority_domain must be a non-empty string.", authority_domain)
        if not isinstance(subject_type, str) or not subject_type:
            raise GovernanceAuthorityError("AUTHORITY_BINDING_INVALID", "subject_type must be a non-empty string.", subject_type)
        return cls(
            authority_domain=authority_domain,
            repository_binding=RepositoryBinding.from_mapping(repository),
            subject_type=subject_type,
            subject_identity=SubjectIdentity.from_mapping(identity),
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "authority_domain": self.authority_domain,
            "repository_binding": self.repository_binding.as_dict(),
            "subject_type": self.subject_type,
            "subject_identity": self.subject_identity.as_dict(),
        }


@dataclass(frozen=True)
class EligibilityCheck:
    id: str
    status: CheckStatus
    diagnostic: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {"id": self.id, "status": self.status.value, "diagnostic": self.diagnostic}


@dataclass(frozen=True)
class EligibilityResult:
    authority_id: str
    source_ref: str
    status: EligibilityStatus
    checks: tuple[EligibilityCheck, ...]
    lifecycle_state: LifecycleState | None = None
    subject_match: SubjectMatchKind | None = None
    diagnostics: tuple[str, ...] = ()

    @property
    def currently_eligible(self) -> bool:
        return self.status is EligibilityStatus.CURRENTLY_ELIGIBLE

    def as_dict(self) -> dict[str, object]:
        return {
            "schema_version": "ptsip-authority-eligibility-result/v1",
            "authority_id": self.authority_id,
            "source_ref": self.source_ref,
            "status": self.status.value,
            "lifecycle_state": self.lifecycle_state.value if self.lifecycle_state else None,
            "subject_match": self.subject_match.value if self.subject_match else None,
            "checks": [item.as_dict() for item in self.checks],
            "diagnostics": list(self.diagnostics),
        }


@dataclass(frozen=True)
class ProjectAuthorityRecord:
    authority_id: str
    authority_contract: Mapping[str, object]
    authority_semantics: Mapping[str, object]
    authority_role: Mapping[str, object]
    authority_provenance: Mapping[str, object]
    subject_binding: Mapping[str, object]

    def as_dict(self) -> dict[str, object]:
        return {
            "schema_version": "ptsip-project-authority-record/v1",
            "authority_id": self.authority_id,
            "authority_contract": deepcopy(dict(self.authority_contract)),
            "authority_semantics": deepcopy(dict(self.authority_semantics)),
            "authority_role": deepcopy(dict(self.authority_role)),
            "authority_provenance": deepcopy(dict(self.authority_provenance)),
            "subject_binding": deepcopy(dict(self.subject_binding)),
        }


@dataclass(frozen=True)
class AuthorizationTransitionResult:
    scope: str
    state: AuthorizationState
    rule_id: str | None
    failed_predicates: tuple[str, ...] = ()
    blockers: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "scope": self.scope,
            "state": self.state.value,
            "rule_id": self.rule_id,
            "failed_predicates": list(self.failed_predicates),
            "blockers": list(self.blockers),
        }
