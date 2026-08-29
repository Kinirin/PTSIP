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
from ptsip.migration.planner import build_final_point_convergence_plan
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
            "specification": {"source": SPEC_SOURCE, "revision": revision},
        },
        "responsibility_map": {"mode": "explicit"},
    }


def _binding(
    path: str,
    version: str,
    revision: str,
    content_sha256: str,
    *,
    temporary: bool,
) -> SourceGenerationBinding:
    return SourceGenerationBinding(
        profile_path=path,
        declared_version=version,
        specification_revision=revision,
        specification_source=SPEC_SOURCE,
        content_sha256=content_sha256,
        temporary=temporary,
    )


def _identity(source: SourceGenerationBinding) -> ProfileGenerationIdentity:
    version = DraftVersion.from_draft_label(source.declared_version)
    assert version is not None
    return ProfileGenerationIdentity(
        path=source.profile_path,
        version=version,
        declared_version=source.declared_version,
        specification_revision=source.specification_revision,
        specification_source=source.specification_source,
        content_sha256=source.content_sha256,
        temporary=source.temporary,
    )


def _required(
    obligation_id: str = "required:src/a.py",
    *,
    path: str = "src/a.py",
    declaration_id: str = "core",
    resolved: bool = False,
    target_status: TargetCompatibility = TargetCompatibility.NOT_EVALUATED,
) -> RequiredWorkElement:
    return RequiredWorkElement(
        id=obligation_id,
        path=path,
        source_declaration_id=declaration_id,
        source_classification="PRODUCT",
        selector="src/**",
        evidence=EvidenceCorrelation((), (), ()),
        target_status=target_status,
        resolved=resolved,
    )


def _analysis(
    source: SourceGenerationBinding,
    root: Path,
    *,
    required_items: tuple[RequiredWorkElement, ...] = (),
) -> MigrationAnalysis:
    snapshot = capture_snapshot(root)
    resolved = sum(item.resolved for item in required_items)
    return MigrationAnalysis(
        source_generation=source,
        repository_head=snapshot.head,
        repository_status_fingerprint=snapshot.status_fingerprint,
        repository_content_fingerprint=snapshot.tracked_content_fingerprint,
        required=required_items,
        removals=(),
        async_targets=(),
        ambiguous=(),
        lifecycle_findings=(),
        architecture_findings=(),
        issues=(),
        completion=SourceMigrationCompletion(
            required_total=len(required_items),
            required_resolved=resolved,
            required_unresolved=len(required_items) - resolved,
            removal_count=0,
            async_count=0,
        ),
    )


def _accepted_delta(
    source: SourceGenerationBinding,
    analysis_digest: str,
    *,
    entity_id: str = "core",
    decision_id: str = "decision:core",
) -> AcceptedDeltaBundle:
    delta = TargetDelta.build(
        entity_kind=TargetEntityKind.COMPONENT,
        entity_id=entity_id,
        change_kind=DeltaChangeKind.ADD,
        before=None,
        after={
            "id": entity_id,
            "classification": "PRODUCT",
            "include": ["src/**"],
            "purpose": "runtime",
        },
    )
    proposal = ProposalBundle.build(
        source_generation=source,
        analysis_digest=analysis_digest,
        deltas=(delta,),
        purpose=(ProposalPurpose.REQUIRED,),
        rationale="fixture required delta",
    )
    return AcceptedDeltaBundle.from_proposal(proposal, decision_id=decision_id)


def _build_fixture(tmp_path: Path):
    root = tmp_path / "repo"
    _init_git_repository(root)
    (root / "src").mkdir()
    (root / "src/a.py").write_text("VALUE = 1\n", encoding="utf-8")
    canonical_path = _write_yaml(root / "ptsip.yaml", _profile_payload("0.3.6-draft", "6" * 40))
    _commit_fixture(root)

    source = _binding(
        "ptsip.yaml",
        "0.3.6-draft",
        "6" * 40,
        _sha256(canonical_path),
        temporary=False,
    )
    analysis = _analysis(source, root, required_items=(_required(),))
    proposals = derive_source_proposals(analysis)
    accepted = _accepted_delta(source, analysis.deterministic_digest)
    proposals = proposals.__class__(
        source_generation=proposals.source_generation,
        analysis_digest=proposals.analysis_digest,
        suggested=proposals.suggested,
        accepted=(accepted,),
        unresolved=proposals.unresolved,
        no_change_obligation_ids=proposals.no_change_obligation_ids,
        ignored_async_ids=proposals.ignored_async_ids,
        issues=proposals.issues,
    )
    state = ProfileTransitionState(
        canonical=_identity(source),
        temporaries=(),
        all_generations=(_identity(source),),
        issues=(),
    )
    target_path = root / "ptsip_0.3.7.yaml"
    target_payload = _profile_payload("0.3.7-draft", "7" * 40)
    _write_yaml(target_path, target_payload)
    final_source = _binding(
        "ptsip_0.3.7.yaml",
        "0.3.7-draft",
        "7" * 40,
        _sha256(target_path),
        temporary=True,
    )
    state = ProfileTransitionState(
        canonical=state.canonical,
        temporaries=(_identity(final_source),),
        all_generations=(state.canonical, _identity(final_source)),
        issues=(),
    )
    final_point = target_payload
    plan = build_final_point_convergence_plan(
        state,
        {"ptsip.yaml": analysis},
        {"ptsip.yaml": proposals},
        final_point_payload=final_point,
        final_point_path="ptsip_0.3.7.yaml",
    )
    return root, source, analysis, proposals, plan


def test_execution_requires_explicit_authorization_before_apply(tmp_path: Path) -> None:
    root, _source, analysis, proposals, plan = _build_fixture(tmp_path)
    bound = bind_execution_plan(root, plan, {"ptsip.yaml": analysis}, {"ptsip.yaml": proposals})
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)

    proof = build_authorization_proof(
        bound,
        decision_ids=("decision:core",),
        authority_revision="owner:test",
    )
    authorized = authorize_execution(bound, proof, ledger)
    verified = verify_source_preconditions(root, authorized, 0, ledger)
    applied = apply_required_deltas(root, verified, ledger)

    assert applied.applied_delta_ids
    assert ledger.records[-1].phase is ExecutionPhase.FINAL_POINT_APPLIED


def test_stale_repository_is_rejected_before_apply(tmp_path: Path) -> None:
    root, _source, analysis, proposals, plan = _build_fixture(tmp_path)
    bound = bind_execution_plan(root, plan, {"ptsip.yaml": analysis}, {"ptsip.yaml": proposals})
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("decision:core",),
        authority_revision="owner:test",
    )
    authorized = authorize_execution(bound, proof, ledger)

    (root / "src/a.py").write_text("VALUE = 2\n", encoding="utf-8")

    with pytest.raises(ExecutionStateError, match="Repository changed"):
        verify_source_preconditions(root, authorized, 0, ledger)


def test_complete_source_requires_completion_proof(tmp_path: Path) -> None:
    root, _source, analysis, proposals, plan = _build_fixture(tmp_path)
    bound = bind_execution_plan(root, plan, {"ptsip.yaml": analysis}, {"ptsip.yaml": proposals})
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("decision:core",),
        authority_revision="owner:test",
    )
    authorized = authorize_execution(bound, proof, ledger)
    verified = verify_source_preconditions(root, authorized, 0, ledger)
    applied = apply_required_deltas(root, verified, ledger)

    def completion(source_path: str, _final_sha: str, analysis_digest: str) -> SourceCompletionProof:
        return SourceCompletionProof(
            source_path=source_path,
            analysis_digest=analysis_digest,
            required_total=1,
            required_unresolved=0,
            target_valid=True,
            evidence=("fixture",),
        )

    reanalyzed = reanalyze_source(applied, completion, ledger)
    completed = complete_source(reanalyzed, ledger)
    assert completed.reanalyzed.proof.complete


def test_promotion_requires_exact_final_state_and_preserves_audit(tmp_path: Path) -> None:
    root, _source, analysis, proposals, plan = _build_fixture(tmp_path)
    bound = bind_execution_plan(root, plan, {"ptsip.yaml": analysis}, {"ptsip.yaml": proposals})
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("decision:core",),
        authority_revision="owner:test",
    )
    authorized = authorize_execution(bound, proof, ledger)
    verified = verify_source_preconditions(root, authorized, 0, ledger)
    applied = apply_required_deltas(root, verified, ledger)

    def completion(source_path: str, _final_sha: str, analysis_digest: str) -> SourceCompletionProof:
        return SourceCompletionProof(
            source_path=source_path,
            analysis_digest=analysis_digest,
            required_total=1,
            required_unresolved=0,
            target_valid=True,
            evidence=("fixture",),
        )

    reanalyzed = reanalyze_source(applied, completion, ledger)
    completed = complete_source(reanalyzed, ledger)
    canonical_complete = finalize_source(root, completed, ledger)
    ready = prepare_promotion(root, canonical_complete, ledger)
    promoted = promote_canonical(root, ready, ledger)
    post = verify_post_promotion(root, promoted, ledger)

    assert isinstance(post, PostPromotionVerifiedState)
    assert not (root / "ptsip_0.3.7.yaml").exists()
    canonical = yaml.safe_load((root / "ptsip.yaml").read_text(encoding="utf-8"))
    assert canonical["ptsip"]["version"] == "0.3.7-draft"
    assert ledger.verify_integrity()


def test_ledger_tampering_is_detected(tmp_path: Path) -> None:
    root, _source, analysis, proposals, plan = _build_fixture(tmp_path)
    bound = bind_execution_plan(root, plan, {"ptsip.yaml": analysis}, {"ptsip.yaml": proposals})
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("decision:core",),
        authority_revision="owner:test",
    )
    authorize_execution(bound, proof, ledger)

    first = ledger.path.read_text(encoding="utf-8")
    payload = json.loads(first.splitlines()[0])
    payload["payload"] = {"tampered": True}
    ledger.path.write_text(json.dumps(payload, sort_keys=True) + "\n", encoding="utf-8")

    with pytest.raises(LedgerIntegrityError):
        ledger.load()


def test_recovery_inspection_requires_safe_resume_boundary(tmp_path: Path) -> None:
    root, _source, analysis, proposals, plan = _build_fixture(tmp_path)
    bound = bind_execution_plan(root, plan, {"ptsip.yaml": analysis}, {"ptsip.yaml": proposals})
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=("decision:core",),
        authority_revision="owner:test",
    )
    authorize_execution(bound, proof, ledger)

    recovery = inspect_recovery(root, bound, ledger)
    assert recovery.safe_to_resume
    assert recovery.next_source_index == 0
