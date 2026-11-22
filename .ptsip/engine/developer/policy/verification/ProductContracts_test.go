package schema_check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

type productSchemaRegistration struct {
	Path           string `json:"path"`
	ID             string `json:"$id"`
	Version        string `json:"version"`
	DiagnosticOnly bool   `json:"diagnostic_only"`
}

type productSchemaCatalog struct {
	Plan    map[string]productSchemaRegistration            `json:"plan"`
	Storage map[string]map[string]productSchemaRegistration `json:"storage"`
}

func readContractJSON(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("invalid JSON at %s: %v", name, err)
	}
}

func TestRegisteredRBACListsMatchSharedConformanceVectors(t *testing.T) {
	root := routingRepositoryRoot(t)
	schemaRoot := filepath.Join(root, "src", "ptsip", "schemas")
	var catalog productSchemaCatalog
	readContractJSON(t, filepath.Join(schemaRoot, "config.json"), &catalog)
	bindings := catalog.Storage["Responsibility-Authority"]
	if len(bindings) != 3 {
		t.Fatalf("expected RA, RAC and AEC list registrations: %v", bindings)
	}

	var cases []struct {
		Name         string `json:"name"`
		Registration string `json:"registration"`
		Valid        bool   `json:"valid"`
		Instance     any    `json:"instance"`
	}
	readContractJSON(t, filepath.Join(root, "src", "tests", "ptsip", "rbac-list-conformance.json"), &cases)
	wantPaths := map[string]string{
		"RA_list": "storage/rbac/ra-list.schema.json", "RAC_list": "storage/rbac/rac-list.schema.json", "AEC_list": "storage/rbac/aec-list.schema.json",
	}
	for _, key := range []string{"RA_list", "RAC_list", "AEC_list"} {
		t.Run(key, func(t *testing.T) {
			binding := bindings[key]
			if binding.Path != wantPaths[key] || binding.ID == "" || binding.Version == "" || binding.DiagnosticOnly {
				t.Fatalf("incorrect list registration: %+v", binding)
			}
			var doc map[string]any
			readContractJSON(t, filepath.Join(schemaRoot, filepath.FromSlash(binding.Path)), &doc)
			if doc["$id"] != binding.ID {
				t.Fatalf("registered list identity differs from schema: %s != %v", binding.ID, doc["$id"])
			}
			schema, err := validator.Compile(resolver.Graph{Root: binding.ID, Docs: map[string]map[string]any{binding.ID: doc}})
			if err != nil {
				t.Fatal(err)
			}
			var accepted, rejected int
			for _, tc := range cases {
				if _, ok := bindings[tc.Registration]; !ok {
					t.Fatalf("test vector names an unregistered list: %s", tc.Registration)
				}
				if tc.Registration != key {
					continue
				}
				if tc.Valid {
					accepted++
				} else {
					rejected++
				}
				t.Run(tc.Name, func(t *testing.T) {
					err := schema.Validate(tc.Instance)
					if (err == nil) != tc.Valid {
						t.Fatalf("want valid=%t, got %v", tc.Valid, err)
					}
				})
			}
			if accepted == 0 || rejected == 0 {
				t.Fatal("list needs both valid and invalid conformance vectors")
			}
		})
	}
}

func TestPlanDraftDiagnosticBindingsAndVocabulary(t *testing.T) {
	root := routingRepositoryRoot(t)
	schemaRoot := filepath.Join(root, "src", "ptsip", "schemas")
	var catalog productSchemaCatalog
	readContractJSON(t, filepath.Join(schemaRoot, "config.json"), &catalog)
	wantPaths := map[string]string{
		"document": "plan/public-structure-plan.schema.json", "identity": "plan/plan-identity.schema.json",
		"lifecycle": "plan/plan-lifecycle.schema.json", "unit": "plan/plan-unit.schema.json",
	}
	if len(catalog.Plan) != len(wantPaths) {
		t.Fatalf("unexpected Plan bindings: %v", catalog.Plan)
	}
	for _, key := range []string{"document", "identity", "lifecycle", "unit"} {
		t.Run(key, func(t *testing.T) {
			binding := catalog.Plan[key]
			if binding.Path != wantPaths[key] || !binding.DiagnosticOnly || binding.ID != "" || binding.Version != "" {
				t.Fatalf("Plan path is not a diagnostic-only draft: %+v", binding)
			}
			var doc map[string]any
			readContractJSON(t, filepath.Join(schemaRoot, filepath.FromSlash(binding.Path)), &doc)
			if _, exists := doc["$id"]; exists || doc["$schema"] != validator.Draft {
				t.Fatalf("unexpected Plan draft identity or dialect: %v", doc)
			}
			// This test-only resource URI compiles the anonymous draft without registering it.
			id := "urn:ptsip:test:plan-diagnostic:" + key
			schema, err := validator.Compile(resolver.Graph{Root: id, Docs: map[string]map[string]any{id: doc}})
			if err != nil {
				t.Fatal(err)
			}
			if key != "lifecycle" {
				return
			}
			current := []string{
				"Draft.Proposed", "Draft.Under_coordination", "Approval.Scheduled_implementation", "Approval.In_progress",
				"Approval.Implementation_complete", "Halt.Paused", "Halt.Discontinued",
			}
			for _, status := range current {
				if err := schema.Validate(map[string]any{"status": status}); err != nil {
					t.Errorf("Plan diagnostic rejected %s: %v", status, err)
				}
			}
			for _, input := range []map[string]any{
				{}, {"status": "DRAFT"}, {"status": "ACTIVATE"}, {"status": "INTERRUPTION"}, {"status": "APPROVED"},
			} {
				if err := schema.Validate(input); err == nil {
					t.Errorf("Plan diagnostic accepted missing, retired or unrelated status: %v", input)
				}
			}
		})
	}
}
