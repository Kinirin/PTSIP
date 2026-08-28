from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import pytest

from ptsip.migration import (
    AcceptedDeltaBundle,
    CheckpointLedger,
    DeltaChangeKind,
    ExecutionStateError,
    PostPromotionVerifiedState,
    ProposalBundle,
    ProposalPurpose,
    RecoveryRequiredState,
    SourceCompletionProof,
    TargetDelta,
    TargetEntityKind,
    apply_required_deltas,
    authorize_execution,
    bind_execution_plan,
    build_authorization_proof,
    build_final_point_convergence_plan,
    complete_source,
    derive_source_proposals,
    finalize_source,
    inspect_recovery,
    prepare_promotion,
    promote_canonical,
    reanalyze_source,
    verify_post_promotion,
    verify_source_preconditions,
)
from ptsip.repository.profile_transition import ProfileTransitionState

from _execution_fixture import (
    analysis,
    binding,
    commit_all,
    identity,
    init_git_repository,
    profile_payload,
    removal,
    required,
    sha256,
    write_yaml,
)


@dataclass(frozen=True)
class SequentialFixture:
    root: Path
    authorized: object
    ledger: CheckpointLedger
    temporary_analysis_digest: str
    canonical_analysis_digest: str


def _component_delta() -> TargetDelta:
    return TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={"id": "core", "classification": "PRODUCT", "include": ["src/**"]},
        obligation_ids=("required:src/a.py",),
    )


def _build_sequential_fixture(tmp_path: Path) -> SequentialFixture:
    root = init_git_repository(tmp_path / "repo")
    (root / "src").mkdir()
    (root / "src" / "a.py").write_text("VALUE = 1\n", encoding="utf-8")
    canonical_path = write_yaml(root / "ptsip.yaml", profile_payload("0.3.4-draft", "rev4"))
    temporary_path = write_yaml(root / "ptsip_0.3.6.yaml", profile_payload("0.3.6-draft", "rev6"))
    commit_all(root)

    canonical_binding = binding(
        "ptsip.yaml",
        "0.3.4-draft",
        "rev4",
        sha256(canonical_path),
        temporary=False,
    )
    temporary_binding = binding(
        "ptsip_0.3.6.yaml",
        "0.3.6-draft",
        "rev6",
        sha256(temporary_path),
        temporary=True,
    )
    temporary_analysis = analysis(
        temporary_binding,
        root,
        removal_items=(removal(),),
    )
    canonical_analysis = analysis(
        canonical_binding,
        root,
        required_items=(required(),),
    )

    canonical_proposal = ProposalBundle.build(
        source_generation=canonical_binding,
        analysis_digest=canonical_analysis.deterministic_digest,
        deltas=(_component_delta(),),
        purpose=(ProposalPurpose.REQUIRED_MIGRATION,),
        rationale="fixture canonical migration",
    )
    canonical_accepted = AcceptedDeltaBundle.from_proposal(
        canonical_proposal,
        decision_id="ADR-canonical-fixture",
    )
    temporary_set = derive_source_proposals(temporary_analysis)
    canonical_set = derive_source_proposals(canonical_analysis, accepted=(canonical_accepted,))

    temporary_identity = identity(temporary_binding)
    canonical_identity = identity(canonical_binding)
    transition = ProfileTransitionState(
        mode="SEQUENTIAL",
        canonical_source=canonical_identity,
        temporary_profiles=(temporary_identity,),
        final_point=None,
        ordered_sources=(temporary_identity, canonical_identity),
        snapshot=None,
    )
    plan = build_final_point_convergence_plan(
        transition,
        (temporary_analysis, canonical_analysis),
        (temporary_set, canonical_set),
        target_draft_version="0.4.0-draft",
        target_specification_revision="rev40",
    )
    assert plan.preview.ready_for_wu07
    assert plan.preview.ordered_sources == ("ptsip_0.3.6.yaml", "ptsip.yaml")

    bound = bind_execution_plan(
        root,
        plan,
        {
            "ptsip_0.3.6.yaml": temporary_analysis,
            "ptsip.yaml": canonical_analysis,
        },
        {
            "ptsip_0.3.6.yaml": temporary_set,
            "ptsip.yaml": canonical_set,
        },
    )
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("ADR-canonical-fixture",),
        authority_revision="authority-rev-sequential",
    )
    authorized = authorize_execution(bound, proof, ledger)
    return SequentialFixture(
        root,
        authorized,
        ledger,
        temporary_analysis.deterministic_digest,
        canonical_analysis.deterministic_digest,
    )


def _complete(source_path: str, analysis_digest: str, *, required_total: int) -> SourceCompletionProof:
    return SourceCompletionProof(
        source_path=source_path,
        analysis_digest=analysis_digest,
        required_total=required_total,
        required_unresolved=0,
        target_valid=True,
        evidence=("fixture-reanalysis",),
    )


def _finish_temporary(fixture: SequentialFixture) -> None:
    verified = verify_source_preconditions(fixture.root, fixture.authorized, 0, fixture.ledger)
    applied = apply_required_deltas(
        fixture.root,
        verified,
        fixture.ledger,
        planned_final_point_seed=profile_payload("0.4.0-draft", "rev40"),
    )
    reanalyzed = reanalyze_source(
        applied,
        lambda source_path, _final_sha, analysis_digest: _complete(
            source_path,
            analysis_digest,
            required_total=0,
        ),
        fixture.ledger,
    )
    completed = complete_source(reanalyzed, fixture.ledger)
    finalize_source(fixture.root, completed, fixture.ledger)


def _finish_canonical(fixture: SequentialFixture):
    verified = verify_source_preconditions(fixture.root, fixture.authorized, 1, fixture.ledger)
    applied = apply_required_deltas(fixture.root, verified, fixture.ledger)
    reanalyzed = reanalyze_source(
        applied,
        lambda source_path, _final_sha, analysis_digest: _complete(
            source_path,
            analysis_digest,
            required_total=1,
        ),
        fixture.ledger,
    )
    completed = complete_source(reanalyzed, fixture.ledger)
    return finalize_source(fixture.root, completed, fixture.ledger)


def test_sequential_fixture_removes_temporary_then_resumes_at_canonical_and_promotes(tmp_path: Path) -> None:
    fixture = _build_sequential_fixture(tmp_path)

    _finish_temporary(fixture)

    assert not (fixture.root / "ptsip_0.3.6.yaml").exists()
    assert (fixture.root / "ptsip.yaml").exists()
    assert (fixture.root / "ptsip_0.4.0.yaml").exists()
    recovery = inspect_recovery(fixture.root, fixture.authorized.bound, fixture.ledger)
    assert recovery.safe_to_resume
    assert recovery.next_source_index == 1

    canonical_complete = _finish_canonical(fixture)
    ready = prepare_promotion(fixture.root, canonical_complete, fixture.ledger)
    promoted = promote_canonical(fixture.root, ready, fixture.ledger)
    post = verify_post_promotion(fixture.root, promoted, fixture.ledger)

    assert isinstance(post, PostPromotionVerifiedState)
    assert (fixture.root / "ptsip.yaml").exists()
    assert not (fixture.root / "ptsip_0.4.0.yaml").exists()


def test_stale_final_point_blocks_next_source_after_temporary_removal(tmp_path: Path) -> None:
    fixture = _build_sequential_fixture(tmp_path)
    _finish_temporary(fixture)
    final_path = fixture.root / "ptsip_0.4.0.yaml"
    final_path.write_text(final_path.read_text(encoding="utf-8") + "\n", encoding="utf-8")

    with pytest.raises(ExecutionStateError, match="Final Point content changed"):
        verify_source_preconditions(fixture.root, fixture.authorized, 1, fixture.ledger)


def test_post_promotion_unrelated_repository_drift_requires_recovery(tmp_path: Path) -> None:
    fixture = _build_sequential_fixture(tmp_path)
    _finish_temporary(fixture)
    canonical_complete = _finish_canonical(fixture)
    ready = prepare_promotion(fixture.root, canonical_complete, fixture.ledger)
    promoted = promote_canonical(fixture.root, ready, fixture.ledger)

    (fixture.root / "src" / "a.py").write_text("VALUE = 2\n", encoding="utf-8")
    post = verify_post_promotion(fixture.root, promoted, fixture.ledger)

    assert isinstance(post, RecoveryRequiredState)
    assert "repository changed outside controlled profile paths during promotion" in post.reason
