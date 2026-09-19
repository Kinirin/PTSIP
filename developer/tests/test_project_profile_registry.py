from __future__ import annotations

from pathlib import Path

import yaml

from developer.automation.project_profile_registry import (
    current_project_profile_contract,
    load_project_profile_contract_registry,
    load_public_profile_catalog,
    validate_project_profile_registry_plane,
)


ROOT = Path(__file__).resolve().parents[2]


def test_project_profile_registry_plane_is_machine_valid() -> None:
    assert validate_project_profile_registry_plane(ROOT) == ()


def test_contract_registry_owns_single_current_identity() -> None:
    registry = load_project_profile_contract_registry(ROOT)
    assert registry["current"] == "pp.1.01"

    current = current_project_profile_contract(ROOT)
    assert current["version"] == "pp.1.01"
    assert current["lifecycle"] == "CURRENT"
    assert current["schema"] == "schemas/ptsip-profile-pp-1.01.schema.json"


def test_historical_compatibility_identity_is_registered_without_becoming_current() -> None:
    registry = load_project_profile_contract_registry(ROOT)
    contracts = {item["version"]: item for item in registry["contracts"]}

    assert contracts["pp.0.00"]["lifecycle"] == "LEGACY_COMPATIBILITY_ONLY"
    assert contracts["pp.0.00"]["operations"] == ["IDENTIFY"]
    assert contracts["pp.0.00"]["schema"] is None
    assert registry["transitions"] == []


def test_public_profile_catalog_has_no_duplicate_current_authority() -> None:
    catalog = load_public_profile_catalog(ROOT)
    assert "current" not in catalog
    assert "current_contract" not in catalog


def test_public_profile_catalog_exactly_describes_existing_distribution_assets() -> None:
    catalog = load_public_profile_catalog(ROOT)
    assert catalog["profiles"] == [
        {
            "id": "example",
            "resource": "example.ptsip.yaml",
            "contract": "pp.1.01",
            "responsibility_mode": "explicit",
        },
        {
            "id": "hybrid-python-package",
            "resource": "hybrid-python-package.ptsip.yaml",
            "contract": "pp.1.01",
            "responsibility_mode": "hybrid",
        },
        {
            "id": "template-python-package",
            "resource": "template-python-package.ptsip.yaml",
            "contract": "pp.1.01",
            "responsibility_mode": "template",
        },
    ]

    for entry in catalog["profiles"]:
        payload = yaml.safe_load(
            (ROOT / "profiles" / entry["resource"]).read_text(encoding="utf-8")
        )
        assert payload["ptsip"]["version"] == entry["contract"]
        assert payload["responsibility_map"]["mode"] == entry["responsibility_mode"]
