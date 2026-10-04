from __future__ import annotations

from dataclasses import dataclass

@dataclass(frozen=True)
class MigrationAnalysisIssue:
    code: str
    message: str
    subject_id: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {"code": self.code, "message": self.message, "subject_id": self.subject_id}
