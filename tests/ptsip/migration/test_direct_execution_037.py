from __future__ import annotations

import subprocess
from pathlib import Path

import yaml

from ptsip.evidence.contract import (
    EvidenceChannel,
    EvidenceChannelStatus,
    EvidenceEvaluationContext,
    NormalizedEvidenceSet,
    SnapshotBinding,
    SourceGenerationBinding as EvidenceSourceGenerationBinding,
)
from ptsip.migration.direct_convergence import analyze_direct_profile_convergence
from ptsip.migration.direct_execution import (
    build_legacy_target_identity_rewrite_plan,
    prepare_direct_promotion,
    verify_direct_post_promotion,
)
from ptsip.migration.direct_planner import build_direct_final_point_convergence_plan
from ptsip.migration.execution_apply import (
    apply_required_deltas,
    complete_source,
    finalize_source,
    reanalyze_source,
)
from ptsip.migration.execution_binding import (
    authorize_execution,
    bind_execution_plan,
    build_authorization_proof,
    inspect_recovery,
    verify_source_preconditions,
)
from ptsip.migration.execution_ledger import CheckpointLedger
from ptsip.migration.execution_model import PostPromotionVerifiedState, SourceCompletionProof
from ptsip.migration.execution_promotion import promote_canonical
from ptsip.migration.identity_rewrite import authorize_identity_rewrite, execute_identity_rewrite
from ptsip.migration.planner import derive_source_proposals, final_point_state_from_mapping
from ptsip.profile_compatibility import V034_REVISION, V036_REVISION
from ptsip.repository.profile_convergence import discover_direct_profile_convergence
from ptsip.repository.snapshot import capture_snapshot


SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"


def _git(root: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(root), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _write(root: Path, name: str, payload: dict[str, object]) -> Path:
    path = root / name
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8", newline="\n")
    return path


def _source_034() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {"source": SPEC_SOURCE, "revision": V034_REVISION},
        },
        "components": [
            {
                "id": "core",
                "classification": "PRODUCT",
                "include": ["src/**"],
                "purpose": "runtime",
                "shipped": True,
                "runtime_required": True,
                "lifecycle_owner": "PRODUCT",
                "executable": False,
                "analysis_inputs": [],
            }
        ],
        "policies": {
            "product_to_toolchain_runtime_dependency": "deny",
            "toolchain_in_product_package": "deny",
            "independent_build_resolution": "required",
            "shared_executable_cross_boundary": "deny",
            "neutral_contract_sharing": "allow",
        },
    }


def _target_036() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": SPEC_SOURCE, "revision": V036_REVISION},
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "core",
                "classification": "PRODUCT",
                "roles": ["IMPLEMENTATION"],
                "include": ["src/**"],
                "purpose": "runtime",
                "shipped": True,
                "runtime_required": True,
            }
        ],
        "associated_artifacts": [],
        "relationships": [],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
            "shared_executable_cross_lifecycle": "deny",
            "neutral_contract_sharing": "allow",
        },
    }


def _evidence(root: Path, state) -> NormalizedEvidenceSet:
    snapshot = capture_snapshot(root)
    source = state.source
    return NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="direct-execution",
            snapshot=SnapshotBinding(
                repository_root=str(root.resolve()),
                revision=snapshot.head,
                status_fingerprint=snapshot.status_fingerprint,
                tracked_content_fingerprint=snapshot.tracked_content_fingerprint,
            ),
            source_generation=EvidenceSourceGenerationBinding(
                profile_path=source.path,
                version=source.declared_version,
                specification_revision=source.specification_revision,
                content_sha256=source.content_sha256,
            ),
        ),
        records=(),
        channels=(EvidenceChannel("fixture", EvidenceChannelStatus.PRODUCED, 0),),
    )


def test_legacy_alias_runs_direct_pp_execution_and_atomic_promotion(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    root.mkdir()
    _git(root, "init")
    _git(root, "config", "user.name", "PTSIP Direct Execution Fixture")
    _git(root, "config", "user.email", "ptsip-direct@example.invalid")
    (root / "src").mkdir()
    (root / "src/a.py").write_text("VALUE = 1\n", encoding="utf-8")
    _write(root, "ptsip.yaml", _source_034())
    _write(root, "ptsip_0.3.6.yaml", _target_036())
    _git(root, "add", ".")
    _git(root, "commit", "-m", "direct convergence fixture baseline")

    discovery = discover_direct_profile_convergence(root)
    assert discovery.valid and discovery.state is not None
    state = discovery.state
    assert state.target_is_legacy_alias is True
    assert state.intermediate_profiles == ()

    # Preserve the accepted physical legacy target path, but normalize its
    # historical contract identity before binding semantic WU-07 execution.
    rewrite_plan = build_legacy_target_identity_rewrite_plan(state)
    rewrite_authorization = authorize_identity_rewrite(
        rewrite_plan,
        authority_revision="owner:legacy-target-pp101",
    )
    execute_identity_rewrite(root, rewrite_plan, rewrite_authorization)

    target_payload = yaml.safe_load((root / "ptsip_0.3.6.yaml").read_text(encoding="utf-8"))
    assert target_payload["ptsip"]["version"] == "pp.1.01"
    assert not (root / "ptsip_pp1.01.yaml").exists()

    discovery = discover_direct_profile_convergence(root)
    assert discovery.valid and discovery.state is not None
    state = discovery.state
    assert state.target_is_legacy_alias is True
    assert state.target is not None
    assert state.target.declared_version == "pp.1.01"

    direct = analyze_direct_profile_convergence(
        root,
        state,
        evidence=_evidence(root, state),
    )
    assert direct.valid and direct.semantic_analysis is not None
    analysis = direct.semantic_analysis
    assert analysis.completion.complete
    proposal_set = derive_source_proposals(analysis)
    final_state = final_point_state_from_mapping(
        target_payload,
        path=state.target.path,
        content_sha256=state.target.content_sha256,
    )
    plan = build_direct_final_point_convergence_plan(
        state,
        direct,
        proposal_set,
        target_specification_revision=V036_REVISION,
        final_point_state=final_state,
    )
    assert plan.preview.ready_for_wu07
    assert plan.preview.ordered_sources == ("ptsip.yaml",)
    assert plan.final_point.path == "ptsip_0.3.6.yaml"
    assert plan.final_point.draft_version == "pp.1.01"

    bound = bind_execution_plan(
        root,
        plan,
        {"ptsip.yaml": analysis},
        {"ptsip.yaml": proposal_set},
    )
    ledger = CheckpointLedger.for_repository(root, bound.plan_digest)
    proof = build_authorization_proof(
        bound,
        decision_ids=(),
        authority_revision="owner:direct-semantic-promotion",
    )
    authorized = authorize_execution(bound, proof, ledger)
    verified = verify_source_preconditions(root, authorized, 0, ledger)

    pre_apply_recovery = inspect_recovery(root, bound, ledger)
    assert pre_apply_recovery.safe_to_resume
    assert pre_apply_recovery.next_source_index == 0

    applied = apply_required_deltas(root, verified, ledger)

    def completion(source_path: str, _final_sha: str, analysis_digest: str) -> SourceCompletionProof:
        return SourceCompletionProof(
            source_path=source_path,
            analysis_digest=analysis_digest,
            required_total=analysis.completion.required_total,
            required_unresolved=0,
            target_valid=True,
            evidence=("direct-current-target-reanalysis",),
        )

    reanalyzed = reanalyze_source(applied, completion, ledger)
    completed = complete_source(reanalyzed, ledger)
    canonical_complete = finalize_source(root, completed, ledger)
    ready = prepare_direct_promotion(root, canonical_complete, ledger)
    promoted = promote_canonical(root, ready, ledger)
    verified_post = verify_direct_post_promotion(root, promoted, ledger)

    assert isinstance(verified_post, PostPromotionVerifiedState)
    assert not (root / "ptsip_0.3.6.yaml").exists()
    assert not (root / "ptsip_pp1.01.yaml").exists()
    canonical_payload = yaml.safe_load((root / "ptsip.yaml").read_text(encoding="utf-8"))
    assert canonical_payload["ptsip"]["version"] == "pp.1.01"

    final_recovery = inspect_recovery(root, bound, ledger)
    assert final_recovery.safe_to_resume
    assert final_recovery.next_source_index == 1
