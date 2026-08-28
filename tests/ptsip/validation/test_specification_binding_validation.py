from __future__ import annotations

import subprocess
from pathlib import Path

import yaml

from ptsip.specification_binding import (
    SPECIFICATION_036_FAMILY,
    SPECIFICATION_036_REVISION,
    SPECIFICATION_037,
    SPECIFICATION_SOURCE,
)
from ptsip.validation.profile import validate_profile
from ptsip.validation.specification import specification_binding_errors


def _initialize_repository(root: Path) -> None:
    subprocess.run(["git", "init", "-q"], cwd=root, check=True)
    subprocess.run(["git", "config", "user.email", "ptsip-test@example.invalid"], cwd=root, check=True)
    subprocess.run(["git", "config", "user.name", "PTSIP Test"], cwd=root, check=True)


def _write_current_profile(root: Path, *, revision: str = SPECIFICATION_037.revision) -> Path:
    (root / "src").mkdir(exist_ok=True)
    (root / "src" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    profile = {
        "ptsip": {
            "version": "pp.1.01",
            "specification": {
                "family": SPECIFICATION_037.family,
                "source": SPECIFICATION_037.source,
                "revision": revision,
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "product",
                "classification": "PRODUCT",
                "include": ["src/**"],
                "purpose": "fixture_product",
                "shipped": True,
                "runtime_required": True,
                "executable": True,
            }
        ],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    path = root / "ptsip.yaml"
    path.write_text(yaml.safe_dump(profile, sort_keys=False), encoding="utf-8")
    subprocess.run(["git", "add", "src/app.py", "ptsip.yaml"], cwd=root, check=True)
    return path


def test_current_profile_validation_uses_explicit_specification_capability(tmp_path: Path) -> None:
    _initialize_repository(tmp_path)
    profile = _write_current_profile(tmp_path)

    result = validate_profile(tmp_path, profile)

    assert result.valid is True
    assert result.errors == []
    assert result.details is not None
    assert result.details["specification_binding"] == {
        "family": SPECIFICATION_037.family,
        "source": SPECIFICATION_037.source,
        "revision": SPECIFICATION_037.revision,
        "compatibility_tool_target": "0.3.7",
        "operation": "VALIDATE",
    }


def test_current_profile_unknown_exact_revision_fails_capability_closed(tmp_path: Path) -> None:
    _initialize_repository(tmp_path)
    profile = _write_current_profile(tmp_path, revision="0" * 40)

    result = validate_profile(tmp_path, profile)

    assert result.valid is False
    assert any("[SPEC_BINDING_UNSUPPORTED]" in error for error in result.errors)


def test_current_pp_schema_requires_explicit_family(tmp_path: Path) -> None:
    _initialize_repository(tmp_path)
    profile = _write_current_profile(tmp_path)
    payload = yaml.safe_load(profile.read_text(encoding="utf-8"))
    del payload["ptsip"]["specification"]["family"]
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")

    result = validate_profile(tmp_path, profile)

    assert result.valid is False
    assert any("'family' is a required property" in error for error in result.errors)


def test_historical_specification_validation_does_not_require_migration_bridge() -> None:
    details: dict[str, object] = {}
    payload = {
        "ptsip": {
            "version": SPECIFICATION_036_FAMILY,
            "specification": {
                "source": SPECIFICATION_SOURCE,
                "revision": SPECIFICATION_036_REVISION,
            },
        }
    }

    assert specification_binding_errors(payload, details=details) == []
    assert details["specification_binding"] == {
        "family": SPECIFICATION_036_FAMILY,
        "source": SPECIFICATION_SOURCE,
        "revision": SPECIFICATION_036_REVISION,
        "compatibility_tool_target": "0.3.7",
        "operation": "VALIDATE",
    }
