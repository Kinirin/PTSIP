from __future__ import annotations

import pytest

from developer.automation.branch_control import (
    BranchControlError,
    GitHubClient,
    create_authorized_branch,
    recreate_authorized_branch,
    registered_commands,
)


class FakeClient(GitHubClient):
    def __init__(self, repository: str = "Kinirin/PTSIP") -> None:
        super().__init__(repository=repository, token="test-token")
        self.refs = {
            "main": "b" * 40,
            "dev/0.3.8": "a" * 40,
        }
        self.deleted: list[str] = []
        self.created: list[tuple[str, str]] = []
        self.behind_by = 0
        self.fail_create = False

    def resolve_ref_sha(self, ref: str) -> str:
        if ref not in self.refs:
            raise BranchControlError("NOT_FOUND", ref)
        return self.refs[ref]

    def compare(self, base: str, head: str) -> dict[str, object]:
        return {"behind_by": self.behind_by, "ahead_by": 9}

    def delete_branch_ref(self, branch: str) -> None:
        self.deleted.append(branch)
        self.refs.pop(branch)

    def create_branch_at_sha(self, branch: str, sha: str) -> dict[str, object]:
        self.created.append((branch, sha))
        if self.fail_create:
            self.fail_create = False
            raise BranchControlError("CREATE_FAILED", "simulated failure")
        self.refs[branch] = sha
        return {"ref": f"refs/heads/{branch}"}

    def create_branch(self, branch: str, base_ref: str) -> dict[str, object]:
        base_sha = self.resolve_ref_sha(base_ref)
        self.create_branch_at_sha(branch, base_sha)
        return {
            "status": "CREATED",
            "repository": self.repository,
            "branch": branch,
            "base_ref": base_ref,
            "base_sha": base_sha,
            "creation_mechanism": "GITHUB_CREATE_BRANCH_API",
            "ref": f"refs/heads/{branch}",
        }


def test_registry_is_closed_and_contains_only_approved_operations() -> None:
    result = registered_commands()
    assert sorted(result["commands"]) == ["commands", "create", "inspect", "list", "recreate"]
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


def test_recreate_requires_old_branch_to_be_fully_contained_in_base() -> None:
    client = FakeClient()
    client.behind_by = 1
    with pytest.raises(BranchControlError) as exc:
        recreate_authorized_branch(
            repository="Kinirin/PTSIP",
            branch="dev/0.3.8",
            approved_name="dev/0.3.8",
            base_ref="main",
            client=client,
        )
    assert exc.value.code == "BRANCH_HAS_UNMERGED_COMMITS"
    assert client.deleted == []


def test_recreate_deletes_and_recreates_at_current_base_sha() -> None:
    client = FakeClient()
    result = recreate_authorized_branch(
        repository="Kinirin/PTSIP",
        branch="dev/0.3.8",
        approved_name="dev/0.3.8",
        base_ref="main",
        client=client,
    )
    assert result["status"] == "RECREATED"
    assert result["old_sha"] == "a" * 40
    assert result["new_sha"] == "b" * 40
    assert client.deleted == ["dev/0.3.8"]
    assert client.refs["dev/0.3.8"] == client.refs["main"]


def test_recreate_rolls_back_original_ref_if_creation_fails() -> None:
    client = FakeClient()
    client.fail_create = True
    with pytest.raises(BranchControlError) as exc:
        recreate_authorized_branch(
            repository="Kinirin/PTSIP",
            branch="dev/0.3.8",
            approved_name="dev/0.3.8",
            base_ref="main",
            client=client,
        )
    assert exc.value.code == "RECREATE_FAILED_ROLLED_BACK"
    assert client.refs["dev/0.3.8"] == "a" * 40


def test_recreate_rejects_unapproved_name_before_deletion() -> None:
    client = FakeClient()
    with pytest.raises(Exception) as exc:
        recreate_authorized_branch(
            repository="Kinirin/PTSIP",
            branch="dev/0.3.8",
            approved_name="dev/0.3.9",
            base_ref="main",
            client=client,
        )
    assert getattr(exc.value, "code", None) == "BRANCH_NAME_NOT_EXACTLY_APPROVED"
    assert client.deleted == []


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
