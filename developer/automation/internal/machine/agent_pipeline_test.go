package machine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func agentFixture(t *testing.T) *Repository {
	t.Helper()
	source, err := Open("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	target := &Repository{Root: t.TempDir()}
	root := filepath.Join(source.Root, "developer", "policy")
	err = filepath.WalkDir(root, func(ref string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, _ := filepath.Rel(source.Root, ref)
		if item.IsDir() {
			if item.Name() == "legacy" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(ref)
		if err != nil {
			return err
		}
		return target.AtomicWrite(filepath.ToSlash(relative), data, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestAgentClassifierAncestryAndSourceRanges(t *testing.T) {
	text := "# AGENTS.md\n\nWhen editing files, you must preserve compatibility:\n- Read the dependency.\n  - run pytest and verify status.\n\nMysterious fragment\n\nCommands:\n```\ngo test ./...\n```\n"
	atoms := ClassifyAgentMarkdown(text)
	if len(atoms) != 6 {
		t.Fatalf("atoms=%d: %#v", len(atoms), atoms)
	}
	if !agentHas(atoms[1].InheritedLevel1, "APPLICABILITY") || !agentHas(atoms[1].InheritedLevel1, "RULE") {
		t.Fatal("introductory condition and rule not inherited")
	}
	if atoms[2].ParentAtomID == nil || *atoms[2].ParentAtomID != "A0002" {
		t.Fatal("nested list ancestry lost")
	}
	if !agentHas(atoms[2].Level1, "EVIDENCE") || !agentHas(atoms[2].Level2["EVIDENCE"], "TEST") {
		t.Fatal("test evidence not classified")
	}
	if !atoms[3].Unresolved {
		t.Fatal("unmatched text must remain unresolved")
	}
	if atoms[5].Kind != "code_block" || atoms[5].LineStart != 11 || atoms[5].LineEnd != 11 {
		t.Fatalf("code range %#v", atoms[5])
	}
	if !agentHas(atoms[5].Level1, "ACTION") {
		t.Fatal("Go command not recognized")
	}
}
func TestAgentTaxonomyUsesRootOwnersWithoutLegacy(t *testing.T) {
	r := agentFixture(t)
	if err := ValidateAgentTaxonomy(r); err != nil {
		t.Fatal(err)
	}
	if err := r.AtomicWrite("AGENTS.md", []byte("# AGENTS.md\n\nRead the file and verify status.\n\nUnmatched fragment\n"), nil); err != nil {
		t.Fatal(err)
	}
	result, err := ClassifyAgentFile(r, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	if result["policy_ref"] != "MPD-INFO-0001#unit_mpd_0010_f1b93fa1851f" {
		t.Fatal("old identity retained")
	}
	if _, err := ClassifyAgentFile(r, "../outside.md"); err == nil {
		t.Fatal("path escape admitted")
	}
}
func TestAgentMaterializationEntryAndDrift(t *testing.T) {
	r := agentFixture(t)
	_ = r.AtomicWrite("AGENTS.md", []byte("# AGENTS.md\n\nBefore editing, you must preserve compatibility:\n- Read the file.\n\nMystery fragment\n"), nil)
	if _, err := MaterializeAgent(r, "AGENTS.md", ".agent"); err != nil {
		t.Fatal(err)
	}
	current, err := CheckAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil || current["state"] != "CURRENT" {
		t.Fatalf("check=%#v err=%v", current, err)
	}
	entry, err := ResolveAgentEntry(r, "READ", nil, ".agent")
	if err != nil {
		t.Fatal(err)
	}
	if agentInt(entry["unresolved_count"]) != 1 {
		t.Fatal("unresolved atoms omitted")
	}
	if _, err := ResolveAgentEntry(r, "UNKNOWN", nil, ".agent"); err == nil {
		t.Fatal("unknown operation admitted")
	}
	if _, err := ResolveAgentEntry(r, "READ", []string{"UNRESOLVED"}, ".agent"); err == nil {
		t.Fatal("unresolved namespace admitted")
	}
	_ = r.AtomicWrite(".agent/level1/action.yaml", []byte("level_1: ACTION\natom_ids: [unknown]\n"), nil)
	if _, err := ResolveAgentEntry(r, "READ", []string{"ACTION"}, ".agent"); err == nil {
		t.Fatal("unknown projected atom admitted")
	}
	drift, err := CheckAgentMaterialization(r, "AGENTS.md", ".agent")
	if err != nil || drift["state"] != "STALE" {
		t.Fatal("projection drift missed")
	}
}
func TestAgentProgressiveIsolationAndUnresolved(t *testing.T) {
	r := agentFixture(t)
	_ = r.AtomicWrite("AGENTS.md", []byte("# AGENTS.md\n\nRead `file.yaml` before working.\n\nOpaque fragment\n"), nil)
	result, err := MigrateAgentLevel1(r)
	if err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "COMPACTED_FROM_AGENTS" {
		t.Fatal(result)
	}
	errors, err := CheckAgentProgressive(r)
	if err != nil || len(errors) > 0 {
		t.Fatalf("errors=%v err=%v", errors, err)
	}
	text, _ := agentText(r, "AGENTS.md")
	if strings.Contains(text, "Read `file.yaml`") || !strings.Contains(text, "Opaque fragment") {
		t.Fatal("passed text leaked into compact entry or unresolved lost")
	}
	entry, err := ResolveAgentEntry(r, "READ", nil, ".agent")
	if err != nil {
		t.Fatal(err)
	}
	if entry["stage"] != "PROGRESSIVE_LEVEL_1" || entry["previous_level_rerun"] != false || agentInt(entry["unresolved_count"]) != 1 {
		t.Fatal(entry)
	}
	if _, err := MigrateAgentLevel1(r); err == nil {
		t.Fatal("rerun of compact level admitted")
	}
	_ = r.AtomicWrite("AGENTS.md", []byte(text+"tampered\n"), nil)
	errors, err = CheckAgentProgressive(r)
	if err != nil || !Has(errors, "COMPACT_AGENTS_STALE") {
		t.Fatal("stale compact entry not found")
	}
}
func TestAgentStageIndependentCandidatesAndMechanism(t *testing.T) {
	items := []any{Object{"atom_id": "A0001", "level_1": []any{"ACTION", "RULE"}, "text": "Before editing, must not remove `policy.yaml`.", "heading_path": []any{}, "unresolved": false}, Object{"atom_id": "A0002", "level_1": []any{}, "text": "Unknown detail", "heading_path": []any{}, "unresolved": true}}
	stage, unresolved, err := BuildAgentStage(items)
	if err != nil {
		t.Fatal(err)
	}
	if stage["unresolved_blocks_passed_atoms"] != false || stage["unresolved_is_direct_next_level_candidate"] != false || agentInt(stage["pass_count"]) != 1 || agentInt(unresolved["count"]) != 1 {
		t.Fatal("candidate separation broken")
	}
	machine := Map(Map(Map(stage["pass_by_atom"])["A0001"])["machine"])
	if machine["trigger"] != "BEFORE" || machine["modality"] != "PROHIBITION" || !reflect.DeepEqual(Strings(machine["references"]), []string{"policy.yaml"}) {
		t.Fatal(machine)
	}
	bad := append(items, items[0])
	if _, _, err := BuildAgentStage(bad); err == nil {
		t.Fatal("duplicate stage identity admitted")
	}
}
func TestAgentIntegrationReadinessAndApproval(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	if err := r.WriteYAML(".agent/index.yaml", Object{"integration": AgentIntegrationContract()}, nil); err != nil {
		t.Fatal(err)
	}
	status, err := AgentIntegrationStatus(r)
	if err != nil || Map(status["mcp"])["offer_to_user_now"] != false {
		t.Fatal(status, err)
	}
	if _, err := InstallAgentMCP(r, true); err == nil {
		t.Fatal("unavailable installer accepted")
	}
	contract := AgentIntegrationContract()
	Map(contract["mcp"])["implementation_state"] = "READY"
	_ = r.WriteYAML(".agent/index.yaml", Object{"integration": contract}, nil)
	status, err = AgentIntegrationStatus(r)
	if err != nil || Map(status["mcp"])["offer_to_user_now"] != true {
		t.Fatal(status, err)
	}
	if _, err := InstallAgentMCP(r, false); err == nil {
		t.Fatal("unapproved MCP install accepted")
	}
	if _, err := InstallAgentMCP(r, true); err == nil {
		t.Fatal("unbound installer accepted")
	}
}
