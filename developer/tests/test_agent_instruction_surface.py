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
