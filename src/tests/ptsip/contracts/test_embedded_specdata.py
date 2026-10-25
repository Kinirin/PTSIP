from __future__ import annotations

import json
from importlib.resources import files
from pathlib import Path

import yaml


def test_embedded_profile_schema_and_registry_match_bound_snapshot_assets() -> None:
    root = Path(__file__).resolve().parents[4]
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


def test_embedded_project_profile_contract_registry_matches_canonical() -> None:
    root = Path(__file__).resolve().parents[4]
    canonical = yaml.safe_load(
        (root / "registry/project-profile-contracts.yaml").read_text(encoding="utf-8")
    )
    embedded = yaml.safe_load(
        files("ptsip")
        .joinpath("specdata/project-profile-contracts.yaml")
        .read_text(encoding="utf-8")
    )
    assert embedded == canonical


def test_rbac_schema_scaffolds_and_registry_requirements_are_packaged_fail_closed() -> None:
    from jsonschema import Draft202012Validator

    root = Path(__file__).resolve().parents[4]
    canonical = json.loads(
        (root / "registry/compile-sealed-rbac-registry-fields.json").read_text(encoding="utf-8")
    )
    embedded = json.loads(
        files("ptsip").joinpath("specdata/compile-sealed-rbac-registry-fields.json").read_text(encoding="utf-8")
    )
    assert embedded == canonical
    assert [(entry["path"], entry["presence"]) for entry in canonical["field_requirements"]] == [
        ("schema_version", "REQUIRED"),
        ("entries", "REQUIRED"),
        ("entries[].id", "REQUIRED"),
        ("entries[].kind", "REQUIRED"),
        ("entries[].resource", "REQUIRED"),
        ("entries[].revision", "CONDITIONAL"),
        ("metadata", "OPTIONAL"),
    ]
    schema_files = (
        "responsibility.schema.json",
        "condition-atom.schema.json",
        "permission-atom.schema.json",
        "binding.schema.json",
        "sealed-contract.schema.json",
        "activation.schema.json",
    )
    for name in schema_files:
        source = json.loads(
            (root / ".ptsip/contracts/schemas/compile-sealed-rbac" / name).read_text(encoding="utf-8")
        )
        bundled = json.loads(
            files("ptsip").joinpath("specdata", "compile-sealed-rbac", name).read_text(encoding="utf-8")
        )
        assert bundled == source
        Draft202012Validator.check_schema(bundled)
        assert not Draft202012Validator(bundled).is_valid({})
        assert not Draft202012Validator(bundled).is_valid({"Identity": {"kind": "BINDING"}})
