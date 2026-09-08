from __future__ import annotations

from collections.abc import Mapping

from .model import (
    GovernanceAuthorityError,
    RepositoryBinding,
    SolveSubject,
    SubjectIdentity,
    SubjectMatchKind,
)


def _as_mapping(value: object, *, code: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise GovernanceAuthorityError(code, "expected a machine-readable mapping.", value)
    return value


def match_subject_binding(
    authority_binding: Mapping[str, object],
    solve_subject: SolveSubject,
    subject_registry: Mapping[str, object],
) -> SubjectMatchKind:
    """Match exact subject identity first, then an explicitly registered relationship.

    Registered relationships may relax only subject_identity. authority_domain,
    repository_binding, and subject_type remain exact-match dimensions.
    """

    try:
        authority_domain = authority_binding["authority_domain"]
        repository_value = _as_mapping(authority_binding["repository_binding"], code="AUTHORITY_BINDING_INVALID")
        subject_type = authority_binding["subject_type"]
        identity_value = _as_mapping(authority_binding["subject_identity"], code="AUTHORITY_BINDING_INVALID")
    except KeyError as exc:
        raise GovernanceAuthorityError(
            "AUTHORITY_BINDING_INVALID",
            f"authority subject binding is missing {exc.args[0]!r}.",
            authority_binding,
        ) from exc

    authority_repository = RepositoryBinding.from_mapping(repository_value)
    authority_identity = SubjectIdentity.from_mapping(identity_value)

    if authority_domain != solve_subject.authority_domain:
        return SubjectMatchKind.NO_MATCH
    if authority_repository != solve_subject.repository_binding:
        return SubjectMatchKind.NO_MATCH
    if subject_type != solve_subject.subject_type:
        return SubjectMatchKind.NO_MATCH
    if authority_identity == solve_subject.subject_identity:
        return SubjectMatchKind.EXACT

    matching = _as_mapping(subject_registry.get("matching"), code="AUTHORITY_BINDING_INVALID")
    relationships = matching.get("registered_machine_relationships", [])
    if not isinstance(relationships, list):
        raise GovernanceAuthorityError(
            "AUTHORITY_BINDING_INVALID",
            "registered_machine_relationships must be a list.",
            relationships,
        )

    for relation in relationships:
        relation_map = _as_mapping(relation, code="AUTHORITY_BINDING_INVALID")
        if relation_map.get("relationship_type") != "AUTHORITY_SUBJECT_APPLIES_TO_SOLVE_SUBJECT":
            continue
        authority_subject = _as_mapping(relation_map.get("authority_subject"), code="AUTHORITY_BINDING_INVALID")
        solve_target = _as_mapping(relation_map.get("solve_subject"), code="AUTHORITY_BINDING_INVALID")
        if (
            authority_subject.get("scheme") == authority_identity.scheme
            and authority_subject.get("value") == authority_identity.value
            and solve_target.get("scheme") == solve_subject.subject_identity.scheme
            and solve_target.get("value") == solve_subject.subject_identity.value
        ):
            return SubjectMatchKind.REGISTERED_MACHINE_RELATIONSHIP

    return SubjectMatchKind.NO_MATCH
