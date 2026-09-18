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


def test_progressive_policy_allows_clean_level1_bootstrap() -> None:
    policy = yaml.safe_load(
        (ROOT / "developer" / "policy" / "MPD-0010.yaml").read_text(encoding="utf-8")
    )
    bootstrap = policy["rules"]["agent_instruction_entry_taxonomy_trial"][
        "level_1_bootstrap"
    ]

    assert bootstrap["source"] == "AGENTS.md"
    assert bootstrap["existing_agent_surface_required"] is False
    assert bootstrap["legacy_materialization_required"] is False
    assert bootstrap["source_mutation_during_bootstrap"] == "PASS_ATOMS_REMOVED_FROM_AGENTS"
    assert bootstrap["output_index"] == ".agent/index.yaml"
    assert bootstrap["output_pass_stage"] == ".agent/stages/level1.json"
    assert bootstrap["output_unresolved_stage"] == ".agent/unresolved/level1.json"
    assert bootstrap["registry_required"] is False
    assert bootstrap["provenance_markdown_required"] is False


    compact = bootstrap["compact_agents_contract"]
    assert compact["machine_entry_count"] == 1
    assert compact["machine_entry_schema"] == "PTSIP_AGENT_ENTRY_V1"
    assert compact["pass_atom_natural_language_in_agents"] == "FORBIDDEN"
    assert compact["pass_atom_route_lines_in_agents"] == "FORBIDDEN"
    assert compact["unresolved_natural_language_in_agents"] == "PRESERVED"
    assert compact["stage_shape"] == "PASS_BY_ATOM"
    assert compact["semantic_matching_after_entry"] == "FORBIDDEN"
    assert compact["integration_state_in_entry_required"] is True


def test_agent_integration_policy_uses_local_cli_baseline() -> None:
    policy = yaml.safe_load(
        (ROOT / "developer" / "policy" / "MPD-0010.yaml").read_text(encoding="utf-8")
    )
    integration = policy["rules"]["agent_instruction_entry_taxonomy_trial"][
        "agent_integration"
    ]

    assert integration["baseline_mode_without_mcp"] == "LOCAL_CLI_ONLY"
    assert integration["mcp_optional"] is True
    assert integration["mcp_transport_when_available"] == "STDIO"
    assert integration["remote_ptsip_mcp_service_required"] is False
    assert integration["daemon_required"] is False
    assert integration["network_required"] is False
    assert integration["mcp_installation"]["current_mcp_state"] == "ABSENT"
    assert integration["mcp_installation"]["current_implementation_state"] == "NOT_AVAILABLE"
    assert integration["mcp_installation"]["user_approval_required"] is True
    assert integration["mcp_installation"]["offer_to_user_now"] is False


def test_mcp_ready_is_blocked_until_levels_1_through_3_complete() -> None:
    policy = yaml.safe_load(
        (ROOT / "developer" / "policy" / "MPD-0010.yaml").read_text(encoding="utf-8")
    )
    installation = policy["rules"]["agent_instruction_entry_taxonomy_trial"][
        "agent_integration"
    ]["mcp_installation"]
    gate = installation["readiness_gate"]

    assert installation["current_implementation_state"] == "NOT_AVAILABLE"
    assert installation["offer_to_user_now"] is False
    assert gate["current_state"] == "BLOCKED_BY_PROGRESSIVE_REASONING_COMPLETION"
    assert gate["implementation_work_before_gate_completion"] == "ALLOWED"
    assert gate["runtime_mcp_exposure_before_gate_completion"] == "FORBIDDEN"
    assert gate["ready_state_before_gate_completion"] == "FORBIDDEN"
    assert gate["required_completed_levels"] == [
        "LEVEL_1",
        "LEVEL_2",
        "LEVEL_3",
    ]
    assert gate["current_completed_levels"] == ["LEVEL_1"]
    assert gate["level_1_only_is_sufficient"] is False
    assert gate["transition_to_ready_is_automatic"] is False
    assert gate["transition_to_ready_requires_explicit_policy_update"] is True
    assert gate["state_until_all_requirements_satisfied"] == "NOT_AVAILABLE"
    assert gate["offer_to_user_until_all_requirements_satisfied"] is False


def test_progressive_policy_uses_per_atom_advancement_without_global_unresolved_gate() -> None:
    policy = yaml.safe_load(
        (ROOT / "developer" / "policy" / "MPD-0010.yaml").read_text(encoding="utf-8")
    )
    trial = policy["rules"]["agent_instruction_entry_taxonomy_trial"]
    stage = trial["progressive_reasoning_pipeline"]["stage_contract"]
    runtime = trial["progressive_reasoning_pipeline"]["runtime_entry"]
    evolution = trial["progressive_reasoning_pipeline"]["taxonomy_evolution"]

    assert stage["next_level_input"] == "CURRENT_LEVEL_PASS_SUBSET_ONLY"
    reassessment = trial["progressive_reasoning_pipeline"]["unresolved_reassessment"]

    assert stage["unresolved_is_direct_next_level_candidate"] is False
    assert stage["unresolved_may_transition_to_pass_when_current_level_becomes_exactly_classifiable"] is True
    assert stage["pass_transition_immediately_creates_next_level_candidate"] is True
    assert stage["unresolved_blocks_other_passed_atoms"] is False
    assert stage["current_level_unresolved_zero_required_for_next_level"] is False
    assert stage["next_level_candidate_set_is_dynamic"] is True
    assert runtime["progression_unit"] == "ATOM"
    assert reassessment["positive_result_transition"] == "UNRESOLVED_TO_PASS"
    assert reassessment["negative_result_behavior"] == "KEEP_UNRESOLVED"
    assert reassessment["forced_assignment_to_reduce_unresolved_count"] == "FORBIDDEN"
    assert evolution["unresolved_may_persist_without_forced_classification"] is True
    assert evolution["new_level_1_category_may_be_added_when_distinct_semantics_are_evidenced"] is True
    assert evolution["newly_passed_unresolved_atom_becomes_next_level_candidate"] is True
