package validator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

func TestTPBGF3StructuralConformance(t *testing.T) {
	discovery := filepath.Join("..", "..", "engines", "discovery")
	readObject := func(t *testing.T, path string) map[string]any {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		value, err := jsondoc.Decode(raw)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("%s must contain a JSON object", path)
		}
		return object
	}

	doc := readObject(t, filepath.Join(discovery, "tpbg-f3.schema.json"))
	id, ok := doc["$id"].(string)
	if !ok || id == "" || doc["$schema"] != validator.Draft {
		t.Fatal("TPBG F3 must declare a schema identity and Draft 2020-12")
	}
	schema, err := validator.Compile(resolver.Graph{
		Root: id,
		Docs: map[string]map[string]any{id: doc},
	})
	if err != nil {
		t.Fatalf("compile TPBG F3: %v", err)
	}

	fixture := filepath.Join(discovery, "tests", "fixtures", "tpbg-f3.json")
	output := readObject(t, fixture)
	if output["model"] != "TPBG F3" {
		t.Fatalf("fixture model = %v, want TPBG F3", output["model"])
	}
	if err := schema.Validate(output); err != nil {
		t.Fatalf("validate shared Discovery output: %v", err)
	}

	tests := []struct {
		name   string
		change func(*testing.T, map[string]any)
	}{
		{
			name: "missing evidence area",
			change: func(_ *testing.T, output map[string]any) {
				delete(output, "evidence")
			},
		},
		{
			name: "wrong reference pair length",
			change: func(t *testing.T, output map[string]any) {
				references, ok := output["references"].([]any)
				if !ok || len(references) == 0 {
					t.Fatal("fixture must contain a reference row")
				}
				reference, ok := references[0].(map[string]any)
				if !ok {
					t.Fatal("fixture reference row must be an object")
				}
				pair, ok := reference["pair"].([]any)
				if !ok || len(pair) != 2 {
					t.Fatal("fixture reference pair must contain two items")
				}
				reference["pair"] = pair[:1]
			},
		},
		{
			name: "null evidence on resolved edge",
			change: func(t *testing.T, output map[string]any) {
				graph, ok := output["graph"].(map[string]any)
				if !ok {
					t.Fatal("fixture graph must be an object")
				}
				edges, ok := graph["edges"].([]any)
				if !ok {
					t.Fatal("fixture graph edges must be an array")
				}
				for _, value := range edges {
					edge, ok := value.(map[string]any)
					if ok && edge["kind"] == "resolves_to" {
						edge["evidence"] = nil
						return
					}
				}
				t.Fatal("fixture must contain a resolves_to edge")
			},
		},
		{
			name: "execution authorization",
			change: func(_ *testing.T, output map[string]any) {
				output["execution_authorized"] = true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := readObject(t, fixture)
			tt.change(t, output)
			if err := schema.Validate(output); err == nil {
				t.Error("schema accepted altered Discovery output")
			}
		})
	}
}
