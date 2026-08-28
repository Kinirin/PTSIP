from __future__ import annotations

from pathlib import Path

import yaml

from ptsip.profile_compatibility import V034_REVISION, V036_REVISION
from ptsip.repository.profile_convergence import (
    DirectConvergenceMode,
    discover_direct_profile_convergence,
    validate_direct_convergence_snapshot,
)


def _write_profile(
    root: Path,
    filename: str,
    version: str,
    revision: str,
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
                },
                "responsibility_map": {"mode": "explicit"},
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )
    return path


def _codes(result) -> set[str]:
    return {item.code for item in result.diagnostics}


def test_036_canonical_converges_in_place_as_identity_only(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.6-draft", V036_REVISION)

    result = discover_direct_profile_convergence(tmp_path)

    assert result.valid
    assert result.state is not None
    assert result.state.mode is DirectConvergenceMode.IDENTITY_ONLY
    assert result.state.target_contract.canonical == "pp.1.01"
    assert result.state.target_path == "ptsip.yaml"
    assert result.state.requires_temporary_target is False
    assert result.state.intermediate_profiles == ()


def test_034_source_targets_pp101_directly_without_historical_hops(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.4-draft", V034_REVISION)

    result = discover_direct_profile_convergence(tmp_path)

    assert result.valid
    assert result.state is not None
    assert result.state.mode is DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION
    assert result.state.source_compatibility_contract.canonical == "pp.0.00"
    assert result.state.target_contract.canonical == "pp.1.01"
    assert result.state.target_path == "ptsip_pp1.01.yaml"
    assert result.state.requires_temporary_target is True
    assert result.state.intermediate_profiles == ()


def test_existing_legacy_036_target_is_reused_for_pp101(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.4-draft", V034_REVISION)
    _write_profile(tmp_path, "ptsip_0.3.6.yaml", "0.3.6-draft", V036_REVISION)

    result = discover_direct_profile_convergence(tmp_path)

    assert result.valid
    assert result.state is not None
    assert result.state.target_path == "ptsip_0.3.6.yaml"
    assert result.state.target_is_legacy_alias is True
    assert result.state.requires_temporary_target is False


def test_equivalent_legacy_and_pp_targets_fail_closed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.4-draft", V034_REVISION)
    _write_profile(tmp_path, "ptsip_0.3.6.yaml", "0.3.6-draft", V036_REVISION)
    _write_profile(tmp_path, "ptsip_pp1.01.yaml", "pp.1.01", "b" * 40)

    result = discover_direct_profile_convergence(tmp_path)

    assert not result.valid
    assert "DUPLICATE_EQUIVALENT_TARGET" in _codes(result)


def test_synthetic_intermediate_tool_target_is_not_replayed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.4-draft", V034_REVISION)
    _write_profile(tmp_path, "ptsip_0.3.7.yaml", "0.3.7-draft", "7" * 40)

    result = discover_direct_profile_convergence(tmp_path)

    assert not result.valid
    assert "PP_CONVERGENCE_SYNTHETIC_INTERMEDIATE" in _codes(result)


def test_current_pp101_profile_is_already_converged(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "pp.1.01", "c" * 40)

    result = discover_direct_profile_convergence(tmp_path)

    assert result.valid
    assert result.state is not None
    assert result.state.mode is DirectConvergenceMode.CURRENT
    assert result.state.target_path == "ptsip.yaml"


def test_unregistered_historical_source_fails_closed(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.2.0-draft", "2" * 40)

    result = discover_direct_profile_convergence(tmp_path)

    assert not result.valid
    assert "PP_IDENTITY_MALFORMED" in _codes(result)


def test_direct_convergence_snapshot_detects_target_change(tmp_path: Path) -> None:
    _write_profile(tmp_path, "ptsip.yaml", "0.3.4-draft", V034_REVISION)
    target = _write_profile(tmp_path, "ptsip_0.3.6.yaml", "0.3.6-draft", V036_REVISION)

    result = discover_direct_profile_convergence(tmp_path)
    assert result.valid and result.state is not None
    assert validate_direct_convergence_snapshot(tmp_path, result.state) == ()

    target.write_text(target.read_text(encoding="utf-8") + "# changed\n", encoding="utf-8")

    diagnostics = validate_direct_convergence_snapshot(tmp_path, result.state)
    assert [item.code for item in diagnostics] == ["STALE_DIRECT_CONVERGENCE_SNAPSHOT"]
    assert "ptsip_0.3.6.yaml: profile content changed" in diagnostics[0].message
