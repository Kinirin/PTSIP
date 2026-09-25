from __future__ import annotations

import pytest

from developer.automation.branch_creation_guard import (
    BranchCreationPolicyError,
    classify_existing,
    project_profile_transition,
    validate_creation,
)


def _validate(candidate: str, approved_name: str, **overrides: str):
    args = {
        "authorization_source": "USER_EXPLICIT",
        "request_kind": "DEVELOPMENT_VERSION_BRANCH",
        "creation_mechanism": "GITHUB_CREATE_BRANCH_API",
    }
    args.update(overrides)
    return validate_creation(candidate, approved_name, **args)


def test_exact_user_approved_development_branch_is_authorized() -> None:
    decision = _validate("dev/0.4.0", "dev/0.4.0")
    assert decision.status == "AUTHORIZED"
    assert decision.branch_class == "DEVELOPMENT_VERSION"
    assert decision.creation_mechanism == "GITHUB_CREATE_BRANCH_API"


@pytest.mark.parametrize(
    "branch",
    [
        "dev/0.10.0",
        "dev/12.3.45",
        "dev/123.456.789",
    ],
)
def test_multi_digit_semver_components_are_authorized(branch: str) -> None:
    assert _validate(branch, branch).status == "AUTHORIZED"


def test_agent_cannot_invent_or_substitute_branch_name() -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        _validate("verify/context-plane-20260925", "dev/0.4.0")
    assert exc.value.code == "BRANCH_NAME_NOT_EXACTLY_APPROVED"


def test_non_user_authority_fails_closed() -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        _validate(
            "dev/0.4.0",
            "dev/0.4.0",
            authorization_source="AGENT_INFERRED",
        )
    assert exc.value.code == "BRANCH_CREATION_REQUIRES_USER_EXPLICIT"


def test_unregistered_branch_shape_is_not_creation_authority() -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        _validate(
            "fix/js-ts-dependency-resolution",
            "fix/js-ts-dependency-resolution",
        )
    assert exc.value.code == "UNAUTHORIZED_BRANCH_NAME"


@pytest.mark.parametrize(
    "mechanism",
    [
        "GIT_PUSH_BRANCH_CREATION",
        "GIT_SWITCH_CREATE",
        "GIT_CHECKOUT_CREATE",
        "GIT_UPDATE_REF",
        "GITHUB_UPDATE_REF_API",
        "WORKFLOW_REF_CREATION",
    ],
)
def test_only_github_create_branch_api_is_authorized(mechanism: str) -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        _validate(
            "dev/0.4.1",
            "dev/0.4.1",
            creation_mechanism=mechanism,
        )
    assert exc.value.code == "UNAUTHORIZED_BRANCH_CREATION_MECHANISM"


def test_project_profile_change_does_not_change_development_branch_identity() -> None:
    result = project_profile_transition("dev/0.4.0", "pp.1.01", "pp.1.02")
    assert result["branch_change_required"] is False
    assert result["branch_creation_authorized"] is False
    assert result["branch_name"] == "dev/0.4.0"


def test_legacy_tool_branch_is_retention_only() -> None:
    result = classify_existing("tool-0.3.4-authority-consistency")
    assert result["classification"] == "GRANDFATHERED_RETENTION"

    with pytest.raises(BranchCreationPolicyError):
        _validate(
            "tool-0.3.4-authority-consistency",
            "tool-0.3.4-authority-consistency",
        )
