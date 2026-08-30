from __future__ import annotations

import runpy
from pathlib import Path

import yaml


REPO_ROOT = Path(__file__).resolve().parents[3]
VALIDATOR_PATH = REPO_ROOT / ".github" / "scripts" / "validate_test_modes.py"
VALIDATOR = runpy.run_path(str(VALIDATOR_PATH))["validate_registry"]


def _write_profile(root: Path) -> None:
    payload = {
        "ptsip": {
            "version": "pp.1.01",
            "specification": {
                "family": "0.3.7-draft",
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": "3c47816770d194ae42f98faedc911d980db0e62a",
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "product-verification",
                "classification": "PRODUCT",
                "roles": ["VERIFICATION"],
                "include": ["tests/product/**"],
                "purpose": "product_verification",
                "shipped": False,
                "runtime_required": False,
                "executable": True,
                "release_owner": "product",
                "compatibility_owner": "product",
            },
            {
                "id": "product-runtime",
                "classification": "PRODUCT",
                "roles": ["IMPLEMENTATION"],
                "include": ["src/product/**"],
                "purpose": "product_runtime",
                "shipped": True,
                "runtime_required": True,
                "executable": True,
                "release_owner": "product",
                "compatibility_owner": "product",
            },
        ],
    }
    (root / "ptsip.yaml").write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def _write_registry(root: Path, modes: list[dict[str, object]]) -> None:
    registry_dir = root / ".github"
    registry_dir.mkdir(parents=True, exist_ok=True)
    (registry_dir / "test_modes.yaml").write_text(
        yaml.safe_dump({"version": 1, "modes": modes}, sort_keys=False),
        encoding="utf-8",
    )


def _valid_mode() -> dict[str, object]:
    return {
        "id": "product",
        "component_ref": "product-verification",
        "execution": {"pytest": ["tests/product"]},
        "watch": ["src/product/**", "tests/product/**"],
    }


def _validate(root: Path) -> list[str]:
    return VALIDATOR(root / ".github" / "test_modes.yaml", root / "ptsip.yaml", root)


def test_repository_test_mode_registry_v1_is_valid() -> None:
    errors = VALIDATOR(
        REPO_ROOT / ".github" / "test_modes.yaml",
        REPO_ROOT / "ptsip.yaml",
        REPO_ROOT,
    )
    assert errors == []


def test_valid_mode_resolves_declared_verification_component(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    _write_registry(tmp_path, [_valid_mode()])

    assert _validate(tmp_path) == []


def test_unknown_component_ref_is_rejected(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    mode = _valid_mode()
    mode["component_ref"] = "missing-verification"
    _write_registry(tmp_path, [mode])

    errors = _validate(tmp_path)
    assert any("does not exist in ptsip.yaml" in error for error in errors)


def test_non_verification_component_ref_is_rejected(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    mode = _valid_mode()
    mode["component_ref"] = "product-runtime"
    _write_registry(tmp_path, [mode])

    errors = _validate(tmp_path)
    assert any("must reference a VERIFICATION component" in error for error in errors)


def test_missing_pytest_target_is_rejected(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    _write_registry(tmp_path, [_valid_mode()])

    errors = _validate(tmp_path)
    assert any("does not exist in the repository" in error for error in errors)


def test_duplicate_mode_id_is_rejected(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    _write_registry(tmp_path, [_valid_mode(), _valid_mode()])

    errors = _validate(tmp_path)
    assert any("duplicate Test Mode id" in error for error in errors)


def test_registry_cannot_duplicate_architecture_authority(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    mode = _valid_mode()
    mode["classification"] = "PRODUCT"
    mode["roles"] = ["VERIFICATION"]
    mode["purpose"] = "duplicated_authority"
    _write_registry(tmp_path, [mode])

    errors = _validate(tmp_path)
    assert any("duplicates architecture authority fields" in error for error in errors)
