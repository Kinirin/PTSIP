from __future__ import annotations

from pathlib import Path

import yaml

import ptsip.migration as migration
from ptsip.migration import default_target_semantics
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.validation.profile import find_profile


_REPOSITORY_ROOT = Path(__file__).resolve().parents[4]
_CURRENT_PP_RUNTIME_SURFACES = (
    "src/ptsip/migration",
    "src/ptsip/repository/profile_convergence.py",
)


def test_repository_profile_keeps_pp_and_specification_identity_independent() -> None:
    selected = find_profile(_REPOSITORY_ROOT)
    assert selected is not None
    payload = yaml.safe_load(selected.read_text(encoding="utf-8"))

    assert payload["ptsip"]["version"] == CURRENT_PROJECT_PROFILE_VERSION
    specification = payload["ptsip"]["specification"]
    assert specification["revision"]
    assert specification["source"]
    if "family" in specification:
        assert payload["ptsip"]["version"] != specification["family"]


def test_default_migration_target_comes_from_current_pp_contract() -> None:
    semantics = default_target_semantics()

    assert semantics.draft_version == CURRENT_PROJECT_PROFILE_VERSION


def test_current_pp_runtime_surfaces_do_not_use_spec_family_as_profile_target() -> None:
    for relative in _CURRENT_PP_RUNTIME_SURFACES:
        surface = _REPOSITORY_ROOT / relative
        assert surface.exists(), relative
        files = sorted(surface.rglob("*.py")) if surface.is_dir() else [surface]
        assert files, relative
        for path in files:
            text = path.read_text(encoding="utf-8")
            assert '"0.3.7-draft"' not in text, path
            assert "'0.3.7-draft'" not in text, path


def test_public_migration_surface_prefers_direct_pp_convergence() -> None:
    exported = set(migration.__all__)

    assert "build_direct_final_point_convergence_plan" in exported
    assert "analyze_direct_profile_convergence" in exported
    assert "build_identity_rewrite_plan" in exported
    assert "build_final_point_convergence_plan" not in exported
