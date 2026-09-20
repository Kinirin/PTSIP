from __future__ import annotations

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
