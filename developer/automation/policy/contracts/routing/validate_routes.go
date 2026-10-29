// Independent C-mode contract reader. Does not activate legacy policy lifecycle.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

const identitySchemaPath = ".ptsip/contracts/schemas/Policy/Policy_Identity/PTSIP_Policy_Identity.schema.json"
const routingSchemaPath = ".ptsip/contracts/policy/routing/Policy_Routing.schema.json"
const routingDataPath = ".ptsip/contracts/policy/routing/policy_routes.json"
const policyKindSchemaID = "urn:ptsip:identity:policy-kind:v1"
const rootFamilySchemaID = "urn:ptsip:identity:root-family:v1"

type contractError struct {
	Code   string
	Detail string
}

func (e *contractError) Error() string { return e.Code + ": " + e.Detail }
func invalid(code, message string) error {
	return &contractError{Code: code, Detail: message}
}
func asObject(v any) map[string]any {
	result, _ := v.(map[string]any)
	return result
}
func asString(v any) string {
	result, _ := v.(string)
	return result
}

func repositoryFile(root, name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || strings.Contains(name, ":") ||
		strings.HasPrefix(name, "/") || strings.Contains(name, "//") {
		return "", fmt.Errorf("invalid repository path %q", name)
	}
	for _, token := range strings.Split(name, "/") {
		if token == "." || token == ".." || token == "" {
			return "", fmt.Errorf("invalid repository path %q", name)
		}
	}
	location, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, location)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("reference outside repository: %s", name)
	}
	return location, nil
}

func load(root, name, code string, allowYAML bool) (map[string]any, error) {
	location, err := repositoryFile(root, name)
	if err != nil {
		return nil, invalid(code, fmt.Sprintf("%s: %v", name, err))
	}
	raw, err := os.ReadFile(location)
	if err != nil {
		return nil, invalid(code, fmt.Sprintf("%s: %v", name, err))
	}
	var document map[string]any
	if allowYAML && (strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) {
		err = yaml.Unmarshal(raw, &document)
	} else {
		err = json.Unmarshal(raw, &document)
	}
	if err != nil || document == nil {
		return nil, invalid(code, fmt.Sprintf("invalid document %s: %v", name, err))
	}
	return document, nil
}

type schemaRegexp struct{ expression *regexp2.Regexp }
func (r *schemaRegexp) String() string { return r.expression.String() }
func (r *schemaRegexp) MatchString(s string) bool {
	matched, err := r.expression.MatchString(s)
	return err == nil && matched
}
func compilePattern(pattern string) (jsonschema.Regexp, error) {
	expr, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	expr.MatchTimeout = 5 * time.Second
	return &schemaRegexp{expression: expr}, nil
}
type closedLoader struct{}

func (closedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("unregistered schema resource: %s", url)
}

func compileSchema(root, schemaPath, code string) (*jsonschema.Schema, error) {
	schema, err := load(root, schemaPath, code, false)
	if err != nil {
		return nil, err
	}
	id := asString(schema["$id"])
	if id == "" {
		return nil, invalid(code, schemaPath+": missing $id")
	}

	// The routing document provides explicit paths for local schema resources.
	// JSON Schema references are resolved by $id, not by custom x-* keywords.
	routes, err := load(root, routingDataPath, "ROUTING_CONTRACT_INVALID", false)
	if err != nil {
		return nil, err
	}
	sources := asObject(routes["identity_sources"])
	resources := []struct {
		key string
		id  string
	}{
		{"policy_kind", policyKindSchemaID},
		{"root_family", rootFamilySchemaID},
	}
	c := jsonschema.NewCompiler()
	c.UseRegexpEngine(compilePattern)
	c.UseLoader(closedLoader{})
	for _, resource := range resources {
		path := asString(sources[resource.key])
		if path == "" {
			return nil, invalid("CONTRACT_BINDING_INVALID", "missing "+resource.key+" schema source")
		}
		document, err := load(root, path, "IDENTITY_SOURCE_INVALID", false)
		if err != nil {
			return nil, err
		}
		if asString(document["$id"]) != resource.id {
			return nil, invalid("IDENTITY_SOURCE_INVALID", "wrong $id in "+path)
		}
		if err := c.AddResource(resource.id, document); err != nil {
			return nil, invalid("IDENTITY_SOURCE_INVALID", err.Error())
		}
	}
	if err := c.AddResource(id, schema); err != nil {
		return nil, invalid(code, err.Error())
	}
	compiled, err := c.Compile(id)
	if err != nil {
		return nil, invalid(code, err.Error())
	}
	return compiled, nil
}

func validate(root, schemaPath string, value any, code string) error {
	compiled, err := compileSchema(root, schemaPath, code)
	if err != nil {
		return err
	}
	if err := compiled.Validate(value); err != nil {
		return invalid(code, err.Error())
	}
	return nil
}

func schemaVocabulary(root, path, expectedID string) (map[string]bool, error) {
	document, err := load(root, path, "IDENTITY_SOURCE_INVALID", false)
	if err != nil {
		return nil, err
	}
	if asString(document["$id"]) != expectedID ||
		asString(document["$schema"]) != "https://json-schema.org/draft/2020-12/schema" ||
		asString(document["type"]) != "string" {
		return nil, invalid("IDENTITY_SOURCE_INVALID", "invalid vocabulary schema "+path)
	}
	raw, ok := document["enum"].([]any)
	if !ok || len(raw) == 0 {
		return nil, invalid("IDENTITY_SOURCE_INVALID", "empty vocabulary "+path)
	}
	allowed := make(map[string]bool, len(raw))
	for _, entry := range raw {
		name := asString(entry)
		if name == "" || allowed[name] {
			return nil, invalid("IDENTITY_SOURCE_INVALID", "invalid or duplicated value in "+path)
		}
		allowed[name] = true
	}
	return allowed, nil
}

func loadContracts(root string) (map[string]any, map[string]bool, map[string]bool, error) {
	routes, err := load(root, routingDataPath, "ROUTING_CONTRACT_INVALID", false)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := validate(root, routingSchemaPath, routes, "ROUTING_CONTRACT_INVALID"); err != nil {
		return nil, nil, nil, err
	}
	sources := asObject(routes["identity_sources"])
	kinds, err := schemaVocabulary(root, asString(sources["policy_kind"]), policyKindSchemaID)
	if err != nil {
		return nil, nil, nil, err
	}
	families, err := schemaVocabulary(root, asString(sources["root_family"]), rootFamilySchemaID)
	if err != nil {
		return nil, nil, nil, err
	}
	routeKinds := asObject(routes["routes"])
	developer := asObject(routeKinds["PTSIP_DEVELOPER_POLICY"])
	if len(developer) != len(families) {
		return nil, nil, nil, invalid("ROUTING_COVERAGE_INCOMPLETE", "Developer routes must cover every Root Family")
	}
	for name := range families {
		if _, exists := developer[name]; !exists {
			return nil, nil, nil, invalid("ROUTING_COVERAGE_INCOMPLETE", "missing Developer Family "+name)
		}
	}
	for kind, raw := range routeKinds {
		if !kinds[kind] {
			return nil, nil, nil, invalid("INVALID_POLICY_KIND", kind)
		}
		for family := range asObject(raw) {
			if !families[family] {
				return nil, nil, nil, invalid("INVALID_ROOT_FAMILY", kind+"/"+family)
			}
		}
	}
	return routes, kinds, families, nil
}

func inspectReferences(root string, selected map[string]any) error {
	refs := asObject(selected["schema_refs"])
	for _, field := range []string{"public", "rules"} {
		path := asString(refs[field])
		// A referenced schema must exist and compile, but it need not accept
		// an empty policy instance. No recursive discovery or activation here.
		if _, err := compileSchema(root, path, "SCHEMA_REF_UNRESOLVED"); err != nil {
			return invalid("SCHEMA_REF_UNRESOLVED", field+": "+err.Error())
		}
	}
	return nil
}

func execute(root, identityFile, kind, family string, checkFiles bool) (map[string]any, error) {
	routes, kinds, families, err := loadContracts(root)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"status": "ROUTING_STRUCTURE_VALID",
		"registered_routes": len(asObject(asObject(routes["routes"])["PTSIP_DEVELOPER_POLICY"])),
		"policy_creation_authorized": false,
	}
	if identityFile != "" {
		record, err := load(root, identityFile, "POLICY_IDENTITY_INVALID", true)
		if err != nil {
			return nil, err
		}
		identity := asObject(record["Policy_Identity"])
		if identity == nil {
			identity = record
		}
		if err := validate(root, identitySchemaPath, identity, "POLICY_IDENTITY_INVALID"); err != nil {
			return nil, err
		}
		currentKind, currentFamily := asString(identity["policy_kind"]), asString(identity["root_family"])
		if !kinds[currentKind] {
			return nil, invalid("INVALID_POLICY_KIND", currentKind)
		}
		if !families[currentFamily] {
			return nil, invalid("INVALID_ROOT_FAMILY", currentFamily)
		}
		if (kind != "" && kind != currentKind) || (family != "" && family != currentFamily) {
			return nil, invalid("POLICY_IDENTITY_INVALID", "identity disagrees with route selector")
		}
		kind, family = currentKind, currentFamily
		result["identity_validation"] = "VALID"
	}
	if (kind == "") != (family == "") {
		return nil, invalid("ROUTE_UNREGISTERED", "both policy-kind and root-family are required")
	}
	if kind != "" {
		if !kinds[kind] {
			return nil, invalid("INVALID_POLICY_KIND", kind)
		}
		if !families[family] {
			return nil, invalid("INVALID_ROOT_FAMILY", family)
		}
		selected := asObject(asObject(asObject(routes["routes"])[kind])[family])
		if selected == nil {
			return nil, invalid("ROUTE_UNREGISTERED", kind+"/"+family)
		}
		if checkFiles {
			if err := inspectReferences(root, selected); err != nil {
				return nil, err
			}
		}
		result["status"] = "ROUTE_REGISTERED"
		result["route"] = map[string]any{"policy_kind": kind, "root_family": family, "output": selected["output"]}
		result["schema_file_check"] = "NOT_REQUESTED"
		if checkFiles {
			result["schema_file_check"] = "PASSED_NON_RECURSIVE"
		}
	} else if checkFiles {
		for _, raw := range asObject(routes["routes"]) {
			for _, value := range asObject(raw) {
				if err := inspectReferences(root, asObject(value)); err != nil {
					return nil, err
				}
			}
		}
		result["schema_file_check"] = "PASSED_NON_RECURSIVE"
	}
	return result, nil
}

func main() {
	repository := flag.String("repository", ".", "Repository root")
	identity := flag.String("identity", "", "Repository-relative JSON/YAML file containing Policy_Identity")
	kind := flag.String("policy-kind", "", "Policy kind")
	family := flag.String("root-family", "", "Root Family")
	checkFiles := flag.Bool("check-schema-files", false, "Require referenced schema files to resolve")
	flag.Parse()
	root, err := filepath.Abs(*repository)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := execute(root, *identity, *kind, *family, *checkFiles)
	if err != nil {
		code := "ROUTING_VERIFICATION_FAILED"
		if typed, ok := err.(*contractError); ok {
			code = typed.Code
		}
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"status": "FAILED", "code": code, "detail": err.Error(), "policy_creation_authorized": false})
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(result)
}
