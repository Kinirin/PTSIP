"""Normalize the explicitly selected policy families into the common candidate.

The converter preserves the parsed source tree and its declared identities.
Contract publication and evaluator activation remain separate operations.
"""

from __future__ import annotations

import argparse
from collections import Counter
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile

import yaml


FAMILIES = (
    "ARCH", "ASSURE", "CHANGE", "CNTR", "CTRL", "GOV", "INFO",
    "INTENT", "NORM", "OPS", "REAL", "RECORD", "RISK", "SUPPLY",
)
PLANES = {"developer": "PTSIP_DEVELOPER_POLICY", "src": "PTSIP_SUPPORT_FEATURE"}
INVENTORY = ".ptsip/contracts/policy/unified/migration-inventory.json"
ARCH_DECISIONS = "developer/policy/ARCH/Policy_Rules_Decisions.yaml"
CANDIDATE_CONTRACTS = {
    "RAC": ".ptsip/contracts/compile-sealed-rbac/rac/rac.contract.json",
    "AEC": ".ptsip/contracts/compile-sealed-rbac/aec/aec.contract.json",
}


class UniqueLoader(yaml.SafeLoader):
    """Reject ambiguous source mappings before selecting or serializing content."""


def unique_mapping(loader, node, deep=False):
    loader.flatten_mapping(node)
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if not isinstance(key, str):
            raise ValueError("SOURCE_MAPPING_KEY_MUST_BE_STRING")
        if key in result:
            raise ValueError("DUPLICATE_SOURCE_KEY: " + key)
        result[key] = loader.construct_object(value_node, deep=deep)
    return result


UniqueLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, unique_mapping,
)


def parse_document(raw):
    value = yaml.load(raw.decode("utf-8-sig"), Loader=UniqueLoader)
    if not isinstance(value, dict):
        raise ValueError("SOURCE_DOCUMENT_MUST_BE_OBJECT")
    # A canonical JSON tree is the source-preservation contract's data model.
    json.dumps(value, ensure_ascii=True, allow_nan=False)
    return value


def canonical_digest(value):
    raw = json.dumps(
        value, ensure_ascii=True, sort_keys=True, separators=(",", ":"),
        allow_nan=False,
    ).encode("utf-8")
    return hashlib.sha256(raw).hexdigest()


def git(root, *arguments):
    result = subprocess.run(
        ["git", *arguments], cwd=root, capture_output=True, check=False,
    )
    if result.returncode:
        raise ValueError("GIT_COMMAND_FAILED: " + result.stderr.decode("utf-8", "replace"))
    return result.stdout


def child_pointer(parent, key):
    token = str(key).replace("~", "~0").replace("/", "~1")
    return parent + "/" + token


def native_identity(source):
    if "Policy_Identity" in source:
        identity = source["Policy_Identity"]
        lifecycle = source.get("Policy_Lifecycle", source.get("Policy_Status"))
        if not isinstance(lifecycle, dict):
            raise ValueError("NATIVE_LIFECYCLE_UNRESOLVED")
        status = lifecycle.get("status")
        if status is None:
            status = lifecycle.get("constant", {}).get("status")
        lifecycle_pointer = "/Policy_Lifecycle" if "Policy_Lifecycle" in source else "/Policy_Status"
        selected = {key: identity.get(key) for key in ("id", "policy_kind", "root_family", "title")}
    else:
        policy = source.get("policy")
        if not isinstance(policy, dict):
            raise ValueError("NATIVE_POLICY_IDENTITY_UNRESOLVED")
        selected = {
            "id": policy.get("id"),
            "policy_kind": source.get("policy_class"),
            "root_family": source.get("responsibility_family"),
            "title": policy.get("title"),
        }
        status = policy.get("status")
        lifecycle_pointer = "/policy"
    if any(not isinstance(value, str) or not value for value in selected.values()):
        raise ValueError("NATIVE_POLICY_IDENTITY_INCOMPLETE")
    if not isinstance(status, str) or not status:
        raise ValueError("NATIVE_POLICY_STATUS_UNRESOLVED")
    return selected, status, lifecycle_pointer


def partition_source(source, rule_units):
    """Keep each selected unit whole and retain other subtrees as source items."""
    selected = set(rule_units)
    items = []

    def visit(value, address):
        contains_unit = any(pointer.startswith(address + "/") for pointer in selected)
        numeric_mapping = isinstance(value, dict) and bool(value) and all(
            key.isdigit() for key in value
        )
        if address in selected or not contains_unit or numeric_mapping:
            items.append({
                "id": f"SRC-{len(items) + 1:04}",
                "source_pointer": address,
                "value": deepcopy(value),
            })
            return
        values = value.items() if isinstance(value, dict) else enumerate(value)
        for key, child in values:
            visit(child, child_pointer(address, key))

    for key, value in source.items():
        visit(value, child_pointer("", key))
    return items


def containing_item(items, address):
    matches = [
        item for item in items
        if address == item["source_pointer"]
        or address.startswith(item["source_pointer"] + "/")
    ]
    if len(matches) != 1:
        raise ValueError("SOURCE_ITEM_PARTITION_UNRESOLVED: " + address)
    return matches[0]


def metadata_references(items, roots, fields=()):
    refs = []
    for item in items:
        address = item["source_pointer"]
        value = item["value"]
        under_root = any(address == root or address.startswith(root + "/") for root in roots)
        declared_field = any(key in address.split("/") for key in fields) or (
            isinstance(value, dict) and any(key in value for key in fields)
        )
        if under_root or declared_field:
            refs.append(item["id"])
    return refs


def normalize(root, path, source, registration, baseline, version):
    from ptsip._common_contract import source_result

    identity, status, lifecycle_pointer = native_identity(source)
    units = {pointer: (unit["kind"], unit["value"]) for pointer, unit in
             source_result("source-rule-units", "units", document=source).items()}
    items = partition_source(source, units)
    rules = []
    seen = set()
    for address, (kind, value) in units.items():
        item = containing_item(items, address)
        declared = None
        if isinstance(value, dict):
            declared = next((value[key] for key in ("id", "ID", "requirement_id") if key in value), None)
        if declared is not None and (not isinstance(declared, str) or not declared):
            raise ValueError("INVALID_DECLARED_RULE_ID: " + address)
        rule_id = declared or identity["id"] + "#" + item["source_pointer"]
        if rule_id in seen:
            raise ValueError("DUPLICATE_RULE_ID: " + rule_id)
        seen.add(rule_id)
        bindings = {
            key: {"state": "UNRESOLVED", "basis_refs": [item["id"]]}
            for key in ("RA", "RAC", "AEC")
        }
        if identity["id"] == "SFP-ARCH-0004" and address == "/authority_semantics/rac_aec_rego_original_contract_direction":
            for key, reference in CANDIDATE_CONTRACTS.items():
                definition = json.loads((root / reference).read_bytes())
                if definition.get("activation_state") != "UNREGISTERED":
                    raise ValueError("CANDIDATE_REFERENCE_STATE_CHANGED: " + reference)
                if json.loads(git(root, "show", baseline + ":" + reference)) != definition:
                    raise ValueError("CANDIDATE_REFERENCE_NOT_AT_BASELINE: " + reference)
                bindings[key]["contract_ref"] = {
                    "source_ref": reference,
                    "item_ref": "#",
                    "revision_ref": "git:" + baseline,
                }
        rules.append({
            "id": rule_id,
            "identity_scheme": "DECLARED" if declared else "SOURCE_COORDINATE",
            "kind": kind,
            "source_ref": item["id"],
            "contract_bindings": bindings,
        })
    source_format = source.get("schema_version")
    if source_format is None:
        source_format = "PUBLIC_POLICY_V2" if "contracts" in source.get("Policy_Rules", {}) else "LEGACY_POLICY_COMPONENTS"
    return {
        "schema_version": version,
        "Policy_Identity": identity,
        "Policy_Lifecycle": {"status": status, "source_ref": containing_item(items, lifecycle_pointer)["id"]},
        "Policy_Authority": {"source_refs": metadata_references(
            items, ("/Authority", "/authority_subject", "/authority_contract"), ("authority_ref",),
        )},
        "Policy_Rules": {"rules": rules},
        "Policy_Evaluation": {
            "source_refs": metadata_references(
                items, ("/Policy_Rules/Evaluation",), ("judgement_binding", "evidence_requirement_binding"),
            ),
            "state": "UNRESOLVED",
        },
        "Policy_Relationships": {"source_refs": metadata_references(
            items, ("/relations",), ("relation_binding",),
        )},
        "Policy_Content": {
            "items": items,
            "source_document": {"path": path, "sha256": canonical_digest(source)},
        },
        "Policy_Migration": {
            "baseline_head": baseline,
            "source_format": source_format,
            "source_registration": registration,
            "contract_state": "CANDIDATE",
        },
    }


def discover(root, verify_current=False):
    def in_scope(path):
        parts = Path(path).parts
        return (
            len(parts) == 4
            and parts[0] in PLANES
            and parts[1] == "policy"
            and parts[2] in FAMILIES
            and Path(path).suffix in (".yaml", ".yml")
        )

    indexes = {}
    for plane in PLANES:
        index = parse_document((root / plane / "policy/index.yaml").read_bytes())
        for entry in index["policies"]:
            path = entry["path"] if plane == "developer" else "src/policy/" + entry["path"]
            if verify_current and in_scope(path) and not path.startswith(plane + "/policy/"):
                raise ValueError("POLICY_INDEX_PLANE_MISMATCH: " + path)
            if path in indexes:
                raise ValueError("DUPLICATE_POLICY_ROUTE: " + path)
            indexes[path] = entry["id"]
    files = git(root, "ls-files", "-z", "--", "developer/policy", "src/policy").decode("utf-8").split("\0")
    selected = {path for path in files if in_scope(path)}
    if verify_current:
        for plane in PLANES:
            for family in FAMILIES:
                directory = root / plane / "policy" / family
                if directory.exists():
                    selected.update(
                        path.relative_to(root).as_posix()
                        for path in directory.iterdir()
                        if path.suffix in (".yaml", ".yml") and path.is_file()
                    )
    missing = sorted({path for path in indexes if in_scope(path)} - selected)
    if missing:
        problem = "INDEXED_POLICY_NOT_ON_DISK" if verify_current else "INDEXED_POLICY_NOT_TRACKED"
        raise ValueError(problem + ": " + ", ".join(missing))
    return sorted(selected), indexes


def atomic_write(path, raw):
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(dir=path.parent, prefix="." + path.name + ".", delete=False) as stream:
            temporary = Path(stream.name)
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temporary, stat.S_IMODE(path.stat().st_mode) if path.exists() else 0o644)
        os.replace(temporary, path)
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


def run(root, apply, verify_current=False):
    if verify_current and apply:
        raise ValueError("CURRENT_POLICY_VERIFICATION_IS_READ_ONLY")
    root = root.resolve()
    actual = Path(git(root, "rev-parse", "--show-toplevel").decode("utf-8").strip()).resolve()
    if actual != root:
        raise ValueError("EXACT_REPOSITORY_ROOT_REQUIRED")
    sys.path.insert(0, str(root / "src"))
    from ptsip._common_contract import UNIFIED_VERSION, request, source_result

    baseline = git(root, "rev-parse", "HEAD").decode("ascii").strip()
    paths, indexes = discover(root, verify_current)
    original_bytes = {}
    changes = {}
    documents = []
    auxiliary = []
    totals = Counter()
    binding_states = Counter()
    kinds = Counter()
    family_counts = {}
    policy_ids = set()
    for path in paths:
        absolute = root / path
        if verify_current and not absolute.resolve().is_relative_to(root):
            raise ValueError("POLICY_PATH_OUTSIDE_REPOSITORY: " + path)
        raw = absolute.read_bytes()
        parsed = parse_document(raw)
        unified = parsed.get("schema_version") == UNIFIED_VERSION
        if verify_current and not unified:
            auxiliary_decision = (
                path == ARCH_DECISIONS
                and path not in indexes
                and parsed.get("contract_type") == "PTSIP_ARCH_POLICY_RULES_DECISIONS"
                and "Policy_Identity" not in parsed
                and "policy" not in parsed
            )
            if not auxiliary_decision:
                raise ValueError("CURRENT_POLICY_NOT_UNIFIED: " + path)
        if not unified and not isinstance(parsed.get("Policy_Identity", parsed.get("policy")), dict):
            if path in indexes:
                raise ValueError("INDEXED_POLICY_IDENTITY_MISSING: " + path)
            auxiliary.append({"path": path, "classification": "AUXILIARY_NON_POLICY"})
            continue
        original_bytes[path] = raw
        source = source_result("restore-source", "source", document=parsed) if unified else parsed
        identity, _, _ = native_identity(source)
        parts = Path(path).parts
        if identity["policy_kind"] != PLANES[parts[0]] or identity["root_family"] != parts[2]:
            raise ValueError("SOURCE_IDENTITY_SCOPE_MISMATCH: " + path)
        registration = "INDEXED" if path in indexes else "UNINDEXED"
        if registration == "INDEXED" and indexes[path] != identity["id"]:
            raise ValueError("SOURCE_INDEX_IDENTITY_MISMATCH: " + path)
        if verify_current and identity["id"] in policy_ids:
            raise ValueError("DUPLICATE_CURRENT_POLICY_ID: " + identity["id"])
        policy_ids.add(identity["id"])
        candidate = parsed if unified else normalize(root, path, source, registration, baseline, UNIFIED_VERSION)
        result = request("validate-unified", repository_root=str(root), document=candidate, restore=True)
        if result.get("status") != "VALID":
            raise ValueError("CANDIDATE_VALIDATION_FAILED: " + path + ": " + json.dumps(result, ensure_ascii=True))
        if candidate["Policy_Content"]["source_document"]["path"] != path:
            raise ValueError("CANDIDATE_SOURCE_PATH_MISMATCH: " + path)
        if verify_current and candidate["Policy_Migration"]["source_registration"] != registration:
            raise ValueError("CANDIDATE_SOURCE_REGISTRATION_MISMATCH: " + path)
        if result.get("source") != source:
            raise ValueError("SOURCE_RESTORATION_CHANGED: " + path)
        if candidate["Policy_Content"]["source_document"]["sha256"] != canonical_digest(source):
            raise ValueError("SOURCE_DIGEST_CHANGED: " + path)
        if not unified:
            header = "# yaml-language-server: $schema=../../../.ptsip/contracts/policy/unified/Unified_Policy.schema.json\n"
            serialized = (header + yaml.safe_dump(candidate, allow_unicode=True, sort_keys=False, width=120)).encode("utf-8")
            if parse_document(serialized) != candidate:
                raise ValueError("SERIALIZATION_CHANGED_DOCUMENT: " + path)
            changes[path] = serialized
        rules = candidate["Policy_Rules"]["rules"]
        rule_kinds = Counter(rule["kind"] for rule in rules)
        kinds.update(rule_kinds)
        states = Counter(binding["state"] for rule in rules for binding in rule["contract_bindings"].values())
        binding_states.update(states)
        totals[identity["policy_kind"]] += 1
        totals["rules"] += len(rules)
        totals["source_items"] += len(candidate["Policy_Content"]["items"])
        totals[registration] += 1
        family_key = (identity["policy_kind"], identity["root_family"])
        family_totals = family_counts.setdefault(family_key, Counter())
        family_totals["policy_documents"] += 1
        family_totals["converted" if path in changes and apply else "ready" if path in changes else "already_unified"] += 1
        family_totals.update(rule_kinds)
        documents.append({
            "path": path,
            "policy_id": identity["id"],
            "policy_kind": identity["policy_kind"],
            "root_family": identity["root_family"],
            "status": candidate["Policy_Lifecycle"]["status"],
            "source_format": candidate["Policy_Migration"]["source_format"],
            "source_registration": candidate["Policy_Migration"]["source_registration"],
            "source_sha256": canonical_digest(source),
            "source_items": len(candidate["Policy_Content"]["items"]),
            "rules": len(rules),
            "rule_kinds": dict(rule_kinds),
            "contract_binding_states": dict(states),
            "candidate_definition_references": sum("contract_ref" in binding for rule in rules for binding in rule["contract_bindings"].values()),
            "source_preservation": "EXACT_PARSED_JSON_TREE",
            "validation": result["status"],
            "action": "CONVERTED" if path in changes and apply else "READY" if path in changes else "ALREADY_UNIFIED",
        })
    inventory = {
        "format": "ptsip-unified-policy-migration-inventory/v1",
        "baseline_head": baseline,
        "schema_version": UNIFIED_VERSION,
        "contract_state": "CANDIDATE",
        "scope": {"planes": list(PLANES.values()), "families": list(FAMILIES), "family_directories": len(PLANES) * len(FAMILIES)},
        "summary": {"policy_documents": len(documents), "changed_documents": len(changes), "auxiliary_documents": len(auxiliary), **dict(totals)},
        "contract_binding_states": dict(binding_states),
        "rule_kinds": dict(kinds),
        "families": [
            {"policy_kind": plane, "root_family": family, **dict(counts)}
            for (plane, family), counts in sorted(family_counts.items())
        ],
        "source_preservation": "EXACT_PARSED_JSON_TREE",
        "normative_activation": False,
        "documents": documents,
        "auxiliary": auxiliary,
    }
    if apply:
        report_path = root / INVENTORY
        before_report = report_path.read_bytes() if report_path.exists() else None
        if not report_path.parent.is_dir():
            raise ValueError("CANDIDATE_CONTRACT_DIRECTORY_MISSING")
        for path, before in original_bytes.items():
            if (root / path).read_bytes() != before:
                raise ValueError("SOURCE_CHANGED_BEFORE_APPLY: " + path)
        written = []
        try:
            for path, raw in changes.items():
                atomic_write(root / path, raw)
                written.append(path)
            if changes or before_report is None:
                atomic_write(report_path, (json.dumps(inventory, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))
        except BaseException:
            for path in reversed(written):
                atomic_write(root / path, original_bytes[path])
            if before_report is None:
                report_path.unlink(missing_ok=True)
            else:
                atomic_write(report_path, before_report)
            raise
    return inventory


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path.cwd())
    operation = parser.add_mutually_exclusive_group()
    operation.add_argument("--check", action="store_true", help="prepare and validate all documents without writing")
    operation.add_argument("--verify-current", action="store_true", help="validate the candidate documents currently on disk")
    operation.add_argument("--apply", action="store_true", help="write validated candidates and the migration inventory")
    args = parser.parse_args()
    try:
        report = run(args.root, args.apply, args.verify_current)
    except (ValueError, OSError, TypeError, yaml.YAMLError) as error:
        print(json.dumps({"status": "FAIL", "problem": str(error)}, ensure_ascii=True))
        return 2
    name = "APPLY" if args.apply else "VERIFY_CURRENT" if args.verify_current else "CHECK"
    print(json.dumps({"status": "VALID", "operation": name, "summary": report["summary"], "contract_binding_states": report["contract_binding_states"], "inventory": INVENTORY if args.apply else None}, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
