from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.conformance_engine import evaluate_conformance

from _conformance_support import bind_artifact, blocking_gaps, clean_repo


def test_exact_artifact_revision_binding_is_usable(tmp_path: Path) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT"
    assert result.report["evaluators"]["artifact_snapshot_binding"]["status"] == "RAN"


def test_artifact_binding_rejects_dirty_tracked_content_drift(tmp_path: Path) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    (repo / "product" / "app.py").write_text("VALUE = 42\n", encoding="utf-8")
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert any(
        "tracked-content fingerprint" in item["message"]
        for item in blocking_gaps(result)
    )


@pytest.mark.parametrize("binding_state", ["stale", "missing"])
def test_stale_or_missing_artifact_binding_is_incomplete(
    tmp_path: Path,
    binding_state: str,
) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    binding = Path(str(artifact) + ".binding.json")
    if binding_state == "stale":
        bind_artifact(artifact, repo, "deadbeef")
    else:
        binding.unlink()
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert result.report["evaluators"]["artifact_snapshot_binding"]["status"] == "BLOCKED"
