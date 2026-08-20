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


def _selected_profile_candidate(root: Path, explicit_profile: str | Path) -> Path:
    raw = Path(explicit_profile).expanduser()
    return raw.resolve() if raw.is_absolute() else (root / raw).resolve()


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

    G5 recovery state is diagnostic only: it identifies where validation failed,
    exposes the exact selected profile path, and requires a fresh retry after the
    project corrects its own source. It never repairs architecture or falls back
    to partial/raw declarations.
    """

    root = Path(root).resolve()
    profile = find_profile(root, explicit_profile)
    if profile is None:
        if explicit_profile is None:
            return [], [], None, None, None, None
        selected = str(_selected_profile_candidate(root, explicit_profile))
        message = "Selected PTSIP project profile was not found."
        return [], [], selected, message, "PROFILE_DISCOVERY", _recovery(selected, "PROFILE_DISCOVERY")

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
            # Ambiguous effective selector coverage must reopen review without
            # guessing one existing owner. Uncovered candidates may still use
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
        profile_failure_stage=profile_failure_stage,
        profile_recovery=profile_recovery,
    )
