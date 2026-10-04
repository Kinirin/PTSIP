from __future__ import annotations

import hashlib
import os
import subprocess
from pathlib import Path

from ptsip.repository.profile_path import normalize_profile_path
from ptsip.repository.snapshot import repository_files
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.guard.repository_snapshot import RepositorySnapshotExpectation

@dataclass(frozen=True)
class MutationGuardExpectation:
    head: str | None
    status_fingerprint: str
    content_fingerprint: str

    def as_dict(self) -> dict[str, object]:
        return {
            "head": self.head,
            "status_fingerprint": self.status_fingerprint,
            "content_fingerprint": self.content_fingerprint,
        }

def _sha256_bytes(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()

def _sha256_file(path: Path) -> str:
    return _sha256_bytes(path.read_bytes())

def _git(root: Path, *args: str) -> tuple[int, bytes]:
    process = subprocess.run(
        ["git", "-C", str(root), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    return process.returncode, process.stdout

def _controlled_pathspecs(paths: tuple[str, ...]) -> list[str]:
    return [f":(exclude){normalize_profile_path(item)}" for item in sorted(set(paths))]

def _fingerprint_paths(root: Path, paths: list[str]) -> str:
    digest = hashlib.sha256()
    for relative in sorted(paths):
        digest.update(relative.encode("utf-8", errors="surrogateescape"))
        digest.update(b"\0")
        path = root / relative
        if path.is_symlink():
            digest.update(b"symlink\0")
            digest.update(os.readlink(path).encode("utf-8", errors="surrogateescape"))
            digest.update(b"\0")
            continue
        try:
            digest.update(path.read_bytes())
        except OSError as exc:
            raise ExecutionStateError(f"Unable to fingerprint guarded path {relative}: {exc}") from exc
        digest.update(b"\0")
    return digest.hexdigest()

def capture_mutation_guard(repository_root: str | Path, controlled_paths: tuple[str, ...]) -> MutationGuardExpectation:
    root = Path(repository_root).expanduser().resolve()
    controlled = tuple(normalize_profile_path(item) for item in controlled_paths)
    code, _ = _git(root, "rev-parse", "--is-inside-work-tree")
    if code == 0:
        head_code, head_raw = _git(root, "rev-parse", "HEAD")
        if head_code != 0:
            raise ExecutionStateError("Unable to read git HEAD for mutation guard.")
        pathspecs = _controlled_pathspecs(controlled)
        status_code, status = _git(
            root,
            "status",
            "--porcelain=v1",
            "-z",
            "--untracked-files=all",
            "--",
            ".",
            *pathspecs,
        )
        if status_code != 0:
            raise ExecutionStateError("Unable to capture scoped git status for mutation guard.")
        tracked_code, tracked = _git(root, "ls-files", "-z", "--", ".", *pathspecs)
        if tracked_code != 0:
            raise ExecutionStateError("Unable to capture scoped tracked files for mutation guard.")
        paths = [item.decode("utf-8", errors="surrogateescape") for item in tracked.split(b"\0") if item]
        return MutationGuardExpectation(
            head=head_raw.decode("utf-8", errors="replace").strip() or None,
            status_fingerprint=hashlib.sha256(status).hexdigest(),
            content_fingerprint=_fingerprint_paths(root, paths),
        )

    _mode, paths, errors = repository_files(root)
    if errors:
        raise ExecutionStateError("Unable to capture filesystem mutation guard: " + "; ".join(errors))
    controlled_set = set(controlled)
    selected = [item for item in paths if normalize_profile_path(item) not in controlled_set]
    return MutationGuardExpectation(
        head=None,
        status_fingerprint=hashlib.sha256(b"<non-git>").hexdigest(),
        content_fingerprint=_fingerprint_paths(root, selected),
    )

