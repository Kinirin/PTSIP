from __future__ import annotations

from dataclasses import dataclass

from ..artifact_evidence import ArtifactEvidenceLoad
from ..inspection.candidate_evidence import CandidateDiscoveryResult
from ..inspection.dependencies import DependencyScan
from ..model import EvidenceProvenance, ResolutionStatus
from ..review_evidence import AgentDecisionLoad, ExternalEvidenceLoad
from ..validation.components import AMBIGUOUS, UNCOVERED, CandidateCoverage, resolve_candidate_coverage
from .contract import (
    EvidenceChannel,
    EvidenceChannelStatus,
    EvidenceEvaluationContext,
    EvidenceNormalizationIssue,
    SourceGenerationBinding,
    canonical_json,
)
from .normalization import EvidenceFact, build_evaluation_context, normalize_facts


@dataclass(frozen=True)
class _SelectorCandidate:
    include: tuple[str, ...]


def _provenance(value: object, default: str = EvidenceProvenance.OBSERVED.value) -> str:
    if isinstance(value, EvidenceProvenance):
        return value.value
    text = str(value)
    if text in {item.value for item in EvidenceProvenance}:
        return text
    return default


def _channel(
    channel_id: str,
    *,
    analyzed: bool,
    count: int,
    failed: bool = False,
    reason: str | None = None,
) -> EvidenceChannel:
    if not analyzed:
        status = EvidenceChannelStatus.NOT_ANALYZED
    elif failed:
        status = EvidenceChannelStatus.FAILED
    elif count:
        status = EvidenceChannelStatus.PRODUCED
    else:
        status = EvidenceChannelStatus.NO_MATCH
    return EvidenceChannel(channel_id, status, count, reason)


def context_from_candidate_discovery(result: CandidateDiscoveryResult) -> EvidenceEvaluationContext:
    if result.context is None:
        raise ValueError("Candidate discovery result has no source-generation evaluation context")
    source = result.context.source_generation
    source_binding = SourceGenerationBinding(
        profile_path=source.profile_path,
        version=source.version,
        specification_revision=source.specification_revision,
        content_sha256=source.content_sha256,
    )
    return build_evaluation_context(
        repository_root=result.context.repository_root,
        revision=result.context.repository_head,
        status_fingerprint=result.context.repository_status_fingerprint,
        tracked_content_fingerprint=result.context.repository_content_fingerprint,
        source_generation=source_binding,
        evaluation_id=result.context.evaluation_id,
    )


def candidate_facts(result: CandidateDiscoveryResult) -> tuple[list[EvidenceFact], EvidenceChannel, list[EvidenceNormalizationIssue]]:
    facts: list[EvidenceFact] = []
    issues = [
        EvidenceNormalizationIssue(
            code=item.code,
            message=item.message,
            channel="candidate-discovery",
            evidence_ids=(),
        )
        for item in result.issues
    ]
    for candidate in result.candidates:
        basis_ids = tuple(sorted({item.evidence_id for item in candidate.observations}))
        for observation in candidate.observations:
            facts.append(
                EvidenceFact(
                    subject=f"candidate:{candidate.id}",
                    predicate="candidate.observation",
                    qualifiers={
                        "kind": observation.kind,
                        "selectors": list(candidate.include),
                        "path": observation.path,
                    },
                    value=True,
                    provenance=_provenance(observation.provenance),
                    adapter=observation.adapter,
                    evidence_id=observation.evidence_id,
                    source_path=observation.path,
                    detail=observation.detail,
                )
            )
        facts.append(
            EvidenceFact(
                subject=f"candidate:{candidate.id}",
                predicate="candidate.coverage",
                qualifiers={"selectors": list(candidate.include)},
                value=candidate.coverage.as_dict(),
                provenance=EvidenceProvenance.INFERRED.value,
                adapter="selector-coverage",
                evidence_id=f"coverage:{candidate.id}",
                basis_ids=basis_ids,
                detail="Coverage is derived by the shared selector matcher from project declarations; it is not architecture authority.",
            )
        )
    return (
        facts,
        _channel(
            "candidate-discovery",
            analyzed=result.context is not None,
            count=len(facts),
            failed=bool(result.issues),
            reason="Candidate discovery produced diagnostics" if result.issues else None,
        ),
        issues,
    )


def _edge_fact(
    edge,
    *,
    source_path: str | None = None,
    document_sha256: str | None = None,
    producer_id: str | None = None,
) -> EvidenceFact:
    return EvidenceFact(
        subject=f"path:{edge.source}",
        predicate="dependency",
        qualifiers={
            "target": edge.target,
            "relationship_type": edge.edge_type.value,
            "phase": edge.phase.value,
        },
        value={
            "resolution": edge.resolution.value,
            "target_scope": edge.target_scope.value,
            "resolved_path": edge.resolved_path,
            "working_directory": edge.working_directory,
        },
        provenance=_provenance(edge.provenance),
        adapter=edge.adapter,
        evidence_id=edge.evidence_id,
        source_path=source_path or edge.source,
        line=edge.line,
        document_sha256=document_sha256,
        producer_id=producer_id,
        detail=edge.note,
    )


def dependency_facts(
    scan: DependencyScan,
    *,
    channel_id: str = "dependency-evidence",
) -> tuple[list[EvidenceFact], EvidenceChannel, list[EvidenceNormalizationIssue]]:
    facts = [_edge_fact(edge) for edge in scan.edges]
    issues = [
        EvidenceNormalizationIssue(
            code="DEPENDENCY_INPUT_ISSUE",
            message=item.message,
            channel=channel_id,
            evidence_ids=(),
        )
        for item in scan.issues
    ]
    return (
        facts,
        _channel(
            channel_id,
            analyzed=True,
            count=len(facts),
            failed=bool(scan.issues),
            reason="Dependency scan was incomplete" if scan.issues else None,
        ),
        issues,
    )


def _external_origin(edge) -> tuple[str | None, str | None, str | None]:
    note = edge.note
    if not isinstance(note, str) or not note.startswith("Imported from "):
        producer = edge.adapter.removeprefix("external:") if edge.adapter.startswith("external:") else None
        return None, None, producer
    imported, separator, remainder = note.partition("; producer=")
    if not separator:
        return None, None, None
    source_path = imported.removeprefix("Imported from ")
    producer_text, separator, digest_text = remainder.partition("; document_sha256=")
    producer_id = producer_text.split("@", 1)[0] or None
    document_sha256 = digest_text if separator and digest_text else None
    return source_path or None, document_sha256, producer_id


def external_dependency_facts(
    loaded: ExternalEvidenceLoad,
) -> tuple[list[EvidenceFact], EvidenceChannel, list[EvidenceNormalizationIssue]]:
    facts: list[EvidenceFact] = []
    for edge in loaded.edges:
        source_path, digest, producer_id = _external_origin(edge)
        facts.append(
            _edge_fact(
                edge,
                source_path=source_path,
                document_sha256=digest,
                producer_id=producer_id,
            )
        )
    issues = [
        EvidenceNormalizationIssue(
            code="EXTERNAL_EVIDENCE_INPUT_ISSUE",
            message=item.message,
            channel="external-dependency-evidence",
        )
        for item in loaded.issues
    ]
    return (
        facts,
        _channel(
            "external-dependency-evidence",
            analyzed=True,
            count=len(facts),
            failed=bool(loaded.issues),
            reason="External evidence input was invalid or stale" if loaded.issues else None,
        ),
        issues,
    )


def _artifact_contents(value: object) -> object:
    if not isinstance(value, dict):
        return value
    normalized = dict(value)
    for key in ("paths", "components"):
        items = normalized.get(key)
        if isinstance(items, list):
            normalized[key] = sorted({str(item) for item in items})
    return normalized


def _artifact_derivation(value: object) -> object:
    if not isinstance(value, list):
        return value
    normalized = [dict(item) if isinstance(item, dict) else item for item in value]
    return sorted(normalized, key=canonical_json)


def artifact_facts(
    loaded: ArtifactEvidenceLoad,
) -> tuple[list[EvidenceFact], EvidenceChannel, list[EvidenceNormalizationIssue]]:
    facts: list[EvidenceFact] = []
    for document in loaded.documents:
        payload = document.payload
        artifact_id = str(payload.get("artifact_id", "<unknown>"))
        provenance = _provenance(payload.get("provenance"))
        common = {
            "provenance": provenance,
            "adapter": "artifact-evidence",
            "source_path": document.source_path,
            "document_sha256": document.source_sha256,
            "producer_id": str(payload.get("producer_component")) if payload.get("producer_component") is not None else None,
        }
        for predicate, value in (
            ("artifact.classification", payload.get("classification")),
            ("artifact.producer_component", payload.get("producer_component")),
            ("artifact.type", payload.get("artifact_type")),
            ("artifact.shipping_scope", payload.get("shipping_scope")),
            ("artifact.contents", _artifact_contents(payload.get("contents"))),
            ("artifact.derivation", _artifact_derivation(payload.get("derivation", []))),
        ):
            facts.append(
                EvidenceFact(
                    subject=f"artifact:{artifact_id}",
                    predicate=predicate,
                    value=value,
                    evidence_id=f"artifact:{document.source_sha256[:16]}:{predicate}",
                    detail="Artifact document claim; normalized as evidence only and never as project-owned architecture authority.",
                    **common,
                )
            )
        facts.append(
            EvidenceFact(
                subject=f"artifact:{artifact_id}",
                predicate="artifact.binding_valid",
                value=bool(document.binding_valid),
                provenance=EvidenceProvenance.OBSERVED.value,
                adapter="artifact-binding",
                evidence_id=f"artifact-binding:{document.source_sha256[:16]}",
                source_path=document.binding_path or document.source_path,
                document_sha256=document.source_sha256,
            )
        )
    issues = [
        EvidenceNormalizationIssue(
            code="ARTIFACT_EVIDENCE_INPUT_ISSUE",
            message=item.message,
            channel="artifact-evidence",
        )
        for item in loaded.issues
    ]
    return (
        facts,
        _channel(
            "artifact-evidence",
            analyzed=True,
            count=len(facts),
            failed=bool(loaded.issues),
            reason="Artifact evidence input was invalid or stale" if loaded.issues else None,
        ),
        issues,
    )


def agent_decision_facts(
    loaded: AgentDecisionLoad,
) -> tuple[list[EvidenceFact], EvidenceChannel, list[EvidenceNormalizationIssue]]:
    facts: list[EvidenceFact] = []
    for document in loaded.documents:
        payload = document.payload
        component_id = str(payload.get("component_id", "<unknown>"))
        review_detail = {
            "confidence": payload.get("confidence"),
            "rationale": payload.get("rationale"),
            "counter_evidence": sorted(
                str(item) for item in payload.get("counter_evidence", []) if isinstance(item, str)
            ),
        }
        facts.append(
            EvidenceFact(
                subject=f"component:{component_id}",
                predicate="agent.classification-review",
                value={
                    "status": payload.get("status"),
                    "classification": payload.get("classification"),
                },
                provenance=EvidenceProvenance.INFERRED.value,
                adapter="agent-decision",
                evidence_id=f"agent-decision:{document.source_sha256[:16]}:{component_id}",
                source_path=document.source_path,
                document_sha256=document.source_sha256,
                basis_ids=tuple(str(item) for item in payload.get("evidence_ids", []) if isinstance(item, str)),
                detail=(
                    "Agent classification is review evidence only and does not override the Project Profile; "
                    f"review_metadata={canonical_json(review_detail)}"
                ),
            )
        )
    issues = [
        EvidenceNormalizationIssue(
            code="AGENT_DECISION_INPUT_ISSUE",
            message=item.message,
            channel="agent-decision",
        )
        for item in loaded.issues
    ]
    return (
        facts,
        _channel(
            "agent-decision",
            analyzed=True,
            count=len(facts),
            failed=bool(loaded.issues),
            reason="Agent decision input was invalid" if loaded.issues else None,
        ),
        issues,
    )


def _coverage(path: str, components: list[dict[str, object]], associated_artifacts: list[dict[str, object]]) -> CandidateCoverage:
    return resolve_candidate_coverage(_SelectorCandidate((path,)), components, associated_artifacts)


def declared_boundary_facts(
    scan: DependencyScan,
    components: list[dict[str, object]],
    associated_artifacts: list[dict[str, object]] | None = None,
) -> tuple[list[EvidenceFact], EvidenceChannel, list[EvidenceNormalizationIssue]]:
    artifacts = associated_artifacts or []
    facts: list[EvidenceFact] = []
    for edge in scan.edges:
        if edge.resolution != ResolutionStatus.RESOLVED or not edge.resolved_path:
            continue
        source_coverage = _coverage(edge.source, components, artifacts)
        target_coverage = _coverage(edge.resolved_path, components, artifacts)
        source_unique = source_coverage.status not in {AMBIGUOUS, UNCOVERED} and len(source_coverage.owner_ids) == 1
        target_unique = target_coverage.status not in {AMBIGUOUS, UNCOVERED} and len(target_coverage.owner_ids) == 1
        crosses: bool | None
        if source_unique and target_unique:
            crosses = source_coverage.owner_ids[0] != target_coverage.owner_ids[0] or source_coverage.owner_kinds != target_coverage.owner_kinds
        else:
            crosses = None
        facts.append(
            EvidenceFact(
                subject=f"path:{edge.source}",
                predicate="declared-scope-boundary",
                qualifiers={
                    "target": edge.target,
                    "resolved_path": edge.resolved_path,
                    "relationship_type": edge.edge_type.value,
                },
                value={
                    "crosses_declared_scope": crosses,
                    "source_coverage": source_coverage.as_dict(),
                    "target_coverage": target_coverage.as_dict(),
                },
                provenance=EvidenceProvenance.INFERRED.value,
                adapter="selector-coverage-boundary",
                evidence_id=f"boundary:{edge.evidence_id}",
                source_path=edge.source,
                line=edge.line,
                basis_ids=(edge.evidence_id,),
                detail=(
                    "Boundary evidence derives only from a resolved dependency and shared selector coverage. "
                    "Ambiguous or uncovered ownership remains unresolved; no lifecycle classification is inferred."
                ),
            )
        )
    return facts, _channel("declared-boundary", analyzed=True, count=len(facts)), []


def normalize_ptsip_evidence(
    candidate_result: CandidateDiscoveryResult,
    *,
    dependencies: DependencyScan | None = None,
    external_evidence: ExternalEvidenceLoad | None = None,
    artifact_evidence: ArtifactEvidenceLoad | None = None,
    agent_decisions: AgentDecisionLoad | None = None,
    components: list[dict[str, object]] | None = None,
    associated_artifacts: list[dict[str, object]] | None = None,
):
    context = context_from_candidate_discovery(candidate_result)
    facts: list[EvidenceFact] = []
    channels: list[EvidenceChannel] = []
    issues: list[EvidenceNormalizationIssue] = []

    candidate_items, candidate_channel, candidate_issues = candidate_facts(candidate_result)
    facts.extend(candidate_items)
    channels.append(candidate_channel)
    issues.extend(candidate_issues)

    if dependencies is None:
        channels.append(_channel("dependency-evidence", analyzed=False, count=0))
        channels.append(_channel("declared-boundary", analyzed=False, count=0))
    else:
        items, channel, found_issues = dependency_facts(dependencies)
        facts.extend(items)
        channels.append(channel)
        issues.extend(found_issues)
        if components is None:
            channels.append(_channel("declared-boundary", analyzed=False, count=0, reason="Project declarations were not supplied"))
        else:
            boundary_items, boundary_channel, boundary_issues = declared_boundary_facts(
                dependencies,
                components,
                associated_artifacts,
            )
            facts.extend(boundary_items)
            channels.append(boundary_channel)
            issues.extend(boundary_issues)

    for source, channel_id, adapter in (
        (external_evidence, "external-dependency-evidence", external_dependency_facts),
        (artifact_evidence, "artifact-evidence", artifact_facts),
        (agent_decisions, "agent-decision", agent_decision_facts),
    ):
        if source is None:
            channels.append(_channel(channel_id, analyzed=False, count=0))
            continue
        items, channel, found_issues = adapter(source)
        facts.extend(items)
        channels.append(channel)
        issues.extend(found_issues)

    return normalize_facts(facts, context=context, channels=channels, issues=issues)
