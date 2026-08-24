from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from .contract import (
    EvidenceAssertion,
    EvidenceChannel,
    EvidenceEvaluationContext,
    EvidenceNormalizationIssue,
    EvidenceOrigin,
    EvidenceRecord,
    EvidenceRecordStatus,
    NormalizedEvidenceSet,
    SnapshotBinding,
    SourceGenerationBinding,
    assertion_id,
    canonical_json,
    semantic_evidence_id,
    stable_digest,
)


@dataclass(frozen=True)
class EvidenceFact:
    subject: str
    predicate: str
    value: object
    provenance: str
    adapter: str
    evidence_id: str
    qualifiers: dict[str, object] | None = None
    source_path: str | None = None
    line: int | None = None
    document_sha256: str | None = None
    producer_id: str | None = None
    basis_ids: tuple[str, ...] = ()
    detail: str | None = None


def build_evaluation_context(
    *,
    repository_root: str,
    revision: str | None,
    status_fingerprint: str,
    tracked_content_fingerprint: str,
    source_generation: SourceGenerationBinding | None = None,
    evaluation_id: str | None = None,
) -> EvidenceEvaluationContext:
    snapshot = SnapshotBinding(
        repository_root=repository_root,
        revision=revision,
        status_fingerprint=status_fingerprint,
        tracked_content_fingerprint=tracked_content_fingerprint,
    )
    if evaluation_id is None:
        evaluation_id = "evaluation:" + stable_digest(
            {
                "snapshot": snapshot.as_dict(),
                "source_generation": source_generation.as_dict() if source_generation else None,
            }
        )[:24]
    return EvidenceEvaluationContext(
        evaluation_id=evaluation_id,
        snapshot=snapshot,
        source_generation=source_generation,
    )


def _origin_key(origin: EvidenceOrigin) -> str:
    return canonical_json(origin.identity_payload())


def normalize_facts(
    facts: Iterable[EvidenceFact],
    *,
    context: EvidenceEvaluationContext,
    channels: Iterable[EvidenceChannel] = (),
    issues: Iterable[EvidenceNormalizationIssue] = (),
) -> NormalizedEvidenceSet:
    grouped: dict[
        str,
        dict[str, object],
    ] = {}

    for fact in facts:
        qualifiers = dict(fact.qualifiers or {})
        semantic_id = semantic_evidence_id(fact.subject, fact.predicate, qualifiers)
        record = grouped.setdefault(
            semantic_id,
            {
                "subject": fact.subject,
                "predicate": fact.predicate,
                "qualifiers": qualifiers,
                "assertions": {},
            },
        )
        assertion_key = canonical_json(fact.value)
        assertions = record["assertions"]
        assert isinstance(assertions, dict)
        assertion = assertions.setdefault(
            assertion_key,
            {
                "value": fact.value,
                "origins": {},
            },
        )
        origins = assertion["origins"]
        assert isinstance(origins, dict)
        origin = EvidenceOrigin(
            provenance=fact.provenance,
            adapter=fact.adapter,
            evidence_id=fact.evidence_id,
            source_path=fact.source_path,
            line=fact.line,
            document_sha256=fact.document_sha256,
            producer_id=fact.producer_id,
            basis_ids=tuple(sorted(set(fact.basis_ids))),
            detail=fact.detail,
        )
        origins[_origin_key(origin)] = origin

    records: list[EvidenceRecord] = []
    for semantic_id in sorted(grouped):
        raw = grouped[semantic_id]
        raw_assertions = raw["assertions"]
        assert isinstance(raw_assertions, dict)
        assertions: list[EvidenceAssertion] = []
        for value_key in sorted(raw_assertions):
            raw_assertion = raw_assertions[value_key]
            origins_map = raw_assertion["origins"]
            assert isinstance(origins_map, dict)
            value = raw_assertion["value"]
            assertions.append(
                EvidenceAssertion(
                    id=assertion_id(semantic_id, value),
                    value=value,
                    origins=tuple(origins_map[key] for key in sorted(origins_map)),
                )
            )
        records.append(
            EvidenceRecord(
                semantic_id=semantic_id,
                subject=str(raw["subject"]),
                predicate=str(raw["predicate"]),
                qualifiers=dict(raw["qualifiers"]),
                assertions=tuple(assertions),
                status=(
                    EvidenceRecordStatus.CONFLICT
                    if len(assertions) > 1
                    else EvidenceRecordStatus.CONSISTENT
                ),
            )
        )

    ordered_channels = tuple(sorted(channels, key=lambda item: item.id))
    ordered_issues = tuple(
        sorted(
            issues,
            key=lambda item: (
                item.channel or "",
                item.code,
                item.message,
                item.evidence_ids,
            ),
        )
    )
    return NormalizedEvidenceSet(
        context=context,
        records=tuple(records),
        channels=ordered_channels,
        issues=ordered_issues,
    )
