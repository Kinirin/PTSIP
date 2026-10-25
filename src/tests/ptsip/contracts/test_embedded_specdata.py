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
    assert canonical["entry_kind_scope"]["allowed_semantic_kinds"] == [
        "RESPONSIBILITY",
        "CONDITION_ATOM",
        "PERMISSION_ATOM",
        "BINDING",
    ]
    assert canonical["entry_kind_scope"]["excluded_semantic_kinds"] == [
        "SEAL",
        "ACTIVATION",
    ]
    assert canonical["separate_artifact_contracts"]["seal"]["registry_entry_allowed"] is False
    assert canonical["separate_artifact_contracts"]["activation"]["registry_entry_allowed"] is False
    assert canonical["contract_revision"] == 3
    assert canonical["entry_kind_scope"]["serialized_kind_literals"] == [
        "RESPONSIBILITY", "CONDITION_ATOM", "PERMISSION_ATOM", "BINDING"
    ]
    resource_rules = canonical["entry_resource_reference"]
    assert resource_rules["coordinate_system"] == "PROJECT_REPOSITORY_ROOT_RELATIVE"
    assert resource_rules["canonical_prefix"] == ".ptsip/profiles/rbac/"
    assert resource_rules["allowed_lowercase_suffixes"] == [".json", ".yaml", ".yml"]
    assert resource_rules["nested_subdirectories_allowed"] is True
    assert resource_rules["filesystem_requirements"]["symlink_components_forbidden"] is True
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


def test_rbac_central_registry_draft_schema_is_packaged_and_non_authoritative() -> None:
    from jsonschema import Draft202012Validator

    root = Path(__file__).resolve().parents[4]
    name = "central-registry.draft.schema.json"
    source = json.loads(
        (root / ".ptsip/contracts/schemas/compile-sealed-rbac" / name).read_text(encoding="utf-8")
    )
    embedded = json.loads(
        files("ptsip").joinpath("specdata", "compile-sealed-rbac", name).read_text(encoding="utf-8")
    )
    assert source == embedded
    assert "DRAFT_NON_NORMATIVE" in source["$comment"]
    Draft202012Validator.check_schema(source)
    validator = Draft202012Validator(source)
    valid = {
        "schema_version": "sample",
        "entries": [
            {"id": "RESP-001", "kind": "RESPONSIBILITY",
             "resource": ".ptsip/profiles/rbac/responsibilities/RESP-001.yaml"},
            {"id": "BIND-001", "kind": "BINDING",
             "resource": ".ptsip/profiles/rbac/bindings/BIND-001.json"},
        ],
    }
    assert validator.is_valid(valid)
    for invalid in [
        {"schema_version": "sample", "entries": [
            {"id": "S-1", "kind": "SEAL", "resource": ".ptsip/profiles/rbac/s.json"}
        ]},
        {"schema_version": "sample", "entries": [
            {"id": "B-1", "kind": "BINDING", "resource": ".ptsip/profiles/rbac/../escape.json"}
        ]},
        {"schema_version": "sample", "entries": [
            {"id": "B-1", "kind": "BINDING", "resource": "https://example.com/x.yaml"}
        ]},
        {"schema_version": "sample", "entries": [
            {"id": "B-1", "kind": "BINDING", "resource": ".ptsip/profiles/rbac/x.txt"}
        ]},
        {"schema_version": "sample", "entries": [{"id": "B-1", "kind": "BINDING"}]},
    ]:
        assert not validator.is_valid(invalid)


def test_pp200_candidate_schema_declares_rbac_opt_in_without_activation() -> None:
    from jsonschema import Draft202012Validator

    root = Path(__file__).resolve().parents[4]
    name = "ptsip-profile-pp-2.00.candidate.schema.json"
    canonical = json.loads((root / "schemas" / name).read_text(encoding="utf-8"))
    embedded = json.loads(files("ptsip").joinpath("specdata", name).read_text(encoding="utf-8"))
    assert canonical == embedded
    Draft202012Validator.check_schema(canonical)
    assert "DRAFT_NON_NORMATIVE" in canonical["$comment"]
    assert canonical["properties"]["ptsip"]["properties"]["version"]["const"] == "pp.2.00"
    assert "rbac" in canonical["required"]

    # Test the new declaration in isolation; pp.1.02's preexisting fields
    # are inherited for this candidate and remain unchanged.
    rbac = Draft202012Validator(canonical["properties"]["rbac"])
    assert rbac.is_valid({"enabled": False})
    assert rbac.is_valid({"enabled": False, "registry_ref": "ignored-when-disabled"})
    assert rbac.is_valid({"enabled": True, "registry_ref": ".ptsip/profiles/rbac/registry.json"})
    for invalid in [
        {},
        {"enabled": "true"},
        {"enabled": True},
        {"enabled": True, "registry_ref": "../outside.json"},
        {"enabled": True, "registry_ref": ".ptsip/profiles/rbac/../outside.json"},
        {"enabled": True, "registry_ref": ""},
        {"enabled": True, "registry_ref": ".ptsip/profiles/rbac/registry.json", "auto_grant": True},
    ]:
        assert not rbac.is_valid(invalid)
