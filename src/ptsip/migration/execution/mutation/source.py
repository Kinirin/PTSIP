from __future__ import annotations

from typing import Callable

from ptsip.migration.execution.error import ExecutionStateError
from ptsip.migration.execution.ledger.store import CheckpointLedger
from ptsip.migration.execution.state.source_steps import (
    AppliedSourceStep,
    CompletedSourceStep,
    ExecutionPhase,
    ReanalyzedSourceStep,
    SourceCompletionProof,
)

CompletionCallback = Callable[[str, str, str], SourceCompletionProof]

def reanalyze_source(
    applied: AppliedSourceStep,
    completion_callback: CompletionCallback,
    ledger: CheckpointLedger,
) -> ReanalyzedSourceStep:
    proof = completion_callback(
        applied.verified.source.source_path,
        applied.final_point_after_sha256,
        applied.verified.source.analysis_digest,
    )
    if proof.source_path != applied.verified.source.source_path:
        raise ExecutionStateError("Post-apply completion proof is bound to another source.")
    if proof.analysis_digest != applied.verified.source.analysis_digest:
        raise ExecutionStateError("Post-apply completion proof analysis digest does not match the bound source analysis.")
    ledger.append(
        phase=ExecutionPhase.SOURCE_REANALYZED,
        source_path=proof.source_path,
        source_sha256=applied.verified.source.source_content_sha256,
        final_point_after_sha256=applied.final_point_after_sha256,
        analysis_digest=proof.analysis_digest,
        decision_ids=applied.verified.source.decision_ids,
        payload=proof.as_dict(),
    )
    return ReanalyzedSourceStep(applied, proof)

def complete_source(reanalyzed: ReanalyzedSourceStep, ledger: CheckpointLedger) -> CompletedSourceStep:
    if not reanalyzed.proof.complete:
        raise ExecutionStateError("Source Required Work Elements are not complete after apply.")
    completed = CompletedSourceStep(reanalyzed)
    source = reanalyzed.applied.verified.source
    ledger.append(
        phase=ExecutionPhase.SOURCE_COMPLETE,
        source_path=source.source_path,
        source_sha256=source.source_content_sha256,
        final_point_after_sha256=reanalyzed.applied.final_point_after_sha256,
        analysis_digest=reanalyzed.proof.analysis_digest,
        decision_ids=source.decision_ids,
        payload=reanalyzed.proof.as_dict(),
    )
    return completed

