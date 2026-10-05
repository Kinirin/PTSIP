from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Mapping, Sequence

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


REGISTRY = "developer/policy/registries/root-family-entry-registry.json"
REGISTRY_SCHEMA = "developer/policy/schemas/root-family-entry-registry.schema.json"
DEVELOPER_INDEX = "developer/policy/index.yaml"
SUPPORT_INDEX = "src/policy/index.yaml"

ROOT_FAMILIES = ["NORM","GOV","INTENT","ARCH","INFO","CNTR","RISK","SUPPLY","REAL","ASSURE","CTRL","CHANGE","OPS","RECORD"]
LEGACY_DEVELOPER_FAMILIES = ("SPEC", "PLAN", "WORK", "VERI", "MIGR", "RELS")

_ID_PATTERNS = {
    "PTSIP_DEVELOPER_POLICY": re.compile(r"^MPD-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD)-([0-9]{4})$"),
    "PTSIP_SUPPORT_FEATURE": re.compile(r"^SFP-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD)-([0-9]{4})$"),
}


class RootFamilyEntryError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


def _mapping(value: object, *, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise RootFamilyEntryError("INVALID_ROOT_FAMILY_ENTRY", f"{label} must be a mapping")
    return value


def _registry(root: Path) -> Mapping[str, object]:
    record = load_json(REGISTRY, root=root)
    schema = load_json(REGISTRY_SCHEMA, root=root)
    Draft202012Validator.check_schema(schema)
    errors = tuple(Draft202012Validator(schema).iter_errors(record))
    if errors:
        raise RootFamilyEntryError(
            "INVALID_ROOT_FAMILY_REGISTRY",
            "; ".join(error.message for error in errors),
        )
    return record


def _route(root: Path, policy_class: str, family: str) -> Mapping[str, object]:
    record = _registry(root)
    if family in LEGACY_DEVELOPER_FAMILIES:
        raise RootFamilyEntryError(
            "LEGACY_FAMILY_NEW_ALLOCATION_FORBIDDEN",
            f"{family} is readable migration input only and cannot receive a new Root Family policy",
        )
    if family not in ROOT_FAMILIES:
        raise RootFamilyEntryError("UNKNOWN_ROOT_FAMILY", f"unknown Root Family: {family}")
    classes = _mapping(record.get("policy_classes"), label="policy_classes")
    route = classes.get(policy_class)
    if not isinstance(route, Mapping):
        raise RootFamilyEntryError("UNKNOWN_POLICY_CLASS", f"unregistered policy class: {policy_class}")
    recognized = route.get("recognized_root_families")
    if not isinstance(recognized, list) or family not in recognized:
        raise RootFamilyEntryError(
            "ROOT_FAMILY_NOT_REGISTERED_FOR_CLASS",
            f"{policy_class} does not register Root Family {family}",
        )
    return route


def _index_path(policy_class: str) -> str:
    if policy_class == "PTSIP_DEVELOPER_POLICY":
        return DEVELOPER_INDEX
    if policy_class == "PTSIP_SUPPORT_FEATURE":
        return SUPPORT_INDEX
    raise RootFamilyEntryError("UNKNOWN_POLICY_CLASS", f"unregistered policy class: {policy_class}")


def _current_ids(root: Path, policy_class: str) -> tuple[str, ...]:
    payload = load_yaml(_index_path(policy_class), root=root)
    entries = payload.get("policies")
    if not isinstance(entries, list):
        raise RootFamilyEntryError("INVALID_POLICY_INDEX", "policy index must contain policies")
    result: list[str] = []
    for raw in entries:
        if not isinstance(raw, Mapping):
            raise RootFamilyEntryError("INVALID_POLICY_INDEX", "policy index entry must be a mapping")
        if policy_class == "PTSIP_DEVELOPER_POLICY" and raw.get("policy_class") != policy_class:
            continue
        policy_id = raw.get("id")
        if isinstance(policy_id, str):
            result.append(policy_id)
    return tuple(result)


def _assert_root_corpus_consistent(root: Path, policy_class: str, route: Mapping[str, object], ids: Sequence[str]) -> None:
    pattern = _ID_PATTERNS[policy_class]
    indexed = {policy_id for policy_id in ids if pattern.fullmatch(policy_id)}
    canonical_root = root / str(route["canonical_root"])
    discovered = {
        path.stem
        for path in canonical_root.rglob("*.yaml")
        if pattern.fullmatch(path.stem)
    }
    if discovered != indexed:
        raise RootFamilyEntryError(
            "ROOT_FAMILY_CORPUS_MISMATCH",
            f"{policy_class} Root Family files and index membership differ: "
            f"indexed={sorted(indexed)!r}, discovered={sorted(discovered)!r}",
        )

    payload = load_yaml(_index_path(policy_class), root=root)
    entries = payload.get("policies", [])
    if not isinstance(entries, list):
        raise RootFamilyEntryError("INVALID_POLICY_INDEX", "policy index must contain policies")
    for raw in entries:
        if not isinstance(raw, Mapping):
            continue
        policy_id = raw.get("id")
        if not isinstance(policy_id, str):
            continue
        match = pattern.fullmatch(policy_id)
        if match is None:
            continue
        expected = f"{route['canonical_root']}/{match.group(1)}/{policy_id}.yaml"
        indexed_path = raw.get("path")
        if policy_class == "PTSIP_SUPPORT_FEATURE" and isinstance(indexed_path, str):
            indexed_path = f"src/policy/{indexed_path}"
        if indexed_path != expected:
            raise RootFamilyEntryError(
                "ROOT_FAMILY_INDEX_PATH_MISMATCH",
                f"{policy_id} must route to {expected}, got {indexed_path!r}",
            )


def _next_id(ids: Sequence[str], *, policy_class: str, family: str) -> str:
    pattern = _ID_PATTERNS[policy_class]
    numbers = []
    for policy_id in ids:
        match = pattern.fullmatch(policy_id)
        if match is not None and match.group(1) == family:
            numbers.append(int(match.group(2)))
    number = max(numbers, default=0) + 1
    if number > 9999:
        raise RootFamilyEntryError("POLICY_ID_SPACE_EXHAUSTED", f"{policy_class} + {family} ID space exhausted")
    prefix = "MPD" if policy_class == "PTSIP_DEVELOPER_POLICY" else "SFP"
    return f"{prefix}-{family}-{number:04d}"


def canonical_path(policy_class: str, family: str, policy_id: str, *, root: str | Path | None = None) -> str:
    base = repository_root(root)
    route = _route(base, policy_class, family)
    pattern = _ID_PATTERNS[policy_class]
    match = pattern.fullmatch(policy_id)
    if match is None or match.group(1) != family:
        raise RootFamilyEntryError(
            "POLICY_ID_ROUTE_MISMATCH",
            f"{policy_id} is not a {policy_class} + {family} Root Family identity",
        )
    return f"{route['canonical_root']}/{family}/{policy_id}.yaml"


def resolve_entry(policy_class: str, family: str, *, root: str | Path | None = None) -> dict[str, object]:
    base = repository_root(root)
    route = _route(base, policy_class, family)
    ids = _current_ids(base, policy_class)
    _assert_root_corpus_consistent(base, policy_class, route, ids)
    allocated = _next_id(ids, policy_class=policy_class, family=family)
    return {
        "status": "READY",
        "authority_identity": {
            "policy_class": policy_class,
            "responsibility_family": family,
        },
        "semantic_inheritance": "FORBIDDEN",
        "family_state_before_materialization": route["unmaterialized_slot_default"],
        "allocated_policy_id": allocated,
        "canonical_path": canonical_path(policy_class, family, allocated, root=base),
        "schema_ref": route["schema_ref"],
    }


def inspect_id(policy_id: str, *, root: str | Path | None = None) -> dict[str, object]:
    base = repository_root(root)
    matches = []
    for policy_class, pattern in _ID_PATTERNS.items():
        match = pattern.fullmatch(policy_id)
        if match is not None:
            matches.append((policy_class, match.group(1)))
    if len(matches) != 1:
        raise RootFamilyEntryError(
            "NOT_ROOT_FAMILY_POLICY_ID",
            f"{policy_id} is not an exact registered Root Family policy identity",
        )
    policy_class, family = matches[0]
    route = _route(base, policy_class, family)
    expected_path = canonical_path(policy_class, family, policy_id, root=base)
    return {
        "status": "ROOT_FAMILY_ID",
        "policy_id": policy_id,
        "policy_class": policy_class,
        "responsibility_family": family,
        "canonical_path": expected_path,
        "schema_ref": route["schema_ref"],
        "semantic_inheritance": "FORBIDDEN",
    }


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Resolve class-scoped Root Family policy identity and schema entry.")
    parser.add_argument("--root")
    sub = parser.add_subparsers(dest="command", required=True)

    resolve = sub.add_parser("resolve")
    resolve.add_argument("--policy-class", required=True)
    resolve.add_argument("--family", required=True)

    inspect = sub.add_parser("inspect")
    inspect.add_argument("policy_id")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "resolve":
            result = resolve_entry(args.policy_class, args.family, root=args.root)
        else:
            result = inspect_id(args.policy_id, root=args.root)
    except RootFamilyEntryError as exc:
        print(json.dumps({"status": "BLOCKED", "code": exc.code, "message": str(exc)}, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
