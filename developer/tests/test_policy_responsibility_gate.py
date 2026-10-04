from __future__ import annotations

import shutil
from pathlib import Path

import pytest
import yaml

from developer.automation.policy_identity_lifecycle import (
    PolicyIdentityLifecycleError,
    preflight_family_policy,
    register_family_policy,
)
from developer.automation.policy_responsibility_gate import (
    validate_analysis_semantics,
)
from developer.tests.policy_contract_fixtures import neutralize_fixture_catalog


SOURCE_ROOT = Path(__file__).resolve().parents[2]


def _write_yaml(path: Path, payload: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        yaml.safe_dump(payload, sort_keys=False),
        encoding="utf-8",
    )


def _responsibility(
    responsibility_id: str,
    family: str,
    group_id: str,
    *,
    searched_policy_ids: list[str] | None = None,
) -> dict[str, object]:
    return {
        "responsibility_id": responsibility_id,
        "statement": f"{responsibility_id}_STATEMENT",
        "authority_relation": "OWN",
        "authority_subject": f"{responsibility_id}_SUBJECT",
        "lifecycle_scope": "TEST_SCOPE",
        "cohesion_key": f"{family}_COHESION",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "family": family,
        "referenced_policy_class": None,
        "referenced_family": None,
        "existing_authority_lookup": {
            "searched_policy_class": "PTSIP_DEVELOPER_POLICY",
            "searched_family": family,
            "searched_policy_ids": searched_policy_ids or [],
            "lookup_outcome": "NO_MATCH",
            "candidate_comparisons": [],
        },
        "materialization_action": "CREATE_NEW_POLICY",
        "target_group_id": group_id,
    }


def _analysis_payload(
    responsibilities: list[dict[str, object]],
    groups: list[dict[str, object]],
    *,
    owned_family_set: list[str],
    split_required: bool,
    materialization_allowed: bool = True,
) -> dict[str, object]:
    for group in groups:
        group.setdefault("policy_class", "PTSIP_DEVELOPER_POLICY")
    return {
        "schema_version": "developer-policy-responsibility-analysis/v2",
        "artifact_class": "PTSIP_POLICY_RESPONSIBILITY_ANALYSIS",
        "analysis": {
            "analysis_id": "PRA-test",
            "source_ref": "test",
            "responsibilities": responsibilities,
            "decision": {
                "owned_authority_family_set": [{"policy_class": "PTSIP_DEVELOPER_POLICY", "family": family} for family in owned_family_set],
                "split_required": split_required,
                "materialization_allowed": materialization_allowed,
                "materialization_groups": groups,
            },
        },
    }


def test_multiple_owned_families_require_split() -> None:
    responsibilities = [
        _responsibility("R01", "SPEC", "G01"),
        _responsibility("R02", "VERI", "G02"),
    ]
    groups = [
        {
            "group_id": "G01",
            "family": "SPEC",
            "cohesion_key": "SPEC_COHESION",
            "responsibility_ids": ["R01"],
            "cohesion_rationale": "SPEC responsibility is independent.",
        },
        {
            "group_id": "G02",
            "family": "VERI",
            "cohesion_key": "VERI_COHESION",
            "responsibility_ids": ["R02"],
            "cohesion_rationale": "VERI responsibility is independent.",
        },
    ]
    payload = _analysis_payload(
        responsibilities,
        groups,
        owned_family_set=["SPEC", "VERI"],
        split_required=False,
    )
    errors = validate_analysis_semantics(payload)
    assert "decision.split_required must be True" in errors

    payload["analysis"]["decision"]["split_required"] = True
    assert validate_analysis_semantics(payload) == ()


def test_exact_duplicate_reuses_existing_authority() -> None:
    responsibility = _responsibility(
        "R01",
        "SPEC",
        "G01",
        searched_policy_ids=["MPD-SPEC-0001"],
    )
    responsibility["existing_authority_lookup"] = {
        "searched_policy_class": "PTSIP_DEVELOPER_POLICY",
        "searched_family": "SPEC",
        "searched_policy_ids": ["MPD-SPEC-0001"],
        "lookup_outcome": "MATCHES_FOUND",
        "candidate_comparisons": [
            {
                "policy_id": "MPD-SPEC-0001",
                "section": "fixture",
                "scope_relation": "SAME_SCOPE",
                "collision_class": "EXACT_DUPLICATE",
                "resolution_action": "REFERENCE_EXISTING",
            }
        ],
    }
    responsibility["materialization_action"] = "USE_EXISTING_AUTHORITY"
    responsibility["target_group_id"] = None

    payload = _analysis_payload(
        [responsibility],
        [],
        owned_family_set=["SPEC"],
        split_required=False,
    )
    assert validate_analysis_semantics(payload) == ()


def test_conflict_cannot_be_resolved_as_new_policy() -> None:
    responsibility = _responsibility(
        "R01",
        "SPEC",
        "G01",
        searched_policy_ids=["MPD-SPEC-0001"],
    )
    responsibility["existing_authority_lookup"] = {
        "searched_policy_class": "PTSIP_DEVELOPER_POLICY",
        "searched_family": "SPEC",
        "searched_policy_ids": ["MPD-SPEC-0001"],
        "lookup_outcome": "MATCHES_FOUND",
        "candidate_comparisons": [
            {
                "policy_id": "MPD-SPEC-0001",
                "section": "fixture",
                "scope_relation": "SAME_SCOPE",
                "collision_class": "CONFLICT",
                "resolution_action": "CREATE_NEW_SIBLING_POLICY",
            }
        ],
    }
    payload = _analysis_payload(
        [responsibility],
        [
            {
                "group_id": "G01",
                "family": "SPEC",
                "cohesion_key": "SPEC_COHESION",
                "responsibility_ids": ["R01"],
                "cohesion_rationale": "invalid conflict bypass",
            }
        ],
        owned_family_set=["SPEC"],
        split_required=False,
    )
    errors = validate_analysis_semantics(payload)
    assert any("CONFLICT does not allow CREATE_NEW_SIBLING_POLICY" in item for item in errors)


@pytest.fixture()
def repo(tmp_path: Path) -> Path:
    (tmp_path / "pyproject.toml").write_text(
        "[project]\nname='fixture'\nversion='0'\n",
        encoding="utf-8",
    )

    schemas = tmp_path / "developer/policy/schemas"
    schemas.mkdir(parents=True)
    for name in (
        "management-policy.schema.json",
        "policy-approval-provenance.schema.json",
        "policy-responsibility-analysis.schema.json",
        "policy-materialization-analysis-registry.schema.json",
    ):
        shutil.copy(
            SOURCE_ROOT / "developer/policy/schemas" / name,
            schemas / name,
        )

    policies = [
        ("MPD-0012", "ACTIVE", "Legacy active"),
        ("MPD-SPEC-0001", "ACTIVE", "Existing SPEC"),
    ]
    _write_yaml(
        tmp_path / "developer/policy/index.yaml",
        {
            "schema_version": "ptsip-developer-policy-index/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "policies": [
                {
                    "id": policy_id,
                    "path": (
                        f"developer/policy/SPEC/{policy_id}.yaml"
                        if policy_id.startswith("MPD-SPEC-")
                        else f"developer/policy/{policy_id}.yaml"
                    ),
                    "status": status,
                }
                for policy_id, status, _ in policies
            ],
        },
    )

    for policy_id, status, title in policies:
        path = (
            tmp_path / f"developer/policy/SPEC/{policy_id}.yaml"
            if policy_id.startswith("MPD-SPEC-")
            else tmp_path / f"developer/policy/{policy_id}.yaml"
        )
        _write_yaml(
            path,
            {
                "schema_version": "ptsip-developer-policy/v1",
                "policy_class": "PTSIP_DEVELOPER_POLICY",
                "policy": {
                    "id": policy_id,
                    "title": title,
                    "status": status,
                },
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
                    "registered_values": [
                        "MPD-0012",
                        "MPD-SPEC-0001",
                    ],
                }
            },
        },
    )

    _write_yaml(
        tmp_path / "developer/policy/analysis/registry.yaml",
        {
            "schema_version": "developer-policy-materialization-analysis-registry/v2",
            "artifact_class": "PTSIP_POLICY_MATERIALIZATION_ANALYSIS_REGISTRY",
            "bindings": [],
        },
    )
    neutralize_fixture_catalog(tmp_path)
    return tmp_path


def _approval(repo: Path) -> Path:
    path = repo / "developer/policy/approvals/MPA-test-family.yaml"
    _write_yaml(
        path,
        {
            "schema_version": "ptsip-policy-approval-provenance/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "approval": {
                "approval_id": "MPA-test-family",
                "decision": "APPROVED",
                "source_kind": "PROJECT_OWNER_DIRECT_INSTRUCTION",
                "decision_source": "USER_EXPLICIT",
                "source_reference": "test",
                "approval_scope": "FINAL_POLICY",
                "target_status": "DRAFT",
                "implementation_authorized": True,
                "policy_content_review_scope": "FULL",
                "requested_policy_id": "MPD-SPEC-0002",
                "recorded_at": "2026-09-29",
            },
        },
    )
    return path


def _family_analysis(repo: Path, searched_policy_ids: list[str]) -> Path:
    path = repo / "developer/policy/analysis/PRA-test-family.yaml"
    responsibility = _responsibility(
        "R01",
        "SPEC",
        "G01",
        searched_policy_ids=searched_policy_ids,
    )
    payload = _analysis_payload(
        [responsibility],
        [
            {
                "group_id": "G01",
                "family": "SPEC",
                "cohesion_key": "SPEC_COHESION",
                "responsibility_ids": ["R01"],
                "cohesion_rationale": "One new SPEC authority subject.",
            }
        ],
        owned_family_set=["SPEC"],
        split_required=False,
    )
    _write_yaml(path, payload)
    return path


def test_family_preflight_requires_complete_existing_authority_lookup(repo: Path) -> None:
    approval = _approval(repo)
    analysis = _family_analysis(repo, [])

    with pytest.raises(PolicyIdentityLifecycleError) as exc:
        preflight_family_policy(
            "SPEC",
            approval,
            analysis,
            "G01",
            root=repo,
            policy_class="PTSIP_DEVELOPER_POLICY",
        )
    assert exc.value.code == "RESPONSIBILITY_ANALYSIS_BLOCKED"


def test_family_register_binds_analysis_atomically(repo: Path) -> None:
    approval = _approval(repo)
    analysis = _family_analysis(repo, ["MPD-SPEC-0001"])

    preflight = preflight_family_policy(
        "SPEC",
        approval,
        analysis,
        "G01",
        root=repo,
        policy_class="PTSIP_DEVELOPER_POLICY",
    )
    assert preflight["allocated_policy_id"] == "MPD-SPEC-0002"

    policy_path = repo / "developer/policy/SPEC/MPD-SPEC-0002.yaml"
    _write_yaml(
        policy_path,
        {
            "schema_version": "ptsip-developer-policy/v1",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "policy": {
                "id": "MPD-SPEC-0002",
                "version": "0.0",
                "title": "New SPEC",
                "status": "DRAFT",
            },
            "rules": {"new_spec": {"enabled": True}},
        },
    )

    result = register_family_policy(
        "SPEC",
        approval,
        analysis,
        "G01",
        policy_path,
        root=repo,
        policy_class="PTSIP_DEVELOPER_POLICY",
    )
    assert result["status"] == "REGISTERED"
    assert result["analysis_id"] == "PRA-test"

    registry = yaml.safe_load(
        (repo / "developer/policy/analysis/registry.yaml").read_text(encoding="utf-8")
    )
    assert registry["bindings"] == [
        {
            "policy_id": "MPD-SPEC-0002",
            "policy_class": "PTSIP_DEVELOPER_POLICY",
            "family": "SPEC",
            "analysis_ref": "developer/policy/analysis/PRA-test-family.yaml",
            "analysis_id": "PRA-test",
            "group_id": "G01",
        }
    ]
