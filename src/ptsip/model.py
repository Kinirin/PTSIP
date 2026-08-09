from __future__ import annotations

from dataclasses import asdict, dataclass
from enum import StrEnum


class Classification(StrEnum):
    PRODUCT = "PRODUCT"
    TOOLCHAIN = "TOOLCHAIN"
    NEUTRAL_CONTRACT = "NEUTRAL_CONTRACT"


class DecisionStatus(StrEnum):
    RESOLVED = "RESOLVED"
    UNKNOWN = "UNKNOWN"
    CONFLICT = "CONFLICT"
    INCOMPLETE = "INCOMPLETE"


class DecisionOrigin(StrEnum):
    PROFILE = "PROFILE"
    AGENT = "AGENT"
    INFERRED = "INFERRED"


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
