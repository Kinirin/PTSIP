from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

class LifecycleFindingKind(StrEnum):
    EXACT_SEMANTIC_PRESERVATION = "EXACT_SEMANTIC_PRESERVATION"
    HISTORICAL_TOOLCHAIN_AMBIGUITY = "HISTORICAL_TOOLCHAIN_AMBIGUITY"
    POSSIBLE_LIFECYCLE_SEPARATION = "POSSIBLE_LIFECYCLE_SEPARATION"
    TARGET_REVIEW_REQUIRED = "TARGET_REVIEW_REQUIRED"

@dataclass(frozen=True)
class LifecycleFinding:
    subject_id: str
    kind: LifecycleFindingKind
    source_classification: str | None
    target_classification: str | None
    rationale: str

    def as_dict(self) -> dict[str, object]:
        return {
            "subject_id": self.subject_id,
            "kind": self.kind.value,
            "source_classification": self.source_classification,
            "target_classification": self.target_classification,
            "rationale": self.rationale,
        }

