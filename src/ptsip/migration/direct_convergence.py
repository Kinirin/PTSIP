from __future__ import annotations

import copy
import hashlib
import json
from dataclasses import dataclass
from pathlib import Path

import yaml

from ..evidence.contract import NormalizedEvidenceSet
from ..model import Classification, ResponsibilityRelationshipType
from ..repository.profile_convergence import (
    DirectConvergenceMode,
    DirectConvergenceState,
    validate_direct_convergence_snapshot,
)
from ..repository.profile_path import profile_path_on_disk
from ..repository.profile_transition import DraftVersion, ProfileGenerationIdentity
from ..source_compat import CompatibilitySourceProfile, read_source_profile
from .analyzer import analyze_source_migration, target_state_from_mapping
from .model import MigrationAnalysis, TargetArchitectureState, TargetSemantics


@dataclass(frozen=True)
class DirectConvergenceAnalysisIssue:
    code: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return {"code": self.code, "message": self.message}


@dataclass(frozen=True)
class DirectConvergenceAnalysis:
    mode: DirectConvergenceMode
    source_path: str
    source_declared_version: str
    source_compatibility_contract: str
    target_contract: str
    target_path: str
    target_is_legacy_alias: bool
    identity_rewrite_required: bool
    semantic_analysis: MigrationAnalysis | None
    issues: tuple[DirectConvergenceAnalysisIssue, ...]

    @property
    def semantic_migration_required(self) -> bool:
        return self.mode is DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION

    @property
    def semantic_obligation_count(self) -> int:
        if self.semantic_analysis is None:
            return 0
        return (
            len(self.semantic_analysis.required)
            + len(self.semantic_analysis.removals)
            + len(self.semantic_analysis.async_targets)
        )

    @property
    def valid(self) -> bool:
        return not self.issues and (
            self.semantic_analysis is None or self.semantic_analysis.valid
        )

    def content_payload(self) -> dict[str, object]:
        return {
            "mode": self.mode.value,
            "source_path": self.source_path,
            "source_declared_version": self.source_declared_version,
            "source_compatibility_contract": self.source_compatibility_contract,
            "target_contract": self.target_contract,
            "target_path": self.target_path,
            "target_is_legacy_alias": self.target_is_legacy_alias,
            "identity_rewrite_required": self.identity_rewrite_required,
            "semantic_migration_required": self.semantic_migration_required,
            "semantic_analysis_digest": (
                self.semantic_analysis.deterministic_digest
                if self.semantic_analysis is not None
                else None
            ),
            "issues": [item.as_dict() for item in self.issues],
        }

    @property
    def deterministic_digest(self) -> str:
        raw = json.dumps(
            self.content_payload(),
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
        ).encode("utf-8")
        return hashlib.sha256(raw).hexdigest()

    def as_dict(self) -> dict[str, object]:
        payload = self.content_payload()
        payload["valid"] = self.valid
        payload["semantic_obligation_count"] = self.semantic_obligation_count
        payload["semantic_analysis"] = (
            self.semantic_analysis.as_dict()
            if self.semantic_analysis is not None
            else None
        )
        payload["deterministic_digest"] = self.deterministic_digest
        return payload


def current_pp_target_semantics(state: DirectConvergenceState) -> TargetSemantics:
    """Project the WU-09 PP target identity into the existing WU-05 semantic vocabulary."""

    return TargetSemantics(
        draft_version=state.target_contract.canonical,
        classifications=tuple(item.value for item in Classification),
        relationship_types=tuple(item.value for item in ResponsibilityRelationshipType),
    )


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


def analyze_direct_profile_convergence(
    repository_root: str | Path,
    state: DirectConvergenceState,
    *,
    evidence: NormalizedEvidenceSet | None = None,
) -> DirectConvergenceAnalysis:
    root = Path(repository_root).expanduser().resolve()
    stale = validate_direct_convergence_snapshot(root, state)
    if stale:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=state.mode is DirectConvergenceMode.IDENTITY_ONLY,
            semantic_analysis=None,
            issues=tuple(
                DirectConvergenceAnalysisIssue(item.code, item.message)
                for item in stale
            ),
        )

    if state.mode is DirectConvergenceMode.CURRENT:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=False,
            semantic_analysis=None,
            issues=(),
        )

    source_profile, source_issues = _read_historical_source(root, state)
    if source_profile is None:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=state.mode is DirectConvergenceMode.IDENTITY_ONLY,
            semantic_analysis=None,
            issues=source_issues,
        )

    if state.mode is DirectConvergenceMode.IDENTITY_ONLY:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=True,
            semantic_analysis=None,
            issues=(),
        )

    if evidence is None:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=False,
            semantic_analysis=None,
            issues=(
                DirectConvergenceAnalysisIssue(
                    "PP_DIRECT_EVIDENCE_REQUIRED",
                    "Semantic direct convergence requires normalized evidence bound to the actual historical source.",
                ),
            ),
        )

    target_state, target_issues = _project_existing_target_state(root, state)
    if target_issues:
        return DirectConvergenceAnalysis(
            mode=state.mode,
            source_path=state.source.path,
            source_declared_version=state.source.declared_version,
            source_compatibility_contract=state.source_compatibility_contract.canonical,
            target_contract=state.target_contract.canonical,
            target_path=state.target_path,
            target_is_legacy_alias=state.target_is_legacy_alias,
            identity_rewrite_required=False,
            semantic_analysis=None,
            issues=target_issues,
        )

    analysis = analyze_source_migration(
        root,
        source_profile,
        evidence,
        target_semantics=current_pp_target_semantics(state),
        target_state=target_state,
    )
    return DirectConvergenceAnalysis(
        mode=state.mode,
        source_path=state.source.path,
        source_declared_version=state.source.declared_version,
        source_compatibility_contract=state.source_compatibility_contract.canonical,
        target_contract=state.target_contract.canonical,
        target_path=state.target_path,
        target_is_legacy_alias=state.target_is_legacy_alias,
        identity_rewrite_required=False,
        semantic_analysis=analysis,
        issues=(),
    )
