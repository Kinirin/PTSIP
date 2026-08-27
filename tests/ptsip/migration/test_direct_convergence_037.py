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
from ptsip.migration.direct_convergence import (
    analyze_direct_profile_convergence,
    current_pp_target_semantics,
)
from ptsip.profile_compatibility import V034_REVISION, V036_REVISION
from ptsip.repository.profile_convergence import discover_direct_profile_convergence
from ptsip.repository.snapshot import capture_snapshot


def _write(root: Path, name: str, payload: dict[str, object]) -> None:
    (root / name).write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def _profile_034(*, classification: str = "PRODUCT") -> dict[str, object]:
    is_toolchain = classification == "TOOLCHAIN"
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
                "classification": classification,
                "include": ["src/**"],
                "purpose": "runtime",
                "shipped": not is_toolchain,
                "runtime_required": not is_toolchain,
                "lifecycle_owner": "DEVELOPMENT_TOOLING" if is_toolchain else "PRODUCT",
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


def _profile_036(*, version: str = "0.3.6-draft") -> dict[str, object]:
    return {
        "ptsip": {
            "version": version,
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


def _evidence(root: Path, state) -> NormalizedEvidenceSet:
    snapshot = capture_snapshot(root)
    source = state.source
    return NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="direct-convergence",
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


def test_identity_only_bridge_does_not_invoke_semantic_migration(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_036())

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None

    analysis = analyze_direct_profile_convergence(tmp_path, discovery.state)

    assert analysis.valid
    assert analysis.identity_rewrite_required is True
    assert analysis.semantic_migration_required is False
    assert analysis.semantic_analysis is None
    assert analysis.semantic_obligation_count == 0
    assert analysis.target_contract == "pp.1.01"


def test_semantic_direct_convergence_requires_normalized_evidence(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_034())
    (tmp_path / "src").mkdir()
    (tmp_path / "src/a.py").write_text("a", encoding="utf-8")

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None

    analysis = analyze_direct_profile_convergence(tmp_path, discovery.state)

    assert not analysis.valid
    assert [item.code for item in analysis.issues] == ["PP_DIRECT_EVIDENCE_REQUIRED"]


def test_semantic_direct_convergence_binds_pp_target_not_tool_draft(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_034())
    (tmp_path / "src").mkdir()
    (tmp_path / "src/a.py").write_text("a", encoding="utf-8")

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None
    state = discovery.state

    semantics = current_pp_target_semantics(state)
    analysis = analyze_direct_profile_convergence(
        tmp_path,
        state,
        evidence=_evidence(tmp_path, state),
    )

    assert semantics.draft_version == "pp.1.01"
    assert analysis.target_contract == "pp.1.01"
    assert analysis.semantic_migration_required is True
    assert analysis.semantic_analysis is not None
    assert "TARGET_DRAFT_MISMATCH" not in {item.code for item in analysis.semantic_analysis.issues}
    assert analysis.deterministic_digest


def test_legacy_target_alias_is_projected_as_logical_pp_target_for_analysis(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_034())
    _write(tmp_path, "ptsip_0.3.6.yaml", _profile_036())
    (tmp_path / "src").mkdir()
    (tmp_path / "src/a.py").write_text("a", encoding="utf-8")

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None
    state = discovery.state
    assert state.target_is_legacy_alias is True

    analysis = analyze_direct_profile_convergence(
        tmp_path,
        state,
        evidence=_evidence(tmp_path, state),
    )

    assert analysis.semantic_analysis is not None
    assert analysis.target_is_legacy_alias is True
    assert analysis.target_contract == "pp.1.01"
    assert "TARGET_DRAFT_MISMATCH" not in {item.code for item in analysis.semantic_analysis.issues}
    assert analysis.semantic_analysis.required[0].resolved is True


def test_toolchain_ambiguity_survives_direct_convergence(tmp_path: Path) -> None:
    _write(tmp_path, "ptsip.yaml", _profile_034(classification="TOOLCHAIN"))
    (tmp_path / "src").mkdir()
    (tmp_path / "src/a.py").write_text("a", encoding="utf-8")

    discovery = discover_direct_profile_convergence(tmp_path)
    assert discovery.valid and discovery.state is not None
    state = discovery.state

    analysis = analyze_direct_profile_convergence(
        tmp_path,
        state,
        evidence=_evidence(tmp_path, state),
    )

    assert analysis.semantic_analysis is not None
    required = analysis.semantic_analysis.required[0]
    assert required.resolved is False
    assert required.source_classification == "TOOLCHAIN"
    assert any(
        item.kind.value == "HISTORICAL_TOOLCHAIN_AMBIGUITY"
        for item in analysis.semantic_analysis.lifecycle_findings
    )
