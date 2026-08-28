from __future__ import annotations

import hashlib
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable

import yaml

from ..model import DependencyPhase, EvidenceProvenance
from ..repository.profile_path import normalize_profile_path, profile_path_on_disk
from ..repository.profile_transition import (
    ProfileGenerationIdentity,
    ProfileTransitionState,
    discover_profile_transition,
    validate_transition_snapshot,
)
from ..repository.snapshot import repository_files
from ..validation.components import AMBIGUOUS, CandidateCoverage, normalize_selector, resolve_candidate_coverage
from .dependencies import DependencyScan
from .inventory import Inventory, collect_inventory

_SCRIPT_SUFFIXES = {".py", ".ps1", ".sh", ".bash", ".bat", ".cmd"}
_CONTRACT_SUFFIXES = {".json", ".yaml", ".yml", ".proto", ".avsc", ".xsd"}
_PACKAGE_ASSEMBLY_NAMES = {
    "pyproject.toml",
    "setup.py",
    "setup.cfg",
    "MANIFEST.in",
    "package.json",
    "Cargo.toml",
    "go.mod",
    "pom.xml",
    "build.gradle",
    "build.gradle.kts",
}


@dataclass(frozen=True)
class CandidateDiscoveryIssue:
    code: str
    message: str
    path: str | None = None
    adapter: str | None = None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class SourceGenerationContext:
    profile_path: str
    version: str
    specification_revision: str
    content_sha256: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class CandidateDiscoveryContext:
    evaluation_id: str
    repository_root: str
    repository_head: str | None
    repository_status_fingerprint: str
    repository_content_fingerprint: str
    transition_mode: str
    final_point_path: str | None
    source_generation: SourceGenerationContext

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class CandidateObservation:
    kind: str
    evidence_id: str
    provenance: EvidenceProvenance
    adapter: str
    path: str | None = None
    detail: str | None = None

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["provenance"] = self.provenance.value
        return payload


@dataclass(frozen=True)
class CandidateEvidence:
    id: str
    include: tuple[str, ...]
    observations: tuple[CandidateObservation, ...]
    coverage: CandidateCoverage
    context: CandidateDiscoveryContext

    @property
    def ambiguous(self) -> bool:
        return self.coverage.status == AMBIGUOUS

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "include": list(self.include),
            "observations": [item.as_dict() for item in self.observations],
            "coverage": self.coverage.as_dict(),
            "context": self.context.as_dict(),
            "ambiguous": self.ambiguous,
            "authority": "EVIDENCE_ONLY",
        }


@dataclass(frozen=True)
class CandidateDiscoveryResult:
    candidates: tuple[CandidateEvidence, ...]
    issues: tuple[CandidateDiscoveryIssue, ...]
    context: CandidateDiscoveryContext | None
    transition_state: ProfileTransitionState | None

    @property
    def complete(self) -> bool:
        return self.context is not None and not self.issues

    def as_dict(self) -> dict[str, object]:
        return {
            "complete": self.complete,
            "context": self.context.as_dict() if self.context else None,
            "candidates": [item.as_dict() for item in self.candidates],
            "issues": [item.as_dict() for item in self.issues],
        }


def _stable_candidate_id(selectors: tuple[str, ...]) -> str:
    digest = hashlib.sha256("\0".join(selectors).encode("utf-8")).hexdigest()[:16]
    return f"candidate:{digest}"


def _evaluation_id(state: ProfileTransitionState, source: ProfileGenerationIdentity) -> str:
    snapshot = state.snapshot.repository
    parts = (
        state.snapshot.repository_root,
        snapshot.head or "<no-head>",
        snapshot.status_fingerprint,
        snapshot.tracked_content_fingerprint,
        source.path,
        source.declared_version,
        source.specification_revision,
        source.content_sha256,
        state.final_point.path if state.final_point else "<no-final-point>",
    )
    return hashlib.sha256("\0".join(parts).encode("utf-8")).hexdigest()


def _context(state: ProfileTransitionState, source: ProfileGenerationIdentity) -> CandidateDiscoveryContext:
    snapshot = state.snapshot.repository
    return CandidateDiscoveryContext(
        evaluation_id=_evaluation_id(state, source),
        repository_root=state.snapshot.repository_root,
        repository_head=snapshot.head,
        repository_status_fingerprint=snapshot.status_fingerprint,
        repository_content_fingerprint=snapshot.tracked_content_fingerprint,
        transition_mode=state.mode,
        final_point_path=state.final_point.path if state.final_point else None,
        source_generation=SourceGenerationContext(
            profile_path=source.path,
            version=source.declared_version,
            specification_revision=source.specification_revision,
            content_sha256=source.content_sha256,
        ),
    )


def _eligible_sources(state: ProfileTransitionState) -> tuple[ProfileGenerationIdentity, ...]:
    if state.ordered_sources:
        return state.ordered_sources
    return (state.canonical_source,)


def _select_source(
    state: ProfileTransitionState,
    source_profile_path: str | Path | None,
) -> tuple[ProfileGenerationIdentity | None, CandidateDiscoveryIssue | None]:
    eligible = _eligible_sources(state)
    if source_profile_path is None:
        return eligible[0], None
    try:
        requested = normalize_profile_path(source_profile_path)
    except ValueError as exc:
        return None, CandidateDiscoveryIssue("INVALID_SOURCE_PROFILE_PATH", str(exc))

    if state.final_point is not None and requested == state.final_point.path:
        return None, CandidateDiscoveryIssue(
            "FINAL_POINT_IS_NOT_SOURCE",
            "Final PTSIP Point File is a migration target and cannot be used as the source-generation evidence context.",
            requested,
        )
    for item in eligible:
        if item.path == requested:
            return item, None
    return None, CandidateDiscoveryIssue(
        "UNKNOWN_SOURCE_GENERATION",
        f"Profile {requested!r} is not an eligible source generation in the current transition state.",
        requested,
    )


def _source_declarations(
    repository_root: Path,
    source: ProfileGenerationIdentity,
) -> tuple[list[dict[str, object]], list[dict[str, object]], list[CandidateDiscoveryIssue]]:
    path = profile_path_on_disk(repository_root, source.path)
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, yaml.YAMLError) as exc:
        return [], [], [CandidateDiscoveryIssue("SOURCE_PROFILE_READ_ERROR", str(exc), source.path)]
    if not isinstance(payload, dict):
        return [], [], [CandidateDiscoveryIssue("SOURCE_PROFILE_SHAPE_ERROR", "Source profile root must be a mapping.", source.path)]

    components = payload.get("components")
    artifacts = payload.get("associated_artifacts", [])
    component_items = [item for item in components if isinstance(item, dict)] if isinstance(components, list) else []
    artifact_items = [item for item in artifacts if isinstance(item, dict)] if isinstance(artifacts, list) else []
    return component_items, artifact_items, []


def _architecture_root(path: str) -> str:
    parts = Path(path.replace("\\", "/")).parts
    if not parts:
        return ""
    if parts[0] in {"src", "lib", "libs", "packages"} and len(parts) > 1:
        return "/".join(parts[:2])
    if parts[0] == ".github" and len(parts) > 1:
        return "/".join(parts[:2])
    return parts[0]


def _source_package_roots(paths: Iterable[str]) -> tuple[str, ...]:
    roots: set[str] = set()
    for rel in paths:
        path = Path(rel.replace("\\", "/"))
        parts = path.parts
        if len(parts) >= 3 and parts[0] in {"src", "lib", "libs", "packages"}:
            if path.name in {"__init__.py", "package.json", "Cargo.toml", "go.mod"}:
                roots.add("/".join(parts[:2]) + "/**")
    return tuple(sorted(roots))


def _provenance(value: object) -> EvidenceProvenance:
    if isinstance(value, EvidenceProvenance):
        return value
    try:
        return EvidenceProvenance(str(value))
    except ValueError:
        return EvidenceProvenance.OBSERVED


def build_candidate_evidence(
    repository_root: str | Path,
    *,
    inventory: Inventory,
    dependencies: DependencyScan,
    state: ProfileTransitionState,
    source: ProfileGenerationIdentity,
    components: list[dict[str, object]],
    associated_artifacts: list[dict[str, object]],
) -> tuple[CandidateEvidence, ...]:
    root = Path(repository_root).resolve()
    context = _context(state, source)
    _mode, paths, _scan_errors = repository_files(root)
    raw: dict[tuple[str, ...], dict[tuple[str, str, str, str | None, str | None], CandidateObservation]] = {}

    def add(
        selectors: Iterable[str],
        *,
        kind: str,
        evidence_id: str,
        provenance: EvidenceProvenance,
        adapter: str,
        path: str | None = None,
        detail: str | None = None,
    ) -> None:
        cleaned = {normalize_selector(str(item)) for item in selectors}
        normalized = tuple(sorted(item for item in cleaned if item))
        if not normalized:
            return
        observations = raw.setdefault(normalized, {})
        observation = CandidateObservation(kind, evidence_id, provenance, adapter, path, detail)
        key = (kind, evidence_id, adapter, path, detail)
        observations[key] = observation

    for manifest in inventory.manifests:
        parent = Path(manifest).parent.as_posix()
        selector = manifest if parent in {".", ""} else f"{parent}/**"
        add(
            (selector,),
            kind="MANIFEST",
            evidence_id=f"manifest:{manifest}",
            provenance=EvidenceProvenance.OBSERVED,
            adapter="inventory",
            path=manifest,
            detail="Tracked package/build manifest exists.",
        )

    schema_parents: dict[str, int] = {}
    for schema in inventory.schema_candidates:
        parent = Path(schema).parent.as_posix()
        schema_parents[parent] = schema_parents.get(parent, 0) + 1
    for parent, count in sorted(schema_parents.items()):
        selector = f"{parent}/**" if parent not in {".", ""} else "*.schema.*"
        add(
            (selector,),
            kind="CONTRACT_GROUP",
            evidence_id=f"schema-group:{parent}:{count}",
            provenance=EvidenceProvenance.INFERRED,
            adapter="inventory",
            path=None if parent in {".", ""} else parent,
            detail=f"{count} tracked schema/contract candidate(s) share this repository scope.",
        )

    for name in inventory.tool_like_roots:
        add(
            (f"{name}/**",),
            kind="TOOL_ROOT",
            evidence_id=f"root:{name}",
            provenance=EvidenceProvenance.INFERRED,
            adapter="inventory",
            path=name,
            detail="Top-level tooling-like repository root.",
        )
    for name in inventory.test_roots:
        add(
            (f"{name}/**",),
            kind="TEST_ROOT",
            evidence_id=f"root:{name}",
            provenance=EvidenceProvenance.INFERRED,
            adapter="inventory",
            path=name,
            detail="Top-level verification/test repository root.",
        )
    for selector in _source_package_roots(paths):
        root_path = selector.removesuffix("/**")
        add(
            (selector,),
            kind="SOURCE_PACKAGE_ROOT",
            evidence_id=f"source-root:{root_path}",
            provenance=EvidenceProvenance.INFERRED,
            adapter="repository-files",
            path=root_path,
            detail="Package root derived from tracked package marker files.",
        )

    for rel in paths:
        path = Path(rel)
        if path.name in _PACKAGE_ASSEMBLY_NAMES:
            add(
                (rel,),
                kind="PACKAGE_ASSEMBLY_INPUT",
                evidence_id=f"package-assembly:{rel}",
                provenance=EvidenceProvenance.OBSERVED,
                adapter="repository-files",
                path=rel,
                detail="Tracked package/distribution assembly input.",
            )
        lower_parts = tuple(part.lower() for part in path.parts)
        if path.suffix.lower() in _CONTRACT_SUFFIXES and "specdata" in lower_parts:
            parent = path.parent.as_posix()
            add(
                (f"{parent}/**",),
                kind="EMBEDDED_CONTRACT_COPY",
                evidence_id=f"embedded-contract:{rel}",
                provenance=EvidenceProvenance.OBSERVED,
                adapter="repository-files",
                path=rel,
                detail="Tracked contract asset is embedded under a specdata package scope.",
            )
        if rel.startswith(".github/scripts/") or (len(path.parts) == 1 and path.suffix.lower() in _SCRIPT_SUFFIXES):
            add(
                (rel,),
                kind="MAINTENANCE_SCRIPT",
                evidence_id=f"maintenance-script:{rel}",
                provenance=EvidenceProvenance.OBSERVED,
                adapter="repository-files",
                path=rel,
                detail="Tracked repository automation/maintenance script.",
            )

    for edge in dependencies.edges:
        provenance = _provenance(edge.provenance)
        if edge.adapter == "github-actions" and edge.resolved_path:
            add(
                (edge.resolved_path,),
                kind="CI_INVOKED_SCRIPT",
                evidence_id=edge.evidence_id,
                provenance=provenance,
                adapter=edge.adapter,
                path=edge.resolved_path,
                detail=f"Invoked from {edge.source}.",
            )
        if edge.phase == DependencyPhase.RELEASE:
            add(
                (edge.source,),
                kind="RELEASE_ACTIVITY_SOURCE",
                evidence_id=edge.evidence_id,
                provenance=provenance,
                adapter=edge.adapter,
                path=edge.source,
                detail=f"Dependency evidence participates in RELEASE phase ({edge.edge_type.value}).",
            )
            if edge.resolved_path:
                add(
                    (edge.resolved_path,),
                    kind="RELEASE_ACTIVITY_TARGET",
                    evidence_id=edge.evidence_id,
                    provenance=provenance,
                    adapter=edge.adapter,
                    path=edge.resolved_path,
                    detail=f"Resolved RELEASE-phase target from {edge.source}.",
                )
        if edge.resolved_path and _architecture_root(edge.source) != _architecture_root(edge.resolved_path):
            add(
                (edge.resolved_path,),
                kind="DEPENDENCY_BOUNDARY",
                evidence_id=edge.evidence_id,
                provenance=provenance,
                adapter=edge.adapter,
                path=edge.resolved_path,
                detail=f"Resolved edge crosses repository roots: {_architecture_root(edge.source)} -> {_architecture_root(edge.resolved_path)}.",
            )

    candidates: list[CandidateEvidence] = []
    for selectors, observations_by_key in sorted(raw.items(), key=lambda item: item[0]):
        observations = tuple(
            observations_by_key[key]
            for key in sorted(
                observations_by_key,
                key=lambda item: tuple("" if value is None else str(value) for value in item),
            )
        )
        coverage = resolve_candidate_coverage(selectors, components, associated_artifacts)
        candidates.append(
            CandidateEvidence(
                id=_stable_candidate_id(selectors),
                include=selectors,
                observations=observations,
                coverage=coverage,
                context=context,
            )
        )
    return tuple(candidates)


def discover_candidate_evidence(
    repository_root: str | Path,
    *,
    source_profile_path: str | Path | None = None,
    inventory: Inventory | None = None,
    dependencies: DependencyScan | None = None,
) -> CandidateDiscoveryResult:
    root = Path(repository_root).expanduser().resolve()
    transition = discover_profile_transition(root)
    if transition.state is None:
        issues = tuple(
            CandidateDiscoveryIssue(
                f"TRANSITION_{item.code}",
                item.message,
                item.path,
                "profile-transition",
            )
            for item in transition.diagnostics
        )
        return CandidateDiscoveryResult((), issues, None, None)

    state = transition.state
    source, source_issue = _select_source(state, source_profile_path)
    if source_issue is not None or source is None:
        return CandidateDiscoveryResult((), (source_issue,) if source_issue else (), None, state)

    components, associated_artifacts, declaration_issues = _source_declarations(root, source)
    observed_inventory = inventory if inventory is not None else collect_inventory(root)
    if dependencies is None:
        from .dependencies_030 import scan_dependency_edges

        observed_dependencies = scan_dependency_edges(root)
    else:
        observed_dependencies = dependencies

    issues: list[CandidateDiscoveryIssue] = list(declaration_issues)
    issues.extend(
        CandidateDiscoveryIssue(item.category, item.message, item.path, "inventory")
        for item in observed_inventory.scan_issues
    )
    issues.extend(
        CandidateDiscoveryIssue("DEPENDENCY_SCAN_ISSUE", item.message, item.path, item.adapter)
        for item in observed_dependencies.issues
    )

    candidates = build_candidate_evidence(
        root,
        inventory=observed_inventory,
        dependencies=observed_dependencies,
        state=state,
        source=source,
        components=components,
        associated_artifacts=associated_artifacts,
    )

    for diagnostic in validate_transition_snapshot(root, state):
        issues.append(
            CandidateDiscoveryIssue(
                diagnostic.code,
                diagnostic.message,
                diagnostic.path,
                "profile-transition",
            )
        )

    return CandidateDiscoveryResult(
        candidates=candidates,
        issues=tuple(issues),
        context=_context(state, source),
        transition_state=state,
    )


def validate_candidate_discovery_context(
    repository_root: str | Path,
    result: CandidateDiscoveryResult,
) -> tuple[CandidateDiscoveryIssue, ...]:
    if result.transition_state is None or result.context is None:
        return (CandidateDiscoveryIssue("NO_DISCOVERY_CONTEXT", "Candidate discovery result has no transition-bound context."),)
    diagnostics = validate_transition_snapshot(repository_root, result.transition_state)
    return tuple(
        CandidateDiscoveryIssue(item.code, item.message, item.path, "profile-transition")
        for item in diagnostics
    )
