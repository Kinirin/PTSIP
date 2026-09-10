from __future__ import annotations

from pathlib import Path

import pytest

from developer.automation.planning_entry_resolver import (
    PlanningEntryResolutionError,
    main,
    resolve_planning_entry,
)


ROOT_INDEX = """schema_version: ptsip-developer-planning-root/v1
policy_class: PTSIP_DEVELOPER_POLICY

plans:
  - plan_version: 0.4.0
    path: docs/planning/0.4.0/index.yaml
    status: ACTIVE
    integration_branch: dev/0.4.0
    entry_routing:
      model: PARALLEL_DEPENDENCY_EXTRACTION
      canonical_plan: planning/0.4.0.md
      responsibility_control_plane: docs/planning/0.4.0/index.yaml
      branch_creation_parent: dev/0.4.0
      resolver:
        module: developer.automation.planning_entry_resolver
        command: python -m developer.automation.planning_entry_resolver
        branch_source: GIT_CURRENT_BRANCH
        match_mode: EXACT_BRANCH
        unmatched_behavior: FAIL_CLOSED
        ambiguous_behavior: FAIL_CLOSED
      branch_entrypoints:
        - branch: dev/0.4.0
          entry_document: docs/planning/0.4.0/index.yaml
          role: INTEGRATION_CONTROL_PLANE
        - branch: dev/0.4.0-WU-04
          entry_document: docs/planning/0.4.0/WU-04/WU-04.yaml
          role: INDEPENDENT_LEAF
          work_unit: WU-04
        - branch: dev/0.4.0-WU-05
          entry_document: docs/planning/0.4.0/WU-05/WU-05.yaml
          role: INDEPENDENT_LEAF
          work_unit: WU-05
      dependency_bearing_convergence:
        id: WU-03
        responsibility: DOWNSTREAM_DEPENDENCY_BEARING_INTEGRATION_AND_ORCHESTRATION
      independent_leaf_work_units:
        - id: WU-04
          branch: dev/0.4.0-WU-04
          responsibility: EXECUTION
        - id: WU-05
          branch: dev/0.4.0-WU-05
          responsibility: VERIFICATION
"""


def _repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    (repo / "docs/planning/0.4.0/WU-04").mkdir(parents=True)
    (repo / "docs/planning/0.4.0/WU-05").mkdir(parents=True)
    (repo / "pyproject.toml").write_text("[project]\nname='fixture'\nversion='0.0.0'\n", encoding="utf-8")
    (repo / "docs/planning/index.yaml").write_text(ROOT_INDEX, encoding="utf-8")
    (repo / "docs/planning/0.4.0/index.yaml").write_text("plan: fixture\n", encoding="utf-8")
    (repo / "docs/planning/0.4.0/WU-04/WU-04.yaml").write_text("work_unit: WU-04\n", encoding="utf-8")
    (repo / "docs/planning/0.4.0/WU-05/WU-05.yaml").write_text("work_unit: WU-05\n", encoding="utf-8")
    return repo


@pytest.mark.parametrize(
    ("branch", "entry_document", "work_unit", "role"),
    [
        ("dev/0.4.0", "docs/planning/0.4.0/index.yaml", None, "INTEGRATION_CONTROL_PLANE"),
        (
            "dev/0.4.0-WU-04",
            "docs/planning/0.4.0/WU-04/WU-04.yaml",
            "WU-04",
            "INDEPENDENT_LEAF",
        ),
        (
            "dev/0.4.0-WU-05",
            "docs/planning/0.4.0/WU-05/WU-05.yaml",
            "WU-05",
            "INDEPENDENT_LEAF",
        ),
    ],
)
def test_exact_branch_resolves_declared_entry(
    tmp_path: Path,
    branch: str,
    entry_document: str,
    work_unit: str | None,
    role: str,
):
    repo = _repo(tmp_path)

    resolved = resolve_planning_entry(branch, root=repo)

    assert resolved.status == "RESOLVED"
    assert resolved.branch == branch
    assert resolved.entry_document == entry_document
    assert resolved.work_unit == work_unit
    assert resolved.role == role


def test_similar_branch_name_does_not_fuzzy_match(tmp_path: Path):
    repo = _repo(tmp_path)

    with pytest.raises(PlanningEntryResolutionError) as error:
        resolve_planning_entry("dev/0.4.0-WU-05-extra", root=repo)

    assert error.value.code == "UNKNOWN_PLANNING_ENTRY"


def test_duplicate_exact_branch_fails_closed(tmp_path: Path):
    repo = _repo(tmp_path)
    root_index = repo / "docs/planning/index.yaml"
    content = root_index.read_text(encoding="utf-8")
    content += """
  - plan_version: 0.4.1
    path: docs/planning/0.4.0/index.yaml
    status: ACTIVE
    integration_branch: dev/0.4.1
    entry_routing:
      branch_entrypoints:
        - branch: dev/0.4.0-WU-04
          entry_document: docs/planning/0.4.0/WU-04/WU-04.yaml
          role: INDEPENDENT_LEAF
          work_unit: WU-04
"""
    root_index.write_text(content, encoding="utf-8")

    with pytest.raises(PlanningEntryResolutionError) as error:
        resolve_planning_entry("dev/0.4.0-WU-04", root=repo)

    assert error.value.code == "AMBIGUOUS_PLANNING_ENTRY"


def test_missing_entry_document_fails_closed(tmp_path: Path):
    repo = _repo(tmp_path)
    (repo / "docs/planning/0.4.0/WU-05/WU-05.yaml").unlink()

    with pytest.raises(PlanningEntryResolutionError) as error:
        resolve_planning_entry("dev/0.4.0-WU-05", root=repo)

    assert error.value.code == "MISSING_ENTRY_DOCUMENT"


def test_cli_path_only_is_machine_consumable(tmp_path: Path, capsys):
    repo = _repo(tmp_path)

    assert (
        main(
            [
                "--root",
                str(repo),
                "--branch",
                "dev/0.4.0-WU-04",
                "--path-only",
            ]
        )
        == 0
    )

    assert capsys.readouterr().out.strip() == "docs/planning/0.4.0/WU-04/WU-04.yaml"


def test_cli_unknown_branch_returns_nonzero_and_machine_status(tmp_path: Path, capsys):
    repo = _repo(tmp_path)

    assert main(["--root", str(repo), "--branch", "dev/0.4.0-WU-99"]) == 2

    output = capsys.readouterr().out
    assert '"status": "UNRESOLVED"' in output
    assert '"code": "UNKNOWN_PLANNING_ENTRY"' in output
