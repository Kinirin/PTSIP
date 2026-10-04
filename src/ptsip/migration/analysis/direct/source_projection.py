from __future__ import annotations

from pathlib import Path

from ptsip.repository.profile_convergence import DirectConvergenceState
from ptsip.repository.profile_transition import DraftVersion, ProfileGenerationIdentity
from ptsip.source_compat import CompatibilitySourceProfile, read_source_profile
from ptsip.migration.analysis.direct.result import DirectConvergenceAnalysisIssue

def _legacy_generation(state: DirectConvergenceState) -> ProfileGenerationIdentity:
    parsed = DraftVersion.from_draft_label(state.source.declared_version)
    if parsed is None:
        raise ValueError(
            f"Direct historical source {state.source.declared_version!r} is not a legacy draft identity."
        )
    return ProfileGenerationIdentity(
        path=state.source.path,
        version=parsed,
        declared_version=state.source.declared_version,
        specification_revision=state.source.specification_revision,
        specification_source=state.source.specification_source,
        content_sha256=state.source.content_sha256,
        temporary=state.source.temporary,
    )

def _read_historical_source(
    repository_root: Path,
    state: DirectConvergenceState,
) -> tuple[CompatibilitySourceProfile | None, tuple[DirectConvergenceAnalysisIssue, ...]]:
    try:
        generation = _legacy_generation(state)
    except ValueError as exc:
        return None, (DirectConvergenceAnalysisIssue("PP_DIRECT_SOURCE_IDENTITY_INVALID", str(exc)),)

    result = read_source_profile(repository_root, generation)
    if result.profile is None or result.issues:
        return None, tuple(
            DirectConvergenceAnalysisIssue(item.code, item.message)
            for item in result.issues
        )
    return result.profile, ()

