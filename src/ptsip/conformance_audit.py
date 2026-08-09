from __future__ import annotations

import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from .constants import SPEC_REVISION


_ALLOWED_EVALUATOR_STATES = {"RAN", "BLOCKED", "NOT_APPLICABLE"}


def _active_rule_ids() -> set[str]:
    path = Path(__file__).resolve().parent / "specdata" / "ptsip-registry.yaml"
    payload = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    registry = payload.get("ptsip_registry", {}) if isinstance(payload, dict) else {}
    rules = registry.get("rules", []) if isinstance(registry, dict) else []
    return {str(item.get("id")) for item in rules if isinstance(item, dict) and item.get("id")}


def _diagnostic_validator() -> Draft202012Validator:
    path = Path(__file__).resolve().parent / "specdata" / "ptsip-diagnostic.schema.json"
    schema = json.loads(path.read_text(encoding="utf-8"))
    return Draft202012Validator(schema)


def audit_conformance_report(report: dict[str, object]) -> dict[str, object]:
    problems: list[str] = []
    active_rules = _active_rule_ids()
    validator = _diagnostic_validator()

    diagnostics = report.get("diagnostics", [])
    seen_diagnostics: set[str] = set()
    if not isinstance(diagnostics, list):
        problems.append("diagnostics is not a list")
        diagnostics = []
    for index, item in enumerate(diagnostics):
        if not isinstance(item, dict):
            problems.append(f"diagnostics[{index}] is not an object")
            continue
        for error in validator.iter_errors(item):
            location = ".".join(str(part) for part in error.absolute_path) or "<root>"
            problems.append(f"diagnostics[{index}] {location}: {error.message}")
        diagnostic_id = item.get("diagnostic_id")
        if isinstance(diagnostic_id, str):
            if diagnostic_id in seen_diagnostics:
                problems.append(f"duplicate diagnostic_id {diagnostic_id!r}")
            seen_diagnostics.add(diagnostic_id)
        rule_id = item.get("rule_id")
        if isinstance(rule_id, str) and rule_id not in active_rules:
            problems.append(f"diagnostic references non-active rule {rule_id!r}")

    coverage = report.get("coverage", {})
    seen_gap_ids: set[str] = set()
    if not isinstance(coverage, dict):
        problems.append("coverage is not an object")
        coverage = {}
    for bucket in ("blocking_gaps", "non_blocking_gaps"):
        values = coverage.get(bucket, [])
        if not isinstance(values, list):
            problems.append(f"coverage.{bucket} is not a list")
            continue
        for index, item in enumerate(values):
            if not isinstance(item, dict):
                problems.append(f"coverage.{bucket}[{index}] is not an object")
                continue
            gap_id = item.get("id")
            if not isinstance(gap_id, str) or not gap_id:
                problems.append(f"coverage.{bucket}[{index}] has no stable id")
            elif gap_id in seen_gap_ids:
                problems.append(f"duplicate coverage gap id {gap_id!r}")
            else:
                seen_gap_ids.add(gap_id)
            for rule_id in item.get("rule_ids", []):
                if isinstance(rule_id, str) and rule_id not in active_rules:
                    problems.append(f"coverage gap {gap_id!r} references non-active rule {rule_id!r}")

    evaluators = report.get("evaluators", {})
    if not isinstance(evaluators, dict):
        problems.append("evaluators is not an object")
        evaluators = {}
    for evaluator_id, state in evaluators.items():
        if not isinstance(state, dict):
            problems.append(f"evaluator {evaluator_id!r} state is not an object")
            continue
        status = state.get("status")
        if status not in _ALLOWED_EVALUATOR_STATES:
            problems.append(f"evaluator {evaluator_id!r} has unsupported status {status!r}")

    if report.get("outcome") == "CONFORMANT":
        blocking = coverage.get("blocking_gaps", []) if isinstance(coverage, dict) else []
        if blocking:
            problems.append("CONFORMANT report contains blocking coverage gaps")
        if any(
            isinstance(item, dict) and item.get("outcome_effect") in {"NON_CONFORMANT", "INCOMPLETE"}
            for item in diagnostics
        ):
            problems.append("CONFORMANT report contains outcome-affecting diagnostics")
        required = {
            "declared_dependency_boundaries",
            "product_artifact_boundary",
            "artifact_snapshot_binding",
            "independent_build_resolution",
            "lifecycle_independence",
            "profile_validation",
            "source_language_coverage",
            "specification_revision_binding",
            "snapshot_integrity",
        }
        for evaluator_id in sorted(required):
            state = evaluators.get(evaluator_id)
            if not isinstance(state, dict) or state.get("status") != "RAN":
                problems.append(f"CONFORMANT report requires evaluator {evaluator_id!r} to have status RAN")
        snapshot = report.get("snapshot", {})
        comparison = snapshot.get("comparison", {}) if isinstance(snapshot, dict) else {}
        if not isinstance(comparison, dict) or comparison.get("stable") is not True:
            problems.append("CONFORMANT report requires a stable complete-evaluation snapshot")
        profile = report.get("profile", {})
        if not isinstance(profile, dict) or profile.get("valid") is not True:
            problems.append("CONFORMANT report requires a valid Project Profile")
        specification = report.get("specification", {})
        if not isinstance(specification, dict) or specification.get("revision") != SPEC_REVISION:
            problems.append("CONFORMANT report requires the exact supported mutable-draft specification revision")
        applicability = coverage.get("applicability", {}) if isinstance(coverage, dict) else {}
        if not isinstance(applicability, dict) or not applicability:
            problems.append("CONFORMANT report requires applicability and evidence-sufficiency accounting")
        else:
            for area, state in applicability.items():
                if not isinstance(state, dict):
                    problems.append(f"coverage applicability {area!r} is not an object")
                elif state.get("applicable") is True and state.get("status") != "SUFFICIENT":
                    problems.append(f"CONFORMANT report has insufficient applicable coverage for {area!r}")

    return {
        "status": "PASS" if not problems else "FAIL",
        "problem_count": len(problems),
        "problems": problems,
        "active_rule_count": len(active_rules),
        "diagnostic_count": len(diagnostics),
        "coverage_gap_count": len(seen_gap_ids),
    }
