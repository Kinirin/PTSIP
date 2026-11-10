package validator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

func TestRBACProfileStructuralConformance(t *testing.T) {
	root := filepath.Join("..", "..", "compile_sealed_rbac")
	schemaRoot := os.Getenv("PTSIP_PP300_SCHEMA_ROOT")
	if schemaRoot == "" {
		schemaRoot = root
	}
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

	// Compile each schema with only its explicitly selected local resource.
	// This checks structural parity without registering or activating a contract.
	checks := make(map[string]func(any) error)
	for _, role := range []string{"ra", "rac", "aec"} {
		name := role + ".pp.3.00.schema.json"
		id := "urn:ptsip:rbac:consumer:pp300:" + role + ":structure:v1"
		doc, ok := read(filepath.Join(schemaRoot, name)).(map[string]any)
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
		checks[name] = compiled.Validate
	}

	cases, ok := read(filepath.Join(root, "testdata", "conformance.json")).([]any)
	if !ok || len(cases) == 0 {
		t.Fatal("conformance fixtures must be a nonempty array")
	}
	for _, value := range cases {
		fixture, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("invalid conformance fixture: %#v", value)
		}
		name, _ := fixture["name"].(string)
		file, _ := fixture["schema"].(string)
		validate := checks[file]
		want, hasExpected := fixture["valid"].(bool)
		instance, hasInstance := fixture["instance"]
		if name == "" || validate == nil || !hasExpected || !hasInstance {
			t.Fatalf("invalid conformance fixture: %#v", fixture)
		}
		t.Run(name, func(t *testing.T) {
			err := validate(instance)
			if valid := err == nil; valid != want {
				t.Errorf("%s validity = %t, want %t (error: %v)", file, valid, want, err)
			}
		})
	}
}
