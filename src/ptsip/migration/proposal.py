from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from enum import StrEnum
from typing import Iterable, Mapping

from ..source_compat.model import FrozenJson, SourceGenerationBinding, freeze_json, thaw_json


_SET_FIELDS = frozenset({"roles", "include", "exclude", "manifests", "consumers", "analysis_inputs"})


class TargetEntityKind(StrEnum):
    COMPONENT = "COMPONENT"
    ASSOCIATED_ARTIFACT = "ASSOCIATED_ARTIFACT"
    RELATIONSHIP = "RELATIONSHIP"
    COMPONENT_DEPENDENCY_POLICY = "COMPONENT_DEPENDENCY_POLICY"
    POLICIES = "POLICIES"


class DeltaChangeKind(StrEnum):
    ADD = "ADD"
    REMOVE = "REMOVE"
    REPLACE = "REPLACE"


class ProposalPurpose(StrEnum):
    REQUIRED_MIGRATION = "REQUIRED_MIGRATION"
    TARGET_VALIDITY = "TARGET_VALIDITY"
    ASYNC_OPTIONAL = "ASYNC_OPTIONAL"
    ADVISORY = "ADVISORY"


class ProposalAuthority(StrEnum):
    PROPOSAL_ONLY = "PROPOSAL_ONLY"
    ACCEPTED_PROJECT_DECISION = "ACCEPTED_PROJECT_DECISION"
    UNRESOLVED = "UNRESOLVED"


def _canonical(value: object, *, field_name: str | None = None) -> object:
    if isinstance(value, Mapping):
        return {
            str(key): _canonical(item, field_name=str(key))
            for key, item in sorted(value.items(), key=lambda pair: str(pair[0]))
        }
    if isinstance(value, (list, tuple)):
        items = [_canonical(item) for item in value]
        if field_name in _SET_FIELDS:
            items.sort(
                key=lambda item: json.dumps(
                    item,
                    sort_keys=True,
                    separators=(",", ":"),
                    ensure_ascii=False,
                )
            )
        return items
    return value


def canonical_semantics(value: object) -> object:
    return _canonical(value)


def semantic_digest(value: object) -> str:
    encoded = json.dumps(
        canonical_semantics(value),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


@dataclass(frozen=True)
class TargetDelta:
    id: str
    entity_kind: TargetEntityKind
    entity_id: str
    change_kind: DeltaChangeKind
    before: FrozenJson | None
    after: FrozenJson | None
    obligation_ids: tuple[str, ...]
    evidence_ids: tuple[str, ...] = ()

    @classmethod
    def build(
        cls,
        *,
        entity_kind: TargetEntityKind,
        entity_id: str,
        change_kind: DeltaChangeKind,
        before: object | None,
        after: object | None,
        obligation_ids: Iterable[str] = (),
        evidence_ids: Iterable[str] = (),
    ) -> "TargetDelta":
        obligations = tuple(sorted(set(str(item) for item in obligation_ids)))
        evidence = tuple(sorted(set(str(item) for item in evidence_ids)))
        before_semantics = canonical_semantics(before)
        after_semantics = canonical_semantics(after)
        digest = semantic_digest(
            {
                "entity_kind": entity_kind.value,
                "entity_id": entity_id,
                "change_kind": change_kind.value,
                "before": before_semantics,
                "after": after_semantics,
                "obligation_ids": obligations,
                "evidence_ids": evidence,
            }
        )[:24]
        return cls(
            id=f"delta:{digest}",
            entity_kind=entity_kind,
            entity_id=entity_id,
            change_kind=change_kind,
            before=freeze_json(before_semantics) if before is not None else None,
            after=freeze_json(after_semantics) if after is not None else None,
            obligation_ids=obligations,
            evidence_ids=evidence,
        )

    def before_value(self) -> object | None:
        return thaw_json(self.before) if self.before is not None else None

    def after_value(self) -> object | None:
        return thaw_json(self.after) if self.after is not None else None

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "entity_kind": self.entity_kind.value,
            "entity_id": self.entity_id,
            "change_kind": self.change_kind.value,
            "before": self.before_value(),
            "after": self.after_value(),
            "obligation_ids": list(self.obligation_ids),
            "evidence_ids": list(self.evidence_ids),
        }


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
