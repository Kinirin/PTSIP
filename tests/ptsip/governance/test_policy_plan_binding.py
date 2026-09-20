from __future__ import annotations

import json
import shutil
from pathlib import Path

import pytest

from developer.automation.policy_loader import repository_root
from developer.automation.policy_plan_binding.errors import PolicyPlanBindingError
from developer.automation.policy_plan_binding.manager import (
    create_binding,
    link_plan,
    move_plan_ref,
)
from developer.automation.policy_plan_binding.registry import load_registry


def _repo(tmp_path: Path) -> Path:
    root = tmp_path / "repo"
    root.mkdir()
    (root / "pyproject.toml").write_text("[project]\nname='binding-test'\nversion='0'\n", encoding="utf-8")

    source = repository_root()
    schema_src = source / "developer/bindings/schemas/policy-plan-bindings.schema.json"
    schema_dst = root / "developer/bindings/schemas/policy-plan-bindings.schema.json"
    schema_dst.parent.mkdir(parents=True)
    shutil.copyfile(schema_src, schema_dst)

    registry = root / "developer/bindings/policy-plan-bindings.yaml"
    registry.parent.mkdir(parents=True, exist_ok=True)
    registry.write_text(
        "schema_version: ptsip-policy-plan-bindings/v1\n"
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
    return root


def test_binding_ids_are_allocated_and_not_created_has_no_plan_fields(tmp_path: Path) -> None:
    root = _repo(tmp_path)

    first = create_binding("MPD-0001", root=root)
    second = create_binding("MPD-0001", root=root)

    assert first.binding == {
        "binding_id": "PPB-0001",
        "policy_ref": "MPD-0001",
        "planning_state": "NOT_CREATED",
    }
    assert second.binding["binding_id"] == "PPB-0002"


def test_link_plan_automatically_uses_single_uncreated_binding(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    created = create_binding("MPD-0001", root=root)
    plan = root / "developer/planning/0.3.8/WU-07/WU-07.yaml"
    plan.parent.mkdir(parents=True)
    plan.write_text("plan_id: WU-07\n", encoding="utf-8")

    linked = link_plan(
        policy_ref="MPD-0001",
        plan_id="WU-07",
        plan_ref="developer/planning/0.3.8/WU-07/WU-07.yaml",
        root=root,
    )

    assert linked.binding["binding_id"] == created.binding["binding_id"]
    assert linked.binding["planning_state"] == "CREATED"
    assert linked.binding["plan_id"] == "WU-07"


def test_link_plan_fails_closed_when_uncreated_binding_is_ambiguous(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    create_binding("MPD-0001", root=root)
    create_binding("MPD-0001", root=root)
    plan = root / "developer/planning/0.3.8/WU-07/WU-07.yaml"
    plan.parent.mkdir(parents=True)
    plan.write_text("plan_id: WU-07\n", encoding="utf-8")

    with pytest.raises(PolicyPlanBindingError) as exc:
        link_plan(
            policy_ref="MPD-0001",
            plan_id="WU-07",
            plan_ref="developer/planning/0.3.8/WU-07/WU-07.yaml",
            root=root,
        )

    assert exc.value.code == "AMBIGUOUS_UNCREATED_BINDING"


def test_move_updates_ref_without_changing_plan_identity(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    created = create_binding("MPD-0001", root=root)
    old = root / "developer/planning/old/WU-07.yaml"
    old.parent.mkdir(parents=True)
    old.write_text("plan_id: WU-07\n", encoding="utf-8")
    link_plan(
        policy_ref="MPD-0001",
        binding_id=created.binding["binding_id"],
        plan_id="WU-07",
        plan_ref="developer/planning/old/WU-07.yaml",
        root=root,
    )

    new = root / "developer/planning/new/WU-07.yaml"
    new.parent.mkdir(parents=True)
    new.write_text("plan_id: WU-07\n", encoding="utf-8")
    moved = move_plan_ref(
        binding_id=created.binding["binding_id"],
        plan_ref="developer/planning/new/WU-07.yaml",
        root=root,
    )

    assert moved.binding["plan_id"] == "WU-07"
    assert moved.binding["plan_ref"] == "developer/planning/new/WU-07.yaml"

    snapshot = load_registry(root)
    assert snapshot is not None
    assert snapshot.payload["bindings"][0]["plan_id"] == "WU-07"
