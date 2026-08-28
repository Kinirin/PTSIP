from __future__ import annotations

from ptsip.migration.model import (
    ArchitectureFinding,
    ArchitectureFindingKind,
    AsynchronousWorkTarget,
    EvidenceCorrelation,
    MigrationAnalysis,
    RequiredWorkElement,
    SourceMigrationCompletion,
    TargetCompatibility,
)
from ptsip.migration.planner import (
    DeletionGate,
    FinalPointKind,
    ReconciliationStatus,
    build_final_point_convergence_plan,
    derive_source_proposals,
    final_point_state_from_mapping,
    reconcile_delta,
)
from ptsip.migration.proposal import (
    AcceptedDeltaBundle,
    DeltaChangeKind,
    ProposalBundle,
    ProposalPurpose,
    TargetDelta,
    TargetEntityKind,
)
from ptsip.repository.profile_transition import DraftVersion, ProfileGenerationIdentity, ProfileTransitionState
from ptsip.source_compat.model import SourceGenerationBinding


SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"


def _binding(
    path: str = "ptsip.yaml",
    *,
    version: str = "0.3.6-draft",
    revision: str = "rev6",
    content_sha256: str = "source-sha",
    temporary: bool = False,
) -> SourceGenerationBinding:
    return SourceGenerationBinding(
        profile_path=path,
        declared_version=version,
        specification_revision=revision,
        specification_source=SPEC_SOURCE,
        content_sha256=content_sha256,
        temporary=temporary,
    )


def _identity(binding: SourceGenerationBinding) -> ProfileGenerationIdentity:
    version = DraftVersion.from_draft_label(binding.declared_version)
    assert version is not None
    return ProfileGenerationIdentity(
        path=binding.profile_path,
        version=version,
        declared_version=binding.declared_version,
        specification_revision=binding.specification_revision,
        specification_source=binding.specification_source,
        content_sha256=binding.content_sha256,
        temporary=binding.temporary,
    )


def _required(
    obligation_id: str = "required:src/a.py",
    *,
    resolved: bool = False,
    target_status: TargetCompatibility = TargetCompatibility.NOT_EVALUATED,
) -> RequiredWorkElement:
    return RequiredWorkElement(
        id=obligation_id,
        path=obligation_id.removeprefix("required:"),
        source_declaration_id="core",
        source_classification="PRODUCT",
        selector="src/**",
        evidence=EvidenceCorrelation((), (), ()),
        target_status=target_status,
        resolved=resolved,
    )


def _analysis(
    binding: SourceGenerationBinding | None = None,
    *,
    required: tuple[RequiredWorkElement, ...] | None = None,
    async_targets: tuple[AsynchronousWorkTarget, ...] = (),
    findings: tuple[ArchitectureFinding, ...] = (),
) -> MigrationAnalysis:
    binding = binding or _binding()
    required = required if required is not None else (_required(),)
    resolved = sum(item.resolved for item in required)
    completion = SourceMigrationCompletion(
        required_total=len(required),
        required_resolved=resolved,
        required_unresolved=len(required) - resolved,
        removal_count=0,
        async_count=len(async_targets),
    )
    return MigrationAnalysis(
        source_generation=binding,
        repository_head="head",
        repository_status_fingerprint="status",
        repository_content_fingerprint="content",
        required=required,
        removals=(),
        async_targets=async_targets,
        ambiguous=(),
        lifecycle_findings=(),
        architecture_findings=findings,
        issues=(),
        completion=completion,
    )


def _transition(
    *bindings: SourceGenerationBinding,
    final_point: SourceGenerationBinding | None = None,
) -> ProfileTransitionState:
    assert bindings
    identities = tuple(_identity(item) for item in bindings)
    final_identity = _identity(final_point) if final_point else None
    temporary = tuple(item for item in identities if item.temporary)
    if final_identity is not None and final_identity not in temporary:
        temporary = (*temporary, final_identity)
    return ProfileTransitionState(
        mode="SEQUENTIAL" if len(identities) > 1 else "SIMPLE",
        canonical_source=next(item for item in identities if not item.temporary),
        temporary_profiles=temporary,
        final_point=final_identity,
        ordered_sources=identities,
        snapshot=None,  # planning contract consumes already-frozen WU-01 identities only
    )


def _component_delta(
    analysis: MigrationAnalysis,
    *,
    entity_id: str = "core",
    classification: str = "PRODUCT",
    obligation_id: str = "required:src/a.py",
) -> TargetDelta:
    return TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id=entity_id,
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={
            "id": entity_id,
            "classification": classification,
            "include": ["src/**"],
        },
        obligation_ids=(obligation_id,),
    )


def _proposal(analysis: MigrationAnalysis, delta: TargetDelta, *, async_only: bool = False) -> ProposalBundle:
    return ProposalBundle.build(
        source_generation=analysis.source_generation,
        analysis_digest=analysis.deterministic_digest,
        deltas=(delta,),
        purpose=(ProposalPurpose.ASYNC_OPTIONAL if async_only else ProposalPurpose.REQUIRED_MIGRATION,),
        rationale="focused test proposal",
    )


def test_semantic_delta_identity_normalizes_set_like_order() -> None:
    left = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "core", "roles": ["VERIFICATION", "AUTOMATION"], "include": ["b/**", "a/**"]},
        obligation_ids=("b", "a"),
    )
    right = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"include": ["a/**", "b/**"], "roles": ["AUTOMATION", "VERIFICATION"], "id": "core"},
        obligation_ids=("a", "b"),
    )
    assert left.id == right.id


def test_accepted_bundle_requires_project_decision_identity() -> None:
    analysis = _analysis()
    proposal = _proposal(analysis, _component_delta(analysis))
    try:
        AcceptedDeltaBundle.from_proposal(proposal, decision_id="")
    except ValueError:
        pass
    else:
        raise AssertionError("empty decision identity must be rejected")


def test_unresolved_required_is_blocking_without_target_delta() -> None:
    result = derive_source_proposals(_analysis())
    assert len(result.unresolved) == 1
    assert result.unresolved[0].blocking


def test_suggested_required_delta_does_not_become_accepted_implicitly() -> None:
    analysis = _analysis()
    proposal = _proposal(analysis, _component_delta(analysis))
    result = derive_source_proposals(analysis, proposed=(proposal,))
    assert result.unresolved == ()
    assert result.blocking_proposal_ids == (proposal.id,)
    assert result.accepted == ()


def test_accepted_bundle_removes_same_proposal_from_suggested_set() -> None:
    analysis = _analysis()
    proposal = _proposal(analysis, _component_delta(analysis))
    accepted = AcceptedDeltaBundle.from_proposal(proposal, decision_id="ADR-test")
    result = derive_source_proposals(analysis, proposed=(proposal,), accepted=(accepted,))
    assert result.suggested == ()
    assert result.unresolved == ()
    assert result.accepted == (accepted,)


def test_already_resolved_required_becomes_no_change_record() -> None:
    analysis = _analysis(required=(_required(resolved=True, target_status=TargetCompatibility.ALREADY_SATISFIED),))
    result = derive_source_proposals(analysis)
    assert result.no_change_obligation_ids == ("required:src/a.py",)


def test_missing_relationship_stays_target_validity_question() -> None:
    finding = ArchitectureFinding(
        "relationship:a:b:READS",
        ArchitectureFindingKind.MISSING_RELATIONSHIP,
        "missing",
    )
    result = derive_source_proposals(_analysis(required=(), findings=(finding,)))
    assert result.unresolved[0].purpose == (ProposalPurpose.TARGET_VALIDITY,)
    assert result.unresolved[0].blocking


def test_evidence_conflict_is_reviewable_but_not_architecture_authority() -> None:
    finding = ArchitectureFinding(
        "src/a.py",
        ArchitectureFindingKind.EVIDENCE_CONFLICT,
        "conflict",
    )
    result = derive_source_proposals(_analysis(required=(), findings=(finding,)))
    assert result.unresolved[0].purpose == (ProposalPurpose.ADVISORY,)
    assert not result.unresolved[0].blocking


def test_async_targets_are_ignored_until_explicitly_requested() -> None:
    async_target = AsynchronousWorkTarget("async:docs/a.md", "docs/a.md", EvidenceCorrelation((), (), ()))
    result = derive_source_proposals(_analysis(required=(), async_targets=(async_target,)))
    assert result.ignored_async_ids == ("async:docs/a.md",)
    assert result.unresolved == ()


def test_requested_async_without_delta_is_nonblocking_unresolved() -> None:
    async_target = AsynchronousWorkTarget("async:docs/a.md", "docs/a.md", EvidenceCorrelation((), (), ()))
    result = derive_source_proposals(
        _analysis(required=(), async_targets=(async_target,)),
        requested_async_ids=("async:docs/a.md",),
    )
    assert len(result.unresolved) == 1
    assert result.unresolved[0].purpose == (ProposalPurpose.ASYNC_OPTIONAL,)
    assert not result.unresolved[0].blocking


def test_add_reconciliation_adds_absent_entity() -> None:
    analysis = _analysis()
    delta = _component_delta(analysis)
    result = reconcile_delta(delta, None, accepted=True, bundle_id="bundle")
    assert result.status == ReconciliationStatus.ADD_TARGET_DECLARATION


def test_identical_final_point_semantics_require_no_change() -> None:
    analysis = _analysis()
    delta = _component_delta(analysis)
    state = final_point_state_from_mapping(
        {
            "ptsip": {"version": "0.3.7-draft", "specification": {"revision": "rev7"}},
            "components": [{"id": "core", "classification": "PRODUCT", "include": ["src/**"]}],
        },
        path="ptsip_0.3.7.yaml",
    )
    result = reconcile_delta(delta, state, accepted=True, bundle_id="bundle")
    assert result.status == ReconciliationStatus.NO_CHANGE_REQUIRED


def test_accepted_replace_requires_explicit_owner_decision_status() -> None:
    state = final_point_state_from_mapping(
        {
            "ptsip": {"version": "0.3.7-draft", "specification": {"revision": "rev7"}},
            "components": [{"id": "core", "classification": "PRODUCT", "include": ["src/**"]}],
        },
        path="ptsip_0.3.7.yaml",
    )
    delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.REPLACE,
        before={"id": "core", "classification": "PRODUCT", "include": ["src/**"]},
        after={"id": "core", "classification": "DELIVERY", "include": ["src/**"]},
    )
    result = reconcile_delta(delta, state, accepted=True, bundle_id="bundle")
    assert result.status == ReconciliationStatus.REPLACE_WITH_EXPLICIT_OWNER_DECISION


def test_replace_fails_when_final_point_changed_from_reviewed_before_state() -> None:
    state = final_point_state_from_mapping(
        {
            "ptsip": {"version": "0.3.7-draft", "specification": {"revision": "rev7"}},
            "components": [{"id": "core", "classification": "PRODUCT", "include": ["src/**"]}],
        },
        path="ptsip_0.3.7.yaml",
    )
    delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.REPLACE,
        before={"id": "core", "classification": "OPERATIONS", "include": ["src/**"]},
        after={"id": "core", "classification": "DELIVERY", "include": ["src/**"]},
    )
    result = reconcile_delta(delta, state, accepted=True, bundle_id="bundle")
    assert result.status == ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION


def test_planner_creates_planned_final_point_without_mutating_repository() -> None:
    analysis = _analysis()
    proposal = _proposal(analysis, _component_delta(analysis))
    accepted = AcceptedDeltaBundle.from_proposal(proposal, decision_id="ADR-test")
    proposal_set = derive_source_proposals(analysis, accepted=(accepted,))
    transition = _transition(analysis.source_generation)
    plan = build_final_point_convergence_plan(
        transition,
        (analysis,),
        (proposal_set,),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert plan.final_point.kind == FinalPointKind.PLANNED
    assert plan.final_point.path == "ptsip_0.3.7.yaml"
    assert plan.preview.ready_for_wu07
    assert plan.source_steps[0].deletion_gate == DeletionGate.REQUIRES_POST_APPLY_VERIFICATION


def test_blocking_suggested_proposal_prevents_wu07_readiness() -> None:
    analysis = _analysis()
    proposal = _proposal(analysis, _component_delta(analysis))
    proposal_set = derive_source_proposals(analysis, proposed=(proposal,))
    plan = build_final_point_convergence_plan(
        _transition(analysis.source_generation),
        (analysis,),
        (proposal_set,),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert not plan.preview.ready_for_wu07
    assert plan.source_steps[0].deletion_gate == DeletionGate.BLOCKED


def test_required_deltas_precede_accepted_async_deltas_in_execution_preview() -> None:
    analysis = _analysis()
    required_delta = _component_delta(analysis)
    required_proposal = _proposal(analysis, required_delta)
    required_accepted = AcceptedDeltaBundle.from_proposal(required_proposal, decision_id="ADR-required")
    async_delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="docs",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "docs", "classification": "PRODUCT", "include": ["docs/**"]},
        obligation_ids=("async:docs",),
    )
    async_proposal = _proposal(analysis, async_delta, async_only=True)
    async_accepted = AcceptedDeltaBundle.from_proposal(async_proposal, decision_id="ADR-async")
    proposal_set = derive_source_proposals(analysis, accepted=(async_accepted, required_accepted))
    plan = build_final_point_convergence_plan(
        _transition(analysis.source_generation),
        (analysis,),
        (proposal_set,),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    ids = plan.source_steps[0].execution_delta_ids
    assert ids.index(required_delta.id) < ids.index(async_delta.id)


def test_existing_final_point_requires_exact_semantic_state_snapshot() -> None:
    analysis = _analysis()
    final_binding = _binding(
        "ptsip_0.3.7.yaml",
        version="0.3.7-draft",
        revision="rev7",
        content_sha256="final-sha",
        temporary=True,
    )
    plan = build_final_point_convergence_plan(
        _transition(analysis.source_generation, final_point=final_binding),
        (analysis,),
        (derive_source_proposals(analysis),),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert any(item.code == "FINAL_POINT_STATE_REQUIRED" for item in plan.issues)
    assert not plan.preview.ready_for_wu07


def test_multi_source_plan_preserves_wu01_order_and_direct_final_point_target() -> None:
    older = _analysis()
    newer_binding = _binding(
        "ptsip_0.3.6.yaml",
        content_sha256="newer-source",
        temporary=True,
    )
    newer = _analysis(newer_binding, required=())
    older_delta = _component_delta(older)
    older_accepted = AcceptedDeltaBundle.from_proposal(
        _proposal(older, older_delta),
        decision_id="ADR-old",
    )
    plan = build_final_point_convergence_plan(
        _transition(newer_binding, older.source_generation),
        (older, newer),
        (derive_source_proposals(older, accepted=(older_accepted,)), derive_source_proposals(newer)),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert plan.preview.ordered_sources == ("ptsip_0.3.6.yaml", "ptsip.yaml")
    assert plan.source_steps[0].next_source_path == "ptsip.yaml"
    assert plan.source_steps[1].next_source_path is None
    assert all(step.projected_final_state_digest for step in plan.source_steps)


def test_plan_digest_is_stable_under_input_collection_reordering() -> None:
    canonical = _analysis()
    temporary_binding = _binding(
        "ptsip_0.3.6.yaml",
        content_sha256="temp-sha",
        temporary=True,
    )
    temporary = _analysis(temporary_binding, required=())
    proposal = _proposal(canonical, _component_delta(canonical))
    accepted = AcceptedDeltaBundle.from_proposal(proposal, decision_id="ADR-canonical")
    canonical_set = derive_source_proposals(canonical, accepted=(accepted,))
    temporary_set = derive_source_proposals(temporary)
    transition = _transition(temporary_binding, canonical.source_generation)
    left = build_final_point_convergence_plan(
        transition,
        (canonical, temporary),
        (canonical_set, temporary_set),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    right = build_final_point_convergence_plan(
        transition,
        (temporary, canonical),
        (temporary_set, canonical_set),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert left.deterministic_digest == right.deterministic_digest


def test_cumulative_target_state_detects_cross_source_stable_id_conflict() -> None:
    first_binding = _binding(
        "ptsip_0.3.6.yaml",
        content_sha256="first-sha",
        temporary=True,
    )
    first = _analysis(first_binding, required=(_required("required:first.py"),))
    second = _analysis(required=(_required("required:second.py"),))

    first_delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="shared",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "shared", "classification": "PRODUCT", "include": ["first.py"]},
        obligation_ids=("required:first.py",),
    )
    second_delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="shared",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "shared", "classification": "DELIVERY", "include": ["second.py"]},
        obligation_ids=("required:second.py",),
    )
    first_accepted = AcceptedDeltaBundle.from_proposal(
        _proposal(first, first_delta),
        decision_id="ADR-first",
    )
    second_accepted = AcceptedDeltaBundle.from_proposal(
        _proposal(second, second_delta),
        decision_id="ADR-second",
    )
    plan = build_final_point_convergence_plan(
        _transition(first_binding, second.source_generation),
        (second, first),
        (
            derive_source_proposals(first, accepted=(first_accepted,)),
            derive_source_proposals(second, accepted=(second_accepted,)),
        ),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert not plan.preview.ready_for_wu07
    assert any(
        item.status == ReconciliationStatus.CONFLICT_REQUIRES_CONFIRMATION
        for item in plan.source_steps[1].reconciliations
    )
