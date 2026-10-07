package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

type Object = map[string]any

// A Repository instance is an operation-local read snapshot, never an authority cache.
type Repository struct {
	Root     string
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
}

func Open(start string) (*Repository, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		root = filepath.Dir(root)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
			real, err := filepath.EvalSymlinks(root)
			if err != nil {
				return nil, err
			}
			return &Repository{Root: real}, nil
		}
		next := filepath.Dir(root)
		if next == root {
			return nil, fmt.Errorf("unable to locate PTSIP repository root")
		}
		root = next
	}
}

// Path resolves existing symlinks, including those in the parent of a missing file.
func (r *Repository) Path(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("empty repository path")
	}
	input = strings.ReplaceAll(input, "\\", "/")
	candidate := filepath.FromSlash(input)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(r.Root, candidate)
	}
	candidate = filepath.Clean(candidate)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		tail := []string{}
		parent := candidate
		for {
			resolved, err = filepath.EvalSymlinks(parent)
			if err == nil {
				break
			}
			if !os.IsNotExist(err) {
				return "", err
			}
			next := filepath.Dir(parent)
			if next == parent {
				return "", fmt.Errorf("unresolvable path %q", input)
			}
			tail = append(tail, filepath.Base(parent))
			parent = next
		}
		for i := len(tail) - 1; i >= 0; i-- {
			resolved = filepath.Join(resolved, tail[i])
		}
	}
	relative, err := filepath.Rel(r.Root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository: %q", input)
	}
	return resolved, nil
}

func (r *Repository) Scope(input string) (string, error) {
	path, err := r.Path(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(r.Root, path)
	return filepath.ToSlash(relative), err
}

func (r *Repository) Read(input string) (Object, error) {
	path, err := r.Path(input)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	if strings.HasSuffix(path, ".json") {
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.UseNumber()
		value, err = decodeJSONValue(decoder)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", input, err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, fmt.Errorf("%s: multiple JSON documents", input)
		}
	} else {
		decoder := yaml.NewDecoder(bytes.NewReader(content))
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("%s: %w", input, err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, fmt.Errorf("%s: multiple YAML documents", input)
		}
	}
	value, err = normalize(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", input, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected mapping", input)
	}
	return object, nil
}

func normalize(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			v[key] = normalized
		}
	case map[any]any:
		out := Object{}
		for key, child := range v {
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("non-string mapping key")
			}
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			out[name] = normalized
		}
		return out, nil
	case []any:
		for i, child := range v {
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			v[i] = normalized
		}
	}
	return value, nil
}

type closedLoader struct{}

func (closedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("unregistered schema resource: %s", url)
}

func (r *Repository) Compiler() (*jsonschema.Compiler, error) {
	if r.compiler != nil {
		return r.compiler, nil
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseRegexpEngine(compileSchemaRegexp)
	compiler.UseLoader(closedLoader{})
	contracts, err := r.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		return nil, err
	}
	if err := compiler.AddResource(Text(contracts["$id"]), contracts); err != nil {
		return nil, err
	}
	meta, err := r.Read("developer/policy/schemas/developer-policy-catalog-contracts.schema.json")
	if err != nil {
		return nil, err
	}
	if err := compiler.AddResource(Text(meta["$id"]), meta); err != nil {
		return nil, err
	}
	admitted, err := compiler.Compile(Text(meta["$id"]))
	if err != nil {
		return nil, err
	}
	if err := admitted.Validate(contracts); err != nil {
		return nil, err
	}
	for _, raw := range Map(contracts["contracts"]) {
		schema, err := r.Read(Text(Map(raw)["schema_ref"]))
		if err != nil {
			return nil, err
		}
		if err := compiler.AddResource(Text(schema["$id"]), schema); err != nil {
			return nil, err
		}
	}
	r.compiler = compiler
	r.schemas = map[string]*jsonschema.Schema{}
	return compiler, nil
}

func (r *Repository) Validate(schemaPath string, value any) error {
	compiler, err := r.Compiler()
	if err != nil {
		return err
	}
	if compiled := r.schemas[schemaPath]; compiled != nil {
		return compiled.Validate(value)
	}
	schema, err := r.Read(schemaPath)
	if err != nil {
		return err
	}
	identity := Text(schema["$id"])
	if identity == "" {
		// Registered schemas without $id need distinct compiler bases. The derived
		// label remains an implementation detail and does not alter source identity.
		identity = "urn:ptsip:registered-schema-path:" + SHA256([]byte(schemaPath))
	}
	// The shared catalog schemas were already admitted by Compiler.
	if schemaPath != "developer/policy/schemas/developer-policy-catalog.schema.json" && schemaPath != "developer/policy/schemas/developer-policy-subject-catalog.schema.json" {
		if err := compiler.AddResource(identity, schema); err != nil {
			return err
		}
	}
	compiled, err := compiler.Compile(identity)
	if err != nil {
		return err
	}
	r.schemas[schemaPath] = compiled
	return compiled.Validate(value)
}

// ValidateDefinition selects only the exact definition named by a canonical
// schema registry. It never derives a schema from a policy's implementation.
func (r *Repository) ValidateDefinition(schemaPath, definition string, value any) error {
	schema, err := r.Read(schemaPath)
	if err != nil {
		return err
	}
	if Map(Map(schema["$defs"])[definition]) == nil {
		return fmt.Errorf("REGISTERED_SCHEMA_DEFINITION_MISSING: %s", definition)
	}
	compiler, err := r.Compiler()
	if err != nil {
		return err
	}
	identity := Text(schema["$id"])
	if identity == "" {
		identity = "urn:ptsip:registered-schema-path:" + SHA256([]byte(schemaPath))
	}
	key := schemaPath + "#/$defs/" + definition
	if compiled := r.schemas[key]; compiled != nil {
		return compiled.Validate(value)
	}
	if r.schemas[schemaPath] == nil {
		if err := compiler.AddResource(identity, schema); err != nil {
			return err
		}
		base, err := compiler.Compile(identity)
		if err != nil {
			return err
		}
		r.schemas[schemaPath] = base
	}
	compiled, err := compiler.Compile(identity + "#/$defs/" + definition)
	if err != nil {
		return err
	}
	r.schemas[key] = compiled
	return compiled.Validate(value)
}

func Map(value any) Object  { object, _ := value.(map[string]any); return object }
func Text(value any) string { text, _ := value.(string); return text }
func List(value any) []any  { list, _ := value.([]any); return list }
