from dataclasses import replace

from ptsip.dependency_reconciliation import reconcile_dependency_phases
from ptsip.inspection.dependencies import DependencyScan
from ptsip.model import DependencyEdge, DependencyPhase, EdgeType, EvidenceNodeScope, ResolutionStatus
from ptsip.validation.components import ComponentAssignment, ComponentPartition
from ptsip.conformance import _dependency_coverage_gaps


def test_verification_phase_does_not_erase_unknown_dependencies_or_infer_lifecycle():
    component = {"id": "checks", "classification": "PRODUCT", "roles": ["VERIFICATION"],
                 "shipped": False, "runtime_required": False}
    partition = ComponentPartition((ComponentAssignment("check.py", "checks", "check.py"),), (), (), (), ())
    edge = DependencyEdge("test:1", "check.py", "missing_test_dep", EdgeType.IMPORTS,
                          DependencyPhase.UNKNOWN, ResolutionStatus.UNRESOLVED, EvidenceNodeScope.UNRESOLVED_TARGET)
    scan = DependencyScan((edge,), (), ("python",))
    phased = reconcile_dependency_phases(scan, [component], partition)
    assert phased.edges[0].phase == DependencyPhase.TEST
    assert phased.edges[0].resolution == ResolutionStatus.UNRESOLVED
    gap = _dependency_coverage_gaps(phased, [component], partition)[0]
    assert gap["blocking"] and gap["id"].startswith("verification-dependency-target:")
    assert "PTSIP-BLD-001" in gap["rule_ids"]
    for override in ({"shipped": True}, {"runtime_required": True}, {"roles": ["VERIFICATION", "IMPLEMENTATION"]}):
        assert reconcile_dependency_phases(scan, [{**component, **override}], partition).edges[0].phase == DependencyPhase.UNKNOWN


def test_runtime_reachability_prevents_verification_role_escape():
    components = [{"id": "app", "classification": "PRODUCT", "shipped": True, "runtime_required": True},
                  {"id": "checks", "classification": "PRODUCT", "roles": ["VERIFICATION"],
                   "shipped": False, "runtime_required": False}]
    partition = ComponentPartition(tuple(ComponentAssignment(path, owner, path) for path, owner in (
        ("app.py", "app"), ("check.py", "checks"))), (), (), (), ())
    edge = DependencyEdge("test:1", "check.py", "missing_test_dep", EdgeType.IMPORTS,
                          DependencyPhase.UNKNOWN, ResolutionStatus.UNRESOLVED, EvidenceNodeScope.UNRESOLVED_TARGET)
    incoming = replace(edge, evidence_id="app:1", source="app.py", target="check", resolved_path="check.py",
                       resolution=ResolutionStatus.RESOLVED, target_scope=EvidenceNodeScope.PROJECT_COMPONENT)
    scan = DependencyScan((edge, incoming), (), ("python",))
    assert reconcile_dependency_phases(scan, components, partition).edges[0].phase == DependencyPhase.UNKNOWN
