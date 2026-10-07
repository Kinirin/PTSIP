from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

from ptsip.migration.planning.convergence.final_state import FinalPointEntity, FinalPointStateSnapshot
from ptsip.migration.proposal.semantic_identity import canonical_semantics
from ptsip.migration.proposal.target_delta import DeltaChangeKind, TargetDelta

class ReconciliationStatus(StrEnum):
    NO_CHANGE_REQUIRED = "NO_CHANGE_REQUIRED"
    ADD_TARGET_DECLARATION = "ADD_TARGET_DECLARATION"
    REPLACE_WITH_EXPLICIT_OWNER_DECISION = "REPLACE_WITH_EXPLICIT_OWNER_DECISION"
    CONFLICT_REQUIRES_CONFIRMATION = "CONFLICT_REQUIRES_CONFIRMATION"
    UNRESOLVED = "UNRESOLVED"

class DeletionGate(StrEnum):
    ALREADY_ELIGIBLE = "ALREADY_ELIGIBLE"
    REQUIRES_POST_APPLY_VERIFICATION = "REQUIRES_POST_APPLY_VERIFICATION"
    BLOCKED = "BLOCKED"

@dataclass(frozen=True)
class ReconciliationResult:
    bundle_id: str
    delta_id: str
    status: ReconciliationStatus
    rationale: str

    def as_dict(self) -> dict[str, str]:
        return {
            "bundle_id": self.bundle_id,
            "delta_id": self.delta_id,
            "status": self.status.value,
            "rationale": self.rationale,
        }

def _find_entity(state: FinalPointStateSnapshot | None, delta: TargetDelta) -> object | None:
    if state is None:
        return None
    for item in state.entities:
        if item.kind == delta.entity_kind and item.id == delta.entity_id:
            return canonical_semantics(item.payload)
    return None

def reconcile_delta(
    delta: TargetDelta,
    state: FinalPointStateSnapshot | None,
    *,
    accepted: bool,
    bundle_id: str,
) -> ReconciliationResult:
    existing = _find_entity(state, delta)
    before = canonical_semantics(delta.before_value())
    after = canonical_semantics(delta.after_value())

    if delta.change_kind == DeltaChangeKind.ADD:
        if existing is None:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.ADD_TARGET_DECLARATION,
                "Target entity is absent and the delta adds it.",
            )
        if existing == after:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.NO_CHANGE_REQUIRED,
                "Final Point already contains the proposed semantic state.",
            )
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
            "Final Point already contains a different entity with the same stable identity.",
        )

    if delta.change_kind == DeltaChangeKind.REMOVE:
        if existing is None:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.NO_CHANGE_REQUIRED,
                "Target entity is already absent.",
            )
        if before is not None and existing != before:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
                "Final Point entity does not match the state the removal was reviewed against.",
            )
        status = (
            ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION
            if accepted
            else ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION
        )
        return ReconciliationResult(
            bundle_id,
            delta.id,
            status,
            "Removal changes accepted target architecture and therefore requires an explicit project-owned decision.",
        )

    if existing == after:
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.NO_CHANGE_REQUIRED,
            "Final Point already contains the proposed replacement state.",
        )
    if existing is None:
        if before is None:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.ADD_TARGET_DECLARATION,
                "Replacement has no required prior state and the target entity is absent.",
            )
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
            "Expected entity to replace is absent from the Final Point.",
        )
    if before is not None and existing != before:
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
            "Final Point entity changed from the exact state the replacement was reviewed against.",
        )
    status = (
        ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION
        if accepted
        else ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION
    )
    return ReconciliationResult(
        bundle_id,
        delta.id,
        status,
        "Replacement changes target architecture and is executable only when bound to an explicit project-owned decision.",
    )

def _state_with_delta(
    state: FinalPointStateSnapshot,
    delta: TargetDelta,
    reconciliation: ReconciliationResult,
) -> FinalPointStateSnapshot:
    if reconciliation.status not in {
        ReconciliationStatus.ADD_TARGET_DECLARATION,
        ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION,
        ReconciliationStatus.NO_CHANGE_REQUIRED,
    }:
        return state
    if reconciliation.status == ReconciliationStatus.NO_CHANGE_REQUIRED:
        return state

    entities = {(item.kind, item.id): item for item in state.entities}
    key = (delta.entity_kind, delta.entity_id)
    if delta.change_kind == DeltaChangeKind.REMOVE:
        entities.pop(key, None)
    else:
        after = delta.after_value()
        if after is None:
            return state
        entities[key] = FinalPointEntity(delta.entity_kind, delta.entity_id, canonical_semantics(after))
    return FinalPointStateSnapshot(
        state.path,
        state.draft_version,
        state.specification_revision,
        None,
        tuple(sorted(entities.values(), key=lambda item: (item.kind.value, item.id))),
    )

