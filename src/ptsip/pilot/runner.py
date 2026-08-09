from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from ..inspection.inventory import collect_inventory
from ..repository.discover import discover_repository
from ..spec_identity import current_spec_identity
from ..storage.local_state import pilot_directory


@dataclass(frozen=True)
class PilotResult:
    report: dict[str, object]
    report_path: Path


def run_pilot(path: str | Path = ".", report_path: str | Path | None = None) -> PilotResult:
    repo = discover_repository(path)
    inventory = collect_inventory(repo.root)
    spec = current_spec_identity()
    report = {
        "format": "ptsip-pilot-report/v1",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "tool": {"version": spec.tool_version},
        "specification": {
            "name": spec.name,
            "version": spec.version,
            "source": spec.source,
            "revision": spec.revision,
        },
        "repository": repo.as_dict(),
        "inventory": inventory.as_dict(),
        "classification": {
            "status": "evidence-only",
            "note": "PTSIP 0.1 tooling does not automatically assert architectural ownership.",
        },
        "consumer_repository_modified": False,
    }

    if report_path is None:
        destination = pilot_directory(repo.root) / "report.json"
    else:
        destination = Path(report_path).expanduser().resolve()
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return PilotResult(report=report, report_path=destination)
