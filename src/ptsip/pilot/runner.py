from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

import yaml

from ..inspection.components import discover_component_candidates
from ..inspection.dependencies import scan_dependency_edges
from ..inspection.inventory import collect_inventory
from ..model import Classification, DecisionStatus
from ..repository.discover import discover_repository
from ..repository.snapshot import capture_snapshot, compare_snapshots
from ..spec_identity import current_spec_identity
from ..storage.local_state import pilot_directory
from ..validation.components import partition_components
from ..validation.profile import find_profile, validate_profile
from ..validation.rules import evaluate_declared_dependency_boundaries


@dataclass(frozen=True)
class PilotResult:
    report: dict[str, object]
    report_path: Path


def _inside(root: Path, candidate: Path) -> bool:
    try:
        candidate.relative_to(root)
        return True
    except ValueError:
        return False


def run_pilot(path: str | Path = ".", report_path: str | Path | None = None) -> PilotResult:
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    before = capture_snapshot(root)

    inventory = collect_inventory(root)
    dependencies = scan_dependency_edges(root)
    component_candidates = discover_component_candidates(root, inventory, dependencies)

    profile_path = find_profile(root)
    profile_validation = validate_profile(root) if profile_path else None
    profile_result = profile_validation.as_dict() if profile_validation else None
    declared_partition: dict[str, object] | None = None
    findings: list[dict[str, object]] = []
    evaluator: dict[str, object]

    if not profile_path:
        evaluator = {
            "status": "BLOCKED",
            "reason": "NO_PROFILE",
            "finding_count": None,
            "dependency_evidence_coverage_complete": dependencies.as_dict()["coverage_complete"],
        }
    elif not profile_validation or not profile_validation.valid:
        evaluator = {
            "status": "BLOCKED",
            "reason": "PROFILE_INVALID",
            "finding_count": None,
            "dependency_evidence_coverage_complete": dependencies.as_dict()["coverage_complete"],
        }
    else:
        profile_payload = yaml.safe_load(profile_path.read_text(encoding="utf-8-sig"))
        components = profile_payload.get("components") if isinstance(profile_payload, dict) else None
        if isinstance(components, list):
            typed_components = [item for item in components if isinstance(item, dict)]
            partition = partition_components(root, typed_components)
            declared_partition = partition.as_dict()
            evaluated = evaluate_declared_dependency_boundaries(typed_components, partition, dependencies)
            findings = [item.as_dict() for item in evaluated]
            evaluator = {
                "status": "RAN",
                "reason": None,
                "finding_count": len(findings),
                "dependency_evidence_coverage_complete": dependencies.as_dict()["coverage_complete"],
            }
        else:
            evaluator = {
                "status": "BLOCKED",
                "reason": "COMPONENT_DECLARATIONS_REQUIRED",
                "finding_count": None,
                "dependency_evidence_coverage_complete": dependencies.as_dict()["coverage_complete"],
            }

    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    spec = current_spec_identity()

    if report_path is None:
        destination = pilot_directory(root) / "report.json"
    else:
        destination = Path(report_path).expanduser().resolve()
    output_inside_repository = _inside(root, destination)

    if output_inside_repository:
        non_intrusion_status = "USER_AUTHORIZED_OUTPUT_WRITE"
    elif comparison.stable:
        non_intrusion_status = "VERIFIED_NO_OBSERVED_CHANGE"
    else:
        non_intrusion_status = "CHANGE_OBSERVED_DURING_ANALYSIS"

    report = {
        "format": "ptsip-pilot-report/v2",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "tool": {"version": spec.tool_version},
        "specification": {
            "name": spec.name,
            "version": spec.version,
            "source": spec.source,
            "revision": spec.revision,
            "draft_revision_is_normative_identity": True,
        },
        "repository": repo.as_dict(),
        "snapshot": {
            "before": before.as_dict(),
            "after": after.as_dict(),
            "comparison": comparison.as_dict(),
        },
        "non_intrusion": {
            "status": non_intrusion_status,
            "analysis_read_only_by_design": True,
            "output_inside_repository": output_inside_repository,
            "methods": [
                "git-head-before-after",
                "git-status-before-after-including-ignored",
                "tracked-content-fingerprint-before-after",
            ],
            "note": (
                "No observed repository-state change was detected during analysis."
                if comparison.stable and not output_inside_repository
                else "A changed snapshot cannot attribute causality to PTSIP; inspect concurrent repository activity."
            ),
        },
        "inventory": inventory.as_dict(),
        "components": {
            "status": "DECLARED_AND_CANDIDATES" if declared_partition else "CANDIDATES_ONLY",
            "classification_asserted": bool(declared_partition),
            "candidates": [candidate.as_dict() for candidate in component_candidates],
            "declared_partition": declared_partition,
        },
        "dependencies": dependencies.as_dict(),
        "artifacts": {
            "status": "NOT_INSPECTED",
            "note": "Tool 0.2.3 does not build or inspect Product artifacts automatically.",
        },
        "classification": {
            "status": "DECLARATION_AVAILABLE" if declared_partition else "EVIDENCE_ONLY",
            "allowed_classifications": [item.value for item in Classification],
            "decision_statuses": [item.value for item in DecisionStatus],
            "note": "UNKNOWN is a decision status, not a fourth PTSIP architecture classification.",
        },
        "profile": profile_result,
        "evaluation": {
            "declared_dependency_boundaries": evaluator,
        },
        "findings": findings,
        "conformance": {
            "status": "NOT_EVALUATED",
            "reason": "Tool 0.2.3 improves evidence correctness but does not implement rule-relative coverage and Product Artifact evaluation required for a strict conformance result.",
        },
    }

    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return PilotResult(report=report, report_path=destination)
