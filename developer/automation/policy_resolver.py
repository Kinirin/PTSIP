from __future__ import annotations

import argparse
import ast
import json
import subprocess
import sys
from pathlib import Path, PurePosixPath
from typing import Mapping

import yaml
from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


CONTRACT_PATH = "developer/policy/policy-resolver-contract.yaml"


class PolicyResolverError(RuntimeError):
    pass


def _mapping(value: object, *, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise PolicyResolverError(f"{label} must be a mapping")
    return value


def _contract(root: Path) -> dict[str, object]:
    return load_yaml(CONTRACT_PATH, root=root)


def _resolver_config(contract: Mapping[str, object]) -> Mapping[str, object]:
    return _mapping(contract.get("resolver"), label="resolver")


def _configured_path(
    root: Path,
    contract: Mapping[str, object],
    field: str,
) -> str:
    resolver = _resolver_config(contract)
    value = resolver.get(field)
    if not isinstance(value, str) or not value:
        raise PolicyResolverError(f"resolver.{field} must be a non-empty path")
    candidate = (root / value).resolve()
    try:
        candidate.relative_to(root)
    except ValueError as exc:
        raise PolicyResolverError(f"resolver.{field} escapes repository root") from exc
    return value


def _load_bindings(root: Path, contract: Mapping[str, object]) -> dict[str, object]:
    binding_path = _configured_path(root, contract, "binding_registry_ref")
    schema_path = _configured_path(root, contract, "binding_schema_ref")
    payload = load_yaml(binding_path, root=root)
    schema = load_json(schema_path, root=root)
    Draft202012Validator.check_schema(schema)
    errors = sorted(
        Draft202012Validator(schema).iter_errors(payload),
        key=lambda error: tuple(str(part) for part in error.absolute_path),
    )
    if errors:
        rendered = "; ".join(
            f"{'.'.join(str(part) for part in error.absolute_path) or '<root>'}: {error.message}"
            for error in errors
        )
        raise PolicyResolverError(f"Policy Resolver bindings are invalid: {rendered}")
    return payload


def _load_index(
    root: Path,
    contract: Mapping[str, object],
) -> dict[str, dict[str, str]]:
    index_path = _configured_path(root, contract, "policy_index_ref")
    payload = load_yaml(index_path, root=root)
    entries = payload.get("policies")
    if not isinstance(entries, list):
        raise PolicyResolverError("developer policy index must contain policies")
    result: dict[str, dict[str, str]] = {}
    for raw in entries:
        entry = _mapping(raw, label="policy index entry")
        policy_id = entry.get("id")
        path = entry.get("path")
        status = entry.get("status")
        if not isinstance(policy_id, str) or not policy_id:
            raise PolicyResolverError("policy index id must be a non-empty string")
        if not isinstance(path, str) or not path:
            raise PolicyResolverError(f"{policy_id}: policy index path is invalid")
        if not isinstance(status, str) or not status:
            raise PolicyResolverError(f"{policy_id}: policy index status is invalid")
        if policy_id in result:
            raise PolicyResolverError(f"duplicate policy identity: {policy_id}")
        result[policy_id] = {
            "id": policy_id,
            "path": path,
            "status": status,
        }
    return result


def _normalize_scope(root: Path, scope: str) -> str:
    if not isinstance(scope, str) or not scope.strip():
        raise PolicyResolverError("scope must be a non-empty repository path")
    normalized_input = scope.strip().replace("\\", "/")
    raw = Path(normalized_input)
    candidate = raw.resolve() if raw.is_absolute() else (root / raw).resolve()
    try:
        relative = candidate.relative_to(root)
    except ValueError as exc:
        raise PolicyResolverError("scope must remain inside the repository") from exc
    value = relative.as_posix()
    return "." if value in {"", "."} else value


def _ancestor_scopes(scope: str) -> tuple[str, ...]:
    if scope == ".":
        return (".",)
    current = PurePosixPath(scope)
    result = [current.as_posix()]
    while current.parent != current and current.parent.as_posix() not in {"", "."}:
        current = current.parent
        result.append(current.as_posix())
    result.append(".")
    return tuple(result)


def _select_scope_binding(
    bindings: Mapping[str, object],
    scope: str,
) -> tuple[str, Mapping[str, object]]:
    scope_bindings = _mapping(
        bindings.get("scope_bindings"),
        label="scope_bindings",
    )
    for candidate in _ancestor_scopes(scope):
        value = scope_bindings.get(candidate)
        if value is not None:
            return candidate, _mapping(value, label=f"scope_bindings.{candidate}")
    raise PolicyResolverError(f"no registered policy binding for scope {scope!r}")


def _selected_refs(
    bindings: Mapping[str, object],
    binding: Mapping[str, object],
    operation: str,
) -> list[object]:
    vocabulary = bindings.get("operation_vocabulary")
    if not isinstance(vocabulary, list) or operation not in vocabulary:
        raise PolicyResolverError(f"unsupported policy operation: {operation}")
    operations_raw = binding.get("operations", {})
    operations = _mapping(operations_raw, label="operations")
    selected = operations.get(operation, binding.get("default_refs"))
    if not isinstance(selected, list) or not selected:
        raise PolicyResolverError(
            f"registered scope has no refs for operation {operation}"
        )
    return selected


def _validate_policy_ref(
    root: Path,
    index: Mapping[str, Mapping[str, str]],
    raw_ref: object,
) -> dict[str, object]:
    reference = _mapping(raw_ref, label="policy reference")
    policy_id = reference.get("policy_id")
    sections = reference.get("sections")
    if not isinstance(policy_id, str) or policy_id not in index:
        raise PolicyResolverError(
            f"binding references unknown developer policy {policy_id!r}"
        )
    entry = index[policy_id]
    if entry["status"] != "ACTIVE":
        raise PolicyResolverError(
            f"binding references non-active policy {policy_id} ({entry['status']})"
        )
    if not isinstance(sections, list) or not sections:
        raise PolicyResolverError(f"{policy_id}: sections must be non-empty")
    policy = load_yaml(entry["path"], root=root)
    identity = _mapping(policy.get("policy"), label=f"{policy_id}.policy")
    if identity.get("id") != policy_id:
        raise PolicyResolverError(
            f"{policy_id}: canonical record identity does not match index"
        )
    rules = _mapping(policy.get("rules"), label=f"{policy_id}.rules")
    normalized_sections: list[str] = []
    for section in sections:
        if not isinstance(section, str) or section not in rules:
            raise PolicyResolverError(
                f"{policy_id}: binding references unknown rule section {section!r}"
            )
        normalized_sections.append(section)
    if len(normalized_sections) != len(set(normalized_sections)):
        raise PolicyResolverError(f"{policy_id}: duplicate rule section reference")
    return {
        "policy_id": policy_id,
        "path": entry["path"],
        "status": entry["status"],
        "sections": normalized_sections,
    }


def _repository_file(
    root: Path,
    value: object,
    *,
    label: str,
) -> tuple[str, Path]:
    if not isinstance(value, str) or not value.strip():
        raise PolicyResolverError(f"{label} must be a non-empty repository path")
    path_text = value.strip()
    candidate = (root / path_text).resolve()
    try:
        candidate.relative_to(root)
    except ValueError as exc:
        raise PolicyResolverError(f"{label} escapes repository root") from exc
    if not candidate.is_file():
        raise PolicyResolverError(f"{label} does not exist: {path_text}")
    return path_text, candidate


def _validate_repository_ref(root: Path, value: object, *, label: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise PolicyResolverError(f"{label} must be a non-empty repository reference")
    reference = value.strip()
    path_text, marker, fragment = reference.partition("#")
    _repository_file(root, path_text, label=label)
    if marker and not fragment:
        raise PolicyResolverError(f"{label} has an empty fragment")
    return reference


def _current_branch(root: Path) -> str:
    completed = subprocess.run(
        ["git", "-C", str(root), "branch", "--show-current"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if completed.returncode != 0:
        detail = completed.stderr.strip() or "git branch --show-current failed"
        raise PolicyResolverError(f"unable to resolve current Git branch: {detail}")
    branch = completed.stdout.strip()
    if not branch:
        raise PolicyResolverError(
            "Policy Resolver task context requires a named Git branch; detached HEAD fails closed"
        )
    return branch


def _normative_rule_registry(
    root: Path,
    contract: Mapping[str, object],
) -> dict[str, dict[str, object]]:
    registry_path = _configured_path(root, contract, "normative_rule_registry_ref")
    payload = load_yaml(registry_path, root=root)
    registry = _mapping(payload.get("ptsip_registry"), label="ptsip_registry")
    rules = registry.get("rules")
    if not isinstance(rules, list):
        raise PolicyResolverError("normative rule registry has no rules list")
    result: dict[str, dict[str, object]] = {}
    for raw in rules:
        rule = _mapping(raw, label="normative rule")
        rule_id = rule.get("id")
        if not isinstance(rule_id, str) or not rule_id:
            raise PolicyResolverError("normative rule id must be a non-empty string")
        if rule_id in result:
            raise PolicyResolverError(f"duplicate normative rule identity: {rule_id}")
        result[rule_id] = dict(rule)
    return result


def get_normative_rule(
    repository: str | Path,
    *,
    rule_id: str,
) -> dict[str, object]:
    root = repository_root(repository)
    contract = _contract(root)
    registry = _normative_rule_registry(root, contract)
    record = registry.get(rule_id)
    if record is None:
        raise PolicyResolverError(f"unknown normative rule identity: {rule_id}")

    source_path = _configured_path(root, contract, "normative_rule_registry_ref")
    _repository_file(
        root,
        source_path,
        label="normative_rule_registry_ref",
    )
    return {
        "schema_version": "ptsip-normative-rule-projection/v2",
        "rule_id": rule_id,
        "canonical_source": source_path,
        "registry_record": record,
        "projection_authority": False,
    }

def _python_tree(path_text: str, candidate: Path) -> ast.Module:
    try:
        return ast.parse(
            candidate.read_text(encoding="utf-8"),
            filename=path_text,
        )
    except SyntaxError as exc:
        raise PolicyResolverError(
            f"implementation ref source is not valid Python: {path_text}: {exc}"
        ) from exc


def _location(node: ast.AST) -> dict[str, int]:
    line_start = getattr(node, "lineno", None)
    line_end = getattr(node, "end_lineno", line_start)
    if not isinstance(line_start, int) or not isinstance(line_end, int):
        raise PolicyResolverError("implementation selector resolved without source location")
    return {
        "line_start": line_start,
        "line_end": line_end,
    }


def _cli_command_branch_matches(test: ast.AST, command: str) -> bool:
    if not isinstance(test, ast.Compare):
        return False
    if len(test.ops) != 1 or not isinstance(test.ops[0], ast.Eq):
        return False
    if len(test.comparators) != 1:
        return False
    left = test.left
    right = test.comparators[0]
    return (
        isinstance(left, ast.Attribute)
        and left.attr == "command"
        and isinstance(left.value, ast.Name)
        and left.value.id == "args"
        and isinstance(right, ast.Constant)
        and right.value == command
    )


def _validate_implementation_ref(
    root: Path,
    raw_ref: object,
) -> dict[str, object]:
    reference = _mapping(raw_ref, label="implementation reference")
    path_text, candidate = _repository_file(
        root,
        reference.get("path"),
        label="implementation reference path",
    )
    if candidate.suffix != ".py":
        raise PolicyResolverError(
            f"typed implementation ref must target a Python source file: {path_text}"
        )
    selector = _mapping(
        reference.get("selector"),
        label=f"{path_text} implementation selector",
    )
    kind = selector.get("kind")
    if kind not in {
        "PYTHON_FUNCTION",
        "PYTHON_METHOD",
        "CLI_COMMAND_BRANCH",
        "PYTHON_MODULE",
    }:
        raise PolicyResolverError(
            f"{path_text}: unsupported implementation selector kind {kind!r}"
        )

    tree = _python_tree(path_text, candidate)
    selected: ast.AST
    if kind == "PYTHON_FUNCTION":
        name = selector.get("name")
        if not isinstance(name, str) or not name:
            raise PolicyResolverError(f"{path_text}: PYTHON_FUNCTION requires name")
        matches = [
            node
            for node in tree.body
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
            and node.name == name
        ]
        if len(matches) != 1:
            raise PolicyResolverError(
                f"{path_text}: PYTHON_FUNCTION {name!r} must resolve exactly once"
            )
        selected = matches[0]
    elif kind == "PYTHON_METHOD":
        class_name = selector.get("class")
        method_name = selector.get("method")
        if not isinstance(class_name, str) or not class_name:
            raise PolicyResolverError(f"{path_text}: PYTHON_METHOD requires class")
        if not isinstance(method_name, str) or not method_name:
            raise PolicyResolverError(f"{path_text}: PYTHON_METHOD requires method")
        classes = [
            node
            for node in tree.body
            if isinstance(node, ast.ClassDef) and node.name == class_name
        ]
        if len(classes) != 1:
            raise PolicyResolverError(
                f"{path_text}: class {class_name!r} must resolve exactly once"
            )
        methods = [
            node
            for node in classes[0].body
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
            and node.name == method_name
        ]
        if len(methods) != 1:
            raise PolicyResolverError(
                f"{path_text}: method {class_name}.{method_name} must resolve exactly once"
            )
        selected = methods[0]
    elif kind == "CLI_COMMAND_BRANCH":
        command = selector.get("command")
        if not isinstance(command, str) or not command:
            raise PolicyResolverError(f"{path_text}: CLI_COMMAND_BRANCH requires command")
        matches = [
            node
            for node in ast.walk(tree)
            if isinstance(node, ast.If)
            and _cli_command_branch_matches(node.test, command)
        ]
        if len(matches) != 1:
            raise PolicyResolverError(
                f"{path_text}: CLI command branch {command!r} must resolve exactly once"
            )
        selected = matches[0]
    else:
        selected = tree

    if kind == "PYTHON_MODULE":
        line_count = len(candidate.read_text(encoding="utf-8").splitlines())
        resolved_location = {
            "line_start": 1,
            "line_end": max(1, line_count),
        }
    else:
        resolved_location = _location(selected)
    return {
        "path": path_text,
        "selector": dict(selector),
        "resolved_location": resolved_location,
    }


def _task_context(
    root: Path,
    contract: Mapping[str, object],
    *,
    scope: str,
    operation: str,
    enforce_branch: bool,
) -> dict[str, object] | None:
    context_path = _configured_path(root, contract, "task_context_ref")
    payload = load_yaml(context_path, root=root)
    entry = _mapping(payload.get("coding_agent_entry"), label="coding_agent_entry")
    task_bindings = _mapping(
        entry.get("task_bindings"),
        label="coding_agent_entry.task_bindings",
    )
    scope_binding = task_bindings.get(scope)
    if scope_binding is None:
        return None
    operations = _mapping(
        scope_binding,
        label=f"coding_agent_entry.task_bindings.{scope}",
    )
    raw_context = operations.get(operation)
    if raw_context is None:
        return None
    context = _mapping(
        raw_context,
        label=f"coding_agent_entry.task_bindings.{scope}.{operation}",
    )

    declared_branch = entry.get("branch")
    if not isinstance(declared_branch, str) or not declared_branch:
        raise PolicyResolverError("coding_agent_entry.branch must be non-empty")
    actual_branch: str | None = None
    branch_match: bool | None = None
    if enforce_branch:
        actual_branch = _current_branch(root)
        branch_match = actual_branch == declared_branch
        if not branch_match:
            raise PolicyResolverError(
                "task context branch mismatch: "
                f"declared {declared_branch!r}, actual {actual_branch!r}"
            )

    planning_entry = _validate_repository_ref(
        root,
        entry.get("planning_entry"),
        label="planning_entry",
    )

    raw_rules = context.get("normative_rule_refs")
    if not isinstance(raw_rules, list) or not raw_rules:
        raise PolicyResolverError("task context normative_rule_refs must be non-empty")
    normative_rule_refs: list[str] = []
    for item in raw_rules:
        if not isinstance(item, str) or not item:
            raise PolicyResolverError(
                "task context normative_rule_refs must contain non-empty strings"
            )
        normative_rule_refs.append(item)
    if len(normative_rule_refs) != len(set(normative_rule_refs)):
        raise PolicyResolverError("task context normative_rule_refs must be unique")
    normative_rules = [
        get_normative_rule(root, rule_id=rule_id)
        for rule_id in normative_rule_refs
    ]

    raw_implementation_refs = context.get("implementation_refs")
    if not isinstance(raw_implementation_refs, list) or not raw_implementation_refs:
        raise PolicyResolverError("task context implementation_refs must be non-empty")
    implementation_refs = [
        _validate_implementation_ref(root, item)
        for item in raw_implementation_refs
    ]

    raw_test_refs = context.get("test_refs")
    if not isinstance(raw_test_refs, list) or not raw_test_refs:
        raise PolicyResolverError("task context test_refs must be non-empty")
    test_refs = [
        _validate_repository_ref(root, item, label="test_refs item")
        for item in raw_test_refs
    ]
    if len(test_refs) != len(set(test_refs)):
        raise PolicyResolverError("task context test_refs must be unique")

    raw_constraints = context.get("constraints")
    if not isinstance(raw_constraints, list) or not raw_constraints:
        raise PolicyResolverError("task context constraints must be non-empty")
    constraints: list[str] = []
    for item in raw_constraints:
        if not isinstance(item, str) or not item:
            raise PolicyResolverError(
                "task context constraints must contain non-empty strings"
            )
        constraints.append(item)
    if len(constraints) != len(set(constraints)):
        raise PolicyResolverError("task context constraints must be unique")

    return {
        "branch": declared_branch,
        "branch_context": {
            "declared": declared_branch,
            "actual": actual_branch,
            "match": branch_match,
        },
        "planning_entry": planning_entry,
        "normative_rule_refs": normative_rule_refs,
        "normative_rule_source": _configured_path(
            root,
            contract,
            "normative_rule_registry_ref",
        ),
        "normative_rules": normative_rules,
        "implementation_refs": implementation_refs,
        "test_refs": test_refs,
        "constraints": constraints,
    }


def resolve_policies(
    repository: str | Path,
    *,
    scope: str,
    operation: str,
) -> dict[str, object]:
    root = repository_root(repository)
    contract = _contract(root)
    bindings = _load_bindings(root, contract)
    index = _load_index(root, contract)
    normalized_scope = _normalize_scope(root, scope)
    normalized_operation = operation.strip().upper()
    binding_scope, binding = _select_scope_binding(bindings, normalized_scope)
    refs = _selected_refs(bindings, binding, normalized_operation)
    policies = [
        _validate_policy_ref(root, index, reference)
        for reference in refs
    ]
    ids = [str(item["policy_id"]) for item in policies]
    if len(ids) != len(set(ids)):
        raise PolicyResolverError("resolved policy identities must be unique")
    result: dict[str, object] = {
        "schema_version": "ptsip-policy-resolution/v1",
        "resolver_id": bindings.get("resolver_id"),
        "scope": normalized_scope,
        "operation": normalized_operation,
        "binding_scope": binding_scope,
        "policies": policies,
        "authority": "CANONICAL_POLICY_RECORDS",
        "projection_authority": False,
    }
    task_context = _task_context(
        root,
        contract,
        scope=normalized_scope,
        operation=normalized_operation,
        enforce_branch=True,
    )
    if task_context is not None:
        result["task_context"] = task_context
    return result


def validate_policy_resolver(
    repository: str | Path,
) -> tuple[str, ...]:
    try:
        root = repository_root(repository)
        contract = _contract(root)
        resolver = _resolver_config(contract)
        if resolver.get("owner_policy_ref") != "MPD-0010":
            raise PolicyResolverError(
                "Policy Resolver owner_policy_ref must be MPD-0010"
            )
        bindings = _load_bindings(root, contract)
        index = _load_index(root, contract)
        scope_bindings = _mapping(
            bindings.get("scope_bindings"),
            label="scope_bindings",
        )
        for scope, raw_binding in scope_bindings.items():
            if not isinstance(scope, str) or not scope:
                raise PolicyResolverError("scope binding key must be non-empty")
            binding = _mapping(
                raw_binding,
                label=f"scope_bindings.{scope}",
            )
            refs_by_operation: list[tuple[str, object]] = [
                ("DEFAULT", binding.get("default_refs"))
            ]
            operations = _mapping(
                binding.get("operations", {}),
                label=f"scope_bindings.{scope}.operations",
            )
            refs_by_operation.extend(
                (str(operation), refs)
                for operation, refs in operations.items()
            )
            for operation, refs in refs_by_operation:
                if not isinstance(refs, list) or not refs:
                    raise PolicyResolverError(
                        f"{scope}:{operation} must contain policy refs"
                    )
                seen: set[str] = set()
                for reference in refs:
                    resolved = _validate_policy_ref(
                        root,
                        index,
                        reference,
                    )
                    policy_id = str(resolved["policy_id"])
                    if policy_id in seen:
                        raise PolicyResolverError(
                            f"{scope}:{operation} duplicates {policy_id}"
                        )
                    seen.add(policy_id)

        task_context_path = _configured_path(root, contract, "task_context_ref")
        task_payload = load_yaml(task_context_path, root=root)
        entry = _mapping(task_payload.get("coding_agent_entry"), label="coding_agent_entry")
        task_bindings = _mapping(
            entry.get("task_bindings"),
            label="coding_agent_entry.task_bindings",
        )
        vocabulary = bindings.get("operation_vocabulary")
        if not isinstance(vocabulary, list):
            raise PolicyResolverError("operation_vocabulary must be a list")
        for task_scope, raw_operations in task_bindings.items():
            if task_scope not in scope_bindings:
                raise PolicyResolverError(
                    f"task context scope {task_scope!r} has no exact policy binding"
                )
            operations = _mapping(
                raw_operations,
                label=f"coding_agent_entry.task_bindings.{task_scope}",
            )
            for operation in operations:
                if operation not in vocabulary:
                    raise PolicyResolverError(
                        f"task context uses unsupported operation {operation!r}"
                    )
                if _task_context(
                    root,
                    contract,
                    scope=str(task_scope),
                    operation=str(operation),
                    enforce_branch=False,
                ) is None:
                    raise PolicyResolverError(
                        f"task context did not resolve for {task_scope}:{operation}"
                    )
        return ()
    except (OSError, ValueError, yaml.YAMLError, PolicyResolverError) as exc:
        return (str(exc),)


def get_policy(
    repository: str | Path,
    *,
    policy_id: str,
    section: str | None = None,
) -> dict[str, object]:
    root = repository_root(repository)
    contract = _contract(root)
    index = _load_index(root, contract)
    entry = index.get(policy_id)
    if entry is None:
        raise PolicyResolverError(f"unknown developer policy identity: {policy_id}")
    payload = load_yaml(entry["path"], root=root)
    identity = _mapping(payload.get("policy"), label=f"{policy_id}.policy")
    if identity.get("id") != policy_id:
        raise PolicyResolverError(
            f"{policy_id}: canonical record identity does not match index"
        )
    if section is None:
        value: object = payload
        fragment = None
    else:
        rules = _mapping(payload.get("rules"), label=f"{policy_id}.rules")
        if section not in rules:
            raise PolicyResolverError(
                f"{policy_id}: unknown rule section {section!r}"
            )
        value = rules[section]
        fragment = f"rules.{section}"
    return {
        "schema_version": "ptsip-policy-get/v1",
        "policy_id": policy_id,
        "canonical_path": entry["path"],
        "fragment": fragment,
        "record": value,
    }


def explain_policy(
    repository: str | Path,
    *,
    policy_id: str,
) -> dict[str, object]:
    root = repository_root(repository)
    contract = _contract(root)
    index = _load_index(root, contract)
    entry = index.get(policy_id)
    if entry is None:
        raise PolicyResolverError(f"unknown developer policy identity: {policy_id}")
    payload = load_yaml(entry["path"], root=root)
    identity = _mapping(payload.get("policy"), label=f"{policy_id}.policy")
    rules = _mapping(payload.get("rules"), label=f"{policy_id}.rules")
    return {
        "schema_version": "ptsip-policy-explain/v1",
        "policy_id": policy_id,
        "title": identity.get("title"),
        "status": identity.get("status"),
        "canonical_path": entry["path"],
        "rule_sections": list(rules.keys()),
    }


def _emit(payload: object, *, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, indent=2, ensure_ascii=False))
    else:
        print(
            yaml.safe_dump(
                payload,
                sort_keys=False,
                allow_unicode=True,
            ).rstrip()
        )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="python -m developer.automation.policy_resolver",
        description="Deterministic developer-policy lookup.",
    )
    parser.add_argument("--repository", default=".")
    sub = parser.add_subparsers(dest="command", required=True)

    resolve = sub.add_parser("resolve")
    resolve.add_argument("--scope", required=True)
    resolve.add_argument("--operation", required=True)
    resolve.add_argument("--json", action="store_true")

    get = sub.add_parser("get")
    get.add_argument("policy_id")
    get.add_argument("--section")
    get.add_argument("--json", action="store_true")

    explain = sub.add_parser("explain")
    explain.add_argument("policy_id")
    explain.add_argument("--json", action="store_true")

    rule = sub.add_parser("rule")
    rule.add_argument("rule_id")
    rule.add_argument("--json", action="store_true")
    return parser


def main(argv: list[str] | None = None) -> int:
    for stream in (sys.stdout, sys.stderr):
        reconfigure = getattr(stream, "reconfigure", None)
        if callable(reconfigure):
            reconfigure(errors="backslashreplace")
    args = _parser().parse_args(argv)
    try:
        if args.command == "resolve":
            payload = resolve_policies(
                args.repository,
                scope=args.scope,
                operation=args.operation,
            )
        elif args.command == "get":
            payload = get_policy(
                args.repository,
                policy_id=args.policy_id,
                section=args.section,
            )
        elif args.command == "rule":
            payload = get_normative_rule(
                args.repository,
                rule_id=args.rule_id,
            )
        else:
            payload = explain_policy(
                args.repository,
                policy_id=args.policy_id,
            )
    except (OSError, ValueError, yaml.YAMLError, PolicyResolverError) as exc:
        print(f"Policy Resolver error: {exc}")
        return 2
    _emit(payload, as_json=bool(args.json))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
