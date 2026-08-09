from __future__ import annotations

from dataclasses import asdict, dataclass

from ..model import DecisionStatus


FIELD_ORDER = ("purpose", "shipped", "runtime_required", "lifecycle_owner")
REASON_BY_FIELD = {
    "purpose": "MISSING_PURPOSE",
    "shipped": "MISSING_PACKAGING_RESPONSIBILITY",
    "runtime_required": "MISSING_RUNTIME_ROLE",
    "lifecycle_owner": "MISSING_LIFECYCLE_OWNER",
}


@dataclass(frozen=True)
class ClarificationRequest:
    id: str
    component_id: str
    include: tuple[str, ...]
    anchors: tuple[str, ...]
    evidence_ids: tuple[str, ...]
    missing_fields: tuple[str, ...]
    reason_codes: tuple[str, ...]
    status: DecisionStatus = DecisionStatus.INCOMPLETE

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["status"] = self.status.value
        return payload
