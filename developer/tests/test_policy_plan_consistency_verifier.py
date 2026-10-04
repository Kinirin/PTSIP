from __future__ import annotations

import shutil
from pathlib import Path

import yaml

from developer.automation.policy_loader import repository_root
from developer.tests.policy_contract_fixtures import write_catalog_fixture
from developer.automation.policy_plan_binding.manager import create_binding, link_plan
from developer.automation.verification.policy_plan_consistency import (
    verify_policy_plan_consistency,
)


RESOLVED_PLAN_ID = "PLN.MIGR.READ.A7k2Q9mX"
PLAN_FILE_ID = "PLANFILE.MAIN.H7sP2kQ9mXa4"


def _repo(tmp_path: Path) -> Path:
    root = tmp_path / "repo"
    root.mkdir()
    (root / "pyproject.toml").write_text(
        "[project]\nname='verifier-test'\nversion='0'\n",
        encoding="utf-8",
    )

    source = repository_root()
    schema_src = source / "developer/bindings/schemas/policy-plan-bindings.schema.json"
    schema_dst = root / "developer/bindings/schemas/policy-plan-bindings.schema.json"
    schema_dst.parent.mkdir(parents=True)
    shutil.copyfile(schema_src, schema_dst)

    registry = root / "developer/bindings/policy-plan-bindings.yaml"
    registry.parent.mkdir(parents=True, exist_ok=True)
    registry.write_text(
        "schema_version: ptsip-policy-plan-bindings/v2\n"
        "registry_role: POLICY_PLAN_BINDING\n"
        "schema_ref: developer/bindings/schemas/policy-plan-bindings.schema.json\n"
        "bindings: []\n",
        encoding="utf-8",
    )

    policy_index = root / "developer/policy/index.yaml"
    policy_index.parent.mkdir(parents=True, exist_ok=True)
    policy_index.write_text(
        "schema_version: test\n"
        "policy_class: PTSIP_DEVELOPER_POLICY\n"
        "policies:\n"
        "- id: MPD-0001\n"
        "  path: developer/policy/MPD-0001.yaml\n"
        "  status: ACTIVE\n",
        encoding="utf-8",
    )
    write_catalog_fixture(root, ["MPD-0001"])
    return root


def _write_plan(
    root: Path,
    relative: str,
    *,
    resolved_plan_id: str = RESOLVED_PLAN_ID,
    plan_file_id: str = PLAN_FILE_ID,
    version: str = "1.0",
    revision: str = "Rev.0001",
) -> None:
    path = root / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        yaml.safe_dump(
            {
                "plan_identity": {
                    "resolved_plan_id": resolved_plan_id,
                    "plan_file_id": plan_file_id,
                    "version": version,
                    "revision": revision,
                }
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )


def _materialize(root: Path, plan_ref: str) -> str:
    created = create_binding("MPD-0001", root=root)
    _write_plan(root, plan_ref)
    link_plan(
        policy_ref="MPD-0001",
        binding_id=created.binding["binding_id"],
        resolved_plan_id=RESOLVED_PLAN_ID,
        plan_file_id=PLAN_FILE_ID,
        version="1.0",
        revision="Rev.0001",
        plan_ref=plan_ref,
        root=root,
    )
    return created.binding["binding_id"]


def _codes(report: object) -> set[str]:
    return {failure.code for failure in report.failures}


def test_empty_registry_passes_without_inventing_targets(tmp_path: Path) -> None:
    root = _repo(tmp_path)

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "PASS"
    assert report.binding_count == 0
    assert report.checked_binding_count == 0
    assert report.failures == ()


def test_not_created_binding_is_a_valid_registered_relationship(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    create_binding("MPD-0001", root=root)

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "PASS"
    assert report.binding_count == 1


def test_current_created_binding_passes_exact_consistency(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    plan_ref = "developer/planning/current/plan.yaml"
    _materialize(root, plan_ref)

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "PASS"
    assert report.binding_count == 1
    assert report.failures == ()


def test_verifier_is_read_only_when_plan_move_is_detected(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    old_ref = "developer/planning/old/plan.yaml"
    new_ref = "developer/planning/new/plan.yaml"
    _materialize(root, old_ref)

    old_path = root / old_ref
    new_path = root / new_ref
    new_path.parent.mkdir(parents=True, exist_ok=True)
    old_path.replace(new_path)

    registry = root / "developer/bindings/policy-plan-bindings.yaml"
    before = registry.read_bytes()

    report = verify_policy_plan_consistency(root=root)

    after = registry.read_bytes()
    assert report.status == "FAIL"
    assert "PLAN_REF_RECONCILE_REQUIRED" in _codes(report)
    assert before == after


def test_verifier_fails_when_plan_file_identity_cannot_be_resolved(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    plan_ref = "developer/planning/current/plan.yaml"
    _materialize(root, plan_ref)
    (root / plan_ref).unlink()

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "FAIL"
    assert "PLAN_REF_UNRESOLVED" in _codes(report)


def test_verifier_fails_on_version_currentness_mismatch(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    plan_ref = "developer/planning/current/plan.yaml"
    _materialize(root, plan_ref)
    _write_plan(root, plan_ref, version="2.0")

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "FAIL"
    assert "PLAN_VERSION_MISMATCH" in _codes(report)


def test_verifier_fails_closed_on_duplicate_plan_file_identity(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    plan_ref = "developer/planning/current/plan.yaml"
    _materialize(root, plan_ref)
    _write_plan(root, "developer/planning/duplicate/plan.yaml")

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "FAIL"
    assert "PLAN_FILE_ID_AMBIGUOUS" in _codes(report)


def test_schema_failure_is_reported_without_mutation(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    registry = root / "developer/bindings/policy-plan-bindings.yaml"
    registry.write_text(
        "schema_version: ptsip-policy-plan-bindings/v1\n"
        "registry_role: POLICY_PLAN_BINDING\n"
        "schema_ref: developer/bindings/schemas/policy-plan-bindings.schema.json\n"
        "bindings: []\n",
        encoding="utf-8",
    )
    before = registry.read_bytes()

    report = verify_policy_plan_consistency(root=root)

    assert report.status == "FAIL"
    assert "BINDING_REGISTRY_SCHEMA_INVALID" in _codes(report)
    assert registry.read_bytes() == before
