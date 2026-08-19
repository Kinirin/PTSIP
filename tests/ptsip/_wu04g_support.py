from __future__ import annotations

import subprocess
from pathlib import Path


def git(repo: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    """Run Git against one test-owned repository without retaining mutable global state."""
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def init_git_repo(repo: Path, *, remote_url: str | None = None) -> Path:
    """Create a fresh Git fixture repository with deterministic test identity."""
    repo.mkdir(parents=True, exist_ok=False)
    git(repo, "init")
    git(repo, "config", "user.email", "ptsip-test@example.invalid")
    git(repo, "config", "user.name", "PTSIP Test")
    if remote_url is not None:
        git(repo, "remote", "add", "origin", remote_url)
    return repo


def commit_all(repo: Path, message: str = "fixture") -> str:
    """Commit the current fixture state and return the exact resulting HEAD SHA."""
    git(repo, "add", ".")
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD").stdout.strip()


def write_text(root: Path, relative_path: str, content: str) -> Path:
    """Write one UTF-8 fixture file below root, creating only its parent directories."""
    path = root / relative_path
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return path


def clone_repo(source: Path, destination: Path, *, remote_url: str | None = None) -> Path:
    """Clone a test-owned fixture repository and optionally replace its origin URL."""
    subprocess.run(
        ["git", "clone", str(source), str(destination)],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if remote_url is not None:
        git(destination, "remote", "set-url", "origin", remote_url)
    return destination
