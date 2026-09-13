from __future__ import annotations

from pathlib import Path

import developer.automation.implementation_work_packet as work_packet
import developer.automation.policy_resolver as policy_resolver


ROOT = Path(__file__).resolve().parents[2]


def test_workflow_registry_is_machine_valid() -> None:
    payload = work_packet._registry(ROOT)
    assert payload["projection_authority"] is False
    assert len(payload["tasks"]) >= 1


def test_packet_separates_edit_targets_and_read_context(monkeypatch) -> None:
    monkeypatch.setattr(
        policy_resolver,
        "_current_branch",
        lambda _root: "dev/0.3.8a1",
    )
    packet = work_packet.build_packet(
        ROOT,
        scope="src/ptsip/app/github_authority.py",
        operation="MODIFY",
    )

    edit_selectors = [item["selector"] for item in packet["edit_targets"]]
    assert {"kind": "PYTHON_FUNCTION", "name": "_workflow_status"} in edit_selectors
    assert {
        "kind": "PYTHON_METHOD",
        "class": "GithubControlPlaneClient",
        "method": "application",
    } in edit_selectors

    read_selectors = [item["selector"] for item in packet["read_context"]]
    assert {"kind": "PYTHON_FUNCTION", "name": "_global_decision_id"} in read_selectors
    assert {
        "kind": "PYTHON_METHOD",
        "class": "GithubControlPlaneClient",
        "method": "gate",
    } in read_selectors

    verification = packet["verification"]
    assert verification["status"] == "REQUIRES_NEW_TESTS"
    assert len(verification["missing_required_new_tests"]) == 3
    assert packet["test_mode"]["component_ref"] == "ptsip-core-verification"
    assert packet["test_mode"]["status"] == "NOT_REGISTERED"


def test_hunk_scope_guard_is_exact() -> None:
    assert work_packet._HUNK.match("@@ -105,3 +105,4 @@")
    ranges = [(100, 120), (200, 230)]
    assert work_packet._hunk_allowed(105, 3, ranges)
    assert work_packet._hunk_allowed(120, 0, ranges)
    assert not work_packet._hunk_allowed(150, 2, ranges)
    assert not work_packet._hunk_allowed(99, 3, ranges)


def test_check_blocks_unlisted_changed_path(monkeypatch) -> None:
    packet = {
        "task": {"branch": "dev/0.3.8a1"},
        "freshness": {"baseline_head": "abc", "context_files": [], "file_hashes": {}},
        "edit_budget": {
            "allowed_code_paths": ["src/ptsip/app/github_authority.py"],
            "allowed_test_paths": ["tests/ptsip/control_plane/test_github_authority.py"],
        },
        "edit_targets": [],
        "verification": {"required_new_pytest_nodes": []},
    }

    def fake_git(_root, *args):
        if args == ("branch", "--show-current"):
            return "dev/0.3.8a1"
        if args == ("rev-parse", "HEAD"):
            return "abc"
        raise AssertionError(args)

    monkeypatch.setattr(work_packet, "_git", fake_git)
    monkeypatch.setattr(work_packet, "_changed_paths", lambda _root: ["README.md"])
    result = work_packet.check_packet(ROOT, packet)
    assert result["status"] == "BLOCKED"
    assert result["problems"] == ["UNEXPECTED_CHANGED_PATH"]
    assert result["unexpected_changed_paths"] == ["README.md"]
