from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import yaml

from ..inspection.components import ComponentCandidate, discover_component_candidates
from ..inspection.dependencies import scan_dependency_edges
from ..inspection.inventory import collect_inventory
from ..repository.discover import RepositoryInfo, discover_repository
from ..repository.snapshot import RepositorySnapshot, SnapshotComparison, capture_snapshot, compare_snapshots
from ..validation.profile import find_profile
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


def _declared_components(root: Path) -> tuple[list[dict[str, object]], str | None, str | None]:
    profile = find_profile(root)
    if profile is None:
        return [], None, None
    try:
        payload = yaml.safe_load(profile.read_text(encoding="utf-8"))
    except Exception as exc:
        return [], str(profile), str(exc)
    if not isinstance(payload, dict):
        return [], str(profile), "profile root is not a mapping"
    raw = payload.get("components")
    if not isinstance(raw, list):
        return [], str(profile), None
    components = [item for item in raw if isinstance(item, dict)]
    return components, str(profile), None


def analyze_clarifications(
    path: str | Path = ".",
    component_ids: list[str] | tuple[str, ...] | None = None,
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

    declared, profile_path, profile_error = _declared_components(root)
    identity = (
        repo.remote.repository
        if repo.remote and repo.remote.provider == "github" and repo.remote.repository
        else repo.root
    )
    requests = build_requests(identity, candidates, declared)
    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    return ClarificationAnalysis(
        repository=repo,
        before=before,
        after=after,
        comparison=comparison,
        candidate_ids=candidate_ids,
        requests=requests,
        profile_path=profile_path,
        profile_parse_error=profile_error,
    )
