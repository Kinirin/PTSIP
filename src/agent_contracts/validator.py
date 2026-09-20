from __future__ import annotations

import json
from importlib.resources import files
from pathlib import PurePosixPath
from typing import Any, Iterable

import yaml
from jsonschema import Draft202012Validator


class AgentContractValidationError(RuntimeError):
    """Raised when the embedded Agent Contract Plane is inconsistent."""


def _ptsip_root():
    return files("ptsip")


def _root():
    return _ptsip_root().joinpath("agent_contracts")


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


def _strings(value: object) -> Iterable[str]:
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for item in value.values():
            yield from _strings(item)
    elif isinstance(value, list):
        for item in value:
            yield from _strings(item)


def _reject_markdown_dependency(payload: dict[str, Any], ref: str) -> None:
    for value in _strings(payload):
        lowered = value.casefold()
        if lowered.endswith(".md") or ".md#" in lowered:
            raise AgentContractValidationError(
                f"Markdown normative dependency is forbidden in {ref}: {value!r}"
            )


def _validate_external_machine_contract(binding: dict[str, Any]) -> None:
    for contract in binding["external_machine_contracts"]:
        resource_ref = _safe_ref(contract["resource"])
        resource = _ptsip_root().joinpath(*PurePosixPath(resource_ref).parts)
        if not resource.is_file():
            raise AgentContractValidationError(
                f"Missing external machine contract resource: {resource_ref}"
            )
        if contract["contract_id"] == "PTSIP_PROJECT_PROFILE_CONTRACT_IDENTITY":
            payload = yaml.safe_load(resource.read_text(encoding="utf-8"))
            if not isinstance(payload, dict) or payload.get("current") != contract["version"]:
                raise AgentContractValidationError(
                    "Project Profile contract binding does not match embedded current identity."
                )


def validate_agent_contract_plane() -> dict[str, int]:
    index = _yaml("index.yaml")
    schemas = index.get("schemas")
    if not isinstance(schemas, dict):
        raise AgentContractValidationError("index.yaml schemas must be a mapping.")

    _validate(index, _json(_safe_ref(schemas.get("index"))), "index.yaml")
    _reject_markdown_dependency(index, "index.yaml")

    schema_by_kind = {
        "specs": _json(_safe_ref(schemas.get("spec"))),
        "operations": _json(_safe_ref(schemas.get("operation"))),
        "vocabularies": _json(_safe_ref(schemas.get("vocabulary"))),
    }

    payloads: dict[str, dict[str, dict[str, Any]]] = {
        "specs": {},
        "operations": {},
        "vocabularies": {},
    }
    counts: dict[str, int] = {}

    for kind, schema in schema_by_kind.items():
        for entry in index[kind]:
            entry_id = entry["id"]
            if entry_id in payloads[kind]:
                raise AgentContractValidationError(f"Duplicate {kind} id: {entry_id}")
            ref = _safe_ref(entry["ref"])
            payload = _yaml(ref)
            _validate(payload, schema, ref)
            _reject_markdown_dependency(payload, ref)

            identity_key = {
                "specs": "id",
                "operations": "operation_id",
                "vocabularies": "vocabulary_id",
            }[kind]
            if payload[identity_key] != entry_id:
                raise AgentContractValidationError(
                    f"{kind} index identity mismatch for {ref}: "
                    f"{entry_id!r} != {payload[identity_key]!r}"
                )
            payloads[kind][entry_id] = payload
        counts[kind] = len(payloads[kind])

    rule_ids: set[str] = set()
    for spec in payloads["specs"].values():
        for rule in spec["rules"]:
            rule_id = rule["rule_id"]
            if rule_id in rule_ids:
                raise AgentContractValidationError(f"Duplicate rule id: {rule_id}")
            rule_ids.add(rule_id)

    outcome_vocab = payloads["vocabularies"]["PTSIP-VOCAB-OUTCOMES"]
    outcome_ids = {entry["id"] for entry in outcome_vocab["entries"]}

    for operation in payloads["operations"].values():
        unresolved_rules = sorted(set(operation["rule_refs"]) - rule_ids)
        if unresolved_rules:
            raise AgentContractValidationError(
                f"Operation {operation['operation_id']} has unresolved rule refs: "
                f"{unresolved_rules}"
            )
        unresolved_outcomes = sorted(set(operation["outcomes"]) - outcome_ids)
        if unresolved_outcomes:
            raise AgentContractValidationError(
                f"Operation {operation['operation_id']} has undefined outcomes: "
                f"{unresolved_outcomes}"
            )
        for step in operation["steps"]:
            unresolved_failures = sorted(
                set(step.get("fail_closed_outcomes", [])) - outcome_ids
            )
            if unresolved_failures:
                raise AgentContractValidationError(
                    f"Operation {operation['operation_id']} step {step['step_id']} "
                    f"has undefined fail-closed outcomes: {unresolved_failures}"
                )

    binding_ref = _safe_ref(index["binding"]["current_ref"])
    binding = _yaml(binding_ref)
    _validate(binding, _json(_safe_ref(schemas.get("binding"))), binding_ref)
    _reject_markdown_dependency(binding, binding_ref)

    indexed_ids = {
        "active_specs": set(payloads["specs"]),
        "active_operations": set(payloads["operations"]),
        "active_vocabularies": set(payloads["vocabularies"]),
    }
    for field, expected in indexed_ids.items():
        actual = set(binding[field])
        if actual != expected:
            raise AgentContractValidationError(
                f"Binding {field} mismatch: missing={sorted(expected - actual)}, "
                f"extra={sorted(actual - expected)}"
            )

    implementation_ids = {
        item["operation_id"] for item in binding["implementation_bindings"]
    }
    if implementation_ids != indexed_ids["active_operations"]:
        raise AgentContractValidationError(
            "Implementation bindings must resolve every active operation exactly once."
        )

    _validate_external_machine_contract(binding)

    counts["rules"] = len(rule_ids)
    counts["bindings"] = 1
    return counts
