from __future__ import annotations

from pathlib import Path

import developer.automation.implementation_work_packet as work_packet
import developer.automation.policy_resolver as policy_resolver


ROOT = Path(__file__).resolve().parents[2]


def test_workflow_registry_is_machine_valid() -> None:
    payload = work_packet._registry(ROOT)
    assert payload["schema_version"] == "ptsip-implementation-workflows/v2"
    assert payload["projection_authority"] is False
    assert payload["failure_routing"]["recheck_context_at"] == 2
    assert payload["failure_routing"]["reresolve_scope_at"] == 3
    assert len(payload["tasks"]) >= 1


def test_packet_builds_exact_mutation_acceptance_and_regression_plan(monkeypatch) -> None:
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

    assert packet["schema_version"] == "ptsip-implementation-work-packet/v2"
    mutation = packet["mutation_plan"]
    assert mutation["scope_expansion"] == "RE_RESOLVE_REQUIRED"
    assert mutation["allowed_test_paths"] == [
        "tests/ptsip/control_plane/test_github_authority.py"
    ]

    edit_selectors = [item["selector"] for item in mutation["targets"]]
    assert {"kind": "PYTHON_FUNCTION", "name": "_workflow_status"} in edit_selectors
    assert {
        "kind": "PYTHON_METHOD",
        "class": "GithubControlPlaneClient",
        "method": "application",
    } in edit_selectors
    assert all(item["rationale"] for item in mutation["targets"])

    read_selectors = [item["selector"] for item in packet["read_context"]]
    assert {"kind": "PYTHON_FUNCTION", "name": "_global_decision_id"} in read_selectors
    assert {
        "kind": "PYTHON_METHOD",
        "class": "GithubControlPlaneClient",
        "method": "gate",
    } in read_selectors

    verification = packet["verification"]
    assert len(verification["required_new_tests"]) == 3
    assert len(verification["required_new_pytest_nodes"]) == 3
    assert set(verification["missing_required_new_tests"]).issubset(
        set(verification["required_new_pytest_nodes"])
    )
    assert verification["status"] in {"REQUIRES_NEW_TESTS", "READY"}
    assert "tests/ptsip/control_plane" in verification["core_regression"]["pytest_targets"]
    assert "tests/ptsip/test_proposed_component.py" in verification["core_regression"]["pytest_targets"]
    assert len(verification["combined_regression_pytest_targets"]) >= len(
        verification["core_regression"]["pytest_targets"]
    )
    assert verification["commands"]["regression"][0:3] == ["python", "-m", "pytest"]

    coverage = {item["id"]: item for item in packet["acceptance_coverage"]}
    assert set(coverage) == {
        "GITHUB_PROPOSAL_RESOLUTION",
        "GITHUB_PROPOSAL_REPEAT",
        "NORMAL_GITHUB_DECISION_UNCHANGED",
        "LOCAL_RECEIPT_DOES_NOT_CHANGE_WINNER",
    }
    assert coverage["GITHUB_PROPOSAL_RESOLUTION"]["test_nodes"] == [
        "tests/ptsip/control_plane/test_github_authority.py::test_github_proposal_resolution_returns_terminal_local_receipt"
    ]

    assert packet["test_mode"]["component_ref"] == "ptsip-core-verification"
    assert packet["test_mode"]["status"] == "NOT_REGISTERED"
    assert packet["test_mode"]["fallback"] == "CANONICAL_COMPONENT_INCLUDE_SELECTION"
    freshness = packet["freshness"]
    assert freshness["strategy"] == "FILE_AND_SELECTOR_RECHECK_BEFORE_EVERY_VERIFICATION"
    assert "src/ptsip/app/github_authority.py" not in freshness["context_files"]
    same_file_read_context = [
        item for item in freshness["read_context_fingerprints"]
        if item["path"] == "src/ptsip/app/github_authority.py"
    ]
    assert len(same_file_read_context) == 2
    assert all(len(item["fingerprint"]) == 64 for item in same_file_read_context)


def test_core_regression_is_derived_from_canonical_component_profile() -> None:
    targets = work_packet._component_regression_targets(
        ROOT,
        "ptsip-core-verification",
        "ptsip.yaml",
    )
    assert "tests/ptsip/control_plane" in targets
    assert "tests/ptsip/identity" in targets
    assert "tests/ptsip/test_proposed_component.py" in targets
    assert all(target.startswith("tests/") for target in targets)


def test_mutation_selector_integrity_resolves_current_targets() -> None:
    targets = [
        {
            "path": "src/ptsip/app/github_authority.py",
            "selector": {"kind": "PYTHON_FUNCTION", "name": "_workflow_status"},
        },
        {
            "path": "src/ptsip/app/github_authority.py",
            "selector": {
                "kind": "PYTHON_METHOD",
                "class": "GithubControlPlaneClient",
                "method": "application",
            },
        },
    ]
    assert work_packet._selector_integrity(ROOT, targets) == []


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
        "mutation_plan": {"targets": []},
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
    assert result["reprepare_required"] is False
    assert result["safe_to_continue_iteration"] is False
    assert len(result["iteration_fingerprint"]) == 64


def test_repeated_failure_routing_escalates_without_expanding_scope(tmp_path: Path) -> None:
    state = tmp_path / "failure-state.json"
    policy = {"recheck_context_at": 2, "reresolve_scope_at": 3}
    kwargs = {
        "packet_id": "iwp-example",
        "stage": "focused",
        "returncode": 1,
        "output": "FAILED test_example.py::test_case - AssertionError: expected terminal state\n",
        "state_path": state,
        "policy": policy,
    }

    first = work_packet.route_failure(**kwargs)
    second = work_packet.route_failure(**kwargs)
    third = work_packet.route_failure(**kwargs)

    assert first["repeat_count"] == 1
    assert first["next_action"] == "FIX_WITHIN_CURRENT_MUTATION_PLAN"
    assert second["repeat_count"] == 2
    assert second["next_action"] == "RECHECK_PACKET_ACCEPTANCE_AND_CONTEXT"
    assert third["repeat_count"] == 3
    assert third["next_action"] == "RE_RESOLVE_MUTATION_SCOPE"
    assert third["scope_expansion_allowed"] is False


def test_collection_failure_routes_to_test_contract_repair(tmp_path: Path) -> None:
    routed = work_packet.route_failure(
        packet_id="iwp-example",
        stage="focused",
        returncode=4,
        output="ERROR collecting tests/ptsip/control_plane/test_github_authority.py\n",
        state_path=tmp_path / "failure-state.json",
        policy={"recheck_context_at": 2, "reresolve_scope_at": 3},
    )
    assert routed["classification"] == "TEST_CONTRACT_FAILURE"
    assert routed["next_action"] == "REPAIR_TEST_SELECTION_OR_REQUIRED_TEST"
