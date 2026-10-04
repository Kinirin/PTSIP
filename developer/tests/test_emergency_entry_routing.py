from __future__ import annotations

from pathlib import Path

from developer.automation.planning.planning_entry_resolver import (
    EMERGENCY_OVERLAY_PATH,
    resolve_planning_entry,
)


ROOT = Path(__file__).resolve().parents[2]


def test_dev_038a1_resolves_exact_emergency_overlay(tmp_path: Path) -> None:
    (tmp_path / "pyproject.toml").write_text("[project]\nname = 'test'\n", encoding="utf-8")
    overlay = tmp_path / EMERGENCY_OVERLAY_PATH
    overlay.parent.mkdir(parents=True)
    overlay.write_text(
        "branch:\n  name: dev/0.3.8a1\nplan_version: 0.3.8a1\n", encoding="utf-8"
    )
    result = resolve_planning_entry("dev/0.3.8a1", root=tmp_path)
    assert result.status == "RESOLVED"
    assert result.branch == "dev/0.3.8a1"
    assert result.plan_version == "0.3.8a1"
    assert (
        result.entry_document
        == EMERGENCY_OVERLAY_PATH
    )
    assert result.role == "EMERGENCY_RELEASE_OVERLAY"
    assert result.work_unit is None
