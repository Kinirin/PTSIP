from __future__ import annotations

import hashlib
from pathlib import Path

from ptsip.migration.direct_convergence import DirectConvergenceAnalysis
from ptsip.migration.direct_planner import build_direct_final_point_convergence_plan
from ptsip.migration.model import MigrationAnalysis, SourceMigrationCompletion
from ptsip.migration.proposal import SourceProposalSet
from ptsip.profile_identity import ProjectProfileTransitionKind, ProjectProfileVersion
from ptsip.repository.profile_convergence import (
    ConvergenceProfileBinding,
    DirectConvergenceMode,
    DirectConvergenceState,
)
from ptsip.repository.snapshot import capture_snapshot
from ptsip.source_compat.model import SourceGenerationBinding


SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"


def test_future_target_shape_converges_directly_without_pp_history_replay(tmp_path: Path) -> None:
    source_path = tmp_path / "ptsip.yaml"
    source_path.write_text("ptsip:\n  version: 0.3.4-draft\n", encoding="utf-8")
    source_sha = hashlib.sha256(source_path.read_bytes()).hexdigest()
    snapshot = capture_snapshot(tmp_path)

    source = ConvergenceProfileBinding(
        path="ptsip.yaml",
        declared_version="0.3.4-draft",
        specification_revision="historical-revision",
        specification_source=SPEC_SOURCE,
        content_sha256=source_sha,
        temporary=False,
    )
    target = ProjectProfileVersion.parse("pp.2.02", require_canonical=True)
    state = DirectConvergenceState(
        mode=DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION,
        source=source,
        source_compatibility_contract=ProjectProfileVersion.parse(
            "pp.0.00", require_canonical=True
        ),
        target_contract=target,
        transition_kind=ProjectProfileTransitionKind.SEMANTIC_MIGRATION,
        target_path="ptsip_pp2.02.yaml",
        target=None,
        target_is_legacy_alias=False,
        requires_temporary_target=True,
        snapshot=snapshot,
    )
    binding = SourceGenerationBinding(
        profile_path="ptsip.yaml",
        declared_version="0.3.4-draft",
        specification_revision="historical-revision",
        specification_source=SPEC_SOURCE,
        content_sha256=source_sha,
        temporary=False,
    )
    analysis = MigrationAnalysis(
        source_generation=binding,
        repository_head=snapshot.head,
        repository_status_fingerprint=snapshot.status_fingerprint,
        repository_content_fingerprint=snapshot.tracked_content_fingerprint,
        required=(),
        removals=(),
        async_targets=(),
        ambiguous=(),
        lifecycle_findings=(),
        architecture_findings=(),
        issues=(),
        completion=SourceMigrationCompletion(
            required_total=0,
            required_resolved=0,
            required_unresolved=0,
            removal_count=0,
            async_count=0,
        ),
    )
    direct = DirectConvergenceAnalysis(
        mode=DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION,
        source_path="ptsip.yaml",
        source_declared_version="0.3.4-draft",
        source_compatibility_contract="pp.0.00",
        target_contract="pp.2.02",
        target_path="ptsip_pp2.02.yaml",
        target_is_legacy_alias=False,
        identity_rewrite_required=False,
        semantic_analysis=analysis,
        issues=(),
    )
    proposals = SourceProposalSet(
        source_generation=binding,
        analysis_digest=analysis.deterministic_digest,
        suggested=(),
        accepted=(),
        unresolved=(),
    )

    plan = build_direct_final_point_convergence_plan(
        state,
        direct,
        proposals,
        target_specification_revision="future-target-revision",
    )

    assert not plan.issues
    assert plan.preview.ready_for_wu07
    assert plan.final_point.path == "ptsip_pp2.02.yaml"
    assert plan.final_point.draft_version == "pp.2.02"
    assert plan.preview.ordered_sources == ("ptsip.yaml",)
    assert plan.source_steps[0].next_source_path is None
    assert "pp.1.01" not in str(plan.as_dict())
    assert "pp.1.02" not in str(plan.as_dict())
    assert "pp.2.01" not in str(plan.as_dict())
