from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
from difflib import SequenceMatcher
import sys
from pathlib import Path
from typing import Any

import yaml


REVIEW_PACKET_SCHEMA_VERSION = "ptsip-p03-authority-role-review-packet/v1"
COMPACT_PACKET_SCHEMA_VERSION = "ptsip-p03-authority-role-review-packet-compact/v1"
DEFAULT_REGISTRY = "planning/0.4.0/WU-02/p03-authority-role-provisional-dimensions.yaml"
DEFAULT_RAW_SNAPSHOT = "planning/0.4.0/WU-02/p03-authority-role-raw-features.generated.yaml"
DEFAULT_OUTPUT_DIR = "planning/0.4.0/WU-02/p03-authority-role-review-packets"


class ReviewPacketError(ValueError):
    pass


def _load_module(path: Path, name: str):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise ReviewPacketError(f"cannot load module: {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _display_path(path: Path, repo_root: Path) -> str:
    resolved_path = path.resolve()
    resolved_root = repo_root.resolve()
    if resolved_path.is_relative_to(resolved_root):
        return resolved_path.relative_to(resolved_root).as_posix()
    return resolved_path.as_posix()


def _load_yaml(path: Path, *, label: str) -> dict[str, Any]:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except FileNotFoundError as exc:
        raise ReviewPacketError(f"{label} not found: {path}") from exc
    except yaml.YAMLError as exc:
        raise ReviewPacketError(f"{label} is not valid YAML: {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise ReviewPacketError(f"{label} root must be a mapping: {path}")
    return payload


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


def _top_level_semantic_path(path: str) -> str:
    prefix = "authority_semantics."
    if not path.startswith(prefix):
        raise ReviewPacketError(f"predicate path is outside authority_semantics: {path!r}")
    tail = path[len(prefix):]
    head = tail.split(".", 1)[0]
    if not head:
        raise ReviewPacketError(f"predicate path has no semantic field: {path!r}")
    return f"{prefix}{head}"


def _trace_expression(
    analyzer,
    expression: object,
    record: dict[str, Any],
    known_paths: dict[str, list[object]],
) -> tuple[bool, tuple[str, ...]]:
    if not isinstance(expression, dict) or not expression:
        raise ReviewPacketError("dimension expression must be a non-empty mapping")

    branch_keys = set(expression) & {"all", "any", "not"}
    if branch_keys:
        if len(branch_keys) != 1 or set(expression) != branch_keys:
            raise ReviewPacketError("expression branch must contain exactly one of all/any/not")
        branch = next(iter(branch_keys))
        value = expression[branch]

        if branch == "all":
            if not isinstance(value, list) or not value:
                raise ReviewPacketError("all expression must be a non-empty list")
            children = [_trace_expression(analyzer, item, record, known_paths) for item in value]
            if not all(result for result, _ in children):
                return False, ()
            paths = sorted({path for _, child_paths in children for path in child_paths})
            return True, tuple(paths)

        if branch == "any":
            if not isinstance(value, list) or not value:
                raise ReviewPacketError("any expression must be a non-empty list")
            successes = [
                trace
                for item in value
                if (trace := _trace_expression(analyzer, item, record, known_paths))[0]
            ]
            if not successes:
                return False, ()
            successes.sort(key=lambda item: (len(item[1]), item[1]))
            return successes[0]

        child_result, _ = _trace_expression(analyzer, value, record, known_paths)
        return (not child_result), ()

    result = analyzer._evaluate_condition(expression, record, known_paths)
    if not result:
        return False, ()
    path = expression.get("path")
    if not isinstance(path, str):
        raise ReviewPacketError("matched predicate has no canonical path")
    return True, (_top_level_semantic_path(path),)


def _raw_features_for_adr(snapshot: dict[str, Any], adr_id: str) -> list[dict[str, object]]:
    rows = snapshot.get("rows")
    features = snapshot.get("features")
    if not isinstance(rows, dict) or adr_id not in rows:
        raise ReviewPacketError(f"raw snapshot has no row for {adr_id}")
    if not isinstance(features, list):
        raise ReviewPacketError("raw snapshot features must be a list")

    selected: list[dict[str, object]] = []
    for feature in features:
        if not isinstance(feature, dict):
            raise ReviewPacketError("raw snapshot feature must be a mapping")
        occurs_in = feature.get("occurs_in")
        if isinstance(occurs_in, list) and adr_id in occurs_in:
            selected.append(
                {
                    "path": feature["path"],
                    "value_type": feature["value_type"],
                    "value": feature["value"],
                    "occurs_in": list(occurs_in),
                }
            )
    selected.sort(key=lambda item: str(item["path"]))
    expected = rows[adr_id].get("raw_feature_count")
    if expected != len(selected):
        raise ReviewPacketError(
            f"raw feature count mismatch for {adr_id}: expected={expected} actual={len(selected)}"
        )
    return selected


def _expression_paths(expression: object) -> tuple[str, ...]:
    if not isinstance(expression, dict):
        return ()
    if "path" in expression and isinstance(expression["path"], str):
        return (_top_level_semantic_path(expression["path"]),)
    paths: set[str] = set()
    for branch in ("all", "any"):
        value = expression.get(branch)
        if isinstance(value, list):
            for item in value:
                paths.update(_expression_paths(item))
    if "not" in expression:
        paths.update(_expression_paths(expression["not"]))
    return tuple(sorted(paths))


def _field_tail(path: str) -> str:
    return path.rsplit(".", 1)[-1].lower()


def _normalize_token(token: str) -> str:
    value = token.lower()
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
    }
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
    if overlap == 0:
        token_score = 0.0
    else:
        token_score = (2.0 * overlap) / (len(left_set) + len(right_set))

    head_score = 1.0 if left_tokens[-1] == right_tokens[-1] else 0.0
    sequence_score = SequenceMatcher(None, left.lower(), right.lower()).ratio()

    return 0.50 * head_score + 0.35 * token_score + 0.15 * sequence_score


def _existing_dimension_routing_candidates(
    residual: list[dict[str, object]],
    dimensions: dict[str, dict[str, Any]],
) -> list[dict[str, object]]:
    routed: list[dict[str, object]] = []
    for feature in residual:
        raw_path = str(feature["path"])
        raw_tail = _field_tail(raw_path)
        candidates: list[dict[str, object]] = []

        for dimension_id, definition in dimensions.items():
            expression = definition.get("expression")
            predicate_paths = _expression_paths(expression)
            evidence: list[dict[str, object]] = []

            dimension_score = _structural_similarity(raw_tail, dimension_id)
            if dimension_score >= 0.30:
                evidence.append(
                    {
                        "basis": "DIMENSION_ID_STRUCTURAL_SIMILARITY",
                        "target": dimension_id,
                        "score": round(dimension_score, 4),
                    }
                )

            for predicate_path in predicate_paths:
                predicate_tail = _field_tail(predicate_path)
                score = _structural_similarity(raw_tail, predicate_tail)
                if score >= 0.30:
                    evidence.append(
                        {
                            "basis": "PREDICATE_PATH_STRUCTURAL_SIMILARITY",
                            "target": predicate_path,
                            "score": round(score, 4),
                        }
                    )

            if evidence:
                evidence.sort(key=lambda item: (-float(item["score"]), str(item["target"])))
                candidates.append(
                    {
                        "dimension_id": dimension_id,
                        "routing_score": evidence[0]["score"],
                        "evidence": evidence[:3],
                    }
                )

        candidates.sort(
            key=lambda item: (-float(item["routing_score"]), str(item["dimension_id"]))
        )
        if candidates:
            routed.append(
                {
                    "candidate_id": _raw_candidate_id(feature),
                    "raw_path": raw_path,
                    "routing_only": True,
                    "semantic_authority": False,
                    "candidate_dimensions": candidates[:3],
                }
            )
    return routed


def _machine_residual_candidates(
    residual: list[dict[str, object]],
) -> list[dict[str, object]]:
    candidates: list[dict[str, object]] = []
    for feature in residual:
        candidates.append(
            {
                "candidate_id": _raw_candidate_id(feature),
                "status": "UNINTERPRETED_RAW_BACKED_CANDIDATE",
                "path": feature["path"],
                "value_type": feature["value_type"],
                "value": feature["value"],
                "occurs_in": feature["occurs_in"],
                "semantic_effect_dimension": "NOT_DECIDED",
            }
        )
    return candidates


def build_review_packet(
    repo_root: Path,
    adr_id: str,
    registry_path: Path,
    raw_snapshot_path: Path,
) -> dict[str, Any]:
    analyzer = _load_module(
        repo_root / ".github/scripts/p03_authority_role_matrix.py",
        "p03_authority_role_matrix",
    )
    raw_collector = _load_module(
        repo_root / ".github/scripts/p03_authority_role_raw_features.py",
        "p03_authority_role_raw_features",
    )

    expected_raw = raw_collector.build_snapshot(repo_root)
    persisted_raw = _load_yaml(raw_snapshot_path, label="P03 raw feature snapshot")
    raw_collector.validate_snapshot(persisted_raw)
    if persisted_raw != expected_raw:
        raise ReviewPacketError(
            "P03 raw feature snapshot is stale; regenerate it before semantic review"
        )

    contracts, known_paths = analyzer._load_authority_contracts(repo_root)
    registry = analyzer._load_registry(registry_path, known_paths)
    index = _load_yaml(repo_root / "decisions/INDEX.yaml", label="ADR INDEX")
    routes = analyzer._index_routes(index)
    route_path = routes.get(adr_id)
    if route_path is None:
        raise ReviewPacketError(f"ADR INDEX has no current route for {adr_id}")

    record = _load_yaml(repo_root / route_path, label=adr_id)
    analyzer._validate_instance(
        record,
        analyzer._load_schema(repo_root / "schemas/ptsip-adr.schema.json"),
        label=adr_id,
    )
    decision = record.get("decision")
    if not isinstance(decision, dict) or decision.get("id") != adr_id:
        raise ReviewPacketError(f"{route_path} does not contain decision.id {adr_id}")

    contract = record["authority_contract"]
    identity = (
        contract["authority_type"],
        contract["schema_id"],
        contract["schema_version"],
    )
    semantic_schema = contracts.get(identity)
    if semantic_schema is None:
        raise ReviewPacketError(f"{adr_id} has an unregistered authority contract: {identity}")
    analyzer._validate_instance(
        record["authority_semantics"],
        semantic_schema,
        label=f"{adr_id} authority_semantics",
    )

    matched_dimensions: list[dict[str, object]] = []
    covered_paths: set[str] = set()
    dimensions = registry["dimensions"]
    for dimension_id, definition in dimensions.items():
        result, proof_paths = _trace_expression(
            analyzer,
            definition["expression"],
            record,
            known_paths,
        )
        if result:
            matched_dimensions.append(
                {
                    "dimension_id": dimension_id,
                    "proof_paths": list(proof_paths),
                }
            )
            covered_paths.update(proof_paths)

    raw_features = _raw_features_for_adr(persisted_raw, adr_id)
    residual = [item for item in raw_features if item["path"] not in covered_paths]
    covered = [item for item in raw_features if item["path"] in covered_paths]
    candidates = _machine_residual_candidates(residual)
    routing_candidates = _existing_dimension_routing_candidates(residual, dimensions)

    packet: dict[str, Any] = {
        "schema_version": REVIEW_PACKET_SCHEMA_VERSION,
        "target": {
            "adr_id": adr_id,
            "source_path": route_path,
            "authority_type": contract["authority_type"],
        },
        "inputs": {
            "raw_snapshot": _display_path(raw_snapshot_path, repo_root),
            "dimension_registry": _display_path(registry_path, repo_root),
            "dimension_count": len(dimensions),
            "runtime_authority": "NONE",
            "vocabulary_registration": False,
        },
        "automation": {
            "existing_dimension_evaluation": "DETERMINISTIC",
            "matched_predicate_proof_trace": "DETERMINISTIC",
            "residual_raw_feature_extraction": "DETERMINISTIC",
            "machine_residual_candidate_generation": "DETERMINISTIC",
            "existing_dimension_candidate_routing": "DETERMINISTIC_STRUCTURAL_NON_AUTHORITATIVE",
            "manual_full_dimension_scan_required": False,
            "manual_raw_corpus_search_required": False,
            "raw_feature_force_fit": "FORBIDDEN",
            "natural_language_consumption": "FORBIDDEN",
            "semantic_effect_promotion": "DESIGN_REVIEW_ONLY",
        },
        "matched_dimensions": matched_dimensions,
        "covered_raw_features": covered,
        "residual_raw_features": residual,
        "machine_residual_candidates": candidates,
        "candidate_existing_dimension_routing": routing_candidates,
        "summary": {
            "raw_feature_count": len(raw_features),
            "matched_dimension_count": len(matched_dimensions),
            "covered_raw_feature_count": len(covered),
            "residual_raw_feature_count": len(residual),
            "machine_residual_candidate_count": len(candidates),
            "candidate_existing_dimension_routing_count": len(routing_candidates),
            "semantic_review_required": bool(residual),
        },
    }
    validate_review_packet(packet)
    return packet


def compact_review_packet(packet: dict[str, Any]) -> dict[str, Any]:
    validate_review_packet(packet)
    routing_by_id = {
        item["candidate_id"]: item
        for item in packet["candidate_existing_dimension_routing"]
    }
    routed: list[dict[str, object]] = []
    unrouted: list[dict[str, object]] = []

    for candidate in packet["machine_residual_candidates"]:
        route = routing_by_id.get(candidate["candidate_id"])
        base = {
            "candidate_id": candidate["candidate_id"],
            "path": candidate["path"],
            "value_type": candidate["value_type"],
            "value": candidate["value"],
            "occurs_in": candidate["occurs_in"],
            "semantic_effect_dimension": "NOT_DECIDED",
        }
        if route is None:
            unrouted.append(base)
        else:
            routed.append(
                {
                    **base,
                    "candidate_dimensions": route["candidate_dimensions"],
                    "routing_only": True,
                    "semantic_authority": False,
                }
            )

    return {
        "schema_version": COMPACT_PACKET_SCHEMA_VERSION,
        "target": packet["target"],
        "automation": {
            "existing_dimension_evaluation": "DETERMINISTIC",
            "existing_dimension_candidate_routing": "DETERMINISTIC_STRUCTURAL_NON_AUTHORITATIVE",
            "manual_full_dimension_scan_required": False,
            "manual_raw_corpus_search_required": False,
            "raw_feature_force_fit": "FORBIDDEN",
        },
        "matched_dimension_ids": [
            item["dimension_id"] for item in packet["matched_dimensions"]
        ],
        "routed_residual_candidates": routed,
        "unrouted_residual_candidates": unrouted,
        "summary": {
            **packet["summary"],
            "routed_residual_candidate_count": len(routed),
            "unrouted_residual_candidate_count": len(unrouted),
        },
    }


def validate_review_packet(packet: dict[str, Any]) -> None:
    if packet.get("schema_version") != REVIEW_PACKET_SCHEMA_VERSION:
        raise ReviewPacketError("invalid review packet schema_version")
    target = packet.get("target")
    inputs = packet.get("inputs")
    automation = packet.get("automation")
    matched = packet.get("matched_dimensions")
    covered = packet.get("covered_raw_features")
    residual = packet.get("residual_raw_features")
    candidates = packet.get("machine_residual_candidates")
    routing = packet.get("candidate_existing_dimension_routing")
    summary = packet.get("summary")
    if not all(isinstance(value, dict) for value in (target, inputs, automation, summary)):
        raise ReviewPacketError("review packet mappings are malformed")
    if not all(isinstance(value, list) for value in (matched, covered, residual, candidates, routing)):
        raise ReviewPacketError("review packet collections are malformed")
    if automation.get("manual_full_dimension_scan_required") is not False:
        raise ReviewPacketError("manual full dimension scan must not be required")
    if automation.get("manual_raw_corpus_search_required") is not False:
        raise ReviewPacketError("manual raw corpus search must not be required")
    if automation.get("raw_feature_force_fit") != "FORBIDDEN":
        raise ReviewPacketError("raw feature force-fit must stay forbidden")
    if automation.get("natural_language_consumption") != "FORBIDDEN":
        raise ReviewPacketError("review automation must not consume natural language")

    covered_paths = [item.get("path") for item in covered if isinstance(item, dict)]
    residual_paths = [item.get("path") for item in residual if isinstance(item, dict)]
    if set(covered_paths) & set(residual_paths):
        raise ReviewPacketError("raw feature cannot be both covered and residual")

    candidate_ids: set[str] = set()
    for candidate in candidates:
        if not isinstance(candidate, dict):
            raise ReviewPacketError("machine residual candidate must be a mapping")
        candidate_id = candidate.get("candidate_id")
        if not isinstance(candidate_id, str) or not candidate_id.startswith("raw-candidate:"):
            raise ReviewPacketError("machine residual candidate has invalid candidate_id")
        if candidate_id in candidate_ids:
            raise ReviewPacketError("machine residual candidate ids must be unique")
        candidate_ids.add(candidate_id)
        if candidate.get("path") not in residual_paths:
            raise ReviewPacketError("machine residual candidate must correspond to a residual raw feature")
        if candidate.get("semantic_effect_dimension") != "NOT_DECIDED":
            raise ReviewPacketError("machine residual candidate must not auto-create semantic effect authority")

    if summary.get("raw_feature_count") != len(covered) + len(residual):
        raise ReviewPacketError("review packet raw_feature_count is stale")
    if summary.get("covered_raw_feature_count") != len(covered):
        raise ReviewPacketError("review packet covered_raw_feature_count is stale")
    if summary.get("residual_raw_feature_count") != len(residual):
        raise ReviewPacketError("review packet residual_raw_feature_count is stale")
    if summary.get("matched_dimension_count") != len(matched):
        raise ReviewPacketError("review packet matched_dimension_count is stale")
    for route in routing:
        if not isinstance(route, dict):
            raise ReviewPacketError("existing-dimension routing entry must be a mapping")
        if route.get("routing_only") is not True or route.get("semantic_authority") is not False:
            raise ReviewPacketError("existing-dimension routing must remain non-authoritative")
        if route.get("raw_path") not in residual_paths:
            raise ReviewPacketError("existing-dimension routing must reference a residual raw path")
        route_candidates = route.get("candidate_dimensions")
        if not isinstance(route_candidates, list) or not route_candidates:
            raise ReviewPacketError("existing-dimension routing entry requires candidates")
        for candidate in route_candidates:
            if not isinstance(candidate, dict):
                raise ReviewPacketError("routed dimension candidate must be a mapping")
            score = candidate.get("routing_score")
            if not isinstance(score, (int, float)) or score < 0 or score > 1:
                raise ReviewPacketError("routing score must be between zero and one")

    if summary.get("machine_residual_candidate_count") != len(candidates):
        raise ReviewPacketError("review packet machine_residual_candidate_count is stale")
    if summary.get("candidate_existing_dimension_routing_count") != len(routing):
        raise ReviewPacketError("review packet candidate_existing_dimension_routing_count is stale")
    if len(candidates) != len(residual):
        raise ReviewPacketError("every residual raw feature requires one machine residual candidate")
    if summary.get("semantic_review_required") is not bool(residual):
        raise ReviewPacketError("review packet semantic_review_required is stale")


def _dump_yaml(payload: dict[str, Any]) -> str:
    return yaml.safe_dump(
        payload,
        sort_keys=False,
        allow_unicode=True,
        default_flow_style=False,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Build a minimal deterministic P03 semantic review packet for one ADR"
    )
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--adr", required=True)
    parser.add_argument("--registry", default=DEFAULT_REGISTRY)
    parser.add_argument("--raw-snapshot", default=DEFAULT_RAW_SNAPSHOT)
    parser.add_argument("--output")
    parser.add_argument("--write", action="store_true")
    parser.add_argument("--compact", action="store_true")
    args = parser.parse_args(argv)

    repo_root = Path(args.repo_root).resolve()
    registry_path = Path(args.registry)
    if not registry_path.is_absolute():
        registry_path = repo_root / registry_path
    raw_snapshot_path = Path(args.raw_snapshot)
    if not raw_snapshot_path.is_absolute():
        raw_snapshot_path = repo_root / raw_snapshot_path

    try:
        packet = build_review_packet(
            repo_root,
            args.adr,
            registry_path,
            raw_snapshot_path,
        )
        output_payload = compact_review_packet(packet) if args.compact else packet
        if args.write:
            suffix = ".compact.generated.yaml" if args.compact else ".generated.yaml"
            output = Path(args.output) if args.output else Path(DEFAULT_OUTPUT_DIR) / f"{args.adr}{suffix}"
            if not output.is_absolute():
                output = repo_root / output
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(_dump_yaml(output_payload), encoding="utf-8")
            print(output.relative_to(repo_root).as_posix())
        else:
            print(_dump_yaml(output_payload), end="")
    except (OSError, ReviewPacketError, ValueError, KeyError) as exc:
        print(f"P03 review-packet error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
