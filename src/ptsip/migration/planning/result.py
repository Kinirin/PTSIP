from __future__ import annotations

from dataclasses import dataclass

from ptsip.migration.planning.convergence.final_state import FinalPointReference
from ptsip.migration.planning.convergence.reconciliation import DeletionGate, ReconciliationResult
from ptsip.migration.proposal.semantic_identity import semantic_digest

@dataclass(frozen=True)
class PlanningIssue:
    code: str
    message: str
    source_path: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "code": self.code,
            "message": self.message,
            "source_path": self.source_path,
        }

@dataclass(frozen=True)
class SourceConvergencePlan:
    source_path: str
    source_content_sha256: str
    analysis_digest: str
    required_total: int
    required_unresolved_before: int
    accepted_bundle_ids: tuple[str, ...]
    suggested_bundle_ids: tuple[str, ...]
    unresolved_bundle_ids: tuple[str, ...]
    unplanned_required_ids: tuple[str, ...]
    execution_delta_ids: tuple[str, ...]
    reconciliations: tuple[ReconciliationResult, ...]
    deletion_gate: DeletionGate
    next_source_path: str | None
    projected_final_state_digest: str

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "source_content_sha256": self.source_content_sha256,
            "analysis_digest": self.analysis_digest,
            "required_total": self.required_total,
            "required_unresolved_before": self.required_unresolved_before,
            "accepted_bundle_ids": list(self.accepted_bundle_ids),
            "suggested_bundle_ids": list(self.suggested_bundle_ids),
            "unresolved_bundle_ids": list(self.unresolved_bundle_ids),
            "unplanned_required_ids": list(self.unplanned_required_ids),
            "execution_delta_ids": list(self.execution_delta_ids),
            "reconciliations": [item.as_dict() for item in self.reconciliations],
            "deletion_gate": self.deletion_gate.value,
            "next_source_path": self.next_source_path,
            "projected_final_state_digest": self.projected_final_state_digest,
        }

@dataclass(frozen=True)
class ExecutionPreview:
    final_point: FinalPointReference
    ordered_sources: tuple[str, ...]
    source_steps: tuple[SourceConvergencePlan, ...]
    blocking_ids: tuple[str, ...]
    ready_for_wu07: bool
    promotion_gate: str = "WU07_GLOBAL_VALIDATION_REQUIRED"

    def as_dict(self) -> dict[str, object]:
        return {
            "final_point": self.final_point.as_dict(),
            "ordered_sources": list(self.ordered_sources),
            "source_steps": [item.as_dict() for item in self.source_steps],
            "blocking_ids": list(self.blocking_ids),
            "ready_for_wu07": self.ready_for_wu07,
            "promotion_gate": self.promotion_gate,
        }

@dataclass(frozen=True)
class FinalPointConvergencePlan:
    final_point: FinalPointReference
    source_steps: tuple[SourceConvergencePlan, ...]
    issues: tuple[PlanningIssue, ...]
    preview: ExecutionPreview
    projected_final_state_digest: str

    @property
    def deterministic_digest(self) -> str:
        return semantic_digest(
            {
                "final_point": self.final_point.as_dict(),
                "source_steps": [item.as_dict() for item in self.source_steps],
                "issues": [item.as_dict() for item in self.issues],
                "preview": self.preview.as_dict(),
                "projected_final_state_digest": self.projected_final_state_digest,
            }
        )

    def as_dict(self) -> dict[str, object]:
        payload = {
            "final_point": self.final_point.as_dict(),
            "source_steps": [item.as_dict() for item in self.source_steps],
            "issues": [item.as_dict() for item in self.issues],
            "preview": self.preview.as_dict(),
            "projected_final_state_digest": self.projected_final_state_digest,
        }
        payload["deterministic_digest"] = self.deterministic_digest
        return payload

