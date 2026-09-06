from __future__ import annotations

import subprocess
import sys
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]
REVIEW_PACKET = ROOT / ".github" / "scripts" / "p03_authority_role_review_packet.py"


def _run_packet(adr_id: str, *, compact: bool = False) -> dict[str, object]:
    command = [
        sys.executable,
        str(REVIEW_PACKET),
        "--repo-root",
        str(ROOT),
        "--adr",
        adr_id,
    ]
    if compact:
        command.append("--compact")
    result = subprocess.run(
        command,
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


def test_p03_adr_0011_review_packet_is_machine_prepared() -> None:
    packet = _run_packet("ADR-0011")

    assert packet["target"]["adr_id"] == "ADR-0011"
    assert packet["inputs"]["dimension_count"] >= 79
    assert packet["inputs"]["runtime_authority"] == "NONE"
    assert packet["inputs"]["vocabulary_registration"] is False

    automation = packet["automation"]
    assert automation["existing_dimension_evaluation"] == "DETERMINISTIC"
    assert automation["matched_predicate_proof_trace"] == "DETERMINISTIC"
    assert automation["residual_raw_feature_extraction"] == "DETERMINISTIC"
    assert automation["machine_residual_candidate_generation"] == "DETERMINISTIC"
    assert (
        automation["existing_dimension_candidate_routing"]
        == "DETERMINISTIC_STRUCTURAL_NON_AUTHORITATIVE"
    )
    assert automation["manual_full_dimension_scan_required"] is False
    assert automation["manual_raw_corpus_search_required"] is False
    assert automation["raw_feature_force_fit"] == "FORBIDDEN"
    assert automation["natural_language_consumption"] == "FORBIDDEN"
    assert automation["semantic_effect_promotion"] == "DESIGN_REVIEW_ONLY"

    summary = packet["summary"]
    assert summary["raw_feature_count"] == 8
    assert (
        summary["covered_raw_feature_count"]
        + summary["residual_raw_feature_count"]
        == 8
    )
    assert summary["matched_dimension_count"] == len(packet["matched_dimensions"])
    assert summary["machine_residual_candidate_count"] == len(
        packet["machine_residual_candidates"]
    )
    assert summary["candidate_existing_dimension_routing_count"] == len(
        packet["candidate_existing_dimension_routing"]
    )
    assert summary["machine_residual_candidate_count"] == summary["residual_raw_feature_count"]

    covered_paths = {item["path"] for item in packet["covered_raw_features"]}
    residual_paths = {item["path"] for item in packet["residual_raw_features"]}
    assert covered_paths.isdisjoint(residual_paths)

    for match in packet["matched_dimensions"]:
        assert set(match["proof_paths"]).issubset(covered_paths)

    candidate_ids = set()
    for candidate in packet["machine_residual_candidates"]:
        assert candidate["candidate_id"].startswith("raw-candidate:")
        assert candidate["status"] == "UNINTERPRETED_RAW_BACKED_CANDIDATE"
        assert candidate["path"] in residual_paths
        assert candidate["semantic_effect_dimension"] == "NOT_DECIDED"
        assert candidate["candidate_id"] not in candidate_ids
        candidate_ids.add(candidate["candidate_id"])


def test_p03_adr_0011_routing_preselects_obvious_existing_dimension_candidates() -> None:
    packet = _run_packet("ADR-0011")

    routed = {
        item["raw_path"]: {
            candidate["dimension_id"] for candidate in item["candidate_dimensions"]
        }
        for item in packet["candidate_existing_dimension_routing"]
    }

    assert "activate_normative_family" in routed[
        "authority_semantics.activated_family"
    ]
    assert "bind_immutable_normative_snapshot" in routed[
        "authority_semantics.immutable_normative_snapshot"
    ]
    assert "define_classification_vocabulary" in routed[
        "authority_semantics.preserved_classifications"
    ]

    for item in packet["candidate_existing_dimension_routing"]:
        assert item["routing_only"] is True
        assert item["semantic_authority"] is False


def test_p03_compact_packet_is_the_narrow_agent_input_surface() -> None:
    compact = _run_packet("ADR-0011", compact=True)

    assert compact["schema_version"] == "ptsip-p03-authority-role-review-packet-compact/v1"
    assert compact["target"]["adr_id"] == "ADR-0011"
    assert compact["automation"] == {
        "existing_dimension_evaluation": "DETERMINISTIC",
        "existing_dimension_candidate_routing": "DETERMINISTIC_STRUCTURAL_NON_AUTHORITATIVE",
        "manual_full_dimension_scan_required": False,
        "manual_raw_corpus_search_required": False,
        "raw_feature_force_fit": "FORBIDDEN",
    }
    assert isinstance(compact["matched_dimension_ids"], list)
    assert isinstance(compact["routed_residual_candidates"], list)
    assert isinstance(compact["unrouted_residual_candidates"], list)
    assert compact["summary"]["raw_feature_count"] == 8
    assert (
        compact["summary"]["routed_residual_candidate_count"]
        + compact["summary"]["unrouted_residual_candidate_count"]
        == compact["summary"]["residual_raw_feature_count"]
    )
    assert "inputs" not in compact
    assert "covered_raw_features" not in compact
    assert "residual_raw_features" not in compact
    assert "matched_dimensions" not in compact
    assert "candidate_existing_dimension_routing" not in compact
    assert "machine_residual_candidates" not in compact


def test_p03_review_packet_does_not_materialize_semantic_decision() -> None:
    packet = _run_packet("ADR-0011")

    assert "new_dimensions" not in packet
    assert "possible_existing_correspondence" not in packet
    assert "semantic_dimension_decision" not in packet
    assert packet["summary"]["semantic_review_required"] is bool(
        packet["residual_raw_features"]
    )
