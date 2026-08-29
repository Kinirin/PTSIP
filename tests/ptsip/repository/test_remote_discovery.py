from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.repository.discover import discover_repository
from ptsip.repository.remote import parse_remote


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _tool_repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "tools").mkdir()
    (repo / "tools" / "generate.py").write_text("print('generate')\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def test_github_remote_parser_supports_https_and_ssh() -> None:
    for url in (
        "https://github.com/example/product.git",
        "git@github.com:example/product.git",
        "ssh://git@github.com/example/product.git",
    ):
        remote = parse_remote("origin", url)
        assert remote.provider == "github"
        assert remote.repository == "example/product"


def test_repository_discovery_includes_origin(tmp_path: Path) -> None:
    repo = _tool_repo(tmp_path)
    _git(repo, "remote", "add", "origin", "git@github.com:example/product.git")

    info = discover_repository(repo)

    assert info.remote is not None
    assert info.remote.provider == "github"
    assert info.remote.repository == "example/product"
