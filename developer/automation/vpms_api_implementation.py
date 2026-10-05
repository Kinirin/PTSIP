"""Developer-only scope/provenance audit; not product runtime authority."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import subprocess

from jsonschema import Draft202012Validator

from .support_contract_registration import preserved_text_matches, verify_registration

ROOT = Path(__file__).resolve().parents[2]
RECORD = "developer/policy/registries/vpms-api-implementation.json"
SCHEMA = "developer/policy/schemas/vpms-api-implementation.schema.json"


def verify(root: Path = ROOT, *, check_worktree: bool = False):
    from .vpms_runtime_activation import activation_record, verify_preserved
    activation = activation_record(root)
    record = json.loads((root / RECORD).read_text(encoding="utf-8"))
    Draft202012Validator(json.loads((root / SCHEMA).read_text(encoding="utf-8"))).validate(record)
    prior_path = root / record["prior_materialization_ref"]
    if hashlib.sha256(prior_path.read_bytes().replace(b"\r\n", b"\n")).hexdigest() != record["prior_materialization_sha256"]:
        raise ValueError("PRIOR_MATERIALIZATION_CHANGED")
    prior = json.loads(prior_path.read_text(encoding="utf-8"))
    verify_registration(root)
    verify_preserved(root, record["preserved_files"])
    # Keep the original scope as provenance, but honor only exact, approved
    # subsequent retirements. Missing live implementation targets still fail.
    retired = {item["path"] for item in record.get("subsequent_retirements", [])}
    if not retired <= set(record["mutation_targets"]):
        raise ValueError("RETIREMENT_OUTSIDE_ORIGINAL_SCOPE")
    for relative in retired:
        if (root / relative).exists():
            raise ValueError("RETIRED_IMPLEMENTATION_TARGET_PRESENT: " + relative)
    for relative in set(record["mutation_targets"]) - retired:
        if not (root / relative).is_file():
            raise ValueError("MISSING_IMPLEMENTATION_TARGET: " + relative)
    for relative in record["removed_uncommitted_paths"]:
        if (root / relative).exists():
            raise ValueError("RETIRED_WRITE_PATH_PRESENT")
    if record["support_policy_generation_root"] != "src/policy":
        raise ValueError("INVALID_SUPPORT_POLICY_WRITE_ROOT")
    for binding in record["bindings"]:
        role = binding["contract"]
        relative = {"protocol": "protocol.json", "selection": "selection.json",
                    "execution_composition": "execution-composition.json"}[role]
        payload = json.loads((root / "src/vpms/contracts" / relative).read_text(encoding="utf-8"))
        found = [item for item in payload["planned_bindings"] if item["api"] == binding["api"]]
        if len(found) != 1 or found[0]["source"] != binding["source"] or found[0]["availability"] != ("IMPLEMENTED_ACTIVE" if activation else "IMPLEMENTED_INACTIVE"):
            raise ValueError("IMPLEMENTATION_BINDING_MISMATCH")
    expected = set(prior["materialization_targets"]) | set(record["mutation_targets"])
    expected |= {item["path"] for item in record["preserved_files"] if item["path"].startswith("developer/")}
    if check_worktree:
        if activation:
            raise ValueError("ORIGINAL_SCOPE_SUPERSEDED_BY_ACTIVATION")
        if retired:
            raise ValueError("ORIGINAL_WORKTREE_SCOPE_SUPERSEDED_BY_RETIREMENT")
        head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        if head != record["base_head"]:
            raise ValueError("BASE_HEAD_CHANGED")
        changed = subprocess.check_output(["git", "-c", "core.safecrlf=false", "diff", "--name-only", "HEAD"], cwd=root, text=True).splitlines()
        changed += subprocess.check_output(["git", "ls-files", "--others", "--exclude-standard"], cwd=root, text=True).splitlines()
        if set(changed) != expected:
            raise ValueError("EXACT_CHANGE_SCOPE_MISMATCH: " + str(sorted(set(changed) ^ expected)))
    return {"status": "IMPLEMENTED_ACTIVE" if activation else "IMPLEMENTED_INACTIVE", "runtime_activation": bool(activation),
            "commit_push_authorized": True, "implementation_target_count": len(record["mutation_targets"]),
            "combined_changed_path_count": len(expected), "preserved_file_count": len(record["preserved_files"]),
            "retired_target_count": len(retired)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check-worktree", action="store_true")
    args = parser.parse_args()
    print(json.dumps(verify(check_worktree=args.check_worktree), sort_keys=True))


if __name__ == "__main__":
    main()
