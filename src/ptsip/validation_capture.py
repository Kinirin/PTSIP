from __future__ import annotations

import os
import shlex
import shutil
import subprocess
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Sequence


class ValidationCaptureError(RuntimeError):
    pass


@dataclass(frozen=True)
class ValidationCaptureResult:
    captured_at: str
    command: str
    exit_code: int
    log_path: str
    commit_sha: str

    def as_dict(self) -> dict[str, object]:
        return {
            "captured_at": self.captured_at,
            "command": self.command,
            "exit_code": self.exit_code,
            "log_path": self.log_path,
            "commit_sha": self.commit_sha,
            "pushed": False,
        }


def _git(root: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[bytes]:
    try:
        return subprocess.run(
            ["git", "-C", str(root), *args],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=check,
        )
    except OSError as exc:
        raise ValidationCaptureError(f"Unable to execute Git: {exc}") from exc


def _require_repository_root(root: Path) -> None:
    probe = _git(root, "rev-parse", "--show-toplevel", check=False)
    if probe.returncode != 0:
        raise ValidationCaptureError("ptsip validation capture requires a Git working tree.")
    detected = Path(probe.stdout.decode("utf-8", errors="replace").strip()).resolve()
    if detected != root:
        raise ValidationCaptureError(
            f"Validation capture root {root} is not the Git repository root {detected}."
        )


def _display_command(command: Sequence[str]) -> str:
    if os.name == "nt":
        return subprocess.list2cmdline(list(command))
    return shlex.join(command)


def _timestamp(now: datetime | None) -> datetime:
    value = now or datetime.now().astimezone()
    if value.tzinfo is None or value.utcoffset() is None:
        raise ValidationCaptureError("Validation capture timestamp must be timezone-aware.")
    return value


def _timestamp_token(value: datetime) -> str:
    return value.strftime("%Y%m%dT%H%M%S%f%z")


def _resolved_command(command: Sequence[str]) -> list[str]:
    if not command or not command[0]:
        raise ValidationCaptureError("Validation capture requires an exact command after '--'.")
    executable = shutil.which(command[0])
    if executable is None:
        raise ValidationCaptureError(f"Validation command executable {command[0]!r} was not found on PATH.")
    return [executable, *command[1:]]


def capture_validation_command(
    repository_root: str | Path,
    command: Sequence[str],
    *,
    now: datetime | None = None,
) -> ValidationCaptureResult:
    """Execute exactly one command, persist its merged output, commit only that log, and never push."""

    root = Path(repository_root).resolve()
    _require_repository_root(root)
    original_command = [str(item) for item in command]
    executable_command = _resolved_command(original_command)
    captured = _timestamp(now)
    token = _timestamp_token(captured)
    log_dir = root / "docs" / "ptsip" / "validation"
    log_path = log_dir / f"{token}.log"
    if log_path.exists():
        raise ValidationCaptureError(f"Validation log collision at {log_path}.")

    try:
        completed = subprocess.run(
            executable_command,
            cwd=root,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            shell=False,
            check=False,
        )
    except OSError as exc:
        raise ValidationCaptureError(
            f"Unable to execute validation command {original_command[0]!r}: {exc}"
        ) from exc
    output = completed.stdout.decode("utf-8", errors="backslashreplace")
    display_command = _display_command(original_command)
    header = (
        f"captured_at: {captured.isoformat(timespec='microseconds')}\n"
        f"command: {display_command}\n"
        f"exit_code: {completed.returncode}\n"
        "--- output ---\n"
    )
    try:
        log_dir.mkdir(parents=True, exist_ok=True)
        log_path.write_text(header + output, encoding="utf-8", newline="\n")
    except OSError as exc:
        raise ValidationCaptureError(
            f"Unable to persist validation log at {log_path}: {exc}"
        ) from exc

    relative = log_path.relative_to(root).as_posix()
    add = _git(root, "add", "--", relative, check=False)
    if add.returncode != 0:
        raise ValidationCaptureError(
            "Unable to stage validation log: " + add.stderr.decode("utf-8", errors="replace").strip()
        )

    commit = _git(
        root,
        "commit",
        "--only",
        "-m",
        f"chore(ptsip): capture validation {token}",
        "--",
        relative,
        check=False,
    )
    if commit.returncode != 0:
        raise ValidationCaptureError(
            "Unable to commit validation log: " + commit.stderr.decode("utf-8", errors="replace").strip()
        )

    head = _git(root, "rev-parse", "HEAD", check=False)
    if head.returncode != 0:
        raise ValidationCaptureError(
            "Unable to read validation-log commit identity: "
            + head.stderr.decode("utf-8", errors="replace").strip()
        )
    commit_sha = head.stdout.decode("ascii", errors="strict").strip()
    return ValidationCaptureResult(
        captured_at=captured.isoformat(timespec="microseconds"),
        command=display_command,
        exit_code=completed.returncode,
        log_path=relative,
        commit_sha=commit_sha,
    )
