from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol

from ptsip.migration.execution.authorization.binding import BoundExecutionPlan
from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.state.phase import ExecutionPhase
from ptsip.migration.proposal.semantic_identity import semantic_digest

@dataclass(frozen=True)
class AuthorizationProof:
    plan_digest: str
    decision_ids: tuple[str, ...]
    authority_revision: str
    proof_id: str

    def as_dict(self) -> dict[str, object]:
        return {
            "plan_digest": self.plan_digest,
            "decision_ids": list(self.decision_ids),
            "authority_revision": self.authority_revision,
            "proof_id": self.proof_id,
        }

@dataclass(frozen=True)
class AuthorizedExecutionPlan:
    bound: BoundExecutionPlan
    authorization: AuthorizationProof
    phase: ExecutionPhase = ExecutionPhase.AUTHORIZED

class AuthorityHeadStore(Protocol):
    def ensure_head(self) -> str: ...

def build_authorization_proof(
    bound: BoundExecutionPlan,
    *,
    decision_ids: tuple[str, ...],
    authority_revision: str,
) -> AuthorizationProof:
    decisions = tuple(sorted(set(item.strip() for item in decision_ids if item.strip())))
    required = tuple(sorted({item for source in bound.sources for item in source.decision_ids}))
    if decisions != required:
        raise ExecutionStateError("Authorization decision set must exactly match accepted WU-06 decision identities.")
    revision = authority_revision.strip()
    if not revision:
        raise ExecutionStateError("Authorization proof requires a non-empty authority revision.")
    proof_id = "authorization:" + semantic_digest(
        {"plan_digest": bound.plan_digest, "decision_ids": decisions, "authority_revision": revision}
    )[:24]
    return AuthorizationProof(bound.plan_digest, decisions, revision, proof_id)

def authorize_execution(
    bound: BoundExecutionPlan,
    proof: AuthorizationProof,
    ledger: CheckpointLedger,
    *,
    authority_store: AuthorityHeadStore | None = None,
) -> AuthorizedExecutionPlan:
    if proof.plan_digest != bound.plan_digest:
        raise ExecutionStateError("Authorization proof is bound to a different WU-06 plan.")
    required = tuple(sorted({item for source in bound.sources for item in source.decision_ids}))
    if tuple(sorted(proof.decision_ids)) != required:
        raise ExecutionStateError("Authorization proof no longer covers the exact accepted decision set.")
    if authority_store is not None and authority_store.ensure_head() != proof.authority_revision:
        raise ExecutionStateError("Coordinated authority revision changed after authorization.")
    if ledger.plan_digest != bound.plan_digest:
        raise ExecutionStateError("Checkpoint ledger belongs to a different WU-06 plan.")
    latest = ledger.latest()
    if latest is not None:
        raise ExecutionStateError("Execution ledger is not empty; use recovery inspection before resuming.")
    ledger.append(phase=ExecutionPhase.PLAN_BOUND, payload=bound.as_dict())
    ledger.append(
        phase=ExecutionPhase.AUTHORIZED,
        decision_ids=proof.decision_ids,
        payload=proof.as_dict(),
    )
    return AuthorizedExecutionPlan(bound, proof)

