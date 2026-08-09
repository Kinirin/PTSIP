from __future__ import annotations

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
from .inspection.dependencies_030 import scan_dependency_edges
from .lifecycle_evidence import evaluate_lifecycle_evidence
from .repository.discover import discover_repository
from .repository.snapshot import repository_files
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


def _unsupported_language_gaps(root: Path, components: list[dict[str, object]], assignments: dict[str, str]) -> list[dict[str, object]]:
    classifications = {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }
    _mode, paths, _errors = repository_files(root)
    go_files = [path for path in paths if Path(path).suffix.lower() == ".go" and classifications.get(assignments.get(path)) in {"PRODUCT", "TOOLCHAIN"}]
    cs_files = [path for path in paths if Path(path).suffix.lower() == ".cs" and classifications.get(assignments.get(path)) in {"PRODUCT", "TOOLCHAIN"}]
    gaps: list[dict[str, object]] = []
    if go_files:
        gaps.append(
            {
                "id": "language-coverage:go-source",
                "blocking": True,
                "rule_ids": ["PTSIP-DEP-001", "PTSIP-EVD-003"],
                "evidence_ids": [f"unsupported:go:{path}" for path in go_files[:50]],
                "message": f"{len(go_files)} classified Go source file(s) are present but Go source dependency analysis is not implemented in this Tool 0.3.0 tranche.",
            }
        )
    if cs_files:
        gaps.append(
            {
                "id": "language-coverage:dotnet-source",
                "blocking": True,
                "rule_ids": ["PTSIP-DEP-001", "PTSIP-EVD-003"],
                "evidence_ids": [f"unsupported:dotnet-source:{path}" for path in cs_files[:50]],
                "message": f"{len(cs_files)} classified C# source file(s) are present; .csproj ProjectReference evidence is supported but source-level .NET dependency coverage remains incomplete.",
            }
        )
    return gaps


def evaluate_conformance(
    path: str | Path = ".",
    profile_path: str | Path | None = None,
    artifact_evidence_paths: list[str | Path] | tuple[str | Path, ...] | None = None,
) -> ConformanceResult:
    base = evaluate_base_conformance(path, profile_path, artifact_evidence_paths)
    report = base.report
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    profile = find_profile(root, profile_path)
    validation = validate_profile(root, profile_path)
    dependencies = scan_dependency_edges(root)
    report["dependencies"] = dependencies.as_dict()

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
    )
    blocking = [item for item in blocking if not (isinstance(item, dict) and str(item.get("id", "")).startswith(replace_prefixes))]
    non_blocking = [item for item in non_blocking if not (isinstance(item, dict) and str(item.get("id", "")).startswith(replace_prefixes))]

    evaluators = report.get("evaluators")
    if not isinstance(evaluators, dict):
        evaluators = {}
        report["evaluators"] = evaluators

    components: list[dict[str, object]] = []
    payload: dict[str, object] | None = None
    if profile is not None and validation.valid:
        loaded = yaml.safe_load(profile.read_text(encoding="utf-8-sig"))
        payload = loaded if isinstance(loaded, dict) else None
        declared = payload.get("components") if payload is not None else None
        if isinstance(declared, list):
            components = [item for item in declared if isinstance(item, dict)]

    if components:
        partition = partition_components(root, components)
        boundary_findings = evaluate_declared_dependency_boundaries(components, partition, dependencies)
        diagnostics.extend(_finding_diagnostic(item, "declared-dependency-boundaries") for item in boundary_findings)
        blocking.extend(item for item in _dependency_coverage_gaps(dependencies, components, partition) if item.get("blocking"))
        non_blocking.extend(item for item in _dependency_coverage_gaps(dependencies, components, partition) if not item.get("blocking"))
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

        assignments = {item.path: item.component_id for item in partition.assignments}
        blocking.extend(_unsupported_language_gaps(root, components, assignments))
    else:
        evaluators["independent_build_resolution"] = {
            "status": "BLOCKED",
            "reason": "COMPONENT_DECLARATIONS_REQUIRED",
        }
        evaluators["lifecycle_independence"] = {
            "status": "BLOCKED",
            "reason": "COMPONENT_DECLARATIONS_REQUIRED",
        }
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
    return ConformanceResult(report=report)
