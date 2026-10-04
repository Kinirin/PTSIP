from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

from ptsip.migration.analysis.source.evidence_correlation import EvidenceCorrelation
from ptsip.migration.analysis.target.compatibility import TargetCompatibility

class ObligationCategory(StrEnum):
    REQUIRED = "REQUIRED"
    REMOVAL = "REMOVAL"
    ASYNC = "ASYNC"

@dataclass(frozen=True)
class RequiredWorkElement:
    id: str
    path: str
    source_declaration_id: str
    source_classification: str | None
    selector: str
    evidence: EvidenceCorrelation
    target_status: TargetCompatibility
    resolved: bool

    @property
    def category(self) -> ObligationCategory:
        return ObligationCategory.REQUIRED

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "category": self.category.value,
            "path": self.path,
            "source_declaration_id": self.source_declaration_id,
            "source_classification": self.source_classification,
            "selector": self.selector,
            "evidence": self.evidence.as_dict(),
            "target_status": self.target_status.value,
            "resolved": self.resolved,
        }

@dataclass(frozen=True)
class RemovalMigrationElement:
    id: str
    source_declaration_id: str
    source_classification: str | None
    selector: str
    rationale: str

    @property
    def category(self) -> ObligationCategory:
        return ObligationCategory.REMOVAL

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "category": self.category.value,
            "source_declaration_id": self.source_declaration_id,
            "source_classification": self.source_classification,
            "selector": self.selector,
            "rationale": self.rationale,
        }

@dataclass(frozen=True)
class AsynchronousWorkTarget:
    id: str
    path: str
    evidence: EvidenceCorrelation

    @property
    def category(self) -> ObligationCategory:
        return ObligationCategory.ASYNC

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "category": self.category.value,
            "path": self.path,
            "evidence": self.evidence.as_dict(),
        }

@dataclass(frozen=True)
class SourceMigrationCompletion:
    required_total: int
    required_resolved: int
    required_unresolved: int
    removal_count: int
    async_count: int

    @property
    def complete(self) -> bool:
        return self.required_unresolved == 0

    def as_dict(self) -> dict[str, object]:
        return {
            "required_total": self.required_total,
            "required_resolved": self.required_resolved,
            "required_unresolved": self.required_unresolved,
            "removal_count": self.removal_count,
            "async_count": self.async_count,
            "complete": self.complete,
        }

