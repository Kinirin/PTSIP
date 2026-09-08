from __future__ import annotations

from pathlib import Path

from ptsip.governance import (
    EligibilityStatus,
    LifecycleState,
    ProjectAuthorityRuntime,
    RepositoryBinding,
    SolveSubject,
    SubjectIdentity,
    SubjectMatchKind,
    match_subject_binding,
)


ROOT = Path(__file__).resolve().parents[3]
REVISION = "a" * 40


def _subject(topic_id: str, *, repository_id: str = "1327447827") -> SolveSubject:
    return SolveSubject(
        authority_domain="PROJECT_GOVERNANCE",
        repository_binding=RepositoryBinding(
            scheme="GITHUB_REPOSITORY_ID",
            host="github.com",
            repository_id=repository_id,
        ),
        subject_type="GOVERNANCE_TOPIC",
        subject_identity=SubjectIdentity(scheme="DECISION_TOPIC_ID", value=topic_id),
    )


def test_exact_current_subject_projects_one_current_authority_record() -> None:
    runtime = ProjectAuthorityRuntime(ROOT)
    subject = _subject("ptsip_establishment")
    result = runtime.evaluate_topic("ptsip_establishment", subject, source_revision=REVISION)
    assert result.status is EligibilityStatus.CURRENTLY_ELIGIBLE
    assert result.subject_match is SubjectMatchKind.EXACT
    assert result.lifecycle_state is LifecycleState.ACTIVE
    projected = runtime.project_current_authorities(subject, source_revision=REVISION)
    assert len(projected) == 1
    record = projected[0].as_dict()
    assert record["authority_id"] == "ADR-0001"
    assert record["authority_role"]["projection_role"] == "PROJECT_ARCHITECTURE_AUTHORITY"
    assert record["subject_binding"]["subject_identity"]["value"] == "ptsip_establishment"
    assert record["authority_provenance"]["source_revision"] == REVISION
    assert len(record["authority_provenance"]["source_digest"]) == 64


def test_draft_lifecycle_current_route_is_not_automatically_projected() -> None:
    runtime = ProjectAuthorityRuntime(ROOT)
    subject = _subject("evidence_artifact_conformance_semantics")
    result = runtime.evaluate_topic("evidence_artifact_conformance_semantics", subject, source_revision=REVISION)
    assert result.status is EligibilityStatus.CURRENTLY_INELIGIBLE
    assert result.lifecycle_state is LifecycleState.DRAFT
    assert "AUTHORITY_LIFECYCLE_DRAFT" in result.diagnostics
    assert runtime.project_topic("evidence_artifact_conformance_semantics", subject, source_revision=REVISION) is None


def test_repository_mismatch_is_not_applicable_not_authority_promotion() -> None:
    runtime = ProjectAuthorityRuntime(ROOT)
    result = runtime.evaluate_topic(
        "ptsip_establishment",
        _subject("ptsip_establishment", repository_id="999"),
        source_revision=REVISION,
    )
    assert result.status is EligibilityStatus.CURRENTLY_INELIGIBLE
    assert "REPOSITORY_BINDING_NOT_APPLICABLE" in result.diagnostics


def test_registered_relationship_is_the_only_relaxed_subject_match() -> None:
    authority_binding = _subject("ptsip_establishment").as_dict()
    solve = _subject("consumer_repository_non_intrusion")
    registry = {"matching": {"registered_machine_relationships": [{
        "relationship_id":"TEST-REL-001",
        "relationship_type":"AUTHORITY_SUBJECT_APPLIES_TO_SOLVE_SUBJECT",
        "authority_subject":{"scheme":"DECISION_TOPIC_ID","value":"ptsip_establishment"},
        "solve_subject":{"scheme":"DECISION_TOPIC_ID","value":"consumer_repository_non_intrusion"},
    }]}}
    assert match_subject_binding(authority_binding, solve, registry) is SubjectMatchKind.REGISTERED_MACHINE_RELATIONSHIP
    registry["matching"]["registered_machine_relationships"] = []
    assert match_subject_binding(authority_binding, solve, registry) is SubjectMatchKind.NO_MATCH


def test_fresh_solve_does_not_persist_historical_eligibility() -> None:
    runtime = ProjectAuthorityRuntime(ROOT)
    eligible = runtime.evaluate_topic("ptsip_establishment", _subject("ptsip_establishment"), source_revision=REVISION)
    mismatched = runtime.evaluate_topic("ptsip_establishment", _subject("consumer_repository_non_intrusion"), source_revision=REVISION)
    assert eligible.currently_eligible is True
    assert mismatched.currently_eligible is False
