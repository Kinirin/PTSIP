from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Iterable

from ptsip.source_compat.model import FrozenJson, freeze_json, thaw_json
from ptsip.migration.proposal.semantic_identity import canonical_semantics, semantic_digest

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

