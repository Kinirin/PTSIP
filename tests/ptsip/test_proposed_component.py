from __future__ import annotations

import json
import subprocess
from pathlib import Path

import yaml

from ptsip.cli import main
from ptsip.proposed_component import (
    ProposedComponentError,
    build_proposed_component_candidate,
)


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _init_repo(repo: Path) -> None:
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "README.md").write_text("# fixture\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def test_proposed_candidate_identity_is_deterministic_without_filesystem_discovery(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    root.mkdir()
    assert not (root / "future").exists()

    first = build_proposed_component_candidate(
        "local:repository",
        "future-service",
        ["future/service/**", "generated/api/**"],
        "ptsip.yaml",
    )
    second = build_proposed_component_candidate(
        "local:repository",
        "future-service",
        ["generated/api/**", "future\\service\\**"],
        "ptsip.yaml",
    )

    assert first == second
    assert first.as_dict()["materialized"] is False
    assert first.as_dict()["authoritative"] is False
    assert not (root / "future").exists()


def test_proposed_candidate_rejects_ambiguous_normalized_selectors() -> None:
    try:
        build_proposed_component_candidate(
            "local:repository",
            "future-service",
            ["future/service/**", "future\\service\\**"],
            "ptsip.yaml",
        )
    except ProposedComponentError as exc:
        assert "unique" in str(exc)
    else:
        raise AssertionError("duplicate normalized selectors must fail closed")


def test_cli_registers_nonexistent_component_then_existing_resolve_flow_applies_declaration(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = tmp_path / "repo"
    _init_repo(repo)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    assert not (repo / "future").exists()

    assert (
        main(
            [
                "propose-component",
                str(repo),
                "--component",
                "future-service",
                "--include",
                "future/service/**",
                "--coordination",
                "local",
                "--json",
            ]
        )
        == 0
    )
    registered = json.loads(capsys.readouterr().out)
    assert registered["status"] == "DECISION_REQUIRED"
    assert registered["candidate"]["component_id"] == "future-service"
    assert registered["candidate"]["materialized"] is False
    assert registered["candidate"]["authoritative"] is False
    assert not (repo / "future").exists()

    decision = registered["decision"]
    assert isinstance(decision, dict)
    decision_id = str(decision["id"])

    assert (
        main(
            [
                "resolve",
                str(repo),
                "--decision",
                decision_id,
                "--classification",
                "DEVELOPMENT_TOOLING",
                "--purpose",
                "Future repository development service",
                "--shipped",
                "no",
                "--runtime-required",
                "no",
                "--executable",
                "yes",
                "--coordination",
                "local",
                "--json",
            ]
        )
        == 0
    )
    resolved = json.loads(capsys.readouterr().out)
    assert resolved["status"] in {"RESOLVED", "ALREADY_APPLIED", "LOCAL_APPLIED"}

    profile = yaml.safe_load((repo / "ptsip.yaml").read_text(encoding="utf-8"))
    components = profile["components"]
    component = next(item for item in components if item["id"] == "future-service")
    assert component["include"] == ["future/service/**"]
    assert component["classification"] == "DEVELOPMENT_TOOLING"
    assert not (repo / "future").exists()
