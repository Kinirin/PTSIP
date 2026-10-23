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
    if routingString(doc["schema_version"]) != "ptsip-policy-routing/v2" { t.Fatal("routing version mismatch") }
    if routingString(schema["$id"]) != "urn:ptsip:contract:policy-routing:v2" { t.Fatal("schema ID mismatch") }
    if len(resources) != 2 || len(sources) != 2 { t.Fatalf("missing identity sources: %d/%d",len(resources),len(sources)) }
    if registered != 14 || refs != 3*registered { t.Fatalf("route coverage or schema reference count: %d/%d",registered,refs) }
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
