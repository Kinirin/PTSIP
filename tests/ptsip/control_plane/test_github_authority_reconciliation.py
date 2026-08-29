from __future__ import annotations

import copy
import json
import subprocess
from pathlib import Path

import yaml

import ptsip.cli as cli_module
from ptsip.app.github_authority import AuthorityConflict, CoordinationUnavailable, GithubControlPlaneClient
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

    def read_json_if_exists(self, path: str) -> tuple[str | None, dict[str, object] | None]:
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


class UnavailableClient:
    def peek(self, payload: dict[str, object]) -> dict[str, object]:
        del payload
        raise CoordinationUnavailable("test authority unavailable")


class MissingAuthorityApi:
    def __init__(self) -> None:
        self.mutations = 0

    def request(
        self,
        method: str,
        path: str,
        payload: dict[str, object] | None = None,
    ) -> dict[str, object]:
        del payload
        from ptsip.app.github_authority import GitHubApiError

        if method == "GET" and path.endswith("git/ref/heads/ptsip-policy"):
            raise GitHubApiError("missing", 404)
        self.mutations += 1
        raise AssertionError(f"read-only lookup unexpectedly mutated authority: {method} {path}")


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path, name: str) -> Path:
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


def _development_tooling_args(repo: Path, *extra: str) -> list[str]:
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
        *extra,
    ]


def _product_args(repo: Path, *extra: str) -> list[str]:
    return [
        "adopt",
        str(repo),
        "--component",
        "tools",
        "--classification",
        "PRODUCT",
        "--purpose",
        "Product runtime component",
        "--shipped",
        "yes",
        "--runtime-required",
        "yes",
        "--executable",
        "yes",
        "--apply",
        "--json",
        *extra,
    ]


def _install_shared_client(monkeypatch, shared: MemoryAuthority) -> None:
    def github_client(repository: str):
        return GithubControlPlaneClient(repository, store=shared)

    monkeypatch.setattr(cli_module, "GithubControlPlaneClient", github_client)


def test_read_only_peek_does_not_bootstrap_missing_authority() -> None:
    from ptsip.app.github_authority import GitHubAuthorityStore

    api = MissingAuthorityApi()
    store = GitHubAuthorityStore("example/project", api=api)  # type: ignore[arg-type]
    head, record = store.read_json_if_exists("decisions/gdec-example.json")
    assert head is None
    assert record is None
    assert api.mutations == 0


def test_complete_equivalent_local_profile_still_checks_authority(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo_a = _repo(tmp_path, "repo-a")
    repo_b = _repo(tmp_path, "repo-b")
    shared = MemoryAuthority()
    _install_shared_client(monkeypatch, shared)

    assert main(_development_tooling_args(repo_a)) == 0
    capsys.readouterr()
    assert main(_development_tooling_args(repo_b, "--coordination", "local")) == 0
    capsys.readouterr()

    before = (repo_b / "ptsip.yaml").read_text(encoding="utf-8")
    assert main(["gate", str(repo_b), "--component", "tools", "--json"]) == 0
    gated = json.loads(capsys.readouterr().out)

    assert gated["status"] == "RESOLVED"
    assert gated["backend"] == "GITHUB"
    assert gated["decisions"][0]["reconciliation"]["status"] == "CONSISTENT"
    assert "lifecycle_owner" not in gated["decisions"][0]["decision"]["answer"]
    assert (repo_b / "ptsip.yaml").read_text(encoding="utf-8") == before


def test_complete_conflicting_local_profile_returns_explicit_conflict_without_overwrite(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo_a = _repo(tmp_path, "repo-a")
    repo_b = _repo(tmp_path, "repo-b")
    shared = MemoryAuthority()
    _install_shared_client(monkeypatch, shared)

    assert main(_development_tooling_args(repo_a)) == 0
    capsys.readouterr()
    assert main(_product_args(repo_b, "--coordination", "local")) == 0
    capsys.readouterr()

    before = (repo_b / "ptsip.yaml").read_text(encoding="utf-8")
    assert main(["gate", str(repo_b), "--component", "tools", "--json"]) == 8
    gated = json.loads(capsys.readouterr().out)

    assert gated["status"] == "AUTHORITY_PROFILE_CONFLICT"
    assert gated["backend"] == "GITHUB"
    assert gated["decisions"][0]["status"] == "AUTHORITY_PROFILE_CONFLICT"
    assert (repo_b / "ptsip.yaml").read_text(encoding="utf-8") == before
    profile = yaml.safe_load(before)
    assert profile["components"][0]["classification"] == "PRODUCT"


def test_complete_local_declaration_without_remote_record_does_not_fabricate_authority_history(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path, "repo")
    shared = MemoryAuthority()
    _install_shared_client(monkeypatch, shared)

    assert main(_development_tooling_args(repo, "--coordination", "local")) == 0
    capsys.readouterr()
    assert shared.documents == {}

    assert main(["gate", str(repo), "--component", "tools", "--json"]) == 0
    gated = json.loads(capsys.readouterr().out)

    assert gated["status"] == "NO_DECISION_REQUIRED"
    assert gated["backend"] == "GITHUB"
    assert gated["decisions"][0]["reconciliation"]["status"] == "LOCAL_DECLARATION_ONLY"
    assert shared.documents == {}


def test_github_coordination_unavailable_fails_closed_with_json_and_no_local_store(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path, "repo")
    monkeypatch.setattr(cli_module, "GithubControlPlaneClient", lambda repository: UnavailableClient())
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    assert main(["gate", str(repo), "--component", "tools", "--json"]) == 8
    gated = json.loads(capsys.readouterr().out)

    assert gated["status"] == "COORDINATION_UNAVAILABLE"
    assert gated["backend"] == "GITHUB"
    assert not decision_store_path(repo).exists()


def test_application_receipt_is_explicitly_local_projection() -> None:
    client = GithubControlPlaneClient("example/project", store=MemoryAuthority())
    receipt = client.application(
        {
            "decision_id": "gdec-example",
            "status": "LOCAL_APPLIED",
            "applied_revision": "a" * 40,
        }
    )
    assert receipt["scope"] == "LOCAL_PROJECTION"
    assert receipt["status"] == "LOCAL_APPLIED"
