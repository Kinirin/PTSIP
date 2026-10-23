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
)

const policyRoutingData = ".ptsip/contracts/policy/routing/policy_routes.json"
const policyRoutingSchema = ".ptsip/contracts/policy/routing/Policy_Routing.schema.json"
const policyKindVocabulary = ".ptsip/contracts/policy/identity/policy_kind.json"
const rootFamilyVocabulary = ".ptsip/contracts/policy/identity/root_family.json"

type RoutingProbe struct {
    Status string `json:"status"`
    RoutingPath string `json:"routing_path"`
    SchemaPath string `json:"schema_path"`
    RegisteredRoutes int `json:"registered_routes"`
    SchemaReferences int `json:"schema_references_checked"`
    IdentitySources []string `json:"identity_sources"`
    PolicyConformance string `json:"policy_conformance"`
    Problem string `json:"problem,omitempty"`
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
    if err != nil { return nil, fmt.Errorf("ROUTING_RESOURCE_MISSING: %s: %w", relative, err) }
    if !info.Mode().IsRegular() {
        return nil, fmt.Errorf("ROUTING_RESOURCE_NOT_REGULAR: %s", relative)
    }
    data, err := os.ReadFile(filename)
    if err != nil { return nil, err }
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
    routing, err := routingFile(root, policyRoutingData, ".ptsip/contracts/policy/routing/")
    if err != nil { return nil, nil, nil, 0, 0, nil, err }
    schema, err := routingFile(root, policyRoutingSchema, ".ptsip/contracts/policy/routing/")
    if err != nil { return nil, nil, nil, 0, 0, nil, err }
    if routingString(schema["$id"]) != "urn:ptsip:contract:policy-routing:v2" {
        return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTING_SCHEMA_ID_MISMATCH")
    }
    sources := routingMembers(routing["identity_sources"])
    expected := []struct{key, file, id string}{
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
        if err != nil { return nil, nil, nil, 0, 0, nil, err }
        if routingString(doc["$id"]) != item.id { return nil, nil, nil, 0, 0, nil, fmt.Errorf("IDENTITY_SCHEMA_ID_MISMATCH: %s", item.key) }
        values, ok := doc["enum"].([]any)
        if !ok || len(values) == 0 { return nil, nil, nil, 0, 0, nil, fmt.Errorf("EMPTY_IDENTITY_VOCABULARY: %s", item.key) }
        names := map[string]bool{}
        for _, value := range values {
            name := routingString(value)
            if name == "" || names[name] { return nil, nil, nil, 0, 0, nil, fmt.Errorf("DUPLICATE_IDENTITY_VALUE: %s", item.key) }
            names[name] = true
        }
        vocabulary[item.key] = names
        resources = append(resources, doc)
        identities = append(identities, item.file)
    }
    kinds := routingMembers(routing["routes"])
    if len(kinds) == 0 { return nil, nil, nil, 0, 0, nil, fmt.Errorf("EMPTY_POLICY_ROUTES") }
    registered, references := 0, 0
    for kind, entries := range kinds {
        if !vocabulary["policy_kind"][kind] { return nil, nil, nil, 0, 0, nil, fmt.Errorf("UNKNOWN_POLICY_KIND: %s", kind) }
        families := routingMembers(entries)
        for family, raw := range families {
            if !vocabulary["root_family"][family] { return nil, nil, nil, 0, 0, nil, fmt.Errorf("UNKNOWN_POLICY_FAMILY: %s", family) }
            selected := routingMembers(raw)
            output := routingString(routingMembers(selected["output"])["directory"])
            prefix := "developer/policy/"
            if kind == "PTSIP_SUPPORT_FEATURE" { prefix = "src/policy/" }
            if output != prefix+family { return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTE_DIRECTORY_MISMATCH: %s/%s", kind, family) }
            schemaRefs := routingMembers(selected["schema_refs"])
            for _, key := range []string{"public", "authority", "rules"} {
                target := routingString(schemaRefs[key])
                doc, err := routingFile(root, target, ".ptsip/contracts/schemas/")
                if err != nil { return nil, nil, nil, 0, 0, nil, err }
                if routingString(doc["$id"]) == "" { return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTE_SCHEMA_ID_MISSING: %s/%s/%s", kind, family, key) }
                references++
            }
            registered++
        }
    }
    // An explicitly listed developer family is required; discovery must not
    // silently omit a defined Root Family from route coverage.
    developer := routingMembers(kinds["PTSIP_DEVELOPER_POLICY"])
    for family := range vocabulary["root_family"] {
        if developer[family] == nil { return nil, nil, nil, 0, 0, nil, fmt.Errorf("ROUTING_COVERAGE_MISSING: %s", family) }
    }
    return routing, schema, resources, registered, references, identities, nil
}

func probePolicyRouting(root string, e Engine) RoutingProbe {
    p := RoutingProbe{
        Status:"UNRESOLVED", RoutingPath:policyRoutingData,
        SchemaPath:policyRoutingSchema, PolicyConformance:"NOT_ASSERTED",
    }
    routing, schema, resources, routes, refs, identities, err := loadRoutingInputs(root)
    if err != nil { p.Problem = err.Error(); return p }
    p.RegisteredRoutes = routes
    p.SchemaReferences = refs
    p.IdentitySources = identities
    valid, err := nativeValidate(e, schema, routing, resources)
    if err != nil { p.Problem = err.Error(); return p }
    if !valid { p.Problem = "ROUTING_CONTRACT_SCHEMA_REJECTED"; return p }
    p.Status = "PASS"
    return p
}
