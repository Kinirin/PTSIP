from __future__ import annotations

from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[3]
WU02 = ROOT / "planning" / "0.4.0" / "WU-02"


def test_wu02_active_planning_surface_is_exact_and_small() -> None:
    work_unit = yaml.safe_load((WU02 / "WU-02.yaml").read_text(encoding="utf-8"))
    policy = work_unit["active_planning_surface"]
    assert policy["policy"] == "EXACT_FILE_SET"
    assert policy["consumed_artifact_storage"] == "GIT_HISTORY_ONLY"
    assert policy["generated_review_artifact_retention_after_consumption"] == "FORBIDDEN"
    allowed = set(policy["allowed_files"])
    actual = {path.name for path in WU02.iterdir()}
    assert actual == allowed == {
        "WU-02.yaml",
        "solution-space-deterministic-rule-coverage-routing.yaml",
    }


def test_consumed_p03_review_automation_is_not_retained_in_active_tree() -> None:
    assert list((ROOT / ".github" / "scripts").glob("p03_authority_role_*.py")) == []
    assert list((ROOT / "tests" / "ptsip").glob("test_p03_*.py")) == []
    assert not (ROOT / "tests" / "ptsip" / "test_adr_governance_assets.py").exists()


def test_governance_runtime_has_no_concrete_planning_path_dependency() -> None:
    runtime_dir = ROOT / "src" / "ptsip" / "governance"
    assert runtime_dir.is_dir()
    for path in runtime_dir.glob("*.py"):
        assert "planning/0.4.0" not in path.read_text(encoding="utf-8")
