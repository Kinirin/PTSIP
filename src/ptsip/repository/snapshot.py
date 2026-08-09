from __future__ import annotations

import hashlib
import os
import subprocess
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path


@dataclass(frozen=True)
class RepositorySnapshot:
    captured_at: str
    is_git: bool
    head: str | None
    status_fingerprint: str
    tracked_content_fingerprint: str
    tracked_files: int
    observation_errors: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class SnapshotComparison:
    status: str
    stable: bool
    reasons: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def _run_git(root: Path, *args: str) -> tuple[int, bytes]:
    proc = subprocess.run(
        ["git", "-C", str(root), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    return proc.returncode, proc.stdout


def git_tracked_files(root: str | Path) -> tuple[list[str], list[str]]:
    root = Path(root).resolve()
    code, output = _run_git(root, "ls-files", "-z")
    if code != 0:
        return [], ["git ls-files failed"]
    paths = [item.decode("utf-8", errors="surrogateescape") for item in output.split(b"\0") if item]
    return sorted(paths), []


def _filesystem_files(root: Path) -> tuple[list[str], list[str]]:
    files: list[str] = []
    errors: list[str] = []
    stack = [root]
    while stack:
        current = stack.pop()
        try:
            entries = list(current.iterdir())
        except OSError as exc:
            errors.append(f"{current}: {exc}")
            continue
        for entry in entries:
            if entry.name == ".git":
                continue
            try:
                if entry.is_symlink() or entry.is_file():
                    files.append(entry.relative_to(root).as_posix())
                elif entry.is_dir():
                    stack.append(entry)
            except OSError as exc:
                errors.append(f"{entry}: {exc}")
    return sorted(files), errors


def repository_files(root: str | Path) -> tuple[str, list[str], list[str]]:
    root = Path(root).resolve()
    code, _ = _run_git(root, "rev-parse", "--is-inside-work-tree")
    if code == 0:
        paths, errors = git_tracked_files(root)
        return "git-tracked", paths, errors
    paths, errors = _filesystem_files(root)
    return "filesystem", paths, errors


def _content_fingerprint(root: Path, paths: list[str]) -> tuple[str, list[str]]:
    digest = hashlib.sha256()
    errors: list[str] = []
    for rel in paths:
        path = root / rel
        digest.update(rel.encode("utf-8", errors="surrogateescape"))
        digest.update(b"\0")
        try:
            if path.is_symlink():
                digest.update(b"symlink\0")
                digest.update(os.readlink(path).encode("utf-8", errors="surrogateescape"))
                digest.update(b"\0")
                continue
            with path.open("rb") as handle:
                while True:
                    chunk = handle.read(1024 * 1024)
                    if not chunk:
                        break
                    digest.update(chunk)
            digest.update(b"\0")
        except OSError as exc:
            errors.append(f"{rel}: {exc}")
            digest.update(b"<unreadable>\0")
    return digest.hexdigest(), errors


def capture_snapshot(root: str | Path) -> RepositorySnapshot:
    root = Path(root).resolve()
    git_code, _ = _run_git(root, "rev-parse", "--is-inside-work-tree")
    is_git = git_code == 0
    head: str | None = None
    errors: list[str] = []

    if is_git:
        head_code, head_bytes = _run_git(root, "rev-parse", "HEAD")
        if head_code == 0:
            head = head_bytes.decode("utf-8", errors="replace").strip() or None
        else:
            errors.append("git rev-parse HEAD failed")
        status_code, status_bytes = _run_git(
            root,
            "status",
            "--porcelain=v1",
            "-z",
            "--untracked-files=all",
            "--ignored=matching",
        )
        if status_code != 0:
            errors.append("git status failed")
            status_bytes = b"<git-status-failed>"
    else:
        status_bytes = b"<non-git>"

    _mode, paths, path_errors = repository_files(root)
    errors.extend(path_errors)
    content_fingerprint, content_errors = _content_fingerprint(root, paths)
    errors.extend(content_errors)
    return RepositorySnapshot(
        captured_at=datetime.now(timezone.utc).isoformat(),
        is_git=is_git,
        head=head,
        status_fingerprint=hashlib.sha256(status_bytes).hexdigest(),
        tracked_content_fingerprint=content_fingerprint,
        tracked_files=len(paths),
        observation_errors=tuple(errors),
    )


def compare_snapshots(before: RepositorySnapshot, after: RepositorySnapshot) -> SnapshotComparison:
    reasons: list[str] = []
    if before.is_git != after.is_git:
        reasons.append("repository git identity changed")
    if before.head != after.head:
        reasons.append("HEAD changed during observation")
    if before.status_fingerprint != after.status_fingerprint:
        reasons.append("working-tree/index/untracked-or-ignored state changed")
    if before.tracked_content_fingerprint != after.tracked_content_fingerprint:
        reasons.append("tracked file content changed")
    if before.observation_errors or after.observation_errors:
        reasons.append("snapshot observation was incomplete")
    stable = not reasons
    return SnapshotComparison(
        status="STABLE" if stable else "INVALIDATED",
        stable=stable,
        reasons=tuple(reasons),
    )
