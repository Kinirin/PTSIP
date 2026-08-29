from __future__ import annotations

import json
from importlib.resources import files
from pathlib import Path

import yaml


def test_embedded_profile_schema_and_registry_match_bound_snapshot_assets() -> None:
    root = Path(__file__).resolve().parents[3]
    canonical_schema = json.loads(
        (root / "schemas/ptsip-profile.schema.json").read_text(encoding="utf-8")
    )
    embedded_schema = json.loads(
        files("ptsip").joinpath("specdata/ptsip-profile.schema.json").read_text(encoding="utf-8")
    )
    assert embedded_schema == canonical_schema

    canonical_registry = yaml.safe_load(
        (root / "registry/ptsip-registry.yaml").read_text(encoding="utf-8")
    )
    embedded_registry = yaml.safe_load(
        files("ptsip").joinpath("specdata/ptsip-registry.yaml").read_text(encoding="utf-8")
    )
    assert embedded_registry == canonical_registry
