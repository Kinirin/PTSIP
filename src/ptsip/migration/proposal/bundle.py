from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Iterable

from ptsip.source_compat.model import SourceGenerationBinding
from ptsip.migration.proposal.semantic_identity import semantic_digest
from ptsip.migration.proposal.target_delta import TargetDelta

class ProposalPurpose(StrEnum):
    REQUIRED_MIGRATION = "REQUIRED_MIGRATION"
    TARGET_VALIDITY = "TARGET_VALIDITY"
    ASYNC_OPTIONAL = "ASYNC_OPTIONAL"
    ADVISORY = "ADVISORY"

class ProposalAuthority(StrEnum):
    PROPOSAL_ONLY = "PROPOSAL_ONLY"
    ACCEPTED_PROJECT_DECISION = "ACCEPTED_PROJECT_DECISION"
    UNRESOLVED = "UNRESOLVED"

def _bundle_id(
    source: SourceGenerationBinding,
    analysis_digest: str,
    deltas: Iterable[TargetDelta],
    purpose: Iterable[ProposalPurpose],
    rationale: str,
) -> str:
    payload = {
        "source": source.as_dict(),
        "analysis_digest": analysis_digest,
        "deltas": [item.as_dict() for item in sorted(deltas, key=lambda item: item.id)],
        "purpose": sorted(item.value for item in purpose),
        "rationale": rationale,
    }
    return "bundle:" + semantic_digest(payload)[:24]

@dataclass(frozen=True)
class ProposalBundle:
    id: str
    source_generation: SourceGenerationBinding
    analysis_digest: str
    deltas: tuple[TargetDelta, ...]
    purpose: tuple[ProposalPurpose, ...]
    rationale: str
    alternative_group: str | None = None
    authority: ProposalAuthority = ProposalAuthority.PROPOSAL_ONLY

    @classmethod
    def build(
        cls,
        *,
        source_generation: SourceGenerationBinding,
        analysis_digest: str,
        deltas: Iterable[TargetDelta],
        purpose: Iterable[ProposalPurpose],
        rationale: str,
        alternative_group: str | None = None,
    ) -> "ProposalBundle":
        delta_rows = tuple(sorted(deltas, key=lambda item: item.id))
        purposes = tuple(sorted(set(purpose), key=lambda item: item.value))
        return cls(
            id=_bundle_id(source_generation, analysis_digest, delta_rows, purposes, rationale),
            source_generation=source_generation,
            analysis_digest=analysis_digest,
            deltas=delta_rows,
            purpose=purposes,
            rationale=rationale,
            alternative_group=alternative_group,
        )

    @property
    def blocking(self) -> bool:
        return any(
            item in {ProposalPurpose.REQUIRED_MIGRATION, ProposalPurpose.TARGET_VALIDITY}
            for item in self.purpose
        )

    @property
    def async_only(self) -> bool:
        return bool(self.purpose) and all(item == ProposalPurpose.ASYNC_OPTIONAL for item in self.purpose)

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "authority": self.authority.value,
            "source_generation": self.source_generation.as_dict(),
            "analysis_digest": self.analysis_digest,
            "deltas": [item.as_dict() for item in self.deltas],
            "purpose": [item.value for item in self.purpose],
            "rationale": self.rationale,
            "alternative_group": self.alternative_group,
        }

@dataclass(frozen=True)
class AcceptedDeltaBundle:
    proposal_id: str
    source_generation: SourceGenerationBinding
    analysis_digest: str
    deltas: tuple[TargetDelta, ...]
    purpose: tuple[ProposalPurpose, ...]
    rationale: str
    decision_id: str
    authority: ProposalAuthority = ProposalAuthority.ACCEPTED_PROJECT_DECISION

    @classmethod
    def from_proposal(cls, proposal: ProposalBundle, *, decision_id: str) -> "AcceptedDeltaBundle":
        decision = decision_id.strip()
        if not decision:
            raise ValueError("Accepted target delta requires a non-empty project-owned decision identity.")
        return cls(
            proposal_id=proposal.id,
            source_generation=proposal.source_generation,
            analysis_digest=proposal.analysis_digest,
            deltas=proposal.deltas,
            purpose=proposal.purpose,
            rationale=proposal.rationale,
            decision_id=decision,
        )

    @property
    def blocking(self) -> bool:
        return any(
            item in {ProposalPurpose.REQUIRED_MIGRATION, ProposalPurpose.TARGET_VALIDITY}
            for item in self.purpose
        )

    @property
    def async_only(self) -> bool:
        return bool(self.purpose) and all(item == ProposalPurpose.ASYNC_OPTIONAL for item in self.purpose)

    def as_dict(self) -> dict[str, object]:
        return {
            "proposal_id": self.proposal_id,
            "authority": self.authority.value,
            "source_generation": self.source_generation.as_dict(),
            "analysis_digest": self.analysis_digest,
            "deltas": [item.as_dict() for item in self.deltas],
            "purpose": [item.value for item in self.purpose],
            "rationale": self.rationale,
            "decision_id": self.decision_id,
        }

@dataclass(frozen=True)
class UnresolvedBundle:
    id: str
    source_generation: SourceGenerationBinding
    analysis_digest: str
    subject_ids: tuple[str, ...]
    purpose: tuple[ProposalPurpose, ...]
    question: str
    alternative_proposal_ids: tuple[str, ...] = ()
    authority: ProposalAuthority = ProposalAuthority.UNRESOLVED

    @classmethod
    def build(
        cls,
        *,
        source_generation: SourceGenerationBinding,
        analysis_digest: str,
        subject_ids: Iterable[str],
        purpose: Iterable[ProposalPurpose],
        question: str,
        alternative_proposal_ids: Iterable[str] = (),
    ) -> "UnresolvedBundle":
        subjects = tuple(sorted(set(str(item) for item in subject_ids)))
        purposes = tuple(sorted(set(purpose), key=lambda item: item.value))
        alternatives = tuple(sorted(set(str(item) for item in alternative_proposal_ids)))
        digest = semantic_digest(
            {
                "source": source_generation.as_dict(),
                "analysis_digest": analysis_digest,
                "subject_ids": subjects,
                "purpose": [item.value for item in purposes],
                "question": question,
                "alternative_proposal_ids": alternatives,
            }
        )[:24]
        return cls(
            id=f"unresolved:{digest}",
            source_generation=source_generation,
            analysis_digest=analysis_digest,
            subject_ids=subjects,
            purpose=purposes,
            question=question,
            alternative_proposal_ids=alternatives,
        )

    @property
    def blocking(self) -> bool:
        return any(
            item in {ProposalPurpose.REQUIRED_MIGRATION, ProposalPurpose.TARGET_VALIDITY}
            for item in self.purpose
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "authority": self.authority.value,
            "source_generation": self.source_generation.as_dict(),
            "analysis_digest": self.analysis_digest,
            "subject_ids": list(self.subject_ids),
            "purpose": [item.value for item in self.purpose],
            "question": self.question,
            "alternative_proposal_ids": list(self.alternative_proposal_ids),
        }

