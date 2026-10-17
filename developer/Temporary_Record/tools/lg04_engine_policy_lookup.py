#!/usr/bin/env python3
"""LG-04 temporary read-only registered route lookup; NOT a validator.

Only the two approved indexes are inputs. No legacy-policy authority, runtime
engine conformance, or execution permission is granted by this lookup.
"""
import argparse
import json
from pathlib import Path, PurePosixPath
import sys
import yaml

POLICY_INDEX = "src/policy/index.yaml"
ENGINE_INDEX = "src/ptsip/engines/index.json"
OWNER_FIELDS = ("architecture_owner", "contract_owner")


class Unresolved(Exception):
    def __init__(self, reason):
        super().__init__(reason)
        self.reason = reason


def _registered_path(value, prefix):
    if not isinstance(value, str) or "\\" in value:
        raise Unresolved("INVALID_REGISTERED_PATH")
    p = PurePosixPath(value)
    if not value or p.is_absolute() or any(part in (".", "..") for part in value.split("/")):
        raise Unresolved("INVALID_REGISTERED_PATH")
    if str(p) != value or not value.startswith(prefix):
        raise Unresolved("INVALID_REGISTERED_PATH")
    return value


def _load(repo_root):
    try:
        policy = yaml.safe_load((repo_root / POLICY_INDEX).read_text(encoding="utf-8"))
        engines = json.loads((repo_root / ENGINE_INDEX).read_text(encoding="utf-8"))
    except (OSError, ValueError, yaml.YAMLError) as exc:
        raise Unresolved("INDEX_UNREADABLE_OR_INVALID") from exc
    if not isinstance(policy, dict) or not isinstance(engines, dict):
        raise Unresolved("INVALID_INDEX_SHAPE")
    if policy.get("policy_class") != engines.get("policy_class"):
        raise Unresolved("POLICY_CLASS_MISMATCH")
    if not isinstance(policy.get("policies"), list) or not isinstance(engines.get("official_engine_entries"), list):
        raise Unresolved("INVALID_INDEX_SHAPE")
    return policy, engines


def _owner(policy_index, path):
    _registered_path(path, "src/policy/")
    relative = path[len("src/policy/"):]
    candidates = []
    for entry in policy_index["policies"]:
        if not isinstance(entry, dict):
            raise Unresolved("INVALID_POLICY_INDEX_ENTRY")
        if entry.get("path") == relative:
            candidates.append(entry)
    if len(candidates) != 1:
        raise Unresolved("OWNER_ROUTE_MISSING_OR_AMBIGUOUS")
    entry = candidates[0]
    if entry.get("status") != "ACTIVE" or entry.get("authority_role") != "CANONICAL_AUTHORITY":
        raise Unresolved("OWNER_NOT_ACTIVE_CANONICAL")
    if not isinstance(entry.get("id"), str) or not entry["id"]:
        raise Unresolved("INVALID_OWNER_IDENTITY")
    return {
        "policy_id": entry["id"], "policy_path": path,
        "status": entry["status"], "authority_role": entry["authority_role"]
    }


def resolve(repo_root, engine_kind, operation=None):
    policy_index, engine_index = _load(repo_root)
    matches = []
    for engine in engine_index["official_engine_entries"]:
        if not isinstance(engine, dict):
            raise Unresolved("INVALID_ENGINE_INDEX_ENTRY")
        if engine.get("kind") == engine_kind:
            matches.append(engine)
    if len(matches) != 1:
        raise Unresolved("ENGINE_NOT_REGISTERED_OR_AMBIGUOUS")
    engine = matches[0]
    ops = engine.get("operations")
    if not isinstance(ops, list) or any(not isinstance(v, str) for v in ops):
        raise Unresolved("INVALID_OPERATION_REGISTRY")
    if operation is not None and operation not in ops:
        raise Unresolved("OPERATION_NOT_REGISTERED")
    owners = {
        field: _owner(policy_index, engine_index.get(field)) for field in OWNER_FIELDS
    }
    return {
        "status": "ROUTE_RESOLVED_NOT_VALIDATED",
        "lookup": "EXACT_REGISTERED_INDEX_RELATION",
        "sources": {"policy_index": POLICY_INDEX, "engine_index": ENGINE_INDEX},
        "engine": {
            "kind": engine_kind,
            "engine_id": engine.get("engine_id"),
            "version": engine.get("version"),
            "capability_id": engine.get("capability_id"),
            "operation": operation,
            "registered_operations": ops,
            "contract_ref": engine.get("contract_ref"),
            "execution_binding_ref": engine.get("execution_binding_ref")
        },
        "declared_policy_owners": owners,
        "other_responsibility_routes": "UNRESOLVED_NOT_DECLARED_BY_APPROVED_INDICES",
        "execution_permission": "NOT_GRANTED_OR_RESOLVED",
        "engine_execution_performed": False
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", default=".", help="Repository root")
    parser.add_argument("--engine-kind", required=True, help="Exact registered kind")
    parser.add_argument("--operation", help="Exact registered operation")
    args = parser.parse_args(argv)
    try:
        result = resolve(Path(args.repository), args.engine_kind, args.operation)
        code = 0
    except Unresolved as exc:
        result = {
            "status": "UNRESOLVED", "reason": exc.reason,
            "execution_permission": "NOT_GRANTED_OR_RESOLVED"
        }
        code = 2
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))
    return code


if __name__ == "__main__":
    sys.exit(main())
