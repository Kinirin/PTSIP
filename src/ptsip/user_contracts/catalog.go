package usercontracts

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Validator struct {
	Root       string
	schemas    map[string]*jsonschema.Schema
	documents  map[string]any
	registered map[string]string
	policy     map[string]any
	official   *jsonschema.Schema
	metaschema *jsonschema.Schema
}

type schemaNode struct {
	Value   any
	Pointer string
	Base    string
}

type schemaDocument struct {
	URI     string
	Value   any
	Nodes   []schemaNode
	Aliases map[string]string
}

type offlineLoader struct{}

func (offlineLoader) Load(uri string) (any, error) {
	return nil, fmt.Errorf("schema reference is not registered locally: %s", uri)
}

func newCompiler() *jsonschema.Compiler {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(offlineLoader{})
	compiler.UseRegexpEngine(compileRegexp)
	return compiler
}

func New(root string) (*Validator, error) {
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	configValue, err := readJSON(filepath.Join(resolved, "schemas", "config.json"), false)
	if err != nil {
		return nil, err
	}
	config, ok := configValue.(map[string]any)
	if !ok {
		return nil, errors.New("schema catalog must be an object")
	}
	storage, ok := config["storage"].(map[string]any)
	if !ok {
		return nil, errors.New("schema catalog requires storage")
	}
	v := &Validator{
		Root: resolved, schemas: make(map[string]*jsonschema.Schema),
		documents: make(map[string]any), registered: make(map[string]string),
	}
	compiler := newCompiler()
	v.metaschema, err = compiler.Compile(Dialect)
	if err != nil {
		return nil, err
	}
	documents := make(map[string]schemaDocument)
	paths := make(map[string]bool)
	load := func(relative string) (schemaDocument, error) {
		if relative == "" || path.IsAbs(relative) || filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
			return schemaDocument{}, fmt.Errorf("invalid schema path: %s", relative)
		}
		for _, part := range strings.Split(relative, "/") {
			if part == ".." {
				return schemaDocument{}, fmt.Errorf("invalid schema path: %s", relative)
			}
		}
		if paths[relative] {
			return schemaDocument{}, fmt.Errorf("duplicate schema path: %s", relative)
		}
		paths[relative] = true
		filename := filepath.Join(resolved, "schemas", filepath.FromSlash(relative))
		value, err := readJSON(filename, false)
		if err != nil {
			return schemaDocument{}, err
		}
		if status, message := v.schemaProblem(value); status != "" {
			return schemaDocument{}, fmt.Errorf("%s schema %s: %s", status, relative, message)
		}
		document, err := describeSchema(fileURI(filename), value)
		if err != nil {
			return schemaDocument{}, err
		}
		object, ok := value.(map[string]any)
		if !ok {
			return schemaDocument{}, fmt.Errorf("catalog schema requires an $id: %s", relative)
		}
		id, ok := object["$id"].(string)
		if !ok || id == "" {
			return schemaDocument{}, fmt.Errorf("catalog schema requires an $id: %s", relative)
		}
		if _, duplicate := v.documents[id]; duplicate {
			return schemaDocument{}, fmt.Errorf("duplicate schema resource ID: %s", id)
		}
		v.documents[id] = value
		documents[document.URI] = document
		return document, nil
	}
	for _, raw := range []any{config["Policy"], config["plan"], storage["Responsibility-Authority"]} {
		group, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.New("schema registration group must be an object")
		}
		for _, key := range sortedKeys(group) {
			entry, ok := group[key].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid schema registration: %s", key)
			}
			if _, duplicate := v.registered[key]; duplicate {
				return nil, fmt.Errorf("duplicate schema registration: %s", key)
			}
			version, ok := entry["version"].(string)
			if !ok || version == "" {
				return nil, fmt.Errorf("missing schema registration version: %s", key)
			}
			relative, _ := entry["path"].(string)
			document, err := load(relative)
			if err != nil {
				return nil, err
			}
			object := document.Value.(map[string]any)
			if entry["$id"] != object["$id"] {
				return nil, fmt.Errorf("schema resource ID mismatch: %s", key)
			}
			v.registered[key] = document.URI
		}
	}
	for _, relative := range []string{
		"policy/identity/PTSIP_policy_identity.schema.json",
		"policy/lifecycle/PTSIP_policy_lifecycle.schema.json",
		"policy/rules/PTSIP_policy_rules.schema.json",
		"policy/content/PTSIP_policy_content.schema.json",
	} {
		if _, err := load(relative); err != nil {
			return nil, err
		}
	}
	compiled, err := compileDocuments(documents)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{PolicyKey, "RA_list", "RAC_list", "AEC_list"} {
		uri, ok := v.registered[key]
		if !ok {
			return nil, fmt.Errorf("missing required schema registration: %s", key)
		}
		v.schemas[key], err = compileClosure(compiled, documents[uri], documents)
		if err != nil {
			return nil, err
		}
	}
	v.policy, _ = documents[v.registered[PolicyKey]].Value.(map[string]any)
	properties, _ := v.policy["properties"].(map[string]any)
	identity, _ := properties["Policy_Identity"].(map[string]any)
	identityRef, _ := identity["$ref"].(string)
	v.official, err = compiled.Compile(identityRef + "#/$defs/officialVersion")
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Validator) StructureIssues(key string, instance any) []Issue {
	schema, ok := v.schemas[key]
	if !ok {
		return []Issue{{Code: "SCHEMA", Path: []any{}, Message: "Unknown schema registration: " + key}}
	}
	return validationIssues(schema, instance, "SCHEMA", nil)
}

func (v *Validator) OfficialVersionValid(value any) bool {
	return v.official.Validate(value) == nil
}

func readJSON(filename string, bom bool) (any, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if bom {
		data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	}
	return ParseJSON(data)
}

func fileURI(filename string) string {
	normalized := filepath.ToSlash(filename)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	return (&url.URL{Scheme: "file", Path: normalized}).String()
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (v *Validator) schemaProblem(value any) (string, string) {
	nodes := schemaNodes(value, "")
	for _, node := range nodes {
		if object, ok := node.Value.(map[string]any); ok {
			if raw, present := object["$schema"]; present && raw != Dialect && raw != Dialect+"#" {
				return "UNSUPPORTED", "Expected JSON Schema Draft 2020-12"
			}
		}
	}
	if err := v.metaschema.Validate(value); err != nil {
		return "INVALID", err.Error()
	}
	return "", ""
}

func schemaNodes(value any, pointer string) []schemaNode {
	nodes := []schemaNode{{Value: value, Pointer: pointer}}
	object, ok := value.(map[string]any)
	if !ok {
		return nodes
	}
	for _, keyword := range []string{
		"additionalProperties", "unevaluatedProperties", "propertyNames", "items", "contains",
		"unevaluatedItems", "additionalItems", "not", "if", "then", "else", "contentSchema",
	} {
		child, exists := object[keyword]
		if exists {
			nodes = append(nodes, schemaNodes(child, pointer+"/"+keyword)...)
		}
	}
	for _, keyword := range []string{"$defs", "definitions", "properties", "patternProperties", "dependentSchemas"} {
		children, _ := object[keyword].(map[string]any)
		for _, name := range sortedKeys(children) {
			nodes = append(nodes, schemaNodes(children[name], pointer+"/"+keyword+"/"+escapePointer(name))...)
		}
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		children, _ := object[keyword].([]any)
		for index, child := range children {
			nodes = append(nodes, schemaNodes(child, fmt.Sprintf("%s/%s/%d", pointer, keyword, index))...)
		}
	}
	return nodes
}

func escapePointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func describeSchema(uri string, value any) (schemaDocument, error) {
	document := schemaDocument{URI: uri, Value: value, Aliases: make(map[string]string)}
	bases := map[string]string{"": uri}
	for _, node := range schemaNodes(value, "") {
		parent := node.Pointer
		base := uri
		for parent != "" {
			parent = parent[:strings.LastIndex(parent, "/")]
			if candidate, exists := bases[parent]; exists {
				base = candidate
				break
			}
		}
		if object, ok := node.Value.(map[string]any); ok {
			if raw, present := object["$id"]; present {
				id, ok := raw.(string)
				if !ok {
					return document, errors.New("schema $id must be a string")
				}
				parsedBase, err := url.Parse(base)
				if err != nil {
					return document, err
				}
				reference, err := url.Parse(id)
				if err != nil {
					return document, err
				}
				base = parsedBase.ResolveReference(reference).String()
				if _, duplicate := document.Aliases[base]; duplicate {
					return document, fmt.Errorf("duplicate schema resource ID: %s", base)
				}
				document.Aliases[base] = uri + "#" + node.Pointer
			}
		}
		node.Base = base
		bases[node.Pointer] = base
		document.Nodes = append(document.Nodes, node)
	}
	return document, nil
}

func compileDocuments(documents map[string]schemaDocument) (*jsonschema.Compiler, error) {
	compiler := newCompiler()
	aliases := make(map[string]string)
	for _, uri := range sortedKeys(documents) {
		if err := compiler.AddResource(uri, documents[uri].Value); err != nil {
			return nil, err
		}
	}
	for _, uri := range sortedKeys(documents) {
		for _, alias := range sortedKeys(documents[uri].Aliases) {
			target := documents[uri].Aliases[alias]
			if existing, exists := aliases[alias]; exists && existing != target {
				return nil, fmt.Errorf("duplicate schema resource ID: %s", alias)
			}
			aliases[alias] = target
			if _, retrieval := documents[alias]; retrieval {
				if strings.TrimSuffix(target, "#") != alias {
					return nil, fmt.Errorf("schema ID conflicts with retrieval URI: %s", alias)
				}
				continue
			}
			var aliased map[string]any
			for _, node := range documents[uri].Nodes {
				if documents[uri].URI+"#"+node.Pointer == target {
					original, _ := node.Value.(map[string]any)
					aliased = make(map[string]any, len(original))
					for key, value := range original {
						aliased[key] = value
					}
					aliased["$id"] = alias
					break
				}
			}
			if err := compiler.AddResource(alias, aliased); err != nil {
				return nil, err
			}
		}
	}
	return compiler, nil
}

func compileClosure(compiler *jsonschema.Compiler, document schemaDocument, documents map[string]schemaDocument) (*jsonschema.Schema, error) {
	root, err := compiler.Compile(document.URI)
	if err != nil {
		return nil, err
	}
	owners := make(map[string]string)
	for uri, candidate := range documents {
		owners[uri] = uri
		for alias := range candidate.Aliases {
			owners[alias] = uri
		}
	}
	pending := []string{document.URI}
	visited := make(map[string]bool)
	for len(pending) > 0 {
		uri := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[uri] {
			continue
		}
		visited[uri] = true
		for _, node := range documents[uri].Nodes {
			if _, err := compiler.Compile(uri + "#" + node.Pointer); err != nil {
				return nil, err
			}
			object, _ := node.Value.(map[string]any)
			for _, keyword := range []string{"$ref", "$dynamicRef"} {
				reference, exists := object[keyword].(string)
				if !exists {
					continue
				}
				base, err := url.Parse(node.Base)
				if err != nil {
					return nil, err
				}
				target, err := url.Parse(reference)
				if err != nil {
					return nil, err
				}
				resolved := base.ResolveReference(target)
				resolved.Fragment = ""
				if owner, exists := owners[resolved.String()]; exists {
					pending = append(pending, owner)
				}
			}
		}
	}
	return root, nil
}
