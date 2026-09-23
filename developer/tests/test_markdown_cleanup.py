from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml

from developer.automation.markdown_cleanup import (
    MarkdownCleanupError,
    apply_cleanup_plan,
    inspect_repository,
    load_cleanup_workflow,
    scan_reference_hits,
)


ROOT = Path(__file__).resolve().parents[2]


def _repo(tmp_path: Path) -> Path:
    (tmp_path / "pyproject.toml").write_text("[project]\nname='synthetic'\n", encoding="utf-8")
    return tmp_path


def _write_workflow(root: Path, *, targets: list[dict[str, object]]) -> None:
    path = root / "developer/automation/markdown_cleanup_workflow.yaml"
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "schema_version": "ptsip-markdown-cleanup-workflow/v1",
        "artifact_class": "DEVELOPER_EXECUTION_PROJECTION",
        "projection_authority": False,
        "policy_ref": "MPD-0012#rules.approved_decisions.markdown_cleanup_preconditions",
        "stages": ["INSPECT", "PLAN", "VERIFY", "APPLY", "POST_VALIDATE"],
        "default_scopes": ["spec"],
        "reference_scan_exclusions": [
            "developer/automation/markdown_cleanup_workflow.yaml",
            "developer/policy/MPD-0012.yaml",
        ],
        "apply_requires_policy_status": "ACTIVE",
        "targets": targets,
    }
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def _write_policy(root: Path, *, status: str = "DRAFT") -> None:
    path = root / "developer/policy/MPD-0012.yaml"
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "schema_version": "ptsip-developer-policy/v1",
        "policy_class": "PTSIP_DEVELOPER_POLICY",
        "policy": {
            "id": "MPD-0012",
            "title": "Synthetic",
            "status": status,
        },
        "rules": {
            "approved_decisions": {
                "markdown_cleanup_preconditions": {
                    "decision": "APPROVED",
                }
            }
        },
    }
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def test_current_cleanup_workflow_is_machine_valid() -> None:
    workflow = load_cleanup_workflow(ROOT)

    assert workflow["projection_authority"] is False
    assert workflow["default_scopes"] == ["adoption", "agents", "spec"]
    assert "developer/tests/test_markdown_cleanup.py" in workflow["reference_scan_exclusions"]
    assert [item["path"] for item in workflow["targets"]] == [
        "adoption/ADOPTION-GUIDE.md",
        "agents/AGENT-CONTRACT.md",
        "spec/PTSIP-CONFORMANCE.md",
        "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
        "spec/PTSIP-GOVERNANCE.md",
        "spec/PTSIP-RESPONSIBILITY-MAP.md",
        "spec/PTSIP-SPEC.md",
        "spec/PTSIP-TERMINOLOGY.md",
    ]


def test_machine_reference_blocks_registered_target(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    target = root / "spec/example.md"
    target.parent.mkdir(parents=True)
    target.write_text("# Normative\n", encoding="utf-8")
    (root / "tool.py").write_text(
        'SOURCE = "spec/example.md"\n',
        encoding="utf-8",
    )
    _write_policy(root)
    _write_workflow(
        root,
        targets=[
            {
                "path": "spec/example.md",
                "semantic_role": "NORMATIVE_RULE",
                "disposition": "REMOVE_CANDIDATE",
            }
        ],
    )

    result = inspect_repository(root)

    assert result["items"][0]["state"] == "BLOCKED_BY_ACTIVE_REFERENCE"
    assert result["items"][0]["references"] == [
        {
            "source_path": "tool.py",
            "line": 1,
            "kind": "ACTIVE_MACHINE_DEPENDENCY",
        }
    ]


def test_human_reference_does_not_block_removal_candidate(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    target = root / "spec/example.md"
    target.parent.mkdir(parents=True)
    target.write_text("# Normative\n", encoding="utf-8")
    (root / "history.md").write_text(
        "Historical: spec/example.md\n",
        encoding="utf-8",
    )
    _write_policy(root)
    _write_workflow(
        root,
        targets=[
            {
                "path": "spec/example.md",
                "semantic_role": "NORMATIVE_RULE",
                "disposition": "REMOVE_CANDIDATE",
            }
        ],
    )

    result = inspect_repository(root)

    assert result["items"][0]["state"] == "REMOVE_CANDIDATE"
    assert result["items"][0]["references"][0]["kind"] == "HISTORICAL_OR_HUMAN_REFERENCE"


def test_unregistered_markdown_remains_unresolved(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    target = root / "spec/unregistered.md"
    target.parent.mkdir(parents=True)
    target.write_text("# Unknown role\n", encoding="utf-8")
    _write_policy(root)
    _write_workflow(
        root,
        targets=[
            {
                "path": "spec/registered.md",
                "semantic_role": "NORMATIVE_RULE",
                "disposition": "REMOVE_CANDIDATE",
            }
        ],
    )

    result = inspect_repository(root)

    states = {item["path"]: item["state"] for item in result["items"]}
    assert states["spec/unregistered.md"] == "UNRESOLVED"
    assert states["spec/registered.md"] == "ALREADY_ABSENT"


def test_cleanup_workflow_registry_is_excluded_from_dependency_hits(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    target = root / "spec/example.md"
    target.parent.mkdir(parents=True)
    target.write_text("# Normative\n", encoding="utf-8")
    _write_policy(root)
    _write_workflow(
        root,
        targets=[
            {
                "path": "spec/example.md",
                "semantic_role": "NORMATIVE_RULE",
                "disposition": "REMOVE_CANDIDATE",
            }
        ],
    )

    result = inspect_repository(root)

    assert result["items"][0]["state"] == "REMOVE_CANDIDATE"
    assert result["items"][0]["references"] == []


def test_apply_fails_closed_while_mpd_0012_is_draft(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    _write_policy(root, status="DRAFT")

    with pytest.raises(MarkdownCleanupError, match="not ACTIVE"):
        apply_cleanup_plan(
            root,
            {
                "schema_version": "ptsip-markdown-cleanup-plan/v1",
                "state": "READY",
                "apply_authorized": True,
                "repository_head": None,
                "target_hashes": {},
                "remove": [],
            },
        )


def test_scan_reference_hits_accepts_windows_path_form(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    (root / "spec").mkdir()
    (root / "spec/example.md").write_text("# Normative\n", encoding="utf-8")
    (root / "tool.ps1").write_text(
        '$p = "spec\\example.md"\n',
        encoding="utf-8",
    )

    hits = scan_reference_hits(root, "spec/example.md")

    assert [(hit.source_path, hit.line, hit.kind) for hit in hits] == [
        ("tool.ps1", 1, "ACTIVE_MACHINE_DEPENDENCY")
    ]
