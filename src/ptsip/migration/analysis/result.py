from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from enum import StrEnum

from ptsip.source_compat.model import SourceFamily, SourceGenerationBinding
from ptsip.migration.analysis.findings.architecture import ArchitectureFinding
from ptsip.migration.analysis.issue import MigrationAnalysisIssue
from ptsip.migration.analysis.findings.lifecycle import LifecycleFinding
from ptsip.migration.analysis.findings.work_requirements import (
    AsynchronousWorkTarget,
    RemovalMigrationElement,
    RequiredWorkElement,
    SourceMigrationCompletion,
)

class SourceProjectionKind(StrEnum):
    COMPONENT = "COMPONENT"
    ASSOCIATED_ARTIFACT = "ASSOCIATED_ARTIFACT"
    BOUNDARY = "BOUNDARY"


class RepositoryResolutionKind(StrEnum):
    AMBIGUOUS_SOURCE_ELEMENT = "AMBIGUOUS_SOURCE_ELEMENT"


@dataclass(frozen=True)
class SourceCoverageProjection:
    declaration_id: str
    kind: SourceProjectionKind
    source_classification: str | None
    include: tuple[str, ...]
    exclude: tuple[str, ...]
    purpose: str
    source_pointer: str
    source_family: SourceFamily
    origin: str

    @classmethod
    def from_native(cls, row: dict[str, object]) -> SourceCoverageProjection:
        return cls(**{
            **row,
            "kind": SourceProjectionKind(row["kind"]),
            "include": tuple(row["include"]),
            "exclude": tuple(row["exclude"]),
            "source_family": SourceFamily(row["source_family"]),
        })

    def as_dict(self) -> dict[str, object]:
        return {
            "declaration_id": self.declaration_id,
            "kind": self.kind.value,
            "source_classification": self.source_classification,
            "include": list(self.include),
            "exclude": list(self.exclude),
            "purpose": self.purpose,
            "source_pointer": self.source_pointer,
            "source_family": self.source_family.value,
            "origin": self.origin,
        }


@dataclass(frozen=True)
class AmbiguousSourceElement:
    path: str
    coverages: tuple[SourceCoverageProjection, ...]
    selectors: tuple[str, ...]
    kind: RepositoryResolutionKind = RepositoryResolutionKind.AMBIGUOUS_SOURCE_ELEMENT

    @classmethod
    def from_native(cls, row: dict[str, object]) -> AmbiguousSourceElement:
        return cls(
            row["path"],
            tuple(SourceCoverageProjection.from_native(item) for item in row["coverages"]),
            tuple(row["selectors"]),
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "coverages": [item.as_dict() for item in self.coverages],
            "selectors": list(self.selectors),
        }

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

