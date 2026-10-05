from __future__ import annotations

from pathlib import Path

import pytest
from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml

from developer.automation.root_family_policy_entry import (
    ROOT_FAMILIES,
    RootFamilyEntryError,
    inspect_id,
    resolve_entry,
)


ROOT = Path(__file__).resolve().parents[2]


def test_root_family_vocabulary_is_exactly_registered_fourteen() -> None:
    assert tuple(ROOT_FAMILIES) == (
        "NORM",
        "GOV",
        "INTENT",
        "ARCH",
        "INFO",
        "CNTR",
        "RISK",
        "SUPPLY",
        "REAL",
        "ASSURE",
        "CTRL",
        "CHANGE",
        "OPS",
        "RECORD",
    )


def test_same_arch_token_resolves_to_distinct_class_scoped_authority_entries() -> None:
    developer = resolve_entry("PTSIP_DEVELOPER_POLICY", "ARCH", root=ROOT)
    support = resolve_entry("PTSIP_SUPPORT_FEATURE", "ARCH", root=ROOT)

    assert developer["authority_identity"] == {
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "responsibility_family": "ARCH",
    }
    assert support["authority_identity"] == {
        "policy_class": "PTSIP_SUPPORT_FEATURE",
        "responsibility_family": "ARCH",
    }
    assert developer["semantic_inheritance"] == "FORBIDDEN"
    assert support["semantic_inheritance"] == "FORBIDDEN"

    developer_id = developer["allocated_policy_id"]
    support_id = support["allocated_policy_id"]
    assert isinstance(developer_id, str) and developer_id.startswith("MPD-ARCH-")
    assert isinstance(support_id, str) and support_id.startswith("SFP-ARCH-")
    assert developer["canonical_path"] == f"developer/policy/ARCH/{developer_id}.yaml"
    assert support["canonical_path"] == f"src/policy/ARCH/{support_id}.yaml"
    assert developer["schema_ref"] == "developer/policy/schemas/root-family-policy.schema.json"
    assert support["schema_ref"] == "src/policy/schemas/ptsip-support-root-family-policy.schema.json"


def test_legacy_developer_family_is_readable_migration_input_not_new_entry() -> None:
    with pytest.raises(RootFamilyEntryError) as exc:
        resolve_entry("PTSIP_DEVELOPER_POLICY", "SPEC", root=ROOT)

    assert exc.value.code == "LEGACY_FAMILY_NEW_ALLOCATION_FORBIDDEN"


@pytest.mark.parametrize(
    ("policy_id", "policy_class", "family", "path"),
    [
        (
            "MPD-ARCH-0001",
            "PTSIP_DEVELOPER_POLICY",
            "ARCH",
            "developer/policy/ARCH/MPD-ARCH-0001.yaml",
        ),
        (
            "SFP-ARCH-0001",
            "PTSIP_SUPPORT_FEATURE",
            "ARCH",
            "src/policy/ARCH/SFP-ARCH-0001.yaml",
        ),
    ],
)
def test_root_family_id_inspection_is_exact_and_class_scoped(
    policy_id: str,
    policy_class: str,
    family: str,
    path: str,
) -> None:
    result = inspect_id(policy_id, root=ROOT)
    assert result["policy_class"] == policy_class
    assert result["responsibility_family"] == family
    assert result["canonical_path"] == path


def test_unregistered_family_does_not_fall_back() -> None:
    with pytest.raises(RootFamilyEntryError) as exc:
        resolve_entry("PTSIP_DEVELOPER_POLICY", "UNKNOWN", root=ROOT)

    assert exc.value.code == "UNKNOWN_ROOT_FAMILY"


@pytest.mark.parametrize("family", ROOT_FAMILIES)
def test_support_index_accepts_exact_family_paths_and_rejects_misrouting(family: str) -> None:
    schema = load_json(
        "src/policy/schemas/ptsip-support-feature-policy-index.schema.json", root=ROOT
    )
    validator = Draft202012Validator(schema)
    index = load_yaml("src/policy/index.yaml", root=ROOT)
    # The legacy corpus must remain readable before any Root policy exists.
    validator.validate(index)
    entry = resolve_entry("PTSIP_SUPPORT_FEATURE", family, root=ROOT)
    route = {
        "id": entry["allocated_policy_id"],
        "path": str(entry["canonical_path"]).removeprefix("src/policy/"),
        "status": "DRAFT",
    }
    index["policies"].append(route)
    validator.validate(index)

    valid_path = str(route["path"])
    other_family = next(value for value in ROOT_FAMILIES if value != family)
    for invalid_path in (
        valid_path.removesuffix(".yaml") + "Xyaml",
        "../" + valid_path,
        other_family + "/" + valid_path.split("/", 1)[1],
    ):
        route["path"] = invalid_path
        assert tuple(validator.iter_errors(index)), invalid_path
