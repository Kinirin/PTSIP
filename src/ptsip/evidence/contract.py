from __future__ import annotations

import hashlib
import json
from dataclasses import asdict, dataclass
from enum import StrEnum
from typing import Any

EVIDENCE_FORMAT = "ptsip-normalized-evidence/v1"
AUTHORITY_EVIDENCE_ONLY = "EVIDENCE_ONLY"


class EvidenceChannelStatus(StrEnum):
    PRODUCED = "PRODUCED"
    NO_MATCH = "NO_MATCH"
    NOT_ANALYZED = "NOT_ANALYZED"
    FAILED = "FAILED"


class EvidenceRecordStatus(StrEnum):
    CONSISTENT = "CONSISTENT"
    CONFLICT = "CONFLICT"


def canonical_value(value: Any) -> Any:
    if isinstance(value, dict):
        return {str(key): canonical_value(value[key]) for key in sorted(value, key=lambda item: str(item))}
    if isinstance(value, (list, tuple)):
        return [canonical_value(item) for item in value]
    if isinstance(value, set):
        normalized = [canonical_value(item) for item in value]
        return sorted(normalized, key=canonical_json)
    if isinstance(value, StrEnum):
        return value.value
    return value


def canonical_json(value: Any) -> str:
    return json.dumps(
        canonical_value(value),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    )


def stable_digest(value: Any) -> str:
    return hashlib.sha256(canonical_json(value).encode("utf-8")).hexdigest()


def semantic_evidence_id(subject: str, predicate: str, qualifiers: dict[str, object] | None = None) -> str:
    digest = stable_digest(
        {
            "subject": subject,
            "predicate": predicate,
            "qualifiers": qualifiers or {},
        }
    )[:24]
    return f"evidence:{digest}"


def assertion_id(semantic_id: str, value: object) -> str:
    digest = stable_digest({"semantic_id": semantic_id, "value": value})[:24]
    return f"assertion:{digest}"


@dataclass(frozen=True)
class SourceGenerationBinding:
    profile_path: str
    version: str
    specification_revision: str
    content_sha256: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class SnapshotBinding:
    repository_root: str
    revision: str | None
    status_fingerprint: str
    tracked_content_fingerprint: str

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class EvidenceEvaluationContext:
    evaluation_id: str
    snapshot: SnapshotBinding
    source_generation: SourceGenerationBinding | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "evaluation_id": self.evaluation_id,
            "snapshot": self.snapshot.as_dict(),
            "source_generation": self.source_generation.as_dict() if self.source_generation else None,
        }


@dataclass(frozen=True)
class EvidenceOrigin:
    provenance: str
    adapter: str
    evidence_id: str
    source_path: str | None = None
    line: int | None = None
    document_sha256: str | None = None
    producer_id: str | None = None
    basis_ids: tuple[str, ...] = ()
    detail: str | None = None

    def identity_payload(self) -> dict[str, object]:
        return {
            "provenance": self.provenance,
            "adapter": self.adapter,
            "evidence_id": self.evidence_id,
            "source_path": self.source_path,
            "line": self.line,
            "document_sha256": self.document_sha256,
            "producer_id": self.producer_id,
            "basis_ids": sorted(set(self.basis_ids)),
            "detail": self.detail,
        }

    def as_dict(self) -> dict[str, object]:
        return self.identity_payload()


@dataclass(frozen=True)
class EvidenceAssertion:
    id: str
    value: object
    origins: tuple[EvidenceOrigin, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "value": canonical_value(self.value),
            "origins": [item.as_dict() for item in self.origins],
        }


@dataclass(frozen=True)
class EvidenceRecord:
    semantic_id: str
    subject: str
    predicate: str
    qualifiers: dict[str, object]
    assertions: tuple[EvidenceAssertion, ...]
    status: EvidenceRecordStatus
    authority: str = AUTHORITY_EVIDENCE_ONLY

    def as_dict(self) -> dict[str, object]:
        return {
            "semantic_id": self.semantic_id,
            "subject": self.subject,
            "predicate": self.predicate,
            "qualifiers": canonical_value(self.qualifiers),
            "assertions": [item.as_dict() for item in self.assertions],
            "status": self.status.value,
            "authority": self.authority,
        }


@dataclass(frozen=True)
class EvidenceChannel:
    id: str
    status: EvidenceChannelStatus
    evidence_count: int
    reason: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "status": self.status.value,
            "evidence_count": self.evidence_count,
            "reason": self.reason,
        }


@dataclass(frozen=True)
class EvidenceNormalizationIssue:
    code: str
    message: str
    channel: str | None = None
    evidence_ids: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "code": self.code,
            "message": self.message,
            "channel": self.channel,
            "evidence_ids": list(self.evidence_ids),
        }


@dataclass(frozen=True)
class NormalizedEvidenceSet:
    context: EvidenceEvaluationContext
    records: tuple[EvidenceRecord, ...]
    channels: tuple[EvidenceChannel, ...]
    issues: tuple[EvidenceNormalizationIssue, ...] = ()
    format: str = EVIDENCE_FORMAT

    @property
    def conflict_count(self) -> int:
        return sum(item.status == EvidenceRecordStatus.CONFLICT for item in self.records)

    def content_payload(self) -> dict[str, object]:
        return {
            "format": self.format,
            "context": self.context.as_dict(),
            "records": [item.as_dict() for item in self.records],
            "channels": [item.as_dict() for item in self.channels],
            "issues": [item.as_dict() for item in self.issues],
        }

    @property
    def deterministic_digest(self) -> str:
        return stable_digest(self.content_payload())

    def as_dict(self) -> dict[str, object]:
        payload = self.content_payload()
        payload["deterministic_digest"] = self.deterministic_digest
        payload["record_count"] = len(self.records)
        payload["conflict_count"] = self.conflict_count
        return payload
