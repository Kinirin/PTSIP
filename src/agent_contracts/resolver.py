from __future__ import annotations

import argparse
import json
from importlib.resources import files
from pathlib import PurePosixPath
from typing import Any, Mapping

import yaml


class AgentContractResolutionError(RuntimeError):
    pass


def _root():
    return files("agent_contracts")


def _safe_ref(value: object) -> str:
    if not isinstance(value, str) or not value:
        raise AgentContractResolutionError("contract ref must be a non-empty string")
    path = PurePosixPath(value)
    if path.is_absolute() or ".." in path.parts:
        raise AgentContractResolutionError(f"unsafe contract ref: {value!r}")
    return value


def _yaml(ref: str) -> dict[str, Any]:
    resource = _root().joinpath(*PurePosixPath(_safe_ref(ref)).parts)
    if not resource.is_file():
        raise AgentContractResolutionError(f"missing contract resource: {ref}")
    value = yaml.safe_load(resource.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise AgentContractResolutionError(f"contract resource must be a mapping: {ref}")
    return value


def _json(ref: str) -> dict[str, Any]:
    resource = _root().joinpath(*PurePosixPath(_safe_ref(ref)).parts)
    if not resource.is_file():
        raise AgentContractResolutionError(f"missing contract JSON resource: {ref}")
    value = json.loads(resource.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise AgentContractResolutionError(f"contract JSON must be an object: {ref}")
    return value


def _entry_map(index: Mapping[str, object], field: str) -> dict[str, str]:
    raw = index.get(field)
    if not isinstance(raw, list):
        raise AgentContractResolutionError(f"index field {field!r} must be a list")
    result: dict[str, str] = {}
    for item in raw:
        if not isinstance(item, Mapping):
            raise AgentContractResolutionError(f"index {field!r} contains a non-mapping entry")
        identity = item.get("id")
        ref = item.get("ref")
        if not isinstance(identity, str) or not isinstance(ref, str):
            raise AgentContractResolutionError(f"index {field!r} entry is invalid")
        if identity in result:
            raise AgentContractResolutionError(f"duplicate indexed identity: {identity}")
        result[identity] = ref
    return result


def resolve_operation(operation_id: str) -> dict[str, object]:
    index = _yaml("index.yaml")
    operations = _entry_map(index, "operations")
    operation_ref = operations.get(operation_id)
    if operation_ref is None:
        raise AgentContractResolutionError(f"unknown operation: {operation_id}")
    operation = _yaml(operation_ref)

    specs = _entry_map(index, "specs")
    rule_map: dict[str, dict[str, object]] = {}
    rule_owner: dict[str, str] = {}
    for spec_id, ref in specs.items():
        payload = _yaml(ref)
        for raw in payload.get("rules", []):
            if not isinstance(raw, dict) or not isinstance(raw.get("rule_id"), str):
                raise AgentContractResolutionError(f"invalid rule in {ref}")
            rule_id = raw["rule_id"]
            if rule_id in rule_map:
                raise AgentContractResolutionError(f"duplicate rule identity: {rule_id}")
            rule_map[rule_id] = raw
            rule_owner[rule_id] = spec_id

    selected_rules: list[dict[str, object]] = []
    selected_specs: dict[str, dict[str, object]] = {}
    for rule_id in operation.get("rule_refs", []):
        if not isinstance(rule_id, str) or rule_id not in rule_map:
            raise AgentContractResolutionError(f"unresolved rule ref: {rule_id!r}")
        selected_rules.append(rule_map[rule_id])
        spec_id = rule_owner[rule_id]
        if spec_id not in selected_specs:
            selected_specs[spec_id] = {"id": spec_id, "rule_ids": []}
        selected_specs[spec_id]["rule_ids"].append(rule_id)

    actions_index = _entry_map(index, "actions")
    conditions_index = _entry_map(index, "conditions")
    gates_index = _entry_map(index, "gates")
    io_index = _entry_map(index, "io_schemas")
    vocab_index = _entry_map(index, "vocabularies")

    action_ids: set[str] = set()
    condition_ids: set[str] = set()
    gate_ids: set[str] = set()
    for check in operation.get("preconditions", []):
        if isinstance(check, Mapping) and isinstance(check.get("condition_ref"), str):
            condition_ids.add(check["condition_ref"])
    for step in operation.get("steps", []):
        if not isinstance(step, Mapping):
            continue
        if step.get("type") == "ACTION":
            if isinstance(step.get("action_ref"), str):
                action_ids.add(step["action_ref"])
            if isinstance(step.get("mutation_gate_ref"), str):
                gate_ids.add(step["mutation_gate_ref"])
        elif step.get("type") == "CONDITION" and isinstance(step.get("condition_ref"), str):
            condition_ids.add(step["condition_ref"])

    actions: dict[str, dict[str, Any]] = {}
    gates: dict[str, dict[str, Any]] = {}
    io_ids: set[str] = {
        str(operation["input_schema_ref"]),
        str(operation["output_schema_ref"]),
    }
    for action_id in sorted(action_ids):
        ref = actions_index.get(action_id)
        if ref is None:
            raise AgentContractResolutionError(f"unresolved action: {action_id}")
        payload = _yaml(ref)
        actions[action_id] = payload
        io_ids.update((str(payload["input_schema_ref"]), str(payload["output_schema_ref"])))
        for field in ("preconditions", "postconditions"):
            for check in payload.get(field, []):
                if isinstance(check, Mapping) and isinstance(check.get("condition_ref"), str):
                    condition_ids.add(check["condition_ref"])

    for gate_id in sorted(gate_ids):
        ref = gates_index.get(gate_id)
        if ref is None:
            raise AgentContractResolutionError(f"unresolved gate: {gate_id}")
        payload = _yaml(ref)
        gates[gate_id] = payload
        io_ids.add(str(payload["input_schema_ref"]))
        for requirement in payload.get("requirements", []):
            if isinstance(requirement, Mapping) and isinstance(requirement.get("condition_ref"), str):
                condition_ids.add(requirement["condition_ref"])

    conditions: dict[str, dict[str, Any]] = {}
    pending = list(sorted(condition_ids))
    while pending:
        condition_id = pending.pop(0)
        if condition_id in conditions:
            continue
        ref = conditions_index.get(condition_id)
        if ref is None:
            raise AgentContractResolutionError(f"unresolved condition: {condition_id}")
        payload = _yaml(ref)
        conditions[condition_id] = payload
        io_ids.add(str(payload["input_schema_ref"]))
        evaluation = payload.get("evaluation")
        if isinstance(evaluation, Mapping):
            child_refs = evaluation.get("condition_refs")
            if isinstance(child_refs, list):
                pending.extend(str(x) for x in child_refs)
            child_ref = evaluation.get("condition_ref")
            if isinstance(child_ref, str):
                pending.append(child_ref)

    io_schemas: dict[str, dict[str, Any]] = {}
    for io_id in sorted(io_ids):
        ref = io_index.get(io_id)
        if ref is None:
            raise AgentContractResolutionError(f"unresolved I/O schema: {io_id}")
        io_schemas[io_id] = _json(ref)

    vocabulary_refs = operation.get("vocabulary_refs")
    if not isinstance(vocabulary_refs, list) or not vocabulary_refs:
        raise AgentContractResolutionError("operation vocabulary_refs must be a non-empty list")
    vocabularies: dict[str, dict[str, Any]] = {}
    for vocab_id in vocabulary_refs:
        if not isinstance(vocab_id, str):
            raise AgentContractResolutionError("invalid vocabulary ref")
        ref = vocab_index.get(vocab_id)
        if ref is None:
            raise AgentContractResolutionError(f"unresolved vocabulary: {vocab_id}")
        vocabularies[vocab_id] = _yaml(ref)

    binding_ref = index.get("binding", {}).get("current_ref") if isinstance(index.get("binding"), Mapping) else None
    binding = _yaml(str(binding_ref))
    selected_action_bindings = [
        item for item in binding.get("action_bindings", [])
        if isinstance(item, Mapping) and item.get("action_id") in action_ids
    ]
    selected_condition_bindings = [
        item for item in binding.get("condition_evaluator_bindings", [])
        if isinstance(item, Mapping) and item.get("condition_id") in conditions
    ]
    selected_entrypoints = [
        item for item in binding.get("operation_entrypoints", [])
        if isinstance(item, Mapping) and item.get("operation_id") == operation_id
    ]

    return {
        "schema_version": "ptsip-agent-operation-resolution/v1",
        "operation_id": operation_id,
        "operation": operation,
        "selected_specs": list(selected_specs.values()),
        "rules": selected_rules,
        "actions": actions,
        "conditions": conditions,
        "gates": gates,
        "io_schemas": io_schemas,
        "vocabularies": vocabularies,
        "implementation_bindings": {
            "actions": selected_action_bindings,
            "conditions": selected_condition_bindings,
            "entrypoints": selected_entrypoints,
        },
        "markdown_dependency": False,
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Resolve one bounded PTSIP Agent Contract operation graph.")
    parser.add_argument("operation_id")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        result = resolve_operation(args.operation_id)
    except (AgentContractResolutionError, OSError, ValueError, yaml.YAMLError, json.JSONDecodeError) as exc:
        print(f"agent-contract-resolver: {exc}")
        return 2
    if args.json:
        print(json.dumps(result, ensure_ascii=True, indent=2))
    else:
        print(
            f"{result['operation_id']}: "
            f"{len(result['rules'])} rules, "
            f"{len(result['actions'])} actions, "
            f"{len(result['conditions'])} conditions, "
            f"{len(result['gates'])} gates, "
            f"{len(result['io_schemas'])} io schemas, "
            f"{len(result['vocabularies'])} vocabularies"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
