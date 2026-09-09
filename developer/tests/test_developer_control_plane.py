from __future__ import annotations

from pathlib import Path

from developer.automation.planning_validator import validate_planning
from developer.automation.policy_validator import validate_developer_policy
from developer.automation.transition_evaluator import evaluate_legacy_decisions_removal


ROOT = Path(__file__).resolve().parents[2]


def test_developer_policy_control_plane_is_machine_valid() -> None:
    assert validate_developer_policy(ROOT) == ()


def test_developer_planning_control_plane_is_machine_valid() -> None:
    assert validate_planning(ROOT) == ()


def test_legacy_decisions_removal_is_preauthorized_but_currently_held() -> None:
    result = evaluate_legacy_decisions_removal(ROOT)
    assert result.state == "HOLD_NOT_AUTHORIZED"
    assert result.action is None
    assert result.confirmation_required is False
    assert "SRC_PTSIP_GOVERNANCE_READS_DECISIONS" in result.blockers


def test_legacy_decision_inventory_is_complete_and_boundary_classified() -> None:
    import yaml

    inventory_path = ROOT / "developer" / "policy" / "legacy-decisions-inventory.yaml"
    inventory = yaml.safe_load(inventory_path.read_text(encoding="utf-8"))
    assert inventory["summary"] == {
        "MPD": 2,
        "SFP": 15,
        "SPLIT": 6,
        "RETIRE": 0,
        "total": 23,
        "planned_sfp_targets": 21,
        "planned_mpd_targets": 5,
        "retired_fragments": 3,
    }
    assert [item["source_id"] for item in inventory["entries"]] == [
        f"ADR-{number:04d}" for number in range(1, 24)
    ]
    assert all(
        output.get("target_id") is None or not output["target_id"].startswith("ADR-")
        for item in inventory["entries"]
        for output in item["outputs"]
    )
