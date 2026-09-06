from __future__ import annotations

import argparse
import copy
import hashlib
import importlib.util
import json
import re
import sys
from pathlib import Path
from typing import Any

import yaml


PACKET_SCHEMA_VERSION = "ptsip-p03-authority-role-ai-decision-packet/v1"
RESPONSE_SCHEMA_VERSION = "ptsip-p03-authority-role-ai-decision-response/v1"
LEDGER_SCHEMA_VERSION = "ptsip-p03-authority-role-semantic-decision-ledger/v1"

DEFAULT_REGISTRY = "planning/0.4.0/WU-02/p03-authority-role-provisional-dimensions.yaml"
DEFAULT_MATRIX = "planning/0.4.0/WU-02/p03-authority-role-matrix.generated.yaml"
DEFAULT_RAW_SNAPSHOT = "planning/0.4.0/WU-02/p03-authority-role-raw-features.generated.yaml"
DEFAULT_LEDGER = "planning/0.4.0/WU-02/p03-authority-role-semantic-decisions.yaml"
DEFAULT_PACKET_DIR = "planning/0.4.0/WU-02/p03-authority-role-decision-packets"

_DECISION_VALUES = {
    "EXISTING",
    "REFINE_EXISTING",
    "NEW_DIMENSION",
    "NON_EFFECT",
    "DEFER",
}
_REASON_VALUES = {
    "SAME_EFFECT_ALREADY_REPRESENTED",
    "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
    "EXISTING_EFFECT_BOUNDARY_NEEDS_REFINEMENT",
    "DISTINCT_REUSABLE_AUTHORITY_EFFECT",
    "PARAMETER_REFERENCE_OR_NON_EFFECT_DETAIL",
    "INSUFFICIENT_SEMANTIC_CONTEXT",
}
_DIMENSION_ID = re.compile(r"^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$")
_ADR_ID = re.compile(r"^ADR-([0-9]{4})$")
_GIT_SHA1 = re.compile(r"^[0-9a-f]{40}$")
_VERSION_LIKE = re.compile(r"^\d+\.\d+\.\d+(?:-[a-z0-9.-]+)?$", re.IGNORECASE)
_RULE_ID = re.compile(r"^([A-Z][A-Z0-9]*-[A-Z][A-Z0-9]*-)(\d+)$")


class SemanticDecisionError(ValueError):
    pass


def _load_module(path: Path, name: str):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise SemanticDecisionError(f"cannot load module: {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _load_yaml(path: Path, *, label: str) -> dict[str, Any]:
    try:
        value = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except FileNotFoundError as exc:
        raise SemanticDecisionError(f"{label} not found: {path}") from exc
    except yaml.YAMLError as exc:
        raise SemanticDecisionError(f"{label} is not valid YAML: {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise SemanticDecisionError(f"{label} root must be a mapping: {path}")
    return value


def _dump_yaml(value: dict[str, Any]) -> str:
    return yaml.safe_dump(
        value,
        sort_keys=False,
        allow_unicode=True,
        default_flow_style=False,
    )


def _canonical_json(value: object) -> str:
    return json.dumps(
        value,
        ensure_ascii=False,
        sort_keys=True,
        separators=(",", ":"),
    )


def _sha256_payload(value: object) -> str:
    return hashlib.sha256(_canonical_json(value).encode("utf-8")).hexdigest()


def _sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _adr_number(value: object) -> int:
    if not isinstance(value, str):
        raise SemanticDecisionError("ADR id must be a string")
    match = _ADR_ID.fullmatch(value)
    if match is None:
        raise SemanticDecisionError(f"ADR id must match ADR-NNNN: {value!r}")
    return int(match.group(1))


def _resolve(repo_root: Path, value: str) -> Path:
    path = Path(value)
    return path if path.is_absolute() else repo_root / path


def _empty_ledger() -> dict[str, Any]:
    return {
        "schema_version": LEDGER_SCHEMA_VERSION,
        "runtime_authority": "NONE",
        "vocabulary_registration": False,
        "semantic_authority": "NONE",
        "review_source": "AI_DESIGN_TIME_ADVISORY",
        "records": {},
    }


def _load_ledger(path: Path) -> dict[str, Any]:
    if not path.exists():
        return _empty_ledger()
    ledger = _load_yaml(path, label="P03 semantic decision ledger")
    validate_ledger(ledger)
    return ledger


def validate_ledger(ledger: dict[str, Any]) -> None:
    if ledger.get("schema_version") != LEDGER_SCHEMA_VERSION:
        raise SemanticDecisionError("semantic decision ledger schema_version is invalid")
    if ledger.get("runtime_authority") != "NONE":
        raise SemanticDecisionError("semantic decision ledger must declare runtime_authority: NONE")
    if ledger.get("vocabulary_registration") is not False:
        raise SemanticDecisionError("semantic decision ledger must not register vocabulary")
    if ledger.get("semantic_authority") != "NONE":
        raise SemanticDecisionError("AI semantic decision ledger must not claim authority")
    if ledger.get("review_source") != "AI_DESIGN_TIME_ADVISORY":
        raise SemanticDecisionError("semantic decision ledger review_source is invalid")
    records = ledger.get("records")
    if not isinstance(records, dict):
        raise SemanticDecisionError("semantic decision ledger records must be a mapping")
    for candidate_id, record in records.items():
        if not isinstance(candidate_id, str) or not candidate_id.startswith("raw-candidate:"):
            raise SemanticDecisionError("semantic decision ledger contains an invalid candidate id")
        if not isinstance(record, dict):
            raise SemanticDecisionError("semantic decision ledger record must be a mapping")
        if record.get("decision") not in _DECISION_VALUES:
            raise SemanticDecisionError(f"ledger record {candidate_id} has an invalid decision")
        if record.get("reason_code") not in _REASON_VALUES:
            raise SemanticDecisionError(f"ledger record {candidate_id} has an invalid reason_code")
        if record.get("semantic_authority") != "NONE":
            raise SemanticDecisionError(f"ledger record {candidate_id} must remain non-authoritative")


def _string_shape(value: str) -> dict[str, object]:
    if _GIT_SHA1.fullmatch(value):
        return {"kind": "GIT_SHA1_REVISION"}
    if _VERSION_LIKE.fullmatch(value):
        return {"kind": "VERSION_LIKE_IDENTIFIER", "sample": value[:48]}
    if "/" in value or "\\" in value:
        suffix = Path(value).suffix or None
        return {"kind": "REPOSITORY_PATH", "suffix": suffix}
    return {"kind": "STRING", "sample": value[:64], "truncated": len(value) > 64}


def _array_shape(value: list[object]) -> dict[str, object]:
    item_types = sorted({type(item).__name__.upper() for item in value})
    result: dict[str, object] = {
        "kind": "ARRAY",
        "count": len(value),
        "item_types": item_types,
    }
    if value and all(isinstance(item, str) for item in value):
        strings = [str(item) for item in value]
        matches = [_RULE_ID.fullmatch(item) for item in strings]
        if all(match is not None for match in matches):
            prefixes = {match.group(1) for match in matches if match is not None}
            widths = {len(match.group(2)) for match in matches if match is not None}
            if len(prefixes) == 1 and len(widths) == 1:
                result["pattern"] = f"{next(iter(prefixes))}{'#' * next(iter(widths))}"
                return result
        if len(strings) <= 6:
            result["members"] = strings
        else:
            result["sample"] = strings[:3]
    return result


def _value_shape(value_type: str, value: object) -> dict[str, object]:
    if value_type == "BOOLEAN":
        return {"kind": "BOOLEAN", "value": value}
    if value_type in {"INTEGER", "NUMBER"}:
        return {"kind": value_type, "value": value}
    if value_type == "NULL":
        return {"kind": "NULL"}
    if value_type == "STRING" and isinstance(value, str):
        return _string_shape(value)
    if value_type == "ARRAY" and isinstance(value, list):
        return _array_shape(value)
    if value_type == "OBJECT" and isinstance(value, dict):
        return {"kind": "OBJECT", "keys": sorted(value)}
    return {"kind": value_type}


def _predicate_modes(value_type: str) -> list[str]:
    if value_type == "BOOLEAN":
        return ["EQUALS_EXACT"]
    if value_type in {"INTEGER", "NUMBER"}:
        return ["EQUALS_EXACT", "PRESENT"]
    if value_type == "STRING":
        return ["NON_EMPTY", "EQUALS_EXACT", "PRESENT"]
    if value_type == "ARRAY":
        return ["NON_EMPTY", "CONTAINS_ALL_EXACT", "EQUALS_EXACT", "PRESENT"]
    if value_type == "OBJECT":
        return ["NON_EMPTY", "EQUALS_EXACT", "PRESENT"]
    if value_type == "NULL":
        return ["PRESENT"]
    raise SemanticDecisionError(f"unsupported raw feature value_type: {value_type!r}")


def _routing_by_candidate(review_packet: dict[str, Any]) -> dict[str, list[dict[str, Any]]]:
    result: dict[str, list[dict[str, Any]]] = {}
    for route in review_packet["candidate_existing_dimension_routing"]:
        candidates = route.get("candidate_dimensions")
        if isinstance(candidates, list):
            result[str(route["candidate_id"])] = candidates
    return result


def _compact_routing(candidates: list[dict[str, Any]]) -> tuple[list[str], bool]:
    if not candidates:
        return [], False
    selected = [str(item["dimension_id"]) for item in candidates[:2]]
    if len(candidates) < 2:
        return selected, False
    first = float(candidates[0]["routing_score"])
    second = float(candidates[1]["routing_score"])
    return selected, (first - second) < 0.08


def build_decision_packet(
    repo_root: Path,
    adr_id: str,
    registry_path: Path,
    raw_snapshot_path: Path,
    ledger_path: Path,
    *,
    include_deferred: bool = False,
) -> dict[str, Any]:
    review_module = _load_module(
        repo_root / ".github/scripts/p03_authority_role_review_packet.py",
        "p03_authority_role_review_packet_for_decision",
    )
    review_packet = review_module.build_review_packet(
        repo_root,
        adr_id,
        registry_path,
        raw_snapshot_path,
    )
    ledger = _load_ledger(ledger_path)
    ledger_records = ledger["records"]
    routing = _routing_by_candidate(review_packet)

    questions: list[dict[str, object]] = []
    excluded_resolved = 0
    excluded_deferred = 0

    for candidate in review_packet["machine_residual_candidates"]:
        candidate_id = str(candidate["candidate_id"])
        prior = ledger_records.get(candidate_id)
        if isinstance(prior, dict):
            if prior.get("decision") == "DEFER" and include_deferred:
                pass
            elif prior.get("decision") == "DEFER":
                excluded_deferred += 1
                continue
            else:
                excluded_resolved += 1
                continue

        top_candidates, ambiguous = _compact_routing(routing.get(candidate_id, []))
        value_type = str(candidate["value_type"])
        questions.append(
            {
                "candidate_id": candidate_id,
                "path": candidate["path"],
                "value_shape": _value_shape(value_type, candidate["value"]),
                "occurrence_count": len(candidate["occurs_in"]),
                "existing_candidates": top_candidates,
                "routing_ambiguous": ambiguous,
                "allowed_predicate_modes": _predicate_modes(value_type),
            }
        )

    packet: dict[str, Any] = {
        "schema_version": PACKET_SCHEMA_VERSION,
        "target": {
            "adr_id": adr_id,
            "authority_type": review_packet["target"]["authority_type"],
        },
        "review_contract": {
            "role": "AI_DESIGN_TIME_SEMANTIC_REVIEW_ONLY",
            "semantic_authority": "NONE",
            "runtime_authority": "NONE",
            "allowed_decisions": sorted(_DECISION_VALUES),
            "allowed_reason_codes": sorted(_REASON_VALUES),
            "decision_rules": {
                "EXISTING": "attach detail to an already deterministic-matched dimension; no predicate mutation",
                "REFINE_EXISTING": "map to one routed candidate and add one generated predicate alternative",
                "NEW_DIMENSION": "create one provisional dimension from this raw candidate",
                "NON_EFFECT": "record as parameter/reference/non-effect detail",
                "DEFER": "record unresolved without semantic mutation",
            },
            "free_text_rationale_required": False,
            "include_deferred_candidates": include_deferred,
        },
        "input_fingerprints": {
            "dimension_registry_sha256": _sha256_file(registry_path),
            "raw_snapshot_sha256": _sha256_file(raw_snapshot_path),
            "ledger_sha256": _sha256_payload(ledger),
        },
        "already_matched_dimensions": [
            item["dimension_id"] for item in review_packet["matched_dimensions"]
        ],
        "questions": questions,
        "summary": {
            "question_count": len(questions),
            "already_resolved_count": excluded_resolved,
            "deferred_suppressed_count": excluded_deferred,
            "full_dimension_scan_required": False,
            "full_raw_corpus_read_required": False,
        },
    }
    packet["packet_fingerprint"] = _sha256_payload(packet)
    validate_decision_packet(packet)
    return packet


def validate_decision_packet(packet: dict[str, Any]) -> None:
    if packet.get("schema_version") != PACKET_SCHEMA_VERSION:
        raise SemanticDecisionError("AI decision packet schema_version is invalid")
    target = packet.get("target")
    contract = packet.get("review_contract")
    fingerprints = packet.get("input_fingerprints")
    questions = packet.get("questions")
    matched_dimensions = packet.get("already_matched_dimensions")
    summary = packet.get("summary")
    if not all(isinstance(value, dict) for value in (target, contract, fingerprints, summary)):
        raise SemanticDecisionError("AI decision packet mappings are malformed")
    if not isinstance(questions, list):
        raise SemanticDecisionError("AI decision packet questions must be a list")
    if not isinstance(matched_dimensions, list) or not all(
        isinstance(item, str) for item in matched_dimensions
    ):
        raise SemanticDecisionError("AI decision packet already_matched_dimensions must be a string list")
    _adr_number(target.get("adr_id"))
    if contract.get("semantic_authority") != "NONE" or contract.get("runtime_authority") != "NONE":
        raise SemanticDecisionError("AI decision packet must remain non-authoritative")
    if contract.get("free_text_rationale_required") is not False:
        raise SemanticDecisionError("AI decision packet must not require free-text rationale")
    if type(contract.get("include_deferred_candidates")) is not bool:
        raise SemanticDecisionError("AI decision packet include_deferred_candidates must be boolean")
    ids: set[str] = set()
    for question in questions:
        if not isinstance(question, dict):
            raise SemanticDecisionError("AI decision packet question must be a mapping")
        candidate_id = question.get("candidate_id")
        if not isinstance(candidate_id, str) or not candidate_id.startswith("raw-candidate:"):
            raise SemanticDecisionError("AI decision packet question has invalid candidate_id")
        if candidate_id in ids:
            raise SemanticDecisionError("AI decision packet candidate ids must be unique")
        ids.add(candidate_id)
        if not isinstance(question.get("existing_candidates"), list):
            raise SemanticDecisionError("AI decision packet existing_candidates must be a list")
        modes = question.get("allowed_predicate_modes")
        if not isinstance(modes, list) or not modes:
            raise SemanticDecisionError("AI decision packet allowed_predicate_modes must be non-empty")
    if summary.get("question_count") != len(questions):
        raise SemanticDecisionError("AI decision packet question_count is stale")

    claimed = packet.get("packet_fingerprint")
    if not isinstance(claimed, str):
        raise SemanticDecisionError("AI decision packet requires packet_fingerprint")
    without = copy.deepcopy(packet)
    without.pop("packet_fingerprint", None)
    expected = _sha256_payload(without)
    if claimed != expected:
        raise SemanticDecisionError("AI decision packet fingerprint is stale")


def build_response_template(packet: dict[str, Any]) -> dict[str, Any]:
    validate_decision_packet(packet)
    return {
        "schema_version": RESPONSE_SCHEMA_VERSION,
        "target": {
            "adr_id": packet["target"]["adr_id"],
            "packet_fingerprint": packet["packet_fingerprint"],
        },
        "reviewer": "AI_DESIGN_TIME_ADVISORY",
        "decisions": [
            {
                "candidate_id": question["candidate_id"],
                "decision": "REQUIRED",
                "target_dimension": None,
                "proposed_dimension_id": None,
                "predicate_mode": None,
                "reason_code": "REQUIRED",
            }
            for question in packet["questions"]
        ],
    }


def _question_map(packet: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {str(item["candidate_id"]): item for item in packet["questions"]}


def validate_response(packet: dict[str, Any], response: dict[str, Any]) -> None:
    validate_decision_packet(packet)
    if response.get("schema_version") != RESPONSE_SCHEMA_VERSION:
        raise SemanticDecisionError("AI decision response schema_version is invalid")
    target = response.get("target")
    if not isinstance(target, dict):
        raise SemanticDecisionError("AI decision response target must be a mapping")
    if target.get("adr_id") != packet["target"]["adr_id"]:
        raise SemanticDecisionError("AI decision response ADR does not match packet")
    if target.get("packet_fingerprint") != packet["packet_fingerprint"]:
        raise SemanticDecisionError("AI decision response references a stale packet")
    if response.get("reviewer") != "AI_DESIGN_TIME_ADVISORY":
        raise SemanticDecisionError("AI decision response reviewer must be AI_DESIGN_TIME_ADVISORY")
    decisions = response.get("decisions")
    if not isinstance(decisions, list):
        raise SemanticDecisionError("AI decision response decisions must be a list")

    questions = _question_map(packet)
    matched_dimensions = set(packet["already_matched_dimensions"])
    seen: set[str] = set()
    for item in decisions:
        if not isinstance(item, dict):
            raise SemanticDecisionError("AI decision response item must be a mapping")
        required_keys = {
            "candidate_id",
            "decision",
            "target_dimension",
            "proposed_dimension_id",
            "predicate_mode",
            "reason_code",
        }
        if set(item) != required_keys:
            raise SemanticDecisionError(
                f"AI decision response item keys must be exactly {sorted(required_keys)}"
            )
        candidate_id = item["candidate_id"]
        if candidate_id not in questions:
            raise SemanticDecisionError(f"AI decision response contains unknown candidate {candidate_id!r}")
        if candidate_id in seen:
            raise SemanticDecisionError("AI decision response contains duplicate candidate ids")
        seen.add(candidate_id)

        decision = item["decision"]
        reason_code = item["reason_code"]
        if decision not in _DECISION_VALUES:
            raise SemanticDecisionError(f"invalid AI semantic decision: {decision!r}")
        if reason_code not in _REASON_VALUES:
            raise SemanticDecisionError(f"invalid AI semantic reason_code: {reason_code!r}")

        question = questions[candidate_id]
        existing = set(question["existing_candidates"])
        target_dimension = item["target_dimension"]
        proposed = item["proposed_dimension_id"]
        mode = item["predicate_mode"]

        if decision == "EXISTING":
            if target_dimension not in matched_dimensions:
                raise SemanticDecisionError(
                    "EXISTING requires a dimension already matched deterministically in this ADR"
                )
            if proposed is not None or mode is not None:
                raise SemanticDecisionError("EXISTING must not define proposed_dimension_id or predicate_mode")
            if reason_code not in {
                "SAME_EFFECT_ALREADY_REPRESENTED",
                "PARAMETER_REFERENCE_OR_NON_EFFECT_DETAIL",
            }:
                raise SemanticDecisionError("EXISTING reason_code is incompatible")
        elif decision == "REFINE_EXISTING":
            if target_dimension not in existing:
                raise SemanticDecisionError("REFINE_EXISTING requires one routed existing candidate")
            if proposed is not None:
                raise SemanticDecisionError("REFINE_EXISTING must not define proposed_dimension_id")
            if mode not in question["allowed_predicate_modes"]:
                raise SemanticDecisionError("REFINE_EXISTING predicate_mode is not allowed for this value")
            if reason_code not in {
                "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
                "EXISTING_EFFECT_BOUNDARY_NEEDS_REFINEMENT",
            }:
                raise SemanticDecisionError("REFINE_EXISTING reason_code is incompatible")
        elif decision == "NEW_DIMENSION":
            if target_dimension is not None:
                raise SemanticDecisionError("NEW_DIMENSION must not define target_dimension")
            if not isinstance(proposed, str) or _DIMENSION_ID.fullmatch(proposed) is None:
                raise SemanticDecisionError("NEW_DIMENSION requires a valid proposed_dimension_id")
            if mode not in question["allowed_predicate_modes"]:
                raise SemanticDecisionError("NEW_DIMENSION predicate_mode is not allowed for this value")
            if reason_code != "DISTINCT_REUSABLE_AUTHORITY_EFFECT":
                raise SemanticDecisionError("NEW_DIMENSION requires DISTINCT_REUSABLE_AUTHORITY_EFFECT")
        elif decision == "NON_EFFECT":
            if target_dimension is not None or proposed is not None or mode is not None:
                raise SemanticDecisionError("NON_EFFECT must not define semantic mutation fields")
            if reason_code != "PARAMETER_REFERENCE_OR_NON_EFFECT_DETAIL":
                raise SemanticDecisionError("NON_EFFECT reason_code is incompatible")
        elif decision == "DEFER":
            if target_dimension is not None or proposed is not None or mode is not None:
                raise SemanticDecisionError("DEFER must not define semantic mutation fields")
            if reason_code != "INSUFFICIENT_SEMANTIC_CONTEXT":
                raise SemanticDecisionError("DEFER reason_code is incompatible")

    if seen != set(questions):
        missing = sorted(set(questions) - seen)
        raise SemanticDecisionError(f"AI decision response must decide every packet question; missing={missing}")


def _candidate_map(full_review_packet: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {
        str(item["candidate_id"]): item
        for item in full_review_packet["machine_residual_candidates"]
    }


def _condition_from_candidate(candidate: dict[str, Any], mode: str) -> dict[str, Any]:
    path = candidate["path"]
    value = candidate["value"]
    if mode == "PRESENT":
        return {"path": path, "operator": "PRESENT"}
    if mode == "NON_EMPTY":
        return {"path": path, "operator": "NON_EMPTY"}
    if mode == "EQUALS_EXACT":
        return {"path": path, "operator": "EQUALS", "value": copy.deepcopy(value)}
    if mode == "CONTAINS_ALL_EXACT":
        if not isinstance(value, list) or not value:
            raise SemanticDecisionError("CONTAINS_ALL_EXACT requires a non-empty array raw value")
        return {"path": path, "operator": "CONTAINS_ALL", "value": copy.deepcopy(value)}
    raise SemanticDecisionError(f"unsupported predicate mode: {mode!r}")


def _append_alternative(expression: dict[str, Any], condition: dict[str, Any]) -> dict[str, Any]:
    if expression == condition:
        return expression
    if set(expression) == {"any"} and isinstance(expression["any"], list):
        if condition not in expression["any"]:
            expression["any"].append(condition)
        return expression
    return {"any": [expression, condition]}


def _write_transactional(paths_to_payloads: list[tuple[Path, dict[str, Any]]]) -> None:
    originals: dict[Path, bytes | None] = {}
    try:
        for path, payload in paths_to_payloads:
            originals[path] = path.read_bytes() if path.exists() else None
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(_dump_yaml(payload), encoding="utf-8")
    except Exception:
        for path, original in originals.items():
            if original is None:
                path.unlink(missing_ok=True)
            else:
                path.write_bytes(original)
        raise


def apply_response(
    repo_root: Path,
    packet: dict[str, Any],
    response: dict[str, Any],
    registry_path: Path,
    matrix_path: Path,
    raw_snapshot_path: Path,
    ledger_path: Path,
) -> dict[str, Any]:
    validate_response(packet, response)

    current_packet = build_decision_packet(
        repo_root,
        str(packet["target"]["adr_id"]),
        registry_path,
        raw_snapshot_path,
        ledger_path,
        include_deferred=bool(packet["review_contract"]["include_deferred_candidates"]),
    )
    if current_packet["packet_fingerprint"] != packet["packet_fingerprint"]:
        raise SemanticDecisionError(
            "AI decision packet inputs changed; regenerate packet before applying response"
        )

    review_module = _load_module(
        repo_root / ".github/scripts/p03_authority_role_review_packet.py",
        "p03_authority_role_review_packet_for_apply",
    )
    matrix_module = _load_module(
        repo_root / ".github/scripts/p03_authority_role_matrix.py",
        "p03_authority_role_matrix_for_apply",
    )
    full_review = review_module.build_review_packet(
        repo_root,
        str(packet["target"]["adr_id"]),
        registry_path,
        raw_snapshot_path,
    )
    candidates = _candidate_map(full_review)

    registry = _load_yaml(registry_path, label="P03 provisional dimension registry")
    ledger = _load_ledger(ledger_path)
    dimensions = registry.get("dimensions")
    analysis = registry.get("analysis")
    if not isinstance(dimensions, dict) or not isinstance(analysis, dict):
        raise SemanticDecisionError("P03 provisional dimension registry is malformed")

    adr_id = str(packet["target"]["adr_id"])
    target_number = _adr_number(adr_id)
    current_reviewed = _adr_number(analysis.get("reviewed_through"))
    if target_number > current_reviewed + 1:
        raise SemanticDecisionError(
            "semantic review must advance sequentially; review the previous ADR first"
        )

    has_defer = any(item["decision"] == "DEFER" for item in response["decisions"])
    created_dimensions: list[str] = []
    refined_dimensions: list[str] = []

    for item in response["decisions"]:
        candidate_id = str(item["candidate_id"])
        candidate = candidates[candidate_id]
        decision = item["decision"]

        if decision == "REFINE_EXISTING":
            target_dimension = str(item["target_dimension"])
            definition = dimensions.get(target_dimension)
            if not isinstance(definition, dict):
                raise SemanticDecisionError(f"target dimension not found: {target_dimension}")
            condition = _condition_from_candidate(candidate, str(item["predicate_mode"]))
            definition["expression"] = _append_alternative(
                copy.deepcopy(definition["expression"]),
                condition,
            )
            refined_dimensions.append(target_dimension)
        elif decision == "NEW_DIMENSION":
            proposed = str(item["proposed_dimension_id"])
            if proposed in dimensions:
                raise SemanticDecisionError(f"proposed dimension already exists: {proposed}")
            condition = _condition_from_candidate(candidate, str(item["predicate_mode"]))
            dimensions[proposed] = {
                "status": "PROVISIONAL",
                "introduced_by": adr_id,
                "expression": condition,
            }
            created_dimensions.append(proposed)

        ledger["records"][candidate_id] = {
            "first_reviewed_in": adr_id,
            "raw_path": candidate["path"],
            "decision": decision,
            "target_dimension": item["target_dimension"],
            "proposed_dimension_id": item["proposed_dimension_id"],
            "predicate_mode": item["predicate_mode"],
            "reason_code": item["reason_code"],
            "packet_fingerprint": packet["packet_fingerprint"],
            "semantic_authority": "NONE",
            "runtime_authority": "NONE",
            "vocabulary_registration": False,
        }

    if created_dimensions and has_defer:
        raise SemanticDecisionError(
            "NEW_DIMENSION cannot be applied while the same ADR review contains DEFER; "
            "resolve or defer without creating a registry dimension"
        )

    if target_number == current_reviewed + 1 and not has_defer:
        analysis["reviewed_through"] = adr_id

    # Validate the candidate registry before replacing the canonical files.
    temp_registry = registry_path.with_name(registry_path.name + ".semantic-decision-tmp")
    temp_matrix = matrix_path.with_name(matrix_path.name + ".semantic-decision-tmp")
    try:
        temp_registry.write_text(_dump_yaml(registry), encoding="utf-8")
        expected_matrix = matrix_module.build_matrix(repo_root, temp_registry)
        expected_matrix["source"]["dimension_registry"] = (
            registry_path.relative_to(repo_root).as_posix()
            if registry_path.is_relative_to(repo_root)
            else str(registry_path)
        )
        matrix_module.validate_matrix(expected_matrix)
    finally:
        temp_registry.unlink(missing_ok=True)
        temp_matrix.unlink(missing_ok=True)

    validate_ledger(ledger)
    _write_transactional(
        [
            (registry_path, registry),
            (matrix_path, expected_matrix),
            (ledger_path, ledger),
        ]
    )

    return {
        "adr_id": adr_id,
        "reviewed_through": analysis["reviewed_through"],
        "created_dimensions": sorted(set(created_dimensions)),
        "refined_dimensions": sorted(set(refined_dimensions)),
        "deferred_count": sum(item["decision"] == "DEFER" for item in response["decisions"]),
        "ledger_record_count": len(ledger["records"]),
        "matrix_row_count": expected_matrix["validation"]["reviewed_row_count"],
        "matrix_dimension_count": expected_matrix["validation"]["dimension_count"],
        "runtime_authority": "NONE",
        "vocabulary_registration": False,
    }


def _write_prepare_outputs(packet: dict[str, Any], packet_path: Path, response_path: Path) -> None:
    packet_path.parent.mkdir(parents=True, exist_ok=True)
    response_path.parent.mkdir(parents=True, exist_ok=True)
    packet_path.write_text(_dump_yaml(packet), encoding="utf-8")
    response_path.write_text(_dump_yaml(build_response_template(packet)), encoding="utf-8")


def _prepare_paths(repo_root: Path, adr_id: str, output: str | None, response_output: str | None) -> tuple[Path, Path]:
    packet_path = (
        _resolve(repo_root, output)
        if output
        else repo_root / DEFAULT_PACKET_DIR / f"{adr_id}.decision.generated.yaml"
    )
    response_path = (
        _resolve(repo_root, response_output)
        if response_output
        else repo_root / DEFAULT_PACKET_DIR / f"{adr_id}.response.yaml"
    )
    return packet_path, response_path


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Prepare, validate, and deterministically apply minimal P03 AI semantic decisions"
    )
    parser.add_argument("--repo-root", default=".")
    sub = parser.add_subparsers(dest="command", required=True)

    prepare = sub.add_parser("prepare")
    prepare.add_argument("--adr", required=True)
    prepare.add_argument("--registry", default=DEFAULT_REGISTRY)
    prepare.add_argument("--raw-snapshot", default=DEFAULT_RAW_SNAPSHOT)
    prepare.add_argument("--ledger", default=DEFAULT_LEDGER)
    prepare.add_argument("--include-deferred", action="store_true")
    prepare.add_argument("--write", action="store_true")
    prepare.add_argument("--output")
    prepare.add_argument("--response-output")

    validate = sub.add_parser("validate")
    validate.add_argument("--packet", required=True)
    validate.add_argument("--response", required=True)

    apply = sub.add_parser("apply")
    apply.add_argument("--packet", required=True)
    apply.add_argument("--response", required=True)
    apply.add_argument("--registry", default=DEFAULT_REGISTRY)
    apply.add_argument("--matrix", default=DEFAULT_MATRIX)
    apply.add_argument("--raw-snapshot", default=DEFAULT_RAW_SNAPSHOT)
    apply.add_argument("--ledger", default=DEFAULT_LEDGER)

    args = parser.parse_args(argv)
    repo_root = Path(args.repo_root).resolve()

    try:
        if args.command == "prepare":
            registry_path = _resolve(repo_root, args.registry)
            raw_snapshot_path = _resolve(repo_root, args.raw_snapshot)
            ledger_path = _resolve(repo_root, args.ledger)
            packet = build_decision_packet(
                repo_root,
                args.adr,
                registry_path,
                raw_snapshot_path,
                ledger_path,
                include_deferred=args.include_deferred,
            )
            if args.write:
                packet_path, response_path = _prepare_paths(
                    repo_root,
                    args.adr,
                    args.output,
                    args.response_output,
                )
                _write_prepare_outputs(packet, packet_path, response_path)
                print(
                    f"packet={packet_path.relative_to(repo_root).as_posix()} "
                    f"response={response_path.relative_to(repo_root).as_posix()} "
                    f"questions={packet['summary']['question_count']}"
                )
            else:
                print(_dump_yaml(packet), end="")
            return 0

        packet = _load_yaml(_resolve(repo_root, args.packet), label="P03 AI decision packet")
        response = _load_yaml(_resolve(repo_root, args.response), label="P03 AI decision response")

        if args.command == "validate":
            validate_response(packet, response)
            print(
                f"P03 AI decision response validated: "
                f"{len(response['decisions'])} decision(s), authority=NONE"
            )
            return 0

        result = apply_response(
            repo_root,
            packet,
            response,
            _resolve(repo_root, args.registry),
            _resolve(repo_root, args.matrix),
            _resolve(repo_root, args.raw_snapshot),
            _resolve(repo_root, args.ledger),
        )
        print(_dump_yaml(result), end="")
        return 0
    except (OSError, SemanticDecisionError, ValueError, KeyError) as exc:
        print(f"P03 semantic-decision error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
