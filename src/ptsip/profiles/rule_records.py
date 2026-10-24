"""Read-only discovery, validation and exact resolution for linked rule profiles.

The record hash segment is an opaque, immutable identity component here.
Prefix propagation and reference rewriting belong to a separate implementation.
"""
from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any, Mapping

import yaml
from jsonschema import Draft202012Validator

from .catalog import LocalProfileCatalogError, load_local_profile_selection


CATALOG_REF = ".ptsip/profiles/index.json"
PROFILE_REF = ".ptsip/profiles/rules.ptsip.yaml"
SCHEMA_REF = ".ptsip/contracts/schemas/Policy_Rules/rule_record.schema.json"

_ROOT = re.compile(r"^rule_sha256:[a-f0-9]{64}$")
_CHILD = re.compile(r"^rule_([A-Z]{1,4})sha256:[a-f0-9]{64}$")
_GRANDCHILD = re.compile(r"^rule_([A-Z]{1,4})\.([A-Z]{1,4})sha256:[a-f0-9]{64}$")
_RESOURCE = re.compile(r"^[a-z][a-z0-9-]*\.ptsip\.yaml$")


class RuleProfileError(ValueError):
    """Invalid declared rule profile or its graph; fail closed."""


def _object_file(root: Path, relative: str, *, yaml_format: bool = False) -> Mapping[str, Any]:
    allowed = {CATALOG_REF, PROFILE_REF, SCHEMA_REF}
    if relative not in allowed:
        raise RuleProfileError(f"unregistered rule resource: {relative!r}")
    path = root.joinpath(*relative.split("/"))
    if not path.is_file() or path.is_symlink():
        raise RuleProfileError(f"missing or nonregular rule resource: {relative}")
    try:
        text = path.read_text(encoding="utf-8-sig")
        parsed = yaml.safe_load(text) if yaml_format else json.loads(text)
    except (OSError, UnicodeError, yaml.YAMLError, ValueError) as exc:
        raise RuleProfileError(f"invalid rule resource {relative}: {exc}") from exc
    if not isinstance(parsed, dict):
        raise RuleProfileError(f"rule resource must be an object: {relative}")
    return parsed


def _linked_resource(root: Path, main_payload: Mapping[str, Any] | None) -> str | None:
    if main_payload is None:
        try:
            selection = load_local_profile_selection(root)
        except LocalProfileCatalogError as exc:
            raise RuleProfileError(f"local profile catalog: {exc}") from exc
        if selection is None:
            return None
        try:
            main_payload = yaml.safe_load(selection.path.read_text(encoding="utf-8-sig"))
        except (OSError, UnicodeError, yaml.YAMLError) as exc:
            raise RuleProfileError(f"unreadable default project profile: {exc}") from exc
    if not isinstance(main_payload, Mapping):
        raise RuleProfileError("main profile must be an object")
    ptsip = main_payload.get("ptsip")
    if not isinstance(ptsip, Mapping):
        return None
    selected = ptsip.get("rule_profile")
    if selected is None:
        return None
    if not isinstance(selected, str) or not _RESOURCE.fullmatch(selected):
        raise RuleProfileError("rule_profile must be one relative *.ptsip.yaml filename")
    if selected != "rules.ptsip.yaml":
        raise RuleProfileError(f"unregistered rule profile resource: {selected}")
    catalog = _object_file(root, CATALOG_REF)
    profiles = catalog.get("profiles")
    if not isinstance(profiles, list):
        raise RuleProfileError("local profile catalog profiles must be an array")
    matches = [
        row for row in profiles
        if isinstance(row, dict) and row.get("id") == "rules" and row.get("resource") == selected
    ]
    if len(matches) != 1 or sum(
        isinstance(row, dict) and (row.get("id") == "rules" or row.get("resource") == selected)
        for row in profiles
    ) != 1:
        raise RuleProfileError("linked rules profile missing, ambiguous or conflicts with catalog")
    return PROFILE_REF


def _kind(identity: str) -> str:
    if _ROOT.fullmatch(identity):
        return "root"
    if _CHILD.fullmatch(identity):
        return "child"
    if _GRANDCHILD.fullmatch(identity):
        return "grandchild"
    raise RuleProfileError(f"unknown rule record identity: {identity!r}")


def _validate_relations(records: list[dict[str, Any]]) -> None:
    by_id: dict[str, dict[str, Any]] = {}
    for record in records:
        identity = record["ID"]
        if identity in by_id:
            raise RuleProfileError(f"duplicate rule ID: {identity}")
        by_id[identity] = record
    for record in records:
        identity = record["ID"]
        kind = _kind(identity)
        parent_ref = record.get("parent_id")
        if kind == "root":
            if parent_ref is not None:
                raise RuleProfileError(f"root cannot have parent: {identity}")
        else:
            parent = by_id.get(parent_ref)
            if parent is None:
                raise RuleProfileError(f"missing parent for {identity}: {parent_ref}")
            if _kind(parent_ref) != ("root" if kind == "child" else "child"):
                raise RuleProfileError(f"wrong parent depth for {identity}: {parent_ref}")
            if kind == "child":
                match = _CHILD.fullmatch(identity)
                if match is None or match.group(1) != parent["rule_value"]["constant"]:
                    raise RuleProfileError(f"child prefix must equal parent constant: {identity}")
            else:
                ancestor_ref = parent["parent_id"]
                ancestor = by_id.get(ancestor_ref)
                if ancestor is None or _kind(ancestor_ref) != "root":
                    raise RuleProfileError(f"missing grandparent for {identity}")
                match = _GRANDCHILD.fullmatch(identity)
                if match is None or (
                    match.group(1) != ancestor["rule_value"]["constant"]
                    or match.group(2) != parent["rule_value"]["constant"]
                ):
                    raise RuleProfileError(f"grandchild prefix disagrees with ancestors: {identity}")
        pointer = record["rule_value"].get("root_id")
        if pointer is not None:
            target = by_id.get(pointer)
            if target is None or target.get("parent_id") != identity:
                raise RuleProfileError(f"root_id is not a direct child of {identity}: {pointer}")
            if _kind(pointer) != ("child" if kind == "root" else "grandchild"):
                raise RuleProfileError(f"root_id target level invalid for {identity}")


def load_rule_records(
    repository_root: str | Path,
    *,
    main_payload: Mapping[str, Any] | None = None,
) -> list[dict[str, Any]]:
    """Read a linked rules profile, validate its schema and exact relation graph."""
    root = Path(repository_root).resolve()
    resource = _linked_resource(root, main_payload)
    if resource is None:
        return []
    schema = _object_file(root, SCHEMA_REF)
    payload = _object_file(root, resource, yaml_format=True)
    try:
        Draft202012Validator.check_schema(schema)
        errors = sorted(
            Draft202012Validator(schema).iter_errors(payload),
            key=lambda item: (tuple(str(v) for v in item.absolute_path), item.message),
        )
    except Exception as exc:
        raise RuleProfileError(f"invalid rule record schema: {exc}") from exc
    if errors:
        first = errors[0]
        location = ".".join(str(v) for v in first.absolute_path) or "<root>"
        raise RuleProfileError(f"rule record schema rejected at {location}: {first.message}")
    records = payload["rules"]
    _validate_relations(records)
    return records


def discover_rule_records(
    repository_root: str | Path,
    *,
    main_payload: Mapping[str, Any] | None = None,
) -> list[str]:
    """List exact registered IDs without inventing or rehashing identities."""
    return sorted(record["ID"] for record in load_rule_records(repository_root, main_payload=main_payload))


def resolve_rule_record(
    repository_root: str | Path,
    rule_id: str,
    *,
    main_payload: Mapping[str, Any] | None = None,
) -> dict[str, Any]:
    """Resolve an existing ID; a read result is not a new policy authority."""
    records = load_rule_records(repository_root, main_payload=main_payload)
    found = next((entry for entry in records if entry["ID"] == rule_id), None)
    result = {
        "rule_id": rule_id,
        "normative_authority": False,
        "execution_authorized": False,
        "identity_rewritten": False,
    }
    if found is None:
        return {**result, "status": "UNRESOLVED", "reason": "UNKNOWN_EXACT_RULE_ID"}
    return {**result, "status": "RESOLVED", "source_ref": PROFILE_REF, "record": found}
