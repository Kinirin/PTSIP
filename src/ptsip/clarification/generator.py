from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from ..inspection.components import ComponentCandidate, discover_component_candidates
from ..inspection.dependencies_030 import scan_dependency_edges
from ..inspection.inventory import collect_inventory
from ..repository.discover import RepositoryInfo, discover_repository
from ..repository.snapshot import RepositorySnapshot, SnapshotComparison, capture_snapshot, compare_snapshots
from ..validation.components import (
    AMBIGUOUS,
    ASSOCIATED_ARTIFACT_COVERED,
    COMPONENT_COVERED,
    resolve_candidate_coverage,
)
from ..validation.profile import find_profile, validate_profile
from .generator_core import build_requests
from .model import ClarificationRequest
from .render import request_payload


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
            },
            "component_candidate_count": len(self.candidate_ids),
            "clarification_count": len(self.requests),
            "requests": [request_payload(item, language) for item in self.requests],
        }


def _declared_profile_scopes(
    root: str | Path,
    explicit_profile: str | Path | None = None,
) -> tuple[list[dict[str, object]], list[dict[str, object]], str | None, str | None]:
    """Return validated effective component/artifact scopes for read consumers.

    Raw Project Profile YAML is not a second architecture interpretation.  A
    present profile must pass canonical validation/materialization and provide a
    ``ResolvedProfile`` before clarification/adoption may consume architecture.
    """

    root = Path(root).resolve()
    profile = find_profile(root, explicit_profile)
    if profile is None:
        return [], [], None, None

    validation = validate_profile(root, profile)
    if not validation.valid:
        message = "; ".join(validation.errors) or "profile validation failed"
        return [], [], str(profile), message
    resolved = validation.resolved_profile
    if resolved is None:
        return [], [], str(profile), "profile validation produced no resolved effective map"

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
    return components, associated_artifacts, str(profile), None


def declared_components(
    root: str | Path,
    explicit_profile: str | Path | None = None,
) -> tuple[list[dict[str, object]], str | None, str | None]:
    """Return validated effective component declarations for read-side callers.

    Associated artifacts are deliberately not returned as components. They are
    project-owned non-component scopes and are handled separately by clarification
    analysis so they do not become speculative component questions.
    """

    components, _associated_artifacts, profile_path, error = _declared_profile_scopes(
        root, explicit_profile
    )
    return components, profile_path, error


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

    declared, associated_artifacts, resolved_profile_path, profile_error = _declared_profile_scopes(
        root, profile_path
    )

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
            # Ambiguous effective selector coverage must reopen review without
            # guessing one existing owner.  Uncovered candidates may still use
            # the declared component set so canonical best-coverage semantics
            # remain centralized in generator_core's compatibility wrapper.
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
    )
