from __future__ import annotations

from pathlib import Path

from ptsip.lifecycle_evidence import evaluate_lifecycle_evidence
from ptsip.validation.components import partition_components

from _conformance_support import clean_repo, commit, components, write_workflow


def test_scoped_release_workflows_are_sufficient(tmp_path: Path) -> None:
    repo, _artifact_path, _revision = clean_repo(tmp_path)
    declared = components()
    result = evaluate_lifecycle_evidence(
        repo,
        declared,
        partition_components(repo, declared),
    )
    assert result.status == "RAN"


def test_unscoped_release_workflow_is_observation_not_lifecycle_authority(
    tmp_path: Path,
) -> None:
    repo, _artifact_path, _revision = clean_repo(tmp_path)
    write_workflow(repo, "global-release.yml", None)
    commit(repo)
    declared = components()
    result = evaluate_lifecycle_evidence(
        repo,
        declared,
        partition_components(repo, declared),
    )
    assert result.status == "RAN"
    assert not result.blocking_gaps
    workflow = next(item for item in result.workflows if item.path.endswith("global-release.yml"))
    assert workflow.trigger_paths == ()
    assert workflow.scope_complete is False
    assert workflow.reason is not None
    assert "not a classification failure" in workflow.reason


def test_global_non_release_ci_is_not_lifecycle_blocker(tmp_path: Path) -> None:
    repo, _artifact_path, _revision = clean_repo(tmp_path)
    write_workflow(repo, "ci.yml", None, release=False)
    commit(repo)
    declared = components()
    result = evaluate_lifecycle_evidence(
        repo,
        declared,
        partition_components(repo, declared),
    )
    assert result.status == "RAN"
