from __future__ import annotations

from developer.automation.wu02_lane_validator import LANES, validate_lane_documents


def test_wu02_lane_documents_are_centrally_consistent() -> None:
    assert validate_lane_documents() == ()


def test_wu02_lane_paths_are_unique() -> None:
    paths = tuple(item["path"] for item in LANES.values())
    branches = tuple(item["branch"] for item in LANES.values())
    assert len(paths) == len(set(paths))
    assert len(branches) == len(set(branches))
