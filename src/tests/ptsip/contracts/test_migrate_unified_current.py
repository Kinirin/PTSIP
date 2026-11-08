import json
from pathlib import Path
from runpy import run_path
import subprocess
import sys

import yaml


ROOT = Path(__file__).resolve().parents[4]


def test_current_migration_roundtrip_is_read_only_and_covers_registered_policies():
    tool = ROOT / ".ptsip/engine/developer/policy/migrate_unified.py"
    families = set(run_path(str(tool))["FAMILIES"])
    indexed = 0
    for plane in ("src", "developer"):
        index = yaml.safe_load((ROOT / plane / "policy/index.yaml").read_bytes())
        for entry in index["policies"]:
            path = Path(entry["path"])
            if plane == "developer":
                path = path.relative_to("developer/policy")
            if len(path.parts) == 2 and path.parent.name in families and path.suffix in {".yaml", ".yml"}:
                indexed += 1
    run = subprocess.run([sys.executable, str(tool), "--root", str(ROOT), "--verify-current"],
                         cwd=ROOT, capture_output=True, check=True, timeout=180)
    report = json.loads(run.stdout)
    assert report["status"] == "VALID"
    assert report["operation"] == "VERIFY_CURRENT"
    assert report["summary"]["INDEXED"] == indexed
    assert report["summary"]["policy_documents"] >= indexed
    assert report["summary"]["changed_documents"] == 0
    assert sum(report["contract_binding_states"].values()) == report["summary"]["rules"] * 3
