package machine

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func agentRegressionFixture(t *testing.T, text string) *Repository {
	t.Helper()
	r := agentFixture(t)
	if err := r.AtomicWrite("AGENTS.md", []byte(text), nil); err != nil {
		t.Fatal(err)
	}
	return r
}

func agentTrialValue(t *testing.T, r *Repository, suffix string) any {
	t.Helper()
	graph, err := r.Read("developer/policy/registries/root-family-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	pointer := "/rules/agent_instruction_entry_taxonomy_trial/" + suffix
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		if source["source_policy_id"] != "MPD-0010" {
			continue
		}
		for _, raw := range List(source["units"]) {
			unit := Map(raw)
			if unit["source_pointer"] == pointer {
				value, err := agentSection(r, Text(unit["policy_id"]), Text(unit["section"]))
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
		}
	}
	t.Fatal("unregistered agent responsibility", pointer)
	return nil
}

func TestAgentClassifierPreservesOriginalPositiveAndUnresolvedCases(t *testing.T) {
	for _, test := range []struct {
		name, text    string
		labels        []string
		parent, child string
		unresolved    bool
	}{
		{"multilabel", "Before publishing, run pytest and report the status.\n", []string{"APPLICABILITY", "ACTION", "EVIDENCE"}, "ACTION", "COMMAND", false},
		{"goal_word_is_not_goal", "Do not reconstruct goals from filenames.\n", []string{"RULE"}, "OTHER", "", false},
		{"explicit_purpose", "The purpose of this component is to route instructions cheaply.\n", []string{"OTHER"}, "OTHER", "GOAL", false},
		{"other_is_not_fallback", "frobnicator quux zed\n", []string{}, "", "", true},
		{"descriptive_context", "Architecture baseline: MVC plus EDA\n", []string{"OTHER"}, "OTHER", "CONTEXT", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			atoms := ClassifyAgentMarkdown(test.text)
			if len(atoms) != 1 {
				t.Fatal(atoms)
			}
			atom := atoms[0]
			if !reflect.DeepEqual(atom.Level1, test.labels) || atom.Unresolved != test.unresolved {
				t.Fatal(atom)
			}
			if test.child != "" && !Has(atom.Level2[test.parent], test.child) {
				t.Fatal(atom.Level2)
			}
			if test.name == "goal_word_is_not_goal" && len(atom.Level2["OTHER"]) != 0 {
				t.Fatal("goal inferred from an isolated word")
			}
			if test.name == "multilabel" && (!Has(atom.Level2["EVIDENCE"], "TEST") || !Has(atom.Level2["EVIDENCE"], "STATUS")) {
				t.Fatal(atom)
			}
			if test.name == "explicit_purpose" && !reflect.DeepEqual(atom.Level2["OTHER"], []string{"CONTEXT", "GOAL"}) {
				t.Fatal(atom)
			}
		})
	}
}

func TestAgentClassifierInheritsParentApplicabilityForListActions(t *testing.T) {
	atoms := ClassifyAgentMarkdown("For SDK work:\n\n1. Read `docs/policy/sdk_governance.yaml`.\n2. Run `ptsip gate . --json`.\n")
	found := 0
	for _, atom := range atoms {
		if strings.HasPrefix(atom.Text, "Read ") || strings.HasPrefix(atom.Text, "Run ") {
			found++
			if !Has(atom.InheritedLevel1, "APPLICABILITY") || !reflect.DeepEqual(atom.Level1, []string{"APPLICABILITY", "ACTION"}) {
				t.Fatal(atom)
			}
			expected := "READ"
			if strings.HasPrefix(atom.Text, "Run ") {
				expected = "COMMAND"
			}
			if !Has(atom.Level2["ACTION"], expected) {
				t.Fatal(atom)
			}
		}
	}
	if found != 2 {
		t.Fatal(atoms)
	}
}

func TestAgentTrialVocabularyAndRepositoryBindingsRemainExact(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAgentTaxonomy(r); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(AgentLevel1, []string{"APPLICABILITY", "RULE", "ACTION", "EVIDENCE", "OTHER"}) || agentTrialValue(t, r, "unresolved_is_namespace") != false {
		t.Fatal("taxonomy drift")
	}
	resolver, err := NewResolver(r)
	if err != nil {
		t.Fatal(err)
	}
	row := resolver.Bindings["developer/automation/agent_instruction_classifier.py"]
	if row == nil {
		t.Fatal("classifier binding missing")
	}
	result, err := resolver.Resolve("developer/automation/agent_instruction_classifier.py", "READ")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, raw := range List(result["policies"]) {
		row := Map(raw)
		if row["policy_id"] == "MPD-INFO-0001" && Has(Strings(row["sections"]), "unit_mpd_0010_f1b93fa1851f") {
			found = true
		}
		if strings.Contains(Text(row["path"]), "/legacy/") {
			t.Fatal("historical runtime owner admitted")
		}
	}
	if !found {
		t.Fatal(result)
	}
	for suffix, expected := range map[string]any{"decision": "APPROVED", "stage": "REPOSITORY_TRIAL", "distribution": "FORBIDDEN", "product_integration": "NOT_AUTHORIZED"} {
		if agentTrialValue(t, r, suffix) != expected {
			t.Fatal(suffix)
		}
	}
}

func TestAgentMaterializationStoresEachAtomOnceAndProjectsIDsOnly(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\nFor SDK work:\n\n- Read `docs/sdk.md`.\n- Read source files; do not edit generated files.\n\nArchitecture baseline: MVC plus EDA\n")
	files, err := BuildAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil {
		t.Fatal(err)
	}
	registry, index := files[".agent/registry.yaml"], files[".agent/index.yaml"]
	seen := map[string]bool{}
	for _, raw := range List(registry["atoms"]) {
		id := Text(Map(raw)["atom_id"])
		if seen[id] {
			t.Fatal("duplicate atom", id)
		}
		seen[id] = true
	}
	if registry["level_2_materialized"] != false || index["level_2_materialized"] != false {
		t.Fatal("Level 2 materialized early")
	}
	rules, actions := Strings(files[".agent/level1/rule.yaml"]["atom_ids"]), Strings(files[".agent/level1/action.yaml"]["atom_ids"])
	overlap := false
	for _, id := range rules {
		if Has(actions, id) {
			overlap = true
		}
	}
	if !overlap {
		t.Fatal("multilabel atom did not share identity")
	}
	for ref, projection := range files {
		if strings.Contains(ref, "/level1/") && projection["normalized_text"] != nil {
			t.Fatal("text duplicated in projection", ref)
		}
	}
	if files[".agent/unresolved.yaml"]["routing_state"] != "UNRESOLVED" {
		t.Fatal(files)
	}
}

func TestAgentMaterializationPreservesRawExcerptAndDigest(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\n- Never edit generated files directly.\n")
	files, err := BuildAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(files[".agent/registry.yaml"]["atoms"]) {
		row := Map(raw)
		if row["normalized_text"] == "Never edit generated files directly." {
			source := Map(row["source"])
			if source["raw_excerpt"] != "- Never edit generated files directly.\n" || len(Text(source["raw_excerpt_sha256"])) != 64 {
				t.Fatal(source)
			}
			return
		}
	}
	t.Fatal("source atom missing")
}

func TestAgentLevel2RemainsUnmaterialized(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\nBefore publishing, run pytest and report the status.\n")
	files, err := BuildAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := CanonicalJSON(file)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("\"level_2\"")) || bytes.Contains(data, []byte("\"PUBLISH\"")) || bytes.Contains(data, []byte("\"TEST\"")) {
			t.Fatal("Level 2 labels leaked into Level 1 materialization")
		}
	}
}

func TestAgentUnresolvedAtomsAreSeparateFromOther(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\nCanonical roles:\n\nALPHA BETA GAMMA\n")
	files, err := BuildAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil {
		t.Fatal(err)
	}
	unresolved := Strings(files[".agent/unresolved.yaml"]["atom_ids"])
	if len(unresolved) == 0 {
		t.Fatal("unresolved atoms dropped")
	}
	other := Strings(files[".agent/level1/other.yaml"]["atom_ids"])
	for _, id := range unresolved {
		if Has(other, id) {
			t.Fatal("unresolved coerced to OTHER", id)
		}
	}
}

func TestAgentMaterializeCheckIsReproducibleAndSourceDriftIsStale(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\nRead `README.md` before editing.\n")
	written, err := MaterializeAgent(r, "AGENTS.md", ".agent")
	if err != nil || len(Strings(written["written"])) == 0 {
		t.Fatal(written, err)
	}
	result, err := CheckAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil || result["state"] != "CURRENT" {
		t.Fatal(result, err)
	}
	if err := r.AtomicWrite("AGENTS.md", []byte("# AGENTS\n\nRead `README.md` before editing.\n\nNever rewrite history.\n"), nil); err != nil {
		t.Fatal(err)
	}
	result, err = CheckAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil || result["state"] != "STALE" {
		t.Fatal(result, err)
	}
}

func TestAgentEntryResolutionUsesLevel1AndExplicitWidening(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\nBefore editing, read README.md.\n\nNever edit generated files directly.\n\nValidation evidence must include pytest status.\n\nArchitecture baseline: MVC plus EDA\n\nCanonical roles:\n\nALPHA BETA GAMMA\n")
	if _, err := MaterializeAgent(r, "AGENTS.md", ".agent"); err != nil {
		t.Fatal(err)
	}
	base, err := ResolveAgentEntry(r, "MODIFY", nil, ".agent")
	if err != nil || !reflect.DeepEqual(Strings(base["selected_level_1"]), []string{"APPLICABILITY", "RULE", "ACTION"}) || base["level_2_used"] != false || agentInt(base["unresolved_count"]) < 1 {
		t.Fatal(base, err)
	}
	widened, err := ResolveAgentEntry(r, "MODIFY", []string{"OTHER"}, ".agent")
	if err != nil || !Has(Strings(widened["selected_level_1"]), "OTHER") || agentInt(widened["instruction_count"]) < agentInt(base["instruction_count"]) {
		t.Fatal(widened, err)
	}
	verify, err := ResolveAgentEntry(r, "VERIFY", nil, ".agent")
	if err != nil || !reflect.DeepEqual(Strings(verify["selected_level_1"]), []string{"APPLICABILITY", "RULE", "ACTION", "EVIDENCE"}) || verify["level_2_used"] != false {
		t.Fatal(verify, err)
	}
	if _, err := ResolveAgentEntry(r, "DEPLOY", nil, ".agent"); err == nil {
		t.Fatal("unregistered operation admitted")
	}
}

func progressiveRegressionFixture(t *testing.T) *Repository {
	t.Helper()
	return agentRegressionFixture(t, "# AGENTS\n\nThese instructions apply to coding agents working anywhere in this repository.\n\nBefore broadly reading repository policy or planning prose, resolve the developer-policy context mechanically:\n\n- Do not scan developer/policy/** merely to discover policy.\n\nCanonical roles:\n\nALPHA BETA GAMMA\n")
}

func TestAgentProgressiveBootstrapCompactsToOneMachineEntry(t *testing.T) {
	r := progressiveRegressionFixture(t)
	result, err := MigrateAgentLevel1(r)
	if err != nil || result["mode"] != "COMPACTED_FROM_AGENTS" || agentInt(result["pass_count"]) == 0 || agentInt(result["unresolved_count"]) == 0 || result["integration"] != "LOCAL_CLI_ONLY" {
		t.Fatal(result, err)
	}
	text, err := agentText(r, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(text, agentDirective) != 1 || strings.Contains(text, "PTSIP_AGENT_ROUTE") || strings.Contains(text, "These instructions apply to coding agents working anywhere") || !strings.Contains(text, "Canonical roles:") {
		t.Fatal(text)
	}
	for _, ref := range []string{".agent/index.yaml", ".agent/stages/level1.json", ".agent/unresolved/level1.json"} {
		if !iwpPathExists(r, ref) {
			t.Fatal("missing compact projection", ref)
		}
	}
	if iwpPathExists(r, ".agent/registry.yaml") {
		t.Fatal("legacy registry created")
	}
	if failures, err := CheckAgentProgressive(r); err != nil || len(failures) != 0 {
		t.Fatal(failures, err)
	}
}

func TestAgentProgressiveIndexKeepsExactLookupAndLocalCLIIntegration(t *testing.T) {
	r := progressiveRegressionFixture(t)
	if _, err := MigrateAgentLevel1(r); err != nil {
		t.Fatal(err)
	}
	stage, err := r.Read(".agent/stages/level1.json")
	if err != nil {
		t.Fatal(err)
	}
	if agentInt(stage["pass_count"]) != len(List(stage["pass_order"])) || agentInt(stage["pass_count"]) != len(Map(stage["pass_by_atom"])) {
		t.Fatal(stage)
	}
	text, err := agentText(r, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(stage["pass_order"]) {
		id := Text(raw)
		if Map(Map(stage["pass_by_atom"])[id])["atom_id"] != id || strings.Contains(text, id) {
			t.Fatal("atom lookup leaked into compact entry", id)
		}
	}
	index, err := r.Read(".agent/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if Map(index["progressive_reasoning"])["source_state"] != agentCompactState || Map(index["source"])["sha256"] != agentHash(text) {
		t.Fatal(index)
	}
	integration := Map(index["integration"])
	mcp := Map(integration["mcp"])
	if integration["mode"] != "LOCAL_CLI_ONLY" || Map(integration["local_cli"])["state"] != "READY" || mcp["state"] != "ABSENT" || mcp["implementation_state"] != "NOT_AVAILABLE" || mcp["user_approval_required_before_install"] != true || mcp["offer_to_user_now"] != false {
		t.Fatal(integration)
	}
}

func TestAgentProgressiveEntryConsumesStageWithoutPreviousLevelRerun(t *testing.T) {
	r := progressiveRegressionFixture(t)
	if _, err := MigrateAgentLevel1(r); err != nil {
		t.Fatal(err)
	}
	result, err := ResolveAgentEntry(r, "MODIFY", nil, ".agent")
	if err != nil || result["stage"] != "PROGRESSIVE_LEVEL_1" || result["previous_level_rerun"] != false || len(List(result["instructions"])) == 0 || len(List(result["unresolved"])) == 0 {
		t.Fatal(result, err)
	}
	for _, raw := range List(result["instructions"]) {
		item := Map(raw)
		if item["machine"] == nil || item["natural_residual"] == nil || !strings.HasSuffix(Text(item["stage_ref"]), Text(item["atom_id"])) {
			t.Fatal(item)
		}
	}
	for _, raw := range List(result["unresolved"]) {
		if Map(raw)["natural_language"] == nil {
			t.Fatal(raw)
		}
	}
}

func TestAgentProgressiveCompactEntryChangesMarkStageStale(t *testing.T) {
	r := progressiveRegressionFixture(t)
	if _, err := MigrateAgentLevel1(r); err != nil {
		t.Fatal(err)
	}
	text, err := agentText(r, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.AtomicWrite("AGENTS.md", []byte(text+"\nNew instruction.\n"), nil); err != nil {
		t.Fatal(err)
	}
	failures, err := CheckAgentProgressive(r)
	if err != nil || !Has(failures, "COMPACT_AGENTS_STALE") {
		t.Fatal(failures, err)
	}
}

func TestAgentProgressiveExistingStageCompactsWithoutReclassification(t *testing.T) {
	r := progressiveRegressionFixture(t)
	stage := Object{"schema_version": "ptsip-agent-progressive-reasoning/v1", "level": 1, "pass_count": 1, "unresolved_count": 1, "pass_order": []any{"A0001"}, "pass_by_atom": Object{"A0001": Object{"atom_id": "A0001", "pass_header": Object{"level": 1, "status": "PASS", "labels": []any{"RULE"}}, "machine": Object{"level_1_labels": []any{"RULE"}}, "natural_residual": []any{}}}}
	unresolved := Object{"schema_version": "ptsip-agent-progressive-reasoning/v1", "level": 1, "routing_state": "UNRESOLVED", "count": 1, "items": []any{Object{"atom_id": "A0002", "status": "UNRESOLVED", "heading_path": []any{"AGENTS.md", "Context"}, "natural_language": "Needs natural reasoning."}}}
	if err := r.WriteJSON(".agent/stages/level1.json", stage, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteJSON(".agent/unresolved/level1.json", unresolved, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteYAML(".agent/index.yaml", Object{"schema_version": "ptsip-agent-progressive-index/v1", "management_mode": "PROGRESSIVE_LEVEL_1", "progressive_reasoning": Object{"highest_materialized_level": 1, "per_atom_advancement": true, "level_1_ref": agentStageRef, "level_1_unresolved_ref": agentUnresolvedRef, "current_next_level_candidate_count": 1, "next_level_candidate_set_is_dynamic": true, "unresolved_reassessment_source": agentUnresolvedRef, "unresolved_blocks_next_level_candidates": false, "source_state": "ROUTED_LEVEL_1"}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.AtomicWrite("AGENTS.md", []byte("# AGENTS\n\nPTSIP_AGENT_ROUTE level=1 atom_id=A0001 ref=\".agent/stages/level1.json#/pass_by_atom/A0001\"\n\nNeeds natural reasoning.\n"), nil); err != nil {
		t.Fatal(err)
	}
	stagePath, _ := r.Path(".agent/stages/level1.json")
	beforeStage, err := os.ReadFile(stagePath)
	if err != nil {
		t.Fatal(err)
	}
	unresolvedPath, _ := r.Path(".agent/unresolved/level1.json")
	beforeUnresolved, err := os.ReadFile(unresolvedPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := MigrateAgentLevel1(r)
	if err != nil || result["mode"] != "COMPACTED_EXISTING_LEVEL_1" {
		t.Fatal(result, err)
	}
	afterStage, err := os.ReadFile(stagePath)
	if err != nil || !bytes.Equal(beforeStage, afterStage) {
		t.Fatal("passed stage reclassified", err)
	}
	afterUnresolved, err := os.ReadFile(unresolvedPath)
	if err != nil || !bytes.Equal(beforeUnresolved, afterUnresolved) {
		t.Fatal("unresolved stage reclassified", err)
	}
	text, err := agentText(r, "AGENTS.md")
	if err != nil || !strings.Contains(text, agentDirective) || strings.Contains(text, "PTSIP_AGENT_ROUTE") || !strings.Contains(text, "Needs natural reasoning.") {
		t.Fatal(text, err)
	}
	if failures, err := CheckAgentProgressive(r); err != nil || len(failures) != 0 {
		t.Fatal(failures, err)
	}
}

func TestAgentActivationAndUnavailableMCPPreserveLocalCLIState(t *testing.T) {
	r := agentRegressionFixture(t, "# AGENTS\n\nBefore editing, read README.md.\n\nCanonical roles:\n\nALPHA BETA GAMMA\n")
	result, err := MigrateAgentLevel1(r)
	if err != nil || result["integration"] != "LOCAL_CLI_ONLY" {
		t.Fatal(result, err)
	}
	status, err := AgentIntegrationStatus(r)
	if err != nil {
		t.Fatal(err)
	}
	mcp := Map(status["mcp"])
	if status["mode"] != "LOCAL_CLI_ONLY" || Map(status["local_cli"])["state"] != "READY" || mcp["state"] != "ABSENT" || mcp["implementation_state"] != "NOT_AVAILABLE" || mcp["user_approval_required_before_install"] != true || mcp["automatic_install_without_user_approval"] != false || mcp["offer_to_user_now"] != false || !strings.Contains(Text(status["entry_directive"]), `mcp="ABSENT"`) || !strings.HasSuffix(Text(mcp["intended_install_command"]), "install-mcp --user-approved") {
		t.Fatal(status)
	}
	indexPath, _ := r.Path(".agent/index.yaml")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InstallAgentMCP(r, false); err == nil {
		t.Fatal("unavailable MCP installed")
	}
	after, err := os.ReadFile(indexPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("unavailable install mutated integration", err)
	}
	text, err := agentText(r, "AGENTS.md")
	if err != nil || !strings.Contains(text, agentDirective) || strings.Contains(text, "PTSIP_AGENT_ROUTE") || !strings.Contains(text, "ALPHA BETA GAMMA") {
		t.Fatal(text, err)
	}
	if failures, err := CheckAgentProgressive(r); err != nil || len(failures) != 0 {
		t.Fatal(failures, err)
	}
}
