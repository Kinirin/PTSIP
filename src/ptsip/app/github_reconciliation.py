from __future__ import annotations

from pathlib import Path
from typing import Iterable

from ..clarification.generator import ClarificationAnalysis, declared_components
from ..clarification.generator_core import covering_components
from ..clarification.model import ClarificationRequest
from ..clarification.render import render_issue
from ..clarification.resolution import (
    DecisionAnswer,
    LegacyDecisionAnswerV1,
    canonicalize_legacy_answer,
    load_profile_text,
    prepare_local_profile,
    validate_answer,
    write_prepared_local_profile,
)
from ..clarification.resolution.model import CANONICAL_ANSWER_FIELDS, LEGACY_V1_ANSWER_FIELDS
from ..inspection.components import ComponentCandidate, discover_component_candidates
from ..inspection.dependencies_030 import scan_dependency_edges
from ..inspection.inventory import collect_inventory
from ..repository.profile_path import normalize_profile_path, profile_path_on_disk, selected_profile_path
from ..repository.snapshot import capture_snapshot, compare_snapshots
from .github_authority import CoordinationUnavailable, GithubControlPlaneClient


def _normalize_selector(value: object) -> str:
    text = str(value).replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    return text.strip("/")


def _scope_key(include: Iterable[object]) -> tuple[str, ...]:
    return tuple(sorted({_normalize_selector(item) for item in include if _normalize_selector(item)}))


def _request_by_scope(analysis: ClarificationAnalysis) -> dict[tuple[str, ...], ClarificationRequest]:
    return {_scope_key(request.include): request for request in analysis.requests}


def _candidate_payload(candidate: ComponentCandidate) -> dict[str, object]:
    return {
        "component_id": candidate.id,
        "include": list(candidate.include),
        "anchors": list(candidate.anchors),
        "evidence_ids": list(candidate.evidence_ids),
        "missing_fields": [],
        "reason_codes": [],
        "status": "DECLARED",
    }


def _selected_candidates(
    analysis: ClarificationAnalysis,
    component_ids: list[str] | tuple[str, ...] | None,
) -> tuple[list[ComponentCandidate], object]:
    root = Path(analysis.repository.root).resolve()
    if not compare_snapshots(analysis.after, capture_snapshot(root)).stable:
        raise RuntimeError("Repository changed before GitHub authority freshness analysis")

    inventory = collect_inventory(root)
    dependencies = scan_dependency_edges(root)
    candidates = discover_component_candidates(root, inventory, dependencies)
    after = capture_snapshot(root)
    if not compare_snapshots(analysis.after, after).stable:
        raise RuntimeError("Repository changed while resolving GitHub authority component scope")

    selected = set(component_ids or ())
    if selected:
        candidates = [candidate for candidate in candidates if candidate.id in selected]
    return candidates, after


def _local_component_id(
    candidate: ComponentCandidate,
    declared: list[dict[str, object]],
) -> str:
    covering = covering_components(candidate, declared)
    if len(covering) == 1:
        declared_id = str(covering[0].get("id", "")).strip()
        if declared_id:
            return declared_id
    return candidate.id


def _profile_needs_projection(expected_source: str | None, projected_source: str) -> bool:
    return load_profile_text(expected_source) != load_profile_text(projected_source)


def _required_string(payload: dict[str, object], key: str) -> str:
    value = payload.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"Persisted authority answer field {key} must be a non-empty string")
    return value.strip()


def _required_bool(payload: dict[str, object], key: str) -> bool:
    value = payload.get(key)
    if not isinstance(value, bool):
        raise ValueError(f"Persisted authority answer field {key} must be a boolean")
    return value


def _stored_answer_from_mapping(payload: dict[str, object]) -> DecisionAnswer:
    """Read v2 authority answers and the explicit persisted-v1 compatibility shape."""

    actual = set(payload)
    if actual == set(CANONICAL_ANSWER_FIELDS):
        answer = DecisionAnswer(
            classification=_required_string(payload, "classification").upper(),
            purpose=_required_string(payload, "purpose"),
            shipped=_required_bool(payload, "shipped"),
            runtime_required=_required_bool(payload, "runtime_required"),
            executable=_required_bool(payload, "executable"),
        )
        validation = validate_answer(answer)
        if not validation.valid:
            raise ValueError("Persisted authority answer is invalid: " + "; ".join(validation.errors))
        return answer
    if actual == set(LEGACY_V1_ANSWER_FIELDS):
        legacy = LegacyDecisionAnswerV1(
            classification=_required_string(payload, "classification").upper(),
            purpose=_required_string(payload, "purpose"),
            shipped=_required_bool(payload, "shipped"),
            runtime_required=_required_bool(payload, "runtime_required"),
            lifecycle_owner=_required_string(payload, "lifecycle_owner").upper(),
            executable=_required_bool(payload, "executable"),
        )
        return canonicalize_legacy_answer(legacy)
    raise ValueError("Persisted authority answer has neither canonical v2 nor exact legacy v1 shape")


def run_github_gate(
    analysis: ClarificationAnalysis,
    client: GithubControlPlaneClient,
    component_ids: list[str] | tuple[str, ...] | None,
    profile_path: str | Path | None,
    language: str,
) -> tuple[dict[str, object], int]:
    """Run GitHub-coordinated gate/reconciliation against one exact profile path."""

    repo = analysis.repository
    if not repo.commit or not repo.branch:
        raise RuntimeError("ptsip gate requires a checked-out Git branch and commit")
    if not repo.remote or repo.remote.provider != "github" or not repo.remote.repository:
        raise RuntimeError("GitHub coordinated gate requires a GitHub repository identity")

    try:
        selected_profile = selected_profile_path(repo.root, profile_path)
        selected_profile_file = profile_path_on_disk(repo.root, selected_profile)
        candidates, baseline = _selected_candidates(analysis, component_ids)
        declared, _, profile_error = declared_components(repo.root, selected_profile_file)
        if profile_error:
            return (
                {
                    "status": "AUTHORITY_PROFILE_CONFLICT",
                    "backend": "GITHUB",
                    "repository": repo.as_dict(),
                    "profile_path": selected_profile,
                    "message": f"Selected Project Profile cannot be compared with GitHub authority: {profile_error}",
                    "decisions": [],
                },
                8,
            )

        requests = _request_by_scope(analysis)
        decisions: list[dict[str, object]] = []
        blocked = False
        errored = False
        authority_profile_conflict = False
        saw_resolved = False

        for candidate in candidates:
            request = requests.get(_scope_key(candidate.include))
            request_payload = request.as_dict() if request is not None else _candidate_payload(candidate)
            peeked = client.peek(
                {
                    "repository": repo.remote.repository,
                    "request": request_payload,
                    "profile_path": selected_profile,
                }
            )
            status = str(peeked.get("status", ""))

            if status == "NO_AUTHORITY_DECISION":
                if request is None:
                    decisions.append(
                        {
                            **peeked,
                            "status": "NO_DECISION_REQUIRED",
                            "component_id": _local_component_id(candidate, declared),
                            "profile_path": selected_profile,
                            "reconciliation": {"status": "LOCAL_DECLARATION_ONLY"},
                        }
                    )
                    continue
                title, body = render_issue(request, language, repo.commit)
                response = client.gate(
                    {
                        "id": request.id,
                        "repository": repo.remote.repository,
                        "branch": repo.branch,
                        "subject_revision": repo.commit,
                        "profile_path": selected_profile,
                        "component_id": request.component_id,
                        "request": request.as_dict(),
                        "issue": {"title": title, "body": body},
                    }
                )
                status = str(response.get("status", ""))
                decisions.append(response)
                if status == "DECISION_REQUIRED":
                    blocked = True
                elif status not in {"RESOLVED", "RESOLVED_APPLICATION_REQUIRED"}:
                    errored = True
                continue

            if status == "DECISION_REQUIRED":
                decisions.append(peeked)
                blocked = True
                continue

            if status != "RESOLVED_APPLICATION_REQUIRED":
                decisions.append(peeked)
                errored = True
                continue

            decision = peeked.get("decision")
            if not isinstance(decision, dict) or not isinstance(decision.get("answer"), dict):
                decisions.append(
                    {
                        **peeked,
                        "status": "DECISION_ERROR",
                        "message": "Resolved GitHub authority record has no valid answer.",
                    }
                )
                errored = True
                continue
            if normalize_profile_path(decision.get("profile_path")) != selected_profile:
                decisions.append(
                    {
                        **peeked,
                        "status": "AUTHORITY_PROFILE_CONFLICT",
                        "message": "Resolved GitHub authority decision targets a different Project Profile path.",
                    }
                )
                authority_profile_conflict = True
                continue

            if not compare_snapshots(baseline, capture_snapshot(repo.root)).stable:
                return (
                    {
                        "status": "STALE_EVIDENCE",
                        "backend": "GITHUB",
                        "repository": repo.as_dict(),
                        "profile_path": selected_profile,
                        "message": "Repository changed before authoritative profile reconciliation.",
                        "decisions": decisions,
                    },
                    8,
                )

            try:
                answer = _stored_answer_from_mapping(decision["answer"])
            except ValueError as exc:
                decisions.append(
                    {
                        **peeked,
                        "status": "DECISION_ERROR",
                        "message": str(exc),
                    }
                )
                errored = True
                continue
            local_component_id = _local_component_id(candidate, declared)
            try:
                prepared = prepare_local_profile(
                    repo.root,
                    local_component_id,
                    list(candidate.include),
                    answer,
                    selected_profile_file,
                )
            except (FileNotFoundError, OSError, RuntimeError, ValueError) as exc:
                decisions.append(
                    {
                        **peeked,
                        "status": "AUTHORITY_PROFILE_CONFLICT",
                        "message": str(exc),
                        "reconciliation": {
                            "status": "CONFLICT",
                            "component_id": local_component_id,
                            "profile_path": selected_profile,
                        },
                    }
                )
                authority_profile_conflict = True
                continue

            if not _profile_needs_projection(prepared.expected_source, prepared.content):
                decisions.append(
                    {
                        **peeked,
                        "status": "RESOLVED",
                        "reconciliation": {
                            "status": "CONSISTENT",
                            "component_id": local_component_id,
                            "profile_path": selected_profile,
                        },
                    }
                )
                saw_resolved = True
                continue

            if not compare_snapshots(baseline, capture_snapshot(repo.root)).stable:
                return (
                    {
                        "status": "STALE_EVIDENCE",
                        "backend": "GITHUB",
                        "repository": repo.as_dict(),
                        "profile_path": selected_profile,
                        "message": "Repository changed after profile projection validation.",
                        "decisions": decisions,
                    },
                    8,
                )

            try:
                profile = write_prepared_local_profile(prepared)
            except (FileNotFoundError, OSError, RuntimeError, ValueError) as exc:
                return (
                    {
                        "status": "STALE_EVIDENCE",
                        "backend": "GITHUB",
                        "repository": repo.as_dict(),
                        "profile_path": selected_profile,
                        "message": str(exc),
                        "decisions": decisions,
                    },
                    8,
                )

            application = client.application(
                {
                    "decision_id": str(decision.get("id", "")),
                    "status": "LOCAL_APPLIED",
                    "profile_path": selected_profile,
                    "applied_revision": repo.commit,
                }
            )
            decisions.append(
                {
                    **peeked,
                    "status": "RESOLVED",
                    "reconciliation": {
                        "status": "LOCAL_APPLIED",
                        "component_id": local_component_id,
                        "profile_path": selected_profile,
                        "application": application,
                    },
                }
            )
            saw_resolved = True
            baseline = capture_snapshot(repo.root)
            declared, _, _ = declared_components(repo.root, selected_profile_file)

        if authority_profile_conflict:
            overall = "AUTHORITY_PROFILE_CONFLICT"
            exit_code = 8
        elif blocked:
            overall = "DECISION_REQUIRED"
            exit_code = 7
        elif errored:
            overall = "DECISION_ERROR"
            exit_code = 8
        elif saw_resolved:
            overall = "RESOLVED"
            exit_code = 0
        else:
            overall = "NO_DECISION_REQUIRED"
            exit_code = 0

        return (
            {
                "status": overall,
                "backend": "GITHUB",
                "repository": repo.as_dict(),
                "profile_path": selected_profile,
                "decisions": decisions,
            },
            exit_code,
        )
    except CoordinationUnavailable as exc:
        return (
            {
                "status": "COORDINATION_UNAVAILABLE",
                "backend": "GITHUB",
                "repository": repo.as_dict(),
                "message": str(exc),
                "decisions": [],
            },
            8,
        )
