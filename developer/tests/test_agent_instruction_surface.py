from __future__ import annotations

from pathlib import Path

import yaml

from developer.automation.agent_instruction_materializer import DEFAULT_OUTPUT_ROOT


ROOT = Path(__file__).resolve().parents[2]


def test_agent_instruction_management_surface_is_repository_root_agent() -> None:
    assert DEFAULT_OUTPUT_ROOT == Path(".agent")


def test_agent_instruction_policy_separates_tooling_from_management_surface() -> None:
    policy = yaml.safe_load(
        (ROOT / "developer" / "policy" / "MPD-0010.yaml").read_text(encoding="utf-8")
    )
    surface = policy["rules"]["agent_instruction_entry_taxonomy_trial"][
        "repository_management_surface"
    ]
    assert surface["canonical_root"] == ".agent/"
    assert surface["developer_tool_implementation_root"] == "developer/automation/"
    assert surface["developer_tool_root_is_management_surface"] is False


def test_policy_binding_uses_root_agent_surface_only() -> None:
    bindings = yaml.safe_load(
        (
            ROOT
            / "developer"
            / "policy"
            / "policy-resolver-bindings.yaml"
        ).read_text(encoding="utf-8")
    )["scope_bindings"]
    assert ".agent" in bindings
    assert "developer/agent_instructions" not in bindings
