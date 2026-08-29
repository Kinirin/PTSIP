from __future__ import annotations

import json
import subprocess
from pathlib import Path

from ptsip.cli import main
from ptsip.storage.local_state import decision_store_path


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path) -> Path:
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


def _adopt_args(repo: Path, *, profile: Path | None = None) -> list[str]:
    args = [
        "adopt",
        str(repo),
        "--component",
        "tools",
        "--classification",
        "DEVELOPMENT_TOOLING",
        "--purpose",
        "Repository-local generation tooling",
        "--shipped",
        "no",
        "--runtime-required",
        "no",
        "--executable",
        "yes",
        "--coordination",
        "local",
        "--json",
        "--apply",
    ]
    if profile is not None:
        args.extend(["--profile", str(profile)])
    return args


def test_adopted_explicit_profile_is_seen_by_clarify_and_gate(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    profile = repo / "config" / "ptsip.yaml"
    profile.parent.mkdir(parents=True, exist_ok=True)

    assert main(_adopt_args(repo, profile=profile)) == 0
    capsys.readouterr()
    assert profile.is_file()
    assert not (repo / "ptsip.yaml").exists()

    assert main(["clarify", str(repo), "--component", "tools", "--profile", str(profile), "--json"]) == 0
    clarified = json.loads(capsys.readouterr().out)
    assert clarified["status"] == "NO_CLARIFICATION_REQUIRED"
    assert clarified["clarification_count"] == 0
    assert Path(str(clarified["profile"]["path"])) == profile.resolve()

    assert main([
        "gate",
        str(repo),
        "--component",
        "tools",
        "--profile",
        str(profile),
        "--coordination",
        "local",
        "--json",
    ]) == 0
    gated = json.loads(capsys.readouterr().out)
    assert gated["status"] == "NO_DECISION_REQUIRED"
    assert gated["backend"] == "LOCAL"
    assert not decision_store_path(repo).exists()


def test_repeated_gate_does_not_reopen_applied_architecture(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    assert main([
        "gate",
        str(repo),
        "--component",
        "tools",
        "--coordination",
        "local",
        "--json",
    ]) == 7
    first = json.loads(capsys.readouterr().out)
    decision_id = str(first["decisions"][0]["decision"]["id"])

    assert main([
        "resolve",
        str(repo),
        "--decision",
        decision_id,
        "--classification",
        "DEVELOPMENT_TOOLING",
        "--purpose",
        "Repository-local generation tooling",
        "--shipped",
        "no",
        "--runtime-required",
        "no",
        "--executable",
        "yes",
        "--coordination",
        "local",
        "--json",
    ]) == 0
    capsys.readouterr()

    assert main([
        "gate",
        str(repo),
        "--component",
        "tools",
        "--coordination",
        "local",
        "--json",
    ]) == 0
    repeated = json.loads(capsys.readouterr().out)
    assert repeated["status"] == "NO_DECISION_REQUIRED"
    assert repeated["decisions"] == []
