from __future__ import annotations

from pathlib import Path

import yaml

from ptsip.repository.profile_transition import (
    discover_profile_transition,
    validate_transition_snapshot,
)


def _write_profile(
    root: Path,
    filename: str,
    version: str,
    *,
    revision: str = "a" * 40,
) -> Path:
    path = root / filename
    path.write_text(
        yaml.safe_dump(
            {
                "ptsip": {
                    "version": version,
                    "specification": {
                        "source": "https://github.com/Kinirin/PTSIP",
                        "revision": revision,
                    },
                }
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )
    return path


def _codes(result) -> set[str]:
    return {item.code for item in result.diagnostics}


def test_discovery_without_temporary_profile_is_idle(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is True
    assert result.state is not None
    assert result.state.mode == "IDLE"
    assert result.state.final_point is None
    assert result.state.temporary_profiles == ()
    assert result.state.ordered_sources == ()


def test_simple_transition_keeps_canonical_as_only_source(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft", revision="6" * 40)
    _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.3.7-draft", revision="7" * 40)

    result = discover_profile_transition(tmp_path)

    assert result.valid is True
    assert result.state is not None
    assert result.state.mode == "SIMPLE"
    assert result.state.final_point is not None
    assert result.state.final_point.path == "ptsip_0.3.7.yaml"
    assert [item.path for item in result.state.ordered_sources] == ["ptsip.yaml"]
    assert result.state.canonical_source.specification_revision == "6" * 40
    assert result.state.final_point.specification_revision == "7" * 40


def test_sequential_transition_orders_nearest_temporary_first_and_canonical_last(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.4-draft")
    _write_profile(tmp_path, "ptsip_0.3.6.yaml", "0.3.6-draft")
    _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.3.7-draft")
    _write_profile(tmp_path, "ptsip_0.4.0.yaml", "0.4.0-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is True
    assert result.state is not None
    assert result.state.mode == "SEQUENTIAL"
    assert result.state.final_point is not None
    assert result.state.final_point.path == "ptsip_0.4.0.yaml"
    assert [item.path for item in result.state.ordered_sources] == [
        "ptsip_0.3.7.yaml",
        "ptsip_0.3.6.yaml",
        "ptsip.yaml",
    ]


def test_filename_and_internal_version_mismatch_fails_closed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft")
    _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.4.0-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert "PROFILE_VERSION_FILENAME_MISMATCH" in _codes(result)


def test_similarly_named_invalid_temporary_profile_is_not_silently_adopted(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft")
    _write_profile(tmp_path, "ptsip_latest.yaml", "0.3.7-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert "INVALID_TEMPORARY_FILENAME" in _codes(result)


def test_missing_profile_identity_fields_fail_closed(tmp_path: Path) -> None:
    (tmp_path / "ptsip.yaml").write_text(
        "ptsip:\n  specification:\n    source: https://github.com/Kinirin/PTSIP\n",
        encoding="utf-8",
    )

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert {"MISSING_PROFILE_VERSION", "MISSING_SPEC_REVISION"} <= _codes(result)


def test_duplicate_semantic_target_identity_fails_closed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft")
    _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.3.7-draft")
    _write_profile(tmp_path, "ptsip_00.3.7.yaml", "00.3.7-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert "DUPLICATE_TARGET_IDENTITY" in _codes(result)
    assert "AMBIGUOUS_FINAL_POINT" in _codes(result)


def test_non_monotonic_temporary_target_fails_closed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft")
    _write_profile(tmp_path, "ptsip_0.3.5.yaml", "0.3.5-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert "NON_MONOTONIC_TARGET" in _codes(result)


def test_no_canonical_profile_fails_closed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.3.7-draft")

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert "NO_CANONICAL_PROFILE" in _codes(result)


def test_transition_snapshot_detects_profile_content_change(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft")
    target = _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.3.7-draft")
    result = discover_profile_transition(tmp_path)
    assert result.valid is True
    assert result.state is not None
    assert validate_transition_snapshot(tmp_path, result.state) == ()

    target.write_text(target.read_text(encoding="utf-8") + "# changed\n", encoding="utf-8")

    diagnostics = validate_transition_snapshot(tmp_path, result.state)
    assert [item.code for item in diagnostics] == ["STALE_TRANSITION_SNAPSHOT"]
    assert "ptsip_0.3.7.yaml: profile content changed" in diagnostics[0].message


def test_transition_snapshot_is_bound_to_repository_root(tmp_path: Path) -> None:
    source = tmp_path / "source"
    other = tmp_path / "other"
    source.mkdir()
    other.mkdir()
    _write_profile(source, "ptsip.yaml", "0.3.6-draft")
    _write_profile(other, "ptsip.yaml", "0.3.6-draft")
    result = discover_profile_transition(source)
    assert result.valid is True
    assert result.state is not None

    diagnostics = validate_transition_snapshot(other, result.state)

    assert [item.code for item in diagnostics] == ["STALE_TRANSITION_SNAPSHOT"]
    assert "repository root changed" in diagnostics[0].message


def test_invalid_profile_yaml_fails_closed(tmp_path: Path) -> None:
    (tmp_path / "ptsip.yaml").write_text("ptsip: [\n", encoding="utf-8")

    result = discover_profile_transition(tmp_path)

    assert result.valid is False
    assert result.state is None
    assert "INVALID_PROFILE_YAML" in _codes(result)
