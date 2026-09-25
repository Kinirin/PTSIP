from __future__ import annotations

import pytest

from developer.automation.branch_creation_guard import (
    BranchCreationPolicyError,
    classify_existing,
    project_profile_transition,
    validate_creation,
)


def test_exact_user_approved_development_branch_is_authorized() -> None:
    decision = validate_creation(
        "dev/0.4.0",
        "dev/0.4.0",
        authorization_source="USER_EXPLICIT",
        request_kind="DEVELOPMENT_VERSION_BRANCH",
    )
    assert decision.status == "AUTHORIZED"
    assert decision.branch_class == "DEVELOPMENT_VERSION"


def test_agent_cannot_invent_or_substitute_branch_name() -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        validate_creation(
            "verify/context-plane-20260925",
            "dev/0.4.0",
            authorization_source="USER_EXPLICIT",
            request_kind="DEVELOPMENT_VERSION_BRANCH",
        )
    assert exc.value.code == "BRANCH_NAME_NOT_EXACTLY_APPROVED"


def test_non_user_authority_fails_closed() -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        validate_creation(
            "dev/0.4.0",
            "dev/0.4.0",
            authorization_source="AGENT_INFERRED",
            request_kind="DEVELOPMENT_VERSION_BRANCH",
        )
    assert exc.value.code == "BRANCH_CREATION_REQUIRES_USER_EXPLICIT"


def test_unregistered_branch_shape_is_not_creation_authority() -> None:
    with pytest.raises(BranchCreationPolicyError) as exc:
        validate_creation(
            "fix/js-ts-dependency-resolution",
            "fix/js-ts-dependency-resolution",
            authorization_source="USER_EXPLICIT",
            request_kind="DEVELOPMENT_VERSION_BRANCH",
        )
    assert exc.value.code == "UNAUTHORIZED_BRANCH_NAME"


def test_project_profile_change_does_not_change_development_branch_identity() -> None:
    result = project_profile_transition("dev/0.4.0", "pp.1.01", "pp.1.02")
    assert result["branch_change_required"] is False
    assert result["branch_creation_authorized"] is False
    assert result["branch_name"] == "dev/0.4.0"


def test_legacy_tool_branch_is_retention_only() -> None:
    result = classify_existing("tool-0.3.4-authority-consistency")
    assert result["classification"] == "GRANDFATHERED_RETENTION"

    with pytest.raises(BranchCreationPolicyError):
        validate_creation(
            "tool-0.3.4-authority-consistency",
            "tool-0.3.4-authority-consistency",
            authorization_source="USER_EXPLICIT",
            request_kind="DEVELOPMENT_VERSION_BRANCH",
        )
