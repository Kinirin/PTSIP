from __future__ import annotations

import pytest

from developer.automation.branch_control import (
    BranchControlError,
    GitHubClient,
    create_authorized_branch,
    registered_commands,
)


class FakeClient(GitHubClient):
    def __init__(self, repository: str = "Kinirin/PTSIP") -> None:
        super().__init__(repository=repository, token="test-token")
        self.created: list[tuple[str, str]] = []

    def create_branch(self, branch: str, base_ref: str) -> dict[str, object]:
        self.created.append((branch, base_ref))
        return {
            "status": "CREATED",
            "repository": self.repository,
            "branch": branch,
            "base_ref": base_ref,
            "base_sha": "a" * 40,
            "creation_mechanism": "GITHUB_CREATE_BRANCH_API",
            "ref": f"refs/heads/{branch}",
        }


def test_registry_is_closed_and_contains_only_approved_operations() -> None:
    result = registered_commands()
    assert sorted(result["commands"]) == ["commands", "create", "inspect", "list"]
    assert result["unregistered_operation"] == "FAIL_CLOSED"


def test_create_uses_exact_approved_name_and_registered_api_path() -> None:
    client = FakeClient()
    result = create_authorized_branch(
        repository="Kinirin/PTSIP",
        branch="dev/0.10.0",
        approved_name="dev/0.10.0",
        base_ref="main",
        client=client,
    )
    assert result["status"] == "CREATED"
    assert result["creation_mechanism"] == "GITHUB_CREATE_BRANCH_API"
    assert client.created == [("dev/0.10.0", "main")]


def test_create_rejects_unapproved_exact_name_before_api_call() -> None:
    client = FakeClient()
    with pytest.raises(Exception) as exc:
        create_authorized_branch(
            repository="Kinirin/PTSIP",
            branch="dev/0.10.1",
            approved_name="dev/0.10.0",
            base_ref="main",
            client=client,
        )
    assert getattr(exc.value, "code", None) == "BRANCH_NAME_NOT_EXACTLY_APPROVED"
    assert client.created == []


def test_create_rejects_unregistered_branch_shape_before_api_call() -> None:
    client = FakeClient()
    with pytest.raises(Exception) as exc:
        create_authorized_branch(
            repository="Kinirin/PTSIP",
            branch="verify/context-plane",
            approved_name="verify/context-plane",
            base_ref="main",
            client=client,
        )
    assert getattr(exc.value, "code", None) == "UNAUTHORIZED_BRANCH_NAME"
    assert client.created == []


def test_client_repository_must_match_requested_repository() -> None:
    with pytest.raises(BranchControlError) as exc:
        create_authorized_branch(
            repository="Kinirin/PTSIP",
            branch="dev/1.2.3",
            approved_name="dev/1.2.3",
            base_ref="main",
            client=FakeClient("Other/Repo"),
        )
    assert exc.value.code == "REPOSITORY_BINDING_MISMATCH"


def test_invalid_repository_shape_fails_closed() -> None:
    with pytest.raises(BranchControlError) as exc:
        GitHubClient(repository="not-a-repository", token="x")
    assert exc.value.code == "INVALID_REPOSITORY"
