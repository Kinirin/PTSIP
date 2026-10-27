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
 rules, err := routingFile(root, ".ptsip/contracts/schemas/Policy_Rules/root-family/GOV_rules.schema.json", ".ptsip/contracts/schemas/")
 if err != nil {t.Fatal(err)}
 if routingString(rules["$id"]) != "urn:ptsip:policy:GOV_rules:v6" {t.Fatal("GOV rules contract ID changed during move")}
 if _,err := os.Stat(filepath.Join(root,filepath.FromSlash(".ptsip/contracts/schemas/Policy_Rules/GOV_rules.schema.json"))); !os.IsNotExist(err) {
  t.Fatalf("noncanonical GOV rules schema still exists: %v",err)
 }
}

func TestGOVPolicyDocumentsUseSharedRootWithoutAuthority(t *testing.T) {
 root := routingRepositoryRoot(t)
 for _, id := range []string{"0001","0004","0005","0006"} {
  rel := "developer/policy/GOV/MPD-GOV-"+id+".yaml"
  data,err := os.ReadFile(filepath.Join(root,filepath.FromSlash(rel)))
  if err != nil {t.Fatal(err)}
  normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
  lines := strings.Split(normalized, "\n")
  roots := []string{}
  for _,line := range lines {
   if strings.HasPrefix(line," ") || strings.HasPrefix(line,"#") || line=="" {continue}
   if strings.HasSuffix(line,":") {roots=append(roots,line)}
  }
  if got:=strings.Join(roots,"|");got!="Policy_Identity:|Policy_Status:|Policy_Rules:" {
   t.Errorf("%s unexpected top-level sections: %s", rel,got)
  }
  if !strings.Contains(normalized,"  status: APPROVED.Applying\n") ||
   !strings.Contains(normalized,"  Policy_Status_Schema_ver:") ||
   !strings.Contains(normalized,"  contract: urn:ptsip:contract:policy:gov:") {
   t.Errorf("%s missing current flat GOV Policy_Status",rel)
  }
 }
}

func TestAllFamilyRulesSchemaRoutesUseCanonicalDirectory(t *testing.T) {
 root:=routingRepositoryRoot(t)
 routing,_,_,_,_,_,err:=loadRoutingInputs(root)
 if err!=nil{t.Fatal(err)}
 families:=routingMembers(routingMembers(routing["routes"])["PTSIP_DEVELOPER_POLICY"])
 if len(families)!=14 {t.Fatalf("expected 14 families: %d",len(families))}
 for family,raw:=range families {
  refs:=routingMembers(routingMembers(raw)["schema_refs"])
  target:=routingString(refs["rules"])
  if !canonicalFamilyRulesSchemaPath(family,target){t.Errorf("%s rules schema outside root-family: %s",family,target)}
 }
}

func TestFamilyRulesSchemaLocationRejectsNoncanonicalPaths(t *testing.T) {
 for _,path:=range []string{
  ".ptsip/contracts/schemas/Policy_Rules/GOV_rules.schema.json",
  ".ptsip/contracts/schemas/Policy_Rules/abortively/v2_GOV.schema.json",
  ".ptsip/contracts/schemas/Policy_Rules/root-family/INFO_rules.schema.json",
  ".ptsip/contracts/schemas/Policy_Rules/root-family/../GOV_rules.schema.json",
  ".ptsip/contracts/schemas/Policy_Rules/root-family/GOV_rules.json",
 }{
  if canonicalFamilyRulesSchemaPath("GOV",path) {t.Fatalf("incorrect schema location accepted: %s",path)}
 }
 for _,suffix:=range []string{".schema.json",".schemas.json"}{
  target:=familyRulesSchemaRoot+"GOV_rules"+suffix
  if !canonicalFamilyRulesSchemaPath("GOV",target){t.Fatalf("valid schema name rejected: %s",target)}
 }
}
