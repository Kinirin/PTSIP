from __future__ import annotations

from dataclasses import dataclass

from ptsip.source_compat.model import SourceGenerationBinding
from ptsip.migration.proposal.bundle import AcceptedDeltaBundle, ProposalBundle, UnresolvedBundle
from ptsip.migration.proposal.semantic_identity import semantic_digest

@dataclass(frozen=True)
class SourceProposalSet:
    source_generation: SourceGenerationBinding
    analysis_digest: str
    suggested: tuple[ProposalBundle, ...]
    accepted: tuple[AcceptedDeltaBundle, ...]
    unresolved: tuple[UnresolvedBundle, ...]
    no_change_obligation_ids: tuple[str, ...] = ()
    ignored_async_ids: tuple[str, ...] = ()
    issues: tuple[str, ...] = ()

    @property
    def blocking_proposal_ids(self) -> tuple[str, ...]:
        return tuple(sorted(item.id for item in self.suggested if item.blocking))

    @property
    def blocking_unresolved_ids(self) -> tuple[str, ...]:
        return tuple(sorted(item.id for item in self.unresolved if item.blocking))

    def content_payload(self) -> dict[str, object]:
        return {
            "source_generation": self.source_generation.as_dict(),
            "analysis_digest": self.analysis_digest,
            "suggested": [item.as_dict() for item in sorted(self.suggested, key=lambda item: item.id)],
            "accepted": [item.as_dict() for item in sorted(self.accepted, key=lambda item: item.proposal_id)],
            "unresolved": [item.as_dict() for item in sorted(self.unresolved, key=lambda item: item.id)],
            "no_change_obligation_ids": list(sorted(self.no_change_obligation_ids)),
            "ignored_async_ids": list(sorted(self.ignored_async_ids)),
            "issues": list(sorted(self.issues)),
        }

    @property
    def deterministic_digest(self) -> str:
        return semantic_digest(self.content_payload())

    def as_dict(self) -> dict[str, object]:
        payload = self.content_payload()
        payload["deterministic_digest"] = self.deterministic_digest
        payload["blocking_proposal_ids"] = list(self.blocking_proposal_ids)
        payload["blocking_unresolved_ids"] = list(self.blocking_unresolved_ids)
        return payload

