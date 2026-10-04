from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass

from ptsip.repository.profile_convergence import DirectConvergenceMode
from ptsip.migration.analysis.result import MigrationAnalysis

@dataclass(frozen=True)
class DirectConvergenceAnalysisIssue:
    code: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return {"code": self.code, "message": self.message}

@dataclass(frozen=True)
class DirectConvergenceAnalysis:
    mode: DirectConvergenceMode
    source_path: str
    source_declared_version: str
    source_compatibility_contract: str
    target_contract: str
    target_path: str
    target_is_legacy_alias: bool
    identity_rewrite_required: bool
    semantic_analysis: MigrationAnalysis | None
    issues: tuple[DirectConvergenceAnalysisIssue, ...]

    @property
    def semantic_migration_required(self) -> bool:
        return self.mode is DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION

    @property
    def semantic_obligation_count(self) -> int:
        if self.semantic_analysis is None:
            return 0
        return (
            len(self.semantic_analysis.required)
            + len(self.semantic_analysis.removals)
            + len(self.semantic_analysis.async_targets)
        )

    @property
    def valid(self) -> bool:
        return not self.issues and (
            self.semantic_analysis is None or self.semantic_analysis.valid
        )

    def content_payload(self) -> dict[str, object]:
        return {
            "mode": self.mode.value,
            "source_path": self.source_path,
            "source_declared_version": self.source_declared_version,
            "source_compatibility_contract": self.source_compatibility_contract,
            "target_contract": self.target_contract,
            "target_path": self.target_path,
            "target_is_legacy_alias": self.target_is_legacy_alias,
            "identity_rewrite_required": self.identity_rewrite_required,
            "semantic_migration_required": self.semantic_migration_required,
            "semantic_analysis_digest": (
                self.semantic_analysis.deterministic_digest
                if self.semantic_analysis is not None
                else None
            ),
            "issues": [item.as_dict() for item in self.issues],
        }

    @property
    def deterministic_digest(self) -> str:
        raw = json.dumps(
            self.content_payload(),
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
        ).encode("utf-8")
        return hashlib.sha256(raw).hexdigest()

    def as_dict(self) -> dict[str, object]:
        payload = self.content_payload()
        payload["valid"] = self.valid
        payload["semantic_obligation_count"] = self.semantic_obligation_count
        payload["semantic_analysis"] = (
            self.semantic_analysis.as_dict()
            if self.semantic_analysis is not None
            else None
        )
        payload["deterministic_digest"] = self.deterministic_digest
        return payload

