from __future__ import annotations

import subprocess
from pathlib import Path
from types import SimpleNamespace

from developer.automation.pp import pp_pre_commit
from developer.automation.dev_setup import HOOKS_PATH, install_developer_hooks
from developer.automation.pp.pp_transition_delta import T2DeltaResult


ROOT = Path(__file__).resolve().parents[2]


def _run(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return result.stdout.strip()


def test_shared_installer_sets_repository_local_hooks_path(tmp_path: Path) -> None:
    _run(tmp_path, "init")
    hook = tmp_path / ".githooks" / "pre-commit"
    hook.parent.mkdir(parents=True)
    hook.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8", newline="\n")

    installed = install_developer_hooks(tmp_path)

    assert installed == hook
    assert _run(tmp_path, "config", "--local", "--get", "core.hooksPath") == HOOKS_PATH


def test_repository_entrypoints_share_canonical_installer() -> None:
    setup = (ROOT / "setup_dev.bat").read_text(encoding="utf-8")
    bootstrap = (ROOT / "bootstrap_repo.ps1").read_text(encoding="utf-8")
    hook = (ROOT / ".githooks" / "pre-commit").read_text(encoding="utf-8")

    assert "python -m developer.automation.dev_setup" in setup
    assert "python -m developer.automation.dev_setup" in bootstrap
    assert "developer.automation.pp.pp_pre_commit" in hook
    assert "git add -A" not in hook


def test_pre_commit_execution_layer_delegates_to_reconciler_and_verifier(
    tmp_path: Path,
    monkeypatch,
) -> None:
    delta = T2DeltaResult(
        classification="NO_T2_AUTHORITY_DELTA",
        triggered=False,
        valid=True,
        reasons=(),
        base_current="pp.1.01",
        candidate_current="pp.1.01",
        expected_next=None,
        candidate_already_reconciled=False,
    )
    monkeypatch.setattr(pp_pre_commit, "repository_root", lambda root=None: tmp_path)
    monkeypatch.setattr(
        pp_pre_commit,
        "verify_staged_parent_authority",
        lambda root: None,
    )
    monkeypatch.setattr(
        pp_pre_commit,
        "reconcile_staged_transition",
        lambda root, apply=False: SimpleNamespace(status="NO_CHANGE"),
    )
    monkeypatch.setattr(
        pp_pre_commit,
        "validate_project_profile_registry_plane",
        lambda root: (),
    )
    monkeypatch.setattr(
        pp_pre_commit,
        "compare_git_snapshots",
        lambda root, base_revision, staged: delta,
    )

    result = pp_pre_commit.verify_staged_pp_transition(tmp_path)

    assert result["status"] == "PASS"
    assert result["reconciliation"] == "NO_CHANGE"
    assert result["classification"] == "NO_T2_AUTHORITY_DELTA"
