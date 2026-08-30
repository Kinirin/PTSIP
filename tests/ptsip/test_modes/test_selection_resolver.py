from __future__ import annotations

import runpy
from pathlib import Path

import pytest
import yaml


REPO_ROOT = Path(__file__).resolve().parents[3]
REGISTRY_PATH = REPO_ROOT / ".github" / "test_modes.yaml"
RESOLVER_PATH = REPO_ROOT / ".github" / "scripts" / "resolve_test_modes.py"
RESOLVER = runpy.run_path(str(RESOLVER_PATH))

MATCHES_WATCH = RESOLVER["matches_watch"]
NORMALIZE_REPO_PATH = RESOLVER["normalize_repo_path"]
SELECT_AUTOMATIC = RESOLVER["select_automatic_modes"]
SELECT_MANUAL = RESOLVER["select_manual_modes"]
BUILD_PLAN = RESOLVER["build_execution_plan"]
SELECTION_ERROR = RESOLVER["TestModeSelectionError"]

EXPECTED_MODE_IDS = [
    "ptsip-migration",
    "ptsip-evidence",
    "ptsip-source-compat",
    "vpms",
]


def _registry() -> dict[str, object]:
    payload = yaml.safe_load(REGISTRY_PATH.read_text(encoding="utf-8-sig"))
    assert isinstance(payload, dict)
    return payload


def _ids(selected: list[dict[str, object]]) -> list[str]:
    return [str(mode["id"]) for mode in selected]


def test_repo_path_normalization_accepts_windows_separators() -> None:
    assert NORMALIZE_REPO_PATH(r"src\ptsip\evidence\contract.py") == "src/ptsip/evidence/contract.py"


def test_watch_matching_respects_recursive_repository_globs() -> None:
    assert MATCHES_WATCH("src/ptsip/evidence/contract.py", "src/ptsip/evidence/**")
    assert not MATCHES_WATCH("src/ptsip/migration/model.py", "src/ptsip/evidence/**")


def test_evidence_change_selects_evidence_and_migration() -> None:
    selected = SELECT_AUTOMATIC(_registry(), ["src/ptsip/evidence/contract.py"])
    assert _ids(selected) == ["ptsip-migration", "ptsip-evidence"]


def test_source_compat_change_selects_source_compat_and_migration() -> None:
    selected = SELECT_AUTOMATIC(_registry(), ["src/ptsip/source_compat/reader.py"])
    assert _ids(selected) == ["ptsip-migration", "ptsip-source-compat"]


def test_migration_change_selects_only_migration() -> None:
    selected = SELECT_AUTOMATIC(_registry(), ["src/ptsip/migration/model.py"])
    assert _ids(selected) == ["ptsip-migration"]


def test_vpms_change_selects_only_vpms() -> None:
    selected = SELECT_AUTOMATIC(_registry(), ["src/vpms/model.py"])
    assert _ids(selected) == ["vpms"]


def test_unrelated_change_selects_no_modes() -> None:
    selected = SELECT_AUTOMATIC(_registry(), ["README.md"])
    assert _ids(selected) == []


def test_control_plane_change_selects_all_registered_modes() -> None:
    selected = SELECT_AUTOMATIC(_registry(), [".github/scripts/resolve_test_modes.py"])
    assert _ids(selected) == EXPECTED_MODE_IDS


def test_manual_all_selects_all_registered_modes() -> None:
    selected = SELECT_MANUAL(_registry(), "all")
    assert _ids(selected) == EXPECTED_MODE_IDS


def test_manual_specific_mode_selects_only_requested_mode() -> None:
    selected = SELECT_MANUAL(_registry(), "ptsip-evidence")
    assert _ids(selected) == ["ptsip-evidence"]


def test_manual_unknown_mode_fails_closed() -> None:
    with pytest.raises(SELECTION_ERROR, match="unknown requested Test Mode"):
        SELECT_MANUAL(_registry(), "missing-mode")


def test_execution_plan_contains_execution_identity_not_architecture_authority() -> None:
    selected = SELECT_AUTOMATIC(_registry(), ["src/ptsip/evidence/contract.py"])
    plan = BUILD_PLAN(selected)

    assert plan == [
        {
            "id": "ptsip-migration",
            "component_ref": "ptsip-migration-verification",
            "pytest": ["tests/ptsip/migration"],
        },
        {
            "id": "ptsip-evidence",
            "component_ref": "ptsip-evidence-verification",
            "pytest": ["tests/ptsip/evidence"],
        },
    ]
    assert all("classification" not in item for item in plan)
    assert all("roles" not in item for item in plan)
    assert all("purpose" not in item for item in plan)
