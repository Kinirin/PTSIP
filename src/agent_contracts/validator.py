from __future__ import annotations

import importlib
import json
import re
from importlib.resources import files
from pathlib import PurePosixPath
from typing import Any, Iterable

import yaml
from jsonschema import Draft202012Validator


class AgentContractValidationError(RuntimeError):
    """Raised when the embedded Agent Contract Plane is inconsistent."""


_SOURCE_REF_RE = re.compile(r"^\$(input|step:([A-Za-z0-9_-]+))#(.*)$")


def _root():
    return files("agent_contracts")


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
        raise AgentContractValidationError(f"Missing agent contract JSON resource: {ref}")
    payload = json.loads(resource.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise AgentContractValidationError(f"Agent contract JSON resource must be an object: {ref}")
    return payload


def _validate(payload: dict[str, Any], schema: dict[str, Any], ref: str) -> None:
    errors = sorted(
        Draft202012Validator(schema).iter_errors(payload),
        key=lambda error: list(error.path),
    )
    if errors:
        detail = "; ".join(error.message for error in errors[:8])
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


def _schema_at_pointer(schema: dict[str, Any], pointer: str, context: str) -> dict[str, Any]:
    if pointer == "":
        return schema
    if not pointer.startswith("/"):
        raise AgentContractValidationError(f"Invalid JSON pointer in {context}: {pointer!r}")
    current: object = schema
    for raw in pointer[1:].split("/"):
        token = raw.replace("~1", "/").replace("~0", "~")
        if not isinstance(current, dict):
            raise AgentContractValidationError(
                f"JSON pointer {pointer!r} cannot be resolved in {context}."
            )
        properties = current.get("properties")
        if not isinstance(properties, dict) or token not in properties:
            raise AgentContractValidationError(
                f"JSON pointer {pointer!r} is not declared by {context}."
            )
        current = properties[token]
    if not isinstance(current, dict):
        raise AgentContractValidationError(
            f"JSON pointer {pointer!r} does not resolve to a schema object in {context}."
        )
    return current


def _compatible_schema(source: dict[str, Any], target: dict[str, Any]) -> bool:
    source_type = source.get("type")
    target_type = target.get("type")
    if source_type is not None and target_type is not None and source_type != target_type:
        return False
    source_enum = source.get("enum")
    target_enum = target.get("enum")
    if isinstance(source_enum, list) and isinstance(target_enum, list):
        if not set(source_enum).issubset(set(target_enum)):
            return False
    return True


def _parse_source_ref(value: str, context: str) -> tuple[str, str | None, str]:
    match = _SOURCE_REF_RE.fullmatch(value)
    if match is None:
        raise AgentContractValidationError(f"Invalid data source ref in {context}: {value!r}")
    source_kind = match.group(1)
    step_id = match.group(2)
    pointer = match.group(3)
    if pointer and not pointer.startswith("/"):
        raise AgentContractValidationError(
            f"Source pointer must be empty or a JSON pointer in {context}: {value!r}"
        )
    return source_kind, step_id, pointer


def _resolve_callable(locator: str) -> None:
    module_name, sep, attribute_path = locator.partition(":")
    if not sep or not module_name or not attribute_path:
        raise AgentContractValidationError(f"Invalid python callable locator: {locator!r}")
    try:
        target: object = importlib.import_module(module_name)
    except Exception as exc:
        raise AgentContractValidationError(
            f"Unable to import action binding module {module_name!r}: {exc}"
        ) from exc
    try:
        for part in attribute_path.split("."):
            target = getattr(target, part)
    except AttributeError as exc:
        raise AgentContractValidationError(
            f"Unable to resolve python callable {locator!r}."
        ) from exc
    if not callable(target):
        raise AgentContractValidationError(f"Binding target is not callable: {locator!r}")


def _validate_external_machine_contract(binding: dict[str, Any]) -> None:
    for contract in binding["external_machine_contracts"]:
        package = contract["package"]
        resource_ref = _safe_ref(contract["resource"])
        try:
            package_root = files(package)
        except Exception as exc:
            raise AgentContractValidationError(
                f"Unable to resolve external machine-contract package {package!r}: {exc}"
            ) from exc
        resource = package_root.joinpath(*PurePosixPath(resource_ref).parts)
        if not resource.is_file():
            raise AgentContractValidationError(
                f"Missing external machine contract resource: {package}:{resource_ref}"
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

    kind_config = {
        "specs": ("spec", "id", False),
        "operations": ("operation", "operation_id", False),
        "actions": ("action", "action_id", False),
        "conditions": ("condition", "condition_id", False),
        "gates": ("gate", "gate_id", False),
        "io_schemas": ("io", "x-ptsip-io-id", True),
        "vocabularies": ("vocabulary", "vocabulary_id", False),
    }
    payloads: dict[str, dict[str, dict[str, Any]]] = {kind: {} for kind in kind_config}
    counts: dict[str, int] = {}

    for kind, (schema_key, identity_key, is_json) in kind_config.items():
        schema = _json(_safe_ref(schemas.get(schema_key)))
        for entry in index[kind]:
            entry_id = entry["id"]
            if entry_id in payloads[kind]:
                raise AgentContractValidationError(f"Duplicate {kind} id: {entry_id}")
            ref = _safe_ref(entry["ref"])
            payload = _json(ref) if is_json else _yaml(ref)
            _validate(payload, schema, ref)
            _reject_markdown_dependency(payload, ref)
            if payload[identity_key] != entry_id:
                raise AgentContractValidationError(
                    f"{kind} index identity mismatch for {ref}: "
                    f"{entry_id!r} != {payload[identity_key]!r}"
                )
            payloads[kind][entry_id] = payload
        counts[kind] = len(payloads[kind])

    semantic_ids: set[str] = set()
    for kind in ("specs", "operations", "actions", "conditions", "gates", "io_schemas", "vocabularies"):
        for item_id in payloads[kind]:
            if item_id in semantic_ids:
                raise AgentContractValidationError(f"Cross-contract identity collision: {item_id}")
            semantic_ids.add(item_id)

    rule_ids: set[str] = set()
    for spec in payloads["specs"].values():
        for rule in spec["rules"]:
            rule_id = rule["rule_id"]
            if rule_id in rule_ids or rule_id in semantic_ids:
                raise AgentContractValidationError(f"Duplicate or colliding rule id: {rule_id}")
            rule_ids.add(rule_id)

    outcome_vocab = payloads["vocabularies"]["PTSIP-VOCAB-OUTCOMES"]
    outcome_ids = {entry["id"] for entry in outcome_vocab["entries"]}
    io_payloads = payloads["io_schemas"]
    conditions = payloads["conditions"]
    actions = payloads["actions"]
    gates = payloads["gates"]

    def require_outcome(value: str, context: str) -> None:
        if value not in outcome_ids:
            raise AgentContractValidationError(
                f"Undefined outcome {value!r} referenced by {context}."
            )

    def require_io(value: str, context: str) -> dict[str, Any]:
        try:
            return io_payloads[value]
        except KeyError as exc:
            raise AgentContractValidationError(
                f"Unresolved I/O schema ref {value!r} in {context}."
            ) from exc

    for condition_id, condition in conditions.items():
        input_schema = require_io(condition["input_schema_ref"], condition_id)
        evaluation = condition["evaluation"]
        evaluation_type = evaluation["type"]
        if evaluation_type == "FIELD_COMPARE":
            field_schema = _schema_at_pointer(
                input_schema,
                evaluation["pointer"],
                f"{condition_id} FIELD_COMPARE",
            )
            if not Draft202012Validator(field_schema).is_valid(evaluation["expected"]):
                raise AgentContractValidationError(
                    f"Condition {condition_id} expected value does not satisfy "
                    f"the schema at {evaluation['pointer']}."
                )
        elif evaluation_type in {"ALL_OF", "ANY_OF"}:
            for child_ref in evaluation["condition_refs"]:
                child = conditions.get(child_ref)
                if child is None:
                    raise AgentContractValidationError(
                        f"Condition {condition_id} has unresolved child {child_ref}."
                    )
                if child["input_schema_ref"] != condition["input_schema_ref"]:
                    raise AgentContractValidationError(
                        f"Composite condition {condition_id} and child {child_ref} "
                        "must use the same input schema."
                    )
        elif evaluation_type == "NOT":
            child_ref = evaluation["condition_ref"]
            child = conditions.get(child_ref)
            if child is None:
                raise AgentContractValidationError(
                    f"Condition {condition_id} has unresolved child {child_ref}."
                )
            if child["input_schema_ref"] != condition["input_schema_ref"]:
                raise AgentContractValidationError(
                    f"NOT condition {condition_id} and child {child_ref} "
                    "must use the same input schema."
                )

    visiting: set[str] = set()
    visited: set[str] = set()

    def visit_condition(condition_id: str) -> None:
        if condition_id in visited:
            return
        if condition_id in visiting:
            raise AgentContractValidationError(
                f"Condition reference cycle detected at {condition_id}."
            )
        visiting.add(condition_id)
        evaluation = conditions[condition_id]["evaluation"]
        children: list[str] = []
        if evaluation["type"] in {"ALL_OF", "ANY_OF"}:
            children = list(evaluation["condition_refs"])
        elif evaluation["type"] == "NOT":
            children = [evaluation["condition_ref"]]
        for child in children:
            visit_condition(child)
        visiting.remove(condition_id)
        visited.add(condition_id)

    for condition_id in conditions:
        visit_condition(condition_id)

    for action_id, action in actions.items():
        require_io(action["input_schema_ref"], action_id)
        require_io(action["output_schema_ref"], action_id)
        declared_failures = set(action["failure_outcomes"])
        for phase, expected_schema in (
            ("preconditions", action["input_schema_ref"]),
            ("postconditions", action["output_schema_ref"]),
        ):
            for check in action[phase]:
                condition = conditions.get(check["condition_ref"])
                if condition is None:
                    raise AgentContractValidationError(
                        f"Action {action_id} has unresolved condition {check['condition_ref']}."
                    )
                if condition["input_schema_ref"] != expected_schema:
                    raise AgentContractValidationError(
                        f"Action {action_id} {phase} condition {check['condition_ref']} "
                        f"expects {condition['input_schema_ref']}, not {expected_schema}."
                    )
                for key in ("on_false", "on_unknown"):
                    require_outcome(check[key], f"{action_id}.{phase}")
                    if check[key] not in declared_failures:
                        raise AgentContractValidationError(
                            f"Action {action_id} must declare {check[key]} in failure_outcomes."
                        )
        for outcome in declared_failures:
            require_outcome(outcome, action_id)

    for gate_id, gate in gates.items():
        require_io(gate["input_schema_ref"], gate_id)
        for requirement in gate["requirements"]:
            condition = conditions.get(requirement["condition_ref"])
            if condition is None:
                raise AgentContractValidationError(
                    f"Gate {gate_id} has unresolved condition {requirement['condition_ref']}."
                )
            if condition["input_schema_ref"] != gate["input_schema_ref"]:
                raise AgentContractValidationError(
                    f"Gate {gate_id} condition {requirement['condition_ref']} "
                    "must consume the gate input schema."
                )
            require_outcome(requirement["on_false"], gate_id)
            require_outcome(requirement["on_unknown"], gate_id)

    def validate_operation(operation: dict[str, Any]) -> None:
        operation_id = operation["operation_id"]
        input_io = operation["input_schema_ref"]
        output_io = operation["output_schema_ref"]
        require_io(input_io, operation_id)
        output_schema = require_io(output_io, operation_id)
        operation_outcomes = set(operation["outcomes"])
        for outcome in operation_outcomes:
            require_outcome(outcome, operation_id)

        output_outcome_schema = _schema_at_pointer(
            output_schema, "/outcome", f"{operation_id} output schema"
        )
        output_enum = set(output_outcome_schema.get("enum", []))
        if output_enum != operation_outcomes:
            raise AgentContractValidationError(
                f"Operation {operation_id} outcomes do not exactly match output schema enum: "
                f"operation={sorted(operation_outcomes)}, schema={sorted(output_enum)}"
            )

        unresolved_rules = sorted(set(operation["rule_refs"]) - rule_ids)
        if unresolved_rules:
            raise AgentContractValidationError(
                f"Operation {operation_id} has unresolved rule refs: {unresolved_rules}"
            )

        for check in operation["preconditions"]:
            condition = conditions.get(check["condition_ref"])
            if condition is None:
                raise AgentContractValidationError(
                    f"Operation {operation_id} has unresolved precondition {check['condition_ref']}."
                )
            if condition["input_schema_ref"] != input_io:
                raise AgentContractValidationError(
                    f"Operation {operation_id} precondition {check['condition_ref']} "
                    "must consume the operation input schema."
                )
            require_outcome(check["on_false"], operation_id)
            require_outcome(check["on_unknown"], operation_id)
            if check["on_false"] not in operation_outcomes or check["on_unknown"] not in operation_outcomes:
                raise AgentContractValidationError(
                    f"Operation {operation_id} precondition terminal outcomes must be declared."
                )

        steps = operation["steps"]
        step_by_id: dict[str, dict[str, Any]] = {}
        step_index: dict[str, int] = {}
        for index_pos, step in enumerate(steps):
            step_id = step["step_id"]
            if step_id in step_by_id:
                raise AgentContractValidationError(
                    f"Operation {operation_id} has duplicate step id {step_id}."
                )
            step_by_id[step_id] = step
            step_index[step_id] = index_pos
        if operation["entry_step"] not in step_by_id:
            raise AgentContractValidationError(
                f"Operation {operation_id} entry_step does not exist."
            )

        def step_output_schema(step_id: str) -> dict[str, Any]:
            step = step_by_id[step_id]
            if step["type"] != "ACTION":
                raise AgentContractValidationError(
                    f"Condition step {step_id} cannot be used as a data source in {operation_id}."
                )
            action = actions[step["action_ref"]]
            return io_payloads[action["output_schema_ref"]]

        def validate_bindings(
            bindings: dict[str, str],
            target_schema_id: str,
            current_step: str,
            *,
            allow_same_step: bool = False,
            implicit_required: set[str] | None = None,
        ) -> None:
            target_schema = require_io(target_schema_id, f"{operation_id}.{current_step}")
            implicit = implicit_required or set()
            required_fields = target_schema.get("required", [])
            if isinstance(required_fields, list):
                for required_name in required_fields:
                    token = str(required_name).replace("~", "~0").replace("/", "~1")
                    if required_name not in implicit and f"/{token}" not in bindings:
                        raise AgentContractValidationError(
                            f"Operation {operation_id} step {current_step} does not bind "
                            f"required target field {required_name!r}."
                        )
            for target_pointer, source_ref in bindings.items():
                target_fragment = _schema_at_pointer(
                    target_schema, target_pointer, f"{operation_id}.{current_step} target"
                )
                source_kind, source_step_id, source_pointer = _parse_source_ref(
                    source_ref, f"{operation_id}.{current_step}"
                )
                if source_kind == "input":
                    source_schema = io_payloads[input_io]
                else:
                    assert source_step_id is not None
                    if source_step_id not in step_by_id:
                        raise AgentContractValidationError(
                            f"Operation {operation_id} binding references unknown step "
                            f"{source_step_id}."
                        )
                    source_pos = step_index[source_step_id]
                    current_pos = step_index[current_step]
                    if source_pos > current_pos or (
                        source_pos == current_pos and not allow_same_step
                    ):
                        raise AgentContractValidationError(
                            f"Operation {operation_id} step {current_step} reads non-prior "
                            f"step {source_step_id}."
                        )
                    source_schema = step_output_schema(source_step_id)
                source_fragment = _schema_at_pointer(
                    source_schema, source_pointer, f"{operation_id}.{current_step} source"
                )
                if not _compatible_schema(source_fragment, target_fragment):
                    raise AgentContractValidationError(
                        f"Operation {operation_id} step {current_step} has incompatible "
                        f"binding {source_ref!r} -> {target_pointer!r}."
                    )

        graph: dict[str, set[str]] = {step_id: set() for step_id in step_by_id}

        def validate_transition(
            transition: dict[str, Any], current_step: str
        ) -> None:
            if "step_ref" in transition:
                target = transition["step_ref"]
                if target not in step_by_id:
                    raise AgentContractValidationError(
                        f"Operation {operation_id} references unknown next step {target}."
                    )
                graph[current_step].add(target)
                return
            if "outcome" in transition:
                outcome = transition["outcome"]
                require_outcome(outcome, f"{operation_id}.{current_step}")
                if outcome not in operation_outcomes:
                    raise AgentContractValidationError(
                        f"Operation {operation_id} step {current_step} emits undeclared "
                        f"outcome {outcome}."
                    )
                if "output_bindings" in transition:
                    validate_bindings(
                        transition["output_bindings"],
                        output_io,
                        current_step,
                        allow_same_step=True,
                        implicit_required={"outcome"},
                    )
                return

            source_kind, source_step_id, source_pointer = _parse_source_ref(
                transition["outcome_from"], f"{operation_id}.{current_step}.outcome_from"
            )
            if source_kind == "input" or source_step_id is None:
                raise AgentContractValidationError(
                    f"Operation {operation_id} outcome_from must read an action step."
                )
            if source_step_id not in step_by_id:
                raise AgentContractValidationError(
                    f"Operation {operation_id} outcome_from references unknown step "
                    f"{source_step_id}."
                )
            if step_index[source_step_id] > step_index[current_step]:
                raise AgentContractValidationError(
                    f"Operation {operation_id} outcome_from reads a future step."
                )
            source_schema = step_output_schema(source_step_id)
            outcome_fragment = _schema_at_pointer(
                source_schema, source_pointer, f"{operation_id}.{current_step}.outcome_from"
            )
            values = outcome_fragment.get("enum")
            if not isinstance(values, list) or not values:
                raise AgentContractValidationError(
                    f"Operation {operation_id} dynamic outcome source must expose a non-empty enum."
                )
            unexpected = sorted(set(values) - operation_outcomes)
            if unexpected:
                raise AgentContractValidationError(
                    f"Operation {operation_id} dynamic outcome source can emit undeclared "
                    f"outcomes: {unexpected}"
                )

        for step in steps:
            step_id = step["step_id"]
            if step["type"] == "ACTION":
                action = actions.get(step["action_ref"])
                if action is None:
                    raise AgentContractValidationError(
                        f"Operation {operation_id} has unresolved action {step['action_ref']}."
                    )
                validate_bindings(step["input_bindings"], action["input_schema_ref"], step_id)
                if action["effect"] == "MUTATE":
                    gate_ref = step.get("mutation_gate_ref")
                    gate_bindings = step.get("mutation_gate_input_bindings")
                    if not gate_ref or not isinstance(gate_bindings, dict):
                        raise AgentContractValidationError(
                            f"MUTATE action step {operation_id}.{step_id} requires one mutation gate."
                        )
                    gate = gates.get(gate_ref)
                    if gate is None:
                        raise AgentContractValidationError(
                            f"Operation {operation_id} has unresolved gate {gate_ref}."
                        )
                    validate_bindings(gate_bindings, gate["input_schema_ref"], step_id)
                    gate_outcomes = {
                        value
                        for requirement in gate["requirements"]
                        for value in (requirement["on_false"], requirement["on_unknown"])
                    }
                    if not gate_outcomes.issubset(operation_outcomes):
                        raise AgentContractValidationError(
                            f"Operation {operation_id} does not declare every gate denial outcome."
                        )
                elif "mutation_gate_ref" in step or "mutation_gate_input_bindings" in step:
                    raise AgentContractValidationError(
                        f"READ_ONLY action step {operation_id}.{step_id} must not have a mutation gate."
                    )

                if not set(action["failure_outcomes"]).issubset(operation_outcomes):
                    raise AgentContractValidationError(
                        f"Operation {operation_id} does not declare every action failure outcome "
                        f"for {action['action_id']}."
                    )
                validate_transition(step["on_success"], step_id)
            else:
                condition = conditions.get(step["condition_ref"])
                if condition is None:
                    raise AgentContractValidationError(
                        f"Operation {operation_id} has unresolved condition {step['condition_ref']}."
                    )
                validate_bindings(step["input_bindings"], condition["input_schema_ref"], step_id)
                for branch_name in ("TRUE", "FALSE", "UNKNOWN"):
                    validate_transition(step["branches"][branch_name], step_id)

        reachable: set[str] = set()
        stack = [operation["entry_step"]]
        while stack:
            current = stack.pop()
            if current in reachable:
                continue
            reachable.add(current)
            stack.extend(graph[current])
        unreachable = sorted(set(step_by_id) - reachable)
        if unreachable:
            raise AgentContractValidationError(
                f"Operation {operation_id} has unreachable steps: {unreachable}"
            )

        visiting_steps: set[str] = set()
        visited_steps: set[str] = set()

        def visit_step(step_id: str) -> None:
            if step_id in visited_steps:
                return
            if step_id in visiting_steps:
                raise AgentContractValidationError(
                    f"Operation {operation_id} contains a step cycle at {step_id}."
                )
            visiting_steps.add(step_id)
            for child in graph[step_id]:
                visit_step(child)
            visiting_steps.remove(step_id)
            visited_steps.add(step_id)

        visit_step(operation["entry_step"])

    for operation in payloads["operations"].values():
        validate_operation(operation)

    binding_ref = _safe_ref(index["binding"]["current_ref"])
    binding = _yaml(binding_ref)
    _validate(binding, _json(_safe_ref(schemas.get("binding"))), binding_ref)
    _reject_markdown_dependency(binding, binding_ref)

    indexed_sets = {
        "active_specs": set(payloads["specs"]),
        "active_operations": set(payloads["operations"]),
        "active_actions": set(payloads["actions"]),
        "active_conditions": set(payloads["conditions"]),
        "active_gates": set(payloads["gates"]),
        "active_io_schemas": set(payloads["io_schemas"]),
        "active_vocabularies": set(payloads["vocabularies"]),
    }
    for field, expected in indexed_sets.items():
        actual = set(binding[field])
        if actual != expected:
            raise AgentContractValidationError(
                f"Binding {field} mismatch: missing={sorted(expected - actual)}, "
                f"extra={sorted(actual - expected)}"
            )

    action_binding_ids = [item["action_id"] for item in binding["action_bindings"]]
    if len(action_binding_ids) != len(set(action_binding_ids)):
        raise AgentContractValidationError("Duplicate action implementation binding.")
    if set(action_binding_ids) != indexed_sets["active_actions"]:
        raise AgentContractValidationError(
            "Action bindings must resolve every active action exactly once."
        )
    for item in binding["action_bindings"]:
        _resolve_callable(item["python_callable"])

    evaluator_binding_ids = [
        item["condition_id"] for item in binding["condition_evaluator_bindings"]
    ]
    if len(evaluator_binding_ids) != len(set(evaluator_binding_ids)):
        raise AgentContractValidationError("Duplicate condition evaluator binding.")
    required_evaluators = {
        condition_id
        for condition_id, condition in conditions.items()
        if condition["evaluation"]["type"] == "BOUND_EVALUATOR"
    }
    if set(evaluator_binding_ids) != required_evaluators:
        raise AgentContractValidationError(
            "Condition evaluator bindings must resolve exactly the BOUND_EVALUATOR conditions."
        )
    for item in binding["condition_evaluator_bindings"]:
        _resolve_callable(item["python_callable"])

    entrypoint_ids = [item["operation_id"] for item in binding["operation_entrypoints"]]
    if len(entrypoint_ids) != len(set(entrypoint_ids)):
        raise AgentContractValidationError("Duplicate operation entrypoint binding.")
    unknown_entrypoints = sorted(set(entrypoint_ids) - indexed_sets["active_operations"])
    if unknown_entrypoints:
        raise AgentContractValidationError(
            f"Operation entrypoints reference inactive operations: {unknown_entrypoints}"
        )

    _validate_external_machine_contract(binding)

    counts["rules"] = len(rule_ids)
    counts["bindings"] = 1
    return counts
