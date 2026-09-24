from __future__ import annotations

import json
import shutil
from pathlib import Path

import pytest
import yaml

from developer.automation.policy_identity_lifecycle import (
    PolicyIdentityLifecycleError,
    inspect_policy,
    preflight_new_policy,
    register_policy,
    status_preflight,
)


SOURCE_ROOT = Path(__file__).resolve().parents[2]


def _write_yaml(path: Path, payload: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


@pytest.fixture()
def repo(tmp_path: Path) -> Path:
    (tmp_path / "pyproject.toml").write_text("[project]\nname='fixture'\nversion='0'\n", encoding="utf-8")
    schemas = tmp_path / "developer/policy/schemas"
    schemas.mkdir(parents=True)
    for name in ("management-policy.schema.json", "policy-approval-provenance.schema.json"):
        shutil.copy(SOURCE_ROOT / "developer/policy/schemas" / name, schemas / name)

    policies = [
        ("MPD-0012", "DRAFT", "Existing draft"),
        ("MPD-0013", "ACTIVE", "Existing active"),
    ]
    _write_yaml(
        tmp_path / "developer/policy/index.yaml",
        {
            "schema_version": "ptsip-developer-policy-index/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "policies": [
                {"id": policy_id, "path": f"developer/policy/{policy_id}.yaml", "status": status}
                for policy_id, status, _ in policies
            ],
        },
    )
    for policy_id, status, title in policies:
        _write_yaml(
            tmp_path / f"developer/policy/{policy_id}.yaml",
            {
                "schema_version": "ptsip-developer-policy/v1",
                "policy_class": "PTSIP_DEVELOPER_POLICY",
                "policy": {"id": policy_id, "title": title, "status": status},
                "rules": {"fixture": {"enabled": True}},
            },
        )
    _write_yaml(
        tmp_path / "developer/policy/registries/authority-subject-registry.yaml",
        {
            "schema_version": "ptsip-developer-authority-subject-registry/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "registry_role": "MANAGEMENT_POLICY_SUBJECT_CATALOG",
            "subject_identity_schemes": {
                "MANAGEMENT_POLICY_ID": {
                    "source_field": "policy.id",
                    "registered_values": ["MPD-0012", "MPD-0013"],
                }
            },
        },
    )
    return tmp_path


def _approval(
    repo: Path,
    *,
    requested_policy_id: str | None = None,
    target_status: str = "ACTIVE",
    scope: str = "TEMPORARY_DIRECTION_AND_IMPLEMENTATION",
) -> Path:
    path = repo / "developer/policy/approvals/MPA-test.yaml"
    approval = {
        "approval_id": "MPA-test",
        "decision": "APPROVED",
        "source_kind": "PROJECT_OWNER_DIRECT_INSTRUCTION",
        "decision_source": "USER_EXPLICIT",
        "source_reference": "test-fixture",
        "approval_scope": scope,
        "target_status": target_status,
        "implementation_authorized": True,
        "policy_content_review_scope": "DIRECTION",
        "recorded_at": "2026-09-24T16:00:00+09:00",
    }
    if requested_policy_id is not None:
        approval["requested_policy_id"] = requested_policy_id
    _write_yaml(
        path,
        {
            "schema_version": "ptsip-policy-approval-provenance/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "approval": approval,
        },
    )
    return path


def test_inspect_reports_draft_without_operational_resolution(repo: Path) -> None:
    result = inspect_policy("MPD-0012", root=repo)
    assert result["status"] == "FOUND"
    assert result["policy_status"] == "DRAFT"
    assert result["index_status"] == "DRAFT"
    assert result["operationally_resolvable"] is False


def test_preflight_rejects_existing_requested_id(repo: Path) -> None:
    approval = _approval(repo, requested_policy_id="MPD-0013")
    with pytest.raises(PolicyIdentityLifecycleError) as exc:
        preflight_new_policy(approval, root=repo)
    assert exc.value.code == "POLICY_ID_ALREADY_EXISTS"


def test_preflight_allocates_next_id_and_keeps_temporary_approval_separate_from_status(repo: Path) -> None:
    approval = _approval(repo, target_status="DRAFT")
    result = preflight_new_policy(approval, root=repo)
    assert result["allocated_policy_id"] == "MPD-0014"
    assert result["approval_scope"] == "TEMPORARY_DIRECTION_AND_IMPLEMENTATION"
    assert result["target_status"] == "DRAFT"
    assert result["implementation_authorized"] is True


def test_invalid_approval_without_explicit_target_status_fails_closed(repo: Path) -> None:
    approval = _approval(repo)
    payload = yaml.safe_load(approval.read_text(encoding="utf-8"))
    del payload["approval"]["target_status"]
    _write_yaml(approval, payload)
    with pytest.raises(PolicyIdentityLifecycleError) as exc:
        preflight_new_policy(approval, root=repo)
    assert exc.value.code == "INVALID_APPROVAL_PROVENANCE"


def test_register_updates_index_and_subject_registry_only_after_exact_policy_file(repo: Path) -> None:
    approval = _approval(repo, requested_policy_id="MPD-0014", target_status="ACTIVE")
    preflight = preflight_new_policy(approval, root=repo)
    assert preflight["allocated_policy_id"] == "MPD-0014"

    policy_path = repo / "developer/policy/MPD-0014.yaml"
    _write_yaml(
        policy_path,
        {
            "schema_version": "ptsip-developer-policy/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "policy": {"id": "MPD-0014", "title": "New policy", "status": "ACTIVE"},
            "rules": {"fixture": {"enabled": True}},
        },
    )
    result = register_policy(approval, policy_path, root=repo)
    assert result["status"] == "REGISTERED"

    index = yaml.safe_load((repo / "developer/policy/index.yaml").read_text(encoding="utf-8"))
    assert index["policies"][-1]["id"] == "MPD-0014"
    assert index["policies"][-1]["status"] == "ACTIVE"
    registry = yaml.safe_load(
        (repo / "developer/policy/registries/authority-subject-registry.yaml").read_text(encoding="utf-8")
    )
    assert registry["subject_identity_schemes"]["MANAGEMENT_POLICY_ID"]["registered_values"][-1] == "MPD-0014"


def test_status_preflight_requires_exact_existing_policy_binding(repo: Path) -> None:
    approval = _approval(repo, requested_policy_id="MPD-0013", target_status="DRAFT")
    result = status_preflight("MPD-0013", approval, root=repo)
    assert result == {
        "status": "READY",
        "policy_id": "MPD-0013",
        "current_status": "ACTIVE",
        "target_status": "DRAFT",
        "approval_id": "MPA-test",
        "implementation_authorized": True,
    }
