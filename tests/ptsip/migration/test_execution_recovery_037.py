from __future__ import annotations

import hashlib
from pathlib import Path
from types import SimpleNamespace

from ptsip.migration import CheckpointLedger, ExecutionPhase, capture_mutation_guard, inspect_recovery


def _sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def test_recovery_rejects_temporary_source_deleted_without_checkpoint(tmp_path: Path) -> None:
    repository = tmp_path / "repo"
    repository.mkdir()
    canonical = repository / "ptsip.yaml"
    temporary = repository / "ptsip_0.3.6.yaml"
    final_point = repository / "ptsip_0.4.0.yaml"
    canonical.write_text("canonical-source\n", encoding="utf-8")
    temporary.write_text("temporary-source\n", encoding="utf-8")
    final_point.write_text("final-point\n", encoding="utf-8")

    sources = (
        SimpleNamespace(source_path="ptsip_0.3.6.yaml", source_content_sha256=_sha256(temporary)),
        SimpleNamespace(source_path="ptsip.yaml", source_content_sha256=_sha256(canonical)),
    )
    final_sha = _sha256(final_point)
    plan_digest = "fixture-plan-digest"
    bound = SimpleNamespace(
        plan_digest=plan_digest,
        sources=sources,
        plan=SimpleNamespace(
            final_point=SimpleNamespace(path="ptsip_0.4.0.yaml", content_sha256=None),
        ),
        mutation_guard=capture_mutation_guard(
            repository,
            ("ptsip_0.3.6.yaml", "ptsip.yaml", "ptsip_0.4.0.yaml"),
        ),
    )
    ledger = CheckpointLedger(tmp_path / "ledger", plan_digest)
    ledger.append(phase=ExecutionPhase.PLAN_BOUND)
    ledger.append(phase=ExecutionPhase.AUTHORIZED)
    ledger.append(
        phase=ExecutionPhase.FINAL_POINT_APPLIED,
        source_path="ptsip_0.3.6.yaml",
        final_point_after_sha256=final_sha,
    )
    ledger.append(
        phase=ExecutionPhase.SOURCE_COMPLETE,
        source_path="ptsip_0.3.6.yaml",
        final_point_after_sha256=final_sha,
    )

    # Simulate a crash after unlink() succeeded but before SOURCE_REMOVED was persisted.
    temporary.unlink()

    inspection = inspect_recovery(repository, bound, ledger)

    assert not inspection.safe_to_resume
    assert inspection.next_source_index == 0
    assert "source disappeared without removal checkpoint: ptsip_0.3.6.yaml" in inspection.reasons
