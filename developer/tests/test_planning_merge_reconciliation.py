from __future__ import annotations

from pathlib import Path

import pytest
import yaml

from developer.automation.planning_entry_resolver import (
    PlanningEntryResolutionError,
    resolve_planning_entry,
)
from developer.automation.planning_merge_reconciler import (
    PlanningStateReconciliationError,
    build_materialized_state,
    reconcile_planning_state,
)


def _approval() -> dict[str, object]:
    return {"status": "APPROVED", "inherited_from": []}


def _write_yaml(path: Path, payload: dict[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        yaml.safe_dump(payload, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )


def _work_unit(
    work_unit_id: str,
    *,
    status: str,
    authorization: str,
    depends_on: list[str],
    extension: bool = False,
    completion: bool = False,
) -> dict[str, object]:
    payload: dict[str, object] = {
        "schema_version": "ptsip-work-unit/v1",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "plan_version": "0.4.0",
        "planning_contract": "ptsip-planning/v1",
        "work_unit": {
            "id": work_unit_id,
            "kind": "WORK_UNIT",
            "title": work_unit_id,
            "classification": "CORE",
            "lifecycle": {"status": status},
            "approval": _approval(),
            "integration_branch": "dev/0.4.0",
            "implementation_authorization": authorization,
            "depends_on": depends_on,
        },
    }
    if extension:
        payload["extensions"] = [
            {
                "id": "WU-02-P01",
                "path": "docs/planning/0.4.0/WU-02/WU-02-P01.yaml",
                "status": "ACTIVE",
            }
        ]
    if completion:
        payload["completion_evidence"] = [
            {
                "type": "MACHINE_VALIDATION",
                "id": f"{work_unit_id}-focused",
                "result": "PASS",
            }
        ]
    return payload


def _repo(
    tmp_path: Path,
    *,
    wu02_status: str = "ACTIVE",
    wu02_extension_status: str = "ACTIVE",
    wu04_status: str = "ACTIVE",
    wu04_complete_evidence: bool = False,
    wu04_entry_state: str = "ACTIVE",
    current_gate: str = "WU-02-P01",
) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir()
    (repo / "pyproject.toml").write_text(
        "[project]\nname='planning-fixture'\nversion='0.0.0'\n",
        encoding="utf-8",
    )

    root = {
        "schema_version": "ptsip-developer-planning-root/v1",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "plans": [
            {
                "plan_version": "0.4.0",
                "path": "docs/planning/0.4.0/index.yaml",
                "status": "ACTIVE",
                "integration_branch": "dev/0.4.0",
                "entry_routing": {
                    "model": "PARALLEL_DEPENDENCY_EXTRACTION",
                    "canonical_plan": "planning/0.4.0.md",
                    "responsibility_control_plane": "docs/planning/0.4.0/index.yaml",
                    "branch_creation_parent": "dev/0.4.0",
                    "resolver": {
                        "module": "developer.automation.planning_entry_resolver",
                        "command": "python -m developer.automation.planning_entry_resolver",
                        "branch_source": "GIT_CURRENT_BRANCH",
                        "match_mode": "EXACT_BRANCH",
                        "unmatched_behavior": "FAIL_CLOSED",
                        "ambiguous_behavior": "FAIL_CLOSED",
                    },
                    "merge_reconciliation": {
                        "module": "developer.automation.planning_merge_reconciler",
                        "reconcile_command": "python -m developer.automation.planning_merge_reconciler",
                        "merge_command": "python -m developer.automation.planning_leaf_merge",
                        "target_branch": "dev/0.4.0",
                        "state_source": "WORK_UNIT_DOCUMENTS",
                        "leaf_shared_index_mutation": "FORBIDDEN",
                        "merged_leaf_entrypoint_state": "MERGED",
                        "unfinished_merged_leaf_continuation": "INTEGRATION_BRANCH",
                        "gate_policy": {
                            "preserve_nonterminal_current_gate": True,
                            "merged_unfinished_leaf_after_gate_closure": True,
                            "integrated_unfinished_leaf_before_unmerged_leaf": True,
                            "unmerged_leaf_order": "DECLARATION_ORDER",
                            "fallback": "FIRST_DEPENDENCY_SATISFIED_NONTERMINAL_WU",
                        },
                    },
                    "branch_entrypoints": [
                        {
                            "branch": "dev/0.4.0",
                            "entry_document": "docs/planning/0.4.0/index.yaml",
                            "role": "INTEGRATION_CONTROL_PLANE",
                            "state": "ACTIVE",
                        },
                        {
                            "branch": "dev/0.4.0-WU-04",
                            "entry_document": "docs/planning/0.4.0/WU-04/WU-04.yaml",
                            "role": "INDEPENDENT_LEAF",
                            "work_unit": "WU-04",
                            "state": wu04_entry_state,
                            **(
                                {"merged_into": "dev/0.4.0"}
                                if wu04_entry_state == "MERGED"
                                else {}
                            ),
                        },
                        {
                            "branch": "dev/0.4.0-WU-05",
                            "entry_document": "docs/planning/0.4.0/WU-05/WU-05.yaml",
                            "role": "INDEPENDENT_LEAF",
                            "work_unit": "WU-05",
                            "state": "ACTIVE",
                        },
                    ],
                    "dependency_bearing_convergence": {
                        "id": "WU-03",
                        "responsibility": "INTEGRATION",
                    },
                    "independent_leaf_work_units": [
                        {
                            "id": "WU-04",
                            "branch": "dev/0.4.0-WU-04",
                            "responsibility": "EXECUTION",
                        },
                        {
                            "id": "WU-05",
                            "branch": "dev/0.4.0-WU-05",
                            "responsibility": "VERIFICATION",
                        },
                    ],
                },
            }
        ],
    }
    _write_yaml(repo / "docs/planning/index.yaml", root)

    version = {
        "schema_version": "ptsip-developer-planning-index/v1",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "plan": {
            "plan_version": "0.4.0",
            "revision": "04",
            "integration_branch": "dev/0.4.0",
            "classification": "CORE",
            "status": "ACTIVE",
            "control_plane": "docs/planning/0.4.0/index.yaml",
            "current_gate": current_gate,
        },
        "work_units": [
            {
                "id": "WU-01",
                "lifecycle": {"status": "COMPLETE"},
                "approval": {"status": "NOT_REQUIRED", "inherited_from": []},
                "implementation_authorization": {"status": "COMPLETE"},
                "depends_on": [],
                "completion_evidence": [
                    {"type": "MACHINE_VALIDATION", "id": "wu01", "result": "PASS"}
                ],
            },
            {
                "id": "WU-02",
                "path": "docs/planning/0.4.0/WU-02/WU-02.yaml",
                "lifecycle": {"status": wu02_status},
                "approval": _approval(),
                "implementation_authorization": {
                    "status": "COMPLETE" if wu02_status == "COMPLETE" else "AUTHORIZED"
                },
                "depends_on": ["WU-01"],
                **(
                    {
                        "completion_evidence": [
                            {
                                "type": "MACHINE_VALIDATION",
                                "id": "wu02",
                                "result": "PASS",
                            }
                        ]
                    }
                    if wu02_status == "COMPLETE"
                    else {}
                ),
            },
            {
                "id": "WU-03",
                "path": "docs/planning/0.4.0/WU-03/WU-03.yaml",
                "lifecycle": {"status": "ACTIVE"},
                "approval": _approval(),
                "implementation_authorization": {"status": "AUTHORIZED"},
                "depends_on": ["WU-02"],
            },
            {
                "id": "WU-04",
                "path": "docs/planning/0.4.0/WU-04/WU-04.yaml",
                "lifecycle": {"status": wu04_status},
                "approval": _approval(),
                "implementation_authorization": {
                    "status": "COMPLETE" if wu04_status == "COMPLETE" else "AUTHORIZED"
                },
                "depends_on": [],
                **(
                    {
                        "completion_evidence": [
                            {
                                "type": "MACHINE_VALIDATION",
                                "id": "WU-04-focused",
                                "result": "PASS",
                            }
                        ]
                    }
                    if wu04_status == "COMPLETE" and wu04_complete_evidence
                    else {}
                ),
            },
            {
                "id": "WU-05",
                "path": "docs/planning/0.4.0/WU-05/WU-05.yaml",
                "lifecycle": {"status": "ACTIVE"},
                "approval": _approval(),
                "implementation_authorization": {"status": "AUTHORIZED"},
                "depends_on": [],
            },
        ],
    }
    _write_yaml(repo / "docs/planning/0.4.0/index.yaml", version)

    _write_yaml(
        repo / "docs/planning/0.4.0/WU-02/WU-02.yaml",
        _work_unit(
            "WU-02",
            status=wu02_status,
            authorization="COMPLETE" if wu02_status == "COMPLETE" else "AUTHORIZED",
            depends_on=["WU-01"],
            extension=True,
            completion=wu02_status == "COMPLETE",
        ),
    )
    _write_yaml(
        repo / "docs/planning/0.4.0/WU-02/WU-02-P01.yaml",
        {
            "extension": {
                "id": "WU-02-P01",
                "lifecycle": {"status": wu02_extension_status},
            }
        },
    )
    _write_yaml(
        repo / "docs/planning/0.4.0/WU-03/WU-03.yaml",
        _work_unit(
            "WU-03",
            status="ACTIVE",
            authorization="AUTHORIZED",
            depends_on=["WU-02"],
        ),
    )
    _write_yaml(
        repo / "docs/planning/0.4.0/WU-04/WU-04.yaml",
        _work_unit(
            "WU-04",
            status=wu04_status,
            authorization="COMPLETE" if wu04_status == "COMPLETE" else "AUTHORIZED",
            depends_on=[],
            completion=wu04_complete_evidence,
        ),
    )
    _write_yaml(
        repo / "docs/planning/0.4.0/WU-05/WU-05.yaml",
        _work_unit(
            "WU-05",
            status="ACTIVE",
            authorization="AUTHORIZED",
            depends_on=[],
        ),
    )
    return repo


def test_completed_wu04_merge_does_not_preempt_active_wu02(tmp_path: Path) -> None:
    repo = _repo(
        tmp_path,
        wu04_status="COMPLETE",
        wu04_complete_evidence=True,
    )

    result = reconcile_planning_state(
        root=repo,
        merged_branch="dev/0.4.0-WU-04",
        current_branch="dev/0.4.0",
    )

    assert result.merged_work_unit == "WU-04"
    assert result.current_gate_after == "WU-02-P01"


def test_wu02_and_wu04_complete_select_wu05(tmp_path: Path) -> None:
    repo = _repo(
        tmp_path,
        wu02_status="COMPLETE",
        wu02_extension_status="COMPLETE",
        wu04_status="COMPLETE",
        wu04_complete_evidence=True,
    )

    result = reconcile_planning_state(
        root=repo,
        merged_branch="dev/0.4.0-WU-04",
        current_branch="dev/0.4.0",
    )

    assert result.current_gate_after == "WU-05"
    assert result.current_gate_document.endswith("/WU-05/WU-05.yaml")


def test_unfinished_wu04_merge_continues_on_parent_after_wu02(tmp_path: Path) -> None:
    repo = _repo(
        tmp_path,
        wu02_status="COMPLETE",
        wu02_extension_status="COMPLETE",
        wu04_status="ACTIVE",
    )

    result = reconcile_planning_state(
        root=repo,
        merged_branch="dev/0.4.0-WU-04",
        current_branch="dev/0.4.0",
    )

    assert result.current_gate_after == "WU-04"


def test_active_wu02_is_not_preempted_by_unfinished_wu04_merge(tmp_path: Path) -> None:
    repo = _repo(tmp_path, wu04_status="ACTIVE")

    result = reconcile_planning_state(
        root=repo,
        merged_branch="dev/0.4.0-WU-04",
        current_branch="dev/0.4.0",
    )

    assert result.current_gate_after == "WU-02-P01"


def test_later_wu02_completion_resumes_previously_merged_wu04(tmp_path: Path) -> None:
    repo = _repo(
        tmp_path,
        wu02_status="COMPLETE",
        wu02_extension_status="COMPLETE",
        wu04_status="ACTIVE",
        wu04_entry_state="MERGED",
    )

    result = reconcile_planning_state(
        root=repo,
        current_branch="dev/0.4.0",
    )

    assert result.current_gate_after == "WU-04"

    root_index = yaml.safe_load((repo / "docs/planning/index.yaml").read_text(encoding="utf-8"))
    version_index = yaml.safe_load(
        (repo / "docs/planning/0.4.0/index.yaml").read_text(encoding="utf-8")
    )
    root_plan = root_index["plans"][0]
    root_plan["materialized_state"] = build_materialized_state(
        root_plan,
        version_index,
        base=repo,
    )
    wu04 = next(
        entry
        for entry in root_plan["materialized_state"]["work_units"]
        if entry["id"] == "WU-04"
    )
    assert wu04["execution_location"] == "INTEGRATION_BRANCH"
    assert wu04["branch"] == "dev/0.4.0"


def test_merged_leaf_is_not_an_active_resolver_entry(tmp_path: Path) -> None:
    repo = _repo(tmp_path, wu04_entry_state="MERGED")

    with pytest.raises(PlanningEntryResolutionError) as error:
        resolve_planning_entry("dev/0.4.0-WU-04", root=repo)

    assert error.value.code == "UNKNOWN_PLANNING_ENTRY"


def test_complete_work_unit_without_completion_evidence_fails_closed(tmp_path: Path) -> None:
    repo = _repo(
        tmp_path,
        wu04_status="COMPLETE",
        wu04_complete_evidence=False,
    )

    with pytest.raises(PlanningStateReconciliationError) as error:
        reconcile_planning_state(
            root=repo,
            merged_branch="dev/0.4.0-WU-04",
            current_branch="dev/0.4.0",
        )

    assert error.value.code == "MISSING_COMPLETION_EVIDENCE"
