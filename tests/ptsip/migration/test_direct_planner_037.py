from __future__ import annotations

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
from ptsip.migration.direct_planner import build_direct_final_point_convergence_plan
from ptsip.migration.planner import (
    FinalPointKind,
    derive_source_proposals,
    final_point_state_from_mapping,
)
from ptsip.profile_compatibility import V034_REVISION, V036_REVISION
from ptsip.repository.profile_convergence import discover_direct_profile_convergence
from ptsip.repository.snapshot import capture_snapshot


def _profile_034() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V034_REVISION,
            },
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


def _profile_036() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V036_REVISION,
            },
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


def _write(root: Path, name: str, payload: dict[str, object]) -> None:
    (root / name).write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def _evidence(root: Path, state) -> NormalizedEvidenceSet:
    snapshot = capture_snapshot(root)
    source = state.source
    return NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="direct-plan",
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


def _direct_analysis(root: Path, state):
    return analyze_direct_profile_convergence(
        root,
        state,
        evidence=_evidence(root, state),
    )


def test_planned_direct_final_point_skips_all_intermediate_versions(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_034())
    (tmp_path / "src").mkdir()
    (tmp_path / "src/a.py").write_text("a", encoding="utf-8")

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None
    state = discovery.state
    direct = _direct_analysis(tmp_path, state)
    assert direct.semantic_analysis is not None
    proposals = derive_source_proposals(direct.semantic_analysis)

    plan = build_direct_final_point_convergence_plan(
        state,
        direct,
        proposals,
        target_specification_revision=V036_REVISION,
    )

    assert plan.final_point.kind is FinalPointKind.PLANNED
    assert plan.final_point.path == "ptsip_pp1.01.yaml"
    assert plan.final_point.profile_contract == "pp.1.01"
    assert "profile_contract" in plan.final_point.as_dict()
    assert "draft_version" not in plan.final_point.as_dict()
    assert plan.preview.ordered_sources == ("ptsip.yaml",)
    assert plan.source_steps[0].next_source_path is None
    assert "ptsip_0.3.6.yaml" not in plan.preview.ordered_sources
    assert "ptsip_0.3.7.yaml" not in plan.preview.ordered_sources
    assert not plan.preview.ready_for_wu07


def test_existing_legacy_target_is_one_logical_pp_final_point(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_034())
    target_payload = _profile_036()
    _write(tmp_path, "ptsip_0.3.6.yaml", target_payload)
    (tmp_path / "src").mkdir()
    (tmp_path / "src/a.py").write_text("a", encoding="utf-8")

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None
    state = discovery.state
    assert state.target is not None
    direct = _direct_analysis(tmp_path, state)
    assert direct.semantic_analysis is not None
    assert direct.semantic_analysis.completion.complete
    proposals = derive_source_proposals(direct.semantic_analysis)
    final_state = final_point_state_from_mapping(
        target_payload,
        path="ptsip_0.3.6.yaml",
        content_sha256=state.target.content_sha256,
    )

    plan = build_direct_final_point_convergence_plan(
        state,
        direct,
        proposals,
        target_specification_revision=V036_REVISION,
        final_point_state=final_state,
    )

    assert not plan.issues
    assert plan.final_point.kind is FinalPointKind.EXISTING
    assert plan.final_point.path == "ptsip_0.3.6.yaml"
    assert plan.final_point.profile_contract == "pp.1.01"
    assert plan.preview.ordered_sources == ("ptsip.yaml",)
    assert plan.source_steps[0].next_source_path is None
    assert plan.preview.ready_for_wu07


def test_identity_only_bridge_is_rejected_by_semantic_direct_planner(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_036())

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None
    state = discovery.state
    direct = analyze_direct_profile_convergence(tmp_path, state)

    # There is deliberately no semantic proposal set for IDENTITY_ONLY.  Feed a
    # source-compatible empty set only to prove this planner fails closed before
    # it can synthesize a semantic migration path.
    from ptsip.migration.proposal import SourceProposalSet
    from ptsip.source_compat.model import SourceGenerationBinding

    source = state.source
    proposals = SourceProposalSet(
        source_generation=SourceGenerationBinding(
            profile_path=source.path,
            declared_version=source.declared_version,
            specification_revision=source.specification_revision,
            specification_source=source.specification_source,
            content_sha256=source.content_sha256,
            temporary=source.temporary,
        ),
        analysis_digest="<identity-only>",
        suggested=(),
        accepted=(),
        unresolved=(),
        no_change_obligation_ids=(),
        ignored_async_ids=(),
        issues=(),
    )

    plan = build_direct_final_point_convergence_plan(
        state,
        direct,
        proposals,
        target_specification_revision=V036_REVISION,
    )

    assert not plan.preview.ready_for_wu07
    assert "PP_DIRECT_SEMANTIC_PLAN_NOT_APPLICABLE" in {item.code for item in plan.issues}
