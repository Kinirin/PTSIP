from __future__ import annotations

import json
import subprocess
from pathlib import Path

import yaml

from ptsip.cli import main
from ptsip.storage.local_state import decision_store_path


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
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


def _adopt_args(repo: Path, *, apply: bool = False, profile: Path | None = None) -> list[str]:
    args = [
        "adopt",
        str(repo),
        "--component",
        "tools",
        "--classification",
        "TOOLCHAIN",
        "--purpose",
        "Repository-local generation tooling",
        "--shipped",
        "no",
        "--runtime-required",
        "no",
        "--lifecycle-owner",
        "DEVELOPMENT_TOOLING",
        "--executable",
        "yes",
        "--coordination",
        "local",
        "--json",
    ]
    if profile is not None:
        args.extend(["--profile", str(profile)])
    if apply:
        args.append("--apply")
    return args


def test_adopt_is_dry_run_by_default_and_apply_persists_full_declaration(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    before = _git(repo, "status", "--porcelain").stdout
    assert before == ""
    assert main(_adopt_args(repo)) == 0
    plan = json.loads(capsys.readouterr().out)
    assert plan["format"] == "ptsip-adoption/v1"
    assert plan["status"] == "ADOPTION_PLAN"
    assert plan["apply"] is False
    assert plan["backend"] == "LOCAL"
    assert not (repo / "ptsip.yaml").exists()
    assert _git(repo, "status", "--porcelain").stdout == before

    assert main(_adopt_args(repo, apply=True)) == 0
    adopted = json.loads(capsys.readouterr().out)
    assert adopted["status"] == "ADOPTED"
    profile = repo / "ptsip.yaml"
    document = yaml.safe_load(profile.read_text(encoding="utf-8"))
    component = next(item for item in document["components"] if item["id"] == "tools")
    assert component == {
        "id": "tools",
        "include": ["tools/**"],
        "classification": "TOOLCHAIN",
        "purpose": "Repository-local generation tooling",
        "shipped": False,
        "runtime_required": False,
        "executable": True,
        "release_owner": "DEVELOPMENT_TOOLING",
    }
    assert not decision_store_path(repo).exists()

    assert main(_adopt_args(repo, apply=True)) == 0
    repeated = json.loads(capsys.readouterr().out)
    assert repeated["status"] == "ALREADY_DECLARED"


def test_adopt_explicit_profile_is_seen_by_clarify_and_gate(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    config = repo / "config"
    config.mkdir()
    profile = config / "ptsip.yaml"

    assert main(_adopt_args(repo, apply=True, profile=profile)) == 0
    capsys.readouterr()
    assert profile.is_file()
    assert not (repo / "ptsip.yaml").exists()

    assert main(["clarify", str(repo), "--component", "tools", "--profile", str(profile), "--json"]) == 0
    clarified = json.loads(capsys.readouterr().out)
    assert clarified["status"] == "NO_CLARIFICATION_REQUIRED"
    assert clarified["clarification_count"] == 0
    assert Path(str(clarified["profile"]["path"])) == profile.resolve()

    assert main(
        [
            "gate",
            str(repo),
            "--component",
            "tools",
            "--profile",
            str(profile),
            "--coordination",
            "local",
            "--json",
        ]
    ) == 0
    gated = json.loads(capsys.readouterr().out)
    assert gated["status"] == "NO_DECISION_REQUIRED"
    assert gated["backend"] == "LOCAL"
    assert not decision_store_path(repo).exists()


def test_invalid_or_unknown_adoption_never_writes_profile(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)

    invalid = _adopt_args(repo, apply=True)
    purpose_index = invalid.index("--runtime-required") + 1
    invalid[purpose_index] = "yes"
    assert main(invalid) == 8
    conflict = json.loads(capsys.readouterr().out)
    assert conflict["status"] == "CONFLICT"
    assert not (repo / "ptsip.yaml").exists()

    unknown = _adopt_args(repo, apply=True)
    unknown[unknown.index("tools")] = "does-not-exist"
    assert main(unknown) == 8
    missing = json.loads(capsys.readouterr().out)
    assert missing["status"] == "UNKNOWN_COMPONENT"
    assert not (repo / "ptsip.yaml").exists()
