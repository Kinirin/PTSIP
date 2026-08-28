from __future__ import annotations

import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

from ptsip.validation_capture import capture_validation_command


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


def test_capture_commits_only_generated_log_and_preserves_unrelated_index(tmp_path: Path) -> None:
    _repository(tmp_path)
    (tmp_path / "README.md").write_text("baseline\nstaged user work\n", encoding="utf-8")
    _git(tmp_path, "add", "README.md")
    (tmp_path / "scratch.txt").write_text("untracked user work\n", encoding="utf-8")
    captured_at = datetime(2026, 8, 28, 11, 17, 0, 123456, tzinfo=timezone.utc)

    result = capture_validation_command(
        tmp_path,
        [sys.executable, "-c", "print('only-this-command')"],
        now=captured_at,
    )

    assert result.exit_code == 0
    assert result.log_path == "docs/ptsip/validation/20260828T111700123456+0000.log"
    assert result.commit_sha == _git(tmp_path, "rev-parse", "HEAD")
    committed_paths = _git(tmp_path, "show", "--pretty=format:", "--name-only", "HEAD").splitlines()
    assert committed_paths == [result.log_path]
    assert _git(tmp_path, "diff", "--cached", "--name-only").splitlines() == ["README.md"]
    assert (tmp_path / "scratch.txt").exists()

    log = (tmp_path / result.log_path).read_text(encoding="utf-8")
    assert "captured_at: 2026-08-28T11:17:00.123456+00:00" in log
    assert "exit_code: 0" in log
    assert "only-this-command" in log
    assert "validate" not in log
    assert "conform" not in log


def test_capture_commits_failed_command_output_and_returns_child_exit_code(tmp_path: Path) -> None:
    _repository(tmp_path)

    result = capture_validation_command(
        tmp_path,
        [sys.executable, "-c", "import sys; print('validation failed'); sys.exit(7)"],
        now=datetime(2026, 8, 28, 11, 18, 0, tzinfo=timezone.utc),
    )

    assert result.exit_code == 7
    assert _git(tmp_path, "show", "--pretty=format:", "--name-only", "HEAD").strip() == result.log_path
    log = (tmp_path / result.log_path).read_text(encoding="utf-8")
    assert "exit_code: 7" in log
    assert "validation failed" in log


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
