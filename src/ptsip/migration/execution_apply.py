from __future__ import annotations

import copy
import os
from pathlib import Path
from typing import Callable, Mapping

import yaml

from ..repository.profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from ..repository.profile_transition import discover_profile_transition
from .execution_binding import ExecutionStateError, _guard_matches, _sha256_bytes, _sha256_file
from .execution_ledger import CheckpointLedger
from .execution_model import (
    AppliedSourceStep,
    AsyncAppliedSourceStep,
    CanonicalSourceComplete,
    CompletedSourceStep,
    ExecutionPhase,
    ReanalyzedSourceStep,
    RemovedTemporarySourceStep,
    SourceCompletionProof,
    VerifiedSourceStep,
)
from .planner import final_point_state_from_mapping
from .proposal import DeltaChangeKind, TargetDelta, TargetEntityKind, canonical_semantics


CompletionCallback = Callable[[str, str, str], SourceCompletionProof]


_ENTITY_COLLECTION = {
    TargetEntityKind.COMPONENT: "components",
    TargetEntityKind.ASSOCIATED_ARTIFACT: "associated_artifacts",
    TargetEntityKind.RELATIONSHIP: "relationships",
}


def _load_yaml(path: Path) -> dict[str, object]:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, yaml.YAMLError) as exc:
        raise ExecutionStateError(f"Unable to read target profile {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise ExecutionStateError("Target profile root must be a mapping.")
    return payload


def _validate_target_identity(payload: Mapping[str, object], verified: VerifiedSourceStep) -> None:
    ptsip = payload.get("ptsip")
    specification = ptsip.get("specification") if isinstance(ptsip, Mapping) else None
    version = ptsip.get("version") if isinstance(ptsip, Mapping) else None
    revision = specification.get("revision") if isinstance(specification, Mapping) else None
    final = verified.authorized.bound.plan.final_point
    if version != final.draft_version or revision != final.specification_revision:
        raise ExecutionStateError("Final Point payload does not match the WU-06 target draft/revision identity.")

    responsibility_map = payload.get("responsibility_map")
    if not isinstance(responsibility_map, Mapping) or responsibility_map.get("mode") != "explicit":
        raise ExecutionStateError(
            "WU-07 target apply currently supports explicit Responsibility Map targets only."
        )
    if any(key in responsibility_map for key in ("template", "overrides", "removals")):
        raise ExecutionStateError(
            "Explicit WU-07 target payload must not carry template/hybrid declaration-authority fields."
        )


def _validate_planned_seed_payload(payload: Mapping[str, object], verified: VerifiedSourceStep) -> None:
    snapshot = final_point_state_from_mapping(
        payload,
        path=verified.authorized.bound.plan.final_point.path,
    )
    if snapshot.entities:
        raise ExecutionStateError(
            "Planned Final Point seed must not contain target entities that were not accumulated by WU-06 accepted deltas."
        )


def _find_list_entity(rows: object, entity_id: str) -> tuple[list[object], int | None, object | None]:
    if rows is None:
        values: list[object] = []
    elif isinstance(rows, list):
        values = copy.deepcopy(rows)
    else:
        raise ExecutionStateError("Target entity collection must be a list.")
    found: list[tuple[int, object]] = []
    for index, item in enumerate(values):
        if isinstance(item, Mapping) and item.get("id") == entity_id:
            found.append((index, item))
    if len(found) > 1:
        raise ExecutionStateError(f"Target contains duplicate stable ID {entity_id!r}.")
    if not found:
        return values, None, None
    return values, found[0][0], canonical_semantics(found[0][1])


def _validate_list_delta_identity(delta: TargetDelta, value: object | None, *, label: str) -> None:
    if value is None:
        return
    if not isinstance(value, Mapping):
        raise ExecutionStateError(f"{delta.change_kind.value} delta {delta.id} requires a mapping {label}-state.")
    if value.get("id") != delta.entity_id:
        raise ExecutionStateError(
            f"{delta.change_kind.value} delta {delta.id} {label}-state stable ID does not match {delta.entity_id!r}."
        )


def _apply_delta(payload: dict[str, object], delta: TargetDelta) -> None:
    before = canonical_semantics(delta.before_value())
    after = canonical_semantics(delta.after_value())
    if delta.entity_kind in _ENTITY_COLLECTION:
        _validate_list_delta_identity(delta, before, label="before")
        _validate_list_delta_identity(delta, after, label="after")
        key = _ENTITY_COLLECTION[delta.entity_kind]
        rows, index, existing = _find_list_entity(payload.get(key), delta.entity_id)
        if delta.change_kind == DeltaChangeKind.ADD:
            if after is None:
                raise ExecutionStateError(f"ADD delta {delta.id} requires an after-state.")
            if existing is not None:
                if existing == after:
                    return
                raise ExecutionStateError(f"ADD delta {delta.id} conflicts with existing target entity.")
            rows.append(copy.deepcopy(dict(after)))
        elif delta.change_kind == DeltaChangeKind.REMOVE:
            if existing is None:
                return
            if before is None:
                raise ExecutionStateError(f"REMOVE delta {delta.id} requires an exact before-state.")
            if existing != before:
                raise ExecutionStateError(f"REMOVE delta {delta.id} before-state is stale.")
            assert index is not None
            rows.pop(index)
        else:
            if after is None:
                raise ExecutionStateError(f"REPLACE delta {delta.id} requires an after-state.")
            if existing == after:
                return
            if existing is None:
                if before is not None:
                    raise ExecutionStateError(f"REPLACE delta {delta.id} expected an existing entity.")
                rows.append(copy.deepcopy(dict(after)))
            else:
                if before is None:
                    raise ExecutionStateError(f"REPLACE delta {delta.id} requires an exact before-state for an existing entity.")
                if existing != before:
                    raise ExecutionStateError(f"REPLACE delta {delta.id} before-state is stale.")
                assert index is not None
                rows[index] = copy.deepcopy(dict(after))
        payload[key] = rows
        return

    key = (
        "component_dependency_policy"
        if delta.entity_kind == TargetEntityKind.COMPONENT_DEPENDENCY_POLICY
        else "policies"
    )
    if delta.entity_id != key:
        raise ExecutionStateError(
            f"Singleton delta {delta.id} stable ID must be {key!r}, got {delta.entity_id!r}."
        )
    existing = canonical_semantics(payload.get(key)) if key in payload else None
    if delta.change_kind == DeltaChangeKind.ADD:
        if after is None:
            raise ExecutionStateError(f"ADD delta {delta.id} requires an after-state.")
        if existing is not None and existing != after:
            raise ExecutionStateError(f"ADD delta {delta.id} conflicts with existing singleton target state.")
        if existing is None:
            payload[key] = copy.deepcopy(after)
    elif delta.change_kind == DeltaChangeKind.REMOVE:
        if existing is None:
            return
        if before is None:
            raise ExecutionStateError(f"REMOVE delta {delta.id} requires an exact before-state.")
        if existing != before:
            raise ExecutionStateError(f"REMOVE delta {delta.id} before-state is stale.")
        payload.pop(key, None)
    else:
        if after is None:
            raise ExecutionStateError(f"REPLACE delta {delta.id} requires an after-state.")
        if existing == after:
            return
        if existing is None:
            if before is not None:
                raise ExecutionStateError(f"REPLACE delta {delta.id} expected an existing singleton target state.")
        else:
            if before is None:
                raise ExecutionStateError(f"REPLACE delta {delta.id} requires an exact before-state for existing target state.")
            if existing != before:
                raise ExecutionStateError(f"REPLACE delta {delta.id} before-state is stale.")
        payload[key] = copy.deepcopy(after)


def _atomic_write_yaml(path: Path, payload: Mapping[str, object]) -> str:
    path.parent.mkdir(parents=True, exist_ok=True)
    text = yaml.safe_dump(dict(payload), sort_keys=False, allow_unicode=True)
    raw = text.encode("utf-8")
    temporary = path.with_name(f".{path.name}.ptsip-tmp-{os.getpid()}")
    try:
        with temporary.open("xb") as handle:
            handle.write(raw)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink(missing_ok=True)
    return _sha256_bytes(raw)


def _apply_delta_batch(
    path: Path,
    deltas: tuple[TargetDelta, ...],
    verified: VerifiedSourceStep,
    *,
    seed_payload: Mapping[str, object] | None = None,
) -> tuple[str, dict[str, object]]:
    planned_creation = not path.exists()
    if not planned_creation:
        payload = _load_yaml(path)
    else:
        if seed_payload is None:
            raise ExecutionStateError("Planned Final Point creation requires an explicit target-draft seed payload.")
        payload = copy.deepcopy(dict(seed_payload))
    _validate_target_identity(payload, verified)
    if planned_creation:
        _validate_planned_seed_payload(payload, verified)
    for delta in deltas:
        _apply_delta(payload, delta)
    final_point_state_from_mapping(payload, path=verified.authorized.bound.plan.final_point.path)
    sha = _atomic_write_yaml(path, payload)
    return sha, payload


def apply_required_deltas(
    repository_root: str | Path,
    verified: VerifiedSourceStep,
    ledger: CheckpointLedger,
    *,
    planned_final_point_seed: Mapping[str, object] | None = None,
) -> AppliedSourceStep:
    root = Path(repository_root).expanduser().resolve()
    if not _guard_matches(root, verified.authorized.bound):
        raise ExecutionStateError("Repository changed outside the controlled mutation set before apply.")
    final_path = profile_path_on_disk(root, verified.authorized.bound.plan.final_point.path)
    sha, _payload = _apply_delta_batch(
        final_path,
        verified.source.required_deltas,
        verified,
        seed_payload=planned_final_point_seed,
    )
    ledger.append(
        phase=ExecutionPhase.FINAL_POINT_APPLIED,
        source_path=verified.source.source_path,
        source_sha256=verified.source.source_content_sha256,
        final_point_before_sha256=verified.final_point_before_sha256,
        final_point_after_sha256=sha,
        analysis_digest=verified.source.analysis_digest,
        decision_ids=verified.source.decision_ids,
        payload={"applied_delta_ids": [item.id for item in verified.source.required_deltas]},
    )
    return AppliedSourceStep(verified, sha, tuple(item.id for item in verified.source.required_deltas))


def reanalyze_source(
    applied: AppliedSourceStep,
    completion_callback: CompletionCallback,
    ledger: CheckpointLedger,
) -> ReanalyzedSourceStep:
    proof = completion_callback(
        applied.verified.source.source_path,
        applied.final_point_after_sha256,
        applied.verified.source.analysis_digest,
    )
    if proof.source_path != applied.verified.source.source_path:
        raise ExecutionStateError("Post-apply completion proof is bound to another source.")
    if proof.analysis_digest != applied.verified.source.analysis_digest:
        raise ExecutionStateError("Post-apply completion proof analysis digest does not match the bound source analysis.")
    ledger.append(
        phase=ExecutionPhase.SOURCE_REANALYZED,
        source_path=proof.source_path,
        source_sha256=applied.verified.source.source_content_sha256,
        final_point_after_sha256=applied.final_point_after_sha256,
        analysis_digest=proof.analysis_digest,
        decision_ids=applied.verified.source.decision_ids,
        payload=proof.as_dict(),
    )
    return ReanalyzedSourceStep(applied, proof)


def complete_source(reanalyzed: ReanalyzedSourceStep, ledger: CheckpointLedger) -> CompletedSourceStep:
    if not reanalyzed.proof.complete:
        raise ExecutionStateError("Source Required Work Elements are not complete after apply.")
    completed = CompletedSourceStep(reanalyzed)
    source = reanalyzed.applied.verified.source
    ledger.append(
        phase=ExecutionPhase.SOURCE_COMPLETE,
        source_path=source.source_path,
        source_sha256=source.source_content_sha256,
        final_point_after_sha256=reanalyzed.applied.final_point_after_sha256,
        analysis_digest=reanalyzed.proof.analysis_digest,
        decision_ids=source.decision_ids,
        payload=reanalyzed.proof.as_dict(),
    )
    return completed


def _completed_verified(state: CompletedSourceStep | AsyncAppliedSourceStep) -> VerifiedSourceStep:
    if isinstance(state, AsyncAppliedSourceStep):
        return state.completed.reanalyzed.applied.verified
    return state.reanalyzed.applied.verified


def _current_final_sha(state: CompletedSourceStep | AsyncAppliedSourceStep) -> str:
    if isinstance(state, AsyncAppliedSourceStep):
        return state.final_point_after_sha256
    return state.reanalyzed.applied.final_point_after_sha256


def apply_optional_async_deltas(
    repository_root: str | Path,
    completed: CompletedSourceStep,
    ledger: CheckpointLedger,
) -> CompletedSourceStep | AsyncAppliedSourceStep:
    verified = completed.reanalyzed.applied.verified
    if not verified.source.async_deltas:
        return completed
    root = Path(repository_root).expanduser().resolve()
    if not _guard_matches(root, verified.authorized.bound):
        raise ExecutionStateError("Repository changed outside controlled paths before optional Async apply.")
    final_path = profile_path_on_disk(root, verified.authorized.bound.plan.final_point.path)
    if _sha256_file(final_path) != completed.reanalyzed.applied.final_point_after_sha256:
        raise ExecutionStateError("Final Point changed after Required completion and before Async apply.")
    sha, _payload = _apply_delta_batch(final_path, verified.source.async_deltas, verified)
    result = AsyncAppliedSourceStep(completed, sha, tuple(item.id for item in verified.source.async_deltas))
    ledger.append(
        phase=ExecutionPhase.ASYNC_APPLIED,
        source_path=verified.source.source_path,
        source_sha256=verified.source.source_content_sha256,
        final_point_before_sha256=completed.reanalyzed.applied.final_point_after_sha256,
        final_point_after_sha256=sha,
        analysis_digest=completed.reanalyzed.proof.analysis_digest,
        decision_ids=verified.source.decision_ids,
        payload={"applied_delta_ids": list(result.applied_delta_ids), "completion_contribution": False},
    )
    return result


def _validate_projected_final_state(root: Path, state: CompletedSourceStep | AsyncAppliedSourceStep) -> str:
    verified = _completed_verified(state)
    final_path = profile_path_on_disk(root, verified.authorized.bound.plan.final_point.path)
    current_sha = _current_final_sha(state)
    if not final_path.is_file() or _sha256_file(final_path) != current_final_sha:
        raise ExecutionStateError("Final Point changed after source completion.")
    payload = _load_yaml(final_path)
    snapshot = final_point_state_from_mapping(
        payload,
        path=verified.authorized.bound.plan.final_point.path,
        content_sha256=current_final_sha,
    )
    step_plan = verified.authorized.bound.plan.source_steps[verified.source_index]
    if snapshot.semantic_digest != step_plan.projected_final_state_digest:
        raise ExecutionStateError("Actual Final Point semantics do not match WU-06 projected state for this source.")
    return current_final_sha


def finalize_source(
    repository_root: str | Path,
    completed: CompletedSourceStep | AsyncAppliedSourceStep,
    ledger: CheckpointLedger,
) -> RemovedTemporarySourceStep | CanonicalSourceComplete:
    root = Path(repository_root).expanduser().resolve()
    verified = _completed_verified(completed)
    source = verified.source
    current_final_sha = _validate_projected_final_state(root, completed)
    source_path = profile_path_on_disk(root, source.source_path)
    if not source_path.is_file() or _sha256_file(source_path) != source.source_content_sha256:
        raise ExecutionStateError("Source changed before completion finalization.")
    if normalize_profile_path(source.source_path) == DEFAULT_PROFILE_PATH:
        result = CanonicalSourceComplete(completed)
        ledger.append(
            phase=ExecutionPhase.CANONICAL_SOURCE_COMPLETE,
            source_path=source.source_path,
            source_sha256=source.source_content_sha256,
            final_point_after_sha256=current_final_sha,
            analysis_digest=verified.source.analysis_digest,
            decision_ids=source.decision_ids,
            payload={"canonical_deleted": False},
        )
        return result

    source_path.unlink()
    if source_path.exists():
        raise ExecutionStateError("Temporary source deletion did not complete.")
    discovery = discover_profile_transition(root)
    if not discovery.valid:
        raise ExecutionStateError("Transition discovery became invalid after temporary source deletion.")
    result = RemovedTemporarySourceStep(completed, source.source_path)
    ledger.append(
        phase=ExecutionPhase.SOURCE_REMOVED,
        source_path=source.source_path,
        source_sha256=source.source_content_sha256,
        final_point_after_sha256=current_final_sha,
        analysis_digest=verified.source.analysis_digest,
        decision_ids=source.decision_ids,
        payload={"next_source_path": source.next_source_path},
    )
    return result
