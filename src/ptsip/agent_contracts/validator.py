from __future__ import annotations

import json
from importlib.resources import files
from pathlib import PurePosixPath
from typing import Any

import yaml
from jsonschema import Draft202012Validator


class AgentContractValidationError(RuntimeError):
    """Raised when the embedded agent contract plane is inconsistent."""


def _root():
    return files("ptsip").joinpath("agent_contracts")


def _safe_ref(value: object) -> str:
    if not isinstance(value, str) or not value:
        raise AgentContractValidationError("Agent contract ref must be a non-empty string.")
    path = PurePosixPath(value)
    if path.is_absolute() or ".." in path.parts:
        raise AgentContractValidationError(f"Unsafe agent contract ref: {value!r}")
    return value


def _yaml(ref: str) -> dict[str, Any]:
    resource = _root().joinpath(*PurePosixPath(_safe_ref(ref)).parts)
    if not resource.is_file():
        raise AgentContractValidationError(f"Missing agent contract resource: {ref}")
    payload = yaml.safe_load(resource.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise AgentContractValidationError(f"Agent contract resource must be a mapping: {ref}")
    return payload


def _json(ref: str) -> dict[str, Any]:
    resource = _root().joinpath(*PurePosixPath(_safe_ref(ref)).parts)
    if not resource.is_file():
        raise AgentContractValidationError(f"Missing agent contract schema: {ref}")
    payload = json.loads(resource.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise AgentContractValidationError(f"Agent contract schema must be an object: {ref}")
    return payload


def _validate(payload: dict[str, Any], schema: dict[str, Any], ref: str) -> None:
    errors = sorted(
        Draft202012Validator(schema).iter_errors(payload),
        key=lambda error: list(error.path),
    )
    if errors:
        detail = "; ".join(error.message for error in errors[:5])
        raise AgentContractValidationError(f"Invalid agent contract {ref}: {detail}")


def validate_agent_contract_plane() -> dict[str, int]:
    index = _yaml("index.yaml")
    schemas = index.get("schemas")
    if not isinstance(schemas, dict):
        raise AgentContractValidationError("index.yaml schemas must be a mapping.")

    _validate(index, _json(_safe_ref(schemas.get("index"))), "index.yaml")

    schema_by_kind = {
        "specs": _json(_safe_ref(schemas.get("spec"))),
        "operations": _json(_safe_ref(schemas.get("operation"))),
        "vocabularies": _json(_safe_ref(schemas.get("vocabulary"))),
    }

    counts: dict[str, int] = {}
    for kind, schema in schema_by_kind.items():
        entries = index[kind]
        seen_ids: set[str] = set()
        for entry in entries:
            entry_id = entry["id"]
            if entry_id in seen_ids:
                raise AgentContractValidationError(f"Duplicate {kind} id: {entry_id}")
            seen_ids.add(entry_id)
            ref = _safe_ref(entry["ref"])
            _validate(_yaml(ref), schema, ref)
        counts[kind] = len(entries)

    binding_ref = _safe_ref(index["binding"]["current_ref"])
    binding = _yaml(binding_ref)
    _validate(binding, _json(_safe_ref(schemas.get("binding"))), binding_ref)

    declared_refs = {
        entry["ref"]
        for kind in ("specs", "operations", "vocabularies")
        for entry in index[kind]
    }
    bound_refs = (
        set(binding["active_specs"])
        | set(binding["active_operations"])
        | set(binding["active_vocabularies"])
    )
    if declared_refs != bound_refs:
        missing = sorted(declared_refs - bound_refs)
        extra = sorted(bound_refs - declared_refs)
        raise AgentContractValidationError(
            f"Current binding/index mismatch: missing={missing}, extra={extra}"
        )

    counts["bindings"] = 1
    return counts
