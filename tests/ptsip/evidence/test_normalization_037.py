from __future__ import annotations

import json
from importlib.resources import files
from types import SimpleNamespace

from jsonschema import Draft202012Validator

from ptsip.artifact_evidence import ArtifactEvidenceDocument, ArtifactEvidenceLoad
from ptsip.evidence.adapters import (
    agent_decision_facts,
    artifact_facts,
    candidate_facts,
    declared_boundary_facts,
    dependency_facts,
)
from ptsip.evidence.contract import (
    EvidenceChannel,
    EvidenceChannelStatus,
    EvidenceRecordStatus,
    SourceGenerationBinding,
)
from ptsip.evidence.normalization import EvidenceFact, build_evaluation_context, normalize_facts
from ptsip.inspection.dependencies import DependencyScan
from ptsip.model import (
    DependencyEdge,
    DependencyPhase,
    EdgeType,
    EvidenceNodeScope,
    EvidenceProvenance,
    ResolutionStatus,
)
from ptsip.review_evidence import AgentDecisionDocument, AgentDecisionLoad
from ptsip.validation.components import AMBIGUOUS, CandidateCoverage


def _context(version: str = "0.3.7-draft"):
    generation = SourceGenerationBinding(
        profile_path="ptsip.yaml" if version == "0.3.6-draft" else "ptsip_0.3.7.yaml",
        version=version,
        specification_revision="spec-revision",
        content_sha256="profile-sha-" + version,
    )
    return build_evaluation_context(
        repository_root="/repo",
        revision="deadbeef",
        status_fingerprint="status",
        tracked_content_fingerprint="content",
        source_generation=generation,
    )


def _fact(value=True, *, adapter="native", evidence_id="native:1") -> EvidenceFact:
    return EvidenceFact(
        subject="path:src/a.py",
        predicate="feature.enabled",
        value=value,
        provenance="OBSERVED",
        adapter=adapter,
        evidence_id=evidence_id,
        source_path="src/a.py",
    )


def _edge(
    *,
    evidence_id: str,
    adapter: str = "python",
    resolution: ResolutionStatus = ResolutionStatus.RESOLVED,
    target_scope: EvidenceNodeScope = EvidenceNodeScope.PROJECT_COMPONENT,
    resolved_path: str | None = "src/b/y.py",
) -> DependencyEdge:
    return DependencyEdge(
        evidence_id=evidence_id,
        source="src/a/x.py",
        target="ptsip.b",
        edge_type=EdgeType.IMPORTS,
        phase=DependencyPhase.RUNTIME,
        resolution=resolution,
        target_scope=target_scope,
        provenance=EvidenceProvenance.OBSERVED,
        resolved_path=resolved_path,
        adapter=adapter,
    )


def test_semantic_identity_is_generation_independent_but_evaluation_is_not() -> None:
    first = normalize_facts([_fact()], context=_context("0.3.7-draft"))
    second = normalize_facts([_fact()], context=_context("0.3.6-draft"))

    assert first.records[0].semantic_id == second.records[0].semantic_id
    assert first.context.evaluation_id != second.context.evaluation_id


def test_equivalent_assertions_merge_origins_without_losing_provenance() -> None:
    result = normalize_facts(
        [
            _fact(adapter="native", evidence_id="native:1"),
            _fact(adapter="external:scanner", evidence_id="external:1"),
        ],
        context=_context(),
    )

    record = result.records[0]
    assert record.status == EvidenceRecordStatus.CONSISTENT
    assert len(record.assertions) == 1
    assert {origin.adapter for origin in record.assertions[0].origins} == {"native", "external:scanner"}


def test_incompatible_values_remain_conflict_instead_of_last_writer_wins() -> None:
    result = normalize_facts(
        [_fact(True, evidence_id="a"), _fact(False, adapter="external", evidence_id="b")],
        context=_context(),
    )

    record = result.records[0]
    assert record.status == EvidenceRecordStatus.CONFLICT
    assert {assertion.value for assertion in record.assertions} == {True, False}


def test_explicit_false_is_distinct_from_no_match_and_not_analyzed() -> None:
    result = normalize_facts(
        [_fact(False)],
        context=_context(),
        channels=(
            EvidenceChannel("analyzed-empty", EvidenceChannelStatus.NO_MATCH, 0),
            EvidenceChannel("skipped", EvidenceChannelStatus.NOT_ANALYZED, 0),
        ),
    )

    assert result.records[0].assertions[0].value is False
    statuses = {item.id: item.status for item in result.channels}
    assert statuses["analyzed-empty"] == EvidenceChannelStatus.NO_MATCH
    assert statuses["skipped"] == EvidenceChannelStatus.NOT_ANALYZED


def test_normalization_is_deterministic_under_input_reordering() -> None:
    facts = [_fact(True, evidence_id="z"), _fact(True, adapter="other", evidence_id="a")]
    channels = (
        EvidenceChannel("z", EvidenceChannelStatus.PRODUCED, 1),
        EvidenceChannel("a", EvidenceChannelStatus.NO_MATCH, 0),
    )
    first = normalize_facts(facts, context=_context(), channels=channels)
    second = normalize_facts(reversed(facts), context=_context(), channels=reversed(channels))

    assert first.deterministic_digest == second.deterministic_digest
    assert first.as_dict() == second.as_dict()


def test_dependency_adapters_converge_equivalent_native_and_external_observations() -> None:
    scan = DependencyScan(
        edges=(
            _edge(evidence_id="native", adapter="python"),
            _edge(evidence_id="external", adapter="external:graph"),
        ),
        issues=(),
        adapters=("external:graph", "python"),
    )
    facts, channel, issues = dependency_facts(scan)
    result = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    assert len(result.records) == 1
    assert len(result.records[0].assertions) == 1
    assert len(result.records[0].assertions[0].origins) == 2


def test_dependency_resolution_disagreement_becomes_conflict() -> None:
    scan = DependencyScan(
        edges=(
            _edge(evidence_id="resolved"),
            _edge(
                evidence_id="unresolved",
                adapter="external:graph",
                resolution=ResolutionStatus.UNRESOLVED,
                target_scope=EvidenceNodeScope.UNRESOLVED_TARGET,
                resolved_path=None,
            ),
        ),
        issues=(),
        adapters=("external:graph", "python"),
    )
    facts, channel, issues = dependency_facts(scan)
    result = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    assert result.records[0].status == EvidenceRecordStatus.CONFLICT


def test_candidate_coverage_is_inferred_evidence_not_authority() -> None:
    observation = SimpleNamespace(
        kind="MANIFEST",
        evidence_id="manifest:pyproject.toml",
        provenance=EvidenceProvenance.OBSERVED,
        adapter="inventory",
        path="pyproject.toml",
        detail="manifest exists",
    )
    candidate = SimpleNamespace(
        id="candidate:abc",
        include=("pyproject.toml",),
        observations=(observation,),
        coverage=CandidateCoverage("COMPONENT", ("runtime",), ("COMPONENT",), ("pyproject.toml",)),
    )
    result = SimpleNamespace(candidates=(candidate,), issues=(), context=SimpleNamespace())
    facts, channel, issues = candidate_facts(result)
    normalized = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    coverage_record = next(item for item in normalized.records if item.predicate == "candidate.coverage")
    assert coverage_record.authority == "EVIDENCE_ONLY"
    assert coverage_record.assertions[0].origins[0].provenance == "INFERRED"


def test_artifact_classification_remains_an_evidence_claim() -> None:
    document = ArtifactEvidenceDocument(
        source_path="/tmp/artifact.json",
        source_sha256="a" * 64,
        binding_path="/tmp/artifact.json.binding.json",
        binding_valid=True,
        payload={
            "artifact_id": "wheel",
            "classification": "PRODUCT",
            "producer_component": "runtime",
            "artifact_type": "wheel",
            "shipping_scope": "public",
            "contents": {"paths": ["dist/x.whl"], "components": ["runtime"], "complete": True},
            "provenance": "DECLARED",
        },
    )
    facts, channel, issues = artifact_facts(ArtifactEvidenceLoad((document,), ()))
    normalized = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    classification = next(item for item in normalized.records if item.predicate == "artifact.classification")
    assert classification.assertions[0].value == "PRODUCT"
    assert classification.authority == "EVIDENCE_ONLY"


def test_agent_classification_is_review_evidence_only() -> None:
    document = AgentDecisionDocument(
        source_path="/tmp/agent.json",
        source_sha256="b" * 64,
        payload={
            "component_id": "runtime",
            "status": "RESOLVED",
            "classification": "PRODUCT",
            "confidence": 0.9,
            "rationale": "observed runtime imports",
            "evidence_ids": ["python:1"],
        },
    )
    facts, channel, issues = agent_decision_facts(AgentDecisionLoad((document,), ()))
    normalized = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    record = normalized.records[0]
    assert record.predicate == "agent.classification-review"
    assert record.authority == "EVIDENCE_ONLY"
    assert record.assertions[0].origins[0].provenance == "INFERRED"


def test_declared_boundary_uses_resolved_edge_and_shared_selector_coverage() -> None:
    components = [
        {"id": "a", "include": ["src/a/**"]},
        {"id": "b", "include": ["src/b/**"]},
    ]
    scan = DependencyScan((_edge(evidence_id="edge"),), (), ("python",))
    facts, channel, issues = declared_boundary_facts(scan, components)
    normalized = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    value = normalized.records[0].assertions[0].value
    assert value["crosses_declared_scope"] is True
    assert value["source_coverage"]["owner_ids"] == ["a"]
    assert value["target_coverage"]["owner_ids"] == ["b"]


def test_ambiguous_boundary_does_not_guess_owner() -> None:
    components = [
        {"id": "a1", "include": ["src/a/**"]},
        {"id": "a2", "include": ["src/a/**"]},
        {"id": "b", "include": ["src/b/**"]},
    ]
    scan = DependencyScan((_edge(evidence_id="edge"),), (), ("python",))
    facts, channel, issues = declared_boundary_facts(scan, components)
    normalized = normalize_facts(facts, context=_context(), channels=(channel,), issues=issues)

    value = normalized.records[0].assertions[0].value
    assert value["crosses_declared_scope"] is None
    assert value["source_coverage"]["status"] == AMBIGUOUS
    assert value["source_coverage"]["owner_ids"] == ["a1", "a2"]


def test_serialized_contract_validates_against_embedded_interchange_schema() -> None:
    normalized = normalize_facts([_fact()], context=_context())
    schema_path = files("ptsip").joinpath("specdata/ptsip-normalized-evidence.schema.json")
    schema = json.loads(schema_path.read_text(encoding="utf-8"))

    errors = list(Draft202012Validator(schema).iter_errors(normalized.as_dict()))
    assert errors == []
