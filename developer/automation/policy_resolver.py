from __future__ import annotations

import argparse
import json
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
    return {
        "schema_version": "ptsip-policy-resolution/v1",
        "resolver_id": bindings.get("resolver_id"),
        "scope": normalized_scope,
        "operation": normalized_operation,
        "binding_scope": binding_scope,
        "policies": policies,
        "authority": "CANONICAL_POLICY_RECORDS",
        "projection_authority": False,
    }


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
        return ()
    except (OSError, ValueError, PolicyResolverError) as exc:
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
    return parser


def main(argv: list[str] | None = None) -> int:
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
