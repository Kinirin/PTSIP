from __future__ import annotations

from pathlib import Path

from developer.automation.agent_context_migration import verify


ROOT = Path(__file__).resolve().parents[2]


def test_current_agent_context_candidate_passes_m5_verification() -> None:
    result = verify("M5", ROOT)

    assert result["status"] == "PASS"
    assert all(check["status"] == "PASS" for check in result["checks"])
