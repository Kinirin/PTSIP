from __future__ import annotations

import importlib.util
import subprocess
import sys
from pathlib import Path

import pytest
import yaml


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / ".github" / "scripts" / "p03_authority_role_semantic_decision.py"
REGISTRY = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-provisional-dimensions.yaml"
MATRIX = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-matrix.generated.yaml"
RAW = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-raw-features.generated.yaml"
LEDGER = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-semantic-decisions.yaml"


def _yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def _load_module():
    spec = importlib.util.spec_from_file_location("p03_authority_role_semantic_decision", SCRIPT)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _pre_adr_0011_inputs(module, tmp_path: Path) -> tuple[Path, Path, Path]:
    registry = _yaml(REGISTRY)
    registry["analysis"]["reviewed_through"] = "ADR-0010"

    for dimension_id in (
        "preserve_classification_vocabulary",
        "separate_specification_activation_from_profile_mutation",
        "separate_specification_activation_from_tool_implementation",
    ):
        del registry["dimensions"][dimension_id]

    refinements = {
        "activate_normative_family": {
            "path": "authority_semantics.activated_family",
            "operator": "NON_EMPTY",
        },
        "bind_immutable_normative_snapshot": {
            "path": "authority_semantics.immutable_normative_snapshot",
            "operator": "NON_EMPTY",
        },
        "define_classification_vocabulary": {
            "path": "authority_semantics.toolchain_is_current_ptsip_classification",
            "operator": "EQUALS",
            "value": False,
        },
    }
    for dimension_id, condition in refinements.items():
        expression = registry["dimensions"][dimension_id]["expression"]
        alternatives = expression["any"]
        assert condition in alternatives
        alternatives.remove(condition)
        registry["dimensions"][dimension_id]["expression"] = (
            alternatives[0] if len(alternatives) == 1 else {"any": alternatives}
        )

    registry_path = tmp_path / "dimensions.yaml"
    matrix_path = tmp_path / "matrix.yaml"
    ledger_path = tmp_path / "ledger.yaml"
    registry_path.write_text(
        yaml.safe_dump(registry, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )
    matrix_path.write_bytes(MATRIX.read_bytes())
    ledger_path.write_text(
        yaml.safe_dump(module._empty_ledger(), sort_keys=False),
        encoding="utf-8",
    )
    return registry_path, matrix_path, ledger_path


def _prepare_stdout(adr_id: str) -> dict[str, object]:
    result = subprocess.run(
        [
            sys.executable,
            str(SCRIPT),
            "--repo-root",
            str(ROOT),
            "prepare",
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


def test_p03_ai_decision_packet_is_small_and_non_authoritative() -> None:
    packet = _prepare_stdout("ADR-0011")

    assert packet["schema_version"] == "ptsip-p03-authority-role-ai-decision-packet/v1"
    assert packet["target"]["adr_id"] == "ADR-0011"
    assert packet["review_contract"]["semantic_authority"] == "NONE"
    assert packet["review_contract"]["runtime_authority"] == "NONE"
    assert packet["review_contract"]["free_text_rationale_required"] is False
    assert packet["summary"]["question_count"] == 0
    assert packet["summary"]["already_resolved_count"] == 2
    assert packet["summary"]["full_dimension_scan_required"] is False
    assert packet["summary"]["full_raw_corpus_read_required"] is False
    assert set(packet["already_matched_dimensions"]) == {
        "activate_normative_family",
        "bind_immutable_normative_snapshot",
        "define_classification_vocabulary",
        "preserve_classification_vocabulary",
        "separate_specification_activation_from_profile_mutation",
        "separate_specification_activation_from_tool_implementation",
    }

    # The packet must not expose the exact immutable SHA or the twelve migration rule values.
    serialized = yaml.safe_dump(packet, sort_keys=False)
    assert "b648d9e026f502b14481ba2d0606d9acc88a31fc" not in serialized
    assert "PTSIP-MIG-015" not in serialized


def test_p03_ai_response_contract_is_closed_and_stale_safe() -> None:
    module = _load_module()
    packet = module.build_decision_packet(ROOT, "ADR-0011", REGISTRY, RAW, LEDGER)
    response = module.build_response_template(packet)

    for item in response["decisions"]:
        item.update(
            {
                "decision": "DEFER",
                "target_dimension": None,
                "proposed_dimension_id": None,
                "predicate_mode": None,
                "reason_code": "INSUFFICIENT_SEMANTIC_CONTEXT",
            }
        )

    module.validate_response(packet, response)

    stale = yaml.safe_load(yaml.safe_dump(response))
    stale["target"]["packet_fingerprint"] = "0" * 64
    with pytest.raises(module.SemanticDecisionError, match="stale packet"):
        module.validate_response(packet, stale)


def test_p03_semantic_apply_generates_predicates_ledger_and_rectangular_backfill(
    tmp_path: Path,
) -> None:
    module = _load_module()
    registry_path, matrix_path, ledger_path = _pre_adr_0011_inputs(module, tmp_path)

    packet = module.build_decision_packet(
        ROOT,
        "ADR-0011",
        registry_path,
        RAW,
        ledger_path,
    )
    response = module.build_response_template(packet)

    # These are test-only mechanical resolutions used to verify deterministic application.
    # They do not record or approve ADR-0011 semantics in the repository.
    for item in response["decisions"]:
        question = next(
            question
            for question in packet["questions"]
            if question["candidate_id"] == item["candidate_id"]
        )
        path = question["path"]
        if path == "authority_semantics.activated_family":
            item.update(
                {
                    "decision": "REFINE_EXISTING",
                    "target_dimension": "activate_normative_family",
                    "proposed_dimension_id": None,
                    "predicate_mode": "NON_EMPTY",
                    "reason_code": "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
                }
            )
        elif path == "authority_semantics.tool_runtime_claims_full_implementation_by_activation":
            item.update(
                {
                    "decision": "NEW_DIMENSION",
                    "target_dimension": None,
                    "proposed_dimension_id": "test_only_separate_activation_runtime_claim",
                    "predicate_mode": "EQUALS_EXACT",
                    "reason_code": "DISTINCT_REUSABLE_AUTHORITY_EFFECT",
                }
            )
        else:
            item.update(
                {
                    "decision": "NON_EFFECT",
                    "target_dimension": None,
                    "proposed_dimension_id": None,
                    "predicate_mode": None,
                    "reason_code": "PARAMETER_REFERENCE_OR_NON_EFFECT_DETAIL",
                }
            )

    result = module.apply_response(
        ROOT,
        packet,
        response,
        registry_path,
        matrix_path,
        RAW,
        ledger_path,
    )

    assert result["reviewed_through"] == "ADR-0011"
    assert result["created_dimensions"] == ["test_only_separate_activation_runtime_claim"]
    assert result["refined_dimensions"] == ["activate_normative_family"]
    assert result["matrix_row_count"] == 11
    assert result["matrix_dimension_count"] == 80
    assert result["runtime_authority"] == "NONE"
    assert result["vocabulary_registration"] is False

    registry = _yaml(registry_path)
    assert registry["analysis"]["reviewed_through"] == "ADR-0011"
    expression = registry["dimensions"]["activate_normative_family"]["expression"]
    assert {
        "path": "authority_semantics.activated_family",
        "operator": "NON_EMPTY",
    } in expression["any"]

    matrix = _yaml(matrix_path)
    assert len(matrix["rows"]) == 11
    assert matrix["validation"]["dimension_count"] == 80
    assert all(
        len(row["role_effect_analysis"]) == 80
        for row in matrix["rows"].values()
    )

    ledger = _yaml(ledger_path)
    assert len(ledger["records"]) == 8
    assert all(
        record["semantic_authority"] == "NONE"
        for record in ledger["records"].values()
    )


def test_p03_new_dimension_is_not_applied_while_same_review_is_deferred(
    tmp_path: Path,
) -> None:
    module = _load_module()
    registry_path, matrix_path, ledger_path = _pre_adr_0011_inputs(module, tmp_path)

    packet = module.build_decision_packet(
        ROOT,
        "ADR-0011",
        registry_path,
        RAW,
        ledger_path,
    )
    response = module.build_response_template(packet)

    for index, item in enumerate(response["decisions"]):
        if index == 0:
            item.update(
                {
                    "decision": "NEW_DIMENSION",
                    "target_dimension": None,
                    "proposed_dimension_id": "test_only_new_dimension",
                    "predicate_mode": packet["questions"][0]["allowed_predicate_modes"][0],
                    "reason_code": "DISTINCT_REUSABLE_AUTHORITY_EFFECT",
                }
            )
        else:
            item.update(
                {
                    "decision": "DEFER",
                    "target_dimension": None,
                    "proposed_dimension_id": None,
                    "predicate_mode": None,
                    "reason_code": "INSUFFICIENT_SEMANTIC_CONTEXT",
                }
            )

    with pytest.raises(
        module.SemanticDecisionError,
        match="NEW_DIMENSION cannot be applied",
    ):
        module.apply_response(
            ROOT,
            packet,
            response,
            registry_path,
            matrix_path,
            RAW,
            ledger_path,
        )

    assert _yaml(registry_path)["analysis"]["reviewed_through"] == "ADR-0010"
    assert _yaml(ledger_path)["records"] == {}


def test_p03_decision_ledger_suppresses_repeat_ai_reasoning_and_allows_explicit_defer_revisit(
    tmp_path: Path,
) -> None:
    module = _load_module()
    registry_path, _, empty_ledger_path = _pre_adr_0011_inputs(module, tmp_path)
    base_packet = module.build_decision_packet(
        ROOT, "ADR-0011", registry_path, RAW, empty_ledger_path
    )
    first, second = base_packet["questions"][:2]

    ledger = module._empty_ledger()
    ledger["records"][first["candidate_id"]] = {
        "first_reviewed_in": "ADR-0011",
        "raw_path": first["path"],
        "decision": "NON_EFFECT",
        "target_dimension": None,
        "proposed_dimension_id": None,
        "predicate_mode": None,
        "reason_code": "PARAMETER_REFERENCE_OR_NON_EFFECT_DETAIL",
        "packet_fingerprint": "test-only",
        "semantic_authority": "NONE",
        "runtime_authority": "NONE",
        "vocabulary_registration": False,
    }
    ledger["records"][second["candidate_id"]] = {
        "first_reviewed_in": "ADR-0011",
        "raw_path": second["path"],
        "decision": "DEFER",
        "target_dimension": None,
        "proposed_dimension_id": None,
        "predicate_mode": None,
        "reason_code": "INSUFFICIENT_SEMANTIC_CONTEXT",
        "packet_fingerprint": "test-only",
        "semantic_authority": "NONE",
        "runtime_authority": "NONE",
        "vocabulary_registration": False,
    }
    ledger_path = tmp_path / "ledger.yaml"
    ledger_path.write_text(
        yaml.safe_dump(ledger, sort_keys=False),
        encoding="utf-8",
    )

    default_packet = module.build_decision_packet(
        ROOT,
        "ADR-0011",
        registry_path,
        RAW,
        ledger_path,
    )
    assert default_packet["summary"]["question_count"] == 6
    assert default_packet["summary"]["already_resolved_count"] == 1
    assert default_packet["summary"]["deferred_suppressed_count"] == 1
    assert first["candidate_id"] not in {
        item["candidate_id"] for item in default_packet["questions"]
    }
    assert second["candidate_id"] not in {
        item["candidate_id"] for item in default_packet["questions"]
    }

    revisit_packet = module.build_decision_packet(
        ROOT,
        "ADR-0011",
        registry_path,
        RAW,
        ledger_path,
        include_deferred=True,
    )
    assert revisit_packet["summary"]["question_count"] == 7
    assert revisit_packet["review_contract"]["include_deferred_candidates"] is True
    assert first["candidate_id"] not in {
        item["candidate_id"] for item in revisit_packet["questions"]
    }
    assert second["candidate_id"] in {
        item["candidate_id"] for item in revisit_packet["questions"]
    }
