from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from .clarification.generator import declared_components
from .clarification.generator_core import build_requests, covering_components
from .clarification.model import ClarificationRequest
from .clarification.resolution import (
    DecisionAnswer,
    PreparedLocalProfile,
    prepare_local_profile,
    validate_answer,
    write_prepared_local_profile,
)
from .inspection.components import ComponentCandidate, discover_component_candidates
from .inspection.dependencies_030 import scan_dependency_edges
from .inspection.inventory import collect_inventory
from .repository.discover import RepositoryInfo, discover_repository
from .repository.snapshot import RepositorySnapshot, SnapshotComparison, capture_snapshot, compare_snapshots
from .validation.profile import validate_profile


@dataclass(frozen=True)
class AdoptionPreparation:
    repository: RepositoryInfo
    before: RepositorySnapshot
    after: RepositorySnapshot
    comparison: SnapshotComparison
    candidate: ComponentCandidate | None
    answer: DecisionAnswer
    profile_path: str | None
    prepared: PreparedLocalProfile | None
    request: ClarificationRequest | None
    status: str
    message: str | None = None

    def as_dict(self, *, apply: bool, backend: str) -> dict[str, object]:
        candidate_payload = self.candidate.as_dict() if self.candidate is not None else None
        profile_path = str(self.prepared.path) if self.prepared is not None else self.profile_path
        return {
            "format": "ptsip-adoption/v1",
            "status": self.status,
            "apply": apply,
            "backend": backend,
            "repository": self.repository.as_dict(),
            "candidate": candidate_payload,
            "decision_id": self.request.id if self.request is not None else None,
            "declaration": self.answer.as_dict(),
            "profile": {
                "path": profile_path,
                "projected_valid": self.prepared is not None
                and self.status in {"ADOPTION_PLAN", "ALREADY_DECLARED"},
            },
            "snapshot": {
                "before": self.before.as_dict(),
                "after": self.after.as_dict(),
                "comparison": self.comparison.as_dict(),
            },
            "message": self.message,
        }


def _repository_identity(repo: RepositoryInfo) -> str:
    if repo.remote and repo.remote.provider == "github" and repo.remote.repository:
        return repo.remote.repository
    return repo.root


def prepare_adoption(
    path: str | Path,
    component_id: str,
    answer: DecisionAnswer,
    profile_path: str | Path | None = None,
) -> AdoptionPreparation:
    repo = discover_repository(path)
    root = Path(repo.root).resolve()
    before = capture_snapshot(root)
    inventory = collect_inventory(root)
    dependencies = scan_dependency_edges(root)
    candidates = discover_component_candidates(root, inventory, dependencies)
    candidate = next((item for item in candidates if item.id == component_id), None)
    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)

    if candidate is None:
        return AdoptionPreparation(
            repo,
            before,
            after,
            comparison,
            None,
            answer,
            str(profile_path) if profile_path is not None else None,
            None,
            None,
            "UNKNOWN_COMPONENT",
            f"Unknown component candidate: {component_id}",
        )
    if not comparison.stable:
        return AdoptionPreparation(
            repo,
            before,
            after,
            comparison,
            candidate,
            answer,
            str(profile_path) if profile_path is not None else None,
            None,
            None,
            "STALE_EVIDENCE",
            "Repository state changed during adoption analysis.",
        )

    validation = validate_answer(answer)
    if not validation.valid:
        return AdoptionPreparation(
            repo,
            before,
            after,
            comparison,
            candidate,
            answer,
            str(profile_path) if profile_path is not None else None,
            None,
            None,
            "CONFLICT",
            "; ".join(validation.errors),
        )

    declared, _resolved_profile, profile_error = declared_components(root, profile_path)
    if profile_error:
        return AdoptionPreparation(
            repo,
            before,
            after,
            comparison,
            candidate,
            answer,
            str(profile_path) if profile_path is not None else None,
            None,
            None,
            "CONFLICT",
            f"Unable to parse existing PTSIP profile: {profile_error}",
        )
    requests = build_requests(_repository_identity(repo), [candidate], declared)
    request = requests[0] if requests else None
    covering = covering_components(candidate, declared)
    target_component_id = candidate.id
    if len(covering) == 1:
        declared_id = str(covering[0].get("id", "")).strip()
        if declared_id:
            target_component_id = declared_id
    elif request is not None:
        target_component_id = request.component_id

    try:
        prepared = prepare_local_profile(
            root,
            target_component_id,
            list(candidate.include),
            answer,
            profile_path,
        )
    except (FileNotFoundError, OSError, RuntimeError, ValueError) as exc:
        return AdoptionPreparation(
            repo,
            before,
            after,
            comparison,
            candidate,
            answer,
            str(profile_path) if profile_path is not None else None,
            None,
            request,
            "CONFLICT",
            str(exc),
        )

    semantic_noop = False
    if prepared.expected_source is not None:
        try:
            import yaml

            existing = yaml.safe_load(prepared.expected_source)
            projected = yaml.safe_load(prepared.content)
            semantic_noop = existing == projected
        except Exception:
            semantic_noop = prepared.expected_source == prepared.content

    return AdoptionPreparation(
        repo,
        before,
        after,
        comparison,
        candidate,
        answer,
        str(prepared.path),
        prepared,
        request,
        "ALREADY_DECLARED" if semantic_noop else "ADOPTION_PLAN",
    )


def apply_adoption(preparation: AdoptionPreparation) -> tuple[str, str | None, str | None]:
    if preparation.status == "ALREADY_DECLARED":
        path = str(preparation.prepared.path) if preparation.prepared is not None else preparation.profile_path
        return "ALREADY_DECLARED", path, None
    if preparation.status != "ADOPTION_PLAN" or preparation.prepared is None:
        return preparation.status, preparation.profile_path, preparation.message

    current = capture_snapshot(preparation.repository.root)
    stale = compare_snapshots(preparation.after, current)
    if not stale.stable:
        return (
            "STALE_EVIDENCE",
            str(preparation.prepared.path),
            "Repository evidence changed before adoption application.",
        )

    profile = write_prepared_local_profile(preparation.prepared)
    result = validate_profile(preparation.repository.root, profile)
    if not result.valid:
        return "CONFLICT", str(profile), "Applied PTSIP profile did not validate: " + "; ".join(result.errors)
    return "ADOPTED", str(profile), None
