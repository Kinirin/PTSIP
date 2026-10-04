from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass

from ptsip.source_compat.model import SourceGenerationBinding
from ptsip.migration.analysis.findings.architecture import ArchitectureFinding
from ptsip.migration.analysis.issue import MigrationAnalysisIssue
from ptsip.migration.analysis.findings.lifecycle import LifecycleFinding
from ptsip.migration.analysis.findings.work_requirements import (
    AsynchronousWorkTarget,
    RemovalMigrationElement,
    RequiredWorkElement,
    SourceMigrationCompletion,
)
from ptsip.migration.analysis.source.projection import AmbiguousSourceElement

@dataclass(frozen=True)
class MigrationAnalysis:
    source_generation: SourceGenerationBinding
    repository_head: str | None
    repository_status_fingerprint: str
    repository_content_fingerprint: str
    required: tuple[RequiredWorkElement, ...]
    removals: tuple[RemovalMigrationElement, ...]
    async_targets: tuple[AsynchronousWorkTarget, ...]
    ambiguous: tuple[AmbiguousSourceElement, ...]
    lifecycle_findings: tuple[LifecycleFinding, ...]
    architecture_findings: tuple[ArchitectureFinding, ...]
    issues: tuple[MigrationAnalysisIssue, ...]
    completion: SourceMigrationCompletion

    @property
    def valid(self) -> bool:
        return not self.issues and not self.ambiguous

    def content_payload(self) -> dict[str, object]:
        return {
            "source_generation": self.source_generation.as_dict(),
            "repository_head": self.repository_head,
            "repository_status_fingerprint": self.repository_status_fingerprint,
            "repository_content_fingerprint": self.repository_content_fingerprint,
            "required": [item.as_dict() for item in self.required],
            "removals": [item.as_dict() for item in self.removals],
            "async_targets": [item.as_dict() for item in self.async_targets],
            "ambiguous": [item.as_dict() for item in self.ambiguous],
            "lifecycle_findings": [item.as_dict() for item in self.lifecycle_findings],
            "architecture_findings": [item.as_dict() for item in self.architecture_findings],
            "issues": [item.as_dict() for item in self.issues],
            "completion": self.completion.as_dict(),
        }

    @property
    def deterministic_digest(self) -> str:
        encoded = json.dumps(
            self.content_payload(),
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
        ).encode("utf-8")
        return hashlib.sha256(encoded).hexdigest()

    def as_dict(self) -> dict[str, object]:
        payload = self.content_payload()
        payload["valid"] = self.valid
        payload["deterministic_digest"] = self.deterministic_digest
        return payload

