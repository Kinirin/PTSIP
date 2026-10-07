from __future__ import annotations

from dataclasses import dataclass

from ptsip.evidence.contract import EvidenceChannelStatus, EvidenceRecordStatus, NormalizedEvidenceSet
from ptsip.validation.components import normalize_selector

@dataclass(frozen=True)
class EvidenceCorrelation:
    semantic_ids: tuple[str, ...]
    conflict_ids: tuple[str, ...]
    incomplete_channels: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "semantic_ids": list(self.semantic_ids),
            "conflict_ids": list(self.conflict_ids),
            "incomplete_channels": list(self.incomplete_channels),
        }

def _evidence_for_path(evidence: NormalizedEvidenceSet, path: str) -> EvidenceCorrelation:
    semantic_ids: set[str] = set()
    conflict_ids: set[str] = set()
    normalized_path = normalize_selector(path)
    for record in evidence.records:
        matched = record.subject in {normalized_path, f"path:{normalized_path}"}
        qualifier_path = record.qualifiers.get("path")
        if isinstance(qualifier_path, str) and normalize_selector(qualifier_path) == normalized_path:
            matched = True
        for assertion in record.assertions:
            for origin in assertion.origins:
                if origin.source_path and normalize_selector(origin.source_path) == normalized_path:
                    matched = True
        if matched:
            semantic_ids.add(record.semantic_id)
            if record.status == EvidenceRecordStatus.CONFLICT:
                conflict_ids.add(record.semantic_id)
    incomplete = tuple(sorted(channel.id for channel in evidence.channels if channel.status in {EvidenceChannelStatus.FAILED, EvidenceChannelStatus.NOT_ANALYZED}))
    return EvidenceCorrelation(tuple(sorted(semantic_ids)), tuple(sorted(conflict_ids)), incomplete)

