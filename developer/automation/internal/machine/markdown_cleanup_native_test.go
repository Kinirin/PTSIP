package machine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cleanupFixture(t *testing.T) *Repository {
	t.Helper()
	repo := policyTestRepo(t)
	if err := os.MkdirAll(filepath.Join(repo.Root, "spec"), 0755); err != nil {
		t.Fatal(err)
	}
	policyTestWrite(t, repo, cleanupWorkflow, Object{
		"schema_version": "ptsip-markdown-cleanup-workflow/v1", "artifact_class": "DEVELOPER_EXECUTION_PROJECTION", "projection_authority": false,
		"policy_ref": cleanupRootPolicy + "#rules." + cleanupRootSection + ".markdown_cleanup_preconditions",
		"stages":     []any{"INSPECT", "PLAN", "VERIFY", "APPLY", "POST_VALIDATE"}, "default_scopes": []any{"spec"},
		"reference_scan_exclusions":    []any{cleanupWorkflow, "developer/policy/GOV/MPD-GOV-0001.yaml", "developer/automation/internal/machine/markdown_cleanup_native_test.go", "developer/tests/test_markdown_cleanup.py"},
		"apply_requires_policy_status": "ACTIVE", "targets": []any{Object{"path": "spec/example.md", "semantic_role": "NORMATIVE_RULE", "disposition": "REMOVE_CANDIDATE"}},
	})
	return repo
}

func cleanupWrite(t *testing.T, repo *Repository, ref, content string) {
	t.Helper()
	path, err := repo.Path(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupCurrentWorkflowHasExactRootOwnershipAndTargets(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := repo.CleanupWorkflow()
	if err != nil {
		t.Fatal(err)
	}
	if workflow["projection_authority"] != false || len(List(workflow["targets"])) != 8 {
		t.Fatal("cleanup workflow identity or coverage changed")
	}
	for _, operation := range []string{"inspect", "plan", "verify", "simulate", "apply"} {
		opts := map[string]string{}
		if operation == "apply" {
			opts["--plan"] = "fixture.json"
		}
		if err := repo.AdmitCommand([]string{"markdown-cleanup", operation}, opts); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCleanupReferenceClassesAndWorkflowExclusions(t *testing.T) {
	for _, test := range []struct{ name, source, content, state, kind string }{
		{"machine", "tool.go", "package fixture\nconst Source = \"spec/example.md\"\n", "BLOCKED_BY_ACTIVE_REFERENCE", "ACTIVE_MACHINE_DEPENDENCY"},
		{"human", "history.md", "Historical: spec/example.md\n", "REMOVE_CANDIDATE", "HISTORICAL_OR_HUMAN_REFERENCE"},
		{"workflow exclusion", "", "", "REMOVE_CANDIDATE", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := cleanupFixture(t)
			cleanupWrite(t, repo, "spec/example.md", "# Normative\n")
			if test.source != "" {
				cleanupWrite(t, repo, test.source, test.content)
			}
			result, err := repo.InspectMarkdown([]string{"spec"})
			if err != nil {
				t.Fatal(err)
			}
			item := Map(List(result["items"])[0])
			if item["state"] != test.state {
				t.Fatalf("state=%v, refs=%#v", item["state"], item["references"])
			}
			refs := List(item["references"])
			if test.kind == "" {
				if len(refs) != 0 {
					t.Fatal(refs)
				}
				return
			}
			if len(refs) != 1 || Map(refs[0])["kind"] != test.kind || Map(refs[0])["source_path"] != test.source {
				t.Fatalf("reference classification=%#v", refs)
			}
		})
	}
}

func TestCleanupUnregisteredMarkdownRemainsUnresolved(t *testing.T) {
	repo := cleanupFixture(t)
	cleanupWrite(t, repo, "spec/unregistered.md", "# Unknown role\n")
	result, err := repo.InspectMarkdown([]string{"spec"})
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]string{}
	for _, raw := range List(result["items"]) {
		item := Map(raw)
		states[Text(item["path"])] = Text(item["state"])
	}
	if states["spec/unregistered.md"] != "UNRESOLVED" || states["spec/example.md"] != "ALREADY_ABSENT" {
		t.Fatal(states)
	}
}

func TestCleanupReferenceScanSupportsWindowsPathForm(t *testing.T) {
	repo := cleanupFixture(t)
	cleanupWrite(t, repo, "spec/example.md", "# Normative\n")
	cleanupWrite(t, repo, "tool.ps1", "$p = \"spec\\example.md\"\n")
	workflow, err := repo.CleanupWorkflow()
	if err != nil {
		t.Fatal(err)
	}
	hits, err := repo.MarkdownReferenceHits("spec/example.md", Strings(workflow["reference_scan_exclusions"]))
	if err != nil || len(hits) != 1 || Map(hits[0])["source_path"] != "tool.ps1" || Map(hits[0])["kind"] != "ACTIVE_MACHINE_DEPENDENCY" {
		t.Fatalf("hits=%#v, error=%v", hits, err)
	}
}

func TestCleanupApplyRejectsDraftRootPolicyBeforeAnyMutation(t *testing.T) {
	repo := cleanupFixture(t)
	policy, err := repo.Read("developer/policy/GOV/MPD-GOV-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	Map(policy["policy"])["status"], Map(policy["policy"])["version"] = "DRAFT", "0.0"
	policyTestWrite(t, repo, "developer/policy/GOV/MPD-GOV-0001.yaml", policy)
	index, err := repo.Read("developer/policy/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(index["policies"]) {
		if Map(raw)["id"] == cleanupRootPolicy {
			Map(raw)["status"] = "DRAFT"
		}
	}
	policyTestWrite(t, repo, "developer/policy/index.yaml", index)
	plan := Object{"schema_version": "ptsip-markdown-cleanup-plan/v1", "state": "READY", "apply_authorized": true, "repository_head": nil, "target_hashes": Object{}, "remove": []any{}}
	if _, err := repo.ApplyMarkdownPlan(plan); err == nil || !strings.Contains(err.Error(), "not ACTIVE") {
		t.Fatalf("draft cleanup admitted: %v", err)
	}
}
