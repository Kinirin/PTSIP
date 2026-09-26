from __future__ import annotations

from pathlib import Path

import developer.automation.wu02_lane_validator as lane_validator
from developer.automation.wu02_lane_validator import (
    CONTROL_BRANCH,
    LANES,
    validate_lane_documents,
)


def test_wu02_lane_documents_are_centrally_consistent() -> None:
    assert validate_lane_documents() == ()


def test_wu02_lane_paths_are_unique_and_lanes_use_control_plane() -> None:
    paths = tuple(item["path"] for item in LANES.values())
    branches = tuple(item["branch"] for item in LANES.values())
    assert len(paths) == len(set(paths))
    assert set(branches) == {CONTROL_BRANCH}


def test_explicit_execution_branch_overrides_detached_git(monkeypatch) -> None:
    monkeypatch.setattr(lane_validator, "_git", lambda _base, *_args: "")

    assert lane_validator._execution_branch(Path("."), CONTROL_BRANCH) == CONTROL_BRANCH


def test_detached_git_does_not_infer_github_ref_name(monkeypatch) -> None:
    monkeypatch.setattr(lane_validator, "_git", lambda _base, *_args: "")
    monkeypatch.setenv("GITHUB_REF_NAME", CONTROL_BRANCH)

    assert lane_validator._execution_branch(Path(".")) == ""


def test_control_branch_context_is_fail_closed() -> None:
    assert lane_validator.validate_control_branch_context(CONTROL_BRANCH) == ()
    assert lane_validator.validate_control_branch_context("") != ()
    assert lane_validator.validate_control_branch_context("main") != ()
