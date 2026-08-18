from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import yaml

from ..inspection.components import ComponentCandidate, discover_component_candidates
from ..inspection.dependencies_030 import scan_dependency_edges
from ..inspection.inventory import collect_inventory
from ..repository.discover import RepositoryInfo, discover_repository
from ..repository.snapshot import RepositorySnapshot, SnapshotComparison, capture_snapshot, compare_snapshots
from ..validation.profile import find_profile
from .generator_core import build_requests, covering_components
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
    root = Path(root).resolve()
    profile = find_profile(root, explicit_profile)
    if profile is None:
        return [], [], None, None
    try:
        payload = yaml.safe_load(profile.read_text(encoding="utf-8-sig"))
    except Exception as exc:
        return [], [], str(profile), str(exc)
    if not isinstance(payload, dict):
        return [], [], str(profile), "profile root is not a mapping"

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
    """Return classified component declarations for adoption/reconciliation callers.

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

    # An associated artifact is already explicit project architecture declaring
    # that its covered scope is a non-component support surface. Candidate
    # discovery must not immediately ask the maintainer to classify the same
    # scope as a component. Explicit adoption/promotion remains a separate user
    # action and is intentionally not blocked here.
    candidates_requiring_component_review = [
        candidate
        for candidate in candidates
        if not covering_components(candidate, associated_artifacts)
    ]

    identity = (
        repo.remote.repository
        if repo.remote and repo.remote.provider == "github" and repo.remote.repository
        else repo.root
    )
    requests = build_requests(identity, candidates_requiring_component_review, declared)
    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    return ClarificationAnalysis(
        repository=repo,
        before=before,
        after=after,
        comparison=comparison,
        candidate_ids=candidate_ids,
        requests=requests,
        profile_path=resolved_profile_path,
        profile_parse_error=profile_error,
    )
