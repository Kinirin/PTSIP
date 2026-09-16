from __future__ import annotations

import runpy
from pathlib import Path

import yaml


REPO_ROOT = Path(__file__).resolve().parents[3]
REGISTRY_PATH = REPO_ROOT / ".github" / "test_modes.yaml"
SELF_PROFILE_PATH = (
    REPO_ROOT / "developer" / "profiles" / "ptsip-repository.yaml"
)
VALIDATOR_PATH = REPO_ROOT / ".github" / "scripts" / "validate_test_modes.py"
VALIDATOR = runpy.run_path(str(VALIDATOR_PATH))["validate_registry"]
REPOSITORY_MODE_KEYS = {"id", "component_ref", "execution"}
EXPECTED_REPOSITORY_COMPONENT_REFS = {
    "ptsip-core": "ptsip-core-verification",
    "ptsip-evidence": "ptsip-evidence-verification",
    "ptsip-source-compat": "ptsip-source-compat-verification",
    "ptsip-migration": "ptsip-migration-verification",
    "ptsip-remediation": "ptsip-remediation-verification",
    "vpms": "vpms-verification",
    "ptsip-contract": "ptsip-contract-verification",
    "repository-architecture": "repository-architecture-verification",
    "repository-release": "repository-release-verification",
    "test-mode-control-plane": "repository-test-mode-control-plane",
}


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
                "analysis_inputs": ["src/product/**"],
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
    path = root / "developer" / "profiles" / "ptsip-repository.yaml"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        yaml.safe_dump(payload, sort_keys=False),
        encoding="utf-8",
    )


def _write_registry(root: Path, modes: list[dict[str, object]]) -> None:
    registry_dir = root / ".github"
    registry_dir.mkdir(parents=True, exist_ok=True)
    (registry_dir / "test_modes.yaml").write_text(
        yaml.safe_dump({"version": 2, "modes": modes}, sort_keys=False),
        encoding="utf-8",
    )


def _valid_mode() -> dict[str, object]:
    return {
        "id": "product",
        "component_ref": "product-verification",
        "execution": {"pytest": ["tests/product"]},
    }


def _validate(root: Path) -> list[str]:
    return VALIDATOR(
        root / ".github" / "test_modes.yaml",
        root / "developer" / "profiles" / "ptsip-repository.yaml",
        root,
    )


def _repository_registry() -> dict[str, object]:
    payload = yaml.safe_load(
        REGISTRY_PATH.read_text(encoding="utf-8-sig")
    )
    assert isinstance(payload, dict)
    return payload


def test_repository_test_mode_registry_v2_is_valid() -> None:
    errors = VALIDATOR(
        REGISTRY_PATH,
        SELF_PROFILE_PATH,
        REPO_ROOT,
    )
    assert errors == []


def test_repository_test_mode_registry_covers_test_owning_verification_components() -> None:
    registry = _repository_registry()
    assert registry.get("version") == 2

    modes = registry.get("modes")
    assert isinstance(modes, list)
    assert all(isinstance(mode, dict) for mode in modes)

    modes_by_id = {mode.get("id"): mode for mode in modes}
    assert set(modes_by_id) == set(EXPECTED_REPOSITORY_COMPONENT_REFS)
    assert {
        mode_id: mode.get("component_ref")
        for mode_id, mode in modes_by_id.items()
    } == EXPECTED_REPOSITORY_COMPONENT_REFS

    pytest_targets: list[str] = []
    for mode in modes_by_id.values():
        assert set(mode) == REPOSITORY_MODE_KEYS
        assert "watch" not in mode

        execution = mode.get("execution")
        assert isinstance(execution, dict)
        targets = execution.get("pytest")
        assert isinstance(targets, list) and targets
        assert all(
            isinstance(target, str) and target for target in targets
        )
        pytest_targets.extend(targets)

    assert len(pytest_targets) == len(set(pytest_targets))


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
    assert any(
        "does not exist in the selected Project Profile" in error
        for error in errors
    )


def test_missing_verification_component_mode_is_rejected(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    _write_registry(tmp_path, [])

    errors = _validate(tmp_path)
    assert any(
        "missing test-owning VERIFICATION components" in error
        for error in errors
    )


def test_duplicate_component_ref_is_rejected(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    first = _valid_mode()
    second = _valid_mode()
    second["id"] = "product-secondary"
    second["execution"] = {"pytest": ["tests/product/secondary"]}
    (tmp_path / "tests" / "product" / "secondary").mkdir()
    _write_registry(tmp_path, [first, second])

    errors = _validate(tmp_path)
    assert any("already has a Test Mode" in error for error in errors)


def test_pytest_target_must_stay_inside_component_include(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "other").mkdir(parents=True)
    mode = _valid_mode()
    mode["execution"] = {"pytest": ["tests/other"]}
    _write_registry(tmp_path, [mode])

    errors = _validate(tmp_path)
    assert any(
        "outside component_ref include authority" in error
        for error in errors
    )


def test_watch_is_rejected_as_duplicate_selection_authority(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    mode = _valid_mode()
    mode["watch"] = ["src/product/**"]
    _write_registry(tmp_path, [mode])

    errors = _validate(tmp_path)
    assert any("unsupported fields" in error for error in errors)


def test_registry_cannot_duplicate_architecture_authority(tmp_path: Path) -> None:
    _write_profile(tmp_path)
    (tmp_path / "tests" / "product").mkdir(parents=True)
    mode = _valid_mode()
    mode["classification"] = "PRODUCT"
    mode["roles"] = ["VERIFICATION"]
    mode["purpose"] = "duplicated_authority"
    _write_registry(tmp_path, [mode])

    errors = _validate(tmp_path)
    assert any(
        "duplicates architecture authority fields" in error
        for error in errors
    )
