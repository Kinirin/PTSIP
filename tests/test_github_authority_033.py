from __future__ import annotations

import copy
import json
import subprocess
from pathlib import Path

import pytest
import yaml

import ptsip.cli as cli_module
from ptsip.app.github_authority import (
    AuthorityConflict,
    CoordinationUnavailable,
    GitHubAuthorityStore,
    GithubControlPlaneClient,
    answer_from_mapping,
)
from ptsip.cli import main
from ptsip.storage.local_state import decision_store_path


class MemoryAuthority:
    def __init__(self) -> None:
        self.head = "h0"
        self.counter = 0
        self.documents: dict[str, dict[str, object]] = {}

    def ensure_head(self) -> str:
        return self.head

    def read_json(self, path: str) -> tuple[str, dict[str, object] | None]:
        document = self.documents.get(path)
        return self.head, copy.deepcopy(document) if document is not None else None

    def write_json(
        self,
        path: str,
        payload: dict[str, object],
        expected_head: str,
        message: str,
    ) -> str:
        del message
        if expected_head != self.head:
            raise AuthorityConflict("stale test writer")
        self.counter += 1
        self.head = f"h{self.counter}"
        self.documents[path] = copy.deepcopy(payload)
        return self.head


class NonPtsipBranchApi:
    def request(
        self,
        method: str,
        path: str,
        payload: dict[str, object] | None = None,
    ) -> dict[str, object]:
        del payload
        if method == "GET" and path.endswith("git/ref/heads/ptsip-policy"):
            return {"object": {"sha": "existing-head"}}
        if method == "GET" and path.endswith("git/commits/existing-head"):
            return {"tree": {"sha": "existing-tree"}}
        if method == "GET" and path.endswith("git/trees/existing-tree?recursive=1"):
            return {"tree": []}
        raise AssertionError(f"unexpected API request: {method} {path}")


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path, name: str = "repo") -> Path:
    repo = tmp_path / name
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "tools").mkdir()
    (repo / "tools" / "generate.py").write_text("print('generate')\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    _git(repo, "remote", "add", "origin", "https://github.com/example/project.git")
    return repo


def _toolchain_adopt_args(repo: Path) -> list[str]:
    return [
        "adopt",
        str(repo),
        "--component",
        "tools",
        "--classification",
        "TOOLCHAIN",
        "--purpose",
        "Repository-local generation tooling",
        "--shipped",
        "no",
        "--runtime-required",
        "no",
        "--lifecycle-owner",
        "DEVELOPMENT_TOOLING",
        "--executable",
        "yes",
        "--apply",
        "--json",
    ]


def test_existing_non_ptsip_authority_branch_is_refused() -> None:
    store = GitHubAuthorityStore("example/project", api=NonPtsipBranchApi())  # type: ignore[arg-type]
    with pytest.raises(CoordinationUnavailable, match="not a PTSIP authority"):
        store.ensure_head()


def test_authority_answer_requires_real_booleans() -> None:
    with pytest.raises(ValueError, match="shipped must be a boolean"):
        answer_from_mapping(
            {
                "classification": "TOOLCHAIN",
                "purpose": "Repository-local generation tooling",
                "shipped": "false",
                "runtime_required": False,
                "lifecycle_owner": "DEVELOPMENT_TOOLING",
                "executable": True,
            }
        )


def test_github_authority_uses_component_scope_not_local_clarification_id() -> None:
    store = MemoryAuthority()
    first = GithubControlPlaneClient("example/project", store=store)
    second = GithubControlPlaneClient("example/project", store=store)

    request_a = {
        "component_id": "tools",
        "include": ["tools/**"],
        "missing_fields": ["classification", "purpose"],
    }
    request_b = {
        "component_id": "renamed-tools-component",
        "include": ["./tools/**"],
        "missing_fields": ["purpose"],
    }
    gate_a = first.gate(
        {
            "id": "clr-a",
            "repository": "example/project",
            "branch": "main",
            "subject_revision": "a" * 40,
            "component_id": "tools",
            "request": request_a,
        }
    )
    gate_b = second.gate(
        {
            "id": "clr-b",
            "repository": "example/project",
            "branch": "feature",
            "subject_revision": "b" * 40,
            "component_id": "renamed-tools-component",
            "request": request_b,
        }
    )

    decision_a = gate_a["decision"]
    decision_b = gate_b["decision"]
    assert isinstance(decision_a, dict)
    assert isinstance(decision_b, dict)
    assert decision_a["id"] == decision_b["id"]
    assert str(decision_a["id"]).startswith("gdec-")
    assert len(store.documents) == 1

    accepted = first.resolve(
        {
            "decision_id": str(decision_a["id"]),
            "answer": {
                "classification": "TOOLCHAIN",
                "purpose": "Repository-local generation tooling",
                "shipped": False,
                "runtime_required": False,
                "lifecycle_owner": "DEVELOPMENT_TOOLING",
                "executable": True,
            },
            "actor": "owner-a",
        }
    )
    assert accepted["status"] == "RESOLVED"
    assert accepted["accepted"] is True

    rejected = second.resolve(
        {
            "decision_id": str(decision_b["id"]),
            "answer": {
                "classification": "PRODUCT",
                "purpose": "Product runtime component",
                "shipped": True,
                "runtime_required": True,
                "lifecycle_owner": "PRODUCT",
                "executable": True,
            },
            "actor": "owner-b",
        }
    )
    assert rejected["status"] == "ALREADY_RESOLVED"
    assert rejected["accepted"] is False
    winner = rejected["decision"]
    assert isinstance(winner, dict)
    assert winner["answer"]["classification"] == "TOOLCHAIN"


def test_github_adoption_winner_reconciles_into_stale_clone(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo_a = _repo(tmp_path, "repo-a")
    repo_b = tmp_path / "repo-b"
    subprocess.run(
        ["git", "clone", str(repo_a), str(repo_b)],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    _git(repo_b, "remote", "set-url", "origin", "https://github.com/example/project.git")

    shared = MemoryAuthority()

    def github_client(repository: str):
        return GithubControlPlaneClient(repository, store=shared)

    monkeypatch.setattr(cli_module, "GithubControlPlaneClient", github_client)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    assert main(_toolchain_adopt_args(repo_a)) == 0
    adopted = json.loads(capsys.readouterr().out)
    assert adopted["status"] == "ADOPTED"
    assert adopted["backend"] == "GITHUB"
    assert adopted["authority"]["decision"]["answer"]["classification"] == "TOOLCHAIN"
    assert adopted["authority"]["decision"]["answer"]["runtime_required"] is False
    assert (repo_a / "ptsip.yaml").is_file()
    assert not (repo_b / "ptsip.yaml").exists()

    assert main(["gate", str(repo_b), "--component", "tools", "--json"]) == 0
    gated = json.loads(capsys.readouterr().out)
    assert gated["status"] == "RESOLVED"
    assert gated["backend"] == "GITHUB"
    assert gated["decisions"][0]["reconciliation"]["status"] == "LOCAL_APPLIED"

    profile = yaml.safe_load((repo_b / "ptsip.yaml").read_text(encoding="utf-8"))
    component = next(item for item in profile["components"] if item["id"] == "tools")
    assert component["classification"] == "TOOLCHAIN"
    assert "runtime_required" not in component
    assert not decision_store_path(repo_b).exists()
