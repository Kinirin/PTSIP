package responsibilityauthority_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Responsibility-Authority/verification"
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	schemas "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

func TestResponsibilityAuthoringNeedsNoPolicyOrLegacyTree(t *testing.T) {
	source, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, path := range []string{
		authoring.ProfilePath, authoring.DefinitionsPath,
		authoring.RBACSchemaRoot + "Conditions_exercise/Condition.schema.json",
		authoring.RBACSchemaRoot + "authority/Operational_Authority.schema.json",
		authoring.RBACSchemaRoot + "responsibility/Responsibility.schema.json",
		authoring.RBACSchemaRoot + "Responsibilities_List.schema.json",
		authoring.RBACSchemaRoot + "Responsibility_Authority_Contract.schema.json",
	} {
		raw, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if path == authoring.ProfilePath {
			profile, err := authoring.ReadDocument(source, path)
			if err != nil {
				t.Fatal(err)
			}
			delete(profile, "component_map")
			delete(profile, "component_bindings")
			raw, err = json.Marshal(profile)
			if err != nil {
				t.Fatal(err)
			}
		}
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	validate, err := schemas.InstalledValidator()
	if err != nil {
		t.Skipf("installed PTSIP validation unavailable: %v", err)
	}
	report, code := verification.Run(root, "check", "", "", validate)
	if code != 0 || report.(map[string]any)["status"] != "VALID" {
		t.Fatalf("standalone validation failed: %v", report)
	}
	if report.(map[string]any)["execution_authorized"] != false {
		t.Fatal("structure validation granted authority")
	}
	catalog, err := authoring.LoadResponsibilities(root)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := authoring.ReadDocument(root, authoring.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	profile["responsibility"] = []any{map[string]any{
		"ID": "R-ID-example", "name": "EXAMPLE", "role_name": "EXAMPLE_ROLE",
		"condition_id": []any{"C-ID-missing"}, "Operational_Authority": []any{"OA-ID-missing"}, "runtime_required": false,
	}}
	if err := catalog.CheckProfile(validate, profile); err == nil {
		t.Fatal("unresolved condition and authority accepted")
	}
}
