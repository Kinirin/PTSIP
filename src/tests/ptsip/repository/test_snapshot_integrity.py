from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.repository.snapshot import capture_snapshot, compare_snapshots


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _init_git(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit_all(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def test_snapshot_detects_repository_change(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    target = repo / "app.py"
    target.write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    before = capture_snapshot(repo)
    target.write_text("x = 2\n", encoding="utf-8")
    after = capture_snapshot(repo)
    comparison = compare_snapshots(before, after)
    assert not comparison.stable
    assert comparison.status == "INVALIDATED"
    assert any(
        "tracked file content" in reason or "working-tree" in reason
        for reason in comparison.reasons
    )
