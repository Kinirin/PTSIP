from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator, SchemaError, ValidationError


SNAPSHOT_SCHEMA_VERSION = "ptsip-p03-authority-role-raw-features/v1"
DEFAULT_OUTPUT = "planning/0.4.0/WU-02/p03-authority-role-raw-features.generated.yaml"
DEFAULT_FIRST_ADR = "ADR-0001"
DEFAULT_LAST_ADR = "ADR-0023"
DEFAULT_SEMANTIC_REVIEW_THROUGH = "ADR-0010"


class RawFeatureError(ValueError):
    pass


def _load_yaml(path: Path, *, label: str) -> dict[str, Any]:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except FileNotFoundError as exc:
        raise RawFeatureError(f"{label} not found: {path}") from exc
    except yaml.YAMLError as exc:
        raise RawFeatureError(f"{label} is not valid YAML: {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise RawFeatureError(f"{label} root must be a mapping: {path}")
    return payload


def _load_schema(path: Path) -> dict[str, Any]:
    try:
        schema = json.loads(path.read_text(encoding="utf-8-sig"))
        Draft202012Validator.check_schema(schema)
    except (json.JSONDecodeError, SchemaError) as exc:
        raise RawFeatureError(f"invalid machine schema {path}: {exc}") from exc
    return schema


def _validate_instance(payload: object, schema: dict[str, Any], *, label: str) -> None:
    try:
        Draft202012Validator(schema).validate(payload)
    except ValidationError as exc:
        location = ".".join(str(part) for part in exc.absolute_path) or "<root>"
        raise RawFeatureError(f"{label} invalid at {location}: {exc.message}") from exc


def _adr_number(value: object, *, label: str) -> int:
    if not isinstance(value, str) or len(value) != 8 or not value.startswith("ADR-"):
        raise RawFeatureError(f"{label} must be ADR-NNNN: {value!r}")
    suffix = value[4:]
    if not suffix.isdigit():
        raise RawFeatureError(f"{label} must be ADR-NNNN: {value!r}")
    return int(suffix)


def _canonical_value(value: object) -> object:
    if isinstance(value, dict):
        return {
            str(key): _canonical_value(value[key])
            for key in sorted(value, key=lambda item: str(item))
        }
    if isinstance(value, list):
        return [_canonical_value(item) for item in value]
    return value


def _value_type(value: object) -> str:
    if value is None:
        return "NULL"
    if type(value) is bool:
        return "BOOLEAN"
    if type(value) is int:
        return "INTEGER"
    if type(value) is float:
        return "NUMBER"
    if isinstance(value, str):
        return "STRING"
    if isinstance(value, list):
        return "ARRAY"
    if isinstance(value, dict):
        return "OBJECT"
    raise RawFeatureError(f"unsupported authority semantic value type: {type(value).__name__}")


def _canonical_json(value: object) -> str:
    return json.dumps(
        _canonical_value(value),
        ensure_ascii=False,
        sort_keys=True,
        separators=(",", ":"),
    )


def raw_features_from_semantics(semantics: dict[str, Any]) -> list[dict[str, object]]:
    features: list[dict[str, object]] = []
    for key in sorted(semantics):
        value = _canonical_value(semantics[key])
        features.append(
            {
                "path": f"authority_semantics.{key}",
                "value_type": _value_type(value),
                "value": value,
            }
        )
    return features


def _load_contracts(
    repo_root: Path,
) -> tuple[dict[tuple[str, str, int], dict[str, Any]], dict[str, Any]]:
    registry = _load_yaml(
        repo_root / "decisions/AUTHORITY-SCHEMA-REGISTRY.yaml",
        label="authority schema registry",
    )
    registry_schema = _load_schema(
        repo_root / "schemas/ptsip-governance-authority-registry.schema.json"
    )
    _validate_instance(registry, registry_schema, label="authority schema registry")

    semantics_schema_path = repo_root / str(registry["semantic_schema_document"])
    semantics_schema = _load_schema(semantics_schema_path)
    definitions = semantics_schema.get("$defs")
    if not isinstance(definitions, dict):
        raise RawFeatureError("authority semantics schema has no $defs mapping")

    contracts: dict[tuple[str, str, int], dict[str, Any]] = {}
    for entry in registry["entries"]:
        identity = (
            entry["authority_type"],
            entry["schema_id"],
            entry["schema_version"],
        )
        definition = definitions.get(entry["schema_definition"])
        if not isinstance(definition, dict) or not isinstance(definition.get("const"), dict):
            raise RawFeatureError(
                f"authority contract {identity} does not resolve to an exact semantic object"
            )
        if identity in contracts:
            raise RawFeatureError(f"duplicate authority contract identity: {identity}")
        contracts[identity] = definition
    return contracts, registry


def _index_routes(repo_root: Path) -> dict[str, str]:
    index = _load_yaml(repo_root / "decisions/INDEX.yaml", label="ADR INDEX")
    _validate_instance(
        index,
        _load_schema(repo_root / "schemas/ptsip-adr-index.schema.json"),
        label="ADR INDEX",
    )
    topics = index.get("topics")
    if not isinstance(topics, dict):
        raise RawFeatureError("ADR INDEX topics must be a mapping")

    routes: dict[str, str] = {}
    for topic_id, route in topics.items():
        if not isinstance(route, dict):
            raise RawFeatureError(f"ADR INDEX topic {topic_id!r} route must be a mapping")
        adr_id = route.get("current_adr")
        path = route.get("path")
        _adr_number(adr_id, label=f"ADR INDEX {topic_id}.current_adr")
        if not isinstance(path, str) or not path.endswith(".yaml"):
            raise RawFeatureError(f"ADR INDEX {topic_id}.path must be a YAML path")
        if adr_id in routes:
            raise RawFeatureError(f"ADR INDEX contains duplicate current ADR id: {adr_id}")
        routes[adr_id] = path
    return routes


def _feature_signature(feature: dict[str, object]) -> str:
    return _canonical_json(
        {
            "path": feature["path"],
            "value_type": feature["value_type"],
            "value": feature["value"],
        }
    )


def build_snapshot(
    repo_root: Path,
    *,
    first_adr: str = DEFAULT_FIRST_ADR,
    last_adr: str = DEFAULT_LAST_ADR,
    semantic_review_through: str = DEFAULT_SEMANTIC_REVIEW_THROUGH,
) -> dict[str, Any]:
    first = _adr_number(first_adr, label="first_adr")
    last = _adr_number(last_adr, label="last_adr")
    reviewed = _adr_number(semantic_review_through, label="semantic_review_through")
    if first > last:
        raise RawFeatureError("first_adr must not be after last_adr")
    if not first <= reviewed <= last:
        raise RawFeatureError("semantic_review_through must be inside the collected corpus")

    contracts, _ = _load_contracts(repo_root)
    routes = _index_routes(repo_root)
    adr_schema = _load_schema(repo_root / "schemas/ptsip-adr.schema.json")

    catalog: dict[str, dict[str, object]] = {}
    rows: dict[str, dict[str, object]] = {}

    for number in range(first, last + 1):
        adr_id = f"ADR-{number:04d}"
        route_path = routes.get(adr_id)
        if route_path is None:
            raise RawFeatureError(f"ADR INDEX has no current route for {adr_id}")

        record = _load_yaml(repo_root / route_path, label=adr_id)
        _validate_instance(record, adr_schema, label=adr_id)
        decision = record.get("decision")
        if not isinstance(decision, dict) or decision.get("id") != adr_id:
            raise RawFeatureError(f"{route_path} does not contain decision.id {adr_id}")

        contract = record["authority_contract"]
        identity = (
            contract["authority_type"],
            contract["schema_id"],
            contract["schema_version"],
        )
        semantic_schema = contracts.get(identity)
        if semantic_schema is None:
            raise RawFeatureError(f"{adr_id} has an unregistered authority contract: {identity}")
        semantics = record["authority_semantics"]
        _validate_instance(semantics, semantic_schema, label=f"{adr_id} authority_semantics")

        features = raw_features_from_semantics(semantics)
        for feature in features:
            signature = _feature_signature(feature)
            entry = catalog.get(signature)
            if entry is None:
                entry = {
                    "path": feature["path"],
                    "value_type": feature["value_type"],
                    "value": feature["value"],
                    "occurs_in": [],
                }
                catalog[signature] = entry
            occurrences = entry["occurs_in"]
            assert isinstance(occurrences, list)
            occurrences.append(adr_id)

        rows[adr_id] = {
            "source_path": route_path,
            "raw_feature_count": len(features),
        }

    features = sorted(
        catalog.values(),
        key=lambda item: (
            str(item["path"]),
            str(item["value_type"]),
            _canonical_json(item["value"]),
        ),
    )
    assertion_count = sum(int(row["raw_feature_count"]) for row in rows.values())

    payload: dict[str, Any] = {
        "schema_version": SNAPSHOT_SCHEMA_VERSION,
        "source": {
            "index_path": "decisions/INDEX.yaml",
            "first_adr": first_adr,
            "raw_collection_through": last_adr,
            "semantic_review_through": semantic_review_through,
            "authority_semantics_validation": "EXACT_REGISTERED_MACHINE_SCHEMA",
            "runtime_authority": "NONE",
            "vocabulary_registration": False,
            "semantic_grouping_performed": False,
        },
        "feature_model": {
            "unit": "EXACT_TOP_LEVEL_AUTHORITY_SEMANTIC_PATH_TYPED_VALUE",
            "binary_projection": "SPARSE_MACHINE_DERIVABLE",
            "true_when": "ADR_ID_IS_LISTED_IN_FEATURE_OCCURS_IN",
            "false_when": "ADR_ID_IS_NOT_LISTED_AFTER_EXACT_SOURCE_VALIDATION",
            "false_is_default_fill": False,
            "natural_language_consumption": "FORBIDDEN",
            "semantic_effect_name_generation": "FORBIDDEN",
        },
        "features": features,
        "rows": rows,
        "summary": {
            "adr_count": len(rows),
            "raw_assertion_count": assertion_count,
            "unique_raw_feature_count": len(features),
        },
    }
    validate_snapshot(payload)
    return payload


def validate_snapshot(payload: dict[str, Any]) -> None:
    if payload.get("schema_version") != SNAPSHOT_SCHEMA_VERSION:
        raise RawFeatureError(f"snapshot schema_version must be {SNAPSHOT_SCHEMA_VERSION!r}")

    source = payload.get("source")
    feature_model = payload.get("feature_model")
    features = payload.get("features")
    rows = payload.get("rows")
    summary = payload.get("summary")
    if not all(isinstance(item, dict) for item in (source, feature_model, rows, summary)):
        raise RawFeatureError("snapshot source, feature_model, rows and summary must be mappings")
    if not isinstance(features, list):
        raise RawFeatureError("snapshot features must be a list")

    if source.get("runtime_authority") != "NONE":
        raise RawFeatureError("raw feature snapshot must have runtime_authority: NONE")
    if source.get("vocabulary_registration") is not False:
        raise RawFeatureError("raw feature snapshot must not register vocabulary")
    if source.get("semantic_grouping_performed") is not False:
        raise RawFeatureError("raw collection must not perform semantic grouping")
    if feature_model.get("false_is_default_fill") is not False:
        raise RawFeatureError("raw feature absence may not be represented as blanket default fill")
    if feature_model.get("natural_language_consumption") != "FORBIDDEN":
        raise RawFeatureError("raw feature extraction must not consume natural language")
    if feature_model.get("semantic_effect_name_generation") != "FORBIDDEN":
        raise RawFeatureError("raw feature extraction must not generate semantic effect names")

    first = _adr_number(source.get("first_adr"), label="source.first_adr")
    last = _adr_number(source.get("raw_collection_through"), label="source.raw_collection_through")
    reviewed = _adr_number(
        source.get("semantic_review_through"), label="source.semantic_review_through"
    )
    if not first <= reviewed <= last:
        raise RawFeatureError("semantic review boundary is outside raw corpus")

    expected_rows = [f"ADR-{number:04d}" for number in range(first, last + 1)]
    if list(rows) != expected_rows:
        raise RawFeatureError("raw feature rows must exactly cover the configured ADR corpus")

    seen_signatures: set[str] = set()
    occurrence_count = 0
    row_counts = {adr_id: 0 for adr_id in expected_rows}
    for feature in features:
        if not isinstance(feature, dict) or set(feature) != {
            "path",
            "value_type",
            "value",
            "occurs_in",
        }:
            raise RawFeatureError("each raw feature must contain path, value_type, value, occurs_in")
        path = feature["path"]
        if not isinstance(path, str) or not path.startswith("authority_semantics."):
            raise RawFeatureError(f"raw feature path is outside authority_semantics: {path!r}")
        if feature["value_type"] != _value_type(feature["value"]):
            raise RawFeatureError(f"raw feature value_type mismatch at {path}")

        signature = _feature_signature(feature)
        if signature in seen_signatures:
            raise RawFeatureError(f"duplicate raw feature: {signature}")
        seen_signatures.add(signature)

        occurs_in = feature["occurs_in"]
        if not isinstance(occurs_in, list) or not occurs_in:
            raise RawFeatureError(f"raw feature {path} has no occurrences")
        if occurs_in != sorted(set(occurs_in)):
            raise RawFeatureError(f"raw feature {path} occurrences must be sorted and unique")
        for adr_id in occurs_in:
            if adr_id not in row_counts:
                raise RawFeatureError(f"raw feature {path} references ADR outside corpus: {adr_id}")
            row_counts[adr_id] += 1
            occurrence_count += 1

    for adr_id, row in rows.items():
        if not isinstance(row, dict) or set(row) != {"source_path", "raw_feature_count"}:
            raise RawFeatureError(f"raw row {adr_id} has unsupported fields")
        if row["raw_feature_count"] != row_counts[adr_id]:
            raise RawFeatureError(f"raw row {adr_id} feature count is stale")
        if not isinstance(row["source_path"], str) or not row["source_path"].endswith(".yaml"):
            raise RawFeatureError(f"raw row {adr_id} source_path is invalid")

    if summary.get("adr_count") != len(rows):
        raise RawFeatureError("summary adr_count is stale")
    if summary.get("raw_assertion_count") != occurrence_count:
        raise RawFeatureError("summary raw_assertion_count is stale")
    if summary.get("unique_raw_feature_count") != len(features):
        raise RawFeatureError("summary unique_raw_feature_count is stale")


def _dump_yaml(payload: dict[str, Any]) -> str:
    return yaml.safe_dump(
        payload,
        sort_keys=False,
        allow_unicode=True,
        default_flow_style=False,
    )


def _write_snapshot(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(_dump_yaml(payload), encoding="utf-8")


def _check_snapshot(path: Path, expected: dict[str, Any]) -> None:
    current = _load_yaml(path, label="generated P03 raw feature snapshot")
    validate_snapshot(current)
    if current != expected:
        raise RawFeatureError(
            "generated P03 raw feature snapshot is stale; run with --write to regenerate"
        )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Collect schema-validated raw Authority Role corpus features without semantic grouping"
    )
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--output", default=DEFAULT_OUTPUT)
    parser.add_argument("--first-adr", default=DEFAULT_FIRST_ADR)
    parser.add_argument("--last-adr", default=DEFAULT_LAST_ADR)
    parser.add_argument(
        "--semantic-review-through",
        default=DEFAULT_SEMANTIC_REVIEW_THROUGH,
    )
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--write", action="store_true")
    mode.add_argument("--check", action="store_true")
    args = parser.parse_args(argv)

    repo_root = Path(args.repo_root).resolve()
    output_path = Path(args.output)
    if not output_path.is_absolute():
        output_path = repo_root / output_path

    try:
        payload = build_snapshot(
            repo_root,
            first_adr=args.first_adr,
            last_adr=args.last_adr,
            semantic_review_through=args.semantic_review_through,
        )
        if args.write:
            _write_snapshot(output_path, payload)
            action = "wrote"
        else:
            _check_snapshot(output_path, payload)
            action = "validated"
    except (OSError, RawFeatureError) as exc:
        print(f"P03 raw-feature error: {exc}", file=sys.stderr)
        return 1

    summary = payload["summary"]
    print(
        f"P03 raw features {action}: {summary['adr_count']} ADR(s), "
        f"{summary['raw_assertion_count']} raw assertion(s), "
        f"{summary['unique_raw_feature_count']} unique feature(s)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
