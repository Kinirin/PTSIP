package validator_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

func TestRegisteredRBACStructuralConformance(t *testing.T) {
	schemaRoot := filepath.Join("..", "..", "schemas")
	read := func(path string) any {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		value, err := jsondoc.Decode(raw)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return value
	}

	config, _ := read(filepath.Join(schemaRoot, "config.json")).(map[string]any)
	storage, _ := config["storage"].(map[string]any)
	registrations, _ := storage["Responsibility-Authority"].(map[string]any)
	if len(registrations) != 3 {
		t.Fatalf("expected three RBAC structure registrations, got %v", registrations)
	}
	// Compile each declared RBAC schema with only its selected local resource.
	checks := make(map[string]func(any) error)
	for _, key := range []string{"RA_list", "RAC_list", "AEC_list"} {
		row, _ := registrations[key].(map[string]any)
		name, _ := row["path"].(string)
		id, _ := row["$id"].(string)
		if id == "" || path.Dir(name) != "storage/rbac" || path.Clean(name) != name {
			t.Fatalf("invalid RBAC structure registration %s: %v", key, row)
		}
		doc, ok := read(filepath.Join(schemaRoot, filepath.FromSlash(name))).(map[string]any)
		if !ok {
			t.Fatalf("%s must contain a schema object", name)
		}
		if doc["$id"] != id || doc["$schema"] != validator.Draft {
			t.Fatalf("%s must declare Draft 2020-12 and schema identity %q", name, id)
		}
		compiled, err := validator.Compile(resolver.Graph{
			Root: id,
			Docs: map[string]map[string]any{id: doc},
		})
		if err != nil {
			t.Fatalf("compile %s: %v", name, err)
		}
		checks[key] = compiled.Validate
	}

	cases, ok := read(filepath.Join("..", "..", "..", "tests", "ptsip", "rbac-list-conformance.json")).([]any)
	if !ok || len(cases) == 0 {
		t.Fatal("conformance fixtures must be a nonempty array")
	}
	for _, value := range cases {
		fixture, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("invalid conformance fixture: %#v", value)
		}
		name, _ := fixture["name"].(string)
		registration, _ := fixture["registration"].(string)
		validate := checks[registration]
		want, hasExpected := fixture["valid"].(bool)
		instance, hasInstance := fixture["instance"]
		if name == "" || validate == nil || !hasExpected || !hasInstance {
			t.Fatalf("invalid conformance fixture: %#v", fixture)
		}
		t.Run(name, func(t *testing.T) {
			err := validate(instance)
			if valid := err == nil; valid != want {
				t.Errorf("%s validity = %t, want %t (error: %v)", registration, valid, want, err)
			}
		})
	}
}
