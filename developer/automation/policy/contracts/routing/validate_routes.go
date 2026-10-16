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

const identitySchemaPath = "developer/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json"
const routingSchemaPath = "developer/contracts/schemas/Policy_Identity/Routing/Policy_Routing.schema.json"

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
func validate(root, schemaPath string, value any, code string) error {
	schema, err := load(root, schemaPath, code, false)
	if err != nil {
		return err
	}
	id := asString(schema["$id"])
	if id == "" {
		return invalid(code, schemaPath+": missing $id")
	}
	c := jsonschema.NewCompiler()
	c.UseRegexpEngine(compilePattern)
	if err := c.AddResource(id, schema); err != nil {
		return invalid(code, err.Error())
	}
	compiled, err := c.Compile(id)
	if err != nil {
		return invalid(code, err.Error())
	}
	if err := compiled.Validate(value); err != nil {
		return invalid(code, err.Error())
	}
	return nil
}

func loadContracts(root string) (map[string]any, map[string]bool, error) {
	identitySchema, err := load(root, identitySchemaPath, "IDENTITY_SCHEMA_INVALID", false)
	if err != nil {
		return nil, nil, err
	}
	binding := asObject(identitySchema["x-ptsip-contract-binding"])
	routePath := asString(binding["routing"])
	familyPath := asString(binding["root_family"])
	if routePath == "" || familyPath == "" {
		return nil, nil, invalid("CONTRACT_BINDING_INVALID", "Policy_Identity must declare routing and root_family contracts")
	}
	routes, err := load(root, routePath, "ROUTING_CONTRACT_INVALID", false)
	if err != nil {
		return nil, nil, err
	}
	if err := validate(root, routingSchemaPath, routes, "ROUTING_CONTRACT_INVALID"); err != nil {
		return nil, nil, err
	}
	fromRoutes := asString(asObject(routes["identity_sources"])["root_family"])
	if fromRoutes != familyPath {
		return nil, nil, invalid("CONTRACT_BINDING_INVALID", "Policy_Identity and routing disagree on root_family source")
	}
	familyDocument, err := load(root, familyPath, "IDENTITY_SOURCE_INVALID", false)
	if err != nil {
		return nil, nil, err
	}
	if asString(familyDocument["schema_version"]) != "ptsip-root-family/v1" ||
		len(familyDocument) != 2 {
		return nil, nil, invalid("IDENTITY_SOURCE_INVALID", "unexpected Root Family source contract")
	}
	rawFamilies, ok := familyDocument["root_families"].([]any)
	if !ok || len(rawFamilies) != 14 {
		return nil, nil, invalid("IDENTITY_SOURCE_INVALID", "expected 14 Root Family entries")
	}
	families := map[string]bool{}
	for _, value := range rawFamilies {
		name := asString(value)
		if name == "" || strings.ToUpper(name) != name || families[name] {
			return nil, nil, invalid("IDENTITY_SOURCE_INVALID", "invalid or duplicate Root Family entry")
		}
		families[name] = true
	}
	routeKinds := asObject(routes["routes"])
	developer := asObject(routeKinds["PTSIP_DEVELOPER_POLICY"])
	if len(developer) != len(families) {
		return nil, nil, invalid("ROUTING_COVERAGE_INCOMPLETE", "Developer routes must cover all 14 Root Families")
	}
	for name := range families {
		if _, ok := developer[name]; !ok {
			return nil, nil, invalid("ROUTING_COVERAGE_INCOMPLETE", "missing Developer Family "+name)
		}
	}
	for kind, raw := range routeKinds {
		for name := range asObject(raw) {
			if !families[name] {
				return nil, nil, invalid("INVALID_ROOT_FAMILY", kind+"/"+name)
			}
		}
	}
	return routes, families, nil
}

func inspectReferences(root string, selected map[string]any) error {
	refs := asObject(selected["schema_refs"])
	for _, field := range []string{"public", "authority", "rules"} {
		path := asString(refs[field])
		document, err := load(root, path, "SCHEMA_REF_UNRESOLVED", false)
		if err != nil {
			return err
		}
		if asString(document["$id"]) == "" {
			return invalid("SCHEMA_REF_UNRESOLVED", field+": missing schema $id: "+path)
		}
		// Compile verifies schema syntax; full recursive references are a separate boundary.
		if err := validate(root, path, map[string]any{}, "SCHEMA_REF_UNRESOLVED"); err != nil {
			// A valid schema need not accept the empty instance.
			if strings.Contains(err.Error(), "valid against") {
				continue
			}
			return err
		}
	}
	return nil
}

func execute(root, identityFile, kind, family string, checkFiles bool) (map[string]any, error) {
	routes, families, err := loadContracts(root)
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
