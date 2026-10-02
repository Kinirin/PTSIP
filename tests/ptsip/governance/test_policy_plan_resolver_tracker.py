from __future__ import annotations

import shutil
from pathlib import Path

import pytest
import yaml

from developer.automation.policy_loader import repository_root
from developer.automation.policy_plan_binding.errors import PolicyPlanBindingError
from developer.automation.policy_plan_binding.ref_tracker import track_plan_ref
from developer.automation.policy_plan_binding.resolver import resolve_bindings


RESOLVED_PLAN_ID = "PLN.MIGR.READ.A7k2Q9mX"
PLAN_FILE_ID = "PLANFILE.MAIN.H7sP2kQ9mXa4"


def _repo(tmp_path: Path, *, plan_ref: str = "developer/planning/old/plan.yaml") -> Path:
    root = tmp_path / "repo"
    root.mkdir()
    (root / "pyproject.toml").write_text(
        "[project]\nname='binding-test'\nversion='0'\n",
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
        yaml.safe_dump(
            {
                "schema_version": "ptsip-policy-plan-bindings/v2",
                "registry_role": "POLICY_PLAN_BINDING",
                "schema_ref": "developer/bindings/schemas/policy-plan-bindings.schema.json",
                "bindings": [
                    {
                        "binding_id": "PPB-0001",
                        "policy_ref": "MPD-0001",
                        "planning_state": "CREATED",
                        "resolved_plan_id": RESOLVED_PLAN_ID,
                        "plan_file_id": PLAN_FILE_ID,
                        "version": "1.0",
                        "revision": "Rev.0001",
                        "plan_ref": plan_ref,
                    }
                ],
            },
            sort_keys=False,
        ),
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
    return root


def _write_plan(
    root: Path,
    relative: str,
    *,
    resolved_plan_id: str = RESOLVED_PLAN_ID,
    plan_file_id: str = PLAN_FILE_ID,
) -> None:
    path = root / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        yaml.safe_dump(
            {
                "plan_identity": {
                    "resolved_plan_id": resolved_plan_id,
                    "plan_file_id": plan_file_id,
                    "version": "1.1",
                    "revision": "Rev.0002",
                }
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )


def test_resolver_uses_v2_exact_identity_keys(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    by_resolved = resolve_bindings(
        resolved_plan_id=RESOLVED_PLAN_ID,
        root=root,
    )
    assert by_resolved.status == "BOUND"
    assert len(by_resolved.bindings) == 1
    assert by_resolved.resolved_plan_id == RESOLVED_PLAN_ID

    by_file = resolve_bindings(
        resolved_plan_id=RESOLVED_PLAN_ID,
        plan_file_id=PLAN_FILE_ID,
        root=root,
    )
    assert by_file.status == "BOUND"
    assert len(by_file.bindings) == 1
    assert by_file.plan_file_id == PLAN_FILE_ID


def test_resolver_requires_at_least_one_exact_key(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    with pytest.raises(PolicyPlanBindingError) as exc:
        resolve_bindings(root=root)
    assert exc.value.code == "BINDING_QUERY_EMPTY"


def test_tracker_reports_current_for_exact_registered_location(tmp_path: Path) -> None:
    current = "developer/planning/current/plan.yaml"
    root = _repo(tmp_path, plan_ref=current)
    _write_plan(root, current)

    result = track_plan_ref("PPB-0001", root=root)

    assert result.status == "CURRENT"
    assert result.discovered_plan_ref == current
    assert result.changed is False


def test_tracker_reports_unresolved_when_plan_file_identity_is_missing(tmp_path: Path) -> None:
    root = _repo(tmp_path)

    result = track_plan_ref("PPB-0001", root=root)

    assert result.status == "UNRESOLVED"
    assert result.discovered_plan_ref is None
    assert result.candidates == ()


def test_tracker_detects_moved_plan_without_path_inference(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    moved = "developer/planning/new/location.yaml"
    _write_plan(root, moved)

    result = track_plan_ref("PPB-0001", root=root)

    assert result.status == "RECONCILE_REQUIRED"
    assert result.current_plan_ref == "developer/planning/old/plan.yaml"
    assert result.discovered_plan_ref == moved
    assert result.candidates == (moved,)


def test_tracker_apply_updates_only_plan_ref(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    moved = "developer/planning/new/location.yaml"
    _write_plan(root, moved)

    result = track_plan_ref("PPB-0001", apply=True, root=root)

    assert result.status == "RECONCILED"
    assert result.applied is True

    resolved = resolve_bindings(binding_id="PPB-0001", root=root)
    binding = resolved.bindings[0]
    assert binding["plan_ref"] == moved
    assert binding["resolved_plan_id"] == RESOLVED_PLAN_ID
    assert binding["plan_file_id"] == PLAN_FILE_ID
    assert binding["version"] == "1.0"
    assert binding["revision"] == "Rev.0001"


def test_tracker_fails_closed_on_duplicate_plan_file_identity(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    _write_plan(root, "developer/planning/a/plan.yaml")
    _write_plan(root, "developer/planning/b/plan.yaml")

    with pytest.raises(PolicyPlanBindingError) as exc:
        track_plan_ref("PPB-0001", root=root)

    assert exc.value.code == "PLAN_FILE_ID_AMBIGUOUS"


def test_tracker_fails_closed_when_plan_file_id_binds_different_logical_plan(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    _write_plan(
        root,
        "developer/planning/new/location.yaml",
        resolved_plan_id="PLN.VERI.BIND.Q8m2Za1K",
    )

    with pytest.raises(PolicyPlanBindingError) as exc:
        track_plan_ref("PPB-0001", root=root)

    assert exc.value.code == "PLAN_FILE_ID_CONFLICT"


def test_tracker_ignores_yaml_without_explicit_plan_identity(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    unrelated = root / "developer/planning/index.yaml"
    unrelated.parent.mkdir(parents=True, exist_ok=True)
    unrelated.write_text("schema_version: unrelated\n", encoding="utf-8")
    moved = "developer/planning/new/location.yaml"
    _write_plan(root, moved)

    result = track_plan_ref("PPB-0001", root=root)

    assert result.status == "RECONCILE_REQUIRED"
    assert result.candidates == (moved,)
