from __future__ import annotations

import hashlib
from pathlib import Path

import yaml

from .build_resolution import evaluate_independent_build_resolution
from .conformance import (
    ConformanceResult,
    _dependency_coverage_gaps,
    _finding_diagnostic,
    _unassigned_coverage_gaps,
    evaluate_conformance as evaluate_base_conformance,
)
from .conformance_audit import audit_conformance_report
from .inspection.dependencies import DependencyScan
from .inspection.dependencies_030 import scan_dependency_edges
from .lifecycle_evidence import evaluate_lifecycle_evidence
from .model import EvidenceNodeScope, ResolutionStatus
from .repository.discover import discover_repository
from .review_evidence import (
    evaluate_agent_decisions,
    load_agent_decisions,
    load_external_evidence,
    merge_external_dependencies,
)
from .validation.components import partition_components
from .validation.profile import find_profile, validate_profile
from .validation.rules import evaluate_component_dependency_policy, evaluate_declared_dependency_boundaries


def _recalculate_outcome(report: dict[str, object]) -> None:
    diagnostics = report.get("diagnostics", [])
    coverage = report.get("coverage", {})
    non_conformant = any(
        isinstance(item, dict) and item.get("outcome_effect") == "NON_CONFORMANT"
        for item in diagnostics
        if isinstance(diagnostics, list)
    )
    incomplete_diagnostic = any(
        isinstance(item, dict) and item.get("outcome_effect") == "INCOMPLETE"
        for item in diagnostics
        if isinstance(diagnostics, list)
    )
    blocking = coverage.get("blocking_gaps", []) if isinstance(coverage, dict) else []
    if non_conformant:
        report["outcome"] = "NON_CONFORMANT"
    elif incomplete_diagnostic or bool(blocking):
        report["outcome"] = "INCOMPLETE"
    else:
        report["outcome"] = "CONFORMANT"


def _externally_supplemented_native_ids(native: DependencyScan, external_edges: tuple) -> set[str]:
    resolved_external = [
        edge
        for edge in external_edges
        if edge.resolution in {ResolutionStatus.RESOLVED, ResolutionStatus.EXTERNAL}
        and edge.target_scope != EvidenceNodeScope.UNRESOLVED_TARGET
    ]
    supplemented: set[str] = set()
    for native_edge in native.edges:
        if native_edge.resolution not in {ResolutionStatus.UNRESOLVED, ResolutionStatus.DYNAMIC}:
            continue
        if any(
            external.source == native_edge.source
            and external.target == native_edge.target
            and external.edge_type == native_edge.edge_type
            for external in resolved_external
        ):
            supplemented.add(native_edge.evidence_id)
    return supplemented


def _external_conflict_gaps(native: DependencyScan, external_edges: tuple) -> list[dict[str, object]]:
    gaps: list[dict[str, object]] = []
    for external in external_edges:
        for observed in native.edges:
            if observed.source != external.source or observed.target != external.target or observed.edge_type != external.edge_type:
                continue
            if observed.resolution in {ResolutionStatus.UNRESOLVED, ResolutionStatus.DYNAMIC}:
                continue
            contradictory_scope = observed.target_scope != external.target_scope
            contradictory_path = (
                observed.resolution == ResolutionStatus.RESOLVED
                and external.resolution == ResolutionStatus.RESOLVED
                and observed.resolved_path != external.resolved_path
            )
            if not (contradictory_scope or contradictory_path):
                continue
            gaps.append(
                {
                    "id": "external-evidence:conflict:" + hashlib.sha256((observed.evidence_id + external.evidence_id).encode()).hexdigest()[:16],
                    "blocking": True,
                    "rule_ids": ["PTSIP-EVD-002", "PTSIP-EVD-003", "PTSIP-EVD-004"],
                    "evidence_ids": [observed.evidence_id, external.evidence_id],
                    "message": "Imported external dependency evidence contradicts resolved native repository evidence and therefore cannot override it silently.",
                }
            )
    return gaps


def _external_input_gaps(external_load) -> list[dict[str, object]]:
    return [
        {
            "id": "external-evidence:invalid:" + hashlib.sha256((issue.source_path + issue.message).encode()).hexdigest()[:16],
            "blocking": True,
            "rule_ids": ["PTSIP-EVD-003", "PTSIP-EVD-004"],
            "evidence_ids": [f"external-input:{issue.source_path}"],
            "message": f"Explicit external evidence input is unusable for strict conformance: {issue.message}",
        }
        for issue in external_load.issues
    ]


def _coverage_without_supplemented(gaps: list[dict[str, object]], supplemented_ids: set[str]) -> list[dict[str, object]]:
    result: list[dict[str, object]] = []
    for gap in gaps:
        evidence_ids = {str(item) for item in gap.get("evidence_ids", [])}
        if str(gap.get("id", "")).startswith("dependency-target:") and evidence_ids & supplemented_ids:
            continue
        result.append(gap)
    return result


def evaluate_conformance(
    path: str | Path = ".",
    profile_path: str | Path | None = None,
    artifact_evidence_paths: list[str | Path] | tuple[str | Path, ...] | None = None,
    agent_decision_paths: list[str | Path] | tuple[str | Path, ...] | None = None,
    external_evidence_paths: list[str | Path] | tuple[str | Path, ...] | None = None,
) -> ConformanceResult:
    base = evaluate_base_conformance(path, profile_path, artifact_evidence_paths)
    report = base.report
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    profile = find_profile(root, profile_path)
    validation = validate_profile(root, profile_path)

    native_dependencies = scan_dependency_edges(root)
    external_load = load_external_evidence(external_evidence_paths, repo)
    dependencies = merge_external_dependencies(native_dependencies, external_load)
    report["dependencies"] = dependencies.as_dict()
    report["external_evidence"] = external_load.as_dict()

    diagnostics = list(report.get("diagnostics", [])) if isinstance(report.get("diagnostics"), list) else []
    diagnostics = [
        item
        for item in diagnostics
        if not (
            isinstance(item, dict)
            and isinstance(item.get("evaluator"), dict)
            and item["evaluator"].get("id") == "declared-dependency-boundaries"
        )
    ]
    coverage = report.get("coverage")
    if not isinstance(coverage, dict):
        coverage = {"blocking_gaps": [], "non_blocking_gaps": []}
        report["coverage"] = coverage
    blocking = list(coverage.get("blocking_gaps", [])) if isinstance(coverage.get("blocking_gaps"), list) else []
    non_blocking = list(coverage.get("non_blocking_gaps", [])) if isinstance(coverage.get("non_blocking_gaps"), list) else []
    replace_prefixes = (
        "dependency-scan:",
        "dependency-target:",
        "component-ownership:",
        "build-resolution:coverage",
        "build-resolution:",
        "lifecycle:",
        "language-coverage:",
        "agent-decision:",
        "external-evidence:",
        "conformance-audit:",
    )
    blocking = [item for item in blocking if not (isinstance(item, dict) and str(item.get("id", "")).startswith(replace_prefixes))]
    non_blocking = [item for item in non_blocking if not (isinstance(item, dict) and str(item.get("id", "")).startswith(replace_prefixes))]

    evaluators = report.get("evaluators")
    if not isinstance(evaluators, dict):
        evaluators = {}
        report["evaluators"] = evaluators
    evaluators["external_evidence_import"] = {
        "status": "NOT_APPLICABLE" if not external_evidence_paths else ("BLOCKED" if external_load.issues else "RAN"),
        "reason": None if not external_load.issues else "EXTERNAL_EVIDENCE_INVALID_OR_STALE",
        "document_count": len(external_load.documents),
        "edge_count": len(external_load.edges),
    }
    blocking.extend(_external_input_gaps(external_load))
    blocking.extend(_external_conflict_gaps(native_dependencies, external_load.edges))

    components: list[dict[str, object]] = []
    payload: dict[str, object] | None = None
    if profile is not None and validation.valid:
        loaded = yaml.safe_load(profile.read_text(encoding="utf-8-sig"))
        payload = loaded if isinstance(loaded, dict) else None
        declared = payload.get("components") if payload is not None else None
        if isinstance(declared, list):
            components = [item for item in declared if isinstance(item, dict)]

    agent_load = load_agent_decisions(agent_decision_paths)
    agent_report, agent_gaps = evaluate_agent_decisions(agent_load, components)
    report["agent_decisions"] = agent_report
    blocking.extend(agent_gaps)
    evaluators["agent_decision_review"] = {
        "status": "NOT_APPLICABLE" if not agent_decision_paths else ("BLOCKED" if agent_load.issues else "RAN"),
        "reason": None if not agent_load.issues else "AGENT_DECISION_INPUT_INVALID",
        "document_count": len(agent_load.documents),
    }

    if components:
        partition = partition_components(root, components)
        boundary_findings = evaluate_declared_dependency_boundaries(components, partition, dependencies)
        diagnostics.extend(_finding_diagnostic(item, "declared-dependency-boundaries") for item in boundary_findings)
        dependency_gaps = _dependency_coverage_gaps(dependencies, components, partition)
        supplemented = _externally_supplemented_native_ids(native_dependencies, external_load.edges)
        dependency_gaps = _coverage_without_supplemented(dependency_gaps, supplemented)
        blocking.extend(item for item in dependency_gaps if item.get("blocking"))
        non_blocking.extend(item for item in dependency_gaps if not item.get("blocking"))
        ownership_gaps = _unassigned_coverage_gaps(partition)
        blocking.extend(item for item in ownership_gaps if item.get("blocking"))
        non_blocking.extend(item for item in ownership_gaps if not item.get("blocking"))
        evaluators["declared_dependency_boundaries"] = {
            "status": "RAN",
            "reason": None,
            "finding_count": len(boundary_findings),
        }

        component_policy = payload.get("component_dependency_policy") if payload is not None else None
        policy_results = evaluate_component_dependency_policy(
            component_policy if isinstance(component_policy, dict) else None,
            components,
            partition,
            dependencies,
        )
        report["project_policy"] = {
            "findings": [item.as_dict() for item in policy_results],
            "finding_count": len(policy_results),
            "affects_ptsip_outcome": False,
        }
        evaluators["component_dependency_policy"] = {
            "status": "RAN" if isinstance(component_policy, dict) else "NOT_APPLICABLE",
            "reason": None,
            "finding_count": len(policy_results),
        }

        build = evaluate_independent_build_resolution(root, components)
        evaluators["independent_build_resolution"] = {
            "status": build.status,
            "reason": build.reason,
            "manifest_count": len(build.manifests),
        }
        blocking.extend(build.blocking_gaps)
        report["build_resolution"] = build.as_dict()

        lifecycle = evaluate_lifecycle_evidence(root, components, partition)
        evaluators["lifecycle_independence"] = {
            "status": lifecycle.status,
            "reason": lifecycle.reason,
            "release_workflow_count": len(lifecycle.workflows),
        }
        blocking.extend(lifecycle.blocking_gaps)
        report["lifecycle"] = lifecycle.as_dict()
    else:
        evaluators["independent_build_resolution"] = {"status": "BLOCKED", "reason": "COMPONENT_DECLARATIONS_REQUIRED"}
        evaluators["lifecycle_independence"] = {"status": "BLOCKED", "reason": "COMPONENT_DECLARATIONS_REQUIRED"}
        report["build_resolution"] = {
            "status": "BLOCKED",
            "reason": "COMPONENT_DECLARATIONS_REQUIRED",
            "manifests": [],
            "blocking_gaps": [],
        }
        report["lifecycle"] = {
            "status": "BLOCKED",
            "reason": "COMPONENT_DECLARATIONS_REQUIRED",
            "workflows": [],
            "blocking_gaps": [],
            "observations": [],
        }

    report["diagnostics"] = diagnostics
    coverage["blocking_gaps"] = blocking
    coverage["non_blocking_gaps"] = non_blocking
    _recalculate_outcome(report)

    audit = audit_conformance_report(report)
    report["audit"] = audit
    evaluators["conformance_report_audit"] = {
        "status": "RAN" if audit["status"] == "PASS" else "BLOCKED",
        "reason": None if audit["status"] == "PASS" else "REPORT_CONTRACT_AUDIT_FAILED",
        "problem_count": audit["problem_count"],
    }
    if audit["status"] != "PASS":
        coverage["blocking_gaps"].append(
            {
                "id": "conformance-audit:report-contract",
                "blocking": True,
                "rule_ids": ["PTSIP-DIA-001", "PTSIP-EVD-003"],
                "evidence_ids": ["conformance-audit:report-contract"],
                "message": "Internal conformance report contract audit failed: " + "; ".join(str(item) for item in audit["problems"][:10]),
            }
        )
        _recalculate_outcome(report)
    return ConformanceResult(report=report)
