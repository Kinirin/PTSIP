from __future__ import annotations

from pathlib import Path

import pytest
import yaml

import developer.automation.policy_resolver as policy_resolver_module

from developer.automation.policy_resolver import (
    PolicyResolverError,
    explain_policy,
    get_policy,
    resolve_policies,
    validate_policy_resolver,
)


ROOT = Path(__file__).resolve().parents[2]


def test_policy_resolver_binding_plane_is_machine_valid() -> None:
    assert validate_policy_resolver(ROOT) == ()


def test_policy_resolver_uses_exact_ancestor_scope_binding() -> None:
    result = resolve_policies(
        ROOT,
        scope="src/ptsip/migration/future_engine.py",
        operation="MODIFY",
    )
    assert result["binding_scope"] == "src/ptsip/migration"
    assert result["projection_authority"] is False
    assert result["policies"] == [
        {
            "policy_id": "MPD-0008",
            "path": "developer/policy/MPD-0008.yaml",
            "status": "ACTIVE",
            "sections": ["authority_semantics"],
        },
        {
            "policy_id": "MPD-0010",
            "path": "developer/policy/MPD-0010.yaml",
            "status": "ACTIVE",
            "sections": [
                "identity_and_resolution",
                "canonical_and_runtime_projection",
            ],
        },
    ]


def test_github_authority_scope_returns_exact_implementation_context() -> None:
    result = resolve_policies(
        ROOT,
        scope="src/ptsip/app/github_authority.py",
        operation="MODIFY",
    )
    assert result["binding_scope"] == "src/ptsip/app/github_authority.py"
    assert result["policies"] == [
        {
            "policy_id": "MPD-0010",
            "path": "developer/policy/MPD-0010.yaml",
            "status": "ACTIVE",
            "sections": [
                "identity_and_resolution",
                "canonical_and_runtime_projection",
            ],
        }
    ]

    context = result["task_context"]
    assert context["branch"] == "dev/0.3.8a1"
    assert context["planning_entry"] == (
        "docs/planning/0.3.8a1/emergency-implementation-overlay.yaml"
    )
    assert context["normative_rule_refs"] == [
        "PTSIP-AUT-001",
        "PTSIP-AUT-002",
        "PTSIP-AUT-003",
        "PTSIP-AUT-004",
        "PTSIP-AUT-005",
        "PTSIP-AUT-006",
        "PTSIP-AUT-007",
    ]
    assert "src/ptsip/app/github_authority.py#_workflow_status" in context[
        "implementation_refs"
    ]
    assert "tests/ptsip/test_proposed_component.py" in context["test_refs"]


def test_similar_github_authority_scope_does_not_receive_task_context() -> None:
    result = resolve_policies(
        ROOT,
        scope="src/ptsip/app/github_authority_extra.py",
        operation="MODIFY",
    )
    assert result["binding_scope"] == "."
    assert "task_context" not in result


def test_similar_scope_name_does_not_match_registered_scope() -> None:
    result = resolve_policies(
        ROOT,
        scope="src/ptsip-migrations/future_engine.py",
        operation="MODIFY",
    )
    assert result["binding_scope"] == "."
    assert [item["policy_id"] for item in result["policies"]] == ["MPD-0010"]


def test_operation_override_is_exact() -> None:
    result = resolve_policies(
        ROOT,
        scope="README.md",
        operation="RELEASE",
    )
    assert result["binding_scope"] == "."
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0006",
        "MPD-0007",
        "MPD-0010",
    ]


def test_get_policy_can_return_only_one_rule_section() -> None:
    result = get_policy(
        ROOT,
        policy_id="MPD-0010",
        section="identity_and_resolution",
    )
    assert result["fragment"] == "rules.identity_and_resolution"
    assert result["canonical_path"] == "developer/policy/MPD-0010.yaml"
    assert isinstance(result["record"], dict)
    assert "registry_resolution_budget" in result["record"]
    assert "ptsip_design_priority" not in result["record"]


def test_explain_is_compact_metadata_not_policy_body() -> None:
    result = explain_policy(ROOT, policy_id="MPD-0010")
    assert result["policy_id"] == "MPD-0010"
    assert result["status"] == "ACTIVE"
    assert "identity_and_resolution" in result["rule_sections"]
    assert "record" not in result


@pytest.mark.parametrize("operation", ["SEARCH", "GUESS", ""])
def test_unknown_operation_fails_closed(operation: str) -> None:
    with pytest.raises(PolicyResolverError):
        resolve_policies(
            ROOT,
            scope="src/ptsip/migration/future_engine.py",
            operation=operation,
        )


def test_scope_escape_fails_closed() -> None:
    with pytest.raises(PolicyResolverError):
        resolve_policies(
            ROOT,
            scope="../outside-repository",
            operation="READ",
        )


def test_unknown_policy_identity_fails_closed() -> None:
    with pytest.raises(PolicyResolverError):
        get_policy(ROOT, policy_id="MPD-9999")


def test_cli_fails_closed_cleanly_on_malformed_binding_yaml(
    monkeypatch,
    capsys,
) -> None:
    original = policy_resolver_module.load_yaml

    def malformed(path, *, root=None):
        if str(path).endswith("policy-resolver-bindings.yaml"):
            raise yaml.YAMLError("synthetic malformed binding")
        return original(path, root=root)

    monkeypatch.setattr(policy_resolver_module, "load_yaml", malformed)
    result = policy_resolver_module.main(
        [
            "--repository",
            str(ROOT),
            "resolve",
            "--scope",
            "src/ptsip/migration/engine.py",
            "--operation",
            "MODIFY",
            "--json",
        ]
    )
    captured = capsys.readouterr()
    assert result == 2
    assert "Policy Resolver error: synthetic malformed binding" in captured.out
    assert "Traceback" not in captured.out
