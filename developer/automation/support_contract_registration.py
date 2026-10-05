from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import uuid

from jsonschema import Draft202012Validator
from referencing import Registry, Resource
import yaml

ROOT = Path(__file__).resolve().parents[2]
SCOPE_RECORD = "developer/policy/registries/vpms-contract-materialization.json"
SCOPE_SCHEMA = "developer/policy/schemas/vpms-contract-materialization.schema.json"
CONTRACT_ROOT = "src/vpms/contracts"


class RegistrationError(ValueError):
    """Stable fail-closed error for registration, not a runtime execution grant."""


def preserved_text_matches(raw: bytes, record: dict) -> bool:
    # Original raw sha256 remains provenance. Only Git text newline conversion
    # is equivalent here; this is not semantic canonicalization or a digest scheme.
    return hashlib.sha256(raw.replace(b"\r\n", b"\n")).hexdigest() == record["lf_sha256"]


def _pairs(items):
    result = {}
    for key, value in items:
        if key in result:
            raise RegistrationError("DUPLICATE_JSON_KEY: " + key)
        result[key] = value
    return result


def _json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"), object_pairs_hook=_pairs)


def _bounded(base: Path, relative: str) -> Path:
    candidate = Path(relative)
    if candidate.is_absolute() or ".." in candidate.parts or "\\" in relative:
        raise RegistrationError("UNSAFE_REGISTERED_PATH: " + relative)
    resolved = (base / candidate).resolve()
    if not resolved.is_relative_to(base.resolve()):
        raise RegistrationError("UNSAFE_REGISTERED_PATH: " + relative)
    return resolved


def scope_record(root: Path = ROOT):
    record = _json(root / SCOPE_RECORD)
    Draft202012Validator(_json(root / SCOPE_SCHEMA)).validate(record)
    return record


def preflight(root: Path = ROOT):
    record = scope_record(root)
    if "allocation" in record:
        # An existing allocated identity must be reused, never silently reallocated.
        return {"status": "ALLOCATED", **record["allocation"]}
    from developer.automation.policy_validator import validate_developer_policy
    from ptsip.governance import AuthorityCatalog

    failures = validate_developer_policy(root)
    if failures:
        raise RegistrationError("INVALID_SOURCE_CORPUS: " + "; ".join(failures))
    ids = AuthorityCatalog(root).validate_current_corpus()
    index_path = root / "src/policy/index.yaml"
    indexed = yaml.safe_load(index_path.read_text(encoding="utf-8"))["policies"]
    expected_paths = {entry["path"] for entry in indexed}
    discovered = {path.name for path in (root / "src/policy").glob("SFP-*.yaml")}
    if expected_paths != discovered:
        raise RegistrationError("UNINDEXED_SUPPORT_POLICY")
    numbers = []
    for identity in ids:
        if re.fullmatch(r"SFP-[0-9]{4}", identity) is None:
            raise RegistrationError("INVALID_SUPPORT_ID")
        numbers.append(int(identity[4:]))
    next_number = max(numbers, default=0) + 1
    if next_number > 9999:
        raise RegistrationError("SUPPORT_ID_NAMESPACE_EXHAUSTED")
    policy_id = f"SFP-{next_number:04d}"
    policy_path = f"src/policy/{policy_id}.yaml"
    if (root / policy_path).exists():
        raise RegistrationError("SUPPORT_ID_COLLISION")
    identities = {
        role: "urn:uuid:" + str(uuid.uuid4())
        for role in ("protocol", "selection", "execution_composition")
    }
    return {
        "status": "READY",
        "support_policy_id": policy_id,
        "support_policy_path": policy_path,
        "support_source_index_sha256": hashlib.sha256(index_path.read_bytes()).hexdigest(),
        "product_contract_ids": identities,
        "identity_allocation": "UUID_V4_OPAQUE_EXACT",
    }


def _resources(root: Path):
    base = root / CONTRACT_ROOT
    identity = _json(base / "identity-registry.json")
    resources = Registry().with_resource(
        identity["$id"], Resource.from_contents(identity)
    )
    schemas = {}
    for relative in identity["schema_resources"]:
        schema = _json(_bounded(base, relative))
        Draft202012Validator.check_schema(schema)
        resources = resources.with_resource(schema["$id"], Resource.from_contents(schema))
        schemas[relative] = schema
    catalog = _json(base / "index.json")
    Draft202012Validator(schemas["schemas/catalog.schema.json"], registry=resources).validate(catalog)
    return base, identity, catalog, schemas, resources


def inspect_contract(identity: str, root: Path = ROOT, *, require_active: bool = False):
    base, registry, catalog, schemas, resources = _resources(root)
    # Runtime-style resolution is exact and bounded; it never scans contract prose.
    entry = catalog["contracts"].get(identity)
    if entry is None:
        raise RegistrationError("UNKNOWN_CONTRACT_ID")
    payload = _json(_bounded(base, entry["path"]))
    Draft202012Validator(schemas["schemas/product-contract.schema.json"], registry=resources).validate(payload)
    for field in ("id", "status", "responsibility", "runtime_enabled"):
        expected = identity if field == "id" else entry[field]
        if payload[field] != expected:
            raise RegistrationError("CONTRACT_INDEX_MISMATCH: " + field)
    if payload["contract_class"] != registry["contract_class"]:
        raise RegistrationError("CONTRACT_CLASS_MISMATCH")
    if require_active and (
        catalog["capability"] != "ACTIVE"
        or payload["status"] != "ACTIVE"
        or not payload["runtime_enabled"]
    ):
        raise RegistrationError("CONTRACT_NOT_ACTIVE")
    return payload


def validate_selection_document(kind: str, payload, root: Path = ROOT):
    """Verify declared I/O data only; do not select Cases or execute adapters."""
    _, _, catalog, schemas, resources = _resources(root)
    semantics = inspect_contract(catalog["entrypoints"]["selection"], root)["semantics"]
    relative = {
        "request": "schemas/selection-request.schema.json",
        "result": "schemas/selection-result.schema.json",
        "rule": "schemas/selection-rule.schema.json",
    }.get(kind)
    if relative is None:
        raise RegistrationError("UNKNOWN_SELECTION_DOCUMENT")
    Draft202012Validator(schemas[relative], registry=resources).validate(payload)
    if kind == "result":
        if semantics["ordering"] != "CASE_ID_ASCENDING":
            raise RegistrationError("UNSUPPORTED_SELECTION_ORDER")
        if payload["case_ids"] != sorted(payload["case_ids"]):
            raise RegistrationError("NONDETERMINISTIC_SELECTION_ORDER")
        diagnostics = payload["diagnostics"]
        fields = semantics["diagnostic_order"]
        if fields != ["location", "code", "reference"]:
            raise RegistrationError("UNSUPPORTED_DIAGNOSTIC_ORDER")
        ordered = sorted(diagnostics, key=lambda item: (
            tuple(item.get(field, "") for field in fields)
        ))
        if diagnostics != ordered:
            raise RegistrationError("NONDETERMINISTIC_DIAGNOSTIC_ORDER")
    return {"status": "PASS", "kind": kind, "execution_performed": False}


def verify_registration(root: Path = ROOT, *, check_worktree: bool = False):
    from developer.automation.vpms_runtime_activation import activation_record, verify_preserved
    activation = activation_record(root)
    record = scope_record(root)
    allocation = record.get("allocation")
    if not allocation:
        raise RegistrationError("PREFLIGHT_ALLOCATION_REQUIRED")
    base, identity, catalog, schemas, resources = _resources(root)
    expected = allocation["product_contract_ids"]
    if catalog["entrypoints"] != expected or set(catalog["contracts"]) != set(expected.values()):
        raise RegistrationError("ALLOCATION_CATALOG_MISMATCH")
    if len(set(expected.values())) != 3:
        raise RegistrationError("CONTRACT_ID_COLLISION")
    if identity["contract_class"] != record["approved_design"]["contract_class"]:
        raise RegistrationError("UNAPPROVED_CONTRACT_CLASS")
    for role, contract_id in expected.items():
        payload = inspect_contract(contract_id, root, require_active=bool(activation))
        if payload["status"] != ("ACTIVE" if activation else "APPROVED") or payload["runtime_enabled"] is not bool(activation):
            raise RegistrationError("UNAUTHORIZED_RUNTIME_ACTIVATION")
        if payload["owner_approval"] != "USER_EXPLICIT":
            raise RegistrationError("OWNER_APPROVAL_REQUIRED")
        if set(payload["owns"]) & set(payload["excludes"]):
            raise RegistrationError("OWN_EXCLUDE_OVERLAP")
        if role == "protocol":
            naming = payload["semantics"]["identity"]
            if naming["expanded_identity"] != record["approved_design"]["expanded_identity"]:
                raise RegistrationError("UNAPPROVED_EXPANDED_IDENTITY")
            if naming["governing_question"] != record["approved_design"]["governing_question"]:
                raise RegistrationError("UNAPPROVED_GOVERNING_QUESTION")
        elif role == "selection":
            selection = payload["semantics"]
            for field, approved in (
                ("request_kinds", "selection_request_kinds"),
                ("rule_kind", "selection_rule_kind"),
                ("ordering", "ordering"),
                ("invalid_request", "duplicate_unknown_empty"),
                ("failure_atomicity", "failure_atomicity"),
                ("execution_side_effects", "selection_execution_side_effects"),
            ):
                if selection[field] != record["approved_design"][approved]:
                    raise RegistrationError("UNAPPROVED_SELECTION_SEMANTICS: " + field)
        if role == "execution_composition" and payload["responsibility"] != "VPMS_EXECUTION_COMPOSITION":
            raise RegistrationError("EXECUTION_COMPOSITION_OWNER_MISMATCH")
    invariants = _json(base / "invariants.json")
    if invariants["activation_requires_explicit_owner_approval"] is not True:
        raise RegistrationError("ACTIVATION_GATE_MISSING")
    for path in base.rglob("*.json"):
        raw = path.read_text(encoding="utf-8")
        if "MPD-" in raw or "developer/" in raw:
            raise RegistrationError("PRODUCT_DEVELOPER_POLICY_DEPENDENCY")
    from ptsip.governance import AuthorityCatalog

    support = AuthorityCatalog(root)
    support.validate_current_corpus()
    _, route, sfp = support.load_current_record(allocation["support_policy_id"])
    support_status = "ACTIVE" if activation else "DRAFT"
    if route["status"] != support_status or sfp["policy"]["status"] != support_status:
        raise RegistrationError("UNAUTHORIZED_SUPPORT_ACTIVATION")
    if sfp["feature_contract"]["runtime_surface"] != ["src/vpms/integration/ptsip_bridge.py"]:
        raise RegistrationError("INTEGRATION_SCOPE_EXPANDED")
    verify_preserved(root, record["preserved_files"])
    for relative in record["materialization_targets"]:
        if not _bounded(root, relative).is_file():
            raise RegistrationError("MISSING_SCOPE_TARGET: " + relative)
    if check_worktree:
        if activation:
            raise RegistrationError("ORIGINAL_SCOPE_SUPERSEDED_BY_ACTIVATION")
        head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        if head != record["base_head"]:
            raise RegistrationError("BASE_HEAD_CHANGED")
        changed = subprocess.check_output(
            ["git", "-c", "core.safecrlf=false", "diff", "--name-only", "HEAD"], cwd=root, text=True
        ).splitlines()
        changed += subprocess.check_output(
            ["git", "ls-files", "--others", "--exclude-standard"], cwd=root, text=True
        ).splitlines()
        carried = {
            item["path"] for item in record["preserved_files"]
            if item["path"].startswith("developer/")
        }
        if set(changed) != set(record["materialization_targets"]) | carried:
            raise RegistrationError("EXACT_CHANGE_SCOPE_MISMATCH")
    return {
        "status": "REGISTERED_ACTIVE" if activation else "REGISTERED_NON_ACTIVE", "support_policy_id": allocation["support_policy_id"],
        "support_status": support_status, "product_contract_count": 3,
        "product_status": "ACTIVE" if activation else "APPROVED", "runtime_enabled": bool(activation),
        "scope_target_count": len(record["materialization_targets"]),
        "preserved_file_count": len(record["preserved_files"]),
    }


def main():
    parser = argparse.ArgumentParser(description="Read-only approved contract registration preflight and verification.")
    parser.add_argument("--root", type=Path, default=ROOT)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("preflight")
    verification = commands.add_parser("verify")
    verification.add_argument("--check-worktree", action="store_true")
    inspection = commands.add_parser("inspect")
    inspection.add_argument("identity")
    inspection.add_argument("--require-active", action="store_true")
    args = parser.parse_args()
    try:
        if args.command == "preflight":
            result = preflight(args.root)
        elif args.command == "verify":
            result = verify_registration(args.root, check_worktree=args.check_worktree)
        else:
            result = inspect_contract(args.identity, args.root, require_active=args.require_active)
    except (OSError, ValueError, KeyError) as exc:
        parser.exit(1, str(exc) + "\n")
    print(json.dumps(result, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
