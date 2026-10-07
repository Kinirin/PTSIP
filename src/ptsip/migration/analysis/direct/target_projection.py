from __future__ import annotations

import copy
from pathlib import Path

import yaml

from ptsip.model import Classification, ResponsibilityRelationshipType
from ptsip.repository.profile_convergence import DirectConvergenceState
from ptsip.repository.profile_path import profile_path_on_disk
from ptsip.migration.analysis.direct.result import DirectConvergenceAnalysisIssue
from ptsip.migration.analysis.target.semantics import TargetArchitectureState, TargetSemantics, target_state_from_mapping

def current_pp_target_semantics(state: DirectConvergenceState) -> TargetSemantics:
    """Project the WU-09 PP target identity into the existing WU-05 semantic vocabulary."""

    return TargetSemantics(
        draft_version=state.target_contract.canonical,
        classifications=tuple(item.value for item in Classification),
        relationship_types=tuple(item.value for item in ResponsibilityRelationshipType),
    )

def _project_existing_target_state(
    repository_root: Path,
    state: DirectConvergenceState,
) -> tuple[TargetArchitectureState | None, tuple[DirectConvergenceAnalysisIssue, ...]]:
    if state.target is None:
        return None, ()

    try:
        path = profile_path_on_disk(repository_root, state.target.path)
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, yaml.YAMLError, ValueError) as exc:
        return None, (
            DirectConvergenceAnalysisIssue(
                "PP_DIRECT_TARGET_READ_ERROR",
                f"Unable to read direct-convergence target state: {exc}",
            ),
        )
    if not isinstance(payload, dict):
        return None, (
            DirectConvergenceAnalysisIssue(
                "PP_DIRECT_TARGET_SHAPE_ERROR",
                "Direct-convergence target root must be a mapping.",
            ),
        )

    # A legacy alias is physical continuity, not target-contract authority.  Project
    # its already-accepted architecture under the logical PP target identity for
    # semantic comparison without rewriting repository bytes at analysis time.
    projected = copy.deepcopy(payload)
    ptsip = projected.get("ptsip")
    if not isinstance(ptsip, dict):
        return None, (
            DirectConvergenceAnalysisIssue(
                "PP_DIRECT_TARGET_IDENTITY_MISSING",
                "Direct-convergence target has no ptsip metadata.",
            ),
        )
    ptsip["version"] = state.target_contract.canonical

    try:
        return target_state_from_mapping(projected), ()
    except ValueError as exc:
        return None, (
            DirectConvergenceAnalysisIssue(
                "PP_DIRECT_TARGET_STATE_UNSUPPORTED",
                str(exc),
            ),
        )

