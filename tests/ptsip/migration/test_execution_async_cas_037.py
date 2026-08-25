from __future__ import annotations

import copy
from pathlib import Path

import pytest
import yaml

from ptsip.migration import (
    AcceptedDeltaBundle,
    AsyncAppliedSourceStep,
    CheckpointLedger,
    DeltaChangeKind,
    ExecutionPhase,
    ExecutionStateError,
    ProposalBundle,
    ProposalPurpose,
    SourceCompletionProof,
    TargetDelta,
    TargetEntityKind,
    apply_optional_async_deltas,
    apply_required_deltas,
    authorize_execution,
    bind_execution_plan,
    build_authorization_proof,
    build_final_point_convergence_plan,
    complete_source,
    derive_source_proposals,
    finalize_source,
    reanalyze_source,
    verify_source_preconditions,
)
from ptsip.migration.execution_apply import _apply_delta
from ptsip.repository.profile_transition import ProfileTransitionState

from _execution_fixture import (
    analysis,
    async_target,
    binding,
    commit_all,
    identity,
    init_git_repository,
    profile_payload,
    required,
    sha256,
    write_yaml,
)


def _accepted(
    source,
    analysis_digest: str,
    delta: TargetDelta,
    purpose: ProposalPurpose,
    decision_id: str,
) -> AcceptedDeltaBundle:
    proposal = ProposalBundle.build(
        source_generation=source,
        analysis_digest=analysis_digest,
        deltas=(delta,),
        purpose=(purpose,),
        rationale=f"fixture {purpose.value.lower()}",
    )
    return AcceptedDeltaBundle.from_proposal(proposal, decision_id=decision_id)


def test_async_delta_is_applied_only_after_required_completion_and_contributes_no_completion(tmp_path: Path) -> None:
    root = init_git_repository(tmp_path / "repo")
    (root / "src").mkdir()
    (root / "src" / "a.py").write_text("VALUE = 1\n", encoding="utf-8")
    (root / "docs").mkdir()
    (root / "docs" / "readme.md").write_text("fixture\n", encoding="utf-8")
    canonical_path = write_yaml(root / "ptsip.yaml", profile_payload("0.3.6-draft", "rev6"))
    commit_all(root)

    source = binding(
        "ptsip.yaml",
        "0.3.6-draft",
        "rev6",
        sha256(canonical_path),
        temporary=False,
    )
    async_id = "async:docs/readme.md"
    source_analysis = analysis(
        source,
        root,
        required_items=(required(),),
        async_items=(async_target(async_id),),
    )
    required_delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "core", "classification": "PRODUCT", "include": ["src/**"]},
        obligation_ids=("required:src/a.py",),
    )
    async_delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="docs",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "docs", "classification": "NEUTRAL_CONTRACT", "include": ["docs/**"]},
        obligation_ids=(async_id,),
    )
    required_accepted = _accepted(
        source,
        source_analysis.deterministic_digest,
        required_delta,
        ProposalPurpose.REQUIRED_MIGRATION,
        "ADR-required-fixture",
    )
    async_accepted = _accepted(
        source,
        source_analysis.deterministic_digest,
        async_delta,
        ProposalPurpose.ASYNC_OPTIONAL,
        "ADR-async-fixture",
    )
    proposal_set = derive_source_proposals(
        source_analysis,
        accepted=(async_accepted, required_accepted),
        requested_async_ids=(async_id,),
    )
    source_identity = identity(source)
    transition = ProfileTransitionState(
        mode="SIMPLE",
        canonical_source=source_identity,
        temporary_profiles=(),
        final_point=None,
        ordered_sources=(source_identity,),
        snapshot=None,
    )
    plan = build_final_point_convergence_plan(
        transition,
        (source_analysis,),
        (proposal_set,),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert plan.preview.ready_for_wu07
    execution_ids = plan.source_steps[0].execution_delta_ids
    assert execution_ids.index(required_delta.id) < execution_ids.index(async_delta.id)

    bound = bind_execution_plan(
        root,
        plan,
        {"ptsip.yaml": source_analysis},
        {"ptsip.yaml": proposal_set},
    )
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("ADR-required-fixture", "ADR-async-fixture"),
        authority_revision="authority-rev-async",
    )
    authorized = authorize_execution(bound, proof, ledger)
    verified = verify_source_preconditions(root, authorized, 0, ledger)
    applied = apply_required_deltas(
        root,
        verified,
        ledger,
        planned_final_point_seed=profile_payload("0.3.7-draft", "rev7"),
    )

    required_payload = yaml.safe_load((root / "ptsip_0.3.7.yaml").read_text(encoding="utf-8"))
    assert [item["id"] for item in required_payload["components"]] == ["core"]

    reanalyzed = reanalyze_source(
        applied,
        lambda source_path, _final_sha, analysis_digest: SourceCompletionProof(
            source_path,
            analysis_digest,
            1,
            0,
            True,
            ("required-complete-before-async",),
        ),
        ledger,
    )
    completed = complete_source(reanalyzed, ledger)
    assert completed.reanalyzed.proof.complete
    assert ledger.latest() is not None
    assert ledger.latest().phase == ExecutionPhase.SOURCE_COMPLETE

    async_applied = apply_optional_async_deltas(root, completed, ledger)
    assert isinstance(async_applied, AsyncAppliedSourceStep)
    assert async_applied.completed.reanalyzed.proof == completed.reanalyzed.proof
    latest = ledger.latest()
    assert latest is not None
    assert latest.phase == ExecutionPhase.ASYNC_APPLIED
    assert latest.payload["completion_contribution"] is False

    target_payload = yaml.safe_load((root / "ptsip_0.3.7.yaml").read_text(encoding="utf-8"))
    assert {item["id"] for item in target_payload["components"]} == {"core", "docs"}
    finalize_source(root, async_applied, ledger)
    assert (root / "ptsip.yaml").exists()
    assert (root / "ptsip_0.3.7.yaml").exists()


def test_replace_delta_with_stale_before_state_fails_without_mutating_payload() -> None:
    payload: dict[str, object] = {
        "components": [
            {"id": "core", "classification": "PRODUCT", "include": ["src/**"]}
        ]
    }
    original = copy.deepcopy(payload)
    delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.REPLACE,
        before={"id": "core", "classification": "OPERATIONS", "include": ["src/**"]},
        after={"id": "core", "classification": "DELIVERY", "include": ["src/**"]},
    )

    with pytest.raises(ExecutionStateError, match="before-state is stale"):
        _apply_delta(payload, delta)

    assert payload == original
