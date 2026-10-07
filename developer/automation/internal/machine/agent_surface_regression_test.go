package machine

import (
	"reflect"
	"strings"
	"testing"
)

func requireAgentFields(t *testing.T, value Object, fields Object) {
	t.Helper()
	for key, expected := range fields {
		if !reflect.DeepEqual(value[key], expected) {
			t.Fatalf("%s: got %#v, expected %#v", key, value[key], expected)
		}
	}
}

func TestAgentManagementSurfaceIsRootAgentAndNotToolingRoot(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	surface := Map(agentTrialValue(t, r, "repository_management_surface"))
	requireAgentFields(t, surface, Object{"canonical_root": ".agent/", "developer_tool_implementation_root": "developer/automation/", "developer_tool_root_is_management_surface": false})
	resolver, err := NewResolver(r)
	if err != nil {
		t.Fatal(err)
	}
	if resolver.Bindings[".agent"] == nil || resolver.Bindings["developer/agent_instructions"] != nil {
		t.Fatal("agent namespace routing drift")
	}
	files, err := BuildAgentMaterialization(agentRegressionFixture(t, "Read the file.\n"), "", "")
	if err != nil || files[".agent/index.yaml"] == nil {
		t.Fatal("default management surface is not .agent", err)
	}
}

func TestAgentBootstrapSurfaceContractPreservesCleanLevel1Entry(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := Map(agentTrialValue(t, r, "level_1_bootstrap"))
	requireAgentFields(t, bootstrap, Object{"source": "AGENTS.md", "existing_agent_surface_required": false, "legacy_materialization_required": false, "source_mutation_during_bootstrap": "PASS_ATOMS_REMOVED_FROM_AGENTS", "output_index": ".agent/index.yaml", "output_pass_stage": ".agent/stages/level1.json", "output_unresolved_stage": ".agent/unresolved/level1.json", "registry_required": false, "provenance_markdown_required": false})
	compact := Map(bootstrap["compact_agents_contract"])
	requireAgentFields(t, compact, Object{"machine_entry_count": 1, "machine_entry_schema": "PTSIP_AGENT_ENTRY_V1", "pass_atom_natural_language_in_agents": "FORBIDDEN", "pass_atom_route_lines_in_agents": "FORBIDDEN", "unresolved_natural_language_in_agents": "PRESERVED", "stage_shape": "PASS_BY_ATOM", "semantic_matching_after_entry": "FORBIDDEN", "integration_state_in_entry_required": true})
}

func TestAgentIntegrationSurfaceKeepsLocalCLIWithoutImplicitMCP(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for suffix, expected := range map[string]any{"baseline_mode_without_mcp": "LOCAL_CLI_ONLY", "mcp_optional": true, "mcp_transport_when_available": "STDIO", "remote_ptsip_mcp_service_required": false, "daemon_required": false, "network_required": false, "mcp_installation/current_mcp_state": "ABSENT", "mcp_installation/current_implementation_state": "NOT_AVAILABLE", "mcp_installation/user_approval_required": true, "mcp_installation/offer_to_user_now": false} {
		if got := agentTrialValue(t, r, "agent_integration/"+suffix); !reflect.DeepEqual(got, expected) {
			t.Fatal(suffix, got, expected)
		}
	}
}

func TestAgentMCPReadinessRequiresAllThreeLevelsAndExplicitPolicyUpdate(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	gate := Map(agentTrialValue(t, r, "agent_integration/mcp_installation/readiness_gate"))
	requireAgentFields(t, gate, Object{"current_state": "BLOCKED_BY_PROGRESSIVE_REASONING_COMPLETION", "implementation_work_before_gate_completion": "ALLOWED", "runtime_mcp_exposure_before_gate_completion": "FORBIDDEN", "ready_state_before_gate_completion": "FORBIDDEN", "required_completed_levels": []any{"LEVEL_1", "LEVEL_2", "LEVEL_3"}, "current_completed_levels": []any{"LEVEL_1"}, "level_1_only_is_sufficient": false, "transition_to_ready_is_automatic": false, "transition_to_ready_requires_explicit_policy_update": true, "state_until_all_requirements_satisfied": "NOT_AVAILABLE", "offer_to_user_until_all_requirements_satisfied": false})
}

func TestAgentProgressionIsPerAtomAndUnresolvedDoesNotBlockPassedAtoms(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	stage := Map(agentTrialValue(t, r, "progressive_reasoning_pipeline/stage_contract"))
	requireAgentFields(t, stage, Object{"next_level_input": "CURRENT_LEVEL_PASS_SUBSET_ONLY", "unresolved_is_direct_next_level_candidate": false, "unresolved_may_transition_to_pass_when_current_level_becomes_exactly_classifiable": true, "pass_transition_immediately_creates_next_level_candidate": true, "unresolved_blocks_other_passed_atoms": false, "current_level_unresolved_zero_required_for_next_level": false, "next_level_candidate_set_is_dynamic": true})
	runtime := Map(agentTrialValue(t, r, "progressive_reasoning_pipeline/runtime_entry"))
	requireAgentFields(t, runtime, Object{"progression_unit": "ATOM"})
	reassessment := Map(agentTrialValue(t, r, "progressive_reasoning_pipeline/unresolved_reassessment"))
	requireAgentFields(t, reassessment, Object{"positive_result_transition": "UNRESOLVED_TO_PASS", "negative_result_behavior": "KEEP_UNRESOLVED", "forced_assignment_to_reduce_unresolved_count": "FORBIDDEN"})
	evolution := Map(agentTrialValue(t, r, "progressive_reasoning_pipeline/taxonomy_evolution"))
	requireAgentFields(t, evolution, Object{"unresolved_may_persist_without_forced_classification": true, "new_level_1_category_may_be_added_when_distinct_semantics_are_evidenced": true, "newly_passed_unresolved_atom_becomes_next_level_candidate": true})
	if strings.Contains(AgentBootstrapText(), "PTSIP_AGENT_ROUTE") {
		t.Fatal("per-atom routes leaked into bootstrap")
	}
}
