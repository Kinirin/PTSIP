from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import sys
import sysconfig
from datetime import datetime, timezone
from pathlib import Path

import pytest

from ptsip.validation_capture import ValidationCaptureError, capture_validation_command


def _git(root: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", "-C", str(root), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
    )
    return completed.stdout.decode("utf-8", errors="replace").strip()


def _repository(root: Path) -> None:
    _git(root, "init", "-q")
    _git(root, "config", "user.email", "ptsip-test@example.invalid")
    _git(root, "config", "user.name", "PTSIP Test")
    (root / "README.md").write_text("baseline\n", encoding="utf-8")
    _git(root, "add", "README.md")
    _git(root, "commit", "-q", "-m", "baseline")


def _display_command(command: list[str]) -> str:
    if os.name == "nt":
        return subprocess.list2cmdline(command)
    return shlex.join(command)


def _console_script() -> str:
    executable_name = "ptsip.exe" if os.name == "nt" else "ptsip"
    candidates = [
        Path(sys.executable).with_name(executable_name),
        Path(sysconfig.get_path("scripts")) / executable_name,
    ]
    user_scheme = "nt_user" if os.name == "nt" else "posix_user"
    if user_scheme in sysconfig.get_scheme_names():
        candidates.append(Path(sysconfig.get_path("scripts", scheme=user_scheme)) / executable_name)
    resolved = shutil.which("ptsip")
    if resolved:
        candidates.insert(0, Path(resolved))
    for candidate in dict.fromkeys(path.resolve() for path in candidates):
        if candidate.is_file():
            return str(candidate)
    rendered = ", ".join(str(path) for path in candidates)
    raise AssertionError(f"PTSIP console script is not installed; checked: {rendered}")


def _run_cli(root: Path, surface: str, child: list[str]) -> subprocess.CompletedProcess[str]:
    prefix = (
        [_console_script()]
        if surface == "console-script"
        else [sys.executable, "-m", "ptsip"]
    )
    return subprocess.run(
        [*prefix, "validation", "capture", "--", *child],
        cwd=root,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )


def test_capture_commits_only_generated_log_and_preserves_unrelated_index(tmp_path: Path) -> None:
    _repository(tmp_path)
    (tmp_path / "README.md").write_text("baseline\nstaged user work\n", encoding="utf-8")
    _git(tmp_path, "add", "README.md")
    (tmp_path / "scratch.txt").write_text("untracked user work\n", encoding="utf-8")
    captured_at = datetime(2026, 8, 28, 11, 17, 0, 123456, tzinfo=timezone.utc)

    command = [
        sys.executable,
        "-c",
        "import sys; print('only-this-command'); print('captured-stderr', file=sys.stderr)",
    ]
    result = capture_validation_command(tmp_path, command, now=captured_at)

    assert result.exit_code == 0
    assert result.log_path == "docs/ptsip/validation/20260828T111700123456+0000.log"
    assert result.commit_sha == _git(tmp_path, "rev-parse", "HEAD")
    committed_paths = _git(tmp_path, "show", "--pretty=format:", "--name-only", "HEAD").splitlines()
    assert committed_paths == [result.log_path]
    assert _git(tmp_path, "diff", "--cached", "--name-only").splitlines() == ["README.md"]
    assert (tmp_path / "scratch.txt").exists()

    log = (tmp_path / result.log_path).read_text(encoding="utf-8")
    assert "captured_at: 2026-08-28T11:17:00.123456+00:00" in log
    assert f"command: {_display_command(command)}" in log
    assert "exit_code: 0" in log
    assert "only-this-command" in log
    assert "captured-stderr" in log
    assert "validate" not in log
    assert "conform" not in log


def test_capture_commits_failed_command_output_and_returns_child_exit_code(tmp_path: Path) -> None:
    _repository(tmp_path)
    (tmp_path / "README.md").write_text("baseline\nstaged user work\n", encoding="utf-8")
    _git(tmp_path, "add", "README.md")
    (tmp_path / "scratch.txt").write_text("untracked user work\n", encoding="utf-8")

    result = capture_validation_command(
        tmp_path,
        [
            sys.executable,
            "-c",
            "import sys; print('failed stdout'); print('failed stderr', file=sys.stderr); sys.exit(7)",
        ],
        now=datetime(2026, 8, 28, 11, 18, 0, tzinfo=timezone.utc),
    )

    assert result.exit_code == 7
    assert _git(tmp_path, "show", "--pretty=format:", "--name-only", "HEAD").strip() == result.log_path
    log = (tmp_path / result.log_path).read_text(encoding="utf-8")
    assert "exit_code: 7" in log
    assert "failed stdout" in log
    assert "failed stderr" in log
    assert _git(tmp_path, "diff", "--cached", "--name-only").splitlines() == ["README.md"]
    assert (tmp_path / "scratch.txt").read_text(encoding="utf-8") == "untracked user work\n"


def test_capture_does_not_need_or_create_stage_named_logs(tmp_path: Path) -> None:
    _repository(tmp_path)

    result = capture_validation_command(
        tmp_path,
        [sys.executable, "-c", "print('custom sdk verification')"],
        now=datetime(2026, 8, 28, 11, 19, 0, tzinfo=timezone.utc),
    )

    validation_dir = tmp_path / "docs" / "ptsip" / "validation"
    assert sorted(path.name for path in validation_dir.iterdir()) == [Path(result.log_path).name]
    assert "doctor" not in Path(result.log_path).name
    assert "validate" not in Path(result.log_path).name
    assert "conform" not in Path(result.log_path).name


def test_repeated_capture_creates_new_logs_without_overwriting_prior_evidence(tmp_path: Path) -> None:
    _repository(tmp_path)

    first = capture_validation_command(
        tmp_path,
        [sys.executable, "-c", "print('first capture')"],
        now=datetime(2026, 8, 28, 11, 20, 0, tzinfo=timezone.utc),
    )
    second = capture_validation_command(
        tmp_path,
        [sys.executable, "-c", "print('second capture')"],
        now=datetime(2026, 8, 28, 11, 20, 1, tzinfo=timezone.utc),
    )

    assert first.log_path != second.log_path
    assert (tmp_path / first.log_path).read_text(encoding="utf-8").endswith("first capture\n")
    assert (tmp_path / second.log_path).read_text(encoding="utf-8").endswith("second capture\n")
    assert not (tmp_path / "docs" / "ptsip" / "validation" / "summary.md").exists()


def test_capture_never_pushes_its_local_commit(tmp_path: Path) -> None:
    repository = tmp_path / "repository"
    remote = tmp_path / "remote.git"
    repository.mkdir()
    _repository(repository)
    subprocess.run(["git", "init", "--bare", "-q", str(remote)], check=True)
    _git(repository, "remote", "add", "origin", str(remote))
    _git(repository, "push", "-q", "origin", "HEAD:main")
    remote_before = _git(repository, "ls-remote", "origin", "refs/heads/main").split()[0]

    result = capture_validation_command(
        repository,
        [sys.executable, "-c", "print('local only')"],
        now=datetime(2026, 8, 28, 11, 21, 0, tzinfo=timezone.utc),
    )

    assert result.commit_sha != remote_before
    assert _git(repository, "ls-remote", "origin", "refs/heads/main").split()[0] == remote_before


def test_both_cli_surfaces_capture_success_and_failure_exit_codes(tmp_path: Path) -> None:
    for surface in ("console-script", "module"):
        for exit_code in (0, 7):
            repository = tmp_path / f"{surface}-{exit_code}"
            repository.mkdir()
            _repository(repository)
            child = [
                sys.executable,
                "-c",
                f"import sys; print('{surface} stdout'); print('{surface} stderr', file=sys.stderr); sys.exit({exit_code})",
            ]

            completed = _run_cli(repository, surface, child)

            assert completed.returncode == exit_code
            assert completed.stderr == ""
            assert "log_path: docs/ptsip/validation/" in completed.stdout
            log_paths = list((repository / "docs" / "ptsip" / "validation").glob("*.log"))
            assert len(log_paths) == 1
            log = log_paths[0].read_text(encoding="utf-8")
            assert f"command: {_display_command(child)}" in log
            assert f"exit_code: {exit_code}" in log
            assert f"{surface} stdout" in log
            assert f"{surface} stderr" in log
            assert _git(repository, "show", "--pretty=format:", "--name-only", "HEAD").strip() == (
                log_paths[0].relative_to(repository).as_posix()
            )


def test_cli_fails_closed_without_separator_or_resolvable_executable(tmp_path: Path) -> None:
    _repository(tmp_path)
    module = [sys.executable, "-m", "ptsip", "validation", "capture"]

    missing_separator = subprocess.run(
        [*module, sys.executable],
        cwd=tmp_path,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    missing_executable = subprocess.run(
        [*module, "--", "ptsip-command-that-does-not-exist"],
        cwd=tmp_path,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )

    assert missing_separator.returncode == 2
    assert "Usage: ptsip validation capture -- <command> [args...]" in missing_separator.stderr
    assert missing_executable.returncode == 2
    assert "was not found on PATH" in missing_executable.stderr
    assert not (tmp_path / "docs").exists()


def test_capture_executes_once_and_preserves_literal_child_arguments(tmp_path: Path) -> None:
    repository = tmp_path / "repository"
    repository.mkdir()
    _repository(repository)
    counter = tmp_path / "execution-count.txt"
    literal_args = ["space value", "ampersand&echo-not-run", "$(echo-not-run)"]
    child = [
        sys.executable,
        "-c",
        (
            "from pathlib import Path; import sys; "
            "p=Path(sys.argv[1]); "
            "n=int(p.read_text())+1 if p.exists() else 1; "
            "p.write_text(str(n)); print(repr(sys.argv[2:]))"
        ),
        str(counter),
        *literal_args,
    ]

    result = capture_validation_command(
        repository,
        child,
        now=datetime(2026, 8, 28, 11, 22, 0, tzinfo=timezone.utc),
    )

    assert counter.read_text(encoding="utf-8") == "1"
    log = (repository / result.log_path).read_text(encoding="utf-8")
    assert repr(literal_args) in log


def test_commit_failure_keeps_generated_log_for_user_recovery(tmp_path: Path) -> None:
    _repository(tmp_path)
    hook = tmp_path / ".git" / "hooks" / "pre-commit"
    hook.write_text("#!/bin/sh\nexit 1\n", encoding="utf-8", newline="\n")
    hook.chmod(0o755)

    with pytest.raises(ValidationCaptureError, match="Unable to commit validation log"):
        capture_validation_command(
            tmp_path,
            [sys.executable, "-c", "print('preserve me')"],
            now=datetime(2026, 8, 28, 11, 23, 0, tzinfo=timezone.utc),
        )

    logs = list((tmp_path / "docs" / "ptsip" / "validation").glob("*.log"))
    assert len(logs) == 1
    assert "preserve me" in logs[0].read_text(encoding="utf-8")
