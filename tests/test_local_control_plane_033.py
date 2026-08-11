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


def _tool_repo(tmp_path: Path) -> Path:
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


def _resolve_args(repo: Path, decision_id: str, classification: str = "TOOLCHAIN") -> list[str]:
    if classification == "TOOLCHAIN":
        return [
            "resolve",
            str(repo),
            "--decision",
            decision_id,
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
            "--json",
        ]
    return [
        "resolve",
        str(repo),
        "--decision",
        decision_id,
        "--classification",
        "PRODUCT",
        "--purpose",
        "Product runtime component",
        "--shipped",
        "yes",
        "--runtime-required",
        "yes",
        "--lifecycle-owner",
        "PRODUCT",
        "--executable",
        "yes",
        "--json",
    ]


def test_gate_and_resolve_use_local_control_plane_without_server_or_github_origin(
    tmp_path: Path,
    monkeypatch,
    capsys,
):
    repo = _tool_repo(tmp_path)
    state = tmp_path / "ptsip-state"
    monkeypatch.setenv("PTSIP_HOME", str(state))
    monkeypatch.delenv("PTSIP_CONTROL_PLANE_URL", raising=False)
    monkeypatch.delenv("PTSIP_CONTROL_PLANE_TOKEN", raising=False)

    before_status = _git(repo, "status", "--porcelain").stdout
    assert before_status == ""

    exit_code = main(["gate", str(repo), "--component", "tools", "--json"])
    assert exit_code == 7
    gate_payload = json.loads(capsys.readouterr().out)
    assert gate_payload["status"] == "DECISION_REQUIRED"
    assert gate_payload["backend"] == "LOCAL"
    assert len(gate_payload["decisions"]) == 1
    gated = gate_payload["decisions"][0]
    assert gated["backend"] == "LOCAL"
    assert gated["status"] == "DECISION_REQUIRED"
    decision = gated["decision"]
    assert decision["repository"].startswith("local:")
    assert decision["status"] == "PENDING"
    assert decision["application_status"] == "NOT_APPLIED"

    store_path = decision_store_path(repo)
    assert store_path.is_file()
    assert state in store_path.parents
    assert not (repo / "ptsip-control-plane.sqlite3").exists()
    assert _git(repo, "status", "--porcelain").stdout == before_status

    exit_code = main(_resolve_args(repo, str(decision["id"])))
    assert exit_code == 0
    resolved_payload = json.loads(capsys.readouterr().out)
    assert resolved_payload["status"] == "RESOLVED"
    assert resolved_payload["backend"] == "LOCAL"
    assert resolved_payload["application"]["backend"] == "LOCAL"
    assert resolved_payload["application"]["status"] == "LOCAL_APPLIED"

    profile = repo / "ptsip.yaml"
    assert profile.is_file()
    document = yaml.safe_load(profile.read_text(encoding="utf-8"))
    component = next(item for item in document["components"] if item["id"] == "tools")
    assert component["classification"] == "TOOLCHAIN"
    assert component["release_owner"] == "DEVELOPMENT_TOOLING"

    assert main(["validate", str(repo), "--profile", str(profile), "--json"]) == 0
    validation_payload = json.loads(capsys.readouterr().out)
    assert validation_payload["valid"] is True


def test_local_first_valid_resolution_cannot_be_replaced(tmp_path: Path, monkeypatch, capsys):
    repo = _tool_repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "ptsip-state"))
    monkeypatch.delenv("PTSIP_CONTROL_PLANE_URL", raising=False)
    monkeypatch.delenv("PTSIP_CONTROL_PLANE_TOKEN", raising=False)

    assert main(["gate", str(repo), "--component", "tools", "--json"]) == 7
    gate_payload = json.loads(capsys.readouterr().out)
    decision_id = str(gate_payload["decisions"][0]["decision"]["id"])

    assert main(_resolve_args(repo, decision_id, "TOOLCHAIN")) == 0
    capsys.readouterr()

    assert main(_resolve_args(repo, decision_id, "PRODUCT")) == 9
    later_payload = json.loads(capsys.readouterr().out)
    assert later_payload["status"] == "ALREADY_RESOLVED"
    assert later_payload["accepted"] is False
    assert later_payload["decision"]["answer"]["classification"] == "TOOLCHAIN"
