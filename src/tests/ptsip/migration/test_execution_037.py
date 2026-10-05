from __future__ import annotations

import hashlib
import json
import subprocess
from pathlib import Path

import pytest
import yaml

from ptsip.migration import (
    AcceptedDeltaBundle,
    CheckpointLedger,
    DeltaChangeKind,
    EvidenceCorrelation,
    ExecutionPhase,
    ExecutionStateError,
    LedgerIntegrityError,
    MigrationAnalysis,
    PostPromotionVerifiedState,
    ProposalBundle,
    ProposalPurpose,
    RequiredWorkElement,
    SourceCompletionProof,
    SourceMigrationCompletion,
    TargetCompatibility,
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
from ptsip.repository.profile_transition import DraftVersion, ProfileGenerationIdentity, ProfileTransitionState
from ptsip.repository.snapshot import capture_snapshot
from ptsip.source_compat.model import SourceGenerationBinding


SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"


def _git(root: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(root), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _write_yaml(path: Path, payload: dict[str, object]) -> None:
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8", newline="\n")


def _sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _init_git_repository(root: Path) -> None:
    root.mkdir(parents=True, exist_ok=True)
    _git(root, "init")
    _git(root, "config", "user.name", "PTSIP Fixture")
    _git(root, "config", "user.email", "ptsip-fixture@example.invalid")


def _commit_fixture(root: Path, message: str = "fixture baseline") -> None:
    _git(root, "add", ".")
    _git(root, "commit", "-m", message)


def _profile_payload(version: str, revision: str) -> dict[str, object]:
    return {
        "ptsip": {
            "version": version,
            "specification": {
                "source": SPEC_SOURCE,
                "revision": revision,
            },
        },
        "responsibility_map": {"mode": "explicit"},
    }


def _binding(path: str, version: str, revision: str, content_sha256: str, *, temporary: bool) -> SourceGenerationBinding:
    return SourceGenerationBinding(
        profile_path=path,
        declared_version=version,
        specification_revision=revision,
        specification_source=SPEC_SOURCE,
        content_sha256=content_sha256,
        temporary=temporary,
    )


def _identity(binding: SourceGenerationBinding) -> ProfileGenerationIdentity:
    parsed = DraftVersion.from_draft_label(binding.declared_version)
    assert parsed is not None
    return ProfileGenerationIdentity(
        path=binding.profile_path,
        version=parsed,
        declared_version=binding.declared_version,
        specification_revision=binding.specification_revision,
        specification_source=binding.specification_source,
        content_sha256=binding.content_sha256,
        temporary=binding.temporary,
    )


def _required() -> RequiredWorkElement:
    return RequiredWorkElement(
        id="required:src/a.py",
        path="src/a.py",
        source_declaration_id="core",
        source_classification="PRODUCT",
        selector="src/**",
        evidence=EvidenceCorrelation((), (), ()),
        target_status=TargetCompatibility.NOT_EVALUATED,
        resolved=False,
    )


def _analysis(
    binding: SourceGenerationBinding,
    root: Path,
    *,
    required: tuple[RequiredWorkElement, ...],
) -> MigrationAnalysis:
    snapshot = capture_snapshot(root)
    assert not snapshot.observation_errors
    resolved = sum(item.resolved for item in required)
    return MigrationAnalysis(
        source_generation=binding,
        repository_head=snapshot.head,
        repository_status_fingerprint=snapshot.status_fingerprint,
        repository_content_fingerprint=snapshot.tracked_content_fingerprint,
        required=required,
        removals=(),
        async_targets=(),
        ambiguous=(),
        lifecycle_findings=(),
        architecture_findings=(),
        issues=(),
        completion=SourceMigrationCompletion(
            required_total=len(required),
            required_resolved=resolved,
            required_unresolved=len(required) - resolved,
            removal_count=0,
            async_count=0,
        ),
    )


def _component_add_delta() -> TargetDelta:
    return TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id="core",
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={
            "id": "core",
            "classification": "PRODUCT",
            "include": ["src/**"],
        },
        obligation_ids=("required:src/a.py",),
    )


def _accepted_set(analysis: MigrationAnalysis, *, decision_id: str = "ADR-fixture"):
    proposal = ProposalBundle.build(
        source_generation=analysis.source_generation,
        analysis_digest=analysis.deterministic_digest,
        deltas=(_component_add_delta(),),
        purpose=(ProposalPurpose.REQUIRED_MIGRATION,),
        rationale="WU-07 fixture accepted migration delta",
    )
    accepted = AcceptedDeltaBundle.from_proposal(proposal, decision_id=decision_id)
    return derive_source_proposals(analysis, accepted=(accepted,))


def _simple_bound_fixture(root: Path):
    _init_git_repository(root)
    (root / "src").mkdir()
    (root / "src" / "a.py").write_text("VALUE = 1\n", encoding="utf-8")
    canonical_path = root / "ptsip.yaml"
    _write_yaml(canonical_path, _profile_payload("0.3.6-draft", "rev6"))
    _commit_fixture(root)

    binding = _binding("ptsip.yaml", "0.3.6-draft", "rev6", _sha256(canonical_path), temporary=False)
    analysis = _analysis(binding, root, required=(_required(),))
    proposal_set = _accepted_set(analysis)
    identity = _identity(binding)
    transition = ProfileTransitionState(
        mode="SIMPLE",
        canonical_source=identity,
        temporary_profiles=(),
        final_point=None,
        ordered_sources=(identity,),
        snapshot=None,
    )
    plan = build_final_point_convergence_plan(
        transition,
        (analysis,),
        (proposal_set,),
        target_draft_version="0.3.7-draft",
        target_specification_revision="rev7",
    )
    assert plan.preview.ready_for_wu07
    bound = bind_execution_plan(
        root,
        plan,
        {"ptsip.yaml": analysis},
        {"ptsip.yaml": proposal_set},
    )
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("ADR-fixture",),
        authority_revision="authority-rev-1",
    )
    authorized = authorize_execution(bound, proof, ledger)
    return bound, authorized, ledger


def _complete_proof(source_path: str, _final_sha: str, analysis_digest: str) -> SourceCompletionProof:
    return SourceCompletionProof(
        source_path=source_path,
        analysis_digest=analysis_digest,
        required_total=1,
        required_unresolved=0,
        target_valid=True,
        evidence=("fixture-post-apply-analysis",),
    )


def test_fixture_simple_transition_runs_authorized_apply_and_guarded_promotion(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _bound, authorized, ledger = _simple_bound_fixture(root)

    verified = verify_source_preconditions(root, authorized, 0, ledger)
    seed = _profile_payload("0.3.7-draft", "rev7")
    seed["x-fixture-metadata"] = {"preserve": ["alpha", "beta"]}
    applied = apply_required_deltas(root, verified, ledger, planned_final_point_seed=seed)

    payload = yaml.safe_load((root / "ptsip_0.3.7.yaml").read_text(encoding="utf-8"))
    assert payload["x-fixture-metadata"] == {"preserve": ["alpha", "beta"]}
    assert payload["components"][0]["id"] == "core"

    reanalyzed = reanalyze_source(applied, _complete_proof, ledger)
    completed = complete_source(reanalyzed, ledger)
    canonical_complete = finalize_source(root, completed, ledger)
    ready = prepare_promotion(root, canonical_complete, ledger)
    promoted = promote_canonical(root, ready, ledger)
    post = verify_post_promotion(root, promoted, ledger)

    assert isinstance(post, PostPromotionVerifiedState)
    assert not (root / "ptsip_0.3.7.yaml").exists()
    canonical_payload = yaml.safe_load((root / "ptsip.yaml").read_text(encoding="utf-8"))
    assert canonical_payload["ptsip"]["version"] == "0.3.7-draft"
    assert canonical_payload["ptsip"]["specification"]["revision"] == "rev7"
    assert [row.phase for row in ledger.read_all()][-1] == ExecutionPhase.POST_PROMOTION_VERIFIED


def test_authorization_requires_exact_accepted_decision_set(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    bound, _authorized, _ledger = _simple_bound_fixture(root)

    with pytest.raises(ExecutionStateError, match="exactly match"):
        build_authorization_proof(
            bound,
            decision_ids=(),
            authority_revision="authority-rev-1",
        )


def test_stale_source_blocks_precondition_verification(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _bound, authorized, ledger = _simple_bound_fixture(root)
    (root / "ptsip.yaml").write_text((root / "ptsip.yaml").read_text(encoding="utf-8") + "\n", encoding="utf-8")

    with pytest.raises(ExecutionStateError, match="Source profile changed"):
        verify_source_preconditions(root, authorized, 0, ledger)


def test_incomplete_required_work_blocks_source_completion(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _bound, authorized, ledger = _simple_bound_fixture(root)
    verified = verify_source_preconditions(root, authorized, 0, ledger)
    applied = apply_required_deltas(
        root,
        verified,
        ledger,
        planned_final_point_seed=_profile_payload("0.3.7-draft", "rev7"),
    )

    def incomplete(source_path: str, _final_sha: str, analysis_digest: str) -> SourceCompletionProof:
        return SourceCompletionProof(source_path, analysis_digest, 1, 1, True, ("still-required",))

    reanalyzed = reanalyze_source(applied, incomplete, ledger)
    with pytest.raises(ExecutionStateError, match="Required Work Elements are not complete"):
        complete_source(reanalyzed, ledger)
    assert (root / "ptsip.yaml").exists()


def test_post_apply_completion_proof_must_match_bound_analysis_digest(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _bound, authorized, ledger = _simple_bound_fixture(root)
    verified = verify_source_preconditions(root, authorized, 0, ledger)
    applied = apply_required_deltas(
        root,
        verified,
        ledger,
        planned_final_point_seed=_profile_payload("0.3.7-draft", "rev7"),
    )

    def wrong_analysis(source_path: str, _final_sha: str, _analysis_digest: str) -> SourceCompletionProof:
        return SourceCompletionProof(source_path, "wrong-analysis-digest", 1, 0, True, ())

    with pytest.raises(ExecutionStateError, match="analysis digest"):
        reanalyze_source(applied, wrong_analysis, ledger)


def test_recovery_rejects_uncheckpointed_planned_final_point(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    bound, _authorized, ledger = _simple_bound_fixture(root)
    _write_yaml(root / "ptsip_0.3.7.yaml", _profile_payload("0.3.7-draft", "rev7"))

    inspection = inspect_recovery(root, bound, ledger)
    assert not inspection.safe_to_resume
    assert any("without a persisted mutation checkpoint" in item for item in inspection.reasons)


def test_checkpoint_ledger_detects_tampering(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _bound, _authorized, ledger = _simple_bound_fixture(root)
    first = sorted(ledger.root.glob("*.json"))[0]
    payload = json.loads(first.read_text(encoding="utf-8"))
    payload["payload"] = {"tampered": True}
    first.write_text(json.dumps(payload, sort_keys=True), encoding="utf-8")

    with pytest.raises(LedgerIntegrityError, match="digest"):
        ledger.read_all()


def test_wrong_source_order_is_rejected_before_mutation(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _init_git_repository(root)
    canonical_path = root / "ptsip.yaml"
    temporary_path = root / "ptsip_0.3.6.yaml"
    _write_yaml(canonical_path, _profile_payload("0.3.4-draft", "rev4"))
    _write_yaml(temporary_path, _profile_payload("0.3.6-draft", "rev6"))
    _commit_fixture(root)

    canonical_binding = _binding("ptsip.yaml", "0.3.4-draft", "rev4", _sha256(canonical_path), temporary=False)
    temporary_binding = _binding(
        "ptsip_0.3.6.yaml",
        "0.3.6-draft",
        "rev6",
        _sha256(temporary_path),
        temporary=True,
    )
    canonical_analysis = _analysis(canonical_binding, root, required=())
    temporary_analysis = _analysis(temporary_binding, root, required=())
    canonical_set = derive_source_proposals(canonical_analysis)
    temporary_set = derive_source_proposals(temporary_analysis)
    canonical_identity = _identity(canonical_binding)
    temporary_identity = _identity(temporary_binding)
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
        (canonical_analysis, temporary_analysis),
        (canonical_set, temporary_set),
        target_draft_version="0.4.0-draft",
        target_specification_revision="rev40",
    )
    assert plan.preview.ready_for_wu07
    bound = bind_execution_plan(
        root,
        plan,
        {
            "ptsip.yaml": canonical_analysis,
            "ptsip_0.3.6.yaml": temporary_analysis,
        },
        {
            "ptsip.yaml": canonical_set,
            "ptsip_0.3.6.yaml": temporary_set,
        },
    )
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(bound, decision_ids=(), authority_revision="authority-rev-1")
    authorized = authorize_execution(bound, proof, ledger)

    with pytest.raises(ExecutionStateError, match="not the next source"):
        verify_source_preconditions(root, authorized, 1, ledger)
