from __future__ import annotations

from pathlib import Path

import pytest

import ptsip.conformance_engine as engine
from ptsip.conformance_engine import evaluate_conformance

from _conformance_support import blocking_gaps, clean_repo


def test_final_snapshot_covers_late_lifecycle_evaluation(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    original = engine.evaluate_lifecycle_evidence

    def mutate_during_lifecycle(*args, **kwargs):
        result = original(*args, **kwargs)
        (repo / "product" / "app.py").write_text("VALUE = 99\n", encoding="utf-8")
        return result

    monkeypatch.setattr(engine, "evaluate_lifecycle_evidence", mutate_during_lifecycle)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert result.report["snapshot"]["comparison"]["stable"] is False
    assert any(item["id"] == "snapshot:invalidated" for item in blocking_gaps(result))
