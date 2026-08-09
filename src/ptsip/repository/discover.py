from __future__ import annotations

import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path


@dataclass(frozen=True)
class RepositoryInfo:
    requested_path: str
    root: str
    is_git: bool
    commit: str | None
    branch: str | None
    dirty: bool | None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def _git(path: Path, *args: str) -> tuple[int, str]:
    proc = subprocess.run(
        ["git", "-C", str(path), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
        check=False,
    )
    return proc.returncode, proc.stdout.strip()


def discover_repository(path: str | Path = ".") -> RepositoryInfo:
    requested = Path(path).expanduser().resolve()
    if not requested.exists():
        raise FileNotFoundError(f"Path does not exist: {requested}")
    if not requested.is_dir():
        requested = requested.parent

    code, root_text = _git(requested, "rev-parse", "--show-toplevel")
    if code != 0:
        return RepositoryInfo(
            requested_path=str(requested),
            root=str(requested),
            is_git=False,
            commit=None,
            branch=None,
            dirty=None,
        )

    root = Path(root_text).resolve()
    _, commit = _git(root, "rev-parse", "HEAD")
    branch_code, branch = _git(root, "branch", "--show-current")
    status_code, status = _git(root, "status", "--porcelain")
    return RepositoryInfo(
        requested_path=str(requested),
        root=str(root),
        is_git=True,
        commit=commit or None,
        branch=(branch if branch_code == 0 and branch else None),
        dirty=(bool(status) if status_code == 0 else None),
    )
