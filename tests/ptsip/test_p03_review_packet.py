from __future__ import annotations

import subprocess
import sys
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]
REVIEW_PACKET = ROOT / ".github" / "scripts" / "p03_authority_role_review_packet.py"


def _run_packet(adr_id: str) -> dict[str, object]:
    result = subprocess.run(
        [
            sys.executable,
            str(REVIEW_PACKET),
            "--repo-root",
            str(ROOT),
            "--adr",
            adr_id,
        ],
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    assert result.returncode == 0, result.stderr
    payload = yaml.safe_load(result.stdout)
    assert isinstance(payload, dict)
    return payload


def test_p03_adr_0011_review_packet_is_minimal_and_machine_prepared() -> None:
    packet = _run_packet("ADR-0011")

    assert packet["target"]["adr_id"] == "ADR-0011"
    assert packet["inputs"]["dimension_count"] >= 79
    assert packet["inputs"]["runtime_authority"] == "NONE"
    assert packet["inputs"]["vocabulary_registration"] is False

    automation = packet["automation"]
    assert automation["existing_dimension_evaluation"] == "DETERMINISTIC"
    assert automation["manual_full_dimension_scan_required"] is False
    assert automation["raw_feature_force_fit"] == "FORBIDDEN"
    assert automation["natural_language_consumption"] == "FORBIDDEN"
    assert automation["new_dimension_decision"] == "NOT_AUTOMATIC"

    summary = packet["summary"]
    assert summary["raw_feature_count"] == 8
    assert (
        summary["covered_raw_feature_count"]
        + summary["residual_raw_feature_count"]
        == 8
    )
    assert summary["matched_dimension_count"] == len(packet["matched_dimensions"])

    covered_paths = {item["path"] for item in packet["covered_raw_features"]}
    residual_paths = {item["path"] for item in packet["residual_raw_features"]}
    assert covered_paths.isdisjoint(residual_paths)

    for match in packet["matched_dimensions"]:
        assert set(match["proof_paths"]).issubset(covered_paths)


def test_p03_review_packet_does_not_materialize_semantic_decision() -> None:
    packet = _run_packet("ADR-0011")

    assert "new_dimensions" not in packet
    assert "possible_existing_correspondence" not in packet
    assert "semantic_dimension_decision" not in packet
    assert packet["summary"]["semantic_review_required"] is bool(
        packet["residual_raw_features"]
    )
