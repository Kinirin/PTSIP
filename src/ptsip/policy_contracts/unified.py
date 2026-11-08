"""Validate the common candidate policy envelope and restore its exact source.

The candidate envelope records declared rule bindings. Evaluation registration
and execution remain separate decisions owned by their registered sources.
"""

from __future__ import annotations

import argparse
from copy import deepcopy
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys
from typing import Any, Mapping

import yaml


VERSION = "ptsip-unified-policy/v3-candidate"
INDEX = ".ptsip/contracts/policy/unified/index.json"
SCHEMA = ".ptsip/contracts/policy/unified/Unified_Policy.schema.json"
SCHEMA_ID = "urn:ptsip:candidate:policy:unified:v3"
EVALUATION_REGISTRY = ".ptsip/contracts/policy/unified/evaluation-registry.json"
EVALUATION_INTERFACE = ".ptsip/contracts/policy/unified/Evaluation_Interface.json"
AUTHORING_BASIS = ".ptsip/contracts/policy/unified/authoring-decision.json"
KINDS = ("RA", "RAC", "AEC")
_SHA40 = re.compile(r"^[0-9a-f]{40}$")
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_ARRAY_INDEX = re.compile(r"^(0|[1-9][0-9]*)$")
_MAX_BYTES = 16 * 1024 * 1024


class PolicyContractError(ValueError):
    """A deterministic structural, source-integrity or registration failure."""


class _UniqueLoader(yaml.SafeLoader):
    pass


def _mapping(loader: _UniqueLoader, node: yaml.MappingNode, deep: bool = False) -> dict:
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if not isinstance(key, str):
            raise PolicyContractError("MAPPING_KEY_MUST_BE_STRING")
        if key in result:
            raise PolicyContractError("DUPLICATE_MAPPING_KEY: " + key)
        result[key] = loader.construct_object(value_node, deep=deep)
    return result


_UniqueLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, _mapping)


def load_document(raw: bytes | str) -> dict:
    """Decode UTF-8 JSON/YAML with duplicate-key rejection."""
    if isinstance(raw, bytes):
        if len(raw) > _MAX_BYTES:
            raise PolicyContractError("POLICY_DOCUMENT_TOO_LARGE")
        raw = raw.decode("utf-8-sig")
    if len(raw.encode("utf-8")) > _MAX_BYTES:
        raise PolicyContractError("POLICY_DOCUMENT_TOO_LARGE")
    try:
        result = yaml.load(raw, Loader=_UniqueLoader)
    except yaml.YAMLError as error:
        raise PolicyContractError("INVALID_POLICY_YAML") from error
    if not isinstance(result, dict):
        raise PolicyContractError("POLICY_DOCUMENT_MUST_BE_OBJECT")
    return result


def _child_pointer(parent: str, key: object) -> str:
    return parent + "/" + str(key).replace("~", "~0").replace("/", "~1")


def source_rule_units(source: dict) -> dict[str, tuple[str, object]]:
    """Select explicit source rule boundaries for migration and coverage checks."""
    units = {}
    policy_rules = source.get("Policy_Rules", {})
    if not isinstance(policy_rules, dict):
        raise PolicyContractError("NATIVE_POLICY_RULES_MUST_BE_OBJECT")
    for contract_index, contract in enumerate(policy_rules.get("contracts", [])):
        base = f"/Policy_Rules/contracts/{contract_index}"
        for field, kind in (("rules", "RULE"), ("requirements", "REQUIREMENT")):
            for item_index, value in enumerate(contract.get(field, [])):
                units[f"{base}/{field}/{item_index}"] = (kind, value)
    for item_index, value in enumerate(policy_rules.get("Definition", [])):
        units[f"/Policy_Rules/Definition/{item_index}"] = ("RULE", value)
    for item_index, value in enumerate(policy_rules.get("Boundary", {}).get("subjects", [])):
        units[f"/Policy_Rules/Boundary/subjects/{item_index}"] = ("REQUIREMENT", value)
    for section in ("rules", "authority_semantics"):
        semantics = source.get(section, {})
        if not isinstance(semantics, dict):
            raise PolicyContractError("NATIVE_SEMANTICS_MUST_BE_OBJECT: " + section)
        for name, value in semantics.items():
            if name == "family_definition":
                for item_index, definition in enumerate(value.get("kernel_definitions", [])):
                    address = f"/{section}/family_definition/kernel_definitions/{item_index}"
                    units[address] = ("RULE", definition)
            else:
                units[_child_pointer("/" + section, name)] = ("SEMANTIC_UNIT", value)
    return units


def is_unified(document: object) -> bool:
    return isinstance(document, Mapping) and document.get("schema_version") == VERSION


def _canonical(value: object) -> bytes:
    try:
        return json.dumps(
            value, ensure_ascii=True, sort_keys=True, separators=(",", ":"), allow_nan=False
        ).encode("utf-8")
    except (ValueError, TypeError, RecursionError) as error:
        raise PolicyContractError("SOURCE_MUST_BE_FINITE_JSON") from error


def _tokens(pointer: str) -> list[str]:
    if pointer in ("", "#"):
        return []
    if not isinstance(pointer, str) or not pointer.startswith("/"):
        raise PolicyContractError("INVALID_SOURCE_POINTER")
    tokens = []
    for token in pointer[1:].split("/"):
        if re.search(r"~(?![01])", token):
            raise PolicyContractError("INVALID_SOURCE_POINTER_ESCAPE")
        tokens.append(token.replace("~1", "/").replace("~0", "~"))
    return tokens


def _point(value: Any, pointer: str) -> Any:
    for token in _tokens(pointer):
        if isinstance(value, dict) and token in value:
            value = value[token]
        elif isinstance(value, list) and _ARRAY_INDEX.fullmatch(token):
            index = int(token)
            if index >= len(value):
                raise PolicyContractError("SOURCE_POINTER_NOT_FOUND: " + pointer)
            value = value[index]
        else:
            raise PolicyContractError("SOURCE_POINTER_NOT_FOUND: " + pointer)
    return value


def restore_source(document: dict) -> dict:
    """Restore the complete original JSON tree from disjoint source items.

    Arrays use contiguous canonical indices. An object containing exclusively
    numeric keys must be retained as a complete item by the serializer.
    """
    if not is_unified(document):
        raise PolicyContractError("UNSUPPORTED_UNIFIED_POLICY_VERSION")
    content = document.get("Policy_Content", {})
    items = content.get("items")
    if not isinstance(items, list) or not items:
        raise PolicyContractError("SOURCE_ITEMS_REQUIRED")
    tree: dict = {}
    ids = set()
    for item in items:
        if not isinstance(item, dict) or not isinstance(item.get("id"), str) or not item["id"]:
            raise PolicyContractError("INVALID_SOURCE_ITEM")
        if item["id"] in ids:
            raise PolicyContractError("DUPLICATE_SOURCE_ITEM_ID: " + item["id"])
        ids.add(item["id"])
        if "value" not in item:
            raise PolicyContractError("SOURCE_ITEM_VALUE_REQUIRED")
        node = tree
        for token in _tokens(item.get("source_pointer")):
            if "value" in node:
                raise PolicyContractError("OVERLAPPING_SOURCE_POINTERS")
            node = node.setdefault("children", {}).setdefault(token, {})
        if "value" in node or node.get("children"):
            raise PolicyContractError("DUPLICATE_OR_OVERLAPPING_SOURCE_POINTER")
        node["value"] = deepcopy(item["value"])

    def rebuild(node: dict) -> Any:
        if "value" in node:
            return node["value"]
        children = node.get("children", {})
        if not children:
            raise PolicyContractError("INCOMPLETE_SOURCE_PARTITION")
        if all(_ARRAY_INDEX.fullmatch(key) for key in children):
            if set(children) != {str(index) for index in range(len(children))}:
                raise PolicyContractError("NONCONTIGUOUS_SOURCE_ARRAY")
            return [rebuild(children[str(index)]) for index in range(len(children))]
        return {key: rebuild(value) for key, value in children.items()}

    source = rebuild(tree)
    if not isinstance(source, dict):
        raise PolicyContractError("RESTORED_SOURCE_MUST_BE_OBJECT")
    expected = content.get("source_document", {}).get("sha256")
    if not isinstance(expected, str) or not _SHA256.fullmatch(expected):
        raise PolicyContractError("SOURCE_DIGEST_REQUIRED")
    actual = hashlib.sha256(_canonical(source)).hexdigest()
    if expected != actual:
        raise PolicyContractError("SOURCE_DIGEST_MISMATCH")
    return source


def _relative(path: str) -> str:
    if not isinstance(path, str) or not path:
        raise PolicyContractError("INVALID_LOCAL_RESOURCE_PATH")
    parsed = PurePosixPath(path)
    if (parsed.is_absolute() or str(parsed) != path or ".." in parsed.parts
            or any(character in path for character in "\\:\x00\r\n")):
        raise PolicyContractError("INVALID_LOCAL_RESOURCE_PATH: " + path)
    if path.casefold().startswith("developer/temporary_record/"):
        raise PolicyContractError("TEMPORARY_POLICY_DEPENDENCY_FORBIDDEN")
    return path


def _local(root: Path, relative: str) -> Path:
    relative = _relative(relative)
    current = root
    for part in relative.split("/"):
        current = current / part
        if current.is_symlink():
            raise PolicyContractError("SYMLINK_POLICY_RESOURCE: " + relative)
    if not current.resolve().is_relative_to(root):
        raise PolicyContractError("ESCAPING_POLICY_RESOURCE: " + relative)
    if not current.is_file():
        raise PolicyContractError("MISSING_POLICY_RESOURCE: " + relative)
    return current


def _resource(root: Path, relative: str) -> dict:
    return load_document(_local(root, relative).read_bytes())


def _registration(root: Path) -> tuple[dict, dict, dict]:
    entry = _resource(root, ".ptsip/index.json")
    namespace = entry.get("namespaces", {}).get("contracts", {})
    if namespace.get("status") != "ACTIVE" or namespace.get("index") != "contracts/index.json":
        raise PolicyContractError("CONTRACT_NAMESPACE_UNRESOLVED")
    catalog = _resource(root, ".ptsip/contracts/index.json")
    boundary = _resource(root, ".ptsip/contracts/" + str(catalog.get("policy", "")))
    if boundary.get("domain") != "policy" or boundary.get("unified_binding", {}).get("index") != INDEX:
        raise PolicyContractError("UNIFIED_POLICY_ROUTE_UNRESOLVED")
    registration = _resource(root, INDEX)
    required = {
        "schema_version": "ptsip-unified-policy-registration/v1",
        "document_schema_version": VERSION,
        "status": "CANDIDATE",
        "schema_ref": SCHEMA,
        "schema_id": SCHEMA_ID,
        "evaluation_registry_ref": EVALUATION_REGISTRY,
        "evaluation_interface_ref": EVALUATION_INTERFACE,
        "authoring_basis_ref": AUTHORING_BASIS,
        "execution_authorized": False,
        "publication_authorized": False,
        "structural_authoring_registered": True,
        "normative_activation": False,
    }
    if any(registration.get(key) != value for key, value in required.items()):
        raise PolicyContractError("UNIFIED_POLICY_REGISTRATION_MISMATCH")
    if registration.get("policy_kinds") != ["PTSIP_DEVELOPER_POLICY", "PTSIP_SUPPORT_FEATURE"]:
        raise PolicyContractError("UNIFIED_POLICY_KIND_REGISTRATION_MISMATCH")
    _resource(root, AUTHORING_BASIS)
    interface = _resource(root, EVALUATION_INTERFACE)
    if (interface.get("status") != "CANDIDATE" or interface.get("registered_engine") is not False
            or interface.get("runtime_activation_authorized") is not False
            or interface.get("execution_authorized") is not False):
        raise PolicyContractError("CANDIDATE_EVALUATION_INTERFACE_CHANGED")
    schema = _resource(root, SCHEMA)
    if schema.get("$id") != SCHEMA_ID:
        raise PolicyContractError("UNIFIED_POLICY_SCHEMA_ID_MISMATCH")

    def local_refs(value: object) -> None:
        if isinstance(value, dict):
            if "$ref" in value and (not isinstance(value["$ref"], str) or not value["$ref"].startswith("#/")):
                raise PolicyContractError("UNREGISTERED_UNIFIED_SCHEMA_REFERENCE")
            for child in value.values():
                local_refs(child)
        elif isinstance(value, list):
            for child in value:
                local_refs(child)

    local_refs(schema)
    evaluation = _resource(root, EVALUATION_REGISTRY)
    if (evaluation.get("status") != "UNREGISTERED" or evaluation.get("entries") != []
            or evaluation.get("execution_authorized") is not False
            or evaluation.get("evaluation_authorized") is not False):
        raise PolicyContractError("CANDIDATE_EVALUATION_REGISTRATION_CHANGED")
    return registration, schema, evaluation


def _identity(source: dict) -> dict:
    identity = source.get("Policy_Identity")
    if isinstance(identity, dict):
        return {key: identity.get(key) for key in ("id", "policy_kind", "root_family", "title")}
    policy = source.get("policy", {})
    return {
        "id": policy.get("id"),
        "policy_kind": source.get("policy_class"),
        "root_family": source.get("responsibility_family"),
        "title": policy.get("title"),
    }


def _status(source: object) -> str | None:
    if not isinstance(source, dict):
        return None
    if "status" in source:
        return source["status"]
    constant = source.get("constant")
    if isinstance(constant, dict) and "status" in constant:
        return constant["status"]
    for key in ("Policy_Lifecycle", "Policy_Status", "policy"):
        value = source.get(key)
        if isinstance(value, dict):
            return _status(value)
    return None


def restore_support_source(document: dict, relative: str) -> dict:
    """Project the original Support record for distribution without candidate bindings."""
    source = restore_source(document)
    identity = _identity(source)
    if (identity != document.get("Policy_Identity")
            or identity["policy_kind"] != "PTSIP_SUPPORT_FEATURE"
            or any(not isinstance(value, str) or not value for value in identity.values())):
        raise PolicyContractError("SUPPORT_PROJECTION_IDENTITY_MISMATCH: " + relative)
    expected = f"{identity['root_family']}/{identity['id']}.yaml"
    content = document["Policy_Content"]
    if relative != expected or _relative(content["source_document"].get("path")) != "src/policy/" + expected:
        raise PolicyContractError("SUPPORT_PROJECTION_PATH_MISMATCH: " + relative)

    status = _status(source)
    lifecycle = document.get("Policy_Lifecycle", {})
    lifecycle_pointer = "/policy"
    if "Policy_Identity" in source:
        lifecycle_pointer = "/Policy_Lifecycle" if "Policy_Lifecycle" in source else "/Policy_Status"
    lifecycle_item = next((item for item in content["items"] if item["id"] == lifecycle.get("source_ref")), {})
    if (not isinstance(status, str) or not status or lifecycle.get("status") != status
            or lifecycle_item.get("source_pointer") != lifecycle_pointer):
        raise PolicyContractError("SUPPORT_PROJECTION_LIFECYCLE_MISMATCH: " + relative)
    return source


def _declared_id(value: object) -> str | None:
    if isinstance(value, dict):
        for key in ("id", "ID", "requirement_id"):
            if key in value:
                identifier = value[key]
                if not isinstance(identifier, str) or not identifier:
                    raise PolicyContractError("INVALID_SOURCE_RULE_ID")
                return identifier
    return None


def _git_reference(root: Path, reference: dict) -> dict:
    path = _relative(reference["source_ref"])
    normalized = path.casefold()
    if (normalized.startswith((".ptsip/contracts/policy/rules/", ".ptsip/contracts/schemas/"))
            or PurePosixPath(normalized).name == "responsibilities_list.schema.json"):
        raise PolicyContractError("RETIRED_CONTRACT_REFERENCE_FORBIDDEN")
    revision = reference["revision_ref"]
    if not revision.startswith("git:") or not _SHA40.fullmatch(revision[4:]):
        raise PolicyContractError("EXACT_GIT_REFERENCE_REQUIRED")
    result = subprocess.run(
        ["git", "show", "--no-textconv", revision[4:] + ":" + path],
        cwd=root, capture_output=True, timeout=15, check=False,
    )
    if result.returncode or len(result.stdout) > _MAX_BYTES:
        raise PolicyContractError("PINNED_CONTRACT_REFERENCE_UNAVAILABLE")
    selected = _point(load_document(result.stdout), reference["item_ref"])
    if not isinstance(selected, dict):
        raise PolicyContractError("DECLARED_CONTRACT_REFERENCE_MUST_BE_OBJECT")
    return selected


def assess_registration(root: str | Path, rule: dict, kind: str) -> dict:
    """Return the candidate binding's registration state without an evaluation."""
    if kind not in KINDS:
        raise PolicyContractError("UNKNOWN_RULE_CONTRACT_KIND")
    _registration(Path(root).resolve())
    binding = rule["contract_bindings"][kind]
    state = binding["state"]
    if state not in ("DECLARED", "UNRESOLVED", "NOT_APPLICABLE"):
        raise PolicyContractError("UNREGISTERED_CONTRACT_BINDING_STATE")
    if state == "DECLARED":
        _git_reference(Path(root).resolve(), binding["contract_ref"])
        result = "UNREGISTERED"
    else:
        result = state
    return {"kind": kind, "state": result, "evaluation_authorized": False, "execution_authorized": False}


def validate_document(root: str | Path, document: dict) -> dict:
    from jsonschema import Draft202012Validator, SchemaError

    report = {
        "status": "FAIL",
        "scope": "STRUCTURE_SOURCE_INTEGRITY_AND_DECLARED_REFERENCES",
        "policy_conformance": "NOT_ASSERTED",
        "mutation_authorized": False,
        "evaluation_authorized": False,
        "execution_authorized": False,
        "publication_authorized": False,
    }
    try:
        root = Path(root).resolve()
        _, schema, evaluation = _registration(root)
        Draft202012Validator.check_schema(schema)
        errors = sorted(Draft202012Validator(schema).iter_errors(document), key=lambda error: str(list(error.path)))
        if errors:
            error = errors[0]
            raise PolicyContractError("UNIFIED_SCHEMA_REJECTED: " + "/".join(map(str, error.path)) + ": " + error.message)
        source = restore_source(document)
        identity = document["Policy_Identity"]
        if identity != _identity(source):
            raise PolicyContractError("SOURCE_POLICY_IDENTITY_MISMATCH")
        if document["Policy_Lifecycle"]["status"] != _status(source):
            raise PolicyContractError("SOURCE_POLICY_STATUS_MISMATCH")
        source_document = document["Policy_Content"]["source_document"]
        path = _relative(source_document["path"])
        expected_root = "developer/policy/" if identity["policy_kind"] == "PTSIP_DEVELOPER_POLICY" else "src/policy/"
        if not path.startswith(expected_root) or path.split("/")[2] != identity["root_family"]:
            raise PolicyContractError("POLICY_SOURCE_OUTSIDE_DECLARED_PLANE_OR_FAMILY")
        items = {item["id"]: item for item in document["Policy_Content"]["items"]}

        def require_refs(references: list[str]) -> None:
            if any(reference not in items for reference in references):
                raise PolicyContractError("UNRESOLVED_POLICY_SOURCE_ITEM")

        lifecycle_ref = document["Policy_Lifecycle"]["source_ref"]
        require_refs([lifecycle_ref])
        if _status(items[lifecycle_ref]["value"]) != _status(source):
            value = items[lifecycle_ref]["value"]
            if not (isinstance(value, dict) and value.get("status") == _status(source)) and value != _status(source):
                raise PolicyContractError("LIFECYCLE_SOURCE_ITEM_MISMATCH")
        for key in ("Policy_Authority", "Policy_Evaluation", "Policy_Relationships"):
            require_refs(document[key]["source_refs"])
        rules = document["Policy_Rules"]["rules"]
        expected_units = source_rule_units(source)
        represented_units = set()
        rule_ids = set()
        declared_bindings = 0
        for rule in rules:
            if rule["id"] in rule_ids:
                raise PolicyContractError("DUPLICATE_POLICY_RULE_ID")
            rule_ids.add(rule["id"])
            require_refs([rule["source_ref"]])
            item = items[rule["source_ref"]]
            address = item["source_pointer"]
            if address not in expected_units or address in represented_units:
                raise PolicyContractError("UNREGISTERED_OR_DUPLICATE_RULE_SOURCE")
            represented_units.add(address)
            kind, original = expected_units[address]
            if rule["kind"] != kind or item["value"] != original:
                raise PolicyContractError("POLICY_RULE_SOURCE_UNIT_MISMATCH")
            declared = _declared_id(original)
            if declared:
                if rule["identity_scheme"] != "DECLARED" or rule["id"] != declared:
                    raise PolicyContractError("DECLARED_RULE_ID_SOURCE_MISMATCH")
            elif rule["identity_scheme"] != "SOURCE_COORDINATE" or rule["id"] != identity["id"] + "#" + address:
                raise PolicyContractError("SCOPED_RULE_COORDINATE_MISMATCH")
            for binding in rule["contract_bindings"].values():
                require_refs(binding["basis_refs"])
                if "contract_ref" in binding:
                    _git_reference(root, binding["contract_ref"])
                if binding["state"] == "DECLARED":
                    declared_bindings += 1
        if represented_units != set(expected_units):
            raise PolicyContractError("INCOMPLETE_POLICY_RULE_COVERAGE")
        report.update(
            status="VALID", policy_id=identity["id"], source_sha256=source_document["sha256"],
            source_items=len(items), rule_count=len(rules),
            evaluation_registration={"status": evaluation["status"], "entry_count": 0, "declared_bindings": declared_bindings},
        )
    except (ValueError, TypeError, KeyError, OSError, RecursionError, AttributeError, SchemaError, subprocess.SubprocessError) as error:
        report["problem"] = str(error)
    return report


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".")
    parser.add_argument("--input", required=True)
    arguments = parser.parse_args()
    try:
        root = Path(arguments.root).resolve()
        raw = sys.stdin.buffer.read(_MAX_BYTES + 1) if arguments.input == "-" else _local(root, arguments.input).read_bytes()
        report = validate_document(root, load_document(raw))
    except (ValueError, TypeError, KeyError, OSError) as error:
        report = {"status": "FAIL", "problem": str(error), "execution_authorized": False, "mutation_authorized": False}
    print(json.dumps(report, ensure_ascii=True, indent=2))
    return 0 if report["status"] == "VALID" else 2


if __name__ == "__main__":
    raise SystemExit(main())
