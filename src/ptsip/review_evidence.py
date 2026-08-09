from __future__ import annotations

import hashlib
import json
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from .inspection.dependencies import DependencyScan, DependencyScanIssue
from .model import (
    DependencyEdge,
    DependencyPhase,
    EdgeType,
    EvidenceNodeScope,
    EvidenceProvenance,
    ResolutionStatus,
)
from .repository.discover import RepositoryInfo
from .repository.snapshot import repository_files


@dataclass(frozen=True)
class EvidenceInputIssue:
    source_path: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class AgentDecisionDocument:
    source_path: str
    source_sha256: str
    payload: dict[str, object]

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "source_sha256": self.source_sha256,
            "decision": self.payload,
        }


@dataclass(frozen=True)
class AgentDecisionLoad:
    documents: tuple[AgentDecisionDocument, ...]
    issues: tuple[EvidenceInputIssue, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "document_count": len(self.documents),
            "issue_count": len(self.issues),
            "documents": [item.as_dict() for item in self.documents],
            "issues": [item.as_dict() for item in self.issues],
        }


@dataclass(frozen=True)
class ExternalEvidenceDocument:
    source_path: str
    source_sha256: str
    producer_id: str
    producer_version: str
    subject_repository: str
    subject_revision: str
    evidence_count: int

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ExternalEvidenceLoad:
    documents: tuple[ExternalEvidenceDocument, ...]
    edges: tuple[DependencyEdge, ...]
    issues: tuple[EvidenceInputIssue, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "document_count": len(self.documents),
            "edge_count": len(self.edges),
            "issue_count": len(self.issues),
            "documents": [item.as_dict() for item in self.documents],
            "issues": [item.as_dict() for item in self.issues],
        }


def _read_structured(path: str | Path) -> tuple[dict[str, object] | None, str | None, str]:
    source = Path(path).expanduser().resolve()
    try:
        raw = source.read_bytes()
    except OSError as exc:
        return None, str(exc), ""
    digest = hashlib.sha256(raw).hexdigest()
    try:
        payload = yaml.safe_load(raw.decode("utf-8-sig"))
    except (UnicodeError, yaml.YAMLError) as exc:
        return None, str(exc), digest
    if not isinstance(payload, dict):
        return None, "document root is not a mapping", digest
    return payload, None, digest


def _agent_schema() -> dict[str, object]:
    path = Path(__file__).resolve().parent / "specdata" / "ptsip-agent-classification.schema.json"
    return json.loads(path.read_text(encoding="utf-8"))


def load_agent_decisions(paths: list[str | Path] | tuple[str | Path, ...] | None) -> AgentDecisionLoad:
    documents: list[AgentDecisionDocument] = []
    issues: list[EvidenceInputIssue] = []
    validator = Draft202012Validator(_agent_schema())
    for raw_path in paths or ():
        source = str(Path(raw_path).expanduser().resolve())
        payload, error, digest = _read_structured(raw_path)
        if error is not None or payload is None:
            issues.append(EvidenceInputIssue(source, error or "Unable to parse agent decision"))
            continue
        validation_errors = sorted(validator.iter_errors(payload), key=lambda item: list(item.absolute_path))
        if validation_errors:
            for item in validation_errors:
                location = ".".join(str(part) for part in item.absolute_path) or "<root>"
                issues.append(EvidenceInputIssue(source, f"{location}: {item.message}"))
            continue
        documents.append(AgentDecisionDocument(source, digest, payload))
    return AgentDecisionLoad(tuple(documents), tuple(issues))


def evaluate_agent_decisions(
    loaded: AgentDecisionLoad,
    components: list[dict[str, object]],
) -> tuple[dict[str, object], list[dict[str, object]]]:
    declarations = {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }
    gaps: list[dict[str, object]] = []
    observations: list[dict[str, object]] = []
    resolved_by_component: dict[str, set[str]] = {}

    for issue in loaded.issues:
        gaps.append(
            {
                "id": "agent-decision:invalid:" + hashlib.sha256((issue.source_path + issue.message).encode()).hexdigest()[:12],
                "blocking": True,
                "rule_ids": ["PTSIP-EVD-003"],
                "evidence_ids": [f"agent-input:{issue.source_path}"],
                "message": f"Explicit agent-decision input is invalid: {issue.message}",
            }
        )

    for document in loaded.documents:
        payload = document.payload
        component_id = str(payload["component_id"])
        status = str(payload["status"])
        classification = payload.get("classification")
        declaration = declarations.get(component_id)
        evidence_id = f"agent-decision:{document.source_sha256[:16]}:{component_id}"
        if declaration is None:
            gaps.append(
                {
                    "id": f"agent-decision:unknown-component:{component_id}",
                    "blocking": True,
                    "rule_ids": ["PTSIP-CLS-001", "PTSIP-EVD-003"],
                    "evidence_ids": [evidence_id],
                    "message": f"Agent decision references component {component_id!r}, which is not declared by the Project Profile.",
                }
            )
            continue
        if status == "RESOLVED" and isinstance(classification, str):
            resolved_by_component.setdefault(component_id, set()).add(classification)
            if classification != declaration:
                gaps.append(
                    {
                        "id": f"agent-decision:declaration-conflict:{component_id}:{classification}",
                        "blocking": True,
                        "rule_ids": ["PTSIP-EVD-002", "PTSIP-EVD-003"],
                        "evidence_ids": [evidence_id, *[str(item) for item in payload.get("evidence_ids", [])]],
                        "message": (
                            f"Agent decision classifies {component_id!r} as {classification}, while the Project Profile declares {declaration}. "
                            "The agent decision is review evidence and does not override the declaration."
                        ),
                    }
                )
            else:
                observations.append(
                    {
                        "component_id": component_id,
                        "status": status,
                        "classification": classification,
                        "effect": "CORROBORATES_DECLARATION",
                        "source_sha256": document.source_sha256,
                    }
                )
        else:
            observations.append(
                {
                    "component_id": component_id,
                    "status": status,
                    "classification": None,
                    "effect": "REVIEW_EVIDENCE_ONLY",
                    "source_sha256": document.source_sha256,
                }
            )

    for component_id, classifications in sorted(resolved_by_component.items()):
        if len(classifications) > 1:
            gaps.append(
                {
                    "id": f"agent-decision:internal-conflict:{component_id}",
                    "blocking": True,
                    "rule_ids": ["PTSIP-EVD-002", "PTSIP-EVD-003"],
                    "evidence_ids": [f"agent-decision-conflict:{component_id}"],
                    "message": f"Resolved agent decisions disagree on the classification of {component_id!r}: {', '.join(sorted(classifications))}.",
                }
            )

    report = loaded.as_dict()
    report["observations"] = observations
    report["affects_declared_classification"] = False
    return report, gaps


def _required_mapping(payload: dict[str, object], key: str) -> dict[str, object] | None:
    value = payload.get(key)
    return value if isinstance(value, dict) else None


def _enum_value(enum_type, value: object, label: str) -> tuple[object | None, str | None]:
    if not isinstance(value, str):
        return None, f"{label} must be a string"
    try:
        return enum_type(value), None
    except ValueError:
        return None, f"Unsupported {label} {value!r}"


def load_external_evidence(
    paths: list[str | Path] | tuple[str | Path, ...] | None,
    repository: RepositoryInfo,
) -> ExternalEvidenceLoad:
    documents: list[ExternalEvidenceDocument] = []
    edges: list[DependencyEdge] = []
    issues: list[EvidenceInputIssue] = []
    root = Path(repository.root).resolve()
    _mode, tracked_paths, tracked_errors = repository_files(root)
    tracked = set(tracked_paths)
    for error in tracked_errors:
        issues.append(EvidenceInputIssue(repository.root, f"Unable to establish tracked source identity: {error}"))
    expected_repositories = {repository.root}
    if repository.remote and repository.remote.repository:
        expected_repositories.add(repository.remote.repository)

    for raw_path in paths or ():
        source_path = str(Path(raw_path).expanduser().resolve())
        payload, error, digest = _read_structured(raw_path)
        if error is not None or payload is None:
            issues.append(EvidenceInputIssue(source_path, error or "Unable to parse external evidence"))
            continue
        if payload.get("format") != "ptsip-external-evidence/v1":
            issues.append(EvidenceInputIssue(source_path, "format must be 'ptsip-external-evidence/v1'"))
            continue
        producer = _required_mapping(payload, "producer")
        subject = _required_mapping(payload, "subject")
        evidence = payload.get("evidence")
        if producer is None or subject is None or not isinstance(evidence, list):
            issues.append(EvidenceInputIssue(source_path, "producer, subject, and evidence are required"))
            continue
        producer_id = producer.get("id")
        producer_version = producer.get("version")
        subject_repository = subject.get("repository")
        subject_revision = subject.get("revision")
        if not all(isinstance(item, str) and item.strip() for item in (producer_id, producer_version, subject_repository, subject_revision)):
            issues.append(EvidenceInputIssue(source_path, "producer.id/version and subject.repository/revision must be non-empty strings"))
            continue
        if subject_repository not in expected_repositories:
            issues.append(EvidenceInputIssue(source_path, f"subject repository {subject_repository!r} does not match the evaluated Consumer Repository"))
            continue
        if not repository.commit or subject_revision != repository.commit:
            issues.append(
                EvidenceInputIssue(
                    source_path,
                    f"subject revision {subject_revision!r} does not match evaluated repository revision {repository.commit!r}",
                )
            )
            continue

        accepted = 0
        for index, item in enumerate(evidence):
            if not isinstance(item, dict):
                issues.append(EvidenceInputIssue(source_path, f"evidence[{index}] must be an object"))
                continue
            if item.get("kind") != "dependency":
                issues.append(EvidenceInputIssue(source_path, f"evidence[{index}].kind must be 'dependency'"))
                continue
            evidence_id = item.get("evidence_id")
            source = item.get("source")
            target = item.get("target")
            if not all(isinstance(value, str) and value.strip() for value in (evidence_id, source, target)):
                issues.append(EvidenceInputIssue(source_path, f"evidence[{index}] requires non-empty evidence_id/source/target"))
                continue
            edge_type, edge_error = _enum_value(EdgeType, item.get("relationship_type"), "relationship_type")
            phase, phase_error = _enum_value(DependencyPhase, item.get("phase"), "phase")
            resolution, resolution_error = _enum_value(ResolutionStatus, item.get("resolution"), "resolution")
            scope, scope_error = _enum_value(EvidenceNodeScope, item.get("target_scope"), "target_scope")
            provenance, provenance_error = _enum_value(EvidenceProvenance, item.get("provenance"), "provenance")
            enum_error = next((value for value in (edge_error, phase_error, resolution_error, scope_error, provenance_error) if value), None)
            if enum_error:
                issues.append(EvidenceInputIssue(source_path, f"evidence[{index}]: {enum_error}"))
                continue

            source_candidate = (root / str(source)).resolve()
            try:
                source_relative = source_candidate.relative_to(root).as_posix()
                source_inside = source_candidate.is_file() and source_relative in tracked
            except (OSError, ValueError):
                source_inside = False
            if not source_inside:
                issues.append(EvidenceInputIssue(source_path, f"evidence[{index}].source does not resolve to a tracked repository file candidate"))
                continue

            resolved_path = item.get("resolved_path")
            normalized_resolved: str | None = None
            if resolved_path is not None:
                if not isinstance(resolved_path, str) or not resolved_path.strip():
                    issues.append(EvidenceInputIssue(source_path, f"evidence[{index}].resolved_path must be a non-empty string when present"))
                    continue
                target_candidate = (root / resolved_path).resolve()
                try:
                    target_inside = target_candidate.is_relative_to(root) and target_candidate.is_file()
                except (OSError, ValueError):
                    target_inside = False
                if not target_inside:
                    issues.append(EvidenceInputIssue(source_path, f"evidence[{index}].resolved_path does not resolve to a repository file"))
                    continue
                normalized_resolved = target_candidate.relative_to(root).as_posix()
            semantic_error: str | None = None
            if resolution == ResolutionStatus.RESOLVED:
                if scope != EvidenceNodeScope.PROJECT_COMPONENT or normalized_resolved is None:
                    semantic_error = "RESOLVED evidence requires PROJECT_COMPONENT target_scope and resolved_path"
            elif resolution == ResolutionStatus.EXTERNAL:
                if scope not in {EvidenceNodeScope.EXTERNAL_DEPENDENCY, EvidenceNodeScope.PLATFORM} or normalized_resolved is not None:
                    semantic_error = "EXTERNAL evidence requires EXTERNAL_DEPENDENCY/PLATFORM target_scope and no resolved_path"
            elif resolution in {ResolutionStatus.UNRESOLVED, ResolutionStatus.DYNAMIC}:
                if scope != EvidenceNodeScope.UNRESOLVED_TARGET or normalized_resolved is not None:
                    semantic_error = f"{resolution.value} evidence requires UNRESOLVED_TARGET target_scope and no resolved_path"
            if semantic_error is not None:
                issues.append(EvidenceInputIssue(source_path, f"evidence[{index}] {semantic_error}"))
                continue

            edges.append(
                DependencyEdge(
                    evidence_id=f"external:{producer_id}:{evidence_id}",
                    source=source_candidate.relative_to(root).as_posix(),
                    target=str(target),
                    edge_type=edge_type,
                    phase=phase,
                    resolution=resolution,
                    target_scope=scope,
                    provenance=provenance,
                    resolved_path=normalized_resolved,
                    adapter=f"external:{producer_id}",
                    note=(
                        f"Imported from {source_path}; producer={producer_id}@{producer_version}; "
                        f"document_sha256={digest}"
                    ),
                )
            )
            accepted += 1
        documents.append(
            ExternalEvidenceDocument(
                source_path=source_path,
                source_sha256=digest,
                producer_id=str(producer_id),
                producer_version=str(producer_version),
                subject_repository=str(subject_repository),
                subject_revision=str(subject_revision),
                evidence_count=accepted,
            )
        )

    unique_edges = {edge.evidence_id: edge for edge in edges}
    return ExternalEvidenceLoad(tuple(documents), tuple(unique_edges[key] for key in sorted(unique_edges)), tuple(issues))


def merge_external_dependencies(native: DependencyScan, external: ExternalEvidenceLoad) -> DependencyScan:
    edges = {edge.evidence_id: edge for edge in native.edges}
    for edge in external.edges:
        if edge.provenance == EvidenceProvenance.OBSERVED and edge.evidence_id not in edges:
            edges[edge.evidence_id] = edge
    issues = list(native.issues)
    adapters = set(native.adapters)
    if external.edges:
        adapters.add("external-evidence")
    for issue in external.issues:
        issues.append(DependencyScanIssue("external-evidence", issue.source_path, issue.message))
    unique_issues = {(item.adapter, item.path, item.message): item for item in issues}
    return DependencyScan(
        tuple(sorted(edges.values(), key=lambda item: (item.source, item.line or 0, item.target, item.evidence_id))),
        tuple(unique_issues[key] for key in sorted(unique_issues)),
        tuple(sorted(adapters)),
    )
