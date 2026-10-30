package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/layout"
)

func TestModuleIndexesAndEntrypointStayInSync(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	report, code := layout.Run(root)
	if code != 0 {
		t.Fatalf("invalid module layout: %v", report)
	}
	raw, err := os.ReadFile("index.json")
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Validators map[string]string `json:"validators"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for name := range validators {
		names[name] = true
	}
	for _, name := range []string{"contract-entry", "release-version-diff", "release-documentation", "policy-rules", "responsibility-authority", "policy-check", "policy-identity-check", "policy-status-check", "plan-status-check"} {
		names[name] = true
	}
	if len(names) != len(index.Validators) {
		t.Fatal("validator index and executable registry differ")
	}
	for name := range names {
		if index.Validators[name] == "" {
			t.Fatalf("validator missing from module index: %s", name)
		}
	}
}

func TestLayoutRejectsMissingOrAuthorityGrantingIndex(t *testing.T) {
	root := t.TempDir()
	if _, code := layout.Run(root); code == 0 {
		t.Fatal("missing module index accepted")
	}
	raw, err := os.ReadFile("index.json")
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]any
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	index["execution_authorized"] = true
	filename := filepath.Join(root, ".ptsip/engine/developer/index.json")
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, raw, 0644); err != nil {
		t.Fatal(err)
	}
	report, code := layout.Run(root)
	if code == 0 || report.(map[string]any)["problem"] != "TEMPORARY_VALIDATOR_BOUNDARY_INVALID" {
		t.Fatalf("invalid authority boundary accepted: %v", report)
	}
}
