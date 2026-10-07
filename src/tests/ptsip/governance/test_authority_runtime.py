from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.governance.model import GovernanceAuthorityError

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


ROOT = Path(__file__).resolve().parents[4]
REVISION = "a" * 40


def _subject(policy_id: str, *, repository_id: str = "1327447827") -> SolveSubject:
    return SolveSubject(
        authority_domain="PROJECT_GOVERNANCE",
        repository_binding=RepositoryBinding(
            scheme="GITHUB_REPOSITORY_ID",
            host="github.com",
            repository_id=repository_id,
        ),
        subject_type="GOVERNANCE_TOPIC",
        subject_identity=SubjectIdentity(scheme="SUPPORT_POLICY_ID", value=policy_id),
    )


@pytest.fixture
def projection_runtime(monkeypatch):
    # Exercise projection/lifecycle/subject gates with a controlled eligible role.
    # Actual shipped Root roles remain contract authority, never project grants.
    runtime = ProjectAuthorityRuntime(ROOT)
    for role in runtime.catalog.role_registry["policy_roles"]:
        if role["policy_id"] in {"SFP-INFO-0001", "SFP-INFO-0002"}:
            monkeypatch.setitem(role, "projection_role", "PROJECT_ARCHITECTURE_AUTHORITY")
    return runtime


def test_actual_root_contracts_are_not_implicit_project_authority_grants():
    runtime = ProjectAuthorityRuntime(ROOT)
    result = runtime.evaluate_policy("SFP-INFO-0001", _subject("SFP-INFO-0001"), source_revision=REVISION)
    assert result.status is EligibilityStatus.CURRENTLY_INELIGIBLE
    assert "AUTHORITY_ROLE_NOT_PROJECT_AUTHORITY_ELIGIBLE" in result.diagnostics
    with pytest.raises(GovernanceAuthorityError, match="route for SFP-0001") as error:
        runtime.catalog.load_current_record("SFP-0001")
    assert error.value.code == "CURRENT_SUPPORT_POLICY_NOT_SELECTED"


def test_exact_current_subject_projects_one_current_authority_record(projection_runtime) -> None:
    runtime = projection_runtime
    subject = _subject("SFP-INFO-0001")
    result = runtime.evaluate_policy("SFP-INFO-0001", subject, source_revision=REVISION)
    assert result.status is EligibilityStatus.CURRENTLY_ELIGIBLE
    assert result.subject_match is SubjectMatchKind.EXACT
    assert result.lifecycle_state is LifecycleState.ACTIVE
    projected = runtime.project_current_authorities(subject, source_revision=REVISION)
    assert len(projected) == 1
    record = projected[0].as_dict()
    assert record["authority_id"] == "SFP-INFO-0001"
    assert record["authority_role"]["projection_role"] == "PROJECT_ARCHITECTURE_AUTHORITY"
    assert record["subject_binding"]["subject_identity"] == {"scheme":"SUPPORT_POLICY_ID","value":"SFP-INFO-0001"}
    assert record["authority_provenance"]["source_type"] == "SUPPORT_FEATURE_POLICY"
    assert record["authority_provenance"]["source_ref"] == "src/policy/INFO/SFP-INFO-0001.yaml"
    assert "canonical_sources" not in record["authority_provenance"]
    assert "source_role" not in record["authority_provenance"]
    assert len(record["authority_provenance"]["source_digest"]) == 64


def test_draft_lifecycle_current_route_is_not_automatically_projected(projection_runtime) -> None:
    runtime = projection_runtime
    subject = _subject("SFP-INFO-0002")
    result = runtime.evaluate_policy("SFP-INFO-0002", subject, source_revision=REVISION)
    assert result.status is EligibilityStatus.CURRENTLY_INELIGIBLE
    assert result.lifecycle_state is LifecycleState.DRAFT
    assert "AUTHORITY_LIFECYCLE_DRAFT" in result.diagnostics


def test_support_policy_has_no_builtin_ptsip_repository_binding(projection_runtime) -> None:
    runtime = projection_runtime
    for repository_id in ("1327447827", "999"):
        result = runtime.evaluate_policy("SFP-INFO-0001", _subject("SFP-INFO-0001", repository_id=repository_id), source_revision=REVISION)
        assert result.status is EligibilityStatus.CURRENTLY_ELIGIBLE


def test_registered_relationship_is_the_only_relaxed_subject_match() -> None:
    authority_binding = _subject("SFP-INFO-0001").as_dict()
    solve = _subject("SFP-INFO-0002")
    registry = {"matching": {"registered_machine_relationships": [{
        "relationship_id":"TEST-REL-001",
        "relationship_type":"AUTHORITY_SUBJECT_APPLIES_TO_SOLVE_SUBJECT",
        "authority_subject":{"scheme":"SUPPORT_POLICY_ID","value":"SFP-INFO-0001"},
        "solve_subject":{"scheme":"SUPPORT_POLICY_ID","value":"SFP-INFO-0002"},
    }]}}
    assert match_subject_binding(authority_binding, solve, registry) is SubjectMatchKind.REGISTERED_MACHINE_RELATIONSHIP
    registry["matching"]["registered_machine_relationships"] = []
    assert match_subject_binding(authority_binding, solve, registry) is SubjectMatchKind.NO_MATCH


def test_fresh_solve_does_not_persist_historical_eligibility(projection_runtime) -> None:
    runtime = projection_runtime
    eligible = runtime.evaluate_policy("SFP-INFO-0001", _subject("SFP-INFO-0001"), source_revision=REVISION)
    mismatched = runtime.evaluate_policy("SFP-INFO-0001", _subject("SFP-INFO-0002"), source_revision=REVISION)
    assert eligible.currently_eligible is True
    assert mismatched.currently_eligible is False


def test_support_policy_id_is_canonical_subject_identity() -> None:
    runtime = ProjectAuthorityRuntime(ROOT)
    subject = _subject("SFP-INFO-0001")

    binding = runtime.catalog.binding_for_policy("SFP-INFO-0001", subject)

    assert binding["subject_type"] == "GOVERNANCE_TOPIC"
    assert binding["subject_identity"] == {
        "scheme": "SUPPORT_POLICY_ID",
        "value": "SFP-INFO-0001",
    }
    assert not hasattr(runtime.catalog, "policy_id_for_legacy_topic")
    assert not hasattr(runtime, "evaluate_topic")
    assert not hasattr(runtime, "project_topic")
