from __future__ import annotations

from dataclasses import asdict, dataclass
from enum import StrEnum


class Classification(StrEnum):
    PRODUCT = "PRODUCT"
    DEVELOPMENT_TOOLING = "DEVELOPMENT_TOOLING"
    DELIVERY = "DELIVERY"
    OPERATIONS = "OPERATIONS"
    NEUTRAL_CONTRACT = "NEUTRAL_CONTRACT"


class ResponsibilityRole(StrEnum):
    IMPLEMENTATION = "IMPLEMENTATION"
    VERIFICATION = "VERIFICATION"
    AUTOMATION = "AUTOMATION"
    CONFIGURATION = "CONFIGURATION"
    DOCUMENTATION = "DOCUMENTATION"
    GOVERNANCE = "GOVERNANCE"


class ResponsibilityRelationshipType(StrEnum):
    IMPORTS = "IMPORTS"
    LINKS = "LINKS"
    LOADS = "LOADS"
    INVOKES = "INVOKES"
    READS = "READS"
    GENERATES = "GENERATES"
    BUILDS = "BUILDS"
    PACKAGES = "PACKAGES"
    PUBLISHES = "PUBLISHES"
    DEPLOYS = "DEPLOYS"
    VERIFIES = "VERIFIES"
    MANAGES = "MANAGES"
    DOCUMENTS = "DOCUMENTS"
    SPECIFIES = "SPECIFIES"
    GOVERNS = "GOVERNS"


class DecisionStatus(StrEnum):
    RESOLVED = "RESOLVED"
    UNKNOWN = "UNKNOWN"
    CONFLICT = "CONFLICT"
    INCOMPLETE = "INCOMPLETE"


class DecisionOrigin(StrEnum):
    PROFILE = "PROFILE"
    AGENT = "AGENT"
    INFERRED = "INFERRED"


# Evidence-edge vocabulary is intentionally distinct from project-owned
# Responsibility Map relationship vocabulary. TESTS remains an evidence term;
# a project-owned declaration uses VERIFIES after architecture confirmation.
class EdgeType(StrEnum):
    IMPORTS = "IMPORTS"
    LINKS = "LINKS"
    LOADS = "LOADS"
    INVOKES = "INVOKES"
    READS = "READS"
    GENERATES = "GENERATES"
    PACKAGES = "PACKAGES"
    TESTS = "TESTS"
    PUBLISHES = "PUBLISHES"


class DependencyPhase(StrEnum):
    RUNTIME = "RUNTIME"
    BUILD = "BUILD"
    TEST = "TEST"
    RELEASE = "RELEASE"
    INSPECTION = "INSPECTION"
    UNKNOWN = "UNKNOWN"


class ResolutionStatus(StrEnum):
    RESOLVED = "RESOLVED"
    EXTERNAL = "EXTERNAL"
    UNRESOLVED = "UNRESOLVED"
    DYNAMIC = "DYNAMIC"


class EvidenceNodeScope(StrEnum):
    PROJECT_COMPONENT = "PROJECT_COMPONENT"
    EXTERNAL_DEPENDENCY = "EXTERNAL_DEPENDENCY"
    PLATFORM = "PLATFORM"
    UNRESOLVED_TARGET = "UNRESOLVED_TARGET"


class EvidenceProvenance(StrEnum):
    DECLARED = "DECLARED"
    OBSERVED = "OBSERVED"
    INFERRED = "INFERRED"


@dataclass(frozen=True)
class AssociatedArtifact:
    id: str
    anchor: str
    include: tuple[str, ...]
    purpose: str
    exclude: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ResponsibilityRelationship:
    id: str
    source: str
    target: str
    relationship_type: ResponsibilityRelationshipType

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["type"] = self.relationship_type.value
        payload["from"] = payload.pop("source")
        payload["to"] = payload.pop("target")
        payload.pop("relationship_type", None)
        return payload


@dataclass(frozen=True)
class ClassificationDecision:
    component_id: str
    status: DecisionStatus
    classification: Classification | None
    origin: DecisionOrigin
    confidence: float | None
    evidence_ids: tuple[str, ...]
    rationale: str
    counter_evidence: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class DependencyEdge:
    evidence_id: str
    source: str
    target: str
    edge_type: EdgeType
    phase: DependencyPhase
    resolution: ResolutionStatus
    target_scope: EvidenceNodeScope
    provenance: EvidenceProvenance = EvidenceProvenance.OBSERVED
    line: int | None = None
    resolved_path: str | None = None
    adapter: str = "unknown"
    working_directory: str | None = None
    note: str | None = None

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["edge_type"] = self.edge_type.value
        payload["phase"] = self.phase.value
        payload["resolution"] = self.resolution.value
        payload["target_scope"] = self.target_scope.value
        payload["provenance"] = self.provenance.value
        return payload
