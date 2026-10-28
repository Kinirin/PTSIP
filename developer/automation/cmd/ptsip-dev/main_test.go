package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBranchCompatibilityEntryUsesOnlySuccessorContract(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	const contract = ".ptsip/engine/developer/Distribution/branch/index.json"
	raw, err := os.ReadFile(filepath.Join(root, contract))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	filename := filepath.Join(fixture, contract)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, raw, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := run([]string{"branch-control", "commands", "--repository", fixture})
	if err != nil {
		t.Fatalf("compatibility call required retired policy catalog: %v", err)
	}
	if result.(map[string]any)["execution_authorized"] != false {
		t.Fatal("compatibility call granted authority")
	}
}

func TestCleanupCLIUsesRegisteredRepeatedScopes(t *testing.T) {
	result, err := run([]string{"markdown-cleanup", "inspect", "--scope", "developer/tests/policy", "--scope", "developer/tests/planning"})
	if err != nil {
		t.Fatal(err)
	}
	scopes, ok := result.(map[string]any)["scopes"].([]string)
	if !ok || !reflect.DeepEqual(scopes, []string{"developer/tests/policy", "developer/tests/planning"}) {
		t.Fatalf("scope arguments lost: %#v", result)
	}
}

func TestCLIRejectsUnregisteredRepetitionAndMissingRepeatedValues(t *testing.T) {
	for _, request := range [][]string{
		{"policy-resolver", "resolve", "--scope", ".", "--scope", "developer/automation", "--operation", "READ"},
		{"markdown-cleanup", "inspect", "--scope", "--scope", "developer/tests/policy"},
	} {
		if _, err := run(request); err == nil || !(strings.Contains(err.Error(), "duplicate option") || strings.Contains(err.Error(), "requires a value")) {
			t.Fatalf("invalid named arguments admitted: %v, error=%v", request, err)
		}
	}
}
