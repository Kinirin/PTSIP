package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAutomaticPlanEmptyTargetsMatchRegistryWithoutWeakeningExactSelection(t *testing.T) {
	goOnly := mode{ID: "repository-release", ComponentRef: "repository-release-verification"}
	goOnly.Execution.Go = []string{"developer/tests/release"}
	pythonOnly := mode{ID: "agent-contract-plane", ComponentRef: "agent-contract-plane-verification"}
	pythonOnly.Execution.Pytest = []string{"src/tests/agent_contracts/test_contract_plane.py"}
	ordered := mode{ID: "ordered", ComponentRef: "ordered-verification"}
	ordered.Execution.Go = []string{"first", "second"}
	r := registry{Modes: []mode{goOnly, pythonOnly, ordered}}
	for _, test := range []struct {
		name    string
		entries []plannedMode
		valid   bool
	}{
		{"empty-pytest", []plannedMode{{ID: goOnly.ID, ComponentRef: goOnly.ComponentRef, Pytest: []string{}, Go: goOnly.Execution.Go}}, true},
		{"empty-go", []plannedMode{{ID: pythonOnly.ID, ComponentRef: pythonOnly.ComponentRef, Pytest: pythonOnly.Execution.Pytest, Go: []string{}}}, true},
		{"added-target", []plannedMode{{ID: goOnly.ID, ComponentRef: goOnly.ComponentRef, Pytest: []string{"unregistered.py"}, Go: goOnly.Execution.Go}}, false},
		{"missing-target", []plannedMode{{ID: goOnly.ID, ComponentRef: goOnly.ComponentRef, Pytest: []string{}, Go: []string{}}}, false},
		{"wrong-component", []plannedMode{{ID: goOnly.ID, ComponentRef: "unregistered", Go: goOnly.Execution.Go}}, false},
		{"duplicate-mode", []plannedMode{toPlan(goOnly), toPlan(goOnly)}, false},
		{"reordered-targets", []plannedMode{{ID: ordered.ID, ComponentRef: ordered.ComponentRef, Go: []string{"second", "first"}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "automatic-plan.json")
			data, err := json.Marshal(selection{Resolution: "SELECTED", Plan: test.entries})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			_, err = resolve(r, "", path)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
		})
	}
}
