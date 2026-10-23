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

func TestPolicyRoutingDetectsCorruptedContract(t *testing.T) {
    root := t.TempDir()
    path := filepath.Join(root,filepath.FromSlash(policyRoutingData))
    if err := os.MkdirAll(filepath.Dir(path),0755); err != nil {t.Fatal(err)}
    corrupted := `{"identity_sources":{"policy_kind":"developer/policy/registries/authority-subject-registry.yaml"}}`
    if err := os.WriteFile(path,[]byte(corrupted),0644); err != nil {t.Fatal(err)}
    _, _, _, _, _, _, err := loadRoutingInputs(root)
    if err == nil {t.Fatal("missing routing schema must fail closed")}
    if !strings.Contains(err.Error(),"MISSING") {t.Fatalf("unexpected error: %v",err)}
}
