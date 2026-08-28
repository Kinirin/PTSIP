from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Iterable, Mapping

from ..repository.profile_transition import DraftVersion, ProfileTransitionState
from .model import ArchitectureFindingKind, MigrationAnalysis
from .proposal import (
    AcceptedDeltaBundle,
    DeltaChangeKind,
    ProposalBundle,
    ProposalPurpose,
    SourceProposalSet,
    TargetDelta,
    TargetEntityKind,
    UnresolvedBundle,
    canonical_semantics,
    semantic_digest,
)


class FinalPointKind(StrEnum):
    EXISTING = "EXISTING"
    PLANNED = "PLANNED"


class ReconciliationStatus(StrEnum):
    NO_CHANGE_REQUIRED = "NO_CHANGE_REQUIRED"
    ADD_TARGET_DECLARATION = "ADD_TARGET_DECLARATION"
    REPLACE_WITH_EXPLICIT_OWNER_DECISION = "REPLACE_WITH_EXPLICIT_OWNER_DECISION"
    CONFLICT_REQUIRES_CONFIRMATION = "CONFLICT_REQUIRES_CONFIRMATION"
    UNRESOLVED = "UNRESOLVED"


class DeletionGate(StrEnum):
    ALREADY_ELIGIBLE = "ALREADY_ELIGIBLE"
    REQUIRES_POST_APPLY_VERIFICATION = "REQUIRES_POST_APPLY_VERIFICATION"
    BLOCKED = "BLOCKED"


@dataclass(frozen=True)
class FinalPointReference:
    kind: FinalPointKind
    path: str
    draft_version: str
    specification_revision: str
    content_sha256: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "draft_version": self.draft_version,
            "specification_revision": self.specification_revision,
            "content_sha256": self.content_sha256,
        }


@dataclass(frozen=True)
class FinalPointEntity:
    kind: TargetEntityKind
    id: str
    payload: object

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "id": self.id,
            "payload": canonical_semantics(self.payload),
        }


@dataclass(frozen=True)
class FinalPointStateSnapshot:
    path: str
    draft_version: str
    specification_revision: str
    content_sha256: str | None
    entities: tuple[FinalPointEntity, ...]

    @property
    def semantic_digest(self) -> str:
        return semantic_digest(
            {
                "draft_version": self.draft_version,
                "specification_revision": self.specification_revision,
                "entities": [
                    item.as_dict()
                    for item in sorted(self.entities, key=lambda item: (item.kind.value, item.id))
                ],
            }
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "path": self.path,
            "draft_version": self.draft_version,
            "specification_revision": self.specification_revision,
            "content_sha256": self.content_sha256,
            "entities": [
                item.as_dict()
                for item in sorted(self.entities, key=lambda item: (item.kind.value, item.id))
            ],
            "semantic_digest": self.semantic_digest,
        }


@dataclass(frozen=True)
class ReconciliationResult:
    bundle_id: str
    delta_id: str
    status: ReconciliationStatus
    rationale: str

    def as_dict(self) -> dict[str, str]:
        return {
            "bundle_id": self.bundle_id,
            "delta_id": self.delta_id,
            "status": self.status.value,
            "rationale": self.rationale,
        }


@dataclass(frozen=True)
class PlanningIssue:
    code: str
    message: str
    source_path: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "code": self.code,
            "message": self.message,
            "source_path": self.source_path,
        }


@dataclass(frozen=True)
class SourceConvergencePlan:
    source_path: str
    source_content_sha256: str
    analysis_digest: str
    required_total: int
    required_unresolved_before: int
    accepted_bundle_ids: tuple[str, ...]
    suggested_bundle_ids: tuple[str, ...]
    unresolved_bundle_ids: tuple[str, ...]
    unplanned_required_ids: tuple[str, ...]
    execution_delta_ids: tuple[str, ...]
    reconciliations: tuple[ReconciliationResult, ...]
    deletion_gate: DeletionGate
    next_source_path: str | None
    projected_final_state_digest: str

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "source_content_sha256": self.source_content_sha256,
            "analysis_digest": self.analysis_digest,
            "required_total": self.required_total,
            "required_unresolved_before": self.required_unresolved_before,
            "accepted_bundle_ids": list(self.accepted_bundle_ids),
            "suggested_bundle_ids": list(self.suggested_bundle_ids),
            "unresolved_bundle_ids": list(self.unresolved_bundle_ids),
            "unplanned_required_ids": list(self.unplanned_required_ids),
            "execution_delta_ids": list(self.execution_delta_ids),
            "reconciliations": [item.as_dict() for item in self.reconciliations],
            "deletion_gate": self.deletion_gate.value,
            "next_source_path": self.next_source_path,
            "projected_final_state_digest": self.projected_final_state_digest,
        }


@dataclass(frozen=True)
class ExecutionPreview:
    final_point: FinalPointReference
    ordered_sources: tuple[str, ...]
    source_steps: tuple[SourceConvergencePlan, ...]
    blocking_ids: tuple[str, ...]
    ready_for_wu07: bool
    promotion_gate: str = "WU07_GLOBAL_VALIDATION_REQUIRED"

    def as_dict(self) -> dict[str, object]:
        return {
            "final_point": self.final_point.as_dict(),
            "ordered_sources": list(self.ordered_sources),
            "source_steps": [item.as_dict() for item in self.source_steps],
            "blocking_ids": list(self.blocking_ids),
            "ready_for_wu07": self.ready_for_wu07,
            "promotion_gate": self.promotion_gate,
        }


@dataclass(frozen=True)
class FinalPointConvergencePlan:
    final_point: FinalPointReference
    source_steps: tuple[SourceConvergencePlan, ...]
    issues: tuple[PlanningIssue, ...]
    preview: ExecutionPreview
    projected_final_state_digest: str

    @property
    def deterministic_digest(self) -> str:
        return semantic_digest(
            {
                "final_point": self.final_point.as_dict(),
                "source_steps": [item.as_dict() for item in self.source_steps],
                "issues": [item.as_dict() for item in self.issues],
                "preview": self.preview.as_dict(),
                "projected_final_state_digest": self.projected_final_state_digest,
            }
        )

    def as_dict(self) -> dict[str, object]:
        payload = {
            "final_point": self.final_point.as_dict(),
            "source_steps": [item.as_dict() for item in self.source_steps],
            "issues": [item.as_dict() for item in self.issues],
            "preview": self.preview.as_dict(),
            "projected_final_state_digest": self.projected_final_state_digest,
        }
        payload["deterministic_digest"] = self.deterministic_digest
        return payload


def final_point_state_from_mapping(
    payload: Mapping[str, object],
    *,
    path: str,
    content_sha256: str | None = None,
) -> FinalPointStateSnapshot:
    ptsip = payload.get("ptsip")
    specification = ptsip.get("specification") if isinstance(ptsip, Mapping) else None
    version = ptsip.get("version") if isinstance(ptsip, Mapping) else None
    revision = specification.get("revision") if isinstance(specification, Mapping) else None
    if not isinstance(version, str) or not isinstance(revision, str) or not revision:
        raise ValueError("Final Point state requires draft version and immutable specification revision.")

    rows: list[FinalPointEntity] = []
    for key, kind in (
        ("components", TargetEntityKind.COMPONENT),
        ("associated_artifacts", TargetEntityKind.ASSOCIATED_ARTIFACT),
        ("relationships", TargetEntityKind.RELATIONSHIP),
    ):
        raw = payload.get(key, [])
        if not isinstance(raw, list):
            raise ValueError(f"{key} must be a list.")
        for item in raw:
            if not isinstance(item, Mapping) or not isinstance(item.get("id"), str) or not item.get("id"):
                raise ValueError(f"{key} entries require stable string ids.")
            rows.append(FinalPointEntity(kind, str(item["id"]), canonical_semantics(dict(item))))

    if "component_dependency_policy" in payload:
        rows.append(
            FinalPointEntity(
                TargetEntityKind.COMPONENT_DEPENDENCY_POLICY,
                "component_dependency_policy",
                canonical_semantics(payload["component_dependency_policy"]),
            )
        )
    if "policies" in payload:
        rows.append(
            FinalPointEntity(
                TargetEntityKind.POLICIES,
                "policies",
                canonical_semantics(payload["policies"]),
            )
        )

    keys = [(item.kind.value, item.id) for item in rows]
    if len(keys) != len(set(keys)):
        raise ValueError("Final Point contains duplicate stable entity identities.")
    return FinalPointStateSnapshot(
        path,
        version,
        revision,
        content_sha256,
        tuple(sorted(rows, key=lambda item: (item.kind.value, item.id))),
    )


def _find_entity(state: FinalPointStateSnapshot | None, delta: TargetDelta) -> object | None:
    if state is None:
        return None
    for item in state.entities:
        if item.kind == delta.entity_kind and item.id == delta.entity_id:
            return canonical_semantics(item.payload)
    return None


def reconcile_delta(
    delta: TargetDelta,
    state: FinalPointStateSnapshot | None,
    *,
    accepted: bool,
    bundle_id: str,
) -> ReconciliationResult:
    existing = _find_entity(state, delta)
    before = canonical_semantics(delta.before_value())
    after = canonical_semantics(delta.after_value())

    if delta.change_kind == DeltaChangeKind.ADD:
        if existing is None:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.ADD_TARGET_DECLARATION,
                "Target entity is absent and the delta adds it.",
            )
        if existing == after:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.NO_CHANGE_REQUIRED,
                "Final Point already contains the proposed semantic state.",
            )
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
            "Final Point already contains a different entity with the same stable identity.",
        )

    if delta.change_kind == DeltaChangeKind.REMOVE:
        if existing is None:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.NO_CHANGE_REQUIRED,
                "Target entity is already absent.",
            )
        if before is not None and existing != before:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
                "Final Point entity does not match the state the removal was reviewed against.",
            )
        status = (
            ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION
            if accepted
            else ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION
        )
        return ReconciliationResult(
            bundle_id,
            delta.id,
            status,
            "Removal changes accepted target architecture and therefore requires an explicit project-owned decision.",
        )

    if existing == after:
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.NO_CHANGE_REQUIRED,
            "Final Point already contains the proposed replacement state.",
        )
    if existing is None:
        if before is None:
            return ReconciliationResult(
                bundle_id,
                delta.id,
                ReconciliationStatus.ADD_TARGET_DECLARATION,
                "Replacement has no required prior state and the target entity is absent.",
            )
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
            "Expected entity to replace is absent from the Final Point.",
        )
    if before is not None and existing != before:
        return ReconciliationResult(
            bundle_id,
            delta.id,
            ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
            "Final Point entity changed from the exact state the replacement was reviewed against.",
        )
    status = (
        ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION
        if accepted
        else ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION
    )
    return ReconciliationResult(
        bundle_id,
        delta.id,
        status,
        "Replacement changes target architecture and is executable only when bound to an explicit project-owned decision.",
    )


def _state_with_delta(
    state: FinalPointStateSnapshot,
    delta: TargetDelta,
    reconciliation: ReconciliationResult,
) -> FinalPointStateSnapshot:
    if reconciliation.status not in {
        ReconciliationStatus.ADD_TARGET_DECLARATION,
        ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION,
        ReconciliationStatus.NO_CHANGE_REQUIRED,
    }:
        return state
    if reconciliation.status == ReconciliationStatus.NO_CHANGE_REQUIRED:
        return state

    entities = {(item.kind, item.id): item for item in state.entities}
    key = (delta.entity_kind, delta.entity_id)
    if delta.change_kind == DeltaChangeKind.REMOVE:
        entities.pop(key, None)
    else:
        after = delta.after_value()
        if after is None:
            return state
        entities[key] = FinalPointEntity(delta.entity_kind, delta.entity_id, canonical_semantics(after))
    return FinalPointStateSnapshot(
        state.path,
        state.draft_version,
        state.specification_revision,
        None,
        tuple(sorted(entities.values(), key=lambda item: (item.kind.value, item.id))),
    )


def _bundle_trace_matches(source, analysis_digest: str, bundle) -> bool:
    binding = bundle.source_generation
    return (
        binding.profile_path == source.profile_path
        and binding.declared_version == source.declared_version
        and binding.specification_revision == source.specification_revision
        and binding.content_sha256 == source.content_sha256
        and bundle.analysis_digest == analysis_digest
    )


def derive_source_proposals(
    analysis: MigrationAnalysis,
    *,
    proposed: Iterable[ProposalBundle] = (),
    accepted: Iterable[AcceptedDeltaBundle] = (),
    requested_async_ids: Iterable[str] = (),
) -> SourceProposalSet:
    issues: list[str] = []
    if not analysis.valid:
        issues.append("Migration analysis is not valid; proposal derivation is fail-closed.")

    good_proposals: list[ProposalBundle] = []
    for bundle in sorted(proposed, key=lambda item: item.id):
        if _bundle_trace_matches(analysis.source_generation, analysis.deterministic_digest, bundle):
            good_proposals.append(bundle)
        else:
            issues.append(f"Proposal {bundle.id} is not bound to this source analysis.")

    good_accepted: list[AcceptedDeltaBundle] = []
    for bundle in sorted(accepted, key=lambda item: item.proposal_id):
        if _bundle_trace_matches(analysis.source_generation, analysis.deterministic_digest, bundle):
            good_accepted.append(bundle)
        else:
            issues.append(f"Accepted bundle {bundle.proposal_id} is not bound to this source analysis.")

    accepted_ids = {item.proposal_id for item in good_accepted}
    good_proposals = [item for item in good_proposals if item.id not in accepted_ids]

    covered_subjects: set[str] = set()
    for bundle in [*good_proposals, *good_accepted]:
        for delta in bundle.deltas:
            covered_subjects.update(delta.obligation_ids)

    no_change: list[str] = []
    unresolved: list[UnresolvedBundle] = []
    for item in analysis.required:
        if item.resolved:
            no_change.append(item.id)
            continue
        if item.id in covered_subjects:
            continue
        unresolved.append(
            UnresolvedBundle.build(
                source_generation=analysis.source_generation,
                analysis_digest=analysis.deterministic_digest,
                subject_ids=(item.id,),
                purpose=(ProposalPurpose.REQUIRED_MIGRATION,),
                question=(
                    f"Required obligation {item.id} needs an explicit target delta or a project-owned "
                    f"no-change resolution; target status is {item.target_status.value}."
                ),
            )
        )

    for finding in analysis.architecture_findings:
        if finding.subject_id in covered_subjects:
            continue
        if finding.kind in {
            ArchitectureFindingKind.MISSING_RELATIONSHIP,
            ArchitectureFindingKind.MISSING_ASSOCIATED_ARTIFACT,
        }:
            unresolved.append(
                UnresolvedBundle.build(
                    source_generation=analysis.source_generation,
                    analysis_digest=analysis.deterministic_digest,
                    subject_ids=(finding.subject_id,),
                    purpose=(ProposalPurpose.TARGET_VALIDITY,),
                    question=(
                        f"{finding.kind.value} requires an explicit target proposal before the Final Point "
                        "can claim semantic convergence."
                    ),
                )
            )
        elif finding.kind in {
            ArchitectureFindingKind.EVIDENCE_CONFLICT,
            ArchitectureFindingKind.EVIDENCE_INCOMPLETE,
        }:
            unresolved.append(
                UnresolvedBundle.build(
                    source_generation=analysis.source_generation,
                    analysis_digest=analysis.deterministic_digest,
                    subject_ids=(finding.subject_id,),
                    purpose=(ProposalPurpose.ADVISORY,),
                    question=(
                        f"{finding.kind.value} remains reviewable evidence context and is not architecture authority."
                    ),
                )
            )

    requested = set(requested_async_ids)
    known_async = {item.id for item in analysis.async_targets}
    for item in sorted(requested - known_async):
        issues.append(f"Requested Async target {item} is not present in this source analysis.")
    for item in sorted(requested & known_async):
        if item in covered_subjects:
            continue
        unresolved.append(
            UnresolvedBundle.build(
                source_generation=analysis.source_generation,
                analysis_digest=analysis.deterministic_digest,
                subject_ids=(item,),
                purpose=(ProposalPurpose.ASYNC_OPTIONAL,),
                question=f"Async target {item} was requested but has no explicit target delta yet.",
            )
        )

    ignored_async = tuple(sorted(known_async - requested))
    return SourceProposalSet(
        source_generation=analysis.source_generation,
        analysis_digest=analysis.deterministic_digest,
        suggested=tuple(good_proposals),
        accepted=tuple(good_accepted),
        unresolved=tuple(sorted(unresolved, key=lambda item: item.id)),
        no_change_obligation_ids=tuple(sorted(no_change)),
        ignored_async_ids=ignored_async,
        issues=tuple(sorted(set(issues))),
    )


def _generation_matches(identity, binding) -> bool:
    return (
        identity.path == binding.profile_path
        and identity.declared_version == binding.declared_version
        and identity.specification_revision == binding.specification_revision
        and identity.content_sha256 == binding.content_sha256
    )


def _final_point_reference(
    transition: ProfileTransitionState,
    target_draft_version: str,
    target_specification_revision: str,
) -> tuple[FinalPointReference, list[PlanningIssue]]:
    issues: list[PlanningIssue] = []
    parsed = DraftVersion.from_draft_label(target_draft_version)
    if parsed is None:
        return (
            FinalPointReference(
                FinalPointKind.PLANNED,
                "<invalid>",
                target_draft_version,
                target_specification_revision,
                None,
            ),
            [PlanningIssue("INVALID_TARGET_DRAFT", "Target draft must be a <major>.<minor>.<micro>-draft label.")],
        )

    if transition.final_point is None:
        return (
            FinalPointReference(
                FinalPointKind.PLANNED,
                f"ptsip_{parsed.semantic}.yaml",
                target_draft_version,
                target_specification_revision,
                None,
            ),
            issues,
        )

    final_point = transition.final_point
    if final_point.declared_version != target_draft_version:
        issues.append(
            PlanningIssue(
                "FINAL_POINT_DRAFT_MISMATCH",
                f"Existing Final Point {final_point.declared_version!r} does not match requested target {target_draft_version!r}.",
                final_point.path,
            )
        )
    if final_point.specification_revision != target_specification_revision:
        issues.append(
            PlanningIssue(
                "FINAL_POINT_REVISION_MISMATCH",
                "Existing Final Point revision does not match requested target revision.",
                final_point.path,
            )
        )
    return (
        FinalPointReference(
            FinalPointKind.EXISTING,
            final_point.path,
            final_point.declared_version,
            final_point.specification_revision,
            final_point.content_sha256,
        ),
        issues,
    )


def build_final_point_convergence_plan(
    transition: ProfileTransitionState,
    analyses: Iterable[MigrationAnalysis],
    proposal_sets: Iterable[SourceProposalSet],
    *,
    target_draft_version: str,
    target_specification_revision: str,
    final_point_state: FinalPointStateSnapshot | None = None,
) -> FinalPointConvergencePlan:
    final_ref, issues = _final_point_reference(
        transition,
        target_draft_version,
        target_specification_revision,
    )

    if final_ref.kind == FinalPointKind.EXISTING:
        if final_point_state is None:
            issues.append(
                PlanningIssue(
                    "FINAL_POINT_STATE_REQUIRED",
                    "Existing Final Point requires an exact semantic state snapshot for reconciliation.",
                    final_ref.path,
                )
            )
        else:
            if (
                final_point_state.path != final_ref.path
                or final_point_state.draft_version != final_ref.draft_version
                or final_point_state.specification_revision != final_ref.specification_revision
            ):
                issues.append(
                    PlanningIssue(
                        "FINAL_POINT_STATE_MISMATCH",
                        "Final Point semantic snapshot identity does not match WU-01 Final Point identity.",
                        final_ref.path,
                    )
                )
            if final_ref.content_sha256 and final_point_state.content_sha256 != final_ref.content_sha256:
                issues.append(
                    PlanningIssue(
                        "FINAL_POINT_CONTENT_STALE",
                        "Final Point content SHA does not match WU-01 identity.",
                        final_ref.path,
                    )
                )
    elif final_point_state is not None:
        issues.append(
            PlanningIssue(
                "PLANNED_FINAL_POINT_ALREADY_HAS_STATE",
                "A planned Final Point must not be represented as pre-existing accepted state.",
                final_ref.path,
            )
        )

    working_state = (
        final_point_state
        if final_point_state is not None
        else FinalPointStateSnapshot(
            final_ref.path,
            final_ref.draft_version,
            final_ref.specification_revision,
            None,
            (),
        )
    )

    analysis_map = {item.source_generation.profile_path: item for item in analyses}
    proposal_map = {item.source_generation.profile_path: item for item in proposal_sets}
    ordered = list(transition.ordered_sources)
    steps: list[SourceConvergencePlan] = []
    blocking: list[str] = []

    for index, identity in enumerate(ordered):
        analysis = analysis_map.get(identity.path)
        proposal_set = proposal_map.get(identity.path)
        if analysis is None:
            issues.append(
                PlanningIssue(
                    "SOURCE_ANALYSIS_MISSING",
                    "WU-05 analysis is missing for ordered source.",
                    identity.path,
                )
            )
            continue
        if not _generation_matches(identity, analysis.source_generation):
            issues.append(
                PlanningIssue(
                    "SOURCE_ANALYSIS_IDENTITY_MISMATCH",
                    "WU-05 source identity does not match WU-01 ordered source.",
                    identity.path,
                )
            )
        if proposal_set is None:
            issues.append(
                PlanningIssue(
                    "SOURCE_PROPOSAL_SET_MISSING",
                    "WU-06 proposal set is missing for ordered source.",
                    identity.path,
                )
            )
            continue
        if proposal_set.analysis_digest != analysis.deterministic_digest:
            issues.append(
                PlanningIssue(
                    "PROPOSAL_ANALYSIS_DIGEST_MISMATCH",
                    "Proposal set is not bound to current WU-05 deterministic analysis.",
                    identity.path,
                )
            )

        accepted_refs: set[str] = set()
        reconciliations: list[ReconciliationResult] = []
        required_delta_ids: list[str] = []
        async_delta_ids: list[str] = []
        accepted_conflicts: list[str] = []

        for bundle in sorted(proposal_set.accepted, key=lambda item: item.proposal_id):
            for delta in bundle.deltas:
                accepted_refs.update(delta.obligation_ids)
                reconciliation = reconcile_delta(
                    delta,
                    working_state,
                    accepted=True,
                    bundle_id=bundle.proposal_id,
                )
                reconciliations.append(reconciliation)
                if reconciliation.status in {
                    ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION,
                    ReconciliationStatus.UNRESOLVED,
                }:
                    accepted_conflicts.append(delta.id)
                else:
                    working_state = _state_with_delta(working_state, delta, reconciliation)
                if bundle.async_only:
                    async_delta_ids.append(delta.id)
                else:
                    required_delta_ids.append(delta.id)

        for bundle in sorted(proposal_set.suggested, key=lambda item: item.id):
            for delta in bundle.deltas:
                reconciliations.append(
                    reconcile_delta(
                        delta,
                        working_state,
                        accepted=False,
                        bundle_id=bundle.id,
                    )
                )

        unplanned_required = tuple(
            sorted(
                item.id
                for item in analysis.required
                if not item.resolved and item.id not in accepted_refs
            )
        )
        blocking_suggested = tuple(
            sorted(item.id for item in proposal_set.suggested if item.blocking)
        )
        blocking_unresolved = tuple(
            sorted(item.id for item in proposal_set.unresolved if item.blocking)
        )
        proposal_issue_ids = tuple(
            f"proposal-issue:{semantic_digest(item)[:16]}" for item in proposal_set.issues
        )
        source_blocking = tuple(
            sorted(
                set(unplanned_required)
                | set(blocking_suggested)
                | set(blocking_unresolved)
                | set(accepted_conflicts)
                | set(proposal_issue_ids)
            )
        )
        blocking.extend(f"{identity.path}:{item}" for item in source_blocking)

        if source_blocking:
            deletion_gate = DeletionGate.BLOCKED
        elif analysis.completion.complete and not required_delta_ids:
            deletion_gate = DeletionGate.ALREADY_ELIGIBLE
        else:
            deletion_gate = DeletionGate.REQUIRES_POST_APPLY_VERIFICATION

        steps.append(
            SourceConvergencePlan(
                source_path=identity.path,
                source_content_sha256=identity.content_sha256,
                analysis_digest=analysis.deterministic_digest,
                required_total=analysis.completion.required_total,
                required_unresolved_before=analysis.completion.required_unresolved,
                accepted_bundle_ids=tuple(
                    sorted(item.proposal_id for item in proposal_set.accepted)
                ),
                suggested_bundle_ids=tuple(
                    sorted(item.id for item in proposal_set.suggested)
                ),
                unresolved_bundle_ids=tuple(
                    sorted(item.id for item in proposal_set.unresolved)
                ),
                unplanned_required_ids=unplanned_required,
                execution_delta_ids=tuple(sorted(required_delta_ids) + sorted(async_delta_ids)),
                reconciliations=tuple(
                    sorted(reconciliations, key=lambda item: (item.bundle_id, item.delta_id))
                ),
                deletion_gate=deletion_gate,
                next_source_path=ordered[index + 1].path if index + 1 < len(ordered) else None,
                projected_final_state_digest=working_state.semantic_digest,
            )
        )

    blocking.extend(
        f"issue:{item.code}:{item.source_path or ''}"
        for item in issues
    )
    step_rows = tuple(steps)
    ready = (
        not blocking
        and len(step_rows) == len(ordered)
        and all(item.deletion_gate != DeletionGate.BLOCKED for item in step_rows)
    )
    preview = ExecutionPreview(
        final_point=final_ref,
        ordered_sources=tuple(item.path for item in ordered),
        source_steps=step_rows,
        blocking_ids=tuple(sorted(set(blocking))),
        ready_for_wu07=ready,
    )
    return FinalPointConvergencePlan(
        final_point=final_ref,
        source_steps=step_rows,
        issues=tuple(sorted(issues, key=lambda item: (item.code, item.source_path or "", item.message))),
        preview=preview,
        projected_final_state_digest=working_state.semantic_digest,
    )
