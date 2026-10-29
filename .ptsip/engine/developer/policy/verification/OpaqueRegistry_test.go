package schema_check

import (
    "encoding/json"
    "strings"
    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
    "os"
    "path/filepath"
    "regexp"
    "testing"
)

func TestOpaqueResponsibilityContractSelection(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err != nil { t.Fatal(err) }
    families := []string{"ARCH","ASSURE","CHANGE","CNTR","CTRL","GOV","INFO","INTENT","NORM","OPS","REAL","RECORD","RISK","SUPPLY"}
    for _, family := range families {
        t.Run(family, func(t *testing.T) {
            path := filepath.Join(root,".ptsip","contracts","policy","Rules",family+"_rules.json")
            raw, err := os.ReadFile(path)
            if err != nil { t.Fatal(err) }
            var contract map[string]any
            if err := json.Unmarshal(raw,&contract); err != nil { t.Fatal(err) }
            structure := contract["structure"].(map[string]any)
            props := structure["properties"].(map[string]any)
            if _, exists := props["Constraints"]; exists { t.Fatal("Constraints must be retired") }
            for _, value := range structure["required"].([]any) {
                if value == "Constraints" { t.Fatal("retired field is still required") }
            }
            def := props["Definition"].(map[string]any)
            items := def["items"].(map[string]any)
            fields := items["properties"].(map[string]any)
            if fields["rule"].(map[string]any)["pattern"] != "^RULE-[A-F0-9]{32}$" {
                t.Fatal("rule constant embeds semantics")
            }
            if fields["id"].(map[string]any)["$ref"] != "urn:ptsip:rbac:responsibility:v1#/$defs/ID" {
                t.Fatal("rule responsibility is not linked")
            }
        })
    }
}

func TestOpaqueResponsibilityLifecycleAndLookupShape(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err != nil { t.Fatal(err) }
    prefix := filepath.Join(root,".ptsip","contracts","schemas","compile-sealed-rbac")
    read := func(path string) map[string]any {
        t.Helper()
        raw, err := os.ReadFile(path)
        if err != nil { t.Fatal(err) }
        var data map[string]any
        if err := json.Unmarshal(raw,&data); err != nil { t.Fatal(err) }
        return data
    }
    status := read(filepath.Join(prefix,"lifecycle","status.json"))
    values := status["enum"].([]any)
    if len(values)!=2 || values[0]!="OBLIGATION" || values[1]!="PROHIBITION" {
        t.Fatal("unapproved responsibility lifecycle status",values)
    }
    profile := read(filepath.Join(prefix,"Responsibilities_List.schema.json"))
    bindings := profile["properties"].(map[string]any)["component_bindings"].(map[string]any)
    pattern := bindings["patternProperties"].(map[string]any)
    row := pattern["^R-ID-[A-F0-9]{32}$"].(map[string]any)
    fields := row["properties"].(map[string]any)
    if _, legacy:=fields["Rule_state"]; legacy { t.Fatal("policy Rule_state reused") }
    if fields["status"].(map[string]any)["$ref"]!="urn:ptsip:rbac:lifecycle:index:v1#/$defs/Status" {
        t.Fatal("status must resolve through lifecycle/index.json")
    }
    for _, retired:=range []string{"source_text","source_policy","source_rule_id"} {
        if _, exists:=fields[retired]; exists { t.Fatal("retired natural-language source field",retired) }
    }
    intent := read(filepath.Join(prefix,"authority","Authorization_intention.schema.json"))
    records := intent["properties"].(map[string]any)["authorizations"].(map[string]any)["items"].(map[string]any)
    required := records["required"].([]any)
    if len(required)!=2 || required[0]!="Authorization_ID" || required[1]!="intention" {
        t.Fatal("intention lookup fields changed")
    }
    keywords := read(filepath.Join(prefix,"lifecycle","keywords.json"))
    idRule := keywords["$defs"].(map[string]any)["AuthorizationID"].(map[string]any)["pattern"].(string)
    re, err := regexp.Compile(idRule)
    if err!=nil {t.Fatal(err)}
    if !re.MatchString("REA-ABCDEFGHIJKLMNO") || re.MatchString("REA-INVALID_NAME") || re.MatchString("NEW-ABCDEFGHIJKLMNO") {
        t.Fatal("authorization pattern permits unregistered category")
    }
}

func TestOpaqueRegistryResolvesActualGovAndFailsClosedOnMissingIntention(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err != nil { t.Fatal(err) }
    catalog, err := authoring.Load(root)
    if err != nil { t.Fatal(err) }
    validate, err := InstalledValidator()
    if err != nil { t.Skipf("installed validator unavailable: %v",err) }
    if err := catalog.CheckContracts(validate); err != nil { t.Fatal(err) }
    profile, err := authoring.ReadDocument(root, authoring.ProfilePath)
    if err != nil { t.Fatal(err) }
    if err := catalog.CheckLocalProfile(validate, profile); err != nil { t.Fatal(err) }
    gov, err := authoring.ReadDocument(root, "developer/policy/GOV/MPD-GOV-0006.yaml")
    if err != nil { t.Fatal(err) }
    if err := catalog.CheckRules(validate, gov); err != nil { t.Fatal(err) }
    if _, err := catalog.LookupAuthorizationIntention(validate,"REA-ABCDEFGHIJKLMNO"); err == nil || !strings.Contains(err.Error(),"UNRESOLVED_AUTHORIZATION_INTENTION") {
        t.Fatalf("undeclared Read authorization was not rejected: %v",err)
    }
    profileData, err := json.Marshal(profile)
    if err != nil { t.Fatal(err) }
    var mutated map[string]any
    if err := json.Unmarshal(profileData, &mutated); err != nil { t.Fatal(err) }
    rows:=mutated["component_bindings"].(map[string]any)
    for _, raw := range rows {
        raw.(map[string]any)["status"]="APPROVED.provisional"
        break
    }
    if err := catalog.CheckProfile(validate,mutated); err == nil {
        t.Fatal("policy approval state must not become an RBAC lifecycle status")
    }
}
