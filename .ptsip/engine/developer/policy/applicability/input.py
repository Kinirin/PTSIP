"""Read current indexed unified policies as exact JSON input for the Go
applicability assessor. Never mutate canonical policy files or the index.
This is deliberately separate from the historical migrate_unified.py.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import subprocess
import sys


# These paths cover the selected policy inputs, registered contract resources,
# and the shared validator that reads them from the working tree.
SOURCE_PATHS = (
    ".ptsip/index.json",
    ".ptsip/contracts",
    "developer/policy",
    "src/policy",
    "src/ptsip/policy_contracts",
)


def git(root: Path, *args: str) -> bytes:
    return subprocess.run(
        ["git", "--no-optional-locks", *args], cwd=root, check=True, capture_output=True,
    ).stdout


def require_source_snapshot(root: Path, head: str) -> None:
    changes = git(root, "status", "--porcelain=v1", "--untracked-files=all", "-z", "--", *SOURCE_PATHS)
    if changes:
        first_change = changes.split(b"\0", 1)[0].decode("utf-8", errors="replace")
        raise ValueError("SOURCE_SNAPSHOT_DIRTY: " + first_change)
    if git(root, "rev-parse", "HEAD").decode("ascii").strip() != head:
        raise ValueError("SOURCE_HEAD_CHANGED_DURING_ASSESSMENT")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path.cwd())
    args = parser.parse_args()
    root = args.root.resolve()
    sys.path.insert(0, str(root / "src"))
    from ptsip.policy_contracts.unified import load_document, validate_document, VERSION

    try:
        if Path(git(root, "rev-parse", "--show-toplevel").decode("utf-8").strip()).resolve() != root:
            raise ValueError("EXACT_REPOSITORY_ROOT_REQUIRED")
        head = git(root, "rev-parse", "HEAD").decode("ascii").strip()
        require_source_snapshot(root, head)

        def read_source(path: str) -> dict:
            return load_document(git(root, "show", "--no-textconv", head + ":" + path))

        registry_ref = ".ptsip/contracts/policy/unified/applicability-registry.json"
        unified_index = read_source(".ptsip/contracts/policy/unified/index.json")
        if unified_index.get("applicability_assessment_ref") != registry_ref:
            raise ValueError("APPLICABILITY_ASSESSOR_NOT_REGISTERED")
        registry = read_source(registry_ref)
        if (
            registry.get("schema_version") != "ptsip-policy-applicability-registration/v1"
            or registry.get("status") != "CANDIDATE_READ_ONLY"
            or registry.get("policy_mutation_authorized") is not False
            or registry.get("normative_policy_activation") is not False
            or registry.get("evaluation_authorized") is not False
            or registry.get("execution_authorized") is not False
        ):
            raise ValueError("APPLICABILITY_REGISTRY_NOT_CANDIDATE_READ_ONLY")
        contracts = {}
        for kind in ("RA", "RAC", "AEC"):
            ref = registry["contract_sources"][kind]
            if not isinstance(ref, str) or not ref.startswith(
                ".ptsip/contracts/compile-sealed-rbac/"
            ) or ".." in Path(ref).parts:
                raise ValueError("INVALID_CONTRACT_SOURCE: " + kind)
            contract = read_source(ref)
            contracts[kind] = {
                "source_ref": ref,
                "activation_state": contract["activation_state"],
                "definition_state": contract.get("definition_state", ""),
            }
        family_registry = read_source(".ptsip/contracts/policy/Policy_Rules/Family_Selection.json")
        families = set(family_registry["family_routes"])
        if len(families) != 14:
            raise ValueError("ROOT_FAMILY_REGISTRY_MISMATCH")
        policies = []
        seen = set()
        for plane, index_path in (
            ("PTSIP_DEVELOPER_POLICY", "developer/policy/index.yaml"),
            ("PTSIP_SUPPORT_FEATURE", "src/policy/index.yaml"),
        ):
            index = read_source(index_path)
            for entry in index["policies"]:
                policy_id = entry["id"]
                path = entry["path"]
                if plane == "PTSIP_SUPPORT_FEATURE":
                    path = "src/policy/" + path
                parts = Path(path).parts
                # Match the exact two-plane / fourteen-Family migration scope.
                # Root-level BOUND records belong to an independent owner.
                if len(parts) != 4 or parts[2] not in families:
                    continue
                if (parts[0] not in ("src", "developer") or parts[1] != "policy"
                        or not path.startswith(("src/policy/", "developer/policy/"))
                        or ".." in parts):
                    raise ValueError("POLICY_INDEX_PATH_OUTSIDE_SCOPE: " + path)
                if policy_id in seen:
                    raise ValueError("DUPLICATE_INDEXED_POLICY: " + policy_id)
                seen.add(policy_id)
                doc = read_source(path)
                if doc.get("schema_version") != VERSION:
                    raise ValueError("NON_UNIFIED_INDEXED_POLICY: " + path)
                identity = doc["Policy_Identity"]
                if identity["id"] != policy_id or identity["policy_kind"] != plane:
                    raise ValueError("POLICY_IDENTITY_MISMATCH: " + path)
                if doc["Policy_Content"]["source_document"]["path"] != path:
                    raise ValueError("POLICY_SOURCE_PATH_MISMATCH: " + path)
                result = validate_document(root, doc)
                if result.get("status") != "VALID":
                    raise ValueError("SOURCE_POLICY_CONTRACT_INVALID: " + path + ": " + str(result.get("problem")))
                policies.append({
                    "policy_id": policy_id,
                    "path": path,
                    "plane": plane,
                    "family": identity["root_family"],
                    "catalog_status": entry["status"],
                    "document_status": doc["Policy_Lifecycle"]["status"],
                    "rules": doc["Policy_Rules"]["rules"],
                    "items": [
                        {"id": item["id"], "source_pointer": item["source_pointer"]}
                        for item in doc["Policy_Content"]["items"]
                    ],
                })
        payload = {
            "schema_version": "ptsip-policy-applicability-input/v1",
            "source_head": head,
            "contracts": contracts,
            "policies": policies,
        }
        require_source_snapshot(root, head)
        print(json.dumps(payload, ensure_ascii=True, separators=(",", ":"), sort_keys=True))
        return 0
    except (ValueError, KeyError, OSError, subprocess.SubprocessError) as error:
        print("POLICY_APPLICABILITY_INPUT_FAILED: " + str(error), file=sys.stderr)
        return 2

if __name__ == "__main__":
    raise SystemExit(main())
