from __future__ import annotations

import argparse
import hashlib
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Sequence

from jsonschema import Draft202012Validator


SOURCE_PATH = Path(".ptsip/context/source/context.source.json")
JSON_PATH = Path(".ptsip/context/context.json")
JSONL_PATH = Path(".ptsip/context/context.jsonl")
SCHEMA_PATH = Path(".ptsip/context/context.schema.json")
MANDATORY_FORMATS = ("json", "jsonl", "schema-json")


class ContextProjectionError(RuntimeError):
    """Raised when the provider-neutral context plane cannot be projected safely."""


@dataclass(frozen=True)
class ContextProjectionSet:
    source: bytes
    json_projection: bytes
    jsonl_projection: bytes
    schema_projection: bytes

    def as_paths(self) -> dict[Path, bytes]:
        return {
            SOURCE_PATH: self.source,
            JSON_PATH: self.json_projection,
            JSONL_PATH: self.jsonl_projection,
            SCHEMA_PATH: self.schema_projection,
        }


def _json_bytes(value: object) -> bytes:
    return (
        json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n"
    ).encode("utf-8")


def _jsonl_bytes(records: Sequence[Mapping[str, object]]) -> bytes:
    return (
        "\n".join(
            json.dumps(
                record,
                ensure_ascii=False,
                sort_keys=True,
                separators=(",", ":"),
            )
            for record in records
        )
        + "\n"
    ).encode("utf-8")


def _sha256(payload: bytes) -> str:
    return hashlib.sha256(payload).hexdigest()


def _mapping(value: object, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise ContextProjectionError(f"{label} must be an object")
    return value


def _validate_source(source: object) -> dict[str, object]:
    payload = dict(_mapping(source, "context source"))
    if payload.get("format") != "ptsip-context-source/v1":
        raise ContextProjectionError("context source format must be ptsip-context-source/v1")

    state = _mapping(payload.get("state"), "context source state")
    if state.get("format") != "ptsip-project-state/v1":
        raise ContextProjectionError("state format must be ptsip-project-state/v1")

    memory = payload.get("memory")
    if not isinstance(memory, list):
        raise ContextProjectionError("context source memory must be an array")
    ids: list[str] = []
    for index, raw in enumerate(memory):
        event = _mapping(raw, f"memory[{index}]")
        if event.get("format") != "ptsip-memory-event/v1":
            raise ContextProjectionError(
                f"memory[{index}] format must be ptsip-memory-event/v1"
            )
        event_id = event.get("id")
        if not isinstance(event_id, str) or not event_id:
            raise ContextProjectionError(f"memory[{index}] id must be non-empty")
        ids.append(event_id)
    if len(ids) != len(set(ids)):
        raise ContextProjectionError("memory event ids must be unique")

    policy = _mapping(payload.get("projection_policy"), "projection_policy")
    if policy.get("authority") != "CANONICAL_SEMANTIC_MODEL":
        raise ContextProjectionError(
            "projection_policy.authority must be CANONICAL_SEMANTIC_MODEL"
        )
    if policy.get("generated_only") is not True:
        raise ContextProjectionError("generated projections must be generated_only")
    if policy.get("semantic_equivalence") != "REQUIRED":
        raise ContextProjectionError("semantic equivalence must be REQUIRED")
    formats = policy.get("mandatory_formats")
    if not isinstance(formats, list) or tuple(formats) != MANDATORY_FORMATS:
        raise ContextProjectionError(
            "mandatory_formats must be exactly json, jsonl, schema-json"
        )
    providers = policy.get("provider_scope")
    if not isinstance(providers, list) or not providers:
        raise ContextProjectionError("provider_scope must be a non-empty array")
    if policy.get("extension_policy") != "EXTENSIBLE":
        raise ContextProjectionError("extension_policy must be EXTENSIBLE")
    return payload


def _projection_schema() -> dict[str, object]:
    object_map: dict[str, object] = {
        "type": "object",
        "additionalProperties": True,
    }
    memory_event: dict[str, object] = {
        "type": "object",
        "required": ["format", "id", "recorded_on", "type", "subject", "status"],
        "properties": {
            "format": {"const": "ptsip-memory-event/v1"},
            "id": {"type": "string", "pattern": "^PTSIP-MEM-[0-9]{6}$"},
            "recorded_on": {"type": "string", "format": "date"},
            "type": {"type": "string", "pattern": "^[A-Z][A-Z0-9_]*$"},
            "subject": {"type": "string", "minLength": 1},
            "status": {"type": "string", "pattern": "^[A-Z][A-Z0-9_]*$"},
            "refs": {
                "type": "array",
                "items": {"type": "string", "minLength": 1},
                "uniqueItems": True,
            },
            "data": {"type": "object"},
        },
        "additionalProperties": False,
    }
    policy: dict[str, object] = {
        "type": "object",
        "required": [
            "authority",
            "generated_only",
            "semantic_equivalence",
            "mandatory_formats",
            "provider_scope",
            "extension_policy",
        ],
        "properties": {
            "authority": {"const": "CANONICAL_SEMANTIC_MODEL"},
            "generated_only": {"const": True},
            "semantic_equivalence": {"const": "REQUIRED"},
            "mandatory_formats": {
                "type": "array",
                "prefixItems": [{"const": value} for value in MANDATORY_FORMATS],
                "items": False,
                "minItems": len(MANDATORY_FORMATS),
                "maxItems": len(MANDATORY_FORMATS),
            },
            "provider_scope": {
                "type": "array",
                "items": {"type": "string", "minLength": 1},
                "uniqueItems": True,
                "minItems": 1,
            },
            "extension_policy": {"const": "EXTENSIBLE"},
        },
        "additionalProperties": False,
    }
    source_binding: dict[str, object] = {
        "type": "object",
        "required": ["path", "sha256"],
        "properties": {
            "path": {"const": SOURCE_PATH.as_posix()},
            "sha256": {"type": "string", "pattern": "^[0-9a-f]{64}$"},
        },
        "additionalProperties": False,
    }
    state: dict[str, object] = {
        "type": "object",
        "required": [
            "format",
            "status",
            "project_profile",
            "specification",
            "tool",
            "planning",
            "agent_contract",
            "verification",
            "context_plane",
            "legacy_markdown_state",
        ],
        "properties": {
            "format": {"const": "ptsip-project-state/v1"},
            "status": {"type": "string", "minLength": 1},
            "project_profile": object_map,
            "specification": object_map,
            "tool": object_map,
            "planning": object_map,
            "agent_contract": object_map,
            "verification": object_map,
            "context_plane": object_map,
            "legacy_markdown_state": object_map,
        },
        "additionalProperties": True,
    }
    return {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "https://github.com/Kinirin/PTSIP/.ptsip/context/context.schema.json",
        "title": "PTSIP Provider-Neutral Context Plane",
        "$defs": {
            "projection_policy": policy,
            "source_binding": source_binding,
            "state": state,
            "memory_event": memory_event,
            "context_bundle": {
                "type": "object",
                "required": [
                    "format",
                    "source_binding",
                    "projection_policy",
                    "state",
                    "memory",
                ],
                "properties": {
                    "format": {"const": "ptsip-context/v1"},
                    "source_binding": {"$ref": "#/$defs/source_binding"},
                    "projection_policy": {"$ref": "#/$defs/projection_policy"},
                    "state": {"$ref": "#/$defs/state"},
                    "memory": {
                        "type": "array",
                        "items": {"$ref": "#/$defs/memory_event"},
                    },
                },
                "additionalProperties": False,
            },
            "manifest_record": {
                "type": "object",
                "required": [
                    "format",
                    "record_type",
                    "source_binding",
                    "projection_policy",
                ],
                "properties": {
                    "format": {"const": "ptsip-context-record/v1"},
                    "record_type": {"const": "manifest"},
                    "source_binding": {"$ref": "#/$defs/source_binding"},
                    "projection_policy": {"$ref": "#/$defs/projection_policy"},
                },
                "additionalProperties": False,
            },
            "state_record": {
                "type": "object",
                "required": ["format", "record_type", "payload"],
                "properties": {
                    "format": {"const": "ptsip-context-record/v1"},
                    "record_type": {"const": "state"},
                    "payload": {"$ref": "#/$defs/state"},
                },
                "additionalProperties": False,
            },
            "memory_record": {
                "type": "object",
                "required": ["format", "record_type", "ordinal", "payload"],
                "properties": {
                    "format": {"const": "ptsip-context-record/v1"},
                    "record_type": {"const": "memory"},
                    "ordinal": {"type": "integer", "minimum": 0},
                    "payload": {"$ref": "#/$defs/memory_event"},
                },
                "additionalProperties": False,
            },
            "context_record": {
                "oneOf": [
                    {"$ref": "#/$defs/manifest_record"},
                    {"$ref": "#/$defs/state_record"},
                    {"$ref": "#/$defs/memory_record"},
                ]
            },
        },
        "oneOf": [
            {"$ref": "#/$defs/context_bundle"},
            {"$ref": "#/$defs/context_record"},
        ],
    }


def _bundle_and_records(
    source: Mapping[str, object],
    source_bytes: bytes,
) -> tuple[dict[str, object], list[dict[str, object]]]:
    binding = {
        "path": SOURCE_PATH.as_posix(),
        "sha256": _sha256(source_bytes),
    }
    policy = dict(_mapping(source["projection_policy"], "projection_policy"))
    state = dict(_mapping(source["state"], "state"))
    memory = [
        dict(_mapping(item, f"memory[{index}]"))
        for index, item in enumerate(source["memory"])
    ]
    bundle = {
        "format": "ptsip-context/v1",
        "source_binding": binding,
        "projection_policy": policy,
        "state": state,
        "memory": memory,
    }
    records: list[dict[str, object]] = [
        {
            "format": "ptsip-context-record/v1",
            "record_type": "manifest",
            "source_binding": binding,
            "projection_policy": policy,
        },
        {
            "format": "ptsip-context-record/v1",
            "record_type": "state",
            "payload": state,
        },
    ]
    records.extend(
        {
            "format": "ptsip-context-record/v1",
            "record_type": "memory",
            "ordinal": index,
            "payload": event,
        }
        for index, event in enumerate(memory)
    )
    return bundle, records


def decode_jsonl_records(records: Sequence[Mapping[str, object]]) -> dict[str, object]:
    if len(records) < 2:
        raise ContextProjectionError("context.jsonl must contain manifest and state records")
    manifest = _mapping(records[0], "jsonl manifest")
    state_record = _mapping(records[1], "jsonl state")
    if manifest.get("record_type") != "manifest":
        raise ContextProjectionError("context.jsonl first record must be manifest")
    if state_record.get("record_type") != "state":
        raise ContextProjectionError("context.jsonl second record must be state")

    memory: list[dict[str, object]] = []
    for expected_ordinal, raw in enumerate(records[2:]):
        record = _mapping(raw, f"jsonl memory record {expected_ordinal}")
        if record.get("record_type") != "memory":
            raise ContextProjectionError("context.jsonl contains non-memory trailing record")
        if record.get("ordinal") != expected_ordinal:
            raise ContextProjectionError("context.jsonl memory ordinals must be contiguous")
        memory.append(dict(_mapping(record.get("payload"), "jsonl memory payload")))

    return {
        "format": "ptsip-context/v1",
        "source_binding": dict(_mapping(manifest.get("source_binding"), "source binding")),
        "projection_policy": dict(
            _mapping(manifest.get("projection_policy"), "projection policy")
        ),
        "state": dict(_mapping(state_record.get("payload"), "state payload")),
        "memory": memory,
    }


def render_projections(source: object) -> ContextProjectionSet:
    validated = _validate_source(source)
    canonical_source = _json_bytes(validated)
    bundle, records = _bundle_and_records(validated, canonical_source)
    schema = _projection_schema()

    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema)
    validator.validate(bundle)
    for record in records:
        validator.validate(record)

    if decode_jsonl_records(records) != bundle:
        raise ContextProjectionError(
            "JSON and JSONL projections are not semantically equivalent"
        )

    return ContextProjectionSet(
        source=canonical_source,
        json_projection=_json_bytes(bundle),
        jsonl_projection=_jsonl_bytes(records),
        schema_projection=_json_bytes(schema),
    )


def _load_json(path: Path) -> object:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ContextProjectionError(f"unable to load JSON {path}: {exc}") from exc


def _write_all(root: Path, projections: ContextProjectionSet) -> None:
    for relative, payload in projections.as_paths().items():
        destination = root / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(payload)


def sync_context(root: str | Path = ".") -> ContextProjectionSet:
    base = Path(root).resolve()
    projections = render_projections(_load_json(base / SOURCE_PATH))
    _write_all(base, projections)
    return projections


def write_context(input_path: str | Path, root: str | Path = ".") -> ContextProjectionSet:
    base = Path(root).resolve()
    input_file = Path(input_path)
    if not input_file.is_absolute():
        input_file = (base / input_file).resolve()
    projections = render_projections(_load_json(input_file))
    _write_all(base, projections)
    return projections


def check_context(root: str | Path = ".") -> ContextProjectionSet:
    base = Path(root).resolve()
    projections = render_projections(_load_json(base / SOURCE_PATH))

    drift: list[str] = []
    for relative, expected in projections.as_paths().items():
        path = base / relative
        if not path.is_file():
            drift.append(f"{relative.as_posix()}: MISSING")
        elif path.read_bytes() != expected:
            drift.append(f"{relative.as_posix()}: DRIFT")
    if drift:
        raise ContextProjectionError(
            "context projection check failed: " + "; ".join(drift)
        )

    json_bundle = _load_json(base / JSON_PATH)
    jsonl_records = [
        json.loads(line)
        for line in (base / JSONL_PATH).read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]
    if json_bundle != decode_jsonl_records(jsonl_records):
        raise ContextProjectionError(
            "stored JSON and JSONL projections are not semantically equivalent"
        )

    schema = _load_json(base / SCHEMA_PATH)
    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema)
    validator.validate(json_bundle)
    for record in jsonl_records:
        validator.validate(record)
    return projections


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Generate and verify provider-neutral PTSIP context projections."
    )
    parser.add_argument("--root", default=".", help="Repository root.")
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser(
        "sync",
        help="Regenerate all projections from the canonical semantic source.",
    )
    subparsers.add_parser(
        "check",
        help="Fail if a projection is missing, stale, or semantically inconsistent.",
    )
    write_parser = subparsers.add_parser(
        "write",
        help="Write one semantic JSON input and generate every mandatory projection.",
    )
    write_parser.add_argument("--input", required=True, help="Semantic source JSON input.")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "sync":
            sync_context(args.root)
        elif args.command == "write":
            write_context(args.input, args.root)
        else:
            check_context(args.root)
    except ContextProjectionError as exc:
        print(json.dumps({"status": "FAIL", "error": str(exc)}, sort_keys=True))
        return 2

    print(
        json.dumps(
            {
                "status": "PASS",
                "command": args.command,
                "source": SOURCE_PATH.as_posix(),
                "projections": [
                    JSON_PATH.as_posix(),
                    JSONL_PATH.as_posix(),
                    SCHEMA_PATH.as_posix(),
                ],
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
