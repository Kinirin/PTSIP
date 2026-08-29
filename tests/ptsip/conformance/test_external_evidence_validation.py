from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.conformance_engine import evaluate_conformance

from _conformance_support import (
    blocking_gaps,
    clean_repo,
    write_external_evidence,
)


@pytest.mark.parametrize("provenance", ["INFERRED", "DECLARED"])
def test_untrusted_external_evidence_cannot_clear_native_uncertainty(
    tmp_path: Path,
    provenance: str,
) -> None:
    repo, artifact, revision = clean_repo(tmp_path, "import vendor_module\n")
    external = tmp_path / "external.json"
    write_external_evidence(
        external,
        repo,
        revision,
        [
            {
                "kind": "dependency",
                "evidence_id": "vendor",
                "source": "product/app.py",
                "target": "vendor_module",
                "relationship_type": "IMPORTS",
                "phase": "RUNTIME",
                "resolution": "EXTERNAL",
                "target_scope": "EXTERNAL_DEPENDENCY",
                "provenance": provenance,
            }
        ],
    )
    result = evaluate_conformance(
        repo,
        artifact_evidence_paths=[artifact],
        external_evidence_paths=[external],
    )
    assert result.outcome == "INCOMPLETE"
    assert any(
        item["id"].startswith("dependency-target:python:product/app.py")
        for item in blocking_gaps(result)
    )


def test_external_repository_mismatch_is_blocking(tmp_path: Path) -> None:
    repo, artifact, revision = clean_repo(tmp_path)
    external = tmp_path / "external.json"
    write_external_evidence(
        external,
        repo,
        revision,
        [],
        repository="other/repository",
    )
    result = evaluate_conformance(
        repo,
        artifact_evidence_paths=[artifact],
        external_evidence_paths=[external],
    )
    assert result.outcome == "INCOMPLETE"
    assert any(
        item["id"].startswith("external-evidence:invalid:")
        for item in blocking_gaps(result)
    )


def test_external_resolution_scope_pair_is_semantically_validated(tmp_path: Path) -> None:
    repo, artifact, revision = clean_repo(tmp_path)
    external = tmp_path / "external.json"
    write_external_evidence(
        external,
        repo,
        revision,
        [
            {
                "kind": "dependency",
                "evidence_id": "bad",
                "source": "product/app.py",
                "target": "vendor",
                "relationship_type": "IMPORTS",
                "phase": "RUNTIME",
                "resolution": "RESOLVED",
                "target_scope": "EXTERNAL_DEPENDENCY",
                "provenance": "OBSERVED",
            }
        ],
    )
    result = evaluate_conformance(
        repo,
        artifact_evidence_paths=[artifact],
        external_evidence_paths=[external],
    )
    assert result.outcome == "INCOMPLETE"
    assert any(
        "RESOLVED evidence requires PROJECT_COMPONENT" in item["message"]
        for item in blocking_gaps(result)
    )


def test_external_conflict_with_native_resolution_is_blocking(tmp_path: Path) -> None:
    repo, artifact, revision = clean_repo(tmp_path, "import tools.check\n")
    external = tmp_path / "external.json"
    write_external_evidence(
        external,
        repo,
        revision,
        [
            {
                "kind": "dependency",
                "evidence_id": "conflict",
                "source": "product/app.py",
                "target": "tools.check",
                "relationship_type": "IMPORTS",
                "phase": "UNKNOWN",
                "resolution": "RESOLVED",
                "target_scope": "PROJECT_COMPONENT",
                "resolved_path": "product/app.py",
                "provenance": "OBSERVED",
            }
        ],
    )
    result = evaluate_conformance(
        repo,
        artifact_evidence_paths=[artifact],
        external_evidence_paths=[external],
    )
    assert any(
        item["id"].startswith("external-evidence:conflict:")
        for item in blocking_gaps(result)
    )
