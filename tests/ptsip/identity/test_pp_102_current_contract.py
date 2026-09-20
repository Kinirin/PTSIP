from __future__ import annotations

import json
from pathlib import Path

import yaml

from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.validation.profile import find_profile, validate_profile


ROOT = Path(__file__).resolve().parents[3]


def test_pp_102_is_current_and_declares_user_revision_semantics() -> None:
    registry = yaml.safe_load(
        (ROOT / "registry" / "project-profile-contracts.yaml").read_text(encoding="utf-8")
    )
    assert registry["current"] == CURRENT_PROJECT_PROFILE_VERSION == "pp.1.02"

    current = next(
        row for row in registry["contracts"] if row["version"] == registry["current"]
    )
    schema = json.loads((ROOT / current["schema"]).read_text(encoding="utf-8"))
    ptsip = schema["properties"]["ptsip"]
    assert ptsip["required"] == ["version", "revision", "profile_role", "specification"]
    properties = ptsip["properties"]
    assert properties["revision"]["pattern"] == r"^Rev\.[0-9]{4}$"
    assert set(properties["profile_role"]["enum"]) == {
        "PROJECT",
        "DISTRIBUTED_EXAMPLE",
    }
    specification = properties["specification"]
    assert specification["required"] == ["source", "revision"]
    assert "family" not in specification["properties"]


def test_distributed_profiles_are_non_authoritative_materialization_sources() -> None:
    catalog = yaml.safe_load(
        (ROOT / "profiles" / "index.yaml").read_text(encoding="utf-8")
    )
    assert catalog["profiles"]

    for row in catalog["profiles"]:
        assert row["contract"] == "pp.1.02"
        assert row["profile_role"] == "DISTRIBUTED_EXAMPLE"
        assert row["materialization"] == "PROJECT_PATH_RESOLUTION_REQUIRED"

        profile = yaml.safe_load(
            (ROOT / "profiles" / row["resource"]).read_text(encoding="utf-8")
        )
        ptsip = profile["ptsip"]
        assert ptsip["version"] == "pp.1.02"
        assert ptsip["revision"] == "Rev.0001"
        assert ptsip["profile_role"] == "DISTRIBUTED_EXAMPLE"
        assert set(ptsip["specification"]) == {"source", "revision"}


def test_repository_default_profile_resolves_through_local_catalog() -> None:
    selected = find_profile(ROOT)
    assert selected == ROOT / ".ptsip" / "profiles" / "main.ptsip.yaml"

    catalog = yaml.safe_load(
        (ROOT / ".ptsip" / "profiles" / "index.yaml").read_text(encoding="utf-8")
    )
    assert catalog["schema_version"] == "ptsip-local-profile-catalog/v1"
    assert catalog["default_profile"] == "main"

    payload = yaml.safe_load(selected.read_text(encoding="utf-8"))
    assert payload["ptsip"]["version"] == "pp.1.02"
    assert payload["ptsip"]["revision"] == "Rev.0001"
    assert payload["ptsip"]["profile_role"] == "PROJECT"
    assert set(payload["ptsip"]["specification"]) == {"source", "revision"}

    validation = validate_profile(ROOT)
    assert validation.valid, validation.errors
