from __future__ import annotations

import json
from pathlib import Path

from ptsip.conformance_engine import evaluate_conformance

from _conformance_support import bind_artifact, commit, init_repo, python_clean


def test_agent_decision_conflict_never_overrides_profile(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    init_repo(repo)
    artifact = python_clean(repo)
    commit(repo)
    decision = tmp_path / "decision.json"
    decision.write_text(
        json.dumps(
            {
                "component_id": "product",
                "status": "RESOLVED",
                "classification": "DEVELOPMENT_TOOLING",
                "origin": "AGENT",
                "confidence": 0.9,
                "evidence_ids": ["agent:evidence:1"],
                "rationale": "Review fixture",
                "counter_evidence": [],
            }
        ),
        encoding="utf-8",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], agent_decision_paths=[decision])
    assert result.outcome == "INCOMPLETE"
    assert result.report["agent_decisions"]["affects_declared_classification"] is False
    assert any(
        item["id"].startswith("agent-decision:declaration-conflict:product")
        for item in result.report["coverage"]["blocking_gaps"]
    )


def test_stale_external_evidence_is_blocking(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    init_repo(repo)
    artifact = python_clean(repo)
    commit(repo)
    external = tmp_path / "external.json"
    external.write_text(
        json.dumps(
            {
                "format": "ptsip-external-evidence/v1",
                "producer": {"id": "fixture-validator", "version": "1"},
                "subject": {"repository": str(repo.resolve()), "revision": "deadbeef"},
                "evidence": [],
            }
        ),
        encoding="utf-8",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "INCOMPLETE"
    assert result.report["evaluators"]["external_evidence_import"]["status"] == "BLOCKED"
    assert any(
        item["id"].startswith("external-evidence:invalid:")
        for item in result.report["coverage"]["blocking_gaps"]
    )


def test_trusted_external_evidence_can_resolve_native_target_uncertainty(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    init_repo(repo)
    artifact = python_clean(repo)
    (repo / "product" / "app.py").write_text("import vendor_module\nVALUE = 1\n", encoding="utf-8")
    revision = commit(repo)
    bind_artifact(artifact, repo, revision)
    external = tmp_path / "external.json"
    external.write_text(
        json.dumps(
            {
                "format": "ptsip-external-evidence/v1",
                "producer": {"id": "fixture-validator", "version": "1"},
                "subject": {"repository": str(repo.resolve()), "revision": revision},
                "evidence": [
                    {
                        "kind": "dependency",
                        "evidence_id": "vendor-module-resolution",
                        "source": "product/app.py",
                        "target": "vendor_module",
                        "relationship_type": "IMPORTS",
                        "phase": "RUNTIME",
                        "resolution": "EXTERNAL",
                        "target_scope": "EXTERNAL_DEPENDENCY",
                        "provenance": "OBSERVED",
                    }
                ],
            }
        ),
        encoding="utf-8",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "CONFORMANT", result.report["coverage"]
    assert result.report["external_evidence"]["edge_count"] == 1
    assert result.report["audit"]["status"] == "PASS"
    assert not any(
        item["id"].startswith("dependency-target:python:product/app.py")
        for item in result.report["coverage"]["blocking_gaps"]
    )
