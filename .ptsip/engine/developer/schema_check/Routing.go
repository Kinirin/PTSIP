// Package schema_check verifies the live policy-routing contract through the
// installed PTSIP VALIDATION engine. A successful routing lookup never grants
// policy approval, execution permission, or full policy conformance.
package schema_check

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/contract_authoring"
)

const policyRoutingData = ".ptsip/contracts/policy/routing/policy_routes.json"
const policyRoutingSchema = ".ptsip/contracts/policy/routing/Policy_Routing.schema.json"
const policyKindVocabulary = ".ptsip/contracts/policy/identity/policy_kind.json"
const rootFamilyVocabulary = ".ptsip/contracts/policy/identity/root_family.json"

type RoutingProbe struct {
	Status            string   `json:"status"`
	RoutingPath       string   `json:"routing_path"`
	SchemaPath        string   `json:"schema_path"`
	RegisteredRoutes  int      `json:"registered_routes"`
	SchemaReferences  int      `json:"schema_references_checked"`
	IdentitySources   []string `json:"identity_sources"`
	PolicyConformance string   `json:"policy_conformance"`
	Problem           string   `json:"problem,omitempty"`
}

// routingFile accepts only repository-relative files below the explicitly
// selected contract area. It never searches retired registries or the network.
func routingFile(root, relative, prefix string) (map[string]any, error) {
	if relative == "" || !strings.HasPrefix(relative, prefix) ||
		strings.Contains(relative, "\\") || strings.Contains(relative, "//") ||
		path.Clean(relative) != relative || strings.Contains(relative, ":") {
		return nil, fmt.Errorf("INVALID_ROUTING_CONTRACT_PATH: %s", relative)
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "." || part == ".." || part == "" {
			return nil, fmt.Errorf("INVALID_ROUTING_CONTRACT_PATH: %s", relative)
		}
	}
	filename := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, fmt.Errorf("ROUTING_RESOURCE_MISSING: %s: %w", relative, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("ROUTING_RESOURCE_NOT_REGULAR: %s", relative)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return nil, fmt.Errorf("INVALID_ROUTING_JSON: %s: %v", relative, err)
	}
	return obj, nil
}

func routingMembers(v any) map[string]any {
	obj, _ := v.(map[string]any)
	return obj
}
func routingString(v any) string {
	s, _ := v.(string)
	return s
}

func loadRoutingInputs(root string) (map[string]any, map[string]any, []any, int, int, []string, error) {
	boundary, err := routingFile(root, ".ptsip/contracts/policy/boundary.json", ".ptsip/contracts/policy/")
	if err != nil {
		return nil, nil, nil, 0, 0, nil, err
	}
	if routingString(boundary["domain"]) != "policy" ||
		routingString(routingMembers(boundary["namespaces"])["routing"]) != "routing/" {
		return nil, nil, nil, 0, 0, nil, fmt.Errorf("POLICY_ROUTING_BOUNDARY_INVALID")
	}
	binding := routingMembers(boundary["routing_binding"])
	dataPath, schemaPath := routingString(binding["data"]), routingString(binding["schema"])
	routing, err := routingFile(root, dataPath, ".ptsip/contracts/policy/routing/")
	if err != nil {
		return nil, nil, nil, 0, 0, nil, err
	}
	schema, err := routingFile(root, schemaPath, ".ptsip/contracts/policy/routing/")
	if err != nil {
		return nil, nil, nil, 0, 0, nil, err
	}
	if routingString(schema["$id"]) != "urn:ptsip:contract:policy-routing:v4" {
		return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTING_SCHEMA_ID_MISMATCH")
	}
	sources := routingMembers(routing["identity_sources"])
	expected := []struct{ key, file, id string }{
		{"policy_kind", policyKindVocabulary, "urn:ptsip:identity:policy-kind:v1"},
		{"root_family", rootFamilyVocabulary, "urn:ptsip:identity:root-family:v1"},
	}
	resources := make([]any, 0, len(expected))
	identities := make([]string, 0, len(expected))
	vocabulary := map[string]map[string]bool{}
	for _, item := range expected {
		if routingString(sources[item.key]) != item.file {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTING_IDENTITY_SOURCE_MISMATCH: %s", item.key)
		}
		doc, err := routingFile(root, item.file, ".ptsip/contracts/policy/identity/")
		if err != nil {
			return nil, nil, nil, 0, 0, nil, err
		}
		if routingString(doc["$id"]) != item.id {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("IDENTITY_SCHEMA_ID_MISMATCH: %s", item.key)
		}
		values, ok := doc["enum"].([]any)
		if !ok || len(values) == 0 {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("EMPTY_IDENTITY_VOCABULARY: %s", item.key)
		}
		names := map[string]bool{}
		for _, value := range values {
			name := routingString(value)
			if name == "" || names[name] {
				return nil, nil, nil, 0, 0, nil, fmt.Errorf("DUPLICATE_IDENTITY_VALUE: %s", item.key)
			}
			names[name] = true
		}
		vocabulary[item.key] = names
		resources = append(resources, doc)
		identities = append(identities, item.file)
	}
	kinds := routingMembers(routing["routes"])
	if len(kinds) == 0 {
		return nil, nil, nil, 0, 0, nil, fmt.Errorf("EMPTY_POLICY_ROUTES")
	}
	references := 0
	// Schema paths are declared once. Root Family selection is owned by the
	// rules contracts, rather than a second map of fourteen schema filenames.
	schemaRefs := routingMembers(routing["schema_refs"])
	expectedSchemas := map[string]string{
		"public": ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json",
		"rules":  contract_authoring.CommonSchema,
	}
	for key, expected := range expectedSchemas {
		if routingString(schemaRefs[key]) != expected {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("COMMON_SCHEMA_BINDING_MISMATCH: %s", key)
		}
		doc, err := routingFile(root, expected, ".ptsip/contracts/schemas/")
		if err != nil {
			return nil, nil, nil, 0, 0, nil, err
		}
		if routingString(doc["$id"]) == "" {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTE_SCHEMA_ID_MISSING: %s", key)
		}
		references++
	}
	for kind, entries := range kinds {
		if !vocabulary["policy_kind"][kind] {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("UNKNOWN_POLICY_KIND: %s", kind)
		}
		output := routingString(routingMembers(entries)["output_root"])
		expected := "developer/policy"
		if kind == "PTSIP_SUPPORT_FEATURE" {
			expected = "src/policy"
		}
		if output != expected {
			return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTE_DIRECTORY_MISMATCH: %s", kind)
		}
	}
	if kinds["PTSIP_DEVELOPER_POLICY"] == nil {
		return nil, nil, nil, 0, 0, nil, fmt.Errorf("DEVELOPER_OUTPUT_ROUTE_MISSING")
	}
	catalog, err := contract_authoring.Load(root)
	if err != nil {
		return nil, nil, nil, 0, 0, nil, err
	}
	registered := len(catalog.Families) * len(kinds)
	return routing, schema, resources, registered, references, identities, nil
}

func probePolicyRouting(root string, e Engine) RoutingProbe {
	p := RoutingProbe{
		Status: "UNRESOLVED", RoutingPath: policyRoutingData,
		SchemaPath: policyRoutingSchema, PolicyConformance: "NOT_ASSERTED",
	}
	routing, schema, resources, routes, refs, identities, err := loadRoutingInputs(root)
	if err != nil {
		p.Problem = err.Error()
		return p
	}
	p.RegisteredRoutes = routes
	p.SchemaReferences = refs
	p.IdentitySources = identities
	valid, err := nativeValidate(e, schema, routing, resources)
	if err != nil {
		p.Problem = err.Error()
		return p
	}
	if !valid {
		p.Problem = "ROUTING_CONTRACT_SCHEMA_REJECTED"
		return p
	}
	catalog, err := contract_authoring.Load(root)
	if err != nil {
		p.Problem = err.Error()
		return p
	}
	if err := catalog.CheckContracts(func(schema, instance any, resources []any) (bool, error) {
		return nativeValidate(e, schema, instance, resources)
	}); err != nil {
		p.Problem = err.Error()
		return p
	}
	p.Status = "PASS"
	return p
}
