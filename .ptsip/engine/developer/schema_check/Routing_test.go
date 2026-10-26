package schema_check

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func routingRepositoryRoot(t *testing.T) string {
    t.Helper()
    root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
    if err != nil { t.Fatal(err) }
    return root
}

func TestPolicyRoutingContractInputs(t *testing.T) {
    root := routingRepositoryRoot(t)
    doc, schema, resources, registered, refs, sources, err := loadRoutingInputs(root)
    if err != nil { t.Fatal(err) }
    if routingString(doc["schema_version"]) != "ptsip-policy-routing/v3" { t.Fatal("routing version mismatch") }
    if routingString(schema["$id"]) != "urn:ptsip:contract:policy-routing:v3" { t.Fatal("schema ID mismatch") }
    if len(resources) != 2 || len(sources) != 2 { t.Fatalf("missing identity sources: %d/%d",len(resources),len(sources)) }
    if registered != 14 || refs != 2*registered { t.Fatalf("route coverage or schema reference count: %d/%d",registered,refs) }
}

func TestPolicyRoutingCannotUseRetiredOrUnsafePaths(t *testing.T) {
    root := routingRepositoryRoot(t)
    for _, file := range []string{
        ".ptsip/contracts/routing/policy_routes.json",
        ".ptsip/contracts/policy/routing/../identity/policy_kind.json",
        ".ptsip/contracts/policy/routing/nonexistent.json",
        ".ptsip/contracts/policy/routing/../../outside.json",
        ".ptsip/contracts/policy/routing\\policy_routes.json",
        "/etc/passwd",
    } {
        if _, err := routingFile(root, file, ".ptsip/contracts/policy/routing/"); err == nil {
            t.Errorf("unexpected acceptance: %s",file)
        }
    }
}

func TestPolicyRoutingRejectsRetiredBinding(t *testing.T) {
    root := t.TempDir()
    boundary := filepath.Join(root, ".ptsip", "contracts", "policy", "boundary.json")
    if err := os.MkdirAll(filepath.Dir(boundary), 0755); err != nil { t.Fatal(err) }
    corrupted := `{"domain":"policy","namespaces":{"routing":"routing/"},"routing_binding":{"data":".ptsip/contracts/routing/policy_routes.json","schema":".ptsip/contracts/policy/routing/Policy_Routing.schema.json"}}`
    if err := os.WriteFile(boundary, []byte(corrupted), 0644); err != nil { t.Fatal(err) }
    _, _, _, _, _, _, err := loadRoutingInputs(root)
    if err == nil { t.Fatal("retired routing path was accepted") }
    if !strings.Contains(err.Error(), "INVALID_ROUTING_CONTRACT_PATH") {
        t.Fatalf("unexpected result: %v", err)
    }
}

func TestPolicyRoutingRequiresBoundSchemaFile(t *testing.T) {
    root := t.TempDir()
    boundary := filepath.Join(root, ".ptsip", "contracts", "policy", "boundary.json")
    if err := os.MkdirAll(filepath.Dir(boundary), 0755); err != nil { t.Fatal(err) }
    canonical := `{"domain":"policy","namespaces":{"routing":"routing/"},"routing_binding":{"data":".ptsip/contracts/policy/routing/policy_routes.json","schema":".ptsip/contracts/policy/routing/Policy_Routing.schema.json"}}`
    if err := os.WriteFile(boundary, []byte(canonical), 0644); err != nil { t.Fatal(err) }
    data := filepath.Join(root, filepath.FromSlash(policyRoutingData))
    if err := os.MkdirAll(filepath.Dir(data), 0755); err != nil { t.Fatal(err) }
    if err := os.WriteFile(data, []byte(`{"identity_sources":{}}`), 0644); err != nil { t.Fatal(err) }
    _, _, _, _, _, _, err := loadRoutingInputs(root)
    if err == nil || !strings.Contains(err.Error(), "ROUTING_RESOURCE_MISSING") {
        t.Fatalf("missing routing schema must fail closed: %v", err)
    }
}

func TestSharedPublicPolicyUsesGOVFlatStatusAndGOVRoute(t *testing.T) {
 root := routingRepositoryRoot(t)
 public, err := routingFile(root, ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json", ".ptsip/contracts/schemas/")
 if err != nil { t.Fatal(err) }
 if routingString(public["$id"]) != "urn:ptsip:policy:public-structure:v6" {
  t.Fatal("shared public schema version mismatch")
 }
 statusRef := routingString(routingMembers(routingMembers(public["properties"])["Policy_Status"])["$ref"])
 if statusRef != "urn:ptsip:policy:Status:v2" { t.Fatalf("unexpected shared Policy_Status reference: %s",statusRef) }
 flat, err := routingFile(root, publicPolicyStatusPath, ".ptsip/contracts/schemas/")
 if err != nil { t.Fatal(err) }
 if routingString(flat["$id"]) != statusRef { t.Fatal("shared flat status ID differs from its public reference") }
 fields := routingMembers(flat["properties"])
 for _, key := range []string{"Policy_Status_Schema_ver", "status", "contract"} {
  if fields[key] == nil { t.Fatalf("missing flat status field: %s",key) }
 }
 if len(fields)!=3 || fields["Schema"]!=nil || fields["constant"]!=nil {
  t.Fatal("legacy nested status representation remains in common schema")
 }
 if flat["additionalProperties"] != false { t.Fatal("flat status schema must reject undeclared fields") }
 required, ok := flat["required"].([]any)
 if !ok || len(required)!=3 { t.Fatal("flat status required fields mismatch") }
 route, _, _, _, _, _, err := loadRoutingInputs(root)
 if err != nil { t.Fatal(err) }
 families := routingMembers(routingMembers(route["routes"])["PTSIP_DEVELOPER_POLICY"])
 govRoute := routingMembers(routingMembers(routingMembers(families["GOV"])["schema_refs"]))
 if routingString(govRoute["public"]) != ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json" {
  t.Fatal("GOV must resolve public schema from the shared common contract")
 }
 if routingString(govRoute["rules"]) != ".ptsip/contracts/schemas/Policy_Rules/root-family/GOV_rules.schema.json" {
  t.Fatal("GOV independent rules binding must be preserved")
 }
}
