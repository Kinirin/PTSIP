package schema_check_test

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
	const planID = "urn:ptsip:contract:plan:public-structure"
	const lifecycleID = "urn:ptsip:contract:plan:lifecycle"
	const version = "SCH.1.00"
	wantPaths := map[string]string{
		planID: "plan/public-structure-plan.schema.json", "urn:ptsip:contract:plan:identity": "plan/plan-identity.schema.json",
		lifecycleID: "plan/plan-lifecycle.schema.json", "urn:ptsip:contract:plan:unit": "plan/plan-unit.schema.json",
	}
	if len(catalog.Plan) != 1 {
		t.Fatalf("unexpected Plan bindings: %v", catalog.Plan)
	}
	binding := catalog.Plan[planID]
	if binding.Path != wantPaths[planID] || binding.DiagnosticOnly || binding.ID != planID || binding.Version != version {
		t.Fatalf("incorrect Plan root registration: %+v", binding)
	}
	docs := make(map[string]map[string]any, len(wantPaths))
	for id, path := range wantPaths {
		var doc map[string]any
		readContractJSON(t, filepath.Join(schemaRoot, filepath.FromSlash(path)), &doc)
		if doc["$id"] != id || doc["$schema"] != validator.Draft || doc["version"] != version {
			t.Fatalf("incorrect Plan resource identity, version or dialect at %s: %v", path, doc)
		}
		docs[id] = doc
	}
	for _, id := range []string{planID, "urn:ptsip:contract:plan:identity", lifecycleID, "urn:ptsip:contract:plan:unit"} {
		t.Run(id, func(t *testing.T) {
			schema, err := validator.Compile(resolver.Graph{Root: id, Docs: docs})
			if err != nil {
				t.Fatal(err)
			}
			if id == planID {
				instance := map[string]any{"SchemaVersion": version, "Plan_lifecycle": map[string]any{"status": "Draft.Proposed"}}
				if err := schema.Validate(instance); err != nil {
					t.Fatalf("Plan root rejected its registered lifecycle resource: %v", err)
				}
				instance["Plan_lifecycle"] = map[string]any{"status": "DRAFT"}
				if err := schema.Validate(instance); err == nil {
					t.Fatal("Plan root accepted a retired status in its lifecycle resource")
				}
			}
			if id != lifecycleID {
				return
			}
			current := []string{
				"Draft.Proposed", "Draft.Under_coordination", "Approval.Scheduled_implementation", "Approval.In_progress",
				"Approval.Implementation_complete", "Halt.Paused", "Halt.Discontinued",
			}
			for _, status := range current {
				if err := schema.Validate(map[string]any{"status": status}); err != nil {
					t.Errorf("Plan lifecycle rejected %s: %v", status, err)
				}
			}
			for _, input := range []map[string]any{
				{}, {"status": "DRAFT"}, {"status": "ACTIVATE"}, {"status": "INTERRUPTION"}, {"status": "APPROVED"},
			} {
				if err := schema.Validate(input); err == nil {
					t.Errorf("Plan lifecycle accepted missing, retired or unrelated status: %v", input)
				}
			}
		})
	}
}
