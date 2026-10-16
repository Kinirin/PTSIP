"""Independent routing and identity verifier; legacy lifecycle remains untouched."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator, FormatChecker

ROUTES_PATH = "developer/contracts/routing/policy_routes.json"
ROUTING_SCHEMA_PATH = "developer/contracts/schemas/Policy_Identity/Routing/Policy_Routing.schema.json"
IDENTITY_SCHEMA_PATH = "developer/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json"


class ContractError(Exception):
    def __init__(self, code: str, detail: str):
        self.code = code
        super().__init__(f"{code}: {detail}")


def repository_path(root: Path, relative: str, code: str) -> Path:
    if not isinstance(relative, str) or not relative or "\\" in relative:
        raise ContractError(code, "invalid repository-relative path")
    raw = Path(relative)
    if raw.is_absolute() or any(part in (".", "..") for part in raw.parts) or ":" in raw.parts[0]:
        raise ContractError(code, f"unsafe path: {relative}")
    resolved = (root / raw).resolve()
    if not resolved.is_relative_to(root):
        raise ContractError(code, f"path outside repository: {relative}")
    return resolved


def read_document(root: Path, path: str, code: str, *, yaml_allowed: bool = False):
    absolute = repository_path(root, path, code)
    try:
        with absolute.open(encoding="utf-8") as f:
            return yaml.safe_load(f) if yaml_allowed and absolute.suffix in (".yaml", ".yml") else json.load(f)
    except (OSError, ValueError, yaml.YAMLError) as exc:
        raise ContractError(code, f"{path}: {exc}") from exc


def check_schema_instance(schema: dict, instance: object, code: str) -> None:
    try:
        Draft202012Validator.check_schema(schema)
        violation = next(iter(Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(instance)), None)
    except Exception as exc:
        raise ContractError(code, f"schema invalid: {exc}") from exc
    if violation is not None:
        raise ContractError(code, f"invalid instance: {violation.message}")


def read_routing(root: Path) -> tuple[dict, set[str]]:
    routes = read_document(root, ROUTES_PATH, "ROUTING_CONTRACT_INVALID")
    schema = read_document(root, ROUTING_SCHEMA_PATH, "ROUTING_SCHEMA_INVALID")
    check_schema_instance(schema, routes, "ROUTING_CONTRACT_INVALID")
    source_ref = routes["identity_sources"]["root_family"]
    source = read_document(root, source_ref, "IDENTITY_SOURCE_INVALID")
    if not isinstance(source, dict) or set(source) != {"schema_version", "root_families"}:
        raise ContractError("IDENTITY_SOURCE_INVALID", "unexpected root_family.json structure")
    if source["schema_version"] != "ptsip-root-family/v1":
        raise ContractError("IDENTITY_SOURCE_INVALID", "unsupported root family source version")
    families = source["root_families"]
    if (not isinstance(families, list) or len(families) != 14
            or any(not isinstance(v, str) or not v or not v.isupper() for v in families)
            or len(set(families)) != len(families)):
        raise ContractError("IDENTITY_SOURCE_INVALID", "expected 14 unique nonempty Root Family tokens")
    allowed = set(families)
    for kind, mapping in routes["routes"].items():
        for family in mapping:
            if family not in allowed:
                raise ContractError("INVALID_ROOT_FAMILY", f"{kind}/{family}")
    return routes, allowed


def resolve_route(routes: dict, kind: str, family: str, allowed: set[str]) -> dict:
    if family not in allowed:
        raise ContractError("INVALID_ROOT_FAMILY", family)
    selected = routes["routes"].get(kind, {}).get(family)
    if selected is None:
        raise ContractError("ROUTE_UNREGISTERED", f"{kind}/{family}")
    return selected


def inspect_references(root: Path, selected: dict) -> None:
    # Readability and JSON Schema syntax only; recursive $ref resolution is out of scope.
    for label, ref in selected["schema_refs"].items():
        document = read_document(root, ref, "SCHEMA_REF_UNRESOLVED")
        if not isinstance(document, dict):
            raise ContractError("SCHEMA_REF_UNRESOLVED", f"{label}: not a schema object")
        try:
            Draft202012Validator.check_schema(document)
        except Exception as exc:
            raise ContractError("SCHEMA_REF_UNRESOLVED", f"{label}: {exc}") from exc


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate the independent Policy Identity routing contract.")
    parser.add_argument("--repository", default=".", help="Repository root")
    parser.add_argument("--identity", help="Repository-relative policy JSON/YAML containing Policy_Identity, or bare identity object")
    parser.add_argument("--policy-kind", help="Select route by policy kind")
    parser.add_argument("--root-family", help="Select route by Root Family")
    parser.add_argument("--check-schema-files", action="store_true", help="Check referenced schema files exist and are valid JSON Schema")
    args = parser.parse_args()
    root = Path(args.repository).resolve()
    try:
        routes, families = read_routing(root)
        result = {
            "status": "ROUTING_STRUCTURE_VALID",
            "registered_routes": sum(len(subroutes) for subroutes in routes["routes"].values()),
            "policy_creation_authorized": False,
        }
        kind, family = args.policy_kind, args.root_family
        if args.identity:
            doc = read_document(root, args.identity, "IDENTITY_SOURCE_INVALID", yaml_allowed=True)
            identity = doc.get("Policy_Identity", doc) if isinstance(doc, dict) else doc
            schema = read_document(root, IDENTITY_SCHEMA_PATH, "IDENTITY_SCHEMA_INVALID")
            check_schema_instance(schema, identity, "POLICY_IDENTITY_INVALID")
            family_from_identity = identity["root_family"]
            if family_from_identity not in families:
                raise ContractError("INVALID_ROOT_FAMILY", family_from_identity)
            if kind and kind != identity["policy_kind"] or family and family != family_from_identity:
                raise ContractError("POLICY_IDENTITY_INVALID", "identity and selected route disagree")
            kind, family = identity["policy_kind"], family_from_identity
            result["identity_validation"] = "VALID"
        if bool(kind) != bool(family):
            raise ContractError("ROUTE_UNREGISTERED", "supply both --policy-kind and --root-family")
        if kind and family:
            chosen = resolve_route(routes, kind, family, families)
            if args.check_schema_files:
                inspect_references(root, chosen)
            result["status"] = "ROUTE_REGISTERED"
            result["route"] = {"policy_kind": kind, "root_family": family, "output": chosen["output"]}
            result["schema_file_check"] = "PASSED_NON_RECURSIVE" if args.check_schema_files else "NOT_REQUESTED"
        elif args.check_schema_files:
            for by_family in routes["routes"].values():
                for entry in by_family.values():
                    inspect_references(root, entry)
            result["schema_file_check"] = "PASSED_NON_RECURSIVE"
        print(json.dumps(result, ensure_ascii=False, sort_keys=True))
        return 0
    except ContractError as exc:
        print(json.dumps({"status": "FAILED", "code": exc.code, "detail": str(exc), "policy_creation_authorized": False}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
