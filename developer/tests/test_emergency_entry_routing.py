from __future__ import annotations

from pathlib import Path

from developer.automation.planning_entry_resolver import resolve_planning_entry


ROOT = Path(__file__).resolve().parents[2]


def test_dev_038a1_resolves_exact_emergency_overlay() -> None:
    result = resolve_planning_entry("dev/0.3.8a1", root=ROOT)
    assert result.status == "RESOLVED"
    assert result.branch == "dev/0.3.8a1"
    assert result.plan_version == "0.3.8a1"
    assert (
        result.entry_document
        == "docs/planning/0.3.8a1/emergency-implementation-overlay.yaml"
    )
    assert result.role == "EMERGENCY_RELEASE_OVERLAY"
    assert result.work_unit is None
