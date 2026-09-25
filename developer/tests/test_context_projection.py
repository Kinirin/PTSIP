from __future__ import annotations

import json
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator

from developer.automation.context_projection import (
    JSON_PATH,
    JSONL_PATH,
    SCHEMA_PATH,
    SOURCE_PATH,
    ContextProjectionError,
    check_context,
    decode_jsonl_records,
    write_context,
)


ROOT = Path(__file__).resolve().parents[2]


def _source() -> dict[str, object]:
    payload = json.loads((ROOT / SOURCE_PATH).read_text(encoding="utf-8"))
    assert isinstance(payload, dict)
    return payload


def _jsonl(path: Path) -> list[dict[str, object]]:
    return [
        json.loads(line)
        for line in path.read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]


def test_repository_context_plane_is_in_sync() -> None:
    check_context(ROOT)


def test_json_and_jsonl_projections_are_semantically_equivalent() -> None:
    bundle = json.loads((ROOT / JSON_PATH).read_text(encoding="utf-8"))
    records = _jsonl(ROOT / JSONL_PATH)
    assert decode_jsonl_records(records) == bundle


def test_schema_accepts_json_bundle_and_every_jsonl_record() -> None:
    schema = json.loads((ROOT / SCHEMA_PATH).read_text(encoding="utf-8"))
    validator = Draft202012Validator(schema)
    validator.validate(json.loads((ROOT / JSON_PATH).read_text(encoding="utf-8")))
    for record in _jsonl(ROOT / JSONL_PATH):
        validator.validate(record)


def test_context_policy_is_provider_neutral_and_extensible() -> None:
    policy = _source()["projection_policy"]
    assert policy == {
        "authority": "CANONICAL_SEMANTIC_MODEL",
        "extension_policy": "EXTENSIBLE",
        "generated_only": True,
        "mandatory_formats": ["json", "jsonl", "schema-json"],
        "provider_scope": ["OPENAI", "ANTHROPIC", "GOOGLE", "XAI"],
        "semantic_equivalence": "REQUIRED",
    }


def test_single_write_generates_all_mandatory_projections(tmp_path: Path) -> None:
    input_path = tmp_path / "semantic-input.json"
    input_path.write_text(
        json.dumps(_source(), ensure_ascii=False),
        encoding="utf-8",
    )

    write_context(input_path, tmp_path)

    assert (tmp_path / SOURCE_PATH).is_file()
    assert (tmp_path / JSON_PATH).is_file()
    assert (tmp_path / JSONL_PATH).is_file()
    assert (tmp_path / SCHEMA_PATH).is_file()
    check_context(tmp_path)


def test_projection_drift_fails_closed(tmp_path: Path) -> None:
    input_path = tmp_path / "semantic-input.json"
    input_path.write_text(
        json.dumps(_source(), ensure_ascii=False),
        encoding="utf-8",
    )
    write_context(input_path, tmp_path)
    (tmp_path / JSON_PATH).write_text("{}\n", encoding="utf-8")

    with pytest.raises(ContextProjectionError, match="DRIFT"):
        check_context(tmp_path)
