from __future__ import annotations

from pathlib import Path
import tomllib

from setuptools import find_packages


_REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
_EXPECTED_VPMS_PACKAGES = {
    "vpms",
    "vpms.domain",
    "vpms.execution",
    "vpms.execution.adapters",
    "vpms.integration",
}
_EXPECTED_PTSIP_SCRIPTS = {
    "ptsip": "ptsip.cli:main",
    "ptsip-app": "ptsip.app.server:main",
}


def _pyproject() -> dict[str, object]:
    with (_REPOSITORY_ROOT / "pyproject.toml").open("rb") as handle:
        return tomllib.load(handle)


def test_setuptools_discovery_includes_vpms_responsibility_packages() -> None:
    config = _pyproject()
    find_config = config["tool"]["setuptools"]["packages"]["find"]

    assert find_config["where"] == ["src"]
    assert find_config["include"] == ["ptsip*", "vpms*"]

    discovered = set(
        find_packages(
            where=str(_REPOSITORY_ROOT / "src"),
            include=tuple(find_config["include"]),
        )
    )

    assert _EXPECTED_VPMS_PACKAGES <= discovered


def test_packaging_change_preserves_existing_ptsip_console_scripts() -> None:
    config = _pyproject()

    assert config["project"]["scripts"] == _EXPECTED_PTSIP_SCRIPTS
