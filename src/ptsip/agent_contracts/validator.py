from __future__ import annotations

import json
from importlib.resources import files
from pathlib import PurePosixPath
from typing import Any, Iterable

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


def agent_contract_resource_size(ref: str) -> int:
    resource = _root().joinpath(*PurePosixPath(_safe_ref(ref)).parts)
    if not resource.is_file():
        raise AgentContractValidationError(f"Missing agent contract resource: {ref}")
    return len(resource.read_bytes())


def load_agent_contract_yaml(ref: str) -> dict[str, Any]:
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


def _assert_markdown_free(ref: str, payload: object) -> None:
    markdown_refs = [value for value in _strings(payload) if ".md" in value.lower()]
    if markdown_refs:
        raise AgentContractValidationError(
            f"Active machine contract {ref} contains Markdown dependency: {markdown_refs[:3]}"
        )


def validate_agent_contract_plane() -> dict[str, int]:
    index = load_agent_contract_yaml("index.yaml")
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
    loaded: dict[str, dict[str, dict[str, Any]]] = {}
    refs_by_kind: dict[str, set[str]] = {}
    for kind, schema in schema_by_kind.items():
        entries = index[kind]
        seen_ids: set[str] = set()
        by_ref: dict[str, dict[str, Any]] = {}
        for entry in entries:
            entry_id = entry["id"]
            if entry_id in seen_ids:
                raise AgentContractValidationError(f"Duplicate {kind} id: {entry_id}")
            seen_ids.add(entry_id)
            ref = _safe_ref(entry["ref"])
            payload = load_agent_contract_yaml(ref)
            _validate(payload, schema, ref)
            by_ref[ref] = payload
        loaded[kind] = by_ref
        refs_by_kind[kind] = set(by_ref)
        counts[kind] = len(entries)

    binding_ref = _safe_ref(index["binding"]["current_ref"])
    binding = load_agent_contract_yaml(binding_ref)
    _validate(binding, _json(_safe_ref(schemas.get("binding"))), binding_ref)

    declared_refs = set().union(*refs_by_kind.values())
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

    for op_ref, operation in loaded["operations"].items():
        spec_refs = set(operation["spec_refs"])
        vocabulary_refs = set(operation["vocabulary_refs"])
        if not spec_refs <= refs_by_kind["specs"]:
            raise AgentContractValidationError(
                f"Operation {op_ref} references undeclared specs: {sorted(spec_refs - refs_by_kind['specs'])}"
            )
        if not vocabulary_refs <= refs_by_kind["vocabularies"]:
            raise AgentContractValidationError(
                f"Operation {op_ref} references undeclared vocabularies: "
                f"{sorted(vocabulary_refs - refs_by_kind['vocabularies'])}"
            )

        unresolved: list[str] = []
        ambiguous: list[str] = []
        for rule_id in operation["policy_refs"]:
            owners = [
                ref for ref in operation["spec_refs"]
                if rule_id in loaded["specs"][ref]["normative_rule_refs"]
            ]
            if not owners:
                unresolved.append(rule_id)
            elif len(owners) > 1:
                ambiguous.append(rule_id)
        if unresolved or ambiguous:
            raise AgentContractValidationError(
                f"Operation {op_ref} policy ownership invalid: "
                f"unresolved={unresolved}, ambiguous={ambiguous}"
            )

    complete = (
        index["migration"]["legacy_markdown_replacement"] == "COMPLETE"
        and binding["coverage"]["legacy_markdown_replacement"] == "COMPLETE"
    )
    if complete:
        if index["contract_set"]["status"] != "CURRENT" or binding["status"] != "CURRENT":
            raise AgentContractValidationError(
                "Complete machine contract coverage requires CURRENT index and binding."
            )
        if binding["activation"]["status"] != "ACTIVE":
            raise AgentContractValidationError(
                "Complete machine contract coverage requires explicit active authority binding."
            )
        if binding["loading"]["strategy"] != "OPERATION_SCOPED":
            raise AgentContractValidationError("Active agent loading must be OPERATION_SCOPED.")
        if binding["loading"]["markdown_dependency"] != "FORBIDDEN":
            raise AgentContractValidationError("Active agent loading must forbid Markdown dependency.")

        _assert_markdown_free("index.yaml", index)
        _assert_markdown_free(binding_ref, binding)
        for kind in ("specs", "operations", "vocabularies"):
            for ref, payload in loaded[kind].items():
                if payload["status"] != "CURRENT":
                    raise AgentContractValidationError(
                        f"Active machine contract must be CURRENT: {ref}"
                    )
                _assert_markdown_free(ref, payload)

    counts["bindings"] = 1
    return counts
