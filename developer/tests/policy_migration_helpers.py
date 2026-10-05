"""Keep historical responsibility assertions while checking actual Root routing.

The archived binding input is test evidence only. The real resolver must first
return the exact Root IDs, paths and units compiled from that input; only then
does the helper expose the old responsibility labels to existing assertions.
"""
import json
from pathlib import Path

from developer.automation.policy_loader import registered_policy_file
from developer.automation.policy_resolver import resolve_policies as resolve_root_policies
from ptsip.governance.authority import migration_registry

ROOT = Path(__file__).resolve().parents[2]


def source_file(path: Path) -> Path:
    return registered_policy_file(path, root=ROOT)


def resolve_source_bindings(repository, *, scope, operation):
    result = resolve_root_policies(repository, scope=scope, operation=operation)
    root = Path(repository)
    evidence = root / "developer/policy/analysis/root-family-transition-bindings.jsonl"
    if not evidence.is_file():
        return result
    rows = [json.loads(line) for line in evidence.read_text(encoding="utf-8").splitlines()]
    binding = next(row for row in rows if row["scope"] == result["binding_scope"])
    refs = binding.get("operations", {}).get(operation.upper(), binding.get("default_refs"))
    graph = migration_registry(root / "developer/policy", "PTSIP_DEVELOPER_POLICY")
    sources = {source["source_policy_id"]: source for source in graph["sources"]}
    grouped = {}
    old = []
    for ref in refs:
        source = sources[ref["policy_id"]]
        assert source["source_status"] == "ACTIVE"
        for section in ref["sections"]:
            pointer = "/rules/" + section
            units = [u for u in source["units"] if u["source_pointer"] == pointer or u["source_pointer"].startswith(pointer + "/")]
            assert units
            for unit in units:
                entry = grouped.setdefault(unit["policy_id"], {"policy_id": unit["policy_id"], "path": "developer/policy/" + unit["policy_path"], "status": "ACTIVE", "sections": []})
                if unit["section"] not in entry["sections"]:
                    entry["sections"].append(unit["section"])
        old.append({"policy_id": ref["policy_id"], "path": "developer/policy/" + source["original_path"], "status": "ACTIVE", "sections": ref["sections"]})
    expected = list(grouped.values())
    neutral = [p for p in result["policies"] if p["policy_id"] in {"MPD-REAL-0004", "MPD-CNTR-0003"}]
    if neutral:
        assert operation.upper() == "MODIFY"
        assert neutral == [
            {"policy_id": "MPD-REAL-0004", "path": "developer/policy/REAL/MPD-REAL-0004.yaml", "status": "ACTIVE", "sections": ["neutral_module_creation"]},
            {"policy_id": "MPD-CNTR-0003", "path": "developer/policy/CNTR/MPD-CNTR-0003.yaml", "status": "ACTIVE", "sections": ["root_family_projection_module"]},
        ]
        expected.extend(neutral)
    assert result["policies"] == expected, "Root selection changed source responsibility coverage"
    return {**result, "policies": old}
