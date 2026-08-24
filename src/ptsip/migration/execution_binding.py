from __future__ import annotations

import hashlib
import os
import subprocess
from pathlib import Path
from typing import Mapping, Protocol

from ..repository.profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from ..repository.snapshot import capture_snapshot, repository_files
from .execution_ledger import CheckpointLedger, LedgerIntegrityError
from .execution_model import (
    AuthorizationProof,
    AuthorizedExecutionPlan,
    BoundExecutionPlan,
    ExecutionPhase,
    MutationGuardExpectation,
    RecoveryInspection,
    RepositorySnapshotExpectation,
    SourceExecutionBinding,
    VerifiedSourceStep,
)
from .model import MigrationAnalysis
from .planner import FinalPointConvergencePlan, FinalPointKind
from .proposal import SourceProposalSet, TargetDelta, semantic_digest


class ExecutionStateError(RuntimeError):
    pass


class AuthorityHeadStore(Protocol):
    def ensure_head(self) -> str: ...


def _sha256_bytes(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def _sha256_file(path: Path) -> str:
    return _sha256_bytes(path.read_bytes())


def _git(root: Path, *args: str) -> tuple[int, bytes]:
    process = subprocess.run(
        ["git", "-C", str(root), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    return process.returncode, process.stdout


def _controlled_pathspecs(paths: tuple[str, ...]) -> list[str]:
    return [f":(exclude){normalize_profile_path(item)}" for item in sorted(set(paths))]


def _fingerprint_paths(root: Path, paths: list[str]) -> str:
    digest = hashlib.sha256()
    for relative in sorted(paths):
        digest.update(relative.encode("utf-8", errors="surrogateescape"))
        digest.update(b"\0")
        path = root / relative
        if path.is_symlink():
            digest.update(b"symlink\0")
            digest.update(os.readlink(path).encode("utf-8", errors="surrogateescape"))
            digest.update(b"\0")
            continue
        try:
            digest.update(path.read_bytes())
        except OSError as exc:
            raise ExecutionStateError(f"Unable to fingerprint guarded path {relative}: {exc}") from exc
        digest.update(b"\0")
    return digest.hexdigest()


def capture_mutation_guard(repository_root: str | Path, controlled_paths: tuple[str, ...]) -> MutationGuardExpectation:
    root = Path(repository_root).expanduser().resolve()
    controlled = tuple(normalize_profile_path(item) for item in controlled_paths)
    code, _ = _git(root, "rev-parse", "--is-inside-work-tree")
    if code == 0:
        head_code, head_raw = _git(root, "rev-parse", "HEAD")
        if head_code != 0:
            raise ExecutionStateError("Unable to read git HEAD for mutation guard.")
        pathspecs = _controlled_pathspecs(controlled)
        status_code, status = _git(
            root,
            "status",
            "--porcelain=v1",
            "-z",
            "--untracked-files=all",
            "--",
            ".",
            *pathspecs,
        )
        if status_code != 0:
            raise ExecutionStateError("Unable to capture scoped git status for mutation guard.")
        tracked_code, tracked = _git(root, "ls-files", "-z", "--", ".", *pathspecs)
        if tracked_code != 0:
            raise ExecutionStateError("Unable to capture scoped tracked files for mutation guard.")
        paths = [item.decode("utf-8", errors="surrogateescape") for item in tracked.split(b"\0") if item]
        return MutationGuardExpectation(
            head=head_raw.decode("utf-8", errors="replace").strip() or None,
            status_fingerprint=hashlib.sha256(status).hexdigest(),
            content_fingerprint=_fingerprint_paths(root, paths),
        )

    _mode, paths, errors = repository_files(root)
    if errors:
        raise ExecutionStateError("Unable to capture filesystem mutation guard: " + "; ".join(errors))
    controlled_set = set(controlled)
    selected = [item for item in paths if normalize_profile_path(item) not in controlled_set]
    return MutationGuardExpectation(
        head=None,
        status_fingerprint=hashlib.sha256(b"<non-git>").hexdigest(),
        content_fingerprint=_fingerprint_paths(root, selected),
    )


def _analysis_snapshot(analysis: MigrationAnalysis) -> RepositorySnapshotExpectation:
    return RepositorySnapshotExpectation(
        analysis.repository_head,
        analysis.repository_status_fingerprint,
        analysis.repository_content_fingerprint,
    )


def _current_snapshot_matches_analysis(analysis: MigrationAnalysis, snapshot) -> bool:
    return (
        analysis.repository_head == snapshot.head
        and analysis.repository_status_fingerprint == snapshot.status_fingerprint
        and analysis.repository_content_fingerprint == snapshot.tracked_content_fingerprint
        and not snapshot.observation_errors
    )


def _generation_matches(left, right) -> bool:
    return (
        left.profile_path == right.profile_path
        and left.declared_version == right.declared_version
        and left.specification_revision == right.specification_revision
        and left.specification_source == right.specification_source
        and left.content_sha256 == right.content_sha256
        and left.temporary == right.temporary
    )


def bind_execution_plan(
    repository_root: str | Path,
    plan: FinalPointConvergencePlan,
    analyses_by_source: Mapping[str, MigrationAnalysis],
    proposals_by_source: Mapping[str, SourceProposalSet],
) -> BoundExecutionPlan:
    if plan.issues:
        raise ExecutionStateError("WU-06 convergence plan contains planning issues.")
    if not plan.preview.ready_for_wu07 or plan.preview.blocking_ids:
        raise ExecutionStateError("WU-06 execution preview is not ready for WU-07.")

    ordered = tuple(item.source_path for item in plan.source_steps)
    if ordered != plan.preview.ordered_sources:
        raise ExecutionStateError("WU-06 source order and execution preview disagree.")

    root = Path(repository_root).expanduser().resolve()
    initial_snapshot = capture_snapshot(root)
    if initial_snapshot.observation_errors:
        raise ExecutionStateError("Repository snapshot is incomplete at WU-07 bind boundary.")

    bindings: list[SourceExecutionBinding] = []
    for step in plan.source_steps:
        analysis = analyses_by_source.get(step.source_path)
        proposal_set = proposals_by_source.get(step.source_path)
        if analysis is None or proposal_set is None:
            raise ExecutionStateError(f"Missing WU-05/WU-06 input for source {step.source_path}.")
        if not analysis.valid or analysis.deterministic_digest != step.analysis_digest:
            raise ExecutionStateError(f"Source analysis for {step.source_path} is invalid or digest-stale.")
        if analysis.source_generation.profile_path != step.source_path:
            raise ExecutionStateError(f"Source analysis path mismatch for {step.source_path}.")
        if analysis.source_generation.content_sha256 != step.source_content_sha256:
            raise ExecutionStateError(f"Source content identity mismatch for {step.source_path}.")
        if not _current_snapshot_matches_analysis(analysis, initial_snapshot):
            raise ExecutionStateError(
                f"Repository changed after accepted analysis for {step.source_path}; WU-07 bind is fail-closed."
            )
        if proposal_set.analysis_digest != step.analysis_digest:
            raise ExecutionStateError(f"Proposal analysis binding mismatch for {step.source_path}.")
        if not _generation_matches(proposal_set.source_generation, analysis.source_generation):
            raise ExecutionStateError(f"Proposal source generation mismatch for {step.source_path}.")
        for bundle in proposal_set.accepted:
            if bundle.analysis_digest != step.analysis_digest or not _generation_matches(
                bundle.source_generation, analysis.source_generation
            ):
                raise ExecutionStateError(
                    f"Accepted bundle {bundle.proposal_id} lost its exact source/analysis binding."
                )
        accepted_ids = tuple(sorted(item.proposal_id for item in proposal_set.accepted))
        if accepted_ids != tuple(sorted(step.accepted_bundle_ids)):
            raise ExecutionStateError(f"Accepted bundle set changed after WU-06 planning for {step.source_path}.")
        if proposal_set.blocking_proposal_ids or proposal_set.blocking_unresolved_ids or proposal_set.issues:
            raise ExecutionStateError(f"Source {step.source_path} still contains blocking proposal state.")

        delta_owner: dict[str, tuple[TargetDelta, bool]] = {}
        for bundle in proposal_set.accepted:
            for delta in bundle.deltas:
                owner = delta_owner.get(delta.id)
                current = (delta, bundle.async_only)
                if owner is not None and owner[1] != current[1]:
                    raise ExecutionStateError(f"Delta {delta.id} is both required and async in accepted bundles.")
                delta_owner[delta.id] = current
        execution_ids = tuple(step.execution_delta_ids)
        missing = [item for item in execution_ids if item not in delta_owner]
        if missing:
            raise ExecutionStateError(
                f"WU-06 execution delta(s) no longer resolve to accepted bundles: {', '.join(missing)}"
            )
        required = tuple(delta_owner[item][0] for item in execution_ids if not delta_owner[item][1])
        async_deltas = tuple(delta_owner[item][0] for item in execution_ids if delta_owner[item][1])
        bindings.append(
            SourceExecutionBinding(
                source_path=step.source_path,
                source_content_sha256=step.source_content_sha256,
                analysis_digest=step.analysis_digest,
                snapshot=_analysis_snapshot(analysis),
                accepted_bundles=tuple(sorted(proposal_set.accepted, key=lambda item: item.proposal_id)),
                required_deltas=required,
                async_deltas=async_deltas,
                next_source_path=step.next_source_path,
            )
        )

    controlled = tuple(sorted(set(ordered + (plan.final_point.path,))))
    guard = capture_mutation_guard(root, controlled)
    return BoundExecutionPlan(plan, plan.deterministic_digest, tuple(bindings), guard)


def build_authorization_proof(
    bound: BoundExecutionPlan,
    *,
    decision_ids: tuple[str, ...],
    authority_revision: str,
) -> AuthorizationProof:
    decisions = tuple(sorted(set(item.strip() for item in decision_ids if item.strip())))
    required = tuple(sorted({item for source in bound.sources for item in source.decision_ids}))
    if decisions != required:
        raise ExecutionStateError("Authorization decision set must exactly match accepted WU-06 decision identities.")
    revision = authority_revision.strip()
    if not revision:
        raise ExecutionStateError("Authorization proof requires a non-empty authority revision.")
    proof_id = "authorization:" + semantic_digest(
        {"plan_digest": bound.plan_digest, "decision_ids": decisions, "authority_revision": revision}
    )[:24]
    return AuthorizationProof(bound.plan_digest, decisions, revision, proof_id)


def authorize_execution(
    bound: BoundExecutionPlan,
    proof: AuthorizationProof,
    ledger: CheckpointLedger,
    *,
    authority_store: AuthorityHeadStore | None = None,
) -> AuthorizedExecutionPlan:
    if proof.plan_digest != bound.plan_digest:
        raise ExecutionStateError("Authorization proof is bound to a different WU-06 plan.")
    required = tuple(sorted({item for source in bound.sources for item in source.decision_ids}))
    if tuple(sorted(proof.decision_ids)) != required:
        raise ExecutionStateError("Authorization proof no longer covers the exact accepted decision set.")
    if authority_store is not None and authority_store.ensure_head() != proof.authority_revision:
        raise ExecutionStateError("Coordinated authority revision changed after authorization.")
    if ledger.plan_digest != bound.plan_digest:
        raise ExecutionStateError("Checkpoint ledger belongs to a different WU-06 plan.")
    latest = ledger.latest()
    if latest is not None:
        raise ExecutionStateError("Execution ledger is not empty; use recovery inspection before resuming.")
    ledger.append(phase=ExecutionPhase.PLAN_BOUND, payload=bound.as_dict())
    ledger.append(
        phase=ExecutionPhase.AUTHORIZED,
        decision_ids=proof.decision_ids,
        payload=proof.as_dict(),
    )
    return AuthorizedExecutionPlan(bound, proof)


def _guard_matches(root: Path, bound: BoundExecutionPlan) -> bool:
    controlled = tuple(sorted(set(tuple(item.source_path for item in bound.sources) + (bound.plan.final_point.path,))))
    return capture_mutation_guard(root, controlled) == bound.mutation_guard


def _expected_final_point_sha(authorized: AuthorizedExecutionPlan, ledger: CheckpointLedger) -> str | None:
    rows = ledger.read_all()
    for row in reversed(rows):
        if row.final_point_after_sha256:
            return row.final_point_after_sha256
    return authorized.bound.plan.final_point.content_sha256


def verify_source_preconditions(
    repository_root: str | Path,
    authorized: AuthorizedExecutionPlan,
    source_index: int,
    ledger: CheckpointLedger,
) -> VerifiedSourceStep:
    root = Path(repository_root).expanduser().resolve()
    if not 0 <= source_index < len(authorized.bound.sources):
        raise ExecutionStateError("Source index is outside the bound execution plan.")
    source = authorized.bound.sources[source_index]
    removed_count = sum(1 for row in ledger.read_all() if row.phase == ExecutionPhase.SOURCE_REMOVED)
    if source_index != removed_count:
        raise ExecutionStateError("Requested source is not the next source allowed by the checkpoint ledger.")
    if not _guard_matches(root, authorized.bound):
        raise ExecutionStateError("Repository changed outside the WU-07 controlled profile mutation set.")

    source_path = profile_path_on_disk(root, source.source_path)
    if not source_path.is_file() or _sha256_file(source_path) != source.source_content_sha256:
        raise ExecutionStateError("Source profile changed or disappeared after WU-06 planning.")

    final_path = profile_path_on_disk(root, authorized.bound.plan.final_point.path)
    expected_final_sha = _expected_final_point_sha(authorized, ledger)
    if expected_final_sha is None:
        if authorized.bound.plan.final_point.kind == FinalPointKind.EXISTING:
            raise ExecutionStateError("Existing Final Point lacks an expected content SHA.")
        if final_path.exists():
            raise ExecutionStateError("Planned Final Point unexpectedly exists before its guarded creation.")
    else:
        if not final_path.is_file() or _sha256_file(final_path) != expected_final_sha:
            raise ExecutionStateError("Final Point content changed from the exact expected checkpoint state.")

    snapshot = RepositorySnapshotExpectation.from_snapshot(capture_snapshot(root))
    ledger.append(
        phase=ExecutionPhase.PRECONDITIONS_VERIFIED,
        source_path=source.source_path,
        source_sha256=source.source_content_sha256,
        final_point_before_sha256=expected_final_sha,
        analysis_digest=source.analysis_digest,
        decision_ids=source.decision_ids,
        repository_snapshot=snapshot,
        payload={"source_index": source_index},
    )
    return VerifiedSourceStep(authorized, source_index, source, snapshot, expected_final_sha)


def inspect_recovery(
    repository_root: str | Path,
    bound: BoundExecutionPlan,
    ledger: CheckpointLedger,
) -> RecoveryInspection:
    root = Path(repository_root).expanduser().resolve()
    try:
        rows = ledger.read_all()
    except LedgerIntegrityError as exc:
        return RecoveryInspection(bound.plan_digest, None, False, 0, (str(exc),))
    if not rows:
        return RecoveryInspection(bound.plan_digest, None, True, 0, ())
    latest = rows[-1]
    reasons: list[str] = []
    if latest.plan_digest != bound.plan_digest:
        reasons.append("ledger plan digest differs from bound WU-06 plan")
    try:
        if not _guard_matches(root, bound):
            reasons.append("repository changed outside controlled profile paths")
    except ExecutionStateError as exc:
        reasons.append(str(exc))

    final_path = profile_path_on_disk(root, bound.plan.final_point.path)
    if latest.phase in {ExecutionPhase.PROMOTED, ExecutionPhase.POST_PROMOTION_VERIFIED}:
        canonical = profile_path_on_disk(root, DEFAULT_PROFILE_PATH)
        if latest.final_point_after_sha256 and (
            not canonical.is_file() or _sha256_file(canonical) != latest.final_point_after_sha256
        ):
            reasons.append("canonical content does not match promoted checkpoint")
        if final_path.exists():
            reasons.append("Final Point unexpectedly exists after promoted checkpoint")
    else:
        expected_final_sha: str | None = None
        for row in reversed(rows):
            if row.final_point_after_sha256:
                expected_final_sha = row.final_point_after_sha256
                break
        if expected_final_sha is None:
            expected_final_sha = bound.plan.final_point.content_sha256
        if expected_final_sha is None:
            if final_path.exists():
                reasons.append("planned Final Point exists without a persisted mutation checkpoint")
        elif not final_path.is_file() or _sha256_file(final_path) != expected_final_sha:
            reasons.append("Final Point content does not match latest persisted checkpoint state")

    removed_sources = [row.source_path for row in rows if row.phase == ExecutionPhase.SOURCE_REMOVED and row.source_path]
    for source_path in removed_sources:
        if profile_path_on_disk(root, source_path).exists():
            reasons.append(f"source reappeared after removal checkpoint: {source_path}")
    next_index = len(removed_sources)
    if latest.phase in {
        ExecutionPhase.CANONICAL_SOURCE_COMPLETE,
        ExecutionPhase.GLOBAL_VALIDATION,
        ExecutionPhase.PROMOTION_READY,
        ExecutionPhase.PROMOTED,
        ExecutionPhase.POST_PROMOTION_VERIFIED,
    }:
        next_index = len(bound.sources)
    return RecoveryInspection(
        bound.plan_digest,
        latest.phase,
        not reasons and latest.phase != ExecutionPhase.RECOVERY_REQUIRED,
        next_index,
        tuple(reasons),
    )
