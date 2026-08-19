from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path

from .artifact_evidence import ArtifactEvidenceLoad, load_artifact_evidence
from .constants import TOOL_VERSION
from .inspection.dependencies import DependencyScan, scan_dependency_edges
from .inspection.source_adapters import (
    SUPPORTED_SOURCE_SUFFIXES,
    is_supported_manifest,
    is_unsupported_mandatory_source,
)
from .model import ResolutionStatus
from .repository.discover import discover_repository
from .repository.snapshot import RepositorySnapshot, capture_snapshot, compare_snapshots
from .spec_identity import current_spec_identity
from .validation.components import ComponentPartition, partition_components
from .validation.profile import find_profile, validate_profile
from .validation.rules import (
    NON_PRODUCT_IMPLEMENTATION_CLASSES,
    RuleFinding,
    evaluate_component_dependency_policy,
    evaluate_declared_dependency_boundaries,
)


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


def _finding_diagnostic(finding: RuleFinding, evaluator_id: str) -> dict[str, object]:
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
        evaluator_id=evaluator_id,
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
                message=(
                    "A Product-owned dependency target is unresolved/dynamic and could conceal a prohibited "
                    "Product-to-non-Product implementation dependency."
                ),
                rule_ids=("PTSIP-DEP-001", "PTSIP-EVD-003"),
                evidence_ids=(edge.evidence_id,),
                blocking=True,
            )
        )
    return gaps


def _unassigned_coverage_gaps(partition: ComponentPartition) -> list[dict[str, object]]:
    relevant = [
        path
        for path in partition.unassigned_files
        if Path(path).suffix.lower() in SUPPORTED_SOURCE_SUFFIXES or is_supported_manifest(path)
    ]
    if not relevant:
        return []
    evidence = tuple(f"unassigned:{path}" for path in relevant[:50])
    return [
        _coverage_gap(
            gap_id="component-ownership:unassigned-relevant-files",
            message=(
                f"{len(relevant)} tracked source/manifest file(s) are outside declared component ownership "
                "and may conceal a mandatory boundary."
            ),
            rule_ids=("PTSIP-CLS-001", "PTSIP-EVD-003"),
            evidence_ids=evidence,
            blocking=True,
        )
    ]


def _unsupported_source_coverage_gaps(
    partition: ComponentPartition,
    components: list[dict[str, object]],
) -> list[dict[str, object]]:
    classifications = _classifications(components)
    unsupported = sorted(
        assignment.path
        for assignment in partition.assignments
        if classifications.get(assignment.component_id) in {"PRODUCT", *NON_PRODUCT_IMPLEMENTATION_CLASSES}
        and is_unsupported_mandatory_source(assignment.path)
    )
    if not unsupported:
        return []
    return [
        _coverage_gap(
            gap_id="language-coverage:unsupported-mandatory-source",
            message=(
                f"{len(unsupported)} lifecycle-owned executable source file(s) use unsupported dependency ecosystems; "
                "mandatory dependency-boundary evidence is incomplete."
            ),
            rule_ids=("PTSIP-DEP-001", "PTSIP-EVD-003"),
            evidence_ids=tuple(f"unsupported-source:{path}" for path in unsupported[:50]),
            blocking=True,
        )
    ]


def _artifact_evidence_ids(payload: dict[str, object]) -> tuple[str, ...]:
    evidence_ids = payload.get("evidence_ids")
    if isinstance(evidence_ids, list) and evidence_ids:
        return tuple(str(item) for item in evidence_ids)
    return (f"artifact:{payload.get('artifact_id', 'unknown')}",)


def _evaluate_artifacts(
    artifact_load: ArtifactEvidenceLoad,
    components: list[dict[str, object]],
    partition: ComponentPartition | None,
) -> tuple[list[dict[str, object]], list[dict[str, object]], dict[str, object]]:
    diagnostics: list[dict[str, object]] = []
    gaps: list[dict[str, object]] = []
    classifications = _classifications(components)
    owners = _owners(partition) if partition is not None else {}

    for issue in artifact_load.issues:
        gaps.append(
            _coverage_gap(
                gap_id=(
                    "artifact-evidence:invalid:"
                    + hashlib.sha256((issue.source_path + issue.message).encode()).hexdigest()[:12]
                ),
                message=f"Artifact evidence input is invalid: {issue.message}",
                rule_ids=("PTSIP-ART-001", "PTSIP-EVD-003"),
                evidence_ids=(f"artifact-input:{issue.source_path}",),
                blocking=True,
            )
        )

    product_documents = [
        item for item in artifact_load.documents if item.payload.get("classification") == "PRODUCT"
    ]
    if not product_documents:
        gaps.append(
            _coverage_gap(
                gap_id="artifact-evidence:product-missing",
                message="No valid PRODUCT artifact evidence was supplied for packaging-isolation evaluation.",
                rule_ids=("PTSIP-PKG-001", "PTSIP-ART-001", "PTSIP-EVD-003"),
                evidence_ids=("artifact-evidence:product-missing",),
                blocking=True,
            )
        )

    for document in product_documents:
        payload = document.payload
        artifact_id = str(payload["artifact_id"])
        evidence_ids = _artifact_evidence_ids(payload)
        contents = payload.get("contents")
        if not isinstance(contents, dict):
            continue

        if contents.get("complete") is not True:
            gaps.append(
                _coverage_gap(
                    gap_id=f"artifact-evidence:{artifact_id}:contents-incomplete",
                    message="Product Artifact content evidence is explicitly incomplete.",
                    rule_ids=("PTSIP-PKG-001", "PTSIP-ART-001", "PTSIP-EVD-003"),
                    evidence_ids=evidence_ids,
                    blocking=True,
                )
            )

        producer = payload.get("producer_component")
        if producer is not None and str(producer) not in classifications:
            gaps.append(
                _coverage_gap(
                    gap_id=f"artifact-evidence:{artifact_id}:unknown-producer",
                    message=f"Artifact producer component {producer!r} is not declared in the Project Profile.",
                    rule_ids=("PTSIP-ART-001", "PTSIP-EVD-003"),
                    evidence_ids=evidence_ids,
                    blocking=True,
                )
            )

        content_components = [
            str(item) for item in contents.get("components", []) if isinstance(item, str)
        ]
        unknown_components = sorted({item for item in content_components if item not in classifications})
        if unknown_components:
            gaps.append(
                _coverage_gap(
                    gap_id=f"artifact-evidence:{artifact_id}:unknown-components",
                    message="Artifact contents reference undeclared component(s): " + ", ".join(unknown_components),
                    rule_ids=("PTSIP-ART-001", "PTSIP-EVD-003"),
                    evidence_ids=evidence_ids,
                    blocking=True,
                )
            )

        nonproduct_components = {
            item
            for item in content_components
            if classifications.get(item) in NON_PRODUCT_IMPLEMENTATION_CLASSES
        }
        for path in contents.get("paths", []):
            if isinstance(path, str):
                owner = owners.get(path)
                if owner and classifications.get(owner) in NON_PRODUCT_IMPLEMENTATION_CLASSES:
                    nonproduct_components.add(owner)
        if nonproduct_components:
            diagnostics.append(
                _diagnostic(
                    rule_id="PTSIP-PKG-001",
                    outcome_effect="NON_CONFORMANT",
                    severity="ERROR",
                    evidence_ids=evidence_ids,
                    message=(
                        f"PRODUCT artifact {artifact_id!r} contains non-Product implementation component(s): "
                        + ", ".join(sorted(nonproduct_components))
                        + ". The artifact producer identity does not waive Product packaging isolation."
                    ),
                    evaluator_id="product-artifact-boundary",
                )
            )

    if artifact_load.documents:
        status = "RAN"
        reason = None
    else:
        status = "BLOCKED"
        reason = "NO_VALID_ARTIFACT_EVIDENCE"
    return diagnostics, gaps, {
        "status": status,
        "reason": reason,
        "artifact_count": len(artifact_load.documents),
        "product_artifact_count": len(product_documents),
    }


def evaluate_conformance(
    path: str | Path = ".",
    profile_path: str | Path | None = None,
    artifact_evidence_paths: list[str | Path] | tuple[str | Path, ...] | None = None,
    *,
    _snapshot_before: RepositorySnapshot | None = None,
    _finalize_snapshot: bool = True,
) -> ConformanceResult:
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    before = _snapshot_before or capture_snapshot(root)
    profile = find_profile(root, profile_path)
    profile_validation = validate_profile(root, profile_path)
    dependencies = scan_dependency_edges(root)
    artifact_load = load_artifact_evidence(artifact_evidence_paths, repo)

    diagnostics: list[dict[str, object]] = []
    project_policy_findings: list[dict[str, object]] = []
    coverage_gaps: list[dict[str, object]] = []
    evaluators: dict[str, dict[str, object]] = {}
    partition: ComponentPartition | None = None
    components: list[dict[str, object]] = []
    payload: dict[str, object] | None = None

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
        evaluators["component_dependency_policy"] = {"status": "BLOCKED", "reason": "NO_PROFILE"}
    elif not profile_validation.valid:
        coverage_gaps.append(
            _coverage_gap(
                gap_id="profile:invalid",
                message=(
                    "Project Profile validation failed, so declared ownership cannot support Enforced Conformance."
                ),
                rule_ids=("PTSIP-SPC-001", "PTSIP-EVD-003"),
                evidence_ids=("profile:invalid",),
                blocking=True,
            )
        )
        evaluators["declared_dependency_boundaries"] = {"status": "BLOCKED", "reason": "INVALID_PROFILE"}
        evaluators["component_dependency_policy"] = {"status": "BLOCKED", "reason": "INVALID_PROFILE"}
    else:
        resolved = profile_validation.resolved_profile
        if resolved is None:
            coverage_gaps.append(
                _coverage_gap(
                    gap_id="profile:resolution-unavailable",
                    message=(
                        "Project Profile validation succeeded without a resolved Responsibility Map runtime view."
                    ),
                    rule_ids=("PTSIP-RMAP-001", "PTSIP-EVD-003"),
                    evidence_ids=("profile:resolution-unavailable",),
                    blocking=True,
                )
            )
            evaluators["declared_dependency_boundaries"] = {
                "status": "BLOCKED",
                "reason": "RESOLVED_PROFILE_REQUIRED",
            }
            evaluators["component_dependency_policy"] = {
                "status": "BLOCKED",
                "reason": "RESOLVED_PROFILE_REQUIRED",
            }
        else:
            payload = resolved.effective_payload
            source_ptsip = resolved.source_payload.get("ptsip")
            source_binding = (
                source_ptsip.get("specification", {})
                if isinstance(source_ptsip, dict)
                else {}
            )
            revision = source_binding.get("revision") if isinstance(source_binding, dict) else None
            if not revision:
                coverage_gaps.append(
                    _coverage_gap(
                        gap_id="profile:missing-immutable-revision",
                        message=(
                            "Enforced Conformance against a mutable draft requires an immutable Specification revision."
                        ),
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
                boundary_findings = evaluate_declared_dependency_boundaries(
                    components, partition, dependencies
                )
                diagnostics.extend(
                    _finding_diagnostic(item, "declared-dependency-boundaries")
                    for item in boundary_findings
                )
                coverage_gaps.extend(_dependency_coverage_gaps(dependencies, components, partition))
                coverage_gaps.extend(_unassigned_coverage_gaps(partition))
                unsupported_gaps = _unsupported_source_coverage_gaps(partition, components)
                coverage_gaps.extend(unsupported_gaps)
                evaluators["source_language_coverage"] = {
                    "status": "BLOCKED" if unsupported_gaps else "RAN",
                    "reason": "UNSUPPORTED_MANDATORY_SOURCE" if unsupported_gaps else None,
                    "supported_suffixes": sorted(SUPPORTED_SOURCE_SUFFIXES),
                    "unsupported_source_count": (
                        len(unsupported_gaps[0]["evidence_ids"]) if unsupported_gaps else 0
                    ),
                }
                evaluators["declared_dependency_boundaries"] = {
                    "status": "RAN",
                    "reason": None,
                    "finding_count": len(boundary_findings),
                }

                component_policy = payload.get("component_dependency_policy")
                policy_results = evaluate_component_dependency_policy(
                    component_policy if isinstance(component_policy, dict) else None,
                    components,
                    partition,
                    dependencies,
                )
                project_policy_findings.extend(item.as_dict() for item in policy_results)
                evaluators["component_dependency_policy"] = {
                    "status": "RAN" if isinstance(component_policy, dict) else "NOT_APPLICABLE",
                    "reason": None,
                    "finding_count": len(policy_results),
                }
            else:
                coverage_gaps.append(
                    _coverage_gap(
                        gap_id="profile:effective-components-missing",
                        message="Validated effective Responsibility Map contains no component declarations.",
                        rule_ids=("PTSIP-RMAP-001", "PTSIP-EVD-003"),
                        evidence_ids=("profile:effective-components-missing",),
                        blocking=True,
                    )
                )
                evaluators["source_language_coverage"] = {
                    "status": "BLOCKED",
                    "reason": "EFFECTIVE_COMPONENT_DECLARATIONS_REQUIRED",
                }
                evaluators["declared_dependency_boundaries"] = {
                    "status": "BLOCKED",
                    "reason": "EFFECTIVE_COMPONENT_DECLARATIONS_REQUIRED",
                }
                evaluators["component_dependency_policy"] = {
                    "status": "BLOCKED",
                    "reason": "EFFECTIVE_COMPONENT_DECLARATIONS_REQUIRED",
                }

    artifact_diagnostics, artifact_gaps, artifact_evaluator = _evaluate_artifacts(
        artifact_load, components, partition
    )
    diagnostics.extend(artifact_diagnostics)
    coverage_gaps.extend(artifact_gaps)
    evaluators["product_artifact_boundary"] = artifact_evaluator

    coverage_gaps.append(
        _coverage_gap(
            gap_id="build-resolution:coverage",
            message="Independent build-environment resolution is not yet evaluated end-to-end by this conformance tranche.",
            rule_ids=("PTSIP-BLD-001", "PTSIP-EVD-003"),
            evidence_ids=("coverage:build-resolution:not-implemented",),
            blocking=True,
        )
    )
    evaluators["independent_build_resolution"] = {
        "status": "BLOCKED",
        "reason": "EVALUATOR_NOT_IMPLEMENTED",
    }

    after = capture_snapshot(root) if _finalize_snapshot else None
    comparison = compare_snapshots(before, after) if after is not None else None
    if comparison is not None and not comparison.stable:
        coverage_gaps.append(
            _coverage_gap(
                gap_id="snapshot:invalidated",
                message="Repository evidence changed or was incomplete during conformance evaluation.",
                rule_ids=("PTSIP-EVD-001", "PTSIP-EVD-003"),
                evidence_ids=("snapshot:comparison",),
                blocking=True,
            )
        )

    non_conformant = any(
        item.get("outcome_effect") == "NON_CONFORMANT" for item in diagnostics
    )
    incomplete_diagnostic = any(
        item.get("outcome_effect") == "INCOMPLETE" for item in diagnostics
    )
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
            "after": after.as_dict() if after is not None else None,
            "comparison": (
                comparison.as_dict()
                if comparison is not None
                else {"status": "PENDING", "stable": False, "reasons": []}
            ),
        },
        "profile": profile_validation.as_dict(),
        "dependencies": dependencies.as_dict(),
        "artifacts": artifact_load.as_dict(),
        "project_policy": {
            "findings": project_policy_findings,
            "finding_count": len(project_policy_findings),
            "affects_ptsip_outcome": False,
        },
        "evaluators": evaluators,
        "coverage": {
            "blocking_gaps": [item for item in coverage_gaps if item["blocking"]],
            "non_blocking_gaps": [item for item in coverage_gaps if not item["blocking"]],
        },
        "diagnostics": diagnostics,
    }
    return ConformanceResult(report=report)
