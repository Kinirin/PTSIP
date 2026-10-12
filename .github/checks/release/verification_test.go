package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func verificationRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, ".github", "checks", "release", "verification-contract.json")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("release verification execution map missing")
		}
		root = parent
	}
}

func TestLegacyChecksAreRetiredOnlyWithExplicitCoverageMapping(t *testing.T) {
	root := verificationRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".github", "checks", "release", "verification-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Authority string `json:"authority"`
		Migration map[string]struct {
			Replacement []string `json:"replacement"`
			Readiness   string   `json:"readiness_replacement"`
			Scope       string   `json:"execution_scope"`
		} `json:"legacy_migration"`
		Targets map[string][]string `json:"targets"`
	}
	if err = json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Migration) != 6 || doc.Authority != "EXECUTION_BINDING_ONLY_CURRENT_DEVELOPER_POLICIES_REMAIN_AUTHORITY" {
		t.Fatal("retirement must preserve a non-authoritative explicit coverage map")
	}
	for old, entry := range doc.Migration {
		if _, err := os.Stat(filepath.Join(root, ".github", "checks", old)); !os.IsNotExist(err) {
			t.Fatalf("retired flat check remains: %s", old)
		}
		if len(entry.Replacement) == 0 {
			t.Fatal("retirement lost coverage", old)
		}
		for _, ref := range entry.Replacement {
			if strings.HasSuffix(ref, ".ps1") {
				if _, err := os.Stat(filepath.Join(root, ".github", "checks", "release", filepath.FromSlash(ref))); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if doc.Migration["release-a3-regression.ps1"].Readiness != "developer/tests/release" {
		t.Fatal("removed Python readiness must have its explicitly approved Go replacement")
	}
	if doc.Migration["release-go-contracts.ps1"].Scope != "EXPLICIT_REPOSITORY_VERIFICATION" {
		t.Fatal("repository regression became an implicit engine dependency")
	}
	tool, err := os.ReadFile(filepath.Join(root, ".github", "checks", "release", "Tool_release.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"verification/Tool_contract_verify.ps1", "verification/Tool_regression_verify.ps1", "Tool_artifact_verify.ps1", "verification/Tool_runtime_verify.ps1"} {
		if !strings.Contains(string(tool), ref) {
			t.Fatal("Tool target lost required migrated verification", ref)
		}
	}
	for _, id := range []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"} {
		text, err := os.ReadFile(filepath.Join(root, ".github", "checks", "release", "engines_release", id+"_release.ps1"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(text), "Tool_regression") || strings.Contains(string(text), "Repository_") {
			t.Fatal("single-engine release invoked unrelated regression", id)
		}
	}
}
