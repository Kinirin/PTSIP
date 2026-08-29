from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.pilot.runner import run_pilot
from ptsip.repository.snapshot import capture_snapshot, compare_snapshots


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


def test_pilot_v2_writes_external_state_only(tmp_path: Path, monkeypatch) -> None:
    repo = tmp_path / "repo"
    state = tmp_path / "state"
    _init_git(repo)
    (repo / "src").mkdir()
    (repo / "src" / "x.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    monkeypatch.setenv("PTSIP_HOME", str(state))
    before = capture_snapshot(repo)
    result = run_pilot(repo)
    after = capture_snapshot(repo)
    assert result.report["format"] == "ptsip-pilot-report/v2"
    assert result.report_path.is_file()
    assert state in result.report_path.parents
    assert result.report["snapshot"]["comparison"]["stable"] is True
    assert result.report["non_intrusion"]["status"] == "VERIFIED_NO_OBSERVED_CHANGE"
    assert compare_snapshots(before, after).stable
    assert result.report["classification"]["allowed_classifications"] == [
        "PRODUCT",
        "DEVELOPMENT_TOOLING",
        "DELIVERY",
        "OPERATIONS",
        "NEUTRAL_CONTRACT",
    ]
    assert "UNKNOWN" in result.report["classification"]["decision_statuses"]
    evaluator = result.report["evaluation"]["declared_dependency_boundaries"]
    assert evaluator["status"] == "BLOCKED"
    assert evaluator["reason"] == "NO_PROFILE"
    assert evaluator["finding_count"] is None
