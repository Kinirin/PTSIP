from __future__ import annotations

import hashlib
import json
import re
from difflib import SequenceMatcher
from pathlib import Path
from typing import Any

import yaml


_GIT_SHA1 = re.compile(r"^[0-9a-f]{40}$")
_VERSION_LIKE = re.compile(r"^\d+\.\d+\.\d+(?:-[a-z0-9.-]+)?$", re.IGNORECASE)
_RULE_ID = re.compile(r"^([A-Z][A-Z0-9]*-[A-Z][A-Z0-9]*-)(\d+)$")


class ReuseAnalysisError(ValueError):
    pass


def _load_yaml(path: Path, *, label: str) -> dict[str, Any]:
    try:
        value = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except FileNotFoundError as exc:
        raise ReuseAnalysisError(f"{label} not found: {path}") from exc
    except yaml.YAMLError as exc:
        raise ReuseAnalysisError(f"{label} is not valid YAML: {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise ReuseAnalysisError(f"{label} root must be a mapping: {path}")
    return value


def _canonical_json(value: object) -> str:
    return json.dumps(
        value,
        ensure_ascii=False,
        sort_keys=True,
        separators=(",", ":"),
    )


def _raw_candidate_id(feature: dict[str, object]) -> str:
    digest = hashlib.sha256(
        _canonical_json(
            {
                "path": feature["path"],
                "value_type": feature["value_type"],
                "value": feature["value"],
            }
        ).encode("utf-8")
    ).hexdigest()[:20]
    return f"raw-candidate:{digest}"


def _candidate_index(raw_snapshot: dict[str, Any]) -> dict[str, dict[str, object]]:
    features = raw_snapshot.get("features")
    if not isinstance(features, list):
        raise ReuseAnalysisError("raw snapshot features must be a list")
    result: dict[str, dict[str, object]] = {}
    for feature in features:
        if not isinstance(feature, dict):
            raise ReuseAnalysisError("raw snapshot feature must be a mapping")
        selected = {
            "path": feature["path"],
            "value_type": feature["value_type"],
            "value": feature["value"],
            "occurs_in": list(feature.get("occurs_in", [])),
        }
        result[_raw_candidate_id(selected)] = selected
    return result


def _index_routes(index: dict[str, Any]) -> dict[str, str]:
    topics = index.get("topics")
    if not isinstance(topics, dict):
        raise ReuseAnalysisError("ADR INDEX topics must be a mapping")
    routes: dict[str, str] = {}
    for route in topics.values():
        if not isinstance(route, dict):
            continue
        adr_id = route.get("current_adr")
        path = route.get("path")
        if isinstance(adr_id, str) and isinstance(path, str):
            routes[adr_id] = path
    return routes


def _authority_contract(repo_root: Path, adr_id: str, routes: dict[str, str]) -> dict[str, object]:
    route = routes.get(adr_id)
    if route is None:
        raise ReuseAnalysisError(f"ADR INDEX has no current route for {adr_id}")
    record = _load_yaml(repo_root / route, label=adr_id)
    contract = record.get("authority_contract")
    if not isinstance(contract, dict):
        raise ReuseAnalysisError(f"{adr_id} authority_contract must be a mapping")
    authority_type = contract.get("authority_type")
    schema_id = contract.get("schema_id")
    schema_version = contract.get("schema_version")
    if not isinstance(authority_type, str) or not isinstance(schema_id, str):
        raise ReuseAnalysisError(f"{adr_id} authority_contract identity is incomplete")
    if type(schema_version) is not int or schema_version < 1:
        raise ReuseAnalysisError(
            f"{adr_id} authority_contract schema_version must be a positive integer"
        )
    return {
        "authority_type": authority_type,
        "schema_id": schema_id,
        "schema_version": schema_version,
    }


def _string_reuse_shape(value: str) -> dict[str, object]:
    if _GIT_SHA1.fullmatch(value):
        return {"kind": "GIT_SHA1_REVISION"}
    if _VERSION_LIKE.fullmatch(value):
        return {"kind": "VERSION_LIKE_IDENTIFIER"}
    if "/" in value or "\\" in value:
        return {"kind": "REPOSITORY_PATH", "suffix": Path(value).suffix or None}
    return {
        "kind": "STRING_EXACT",
        "value_sha256": hashlib.sha256(value.encode("utf-8")).hexdigest(),
    }


def _array_reuse_shape(value: list[object]) -> dict[str, object]:
    if value and all(isinstance(item, str) for item in value):
        strings = [str(item) for item in value]
        matches = [_RULE_ID.fullmatch(item) for item in strings]
        if all(match is not None for match in matches):
            prefixes = {match.group(1) for match in matches if match is not None}
            widths = {len(match.group(2)) for match in matches if match is not None}
            if len(prefixes) == 1 and len(widths) == 1:
                return {
                    "kind": "RULE_ID_ARRAY_PATTERN",
                    "pattern": f"{next(iter(prefixes))}{'#' * next(iter(widths))}",
                }
    return {
        "kind": "ARRAY_EXACT",
        "value_sha256": hashlib.sha256(_canonical_json(value).encode("utf-8")).hexdigest(),
    }


def reuse_shape(value_type: str, value: object) -> dict[str, object]:
    if value_type == "BOOLEAN":
        return {"kind": "BOOLEAN", "value": value}
    if value_type in {"INTEGER", "NUMBER"}:
        return {"kind": value_type, "value": value}
    if value_type == "NULL":
        return {"kind": "NULL"}
    if value_type == "STRING" and isinstance(value, str):
        return _string_reuse_shape(value)
    if value_type == "ARRAY" and isinstance(value, list):
        return _array_reuse_shape(value)
    if value_type == "OBJECT" and isinstance(value, dict):
        return {
            "kind": "OBJECT_KEYSET",
            "keys": sorted(value),
        }
    return {
        "kind": f"{value_type}_EXACT",
        "value_sha256": hashlib.sha256(_canonical_json(value).encode("utf-8")).hexdigest(),
    }


def _reuse_signature(
    authority_contract: dict[str, object],
    feature: dict[str, object],
) -> str:
    payload = {
        "authority_contract": authority_contract,
        "path": feature["path"],
        "value_type": feature["value_type"],
        "reuse_shape": reuse_shape(str(feature["value_type"]), feature["value"]),
    }
    return hashlib.sha256(_canonical_json(payload).encode("utf-8")).hexdigest()


def _normalize_token(token: str) -> str:
    aliases = {
        "activated": "activate",
        "activation": "activate",
        "classifications": "classification",
        "roles": "role",
        "relationships": "relationship",
        "rules": "rule",
        "states": "state",
        "modes": "mode",
        "bindings": "binding",
        "versions": "version",
        "identities": "identity",
        "interfaces": "interface",
        "producers": "producer",
        "origins": "origin",
    }
    value = token.lower()
    if value in aliases:
        return aliases[value]
    if value.endswith("s") and len(value) > 4 and not value.endswith("ss"):
        return value[:-1]
    return value


def _semantic_tokens(value: str) -> tuple[str, ...]:
    return tuple(
        _normalize_token(token)
        for token in value.lower().split("_")
        if token
    )


def _structural_similarity(left: str, right: str) -> float:
    left_tokens = _semantic_tokens(left)
    right_tokens = _semantic_tokens(right)
    if not left_tokens or not right_tokens:
        return 0.0
    left_set = set(left_tokens)
    right_set = set(right_tokens)
    overlap = len(left_set & right_set)
    token_score = (
        (2.0 * overlap) / (len(left_set) + len(right_set))
        if overlap
        else 0.0
    )
    head_score = 1.0 if left_tokens[-1] == right_tokens[-1] else 0.0
    sequence_score = SequenceMatcher(None, left.lower(), right.lower()).ratio()
    return 0.50 * head_score + 0.35 * token_score + 0.15 * sequence_score


def _field_tail(path: str) -> str:
    return path.rsplit(".", 1)[-1].lower()


def _semantic_target(record: dict[str, Any]) -> str | None:
    decision = record.get("decision")
    if decision in {"EXISTING", "REFINE_EXISTING"}:
        target = record.get("target_dimension")
        return str(target) if isinstance(target, str) else None
    if decision == "NEW_DIMENSION":
        target = record.get("proposed_dimension_id")
        return str(target) if isinstance(target, str) else None
    return None


def _concrete_reuse_action(
    record: dict[str, Any],
    *,
    current_matched_dimensions: set[str],
    current_dimensions: set[str],
    allowed_predicate_modes: set[str],
) -> dict[str, object] | None:
    decision = record.get("decision")
    predicate_mode = record.get("predicate_mode")
    semantic_target = _semantic_target(record)

    if decision == "NON_EFFECT":
        return {
            "decision": "NON_EFFECT",
            "target_dimension": None,
            "proposed_dimension_id": None,
            "predicate_mode": None,
            "reason_code": "PARAMETER_REFERENCE_OR_NON_EFFECT_DETAIL",
        }

    if decision == "REFINE_EXISTING":
        if (
            semantic_target in current_dimensions
            and predicate_mode in allowed_predicate_modes
        ):
            return {
                "decision": "REFINE_EXISTING",
                "target_dimension": semantic_target,
                "proposed_dimension_id": None,
                "predicate_mode": predicate_mode,
                "reason_code": "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
            }
        return None

    if decision == "NEW_DIMENSION":
        if (
            semantic_target in current_dimensions
            and predicate_mode in allowed_predicate_modes
        ):
            return {
                "decision": "REFINE_EXISTING",
                "target_dimension": semantic_target,
                "proposed_dimension_id": None,
                "predicate_mode": predicate_mode,
                "reason_code": "SAME_EFFECT_DIFFERENT_MACHINE_FIELD",
            }
        return None

    if decision == "EXISTING" and semantic_target in current_matched_dimensions:
        return {
            "decision": "EXISTING",
            "target_dimension": semantic_target,
            "proposed_dimension_id": None,
            "predicate_mode": None,
            "reason_code": "SAME_EFFECT_ALREADY_REPRESENTED",
        }

    return None


def analyze_reuse(
    repo_root: Path,
    adr_id: str,
    residual_candidates: list[dict[str, Any]],
    ledger: dict[str, Any],
    raw_snapshot: dict[str, Any],
    dimensions: dict[str, Any],
    matched_dimensions: list[str],
    predicate_modes_by_candidate: dict[str, list[str]],
) -> dict[str, Any]:
    records = ledger.get("records")
    if not isinstance(records, dict):
        raise ReuseAnalysisError("semantic decision ledger records must be a mapping")

    candidate_index = _candidate_index(raw_snapshot)
    index = _load_yaml(repo_root / "decisions/INDEX.yaml", label="ADR INDEX")
    routes = _index_routes(index)
    current_contract = _authority_contract(repo_root, adr_id, routes)
    current_dimensions = set(dimensions)
    current_matched = set(matched_dimensions)

    prior_rows: list[dict[str, Any]] = []
    exact_index: dict[str, list[dict[str, Any]]] = {}
    for prior_candidate_id, record in records.items():
        if not isinstance(record, dict) or record.get("decision") == "DEFER":
            continue
        feature = candidate_index.get(str(prior_candidate_id))
        first_reviewed_in = record.get("first_reviewed_in")
        if feature is None or not isinstance(first_reviewed_in, str):
            continue
        try:
            prior_contract = _authority_contract(repo_root, first_reviewed_in, routes)
        except ReuseAnalysisError:
            continue
        row = {
            "candidate_id": str(prior_candidate_id),
            "record": record,
            "feature": feature,
            "authority_contract": prior_contract,
            "reuse_signature": _reuse_signature(prior_contract, feature),
        }
        prior_rows.append(row)
        exact_index.setdefault(row["reuse_signature"], []).append(row)

    automatic_reuse: list[dict[str, object]] = []
    structural_candidates: dict[str, dict[str, object]] = {}

    for candidate in residual_candidates:
        candidate_id = str(candidate["candidate_id"])
        allowed_modes = set(predicate_modes_by_candidate.get(candidate_id, []))
        feature = {
            "path": candidate["path"],
            "value_type": candidate["value_type"],
            "value": candidate["value"],
        }
        signature = _reuse_signature(current_contract, feature)

        exact_rows = exact_index.get(signature, [])
        exact_actions: list[tuple[dict[str, Any], dict[str, object]]] = []
        for row in exact_rows:
            action = _concrete_reuse_action(
                row["record"],
                current_matched_dimensions=current_matched,
                current_dimensions=current_dimensions,
                allowed_predicate_modes=allowed_modes,
            )
            if action is not None:
                exact_actions.append((row, action))

        # Automatic reuse is allowed only when every reusable exact precedent
        # resolves to the same concrete action. Conflicting precedent is sent to AI.
        if exact_rows and len(exact_actions) == len(exact_rows):
            action_fingerprints = {
                _canonical_json(action)
                for _, action in exact_actions
            }
            if len(action_fingerprints) == 1:
                row, action = sorted(
                    exact_actions,
                    key=lambda item: str(item[0]["candidate_id"]),
                )[0]
                automatic_reuse.append(
                    {
                        "candidate_id": candidate_id,
                        **action,
                        "reuse_mode": "DETERMINISTIC_EXACT_CONTEXT_REUSE",
                        "reused_from_candidate_id": row["candidate_id"],
                        "semantic_authority": False,
                    }
                )
                continue

        raw_tail = _field_tail(str(candidate["path"]))
        ranked: list[dict[str, object]] = []
        for row in prior_rows:
            record = row["record"]
            action = _concrete_reuse_action(
                record,
                current_matched_dimensions=current_matched,
                current_dimensions=current_dimensions,
                allowed_predicate_modes=allowed_modes,
            )
            if action is None:
                continue
            prior_tail = _field_tail(str(row["feature"]["path"]))
            structural = _structural_similarity(raw_tail, prior_tail)
            same_contract = row["authority_contract"] == current_contract
            same_value_type = row["feature"]["value_type"] == candidate["value_type"]
            routing_score = (
                0.70 * structural
                + (0.20 if same_contract else 0.0)
                + (0.10 if same_value_type else 0.0)
            )
            if routing_score < 0.66:
                continue
            ranked.append(
                {
                    "prior_candidate_id": row["candidate_id"],
                    "prior_raw_path": row["feature"]["path"],
                    "prior_decision": record["decision"],
                    "resolved_dimension": _semantic_target(record),
                    "concrete_reuse_action": action,
                    "same_authority_contract": same_contract,
                    "same_value_type": same_value_type,
                    "routing_score": round(routing_score, 4),
                    "confirmation_required": True,
                    "semantic_authority": False,
                }
            )

        ranked.sort(
            key=lambda item: (
                -float(item["routing_score"]),
                str(item["prior_candidate_id"]),
            )
        )
        if ranked:
            structural_candidates[candidate_id] = ranked[0]

    return {
        "automatic_reuse_actions": automatic_reuse,
        "structural_reuse_candidates": structural_candidates,
        "invariants": {
            "automatic_reuse_requires_exact_context_signature": True,
            "automatic_reuse_requires_unanimous_prior_concrete_action": True,
            "structural_reuse_requires_ai_confirmation": True,
            "routing_score_is_semantic_authority": False,
        },
    }
