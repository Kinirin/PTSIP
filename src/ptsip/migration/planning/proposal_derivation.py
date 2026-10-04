from __future__ import annotations

from typing import Iterable

from ptsip.migration.analysis.findings.architecture import ArchitectureFindingKind
from ptsip.migration.analysis.result import MigrationAnalysis
from ptsip.migration.proposal.bundle import AcceptedDeltaBundle, ProposalBundle, ProposalPurpose, UnresolvedBundle
from ptsip.migration.proposal.source_set import SourceProposalSet

def _bundle_trace_matches(source, analysis_digest: str, bundle) -> bool:
    binding = bundle.source_generation
    return (
        binding.profile_path == source.profile_path
        and binding.declared_version == source.declared_version
        and binding.specification_revision == source.specification_revision
        and binding.content_sha256 == source.content_sha256
        and bundle.analysis_digest == analysis_digest
    )

def derive_source_proposals(
    analysis: MigrationAnalysis,
    *,
    proposed: Iterable[ProposalBundle] = (),
    accepted: Iterable[AcceptedDeltaBundle] = (),
    requested_async_ids: Iterable[str] = (),
) -> SourceProposalSet:
    issues: list[str] = []
    if not analysis.valid:
        issues.append("Migration analysis is not valid; proposal derivation is fail-closed.")

    good_proposals: list[ProposalBundle] = []
    for bundle in sorted(proposed, key=lambda item: item.id):
        if _bundle_trace_matches(analysis.source_generation, analysis.deterministic_digest, bundle):
            good_proposals.append(bundle)
        else:
            issues.append(f"Proposal {bundle.id} is not bound to this source analysis.")

    good_accepted: list[AcceptedDeltaBundle] = []
    for bundle in sorted(accepted, key=lambda item: item.proposal_id):
        if _bundle_trace_matches(analysis.source_generation, analysis.deterministic_digest, bundle):
            good_accepted.append(bundle)
        else:
            issues.append(f"Accepted bundle {bundle.proposal_id} is not bound to this source analysis.")

    accepted_ids = {item.proposal_id for item in good_accepted}
    good_proposals = [item for item in good_proposals if item.id not in accepted_ids]

    covered_subjects: set[str] = set()
    for bundle in [*good_proposals, *good_accepted]:
        for delta in bundle.deltas:
            covered_subjects.update(delta.obligation_ids)

    no_change: list[str] = []
    unresolved: list[UnresolvedBundle] = []
    for item in analysis.required:
        if item.resolved:
            no_change.append(item.id)
            continue
        if item.id in covered_subjects:
            continue
        unresolved.append(
            UnresolvedBundle.build(
                source_generation=analysis.source_generation,
                analysis_digest=analysis.deterministic_digest,
                subject_ids=(item.id,),
                purpose=(ProposalPurpose.REQUIRED_MIGRATION,),
                question=(
                    f"Required obligation {item.id} needs an explicit target delta or a project-owned "
                    f"no-change resolution; target status is {item.target_status.value}."
                ),
            )
        )

    for finding in analysis.architecture_findings:
        if finding.subject_id in covered_subjects:
            continue
        if finding.kind in {
            ArchitectureFindingKind.MISSING_RELATIONSHIP,
            ArchitectureFindingKind.MISSING_ASSOCIATED_ARTIFACT,
        }:
            unresolved.append(
                UnresolvedBundle.build(
                    source_generation=analysis.source_generation,
                    analysis_digest=analysis.deterministic_digest,
                    subject_ids=(finding.subject_id,),
                    purpose=(ProposalPurpose.TARGET_VALIDITY,),
                    question=(
                        f"{finding.kind.value} requires an explicit target proposal before the Final Point "
                        "can claim semantic convergence."
                    ),
                )
            )
        elif finding.kind in {
            ArchitectureFindingKind.EVIDENCE_CONFLICT,
            ArchitectureFindingKind.EVIDENCE_INCOMPLETE,
        }:
            unresolved.append(
                UnresolvedBundle.build(
                    source_generation=analysis.source_generation,
                    analysis_digest=analysis.deterministic_digest,
                    subject_ids=(finding.subject_id,),
                    purpose=(ProposalPurpose.ADVISORY,),
                    question=(
                        f"{finding.kind.value} remains reviewable evidence context and is not architecture authority."
                    ),
                )
            )

    requested = set(requested_async_ids)
    known_async = {item.id for item in analysis.async_targets}
    for item in sorted(requested - known_async):
        issues.append(f"Requested Async target {item} is not present in this source analysis.")
    for item in sorted(requested & known_async):
        if item in covered_subjects:
            continue
        unresolved.append(
            UnresolvedBundle.build(
                source_generation=analysis.source_generation,
                analysis_digest=analysis.deterministic_digest,
                subject_ids=(item,),
                purpose=(ProposalPurpose.ASYNC_OPTIONAL,),
                question=f"Async target {item} was requested but has no explicit target delta yet.",
            )
        )

    ignored_async = tuple(sorted(known_async - requested))
    return SourceProposalSet(
        source_generation=analysis.source_generation,
        analysis_digest=analysis.deterministic_digest,
        suggested=tuple(good_proposals),
        accepted=tuple(good_accepted),
        unresolved=tuple(sorted(unresolved, key=lambda item: item.id)),
        no_change_obligation_ids=tuple(sorted(no_change)),
        ignored_async_ids=ignored_async,
        issues=tuple(sorted(set(issues))),
    )

