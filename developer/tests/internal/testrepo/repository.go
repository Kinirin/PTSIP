package testrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

type Object = map[string]any

type Repository struct {
	Root     string
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
}

type closedLoader struct{}

func (closedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("unregistered test schema resource: %s", url)
}

func Root(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve repository test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "developer", "automation", "go.mod")); err != nil {
		t.Fatalf("repository root: %v", err)
	}
	return root
}

func Open(root string) *Repository {
	return &Repository{Root: root, schemas: map[string]*jsonschema.Schema{}}
}

func (r *Repository) RootDir() string { return r.Root }

func (r *Repository) Scope(input string) (string, error) {
	if !filepath.IsAbs(input) {
		input = filepath.Join(r.Root, input)
	}
	relative, err := filepath.Rel(r.Root, filepath.Clean(input))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("repository path escapes root: %s", input)
	}
	return filepath.ToSlash(relative), nil
}

func (r *Repository) Path(input string) (string, error) {
	scope, err := r.Scope(input)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.Root, filepath.FromSlash(scope)), nil
}

func (r *Repository) Read(input string) (Object, error) {
	path, err := r.Path(input)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value Object
	if err := yaml.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("expected mapping: %s", input)
	}
	return value, nil
}

func (r *Repository) Validate(input string, value any) error {
	if r.compiler == nil {
		r.compiler = jsonschema.NewCompiler()
		r.compiler.UseLoader(closedLoader{})
		contracts, err := r.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
		if err != nil {
			return err
		}
		meta, err := r.Read("developer/policy/schemas/developer-policy-catalog-contracts.schema.json")
		if err != nil {
			return err
		}
		resources := []Object{contracts, meta}
		for _, raw := range contracts["contracts"].(map[string]any) {
			schema, err := r.Read(raw.(map[string]any)["schema_ref"].(string))
			if err != nil {
				return err
			}
			resources = append(resources, schema)
		}
		for _, schema := range resources {
			if err := r.compiler.AddResource(schema["$id"].(string), schema); err != nil {
				return err
			}
		}
	}
	schema := r.schemas[input]
	if schema == nil {
		document, err := r.Read(input)
		if err != nil {
			return err
		}
		id := document["$id"].(string)
		if input != "developer/policy/schemas/developer-policy-catalog.schema.json" && input != "developer/policy/schemas/developer-policy-subject-catalog.schema.json" {
			if err := r.compiler.AddResource(id, document); err != nil {
				return err
			}
		}
		schema, err = r.compiler.Compile(id)
		if err != nil {
			return err
		}
		r.schemas[input] = schema
	}
	return schema.Validate(value)
}

func CopyFiles(t *testing.T, destination *Repository, refs ...string) {
	t.Helper()
	source := Open(Root(t))
	for _, ref := range refs {
		from, err := source.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		to, err := destination.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func CopyCatalogContracts(t *testing.T, destination *Repository) {
	t.Helper()
	CopyFiles(t, destination, "developer/policy/registries/developer-policy-catalog-contracts.json", "developer/policy/schemas/developer-policy-catalog-contracts.schema.json")
	contracts, err := destination.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range contracts["contracts"].(map[string]any) {
		CopyFiles(t, destination, raw.(map[string]any)["schema_ref"].(string))
	}
}

func Write(t *testing.T, r *Repository, ref string, value any) {
	t.Helper()
	path, err := r.Path(ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
}
