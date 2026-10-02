from __future__ import annotations

import shutil
from pathlib import Path

import pytest
import yaml

from developer.automation.policy_loader import repository_root
from developer.automation.policy_plan_binding.errors import PolicyPlanBindingError
from developer.automation.policy_plan_binding.manager import (
    create_binding,
    link_plan,
    move_plan_ref,
)
from developer.automation.policy_plan_binding.registry import load_registry


RESOLVED_PLAN_ID = "PLN.MIGR.READ.A7k2Q9mX"
PLAN_FILE_ID = "PLANFILE.MAIN.H7sP2kQ9mXa4"


def _repo(tmp_path: Path) -> Path:
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


def _link(root: Path, binding_id: str | None = None, *, plan_ref: str) -> object:
    return link_plan(
        policy_ref="MPD-0001",
        binding_id=binding_id,
        resolved_plan_id=RESOLVED_PLAN_ID,
        plan_file_id=PLAN_FILE_ID,
        version="1.0",
        revision="Rev.0001",
        plan_ref=plan_ref,
        root=root,
    )


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


def test_link_plan_uses_v2_identity_fields(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    created = create_binding("MPD-0001", root=root)
    plan_ref = "developer/planning/0.3.8/WU-07/WU-07.yaml"
    _write_plan(root, plan_ref)

    linked = _link(root, plan_ref=plan_ref)

    assert linked.binding == {
        "binding_id": created.binding["binding_id"],
        "policy_ref": "MPD-0001",
        "planning_state": "CREATED",
        "resolved_plan_id": RESOLVED_PLAN_ID,
        "plan_file_id": PLAN_FILE_ID,
        "version": "1.0",
        "revision": "Rev.0001",
        "plan_ref": plan_ref,
    }
    assert "plan_id" not in linked.binding
    assert "registered_plan_id" not in linked.binding


def test_link_plan_requires_exact_plan_document_identity(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    create_binding("MPD-0001", root=root)
    plan_ref = "developer/planning/0.3.8/WU-07/WU-07.yaml"
    _write_plan(root, plan_ref, plan_file_id="PLANFILE.MAIN.Z9ap7Kx21QmB")

    with pytest.raises(PolicyPlanBindingError) as exc:
        _link(root, plan_ref=plan_ref)

    assert exc.value.code == "PLAN_FILE_ID_MISMATCH"


def test_link_plan_fails_closed_when_uncreated_binding_is_ambiguous(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    create_binding("MPD-0001", root=root)
    create_binding("MPD-0001", root=root)
    plan_ref = "developer/planning/0.3.8/WU-07/WU-07.yaml"
    _write_plan(root, plan_ref)

    with pytest.raises(PolicyPlanBindingError) as exc:
        _link(root, plan_ref=plan_ref)

    assert exc.value.code == "AMBIGUOUS_UNCREATED_BINDING"


def test_move_updates_only_ref_with_exact_identity_guards(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    created = create_binding("MPD-0001", root=root)
    old_ref = "developer/planning/old/WU-07.yaml"
    new_ref = "developer/planning/new/WU-07.yaml"
    _write_plan(root, old_ref)
    _link(root, binding_id=created.binding["binding_id"], plan_ref=old_ref)

    # Version/revision can differ at the destination. move_plan_ref owns only
    # physical reference movement and exact logical/file identity.
    _write_plan(root, new_ref, version="1.1", revision="Rev.0002")

    moved = move_plan_ref(
        binding_id=created.binding["binding_id"],
        policy_ref="MPD-0001",
        resolved_plan_id=RESOLVED_PLAN_ID,
        plan_file_id=PLAN_FILE_ID,
        from_plan_ref=old_ref,
        to_plan_ref=new_ref,
        root=root,
    )

    assert moved.binding["resolved_plan_id"] == RESOLVED_PLAN_ID
    assert moved.binding["plan_file_id"] == PLAN_FILE_ID
    assert moved.binding["version"] == "1.0"
    assert moved.binding["revision"] == "Rev.0001"
    assert moved.binding["plan_ref"] == new_ref

    snapshot = load_registry(root)
    assert snapshot is not None
    stored = snapshot.payload["bindings"][0]
    assert stored["resolved_plan_id"] == RESOLVED_PLAN_ID
    assert stored["plan_file_id"] == PLAN_FILE_ID
    assert stored["version"] == "1.0"
    assert stored["revision"] == "Rev.0001"
    assert stored["plan_ref"] == new_ref


def test_move_fails_closed_on_stale_from_ref(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    created = create_binding("MPD-0001", root=root)
    old_ref = "developer/planning/old/WU-07.yaml"
    new_ref = "developer/planning/new/WU-07.yaml"
    _write_plan(root, old_ref)
    _write_plan(root, new_ref)
    _link(root, binding_id=created.binding["binding_id"], plan_ref=old_ref)

    with pytest.raises(PolicyPlanBindingError) as exc:
        move_plan_ref(
            binding_id=created.binding["binding_id"],
            policy_ref="MPD-0001",
            resolved_plan_id=RESOLVED_PLAN_ID,
            plan_file_id=PLAN_FILE_ID,
            from_plan_ref="developer/planning/stale/WU-07.yaml",
            to_plan_ref=new_ref,
            root=root,
        )

    assert exc.value.code == "STALE_PLAN_REF"


def test_move_fails_closed_on_destination_identity_mismatch(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    created = create_binding("MPD-0001", root=root)
    old_ref = "developer/planning/old/WU-07.yaml"
    new_ref = "developer/planning/new/WU-07.yaml"
    _write_plan(root, old_ref)
    _write_plan(root, new_ref, resolved_plan_id="PLN.VERI.BIND.Q8m2Za1K")
    _link(root, binding_id=created.binding["binding_id"], plan_ref=old_ref)

    with pytest.raises(PolicyPlanBindingError) as exc:
        move_plan_ref(
            binding_id=created.binding["binding_id"],
            policy_ref="MPD-0001",
            resolved_plan_id=RESOLVED_PLAN_ID,
            plan_file_id=PLAN_FILE_ID,
            from_plan_ref=old_ref,
            to_plan_ref=new_ref,
            root=root,
        )

    assert exc.value.code == "RESOLVED_PLAN_ID_MISMATCH"
