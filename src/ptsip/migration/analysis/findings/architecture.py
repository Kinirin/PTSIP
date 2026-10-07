from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

class ArchitectureFindingKind(StrEnum):
    STALE_SOURCE_DECLARATION = "STALE_SOURCE_DECLARATION"
    NEW_REPOSITORY_CANDIDATE = "NEW_REPOSITORY_CANDIDATE"
    AMBIGUOUS_SOURCE_COVERAGE = "AMBIGUOUS_SOURCE_COVERAGE"
    MISSING_RELATIONSHIP = "MISSING_RELATIONSHIP"
    MISSING_ASSOCIATED_ARTIFACT = "MISSING_ASSOCIATED_ARTIFACT"
    EVIDENCE_CONFLICT = "EVIDENCE_CONFLICT"
    EVIDENCE_INCOMPLETE = "EVIDENCE_INCOMPLETE"

@dataclass(frozen=True)
class ArchitectureFinding:
    subject_id: str
    kind: ArchitectureFindingKind
    rationale: str
    evidence_ids: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "subject_id": self.subject_id,
            "kind": self.kind.value,
            "rationale": self.rationale,
            "evidence_ids": list(self.evidence_ids),
        }

