from __future__ import annotations

from dataclasses import dataclass
from typing import Mapping

from ptsip.migration.execution.guard.repository_snapshot import RepositorySnapshotExpectation
from ptsip.migration.execution.state.source_steps import ExecutionPhase
from ptsip.migration.proposal.semantic_identity import canonical_semantics, semantic_digest

LEDGER_FORMAT = "ptsip-migration-checkpoint-ledger/v1"

class LedgerIntegrityError(RuntimeError):
    pass

@dataclass(frozen=True)
class CheckpointRecord:
    sequence: int
    phase: ExecutionPhase
    plan_digest: str
    source_path: str | None
    source_sha256: str | None
    final_point_before_sha256: str | None
    final_point_after_sha256: str | None
    analysis_digest: str | None
    decision_ids: tuple[str, ...]
    repository_snapshot: RepositorySnapshotExpectation | None
    payload: object
    previous_digest: str | None
    digest: str
    format: str = LEDGER_FORMAT

    @classmethod
    def build(
        cls,
        *,
        sequence: int,
        phase: ExecutionPhase,
        plan_digest: str,
        source_path: str | None = None,
        source_sha256: str | None = None,
        final_point_before_sha256: str | None = None,
        final_point_after_sha256: str | None = None,
        analysis_digest: str | None = None,
        decision_ids: tuple[str, ...] = (),
        repository_snapshot: RepositorySnapshotExpectation | None = None,
        payload: object = None,
        previous_digest: str | None = None,
    ) -> "CheckpointRecord":
        body = {
            "format": LEDGER_FORMAT,
            "sequence": sequence,
            "phase": phase.value,
            "plan_digest": plan_digest,
            "source_path": source_path,
            "source_sha256": source_sha256,
            "final_point_before_sha256": final_point_before_sha256,
            "final_point_after_sha256": final_point_after_sha256,
            "analysis_digest": analysis_digest,
            "decision_ids": sorted(set(decision_ids)),
            "repository_snapshot": repository_snapshot.as_dict() if repository_snapshot else None,
            "payload": canonical_semantics(payload),
            "previous_digest": previous_digest,
        }
        return cls(
            sequence=sequence,
            phase=phase,
            plan_digest=plan_digest,
            source_path=source_path,
            source_sha256=source_sha256,
            final_point_before_sha256=final_point_before_sha256,
            final_point_after_sha256=final_point_after_sha256,
            analysis_digest=analysis_digest,
            decision_ids=tuple(body["decision_ids"]),
            repository_snapshot=repository_snapshot,
            payload=body["payload"],
            previous_digest=previous_digest,
            digest=semantic_digest(body),
        )

    def body(self) -> dict[str, object]:
        return {
            "format": self.format,
            "sequence": self.sequence,
            "phase": self.phase.value,
            "plan_digest": self.plan_digest,
            "source_path": self.source_path,
            "source_sha256": self.source_sha256,
            "final_point_before_sha256": self.final_point_before_sha256,
            "final_point_after_sha256": self.final_point_after_sha256,
            "analysis_digest": self.analysis_digest,
            "decision_ids": list(self.decision_ids),
            "repository_snapshot": self.repository_snapshot.as_dict() if self.repository_snapshot else None,
            "payload": canonical_semantics(self.payload),
            "previous_digest": self.previous_digest,
        }

    def as_dict(self) -> dict[str, object]:
        result = self.body()
        result["digest"] = self.digest
        return result

    @classmethod
    def from_mapping(cls, payload: Mapping[str, object]) -> "CheckpointRecord":
        if payload.get("format") != LEDGER_FORMAT:
            raise LedgerIntegrityError("Unsupported migration ledger format.")
        snapshot_payload = payload.get("repository_snapshot")
        snapshot = None
        if snapshot_payload is not None:
            if not isinstance(snapshot_payload, Mapping):
                raise LedgerIntegrityError("Checkpoint repository_snapshot must be a mapping.")
            snapshot = RepositorySnapshotExpectation(
                snapshot_payload.get("head") if isinstance(snapshot_payload.get("head"), str) else None,
                str(snapshot_payload.get("status_fingerprint", "")),
                str(snapshot_payload.get("tracked_content_fingerprint", "")),
            )
        try:
            phase = ExecutionPhase(str(payload["phase"]))
            sequence = int(payload["sequence"])
            plan_digest = str(payload["plan_digest"])
            digest = str(payload["digest"])
        except (KeyError, TypeError, ValueError) as exc:
            raise LedgerIntegrityError(f"Invalid checkpoint shape: {exc}") from exc
        row = cls(
            sequence=sequence,
            phase=phase,
            plan_digest=plan_digest,
            source_path=str(payload["source_path"]) if payload.get("source_path") is not None else None,
            source_sha256=str(payload["source_sha256"]) if payload.get("source_sha256") is not None else None,
            final_point_before_sha256=(
                str(payload["final_point_before_sha256"])
                if payload.get("final_point_before_sha256") is not None
                else None
            ),
            final_point_after_sha256=(
                str(payload["final_point_after_sha256"])
                if payload.get("final_point_after_sha256") is not None
                else None
            ),
            analysis_digest=str(payload["analysis_digest"]) if payload.get("analysis_digest") is not None else None,
            decision_ids=tuple(sorted(str(item) for item in payload.get("decision_ids", []))),
            repository_snapshot=snapshot,
            payload=canonical_semantics(payload.get("payload")),
            previous_digest=str(payload["previous_digest"]) if payload.get("previous_digest") is not None else None,
            digest=digest,
        )
        if semantic_digest(row.body()) != row.digest:
            raise LedgerIntegrityError(f"Checkpoint {sequence} digest does not match its content.")
        return row

