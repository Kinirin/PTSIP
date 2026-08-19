from __future__ import annotations

from pathlib import Path
from typing import Any

from ..clarification.resolution import DecisionAnswer, validate_answer
from ..clarification.resolution.model import CANONICAL_ANSWER_FIELDS
from ..storage.local_state import decision_store_path
from .store import DecisionRecord, DecisionStore


def _workflow_status(record: DecisionRecord) -> str:
    if record.status == "PENDING":
        return "DECISION_REQUIRED"
    if record.status == "RESOLVED" and record.application_status not in {"APPLIED", "LOCAL_APPLIED"}:
        return "RESOLVED_APPLICATION_REQUIRED"
    return record.status


def _answer_from_mapping(payload: dict[str, object]) -> DecisionAnswer:
    actual = set(payload)
    expected = set(CANONICAL_ANSWER_FIELDS)
    if actual != expected:
        missing = sorted(expected - actual)
        extra = sorted(actual - expected)
        details: list[str] = []
        if missing:
            details.append("missing: " + ", ".join(missing))
        if extra:
            details.append("unexpected: " + ", ".join(extra))
        raise ValueError("canonical v2 answer fields are invalid (" + "; ".join(details) + ")")
    if not isinstance(payload["classification"], str) or not str(payload["classification"]).strip():
        raise ValueError("classification must be a non-empty string")
    if not isinstance(payload["purpose"], str) or not str(payload["purpose"]).strip():
        raise ValueError("purpose must be a non-empty string")
    for field in ("shipped", "runtime_required", "executable"):
        if not isinstance(payload[field], bool):
            raise ValueError(f"{field} must be a boolean")
    return DecisionAnswer(
        classification=str(payload["classification"]).strip().upper(),
        purpose=str(payload["purpose"]).strip(),
        shipped=bool(payload["shipped"]),
        runtime_required=bool(payload["runtime_required"]),
        executable=bool(payload["executable"]),
    )


class LocalControlPlaneClient:
    """In-process decision control plane backed by repository-scoped local state.

    This client deliberately implements the same four operations used by the
    HTTP control-plane client while omitting GitHub App and webhook side effects.
    The DecisionStore remains responsible for compare-and-set resolution and
    active-gate rebinding semantics.
    """

    def __init__(self, repository_root: str | Path):
        self.store = DecisionStore(decision_store_path(repository_root))

    def gate(self, payload: dict[str, Any]) -> dict[str, object]:
        for key in ("id", "repository", "branch", "subject_revision", "component_id", "request"):
            if key not in payload:
                raise ValueError(f"gate payload missing {key}")
        record, _stale = self.store.gate(payload)
        return {
            "backend": "LOCAL",
            "status": _workflow_status(record),
            "decision": record.as_dict(),
        }

    def decision(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        if not decision_id:
            raise ValueError("decision_id is required")
        record = self.store.get(decision_id)
        if record is None:
            raise RuntimeError(f"Local decision does not exist: {decision_id}")
        return {
            "backend": "LOCAL",
            "status": _workflow_status(record),
            "decision": record.as_dict(),
        }

    def resolve(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        raw_answer = payload.get("answer")
        if not decision_id or not isinstance(raw_answer, dict):
            raise ValueError("decision_id and answer are required")
        answer = _answer_from_mapping(raw_answer)
        validation = validate_answer(answer)
        if not validation.valid:
            return {
                "backend": "LOCAL",
                "status": "CONFLICT",
                "validation": validation.as_dict(),
            }
        actor = str(payload.get("actor") or "coding-agent-session")
        try:
            record, accepted = self.store.resolve(decision_id, answer.as_dict(), "AGENT_CHAT", actor)
        except KeyError as exc:
            raise RuntimeError(f"Local decision does not exist: {decision_id}") from exc
        return {
            "backend": "LOCAL",
            "status": "RESOLVED" if accepted else "ALREADY_RESOLVED",
            "accepted": accepted,
            "decision": record.as_dict(),
            "validation": validation.as_dict(),
        }

    def application(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        status = str(payload.get("status", ""))
        if status not in {"LOCAL_APPLIED", "FAILED", "STALE"}:
            raise ValueError("unsupported agent application status")
        existing = self.store.get(decision_id)
        if existing is None:
            raise RuntimeError(f"Local decision does not exist: {decision_id}")
        if existing.status != "RESOLVED":
            raise ValueError("application state can be changed only for a resolved decision")
        record = self.store.mark_application(
            decision_id,
            status,
            str(payload.get("applied_revision") or "") or None,
        )
        return {
            "backend": "LOCAL",
            "status": status,
            "decision": record.as_dict(),
        }
