from __future__ import annotations

import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator


ROOT = Path(__file__).resolve().parents[2]
POLICY_ROOT = ROOT / "developer" / "policy"
SCHEMA_PATH = POLICY_ROOT / "schemas" / "management-policy.schema.json"

EXPECTED_DEVELOPER_POLICY_CLASSES = [
    "PTSIP_DEVELOPER_POLICY",
    "VPMS_DEVELOPER_POLICY",
    "PTSIP_BOUND_POLICY",
]


def _yaml(path: Path) -> dict[str, object]:
    payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(payload, dict)
    return payload


def _json(path: Path) -> dict[str, object]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    assert isinstance(payload, dict)
    return payload


def test_management_policy_schema_declares_explicit_owner_classes() -> None:
    schema = _json(SCHEMA_PATH)
    policy_class = schema["properties"]["policy_class"]
    assert policy_class["enum"] == EXPECTED_DEVELOPER_POLICY_CLASSES


def test_new_boundary_and_vpms_policies_validate_under_shared_root() -> None:
    schema = _json(SCHEMA_PATH)
    validator = Draft202012Validator(schema)
    expected = {
        "MPD-0018.yaml": "PTSIP_BOUND_POLICY",
        "MPD-0019.yaml": "PTSIP_BOUND_POLICY",
        "MPD-0020.yaml": "VPMS_DEVELOPER_POLICY",
    }

    for filename, policy_class in expected.items():
        payload = _yaml(POLICY_ROOT / filename)
        assert payload["policy_class"] == policy_class
        assert not list(validator.iter_errors(payload))


def test_shared_policy_root_does_not_imply_ptsip_ownership() -> None:
    ownership = _yaml(POLICY_ROOT / "MPD-0018.yaml")["rules"][
        "developer_policy_ownership_classification"
    ]
    storage = ownership["storage_model"]
    assert storage["shared_root"] == "developer/policy"
    assert storage["owner_specific_directory_required"] is False
    assert storage["physical_directory_implies_policy_owner"] is False
    assert ownership["canonical_owner_discriminator"] == "policy_class"


def test_vpms_redefinition_does_not_silently_override_legacy_support_authority() -> None:
    legacy = _yaml(ROOT / "src" / "policy" / "SFP-0006.yaml")
    boundary = _yaml(POLICY_ROOT / "MPD-0019.yaml")
    vpms = _yaml(POLICY_ROOT / "MPD-0020.yaml")

    assert legacy["policy"]["status"] == "ACTIVE"
    assert boundary["policy"]["status"] == "APPROVED"
    assert vpms["policy"]["status"] == "APPROVED"
    assert (
        boundary["rules"]["ptsip_vpms_verification_boundary"]["legacy_vpms_authority"][
            "direct_override_by_approved_developer_policy"
        ]
        == "FORBIDDEN"
    )
    assert (
        vpms["rules"]["migration"]["legacy_support_policy"][
            "reconciliation_required_before_activation"
        ]
        is True
    )


def test_ptsip_selects_and_vpms_executes_without_taxonomy_mirroring() -> None:
    boundary = _yaml(POLICY_ROOT / "MPD-0019.yaml")["rules"][
        "ptsip_vpms_verification_boundary"
    ]
    vpms = _yaml(POLICY_ROOT / "MPD-0020.yaml")["rules"]

    assert boundary["handoff_contract"]["payload"] == "EXPLICIT_VERIFICATION_REF_SET"
    assert boundary["handoff_contract"]["vpms_may_expand_selection_scope"] is False
    assert vpms["ptsip_independence"]["ptsip_classification_taxonomy_mirroring"] == "FORBIDDEN"
    assert vpms["ptsip_independence"]["ptsip_test_mode_selection_ownership"] == "FORBIDDEN"
    assert vpms["execution_plane"]["language_neutral"] is True
    assert vpms["execution_plane"]["framework_neutral"] is True
