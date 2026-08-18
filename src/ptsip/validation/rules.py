from __future__ import annotations

from dataclasses import asdict, dataclass

from ..inspection.dependencies import DependencyScan
from ..model import DependencyPhase
from .components import ComponentPartition


NON_PRODUCT_IMPLEMENTATION_CLASSES = {
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
}


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


@dataclass(frozen=True)
class ProjectPolicyFinding:
    policy_id: str
    message: str
    evidence_ids: tuple[str, ...]
    source_component: str
    target_component: str

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def _component_context(
    components: list[dict[str, object]],
    partition: ComponentPartition,
) -> tuple[dict[str, str], dict[str, str]]:
    classifications = {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }
    owners = {assignment.path: assignment.component_id for assignment in partition.assignments}
    return classifications, owners


def evaluate_declared_dependency_boundaries(
    components: list[dict[str, object]],
    partition: ComponentPartition,
    dependencies: DependencyScan,
) -> list[RuleFinding]:
    classifications, owners = _component_context(components, partition)
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

        if source_class == "PRODUCT" and target_class in NON_PRODUCT_IMPLEMENTATION_CLASSES:
            if edge.phase == DependencyPhase.RUNTIME:
                findings.append(
                    RuleFinding(
                        rule_id="PTSIP-DEP-001",
                        severity="ERROR",
                        message=(
                            "Declared PRODUCT component has a resolved runtime dependency on "
                            f"declared {target_class} implementation."
                        ),
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
                        message=(
                            "Declared PRODUCT component build invokes/depends on declared "
                            f"{target_class} implementation."
                        ),
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
                        message=(
                            f"Resolved PRODUCT-to-{target_class} edge has unknown lifecycle phase; "
                            "do not treat it as absence of violation."
                        ),
                        evidence_ids=(edge.evidence_id,),
                        source_component=source_component,
                        target_component=target_component,
                    )
                )

        if (
            source_class in NON_PRODUCT_IMPLEMENTATION_CLASSES
            and target_class == "PRODUCT"
            and edge.phase == DependencyPhase.UNKNOWN
        ):
            findings.append(
                RuleFinding(
                    rule_id="PTSIP-DEP-002",
                    severity="REVIEW",
                    message=(
                        f"Resolved {source_class}-to-PRODUCT edge requires purpose/phase review; "
                        "direction alone does not prove that it is bounded lifecycle work."
                    ),
                    evidence_ids=(edge.evidence_id,),
                    source_component=source_component,
                    target_component=target_component,
                )
            )

    return findings


def evaluate_component_dependency_policy(
    policy: dict[str, object] | None,
    components: list[dict[str, object]],
    partition: ComponentPartition,
    dependencies: DependencyScan,
) -> list[ProjectPolicyFinding]:
    if not isinstance(policy, dict):
        return []

    _classifications, owners = _component_context(components, partition)
    default = str(policy.get("default", "allow"))
    allowed = {
        (str(item.get("from")), str(item.get("to")))
        for item in policy.get("allow", [])
        if isinstance(item, dict) and item.get("from") and item.get("to")
    }
    denied = {
        (str(item.get("from")), str(item.get("to")))
        for item in policy.get("deny", [])
        if isinstance(item, dict) and item.get("from") and item.get("to")
    }

    findings: list[ProjectPolicyFinding] = []
    for edge in dependencies.edges:
        if not edge.resolved_path:
            continue
        source_component = owners.get(edge.source)
        target_component = owners.get(edge.resolved_path)
        if not source_component or not target_component or source_component == target_component:
            continue
        pair = (source_component, target_component)
        permitted = pair in allowed if default == "deny" else pair not in denied
        if pair in denied:
            permitted = False
        if pair in allowed:
            permitted = True
        if permitted:
            continue
        findings.append(
            ProjectPolicyFinding(
                policy_id="component_dependency_policy",
                message="Resolved cross-component dependency violates the declared project-specific component dependency policy.",
                evidence_ids=(edge.evidence_id,),
                source_component=source_component,
                target_component=target_component,
            )
        )
    return findings
