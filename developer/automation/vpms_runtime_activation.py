"""Bounded developer approval/provenance audit, never product runtime authority."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import subprocess

from jsonschema import Draft202012Validator

ROOT = Path(__file__).resolve().parents[2]
RECORD = "developer/policy/registries/vpms-runtime-activation.json"
SCHEMA = "developer/policy/schemas/vpms-runtime-activation.schema.json"


def activation_record(root: Path = ROOT):
    path = root / RECORD
    if not path.is_file():
        return None
    record = json.loads(path.read_text(encoding="utf-8"))
    Draft202012Validator(json.loads((root / SCHEMA).read_text(encoding="utf-8"))).validate(record)
    for relative, expected in record["prior_records"].items():
        raw = (root / relative).read_bytes().replace(b"\r\n", b"\n")
        if hashlib.sha256(raw).hexdigest() != expected:
            raise ValueError("ACTIVATION_PRIOR_PROVENANCE_CHANGED: " + relative)
    return record


def preflight(root: Path = ROOT):
    record = activation_record(root)
    if record is None:
        raise ValueError("EXPLICIT_ACTIVATION_APPROVAL_REQUIRED")
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    if head != record["base_head"]:
        raise ValueError("ACTIVATION_BASE_HEAD_CHANGED")
    from .support_contract_registration import _resources, inspect_contract
    from ptsip.governance import AuthorityCatalog
    _, _, catalog, _, _ = _resources(root)
    for identity in catalog["entrypoints"].values():
        contract = inspect_contract(identity, root)
        if contract["status"] != "APPROVED" or contract["runtime_enabled"]:
            raise ValueError("ACTIVATION_PREIMAGE_NOT_APPROVED_INACTIVE")
    support = AuthorityCatalog(root)
    support.validate_current_corpus()
    for identity, expected in (("SFP-0006", "ACTIVE"), ("SFP-0023", "DRAFT")):
        _, _, policy = support.load_current_record(identity)
        if policy["policy"]["status"] != expected:
            raise ValueError("ACTIVATION_SUPPORT_PREIMAGE_MISMATCH")
    return {"status": "READY", "base_head": head, "approved_target_count": len(record["mutation_targets"])}


def verify_preserved(root: Path, preserved):
    """Honor exact successor changes, retaining all other original byte guards."""
    from .support_contract_registration import preserved_text_matches
    record = activation_record(root)
    changes = record["authorized_preservation_changes"] if record else {}
    for item in preserved:
        relative = item["path"]
        operation = changes.get(relative)
        from developer.automation.policy_loader import registered_policy_file
        path = registered_policy_file(relative, root=root)
        if operation == "DELETE":
            if path.exists():
                raise ValueError("RETIRED_SOURCE_PRESENT: " + relative)
            continue
        raw = path.read_bytes()
        if operation == "MODIFY_LIFECYCLE_ONLY":
            raw = raw.replace(b"status: RETIRED", b"status: ACTIVE", 1)
        elif operation == "MODIFY":
            continue
        if not preserved_text_matches(raw, item):
            raise ValueError("PRESERVED_SOURCE_CHANGED: " + relative)


def verify(root: Path = ROOT, *, check_worktree: bool = False):
    record = activation_record(root)
    if record is None:
        raise ValueError("EXPLICIT_ACTIVATION_APPROVAL_REQUIRED")
    from .support_contract_registration import verify_registration
    from .vpms_api_implementation import verify as verify_api
    from ptsip.governance import AuthorityCatalog
    registered = verify_registration(root)
    implementation = verify_api(root)
    support = AuthorityCatalog(root)
    for identity, expected in record["support_targets"].items():
        _, route, policy = support.load_current_record(identity)
        if route["status"] != expected or policy["policy"]["status"] != expected:
            raise ValueError("ACTIVATION_SUPPORT_STATE_MISMATCH: " + identity)
    _, _, boundary = support.load_current_record("SFP-0023")
    if boundary["authority_semantics"]["input"]["concrete_provider_binding"] != record["provider_binding"]:
        raise ValueError("ACTIVATION_PROVIDER_BINDING_MISMATCH")
    if (root / "src/vpms/domain/selector.py").exists():
        raise ValueError("RETIRED_SOURCE_PRESENT: src/vpms/domain/selector.py")
    import ast
    for relative in ("src/vpms/__init__.py", "src/vpms/execution/runner.py"):
        tree = ast.parse((root / relative).read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            if isinstance(node, (ast.Name, ast.FunctionDef, ast.AsyncFunctionDef)):
                name = node.id if isinstance(node, ast.Name) else node.name
                if name in {"SelectionScope", "select_cases", "run_selected_cases"}:
                    raise ValueError("RETIRED_API_PRESENT: " + name)
    for relative in record["mutation_targets"]:
        if relative != "src/vpms/domain/selector.py" and not (root / relative).is_file():
            raise ValueError("MISSING_ACTIVATION_TARGET: " + relative)
    if check_worktree:
        head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        if head != record["base_head"]:
            raise ValueError("ACTIVATION_BASE_HEAD_CHANGED")
        changed = subprocess.check_output(["git", "diff", "--name-only", "HEAD"], cwd=root, text=True).splitlines()
        changed += subprocess.check_output(["git", "ls-files", "--others", "--exclude-standard"], cwd=root, text=True).splitlines()
        if set(changed) != set(record["mutation_targets"]):
            raise ValueError("ACTIVATION_EXACT_SCOPE_MISMATCH: " + str(sorted(set(changed) ^ set(record["mutation_targets"]))))
    return {"status": "ACTIVE_VERIFIED", "product_contract_count": registered["product_contract_count"],
            "runtime_enabled": registered["runtime_enabled"], "support_targets": record["support_targets"],
            "implementation_status": implementation["status"], "approved_target_count": len(record["mutation_targets"])}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=["preflight", "verify"])
    parser.add_argument("--check-worktree", action="store_true")
    args = parser.parse_args()
    result = preflight() if args.operation == "preflight" else verify(check_worktree=args.check_worktree)
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main()
