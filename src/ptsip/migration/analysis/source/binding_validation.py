from __future__ import annotations

import hashlib
from pathlib import Path

from ptsip.evidence.contract import NormalizedEvidenceSet
from ptsip.repository.profile_path import normalize_profile_path, profile_path_on_disk
from ptsip.source_compat.model import CompatibilitySourceProfile
from ptsip.migration.analysis.result import MigrationAnalysisIssue

def _validate_source_binding(repository_root: Path, profile: CompatibilitySourceProfile) -> tuple[MigrationAnalysisIssue, ...]:
    binding = profile.generation
    try:
        path = profile_path_on_disk(repository_root, normalize_profile_path(binding.profile_path))
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
    except (OSError, ValueError) as exc:
        return (MigrationAnalysisIssue("SOURCE_BINDING_INVALID", f"Unable to validate WU-04 source binding: {exc}"),)
    if digest != binding.content_sha256:
        return (MigrationAnalysisIssue("SOURCE_CONTENT_STALE", "Source profile bytes changed after WU-04 compatibility read."),)
    return ()

def _validate_evidence_context(profile: CompatibilitySourceProfile, evidence: NormalizedEvidenceSet, snapshot) -> tuple[MigrationAnalysisIssue, ...]:
    issues: list[MigrationAnalysisIssue] = []
    binding = evidence.context.source_generation
    source = profile.generation
    if binding is None:
        issues.append(MigrationAnalysisIssue("EVIDENCE_SOURCE_CONTEXT_MISSING", "Normalized evidence is not bound to a source profile generation."))
    elif (
        binding.profile_path != source.profile_path
        or binding.version != source.declared_version
        or binding.specification_revision != source.specification_revision
        or binding.content_sha256 != source.content_sha256
    ):
        issues.append(MigrationAnalysisIssue("EVIDENCE_SOURCE_CONTEXT_MISMATCH", "Normalized evidence source generation does not match the WU-04 source profile."))
    evidence_snapshot = evidence.context.snapshot
    if (
        evidence_snapshot.revision != snapshot.head
        or evidence_snapshot.status_fingerprint != snapshot.status_fingerprint
        or evidence_snapshot.tracked_content_fingerprint != snapshot.tracked_content_fingerprint
    ):
        issues.append(MigrationAnalysisIssue("EVIDENCE_SNAPSHOT_STALE", "Normalized evidence is not bound to the repository snapshot being analyzed."))
    return tuple(issues)

