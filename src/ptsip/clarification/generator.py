from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from ..inspection.components import ComponentCandidate, discover_component_candidates
from ..inspection.dependencies_030 import scan_dependency_edges
from ..inspection.inventory import collect_inventory
from ..repository.discover import RepositoryInfo, discover_repository
from ..repository.snapshot import (
    RepositorySnapshot,
    SnapshotComparison,
    capture_snapshot,
    compare_snapshots,
    repository_files,
)
from ..validation.components import (
    AMBIGUOUS,
    ASSOCIATED_ARTIFACT_COVERED,
    COMPONENT_COVERED,
    partition_components,
    resolve_candidate_coverage,
    selector_matches_path,
)
from ..validation.profile import ValidationResult, find_profile, validate_profile
from .generator_core import build_requests
from .model import ClarificationRequest
from .render import request_payload


@dataclass(frozen=True)
class ProfileRecovery:
    failure_stage: str
    selected_profile_path: str

    def as_dict(self) -> dict[str, object]:
        return {
            "status": "PROJECT_CORRECTION_REQUIRED",
            "authoritative": False,
            "failure_stage": self.failure_stage,
            "selected_profile_path": self.selected_profile_path,
            "raw_profile_fallback": False,
            "reuse_partial_effective_state": False,
            "retry_requires_fresh_repository_snapshot": True,
            "actions": [
                {
                    "id": "correct_project_profile",
                    "kind": "PROJECT_SOURCE_CHANGE",
                    "description": "Correct the selected project-owned PTSIP profile without Tool-inferred architecture repair.",
                },
                {
                    "id": "retry_clarification",
                    "kind": "RETRY",
                    "description": "Re-run clarification/adoption after correction so validation resolves a fresh effective profile.",
                },
            ],
        }


@dataclass(frozen=True)
class ClarificationAnalysis:
    repository: RepositoryInfo
    before: RepositorySnapshot
    after: RepositorySnapshot
    comparison: SnapshotComparison
    candidate_ids: tuple[str, ...]
    requests: tuple[ClarificationRequest, ...]
    profile_path: str | None
    profile_parse_error: str | None
    profile_failure_stage: str | None = None
    profile_recovery: ProfileRecovery | None = None

    @property
    def status(self) -> str:
        if not self.comparison.stable:
            return "INVALIDATED"
        if self.profile_parse_error is not None:
            return "PROFILE_INVALID"
        if self.requests:
            return "CLARIFICATION_REQUIRED"
        return "NO_CLARIFICATION_REQUIRED"

    def as_dict(self, language: str) -> dict[str, object]:
        return {
            "format": "ptsip-clarification/v1",
            "status": self.status,
            "language": language,
            "inference": {
                "mode": "DETERMINISTIC_RULES_ONLY",
                "llm_calls": 0,
                "speculative_classification": False,
            },
            "repository": self.repository.as_dict(),
            "snapshot": {
                "before": self.before.as_dict(),
                "after": self.after.as_dict(),
                "comparison": self.comparison.as_dict(),
            },
            "profile": {
                "path": self.profile_path,
                "parse_error": self.profile_parse_error,
                "failure_stage": self.profile_failure_stage,
                "recovery": self.profile_recovery.as_dict() if self.profile_recovery is not None else None,
            },
            "component_candidate_count": len(self.candidate_ids),
            "clarification_count": len(self.requests),
            "requests": [request_payload(item, language) for item in self.requests],
        }


def _validation_failure_stage(validation: ValidationResult) -> str:
    errors = validation.errors
    if any(error.startswith("Unable to parse profile:") or error.startswith("<root>: profile must be") for error in errors):
        return "PARSE"
    if any(
        "Profile specification binding is not supported" in error
        or error.startswith("Profile revision ")
        for error in errors
    ):
        return "BINDING"
    if any("materialization failed:" in error for error in errors):
        return "MATERIALIZATION"
    if any(error.startswith("effective.") for error in errors):
        return "EFFECTIVE_SCHEMA"
    if any(
        (
            error.startswith("components:")
            or error.startswith("associated_artifacts:")
            or error.startswith("responsibility_map: tracked path")
        )
        and (
            "selector" in error
            or "tracked path" in error
            or "scope conflict" in error
            or "ownership conflict" in error
        )
        for error in errors
    ):
        return "SELECTOR"
    if validation.resolved_profile is not None:
        return "EFFECTIVE_SEMANTIC"
    if any(error.startswith("responsibility_map.overrides") for error in errors):
        return "SOURCE_SEMANTIC"
    return "SOURCE_SCHEMA"


def _recovery(profile_path: str, stage: str) -> ProfileRecovery:
    return ProfileRecovery(failure_stage=stage, selected_profile_path=profile_path)


def _declared_profile_scopes(
    root: str | Path,
    explicit_profile: str | Path | None = None,
) -> tuple[
    list[dict[str, object]],
    list[dict[str, object]],
    str | None,
    str | None,
    str | None,
    ProfileRecovery | None,
]:
    """Return validated effective component/artifact scopes for read consumers.

    Raw Project Profile YAML is not a second architecture interpretation. A
    present profile must pass canonical validation/materialization and provide a
    ``ResolvedProfile`` before clarification/adoption may consume architecture.

    G5 recovery state is diagnostic only for an existing invalid profile: it
    identifies where validation failed, exposes the exact selected profile path,
    and requires a fresh retry after the project corrects its own source. A
    missing profile remains the normal pre-adoption state and must not be treated
    as invalid architecture. Recovery never repairs architecture or falls back to
    partial/raw declarations.
    """

    root = Path(root).resolve()
    profile = find_profile(root, explicit_profile)
    if profile is None:
        return [], [], None, None, None, None

    validation = validate_profile(root, profile)
    if not validation.valid:
        message = "; ".join(validation.errors) or "profile validation failed"
        selected = str(profile)
        stage = _validation_failure_stage(validation)
        return [], [], selected, message, stage, _recovery(selected, stage)
    resolved = validation.resolved_profile
    if resolved is None:
        selected = str(profile)
        message = "profile validation produced no resolved effective map"
        stage = "RESOLUTION"
        return [], [], selected, message, stage, _recovery(selected, stage)

    payload = resolved.effective_payload
    raw_components = payload.get("components")
    components = (
        [item for item in raw_components if isinstance(item, dict)]
        if isinstance(raw_components, list)
        else []
    )
    raw_artifacts = payload.get("associated_artifacts")
    associated_artifacts = (
        [item for item in raw_artifacts if isinstance(item, dict)]
        if isinstance(raw_artifacts, list)
        else []
    )
    return components, associated_artifacts, str(profile), None, None, None


def declared_components(
    root: str | Path,
    explicit_profile: str | Path | None = None,
) -> tuple[list[dict[str, object]], str | None, str | None]:
    """Return validated effective component declarations for read-side callers.

    Associated artifacts are deliberately not returned as components. They are
    project-owned non-component scopes and are handled separately by clarification
    analysis so they do not become speculative component questions.
    """

    components, _associated_artifacts, profile_path, error, _stage, _recovery_state = _declared_profile_scopes(
        root, explicit_profile
    )
    return components, profile_path, error


def _candidate_scope_is_fully_partitioned(
    root: Path,
    candidate: ComponentCandidate,
    components: list[dict[str, object]],
    associated_artifacts: list[dict[str, object]],
) -> bool:
    """Return True when current repository evidence is fully owned by declared scopes.

    Discovery candidates are intentionally coarse evidence. A candidate such as
    ``tests/**`` may legitimately span several explicitly declared Verification
    components. That is not architecture ambiguity when every current tracked
    path inside the candidate has exactly one owner in the validated effective
    Responsibility Map.

    This check never invents or chooses an owner. It only proves that existing
    project authority already partitions the observed candidate scope completely.
    """

    _mode, paths, scan_errors = repository_files(root)
    if scan_errors:
        return False

    candidate_paths = {
        path
        for path in paths
        if any(selector_matches_path(path, selector) for selector in candidate.include)
    }
    if not candidate_paths:
        return False

    component_partition = partition_components(root, components)
    if component_partition.scan_errors or component_partition.conflicts:
        return False
    component_paths = {item.path for item in component_partition.assignments}

    artifact_paths: set[str] = set()
    if associated_artifacts:
        artifact_partition = partition_components(root, associated_artifacts)
        if artifact_partition.scan_errors or artifact_partition.conflicts:
            return False
        artifact_paths = {item.path for item in artifact_partition.assignments}

    if component_paths & artifact_paths:
        return False

    return candidate_paths <= (component_paths | artifact_paths)


def analyze_clarifications(
    path: str | Path = ".",
    component_ids: list[str] | tuple[str, ...] | None = None,
    profile_path: str | Path | None = None,
) -> ClarificationAnalysis:
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    before = capture_snapshot(root)
    inventory = collect_inventory(root)
    dependencies = scan_dependency_edges(root)
    all_candidates = discover_component_candidates(root, inventory, dependencies)
    candidate_ids = tuple(item.id for item in all_candidates)

    selected = set(component_ids or ())
    if selected:
        unknown = sorted(selected - set(candidate_ids))
        if unknown:
            raise ValueError("Unknown component candidate(s): " + ", ".join(unknown))
        candidates: list[ComponentCandidate] = [item for item in all_candidates if item.id in selected]
    else:
        candidates = all_candidates

    (
        declared,
        associated_artifacts,
        resolved_profile_path,
        profile_error,
        profile_failure_stage,
        profile_recovery,
    ) = _declared_profile_scopes(root, profile_path)

    identity = (
        repo.remote.repository
        if repo.remote and repo.remote.provider == "github" and repo.remote.repository
        else repo.root
    )

    requests_list: list[ClarificationRequest] = []
    if profile_error is None:
        for candidate in candidates:
            coverage = resolve_candidate_coverage(candidate, declared, associated_artifacts)
            if coverage.status in {COMPONENT_COVERED, ASSOCIATED_ARTIFACT_COVERED}:
                continue
            if coverage.status == AMBIGUOUS and _candidate_scope_is_fully_partitioned(
                root,
                candidate,
                declared,
                associated_artifacts,
            ):
                continue
            # Ambiguous effective selector coverage must reopen review unless the
            # validated Responsibility Map already proves complete path-level
            # partitioning of this coarse discovery candidate. Uncovered
            # candidates may still use the declared component set so canonical
            # best-coverage semantics remain centralized in generator_core's
            # compatibility wrapper.
            request_components = [] if coverage.status == AMBIGUOUS else declared
            requests_list.extend(build_requests(identity, [candidate], request_components))

    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    return ClarificationAnalysis(
        repository=repo,
        before=before,
        after=after,
        comparison=comparison,
        candidate_ids=candidate_ids,
        requests=tuple(requests_list),
        profile_path=resolved_profile_path,
        profile_parse_error=profile_error,
        profile_failure_stage=profile_failure_stage,
        profile_recovery=profile_recovery,
    )
