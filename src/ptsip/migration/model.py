from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from enum import StrEnum

from ..evidence.contract import EvidenceChannelStatus, NormalizedEvidenceSet
from ..source_compat.model import SourceFamily, SourceGenerationBinding


class SourceProjectionKind(StrEnum):
    COMPONENT = "COMPONENT"
    ASSOCIATED_ARTIFACT = "ASSOCIATED_ARTIFACT"
    BOUNDARY = "BOUNDARY"


class RepositoryResolutionKind(StrEnum):
    EXISTING_SOURCE_ELEMENT = "EXISTING_SOURCE_ELEMENT"
    REMOVED_SOURCE_ELEMENT = "REMOVED_SOURCE_ELEMENT"
    UNCOVERED_REPOSITORY_ELEMENT = "UNCOVERED_REPOSITORY_ELEMENT"
    AMBIGUOUS_SOURCE_ELEMENT = "AMBIGUOUS_SOURCE_ELEMENT"


class ObligationCategory(StrEnum):
    REQUIRED = "REQUIRED"
    REMOVAL = "REMOVAL"
    ASYNC = "ASYNC"


class LifecycleFindingKind(StrEnum):
    EXACT_SEMANTIC_PRESERVATION = "EXACT_SEMANTIC_PRESERVATION"
    HISTORICAL_TOOLCHAIN_AMBIGUITY = "HISTORICAL_TOOLCHAIN_AMBIGUITY"
    POSSIBLE_LIFECYCLE_SEPARATION = "POSSIBLE_LIFECYCLE_SEPARATION"
    TARGET_REVIEW_REQUIRED = "TARGET_REVIEW_REQUIRED"


class ArchitectureFindingKind(StrEnum):
    STALE_SOURCE_DECLARATION = "STALE_SOURCE_DECLARATION"
    NEW_REPOSITORY_CANDIDATE = "NEW_REPOSITORY_CANDIDATE"
    AMBIGUOUS_SOURCE_COVERAGE = "AMBIGUOUS_SOURCE_COVERAGE"
    MISSING_RELATIONSHIP = "MISSING_RELATIONSHIP"
    MISSING_ASSOCIATED_ARTIFACT = "MISSING_ASSOCIATED_ARTIFACT"
    EVIDENCE_CONFLICT = "EVIDENCE_CONFLICT"
    EVIDENCE_INCOMPLETE = "EVIDENCE_INCOMPLETE"


class TargetCompatibility(StrEnum):
    NOT_EVALUATED = "NOT_EVALUATED"
    ALREADY_SATISFIED = "ALREADY_SATISFIED"
    COMPATIBLE_TARGET_STATE = "COMPATIBLE_TARGET_STATE"
    CONFLICTING_TARGET_STATE = "CONFLICTING_TARGET_STATE"
    TARGET_REVIEW_REQUIRED = "TARGET_REVIEW_REQUIRED"


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
class ExistingSourceElement:
    path: str
    coverage: SourceCoverageProjection
    selector: str
    kind: RepositoryResolutionKind = RepositoryResolutionKind.EXISTING_SOURCE_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "coverage": self.coverage.as_dict(),
            "selector": self.selector,
        }


@dataclass(frozen=True)
class RemovedSourceElement:
    element_id: str
    coverage: SourceCoverageProjection
    selector: str
    kind: RepositoryResolutionKind = RepositoryResolutionKind.REMOVED_SOURCE_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "element_id": self.element_id,
            "coverage": self.coverage.as_dict(),
            "selector": self.selector,
        }


@dataclass(frozen=True)
class UncoveredRepositoryElement:
    path: str
    kind: RepositoryResolutionKind = RepositoryResolutionKind.UNCOVERED_REPOSITORY_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {"kind": self.kind.value, "path": self.path}


@dataclass(frozen=True)
class AmbiguousSourceElement:
    path: str
    coverages: tuple[SourceCoverageProjection, ...]
    selectors: tuple[str, ...]
    kind: RepositoryResolutionKind = RepositoryResolutionKind.AMBIGUOUS_SOURCE_ELEMENT

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "coverages": [item.as_dict() for item in self.coverages],
            "selectors": list(self.selectors),
        }


@dataclass(frozen=True)
class EvidenceCorrelation:
    semantic_ids: tuple[str, ...]
    conflict_ids: tuple[str, ...]
    incomplete_channels: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "semantic_ids": list(self.semantic_ids),
            "conflict_ids": list(self.conflict_ids),
            "incomplete_channels": list(self.incomplete_channels),
        }


@dataclass(frozen=True)
class LifecycleFinding:
    subject_id: str
    kind: LifecycleFindingKind
    source_classification: str | None
    target_classification: str | None
    rationale: str

    def as_dict(self) -> dict[str, object]:
        return {
            "subject_id": self.subject_id,
            "kind": self.kind.value,
            "source_classification": self.source_classification,
            "target_classification": self.target_classification,
            "rationale": self.rationale,
        }


@dataclass(frozen=True)
class ArchitectureFinding:
    subject_id: str
    kind: ArchitectureFindingKind
    rationale: str
    evidence_ids: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "subject_id": self.subject_id,
            "kind": self.kind.value,
            "rationale": self.rationale,
            "evidence_ids": list(self.evidence_ids),
        }


@dataclass(frozen=True)
class RequiredWorkElement:
    id: str
    path: str
    source_declaration_id: str
    source_classification: str | None
    selector: str
    evidence: EvidenceCorrelation
    target_status: TargetCompatibility
    resolved: bool

    @property
    def category(self) -> ObligationCategory:
        return ObligationCategory.REQUIRED

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "category": self.category.value,
            "path": self.path,
            "source_declaration_id": self.source_declaration_id,
            "source_classification": self.source_classification,
            "selector": self.selector,
            "evidence": self.evidence.as_dict(),
            "target_status": self.target_status.value,
            "resolved": self.resolved,
        }


@dataclass(frozen=True)
class RemovalMigrationElement:
    id: str
    source_declaration_id: str
    source_classification: str | None
    selector: str
    rationale: str

    @property
    def category(self) -> ObligationCategory:
        return ObligationCategory.REMOVAL

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "category": self.category.value,
            "source_declaration_id": self.source_declaration_id,
            "source_classification": self.source_classification,
            "selector": self.selector,
            "rationale": self.rationale,
        }


@dataclass(frozen=True)
class AsynchronousWorkTarget:
    id: str
    path: str
    evidence: EvidenceCorrelation

    @property
    def category(self) -> ObligationCategory:
        return ObligationCategory.ASYNC

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "category": self.category.value,
            "path": self.path,
            "evidence": self.evidence.as_dict(),
        }


@dataclass(frozen=True)
class TargetSemantics:
    draft_version: str
    classifications: tuple[str, ...]
    relationship_types: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "draft_version": self.draft_version,
            "classifications": list(self.classifications),
            "relationship_types": list(self.relationship_types),
        }


@dataclass(frozen=True)
class TargetComponent:
    id: str
    classification: str
    include: tuple[str, ...]
    exclude: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "classification": self.classification,
            "include": list(self.include),
            "exclude": list(self.exclude),
        }


@dataclass(frozen=True)
class TargetAssociatedArtifact:
    id: str
    anchor: str
    include: tuple[str, ...]
    exclude: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "anchor": self.anchor,
            "include": list(self.include),
            "exclude": list(self.exclude),
        }


@dataclass(frozen=True)
class TargetRelationship:
    id: str
    source: str
    target: str
    relationship_type: str

    def as_dict(self) -> dict[str, str]:
        return {
            "id": self.id,
            "from": self.source,
            "to": self.target,
            "type": self.relationship_type,
        }


@dataclass(frozen=True)
class TargetArchitectureState:
    draft_version: str
    specification_revision: str
    components: tuple[TargetComponent, ...]
    associated_artifacts: tuple[TargetAssociatedArtifact, ...] = ()
    relationships: tuple[TargetRelationship, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "draft_version": self.draft_version,
            "specification_revision": self.specification_revision,
            "components": [item.as_dict() for item in self.components],
            "associated_artifacts": [item.as_dict() for item in self.associated_artifacts],
            "relationships": [item.as_dict() for item in self.relationships],
        }


@dataclass(frozen=True)
class SourceMigrationCompletion:
    required_total: int
    required_resolved: int
    required_unresolved: int
    removal_count: int
    async_count: int

    @property
    def complete(self) -> bool:
        return self.required_unresolved == 0

    def as_dict(self) -> dict[str, object]:
        return {
            "required_total": self.required_total,
            "required_resolved": self.required_resolved,
            "required_unresolved": self.required_unresolved,
            "removal_count": self.removal_count,
            "async_count": self.async_count,
            "complete": self.complete,
        }


@dataclass(frozen=True)
class MigrationAnalysisIssue:
    code: str
    message: str
    subject_id: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {"code": self.code, "message": self.message, "subject_id": self.subject_id}


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
