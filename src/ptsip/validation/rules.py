from __future__ import annotations

from dataclasses import asdict, dataclass

from ..inspection.dependencies import DependencyScan
from ..model import DependencyPhase
from .components import ComponentPartition


@dataclass(frozen=True)
class RuleFinding:
    rule_id: str
    severity: str
    message: str
    evidence_ids: tuple[str, ...]
    source_component: str | None = None
    target_component: str | None = None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def evaluate_declared_dependency_boundaries(
    components: list[dict[str, object]],
    partition: ComponentPartition,
    dependencies: DependencyScan,
) -> list[RuleFinding]:
    classifications = {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }
    owners = {assignment.path: assignment.component_id for assignment in partition.assignments}
    findings: list[RuleFinding] = []

    for edge in dependencies.edges:
        if not edge.resolved_path:
            continue
        source_component = owners.get(edge.source)
        target_component = owners.get(edge.resolved_path)
        if not source_component or not target_component or source_component == target_component:
            continue
        source_class = classifications.get(source_component)
        target_class = classifications.get(target_component)

        if source_class == "PRODUCT" and target_class == "TOOLCHAIN":
            if edge.phase == DependencyPhase.RUNTIME:
                findings.append(
                    RuleFinding(
                        rule_id="PTSIP-DEP-001",
                        severity="ERROR",
                        message="Declared PRODUCT component has a resolved runtime dependency on declared TOOLCHAIN component.",
                        evidence_ids=(edge.evidence_id,),
                        source_component=source_component,
                        target_component=target_component,
                    )
                )
            elif edge.phase == DependencyPhase.BUILD:
                findings.append(
                    RuleFinding(
                        rule_id="PTSIP-BLD-002",
                        severity="ERROR",
                        message="Declared PRODUCT component build invokes/depends on declared TOOLCHAIN component.",
                        evidence_ids=(edge.evidence_id,),
                        source_component=source_component,
                        target_component=target_component,
                    )
                )
            elif edge.phase == DependencyPhase.UNKNOWN:
                findings.append(
                    RuleFinding(
                        rule_id="PTSIP-DEP-001",
                        severity="REVIEW",
                        message="Resolved PRODUCT-to-TOOLCHAIN edge has unknown lifecycle phase; do not treat it as absence of violation.",
                        evidence_ids=(edge.evidence_id,),
                        source_component=source_component,
                        target_component=target_component,
                    )
                )

        if source_class == "TOOLCHAIN" and target_class == "PRODUCT" and edge.phase == DependencyPhase.UNKNOWN:
            findings.append(
                RuleFinding(
                    rule_id="PTSIP-DEP-002",
                    severity="REVIEW",
                    message="Resolved TOOLCHAIN-to-PRODUCT edge requires purpose/phase review; direction alone does not prove that it is inspection-only.",
                    evidence_ids=(edge.evidence_id,),
                    source_component=source_component,
                    target_component=target_component,
                )
            )

    return findings
