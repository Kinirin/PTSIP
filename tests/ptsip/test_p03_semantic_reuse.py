from __future__ import annotations

import importlib.util
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]
REUSE_SCRIPT = ROOT / ".github" / "scripts" / "p03_authority_role_semantic_reuse.py"
DECISION_SCRIPT = ROOT / ".github" / "scripts" / "p03_authority_role_semantic_decision.py"
REGISTRY = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-provisional-dimensions.yaml"
RAW = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-raw-features.generated.yaml"
LEDGER = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-semantic-decisions.yaml"


def _yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def _ledger_through(adr_number: int) -> dict[str, object]:
    ledger = _yaml(LEDGER)
    records = ledger["records"]
    ledger["records"] = {
        candidate_id: record
        for candidate_id, record in records.items()
        if int(str(record["first_reviewed_in"]).split("-")[1]) <= adr_number
    }
    return ledger


def _load(path: Path, name: str):
    spec = importlib.util.spec_from_file_location(name, path)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_p03_authority_contract_identity_preserves_integer_schema_version() -> None:
    reuse = _load(REUSE_SCRIPT, "p03_semantic_reuse_contract_identity")
    index = _yaml(ROOT / "decisions" / "INDEX.yaml")
    routes = reuse._index_routes(index)

    contract = reuse._authority_contract(ROOT, "ADR-0011", routes)

    assert contract == {
        "authority_type": "SPECIFICATION_FAMILY_ACTIVATION",
        "schema_id": "ptsip.governance/specification-family-activation",
        "schema_version": 1,
    }
    assert type(contract["schema_version"]) is int


def test_p03_exact_context_reuse_bypasses_duplicate_ai_question() -> None:
    reuse = _load(REUSE_SCRIPT, "p03_semantic_reuse_exact")
    ledger = _ledger_through(11)
    raw = _yaml(RAW)
    registry = _yaml(REGISTRY)

    candidate = {
        "candidate_id": "raw-candidate:test-version-shape",
        "path": "authority_semantics.activated_family",
        "value_type": "STRING",
        "value": "0.3.8-draft",
        "occurs_in": ["ADR-0011"],
    }
    result = reuse.analyze_reuse(
        ROOT,
        "ADR-0011",
        [candidate],
        ledger,
        raw,
        registry["dimensions"],
        [],
        {
            candidate["candidate_id"]: [
                "NON_EMPTY",
                "EQUALS_EXACT",
                "PRESENT",
            ]
        },
    )

    assert len(result["automatic_reuse_actions"]) == 1
    action = result["automatic_reuse_actions"][0]
    assert action["candidate_id"] == candidate["candidate_id"]
    assert action["decision"] == "REFINE_EXISTING"
    assert action["target_dimension"] == "activate_normative_family"
    assert action["reuse_mode"] == "DETERMINISTIC_EXACT_CONTEXT_REUSE"
    assert action["semantic_authority"] is False
    assert result["structural_reuse_candidates"] == {}


def test_p03_structural_reuse_remains_ai_confirmation_only() -> None:
    reuse = _load(REUSE_SCRIPT, "p03_semantic_reuse_structural")
    ledger = _ledger_through(11)
    raw = _yaml(RAW)
    registry = _yaml(REGISTRY)

    candidate = {
        "candidate_id": "raw-candidate:test-structural",
        "path": "authority_semantics.activated_spec_family",
        "value_type": "STRING",
        "value": "0.3.8-draft",
        "occurs_in": ["ADR-0011"],
    }
    result = reuse.analyze_reuse(
        ROOT,
        "ADR-0011",
        [candidate],
        ledger,
        raw,
        registry["dimensions"],
        [],
        {
            candidate["candidate_id"]: [
                "NON_EMPTY",
                "EQUALS_EXACT",
                "PRESENT",
            ]
        },
    )

    assert result["automatic_reuse_actions"] == []
    routed = result["structural_reuse_candidates"][candidate["candidate_id"]]
    assert routed["confirmation_required"] is True
    assert routed["semantic_authority"] is False
    assert routed["resolved_dimension"] == "activate_normative_family"
    assert routed["concrete_reuse_action"]["decision"] == "REFINE_EXISTING"


def test_p03_reuse_prior_response_normalizes_to_concrete_non_authoritative_action() -> None:
    decision = _load(DECISION_SCRIPT, "p03_semantic_decision_normalize")

    question = {
        "candidate_id": "raw-candidate:test-confirm",
        "prior_resolution_candidate": {
            "prior_candidate_id": "raw-candidate:prior",
            "concrete_reuse_action": {
                "decision": "REFINE_EXISTING",
                "target_dimension": "activate_normative_family",
                "proposed_dimension_id": None,
                "predicate_mode": "NON_EMPTY",
                "reason_code": "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
            },
        },
    }
    response_item = {
        "candidate_id": question["candidate_id"],
        "decision": "REUSE_PRIOR",
        "target_dimension": None,
        "proposed_dimension_id": None,
        "predicate_mode": None,
        "reason_code": "CONFIRMED_PRIOR_SEMANTIC_RESOLUTION",
    }

    normalized = decision._normalize_response_item(question, response_item)
    assert normalized == {
        "candidate_id": question["candidate_id"],
        "decision": "REFINE_EXISTING",
        "target_dimension": "activate_normative_family",
        "proposed_dimension_id": None,
        "predicate_mode": "NON_EMPTY",
        "reason_code": "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
        "resolution_source": "AI_CONFIRMED_STRUCTURAL_REUSE",
        "reused_from_candidate_id": "raw-candidate:prior",
    }


def test_p03_prefix_fixture_reconstruction_ignores_later_review_growth() -> None:
    decision = _load(DECISION_SCRIPT, "p03_semantic_decision_prefix")
    registry = decision.build_registry_for_review_prefix(
        ROOT,
        REGISTRY,
        RAW,
        LEDGER,
        "ADR-0010",
    )

    assert registry["analysis"]["reviewed_through"] == "ADR-0010"
    for definition in registry["dimensions"].values():
        introduced = int(str(definition["introduced_by"]).split("-")[1])
        assert introduced <= 10


def test_p03_exact_reuse_fails_closed_when_any_exact_precedent_is_not_reproducible() -> None:
    reuse = _load(REUSE_SCRIPT, "p03_semantic_reuse_fail_closed")
    ledger = _ledger_through(11)
    raw = _yaml(RAW)
    registry = _yaml(REGISTRY)

    synthetic_prior = {
        "path": "authority_semantics.activated_family",
        "value_type": "STRING",
        "value": "0.3.9-draft",
        "occurs_in": ["ADR-0011"],
    }
    synthetic_prior_id = reuse._raw_candidate_id(synthetic_prior)
    raw["features"].append(synthetic_prior)
    ledger["records"][synthetic_prior_id] = {
        "first_reviewed_in": "ADR-0011",
        "raw_path": synthetic_prior["path"],
        "decision": "REFINE_EXISTING",
        "target_dimension": "dimension_that_does_not_exist",
        "proposed_dimension_id": None,
        "predicate_mode": "NON_EMPTY",
        "reason_code": "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
        "packet_fingerprint": "test-only",
        "semantic_authority": "NONE",
        "runtime_authority": "NONE",
        "vocabulary_registration": False,
    }

    candidate = {
        "candidate_id": "raw-candidate:test-fail-closed",
        "path": "authority_semantics.activated_family",
        "value_type": "STRING",
        "value": "0.3.8-draft",
        "occurs_in": ["ADR-0011"],
    }
    result = reuse.analyze_reuse(
        ROOT,
        "ADR-0011",
        [candidate],
        ledger,
        raw,
        registry["dimensions"],
        [],
        {
            candidate["candidate_id"]: [
                "NON_EMPTY",
                "EQUALS_EXACT",
                "PRESENT",
            ]
        },
    )

    assert result["automatic_reuse_actions"] == []
    assert candidate["candidate_id"] in result["structural_reuse_candidates"]
