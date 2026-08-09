from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path

import yaml

from .constants import TOOL_VERSION
from .inspection.dependencies import DependencyScan, scan_dependency_edges
from .model import ResolutionStatus
from .repository.discover import discover_repository
from .repository.snapshot import capture_snapshot, compare_snapshots
from .spec_identity import current_spec_identity
from .validation.components import ComponentPartition, partition_components
from .validation.profile import find_profile, validate_profile
from .validation.rules import RuleFinding, evaluate_declared_dependency_boundaries


@dataclass(frozen=True)
class ConformanceResult:
    report: dict[str, object]

    @property
    def outcome(self) -> str:
        return str(self.report["outcome"])


def _diagnostic_id(rule_id: str, evidence_ids: tuple[str, ...], message: str) -> str:
    material = "\n".join((rule_id, *evidence_ids, message)).encode("utf-8")
    return "diag-" + hashlib.sha256(material).hexdigest()[:20]


def _diagnostic(
    *,
    rule_id: str,
    outcome_effect: str,
    severity: str,
    evidence_ids: tuple[str, ...],
    message: str,
    source_component: str | None = None,
    target_component: str | None = None,
    evaluator_id: str,
) -> dict[str, object]:
    payload: dict[str, object] = {
        "format": "ptsip-diagnostic/v1",
        "diagnostic_id": _diagnostic_id(rule_id, evidence_ids, message),
        "rule_id": rule_id,
        "outcome_effect": outcome_effect,
        "severity": severity,
        "evidence_ids": list(evidence_ids),
        "message": message,
        "evaluator": {
            "id": evaluator_id,
            "version": TOOL_VERSION,
            "provenance": "INFERRED",
        },
    }
    if source_component is not None:
        payload["source_component"] = source_component
    if target_component is not None:
        payload["target_component"] = target_component
    return payload


def _coverage_gap(
    *,
    gap_id: str,
    message: str,
    rule_ids: tuple[str, ...],
    evidence_ids: tuple[str, ...],
    blocking: bool,
) -> dict[str, object]:
    return {
        "id": gap_id,
        "blocking": blocking,
        "rule_ids": list(rule_ids),
        "evidence_ids": list(evidence_ids),
        "message": message,
    }


def _classifications(components: list[dict[str, object]]) -> dict[str, str]:
    return {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }


def _owners(partition: ComponentPartition) -> dict[str, str]:
    return {assignment.path: assignment.component_id for assignment in partition.assignments}


def _finding_diagnostic(finding: RuleFinding) -> dict[str, object]:
    if finding.severity != "REVIEW":
        outcome_effect = "NON_CONFORMANT"
        severity = "ERROR"
    elif finding.rule_id in {"PTSIP-DEP-001", "PTSIP-BLD-002"}:
        outcome_effect = "INCOMPLETE"
        severity = "WARNING"
    else:
        outcome_effect = "NONE"
        severity = "INFO"
    return _diagnostic(
        rule_id=finding.rule_id,
        outcome_effect=outcome_effect,
        severity=severity,
        evidence_ids=finding.evidence_ids,
        message=finding.message,
        source_component=finding.source_component,
        target_component=finding.target_component,
        evaluator_id="declared-dependency-boundaries",
    )


def _dependency_coverage_gaps(
    dependencies: DependencyScan,
    components: list[dict[str, object]],
    partition: ComponentPartition,
) -> list[dict[str, object]]:
    classifications = _classifications(components)
    owners = _owners(partition)
    gaps: list[dict[str, object]] = []

    for issue in dependencies.issues:
        owner = owners.get(issue.path)
        owner_class = classifications.get(owner) if owner else None
        blocking = issue.path == "<repository>" or owner_class == "PRODUCT"
        gaps.append(
            _coverage_gap(
                gap_id=f"dependency-scan:{issue.adapter}:{issue.path}",
                message=f"Dependency evidence collection issue: {issue.message}",
                rule_ids=("PTSIP-DEP-001", "PTSIP-EVD-003") if blocking else ("PTSIP-EVD-003",),
                evidence_ids=(f"scan-issue:{issue.adapter}:{issue.path}",),
                blocking=blocking,
            )
        )

    for edge in dependencies.edges:
        source_component = owners.get(edge.source)
        if classifications.get(source_component) != "PRODUCT":
            continue
        if edge.resolution not in {ResolutionStatus.UNRESOLVED, ResolutionStatus.DYNAMIC}:
            continue
        gaps.append(
            _coverage_gap(
                gap_id=f"dependency-target:{edge.evidence_id}",
                message="A Product-owned dependency target is unresolved/dynamic and could conceal a prohibited Product-to-Toolchain dependency.",
                rule_ids=("PTSIP-DEP-001", "PTSIP-EVD-003"),
                evidence_ids=(edge.evidence_id,),
                blocking=True,
            )
        )
    return gaps


def _unassigned_coverage_gaps(partition: ComponentPartition) -> list[dict[str, object]]:
    relevant_names = {"pyproject.toml", "package.json", "go.mod"}
    relevant_suffixes = {".py", ".js", ".mjs", ".cjs", ".jsx", ".ts", ".tsx", ".csproj", ".go"}
    relevant = [
        path
        for path in partition.unassigned_files
        if Path(path).name in relevant_names or Path(path).suffix.lower() in relevant_suffixes
    ]
    if not relevant:
        return []
    evidence = tuple(f"unassigned:{path}" for path in relevant[:50])
    return [
        _coverage_gap(
            gap_id="component-ownership:unassigned-relevant-files",
            message=f"{len(relevant)} tracked source/manifest file(s) are outside declared component ownership and may conceal a mandatory boundary.",
            rule_ids=("PTSIP-CLS-001", "PTSIP-EVD-003"),
            evidence_ids=evidence,
            blocking=True,
        )
    ]


def evaluate_conformance(path: str | Path = ".", profile_path: str | Path | None = None) -> ConformanceResult:
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    before = capture_snapshot(root)
    profile = find_profile(root, profile_path)
    profile_validation = validate_profile(root, profile_path)
    dependencies = scan_dependency_edges(root)

    diagnostics: list[dict[str, object]] = []
    coverage_gaps: list[dict[str, object]] = []
    evaluators: dict[str, dict[str, object]] = {}
    partition: ComponentPartition | None = None
    components: list[dict[str, object]] = []

    if profile is None:
        coverage_gaps.append(
            _coverage_gap(
                gap_id="profile:missing",
                message="Enforced Conformance requires a machine-readable Project Profile.",
                rule_ids=("PTSIP-SPC-001", "PTSIP-EVD-003"),
                evidence_ids=("profile:missing",),
                blocking=True,
            )
        )
        evaluators["declared_dependency_boundaries"] = {"status": "BLOCKED", "reason": "NO_PROFILE"}
    elif not profile_validation.valid:
        coverage_gaps.append(
            _coverage_gap(
                gap_id="profile:invalid",
                message="Project Profile validation failed, so declared ownership cannot support Enforced Conformance.",
                rule_ids=("PTSIP-SPC-001", "PTSIP-EVD-003"),
                evidence_ids=("profile:invalid",),
                blocking=True,
            )
        )
        evaluators["declared_dependency_boundaries"] = {"status": "BLOCKED", "reason": "INVALID_PROFILE"}
    else:
        payload = yaml.safe_load(profile.read_text(encoding="utf-8-sig"))
        if isinstance(payload, dict):
            binding = payload.get("ptsip", {}).get("specification", {}) if isinstance(payload.get("ptsip"), dict) else {}
            revision = binding.get("revision") if isinstance(binding, dict) else None
            if not revision:
                coverage_gaps.append(
                    _coverage_gap(
                        gap_id="profile:missing-immutable-revision",
                        message="Enforced Conformance against a mutable draft requires an immutable Specification revision.",
                        rule_ids=("PTSIP-SPC-001", "PTSIP-EVD-003"),
                        evidence_ids=("profile:specification:revision:missing",),
                        blocking=True,
                    )
                )
            declared = payload.get("components")
            if isinstance(declared, list):
                components = [item for item in declared if isinstance(item, dict)]

        if components:
            partition = partition_components(root, components)
            findings = evaluate_declared_dependency_boundaries(components, partition, dependencies)
            diagnostics.extend(_finding_diagnostic(item) for item in findings)
            coverage_gaps.extend(_dependency_coverage_gaps(dependencies, components, partition))
            coverage_gaps.extend(_unassigned_coverage_gaps(partition))
            evaluators["declared_dependency_boundaries"] = {
                "status": "RAN",
                "reason": None,
                "finding_count": len(findings),
            }
        else:
            evaluators["declared_dependency_boundaries"] = {
                "status": "BLOCKED",
                "reason": "COMPONENT_DECLARATIONS_REQUIRED",
            }
            coverage_gaps.append(
                _coverage_gap(
                    gap_id="ownership:boundary-shorthand",
                    message="This Tool 0.3.0 conformance evaluator requires explicit components for dependency-to-component attribution; boundary-root shorthand remains valid profile syntax but is insufficient for this evaluator.",
                    rule_ids=("PTSIP-CLS-001", "PTSIP-EVD-003"),
                    evidence_ids=("profile:boundaries",),
                    blocking=True,
                )
            )

    coverage_gaps.append(
        _coverage_gap(
            gap_id="artifact-evidence:not-inspected",
            message="Product Artifact contents have not been inspected by this conformance tranche.",
            rule_ids=("PTSIP-PKG-001", "PTSIP-ART-001", "PTSIP-EVD-003"),
            evidence_ids=("artifact-evidence:not-inspected",),
            blocking=True,
        )
    )
    evaluators["product_artifact_boundary"] = {"status": "BLOCKED", "reason": "ARTIFACT_ADAPTER_NOT_IMPLEMENTED"}

    coverage_gaps.append(
        _coverage_gap(
            gap_id="build-resolution:coverage",
            message="Independent build-environment resolution is not yet evaluated end-to-end by this conformance tranche.",
            rule_ids=("PTSIP-BLD-001", "PTSIP-EVD-003"),
            evidence_ids=("coverage:build-resolution:not-implemented",),
            blocking=True,
        )
    )
    evaluators["independent_build_resolution"] = {"status": "BLOCKED", "reason": "EVALUATOR_NOT_IMPLEMENTED"}

    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    if not comparison.stable:
        coverage_gaps.append(
            _coverage_gap(
                gap_id="snapshot:invalidated",
                message="Repository evidence changed or was incomplete during conformance evaluation.",
                rule_ids=("PTSIP-EVD-001", "PTSIP-EVD-003"),
                evidence_ids=("snapshot:comparison",),
                blocking=True,
            )
        )

    non_conformant = any(item.get("outcome_effect") == "NON_CONFORMANT" for item in diagnostics)
    incomplete_diagnostic = any(item.get("outcome_effect") == "INCOMPLETE" for item in diagnostics)
    blocking_gap = any(bool(item.get("blocking")) for item in coverage_gaps)
    if non_conformant:
        outcome = "NON_CONFORMANT"
    elif incomplete_diagnostic or blocking_gap:
        outcome = "INCOMPLETE"
    else:
        outcome = "CONFORMANT"

    spec = current_spec_identity()
    report = {
        "format": "ptsip-conformance-report/v1",
        "tool": {"version": spec.tool_version},
        "specification": {
            "name": spec.name,
            "version": spec.version,
            "source": spec.source,
            "revision": spec.revision,
        },
        "repository": repo.as_dict(),
        "level": "ENFORCED",
        "outcome": outcome,
        "snapshot": {
            "before": before.as_dict(),
            "after": after.as_dict(),
            "comparison": comparison.as_dict(),
        },
        "profile": profile_validation.as_dict(),
        "dependencies": dependencies.as_dict(),
        "evaluators": evaluators,
        "coverage": {
            "blocking_gaps": [item for item in coverage_gaps if item["blocking"]],
            "non_blocking_gaps": [item for item in coverage_gaps if not item["blocking"]],
        },
        "diagnostics": diagnostics,
    }
    return ConformanceResult(report=report)
