from __future__ import annotations

from pathlib import Path

from ptsip.build_resolution import evaluate_independent_build_resolution
from ptsip.validation.components import partition_components

from _conformance_support import build_repo, commit


def test_owned_build_manifests_are_usable(tmp_path: Path) -> None:
    repo, declared = build_repo(tmp_path)
    result = evaluate_independent_build_resolution(
        repo,
        declared,
        partition_components(repo, declared),
    )
    assert result.status == "RAN"


def test_cross_plane_owned_manifest_is_not_usable_for_declaring_component(tmp_path: Path) -> None:
    repo, declared = build_repo(tmp_path)
    declared[0]["manifests"] = ["tools/requirements.txt"]
    result = evaluate_independent_build_resolution(
        repo,
        declared,
        partition_components(repo, declared),
    )
    assert result.status == "BLOCKED"
    assert any(
        "owned by component 'tools'" in item["message"]
        for item in result.blocking_gaps
    )


def test_unassigned_manifest_is_not_build_isolation_evidence(tmp_path: Path) -> None:
    repo, declared = build_repo(tmp_path)
    (repo / "requirements.txt").write_text("\n", encoding="utf-8")
    commit(repo)
    declared[0]["manifests"] = ["requirements.txt"]
    result = evaluate_independent_build_resolution(
        repo,
        declared,
        partition_components(repo, declared),
    )
    assert result.status == "BLOCKED"
    assert any(
        "outside declared component ownership" in item["message"]
        for item in result.blocking_gaps
    )
