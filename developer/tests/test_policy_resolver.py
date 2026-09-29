from __future__ import annotations

from pathlib import Path

import pytest
import yaml

import developer.automation.policy_resolver as policy_resolver_module

from developer.automation.policy_resolver import (
    PolicyResolverError,
    explain_policy,
    get_normative_rule,
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


def test_github_authority_scope_resolves_policy_without_plan_task_context() -> None:
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
    assert "task_context" not in result

def test_normative_rule_projection_returns_machine_registry_record() -> None:
    result = get_normative_rule(ROOT, rule_id="PTSIP-AUT-007")
    assert result["schema_version"] == "ptsip-normative-rule-projection/v2"
    assert result["canonical_source"] == "registry/ptsip-registry.yaml"
    assert result["registry_record"]["id"] == "PTSIP-AUT-007"
    assert result["projection_authority"] is False
    assert "section_text" not in result
    assert "line_start" not in result


def test_rule_cli_projects_one_normative_section(capsys) -> None:
    result = policy_resolver_module.main(
        [
            "--repository",
            str(ROOT),
            "rule",
            "PTSIP-AUT-007",
            "--json",
        ]
    )
    captured = capsys.readouterr()
    assert result == 0
    assert '"rule_id": "PTSIP-AUT-007"' in captured.out
    assert "## 10. Action-time synchronization" not in captured.out


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


def test_pp_transition_policy_routes_for_public_profile_modify() -> None:
    result = resolve_policies(
        ROOT,
        scope="profiles/example.ptsip.yaml",
        operation="MODIFY",
    )

    assert result["binding_scope"] == "profiles"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]
    assert result["policies"][1]["sections"] == [
        "authority_semantics",
        "commit_candidate_trigger",
        "t2_authority_delta",
        "transition_generation",
        "user_revision_lineage",
        "reconciliation_safety",
        "verification_layers",
    ]


def test_pp_transition_policy_routes_for_current_schema_modify() -> None:
    result = resolve_policies(
        ROOT,
        scope="schemas/ptsip-profile-pp-1.01.schema.json",
        operation="MODIFY",
    )

    assert result["binding_scope"] == "schemas"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]
    assert "t2_authority_delta" in result["policies"][1]["sections"]


def test_pp_transition_policy_routes_for_future_canonical_registry_path() -> None:
    result = resolve_policies(
        ROOT,
        scope="registry/project-profile-contracts.yaml",
        operation="MODIFY",
    )

    assert result["binding_scope"] == "registry/project-profile-contracts.yaml"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]


def test_unrelated_modify_does_not_route_pp_transition_policy() -> None:
    result = resolve_policies(
        ROOT,
        scope="README.md",
        operation="MODIFY",
    )

    assert result["binding_scope"] == "."
    assert [item["policy_id"] for item in result["policies"]] == ["MPD-0010"]


@pytest.mark.parametrize(
    "scope",
    [
        "developer/automation/project_profile_registry.py",
        "developer/automation/pp/pp_transition_delta.py",
        "developer/automation/pp/pp_transition_reconciler.py",
    ],
)
def test_pp_transition_automation_modules_route_mpd_0011(scope: str) -> None:
    result = resolve_policies(
        ROOT,
        scope=scope,
        operation="MODIFY",
    )

    assert result["binding_scope"] == scope
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0001",
        "MPD-0010",
        "MPD-0011",
    ]


@pytest.mark.parametrize(
    "scope",
    [
        "src/ptsip/project_profile_contracts.py",
        "src/ptsip/profile_identity.py",
        "src/ptsip/profile_compatibility.py",
        "src/ptsip/specdata/project-profile-contracts.yaml",
    ],
)
def test_runtime_pp_registry_surfaces_route_transition_policy(scope: str) -> None:
    result = resolve_policies(
        ROOT,
        scope=scope,
        operation="MODIFY",
    )

    assert result["binding_scope"] == scope
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]


@pytest.mark.parametrize(
    "scope",
    [
        "setup.py",
        "pyproject.toml",
        "MANIFEST.in",
        ".github/scripts/verify_distribution_contracts.py",
    ],
)
def test_distribution_projection_surfaces_route_pp_transition_policy(scope: str) -> None:
    result = resolve_policies(
        ROOT,
        scope=scope,
        operation="MODIFY",
    )

    assert result["binding_scope"] == scope
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]


def test_tooling_test_workflow_routes_release_and_pp_transition_policy() -> None:
    result = resolve_policies(
        ROOT,
        scope=".github/workflows/tooling-test.yml",
        operation="MODIFY",
    )

    assert result["binding_scope"] == ".github/workflows/tooling-test.yml"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0007",
        "MPD-0010",
        "MPD-0011",
    ]


@pytest.mark.parametrize(
    "scope",
    [
        "developer/automation/dev_setup.py",
        "developer/automation/pp/pp_pre_commit.py",
        ".githooks/pre-commit",
        "setup_dev.bat",
        "bootstrap_repo.ps1",
    ],
)
def test_h3_hook_surfaces_route_pp_transition_policy(scope: str) -> None:
    result = resolve_policies(
        ROOT,
        scope=scope,
        operation="MODIFY",
    )

    assert result["binding_scope"] == scope
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0001",
        "MPD-0010",
        "MPD-0011",
    ]


def test_remote_pp_verifier_routes_transition_policy() -> None:
    result = resolve_policies(
        ROOT,
        scope="developer/automation/pp/pp_remote_verify.py",
        operation="VERIFY",
    )

    assert result["binding_scope"] == "developer/automation/pp/pp_remote_verify.py"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0001",
        "MPD-0010",
        "MPD-0011",
    ]


def test_release_pp_verifier_routes_release_transition_policy() -> None:
    result = resolve_policies(
        ROOT,
        scope="developer/automation/pp/pp_release_verify.py",
        operation="RELEASE",
    )

    assert result["binding_scope"] == "developer/automation/pp/pp_release_verify.py"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0001",
        "MPD-0010",
        "MPD-0011",
    ]


@pytest.mark.parametrize(
    "scope",
    [
        ".github/workflows/release.yml",
        ".github/workflows/tooling-release.yml",
        ".github/scripts/verify_release_contract.py",
    ],
)
def test_release_surfaces_route_pp_release_verification_policy(scope: str) -> None:
    result = resolve_policies(
        ROOT,
        scope=scope,
        operation="RELEASE",
    )

    assert result["binding_scope"] == scope
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0007",
        "MPD-0010",
        "MPD-0011",
    ]


@pytest.mark.parametrize(
    "scope",
    [
        "src/ptsip/local_profile_catalog.py",
        "src/ptsip/profile_metadata.py",
    ],
)
def test_local_profile_runtime_surfaces_route_pp_policy(scope: str) -> None:
    result = resolve_policies(
        ROOT,
        scope=scope,
        operation="MODIFY",
    )

    assert result["binding_scope"] == scope
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]


def test_pp_102_transition_seed_routes_developer_and_pp_policy() -> None:
    result = resolve_policies(
        ROOT,
        scope="developer/automation/seed_pp_102_transition.py",
        operation="MODIFY",
    )

    assert result["binding_scope"] == "developer/automation/seed_pp_102_transition.py"
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0001",
        "MPD-0010",
        "MPD-0011",
    ]


def test_public_profile_catalog_schema_routes_pp_transition_policy() -> None:
    result = resolve_policies(
        ROOT,
        scope="developer/policy/schemas/public-profile-catalog.schema.json",
        operation="MODIFY",
    )

    assert result["binding_scope"] == (
        "developer/policy/schemas/public-profile-catalog.schema.json"
    )
    assert [item["policy_id"] for item in result["policies"]] == [
        "MPD-0010",
        "MPD-0011",
    ]
