from __future__ import annotations

from pathlib import Path

import pytest
import yaml

from developer.automation.repository_state_resolver import (
    RepositoryStateResolutionError,
    resolve_state,
)


ROOT = Path(__file__).resolve().parents[2]

EXPECTED = {
    "developer_policy": "developer/policy/index.yaml",
    "developer_planning": "developer/planning/index.yaml",
    "project_profile": "developer/profiles/ptsip-repository.yaml",
    "policy_plan_binding": "developer/bindings/policy-plan-bindings.yaml",
    "agent_contract": "src/agent_contracts/bindings/current.yaml",
    "governance_source": "developer/policy/registries/governance-source-registry.yaml",
    "context_migration": "developer/planning/migrations/MPD-0012-agent-context-machine-migration.yaml",
}


@pytest.mark.parametrize(("domain", "ref"), EXPECTED.items())
def test_repository_state_resolves_exact_machine_owner(domain: str, ref: str) -> None:
    result = resolve_state(domain, ROOT)

    assert result["ref"] == ref
    assert result["authority"] is False


def test_repository_state_index_excludes_default_markdown_context() -> None:
    payload = yaml.safe_load((ROOT / "developer/state/index.yaml").read_text(encoding="utf-8"))

    assert payload["authority"] is False
    assert payload["default_agent_context"] == {
        "prose_history_required": False,
        "status_markdown_required": False,
        "memory_markdown_required": False,
        "reference_markdown_required": False,
    }


def test_unknown_repository_state_domain_fails_closed() -> None:
    with pytest.raises(RepositoryStateResolutionError, match="unknown state domain"):
        resolve_state("unknown", ROOT)
