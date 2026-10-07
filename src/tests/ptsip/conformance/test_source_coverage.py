from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.conformance_engine import evaluate_conformance

from _conformance_support import bind_artifact, blocking_gaps, clean_repo, commit


@pytest.mark.parametrize("suffix", [".rs", ".java"])
def test_product_owned_unsupported_source_is_incomplete(tmp_path: Path, suffix: str) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    source = repo / "product" / f"main{suffix}"
    source.write_text("source\n", encoding="utf-8")
    revision = commit(repo)
    bind_artifact(artifact, repo, revision)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert any(
        item["id"] == "language-coverage:unsupported-mandatory-source"
        for item in blocking_gaps(result)
    )


def test_unrelated_markdown_does_not_block_source_coverage(tmp_path: Path) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    (repo / "notes.md").write_text("documentation\n", encoding="utf-8")
    revision = commit(repo)
    bind_artifact(artifact, repo, revision)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT", blocking_gaps(result)


@pytest.mark.parametrize("suffix", [".cs", ".mts", ".cts"])
def test_unassigned_supported_source_is_blocking(tmp_path: Path, suffix: str) -> None:
    repo, artifact, _revision = clean_repo(tmp_path)
    (repo / f"unassigned{suffix}").write_text("// source\n", encoding="utf-8")
    revision = commit(repo)
    bind_artifact(artifact, repo, revision)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    gap = next(
        item
        for item in blocking_gaps(result)
        if item["id"] == "component-ownership:unassigned-relevant-files"
    )
    assert f"unassigned:unassigned{suffix}" in gap["evidence_ids"]
