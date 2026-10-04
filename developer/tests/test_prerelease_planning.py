from __future__ import annotations

import shutil
from pathlib import Path

import pytest
import yaml

from developer.automation.planning.planning_entry_resolver import (
    PlanningEntryResolutionError,
    resolve_planning_entry,
)
from developer.automation.planning.planning_validator import validate_planning


ROOT = Path(__file__).resolve().parents[2]
PLAN = "developer/planning/0.3.8/0.3.8a3/index.yaml"
WU08 = "developer/planning/0.3.8/0.3.8a3/WU-08/WU-08.yaml"


@pytest.fixture
def planning_repo(tmp_path: Path) -> Path:
    shutil.copy2(ROOT / "pyproject.toml", tmp_path / "pyproject.toml")
    shutil.copytree(ROOT / "developer/planning/schemas", tmp_path / "developer/planning/schemas")
    shutil.copytree(
        ROOT / "developer/planning/0.3.8/0.3.8a3",
        tmp_path / "developer/planning/0.3.8/0.3.8a3",
    )
    shutil.copy2(ROOT / "developer/planning/index.yaml", tmp_path / "developer/planning/index.yaml")
    registry = "developer/policy/registries/governance-source-registry.yaml"
    (tmp_path / registry).parent.mkdir(parents=True)
    shutil.copy2(ROOT / registry, tmp_path / registry)
    binding_schema = "developer/bindings/schemas/policy-plan-bindings.schema.json"
    (tmp_path / binding_schema).parent.mkdir(parents=True)
    shutil.copy2(ROOT / binding_schema, tmp_path / binding_schema)
    return tmp_path


def _load(path: Path) -> dict:
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def _save(path: Path, payload: dict) -> None:
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def test_registered_prerelease_and_continuation_resolve_exactly(planning_repo: Path) -> None:
    assert validate_planning(planning_repo) == ()
    for branch in ("dev/0.3.8", "dev/0.3.8a3"):
        result = resolve_planning_entry(branch, root=planning_repo)
        assert result.entry_document == PLAN
        assert result.plan_version == "0.3.8a3"
        assert result.role == "INTEGRATION_CONTROL_PLANE"


def test_similar_unregistered_branch_stays_unresolved(planning_repo: Path) -> None:
    with pytest.raises(PlanningEntryResolutionError) as error:
        resolve_planning_entry("dev/0.3.8a30", root=planning_repo)
    assert error.value.code == "UNKNOWN_PLANNING_ENTRY"


def test_duplicate_branch_registration_is_ambiguous(planning_repo: Path) -> None:
    root_path = planning_repo / "developer/planning/index.yaml"
    root = _load(root_path)
    root["plans"].append(root["plans"][0])
    _save(root_path, root)
    assert validate_planning(planning_repo)
    with pytest.raises(PlanningEntryResolutionError) as error:
        resolve_planning_entry("dev/0.3.8", root=planning_repo)
    assert error.value.code == "AMBIGUOUS_PLANNING_ENTRY"


@pytest.mark.parametrize("tamper", ["missing", "identity", "dependencies", "completion"])
def test_invalid_registered_work_unit_blocks_validation(planning_repo: Path, tamper: str) -> None:
    path = planning_repo / WU08
    if tamper == "missing":
        path.unlink()
    else:
        payload = _load(path)
        if tamper == "identity":
            payload["plan_id"] = "OTHER_PLAN"
        elif tamper == "dependencies":
            payload["work_unit"]["depends_on"] = []
        else:
            payload["work_unit"]["lifecycle"]["status"] = "COMPLETE"
            payload.pop("completion_evidence", None)
        _save(path, payload)
    assert validate_planning(planning_repo)


def test_incomplete_gate_dependency_blocks_validation(planning_repo: Path) -> None:
    path = planning_repo / "developer/planning/0.3.8/0.3.8a3/WU-06/WU-06.yaml"
    payload = _load(path)
    payload["work_unit"]["lifecycle"]["status"] = "BLOCKED"
    _save(path, payload)
    errors = validate_planning(planning_repo)
    assert any("current_gate dependency WU-06 is not complete" in error for error in errors)


def test_prerelease_path_cannot_be_rebound_to_another_identity(planning_repo: Path) -> None:
    path = planning_repo / PLAN
    payload = _load(path)
    payload["plan"]["prerelease"] = "0.3.8a4"
    _save(path, payload)
    errors = validate_planning(planning_repo)
    assert any("prerelease canonical location" in error for error in errors)


def test_dependency_order_cannot_omit_or_repeat_work_units(planning_repo: Path) -> None:
    path = planning_repo / PLAN
    payload = _load(path)
    payload["execution_model"]["dependency_order"][-1] = ["WU-07"]
    _save(path, payload)
    errors = validate_planning(planning_repo)
    assert any("every indexed work unit exactly once" in error for error in errors)


@pytest.mark.parametrize("field,value", [
    ("resolved_plan_id", "PLAN_FROM_BRANCH_NAME"),
    ("plan_file_id", "developer/planning/0.3.8/index.yaml"),
    ("version", ""),
    ("revision", ""),
])
def test_formal_identity_must_match_canonical_binding_contract(
    planning_repo: Path, field: str, value: str,
) -> None:
    path = planning_repo / PLAN
    payload = _load(path)
    payload["plan_identity"][field] = value
    _save(path, payload)
    assert validate_planning(planning_repo)


def test_registered_prerelease_requires_formal_plan_identity(planning_repo: Path) -> None:
    path = planning_repo / PLAN
    payload = _load(path)
    payload.pop("plan_identity")
    _save(path, payload)
    assert any("plan_identity" in error for error in validate_planning(planning_repo))


def test_identity_grammar_is_loaded_from_binding_contract(planning_repo: Path) -> None:
    import json

    schema_path = planning_repo / "developer/bindings/schemas/policy-plan-bindings.schema.json"
    schema = json.loads(schema_path.read_text(encoding="utf-8"))
    schema["$defs"]["binding"]["properties"]["resolved_plan_id"]["pattern"] = "^REGISTERED_ID$"
    schema_path.write_text(json.dumps(schema), encoding="utf-8")
    path = planning_repo / PLAN
    payload = _load(path)
    payload["plan_identity"]["resolved_plan_id"] = "REGISTERED_ID"
    _save(path, payload)
    assert validate_planning(planning_repo) == ()
