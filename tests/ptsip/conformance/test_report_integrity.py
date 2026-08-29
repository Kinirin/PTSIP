from __future__ import annotations

from pathlib import Path

from ptsip.conformance_engine import evaluate_conformance

from _conformance_support import bind_artifact, commit, init_repo, python_clean


def test_clean_supported_report_passes_contract_audit(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    init_repo(repo)
    artifact = python_clean(repo)
    revision = commit(repo)
    bind_artifact(artifact, repo, revision)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT"
    assert result.report["audit"]["status"] == "PASS"
    assert result.report["audit"]["problem_count"] == 0
