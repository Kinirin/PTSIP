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
from _test_support import canonical_v2_answer


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


def _development_tooling_adopt_args(repo: Path) -> list[str]:
    return [
        "adopt",
        str(repo),
        "--component",
        "tools",
        "--classification",
        "DEVELOPMENT_TOOLING",
        "--purpose",
        "Repository-local generation tooling",
        "--shipped",
        "no",
        "--runtime-required",
        "no",
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
    payload = canonical_v2_answer()
    payload["shipped"] = "false"
    with pytest.raises(ValueError, match="shipped must be a boolean"):
        answer_from_mapping(payload)


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
    assert decision_a["profile_path"] == "ptsip.yaml"
    assert str(decision_a["id"]).startswith("gdec-")
    assert len(store.documents) == 1

    accepted = first.resolve(
        {
            "decision_id": str(decision_a["id"]),
            "answer": canonical_v2_answer(),
            "actor": "owner-a",
        }
    )
    assert accepted["status"] == "RESOLVED"
    assert accepted["accepted"] is True
    accepted_record = accepted["decision"]
    assert isinstance(accepted_record, dict)
    assert "lifecycle_owner" not in accepted_record["answer"]

    rejected = second.resolve(
        {
            "decision_id": str(decision_b["id"]),
            "answer": canonical_v2_answer(
                classification="PRODUCT",
                purpose="Product runtime component",
                shipped=True,
                runtime_required=True,
            ),
            "actor": "owner-b",
        }
    )
    assert rejected["status"] == "ALREADY_RESOLVED"
    assert rejected["accepted"] is False
    winner = rejected["decision"]
    assert isinstance(winner, dict)
    assert winner["answer"]["classification"] == "DEVELOPMENT_TOOLING"
    assert "lifecycle_owner" not in winner["answer"]


def test_github_authority_profile_path_is_part_of_scope_identity() -> None:
    store = MemoryAuthority()
    client = GithubControlPlaneClient("example/project", store=store)
    request = {
        "component_id": "tools",
        "include": ["tools/**"],
        "missing_fields": ["classification"],
    }
    common = {
        "repository": "example/project",
        "branch": "main",
        "subject_revision": "a" * 40,
        "component_id": "tools",
        "request": request,
    }

    root = client.gate({**common, "id": "clr-root", "profile_path": "ptsip.yaml"})
    nested = client.gate(
        {**common, "id": "clr-nested", "profile_path": ".\\config\\ptsip.yaml"}
    )
    default_root = client.peek({"repository": "example/project", "request": request})

    root_record = root["decision"]
    nested_record = nested["decision"]
    assert isinstance(root_record, dict)
    assert isinstance(nested_record, dict)
    assert root_record["profile_path"] == "ptsip.yaml"
    assert nested_record["profile_path"] == "config/ptsip.yaml"
    assert root_record["id"] != nested_record["id"]
    assert default_root["decision_id"] == root_record["id"]
    assert len(store.documents) == 2


def test_github_proposal_resolution_returns_terminal_local_receipt(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    store = MemoryAuthority()
    client = GithubControlPlaneClient("example/project", store=store)
    monkeypatch.setattr(cli_module, "GithubControlPlaneClient", lambda repository: client)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    assert main([
        "propose-component", str(repo), "--component", "future-service",
        "--include", "future/service/**", "--coordination", "github", "--json",
    ]) == 0
    registered = json.loads(capsys.readouterr().out)
    assert registered["status"] == "DECISION_REQUIRED"
    assert registered["decision"]["status"] == "PENDING"
    assert registered["decision"]["request"]["origin"] == "EXPLICIT_PROPOSED_COMPONENT"
    decision_id = registered["decision"]["id"]

    receipts = []
    winners = []
    original_application = client.application

    def record_local_receipt(payload):
        before = copy.deepcopy(store.documents)
        head, writes = store.head, store.counter
        winner = client.decision({"decision_id": decision_id})["decision"]
        assert winner["status"] == "RESOLVED"
        receipt = original_application(payload)
        assert store.documents == before
        assert (store.head, store.counter) == (head, writes)
        assert client.decision({"decision_id": decision_id})["decision"] == winner
        winners.append(winner)
        receipts.append(receipt)
        return receipt

    monkeypatch.setattr(client, "application", record_local_receipt)
    assert main([
        "resolve", str(repo), "--decision", decision_id,
        "--classification", "DEVELOPMENT_TOOLING",
        "--purpose", "Future repository development service",
        "--shipped", "no", "--runtime-required", "no", "--executable", "yes",
        "--coordination", "github", "--json",
    ]) == 0
    resolved = json.loads(capsys.readouterr().out)
    assert resolved["status"] == "PROPOSAL_APPROVED"
    assert resolved["backend"] == "GITHUB"
    assert resolved["decision"] == winners[0]
    assert resolved["decision"]["status"] == "RESOLVED"
    assert resolved["decision"]["answer"]["classification"] == "DEVELOPMENT_TOOLING"
    assert len(receipts) == 1
    assert resolved["application"] == receipts[0] == {
        "backend": "GITHUB",
        "scope": "LOCAL_PROJECTION",
        "status": "PROPOSAL_APPROVED",
        "decision_id": decision_id,
        "profile_path": "ptsip.yaml",
        "applied_revision": _git(repo, "rev-parse", "HEAD").stdout.strip(),
    }
    assert "application_status" not in resolved["decision"]
    assert resolved["materialized"] is False
    assert resolved["active_component_declared"] is False
    assert not (repo / "ptsip.yaml").exists()
    assert not (repo / "future").exists()
    assert not decision_store_path(repo).exists()

    authority_before = copy.deepcopy(store.documents)
    head, writes = store.head, store.counter
    rejected = client.resolve({
        "decision_id": decision_id,
        "answer": canonical_v2_answer(
            classification="PRODUCT", purpose="Contradictory later answer",
            shipped=True, runtime_required=True,
        ),
        "actor": "later-owner",
    })
    assert rejected["status"] == "ALREADY_RESOLVED"
    assert rejected["accepted"] is False
    assert rejected["decision"] == winners[0]
    assert store.documents == authority_before
    assert (store.head, store.counter) == (head, writes)


def test_repeated_github_proposal_is_terminal_without_reapplication(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    store = MemoryAuthority()
    client = GithubControlPlaneClient("example/project", store=store)
    monkeypatch.setattr(cli_module, "GithubControlPlaneClient", lambda repository: client)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    propose_args = [
        "propose-component", str(repo), "--component", "future-service",
        "--include", "future/service/**", "--coordination", "github", "--json",
    ]
    assert main(propose_args) == 0
    registered = json.loads(capsys.readouterr().out)
    decision_id = registered["decision"]["id"]
    resolved = client.resolve({
        "decision_id": decision_id,
        "answer": canonical_v2_answer(),
        "actor": "owner",
    })
    assert resolved["status"] == "RESOLVED"
    assert resolved["accepted"] is True
    authority_before = copy.deepcopy(store.documents)
    head, writes = store.head, store.counter

    def unexpected_mutation(*args, **kwargs):
        pytest.fail("A resolved explicit proposal must not require reapplication or authority writes")

    monkeypatch.setattr(client, "application", unexpected_mutation)
    monkeypatch.setattr(client, "resolve", unexpected_mutation)
    monkeypatch.setattr(store, "write_json", unexpected_mutation)
    monkeypatch.setattr(cli_module, "prepare_local_profile", unexpected_mutation)
    monkeypatch.setattr(cli_module, "write_prepared_local_profile", unexpected_mutation)

    for _ in range(3):
        assert main(propose_args) == 0
        repeated = json.loads(capsys.readouterr().out)
        assert repeated["status"] == "RESOLVED"
        assert repeated["decision"] == resolved["decision"]
        assert repeated["candidate"]["materialized"] is False
        assert repeated["candidate"]["authoritative"] is False
        assert store.documents == authority_before
        assert (store.head, store.counter) == (head, writes)
        assert not (repo / "ptsip.yaml").exists()
        assert not (repo / "future").exists()
        assert not decision_store_path(repo).exists()


def test_normal_github_resolution_still_requires_local_application() -> None:
    store = MemoryAuthority()
    client = GithubControlPlaneClient("example/project", store=store)
    payload = {
        "id": "clr-normal", "repository": "example/project", "branch": "main",
        "subject_revision": "a" * 40, "component_id": "tools",
        "request": {
            "component_id": "tools", "include": ["tools/**"],
            "missing_fields": ["classification", "purpose"],
        },
    }
    pending = client.gate(payload)
    assert pending["status"] == "DECISION_REQUIRED"
    decision_id = pending["decision"]["id"]
    resolved = client.resolve({
        "decision_id": decision_id, "answer": canonical_v2_answer(), "actor": "owner",
    })
    assert resolved["status"] == "RESOLVED"
    assert resolved["accepted"] is True
    authority_before = copy.deepcopy(store.documents)
    head, writes = store.head, store.counter
    for response in (
        client.gate(payload), client.peek(payload), client.decision({"decision_id": decision_id}),
    ):
        assert response["status"] == "RESOLVED_APPLICATION_REQUIRED"
        assert response["decision"] == resolved["decision"]
        assert response["decision"]["status"] == "RESOLVED"
    for status in ("LOCAL_APPLIED", "FAILED", "STALE"):
        receipt = client.application({"decision_id": decision_id, "status": status})
        assert receipt["status"] == status
        assert receipt["scope"] == "LOCAL_PROJECTION"
    assert store.documents == authority_before
    assert (store.head, store.counter) == (head, writes)


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

    assert main(_development_tooling_adopt_args(repo_a)) == 0
    adopted = json.loads(capsys.readouterr().out)
    assert adopted["status"] == "ADOPTED"
    assert adopted["backend"] == "GITHUB"
    assert adopted["authority"]["decision"]["answer"]["classification"] == "DEVELOPMENT_TOOLING"
    assert adopted["authority"]["decision"]["answer"]["runtime_required"] is False
    assert adopted["authority"]["decision"]["profile_path"] == "ptsip.yaml"
    assert "lifecycle_owner" not in adopted["authority"]["decision"]["answer"]
    assert (repo_a / "ptsip.yaml").is_file()
    assert not (repo_b / "ptsip.yaml").exists()

    assert main(["gate", str(repo_b), "--component", "tools", "--json"]) == 0
    gated = json.loads(capsys.readouterr().out)
    assert gated["status"] == "RESOLVED"
    assert gated["backend"] == "GITHUB"
    assert gated["profile_path"] == "ptsip.yaml"
    assert gated["decisions"][0]["reconciliation"]["status"] == "LOCAL_APPLIED"

    profile = yaml.safe_load((repo_b / "ptsip.yaml").read_text(encoding="utf-8"))
    component = next(item for item in profile["components"] if item["id"] == "tools")
    assert component["classification"] == "DEVELOPMENT_TOOLING"
    assert component["runtime_required"] is False
    assert "lifecycle_owner" not in component
    assert not decision_store_path(repo_b).exists()
