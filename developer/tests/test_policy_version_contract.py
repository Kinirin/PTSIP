from __future__ import annotations

import json
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator

from developer.automation.policy_identity_lifecycle import (
    PolicyIdentityLifecycleError,
    initial_policy_version,
    resolve_policy_version_transition,
)
from developer.automation.policy_validator import _validate_policy_version_semantics


ROOT = Path(__file__).resolve().parents[2]


def _schema() -> dict[str, object]:
    return json.loads(
        (ROOT / "developer/policy/schemas/management-policy.schema.json").read_text(
            encoding="utf-8"
        )
    )


def _policy(version: str, status: str) -> dict[str, object]:
    return {
        "schema_version": "ptsip-developer-policy/v1",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "policy": {
            "id": "MPD-0010",
            "version": version,
            "title": "Fixture",
            "status": status,
        },
        "rules": {"fixture": {"enabled": True}},
        "relations": {
            "supersedes": [],
            "amends": [],
            "extends": [],
            "depends_on": [],
        },
    }


@pytest.mark.parametrize(
    ("version", "status"),
    [
        ("0.0", "DRAFT"),
        ("0.137", "DRAFT"),
        ("1.0", "APPROVED"),
        ("1.137", "APPROVED"),
        ("2.0", "ACTIVE"),
        ("2.105", "ACTIVE"),
        ("12.248", "ACTIVE"),
        ("3.7", "SUPERSEDED"),
        ("4.0", "RETIRED"),
    ],
)
def test_schema_accepts_lifecycle_aware_policy_versions(version: str, status: str) -> None:
    payload = _policy(version, status)
    if status == "APPROVED":
        payload["transition"] = {
            "target_status": "ACTIVE",
            "state": "PENDING",
            "requirements": [
                {
                    "id": "TR-001",
                    "type": "FIXTURE",
                    "state": "OPEN",
                    "next_action": {
                        "action": "FIXTURE",
                        "execution": "PARALLEL_ALLOWED",
                        "completion_check": "FIXTURE_PASS",
                    },
                }
            ],
        }
    errors = tuple(Draft202012Validator(_schema()).iter_errors(payload))
    assert errors == ()


@pytest.mark.parametrize(
    ("version", "status"),
    [
        ("1.0", "DRAFT"),
        ("0.1", "APPROVED"),
        ("1.9", "ACTIVE"),
        ("0.9", "RETIRED"),
        ("01.2", "DRAFT"),
        ("1.02", "APPROVED"),
        ("1.0-draft", "APPROVED"),
        ("1.0.0", "APPROVED"),
        ("draft-1", "DRAFT"),
    ],
)
def test_schema_rejects_invalid_or_mismatched_policy_versions(version: str, status: str) -> None:
    payload = _policy(version, status)
    assert tuple(Draft202012Validator(_schema()).iter_errors(payload))


def test_semantic_validator_reports_status_version_mismatch() -> None:
    errors = _validate_policy_version_semantics("MPD-0010", _policy("1.4", "ACTIVE"))
    assert any("POLICY_VERSION_STATUS_MISMATCH" in error for error in errors)


def test_new_policy_version_is_fixed_at_zero_zero() -> None:
    assert initial_policy_version() == "0.0"


@pytest.mark.parametrize(
    ("current_version", "current_status", "change_class", "target_status", "expected"),
    [
        ("0.12", "DRAFT", "DRAFT_NORMATIVE", "DRAFT", "0.13"),
        ("0.12", "DRAFT", "LIFECYCLE_TRANSITION", "APPROVED", "1.12"),
        ("1.12", "APPROVED", "LIFECYCLE_TRANSITION", "ACTIVE", "2.12"),
        ("2.99", "ACTIVE", "COMPATIBLE_NORMATIVE", "ACTIVE", "2.100"),
        ("2.99", "ACTIVE", "INCOMPATIBLE_NORMATIVE", "ACTIVE", "3.0"),
        ("3.7", "ACTIVE", "LIFECYCLE_TRANSITION", "RETIRED", "3.7"),
        ("4.2", "ACTIVE", "NON_NORMATIVE", "ACTIVE", "4.2"),
    ],
)
def test_version_transition_resolver_is_deterministic(
    current_version: str,
    current_status: str,
    change_class: str,
    target_status: str,
    expected: str,
) -> None:
    result = resolve_policy_version_transition(
        current_version=current_version,
        current_status=current_status,
        change_class=change_class,
        target_status=target_status,
    )
    assert result["next_version"] == expected


def test_unapproved_version_transition_fails_closed() -> None:
    with pytest.raises(PolicyIdentityLifecycleError) as exc:
        resolve_policy_version_transition(
            current_version="1.4",
            current_status="APPROVED",
            change_class="COMPATIBLE_NORMATIVE",
            target_status="APPROVED",
        )
    assert exc.value.code == "UNSUPPORTED_POLICY_VERSION_TRANSITION"
