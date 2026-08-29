from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

from ptsip.cli import _configure_console_encoding, main


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


def test_console_encoding_fallback_preserves_unencodable_text(monkeypatch) -> None:
    class Stream:
        def __init__(self) -> None:
            self.errors = "strict"

        def reconfigure(self, *, errors: str) -> None:
            self.errors = errors

    stream = Stream()
    error_stream = Stream()
    monkeypatch.setattr(sys, "stdout", stream)
    monkeypatch.setattr(sys, "stderr", error_stream)
    _configure_console_encoding()
    assert stream.errors == "backslashreplace"
    assert error_stream.errors == "backslashreplace"


def test_cli_pilot_json(tmp_path: Path, monkeypatch, capsys) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "app.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    assert main(["pilot", str(repo), "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["format"] == "ptsip-pilot-report/v2"
    assert payload["tool"]["version"] == "0.3.7"
    assert payload["non_intrusion"]["status"] == "VERIFIED_NO_OBSERVED_CHANGE"
