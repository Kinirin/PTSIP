package schema_check

import (
    "encoding/json"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
)

func contractJSON(t *testing.T, root, relative string) map[string]any {
    t.Helper()
    raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
    if err != nil { t.Fatal(err) }
    var obj map[string]any
    if err := json.Unmarshal(raw, &obj); err != nil { t.Fatalf("%s: %v", relative, err) }
    return obj
}
func contractObject(t *testing.T, obj map[string]any, key string) map[string]any {
    t.Helper()
    child, ok := obj[key].(map[string]any)
    if !ok { t.Fatalf("missing object: %s",key) }
    return child
}
func contractArray(t *testing.T, obj map[string]any, key string) []any {
    t.Helper()
    result, ok := obj[key].([]any)
    if !ok { t.Fatalf("missing array: %s",key) }
    return result
}
func containsString(haystack []any, needle string) bool {
    for _, item := range haystack { if item == needle { return true } }
    return false
}

func TestPolicyAuthorityIdentifierIsTopLevel(t *testing.T) {
    root := routingRepositoryRoot(t)
    authority := contractJSON(t,root,".ptsip/contracts/schemas/Authority/PTSIP_Policy_Authority.schema.json")
    if authority["$id"] != "urn:ptsip:policy:Policy_Authority:v3" { t.Fatal("authority contract version mismatch") }
    if !containsString(contractArray(t,authority,"required"),"authority_id") { t.Fatal("authority_id must be required") }
    props := contractObject(t,authority,"properties")
    if props["authority_Identity"] != nil { t.Fatal("obsolete nested authority_Identity") }
    if _, ok := props["authority_id"].(map[string]any); !ok { t.Fatal("authority_id must have a schema") }
    parent := contractJSON(t,root,".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json")
    ref := contractObject(t,contractObject(t,parent,"properties"),"Authority")
    if ref["$ref"] != authority["$id"] { t.Fatal("public policy schema refers to old Authority contract") }
}

func TestEveryFamilyRequiresObligationLevelRuleState(t *testing.T) {
    root := routingRepositoryRoot(t)
    common := contractJSON(t,root,".ptsip/contracts/schemas/Policy_Rules/Public_Structure_Rules.schema.json")
    defs := contractObject(t,common,"$defs")
    obligation := contractObject(t,defs,"ObligationState")
    if !containsString(contractArray(t,obligation,"required"),"Rule_state") { t.Fatal("shared obligation state not required") }
    if _, ok := contractObject(t,defs,"RuleState")["minLength"]; !ok { t.Fatal("RuleState must be a typed, nonempty string") }

    routes := contractJSON(t,root,policyRoutingData)
    developers := contractObject(t,contractObject(t,routes,"routes"),"PTSIP_DEVELOPER_POLICY")
    if len(developers)!=14 { t.Fatalf("unexpected root family count: %d",len(developers)) }
    for family, raw := range developers {
        route := raw.(map[string]any)
        schemaRef := contractObject(t,route,"schema_refs")["rules"].(string)
        schema := contractJSON(t,root,schemaRef)
        schemaDefs := contractObject(t,schema,"$defs")
        var group map[string]any
        if family=="GOV" {
            doc := contractObject(t,schemaDefs,"PolicyDocumentRules")
            layers := contractArray(t,doc,"allOf")
            group = layers[1].(map[string]any)
            identity := contractObject(t,contractObject(t,group,"properties"),"Identity")
            if containsString(contractArray(t,identity,"required"),"Rule_state") ||
               contractObject(t,identity,"properties")["Rule_state"] != nil {
                t.Fatal("GOV Identity incorrectly owns Rule_state")
            }
            if schema["$id"]!="urn:ptsip:policy:GOV_rules:v6" {t.Fatal("GOV rule contract version not migrated")}
            if containsString(contractArray(t,identity,"required"),"family") || containsString(contractArray(t,identity,"required"),"version") ||
                !containsString(contractArray(t,identity,"required"),"Rule_ver") ||
                contractObject(t,identity,"properties")["family"] != nil ||
                contractObject(t,identity,"properties")["version"] != nil {
                t.Fatal("GOV Rule Identity retains duplicate fields or is missing Rule_ver")
            }

        } else {
            doc := contractObject(t,schemaDefs,"IndependentRuleContract")
            group = contractArray(t,doc,"allOf")[1].(map[string]any)
            if schema["$id"]!="urn:ptsip:policy:"+family+"_rules:v3" { t.Fatalf("%s: candidate schema ID not migrated",family) }
        }
        constraints := contractObject(t,contractObject(t,group,"properties"),"Constraints")
        obligations := contractObject(t,contractObject(t,constraints,"properties"),"obligations")
        if obligations["type"]!="array" {t.Fatalf("%s: obligations not an array",family)}
        item := contractObject(t,obligations,"items")
        if family=="GOV" {
            if !containsString(contractArray(t,item,"required"),"id") ||
                !containsString(contractArray(t,item,"required"),"rule") {
                t.Fatal("GOV unified constraint record requires id and rule")
            }
            if contractObject(t,item,"properties")["condition"] != nil ||
                contractObject(t,contractObject(t,constraints,"properties"),"prohibitions") != nil {
                t.Fatal("GOV legacy constraint fields still defined")
            }
            rules := contractArray(t,item,"allOf")
            guarded := contractObject(t,rules[0].(map[string]any),"then")
            if !containsString(contractArray(t,guarded,"required"),"Rule_state") ||
                !containsString(contractArray(t,guarded,"required"),"applies_when") {
                t.Fatal("GOV obligations must retain applicability and status")
            }
        } else if item["$ref"]!="urn:ptsip:policy:Public_Structure_Rules:v1#/$defs/ObligationState" {
            t.Fatalf("%s: missing shared Rule_state contract",family)
        }
    }
}

func TestCurrentRootFamilyPolicyFields(t *testing.T) {
    root := routingRepositoryRoot(t)
    routing := contractJSON(t,root,policyRoutingData)
    developers := contractObject(t,contractObject(t,routing,"routes"),"PTSIP_DEVELOPER_POLICY")
    obligation := regexp.MustCompile(`(?m)^      - id: OBLIGATION/[A-Z0-9_]+`)
    prohibition := regexp.MustCompile(`(?m)^      - id: PROHIBITION/[A-Z0-9_]+`)
    appliesWhen := regexp.MustCompile(`(?m)^        applies_when: `)
    status := regexp.MustCompile(`(?m)^        Rule_state: `)
    direct := regexp.MustCompile(`(?m)^  authority_id: `)
    migrated := 0
    families := 0
    for family, raw := range developers {
        output := contractObject(t,raw.(map[string]any),"output")
        directory, ok := output["directory"].(string)
        if !ok || directory == "" { t.Fatalf("%s: output.directory must be a nonempty string",family) }
        entries, err := os.ReadDir(filepath.Join(root,filepath.FromSlash(directory)))
        if err != nil { t.Fatal(err) }
        modern := 0
        for _, file := range entries {
            if file.IsDir() || !strings.HasSuffix(file.Name(),".yaml") {continue}
            data, err := os.ReadFile(filepath.Join(root,filepath.FromSlash(directory),file.Name()))
            if err!=nil {t.Fatal(err)}
            source:=string(data)
            if !strings.HasPrefix(source,"Policy_Identity:\n") {continue}
            modern++
            migrated++
            if strings.Contains(source,"  authority_Identity:") || !direct.MatchString(source) {
                t.Errorf("%s: noncanonical Authority identifier",file.Name())
            }
            if family=="GOV" {
                if strings.Contains(source,"    family: GOV\n") || strings.Contains(source,"    version: \"1\"\n") || !strings.Contains(source,"    Rule_ver: \"1\"\n") {
                    t.Errorf("%s: duplicate Rule Identity metadata",file.Name())
                }
                if regexp.MustCompile(`(?m)^    Rule_state: `).MatchString(source) {t.Errorf("%s: legacy Identity.Rule_state",file.Name())}
                begin:=strings.Index(source,"    obligations:\n")
                end:=strings.Index(source,"  Evaluation:\n")
                if begin<0 || end<=begin {t.Errorf("%s: no GOV unified constraints",file.Name());continue}
                body:=source[begin:end]
                obligations:=len(obligation.FindAllStringIndex(body,-1))
                prohibitions:=len(prohibition.FindAllStringIndex(body,-1))
                if obligations==0 || prohibitions==0 ||
                    len(status.FindAllStringIndex(body,-1))!=obligations ||
                    len(appliesWhen.FindAllStringIndex(body,-1))!=obligations ||
                    strings.Contains(body,"    prohibitions:") ||
                    strings.Contains(body,"        condition:") ||
                    strings.Contains(body,"        operation:") {
                    t.Errorf("%s: GOV constraints migration incomplete",file.Name())
                }
                if !strings.Contains(source,"  Policy_Status_Schema_ver:") ||
                    !strings.Contains(source,"  status: APPROVED.provisional") ||
                    !strings.Contains(source,"  contract: urn:ptsip:contract:policy:gov:") ||
                    strings.Contains(source,"  constant:") || strings.Contains(source,"  Schema:") ||
                    regexp.MustCompile(`(?m)^    contract_id: .*:v[0-9]+"package schema_check

import (
    "encoding/json"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
)

func contractJSON(t *testing.T, root, relative string) map[string]any {
    t.Helper()
    raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
    if err != nil { t.Fatal(err) }
    var obj map[string]any
    if err := json.Unmarshal(raw, &obj); err != nil { t.Fatalf("%s: %v", relative, err) }
    return obj
}
func contractObject(t *testing.T, obj map[string]any, key string) map[string]any {
    t.Helper()
    child, ok := obj[key].(map[string]any)
    if !ok { t.Fatalf("missing object: %s",key) }
    return child
}
func contractArray(t *testing.T, obj map[string]any, key string) []any {
    t.Helper()
    result, ok := obj[key].([]any)
    if !ok { t.Fatalf("missing array: %s",key) }
    return result
}
func containsString(haystack []any, needle string) bool {
    for _, item := range haystack { if item == needle { return true } }
    return false
}

func TestPolicyAuthorityIdentifierIsTopLevel(t *testing.T) {
    root := routingRepositoryRoot(t)
    authority := contractJSON(t,root,".ptsip/contracts/schemas/Authority/PTSIP_Policy_Authority.schema.json")
    if authority["$id"] != "urn:ptsip:policy:Policy_Authority:v3" { t.Fatal("authority contract version mismatch") }
    if !containsString(contractArray(t,authority,"required"),"authority_id") { t.Fatal("authority_id must be required") }
    props := contractObject(t,authority,"properties")
    if props["authority_Identity"] != nil { t.Fatal("obsolete nested authority_Identity") }
    if _, ok := props["authority_id"].(map[string]any); !ok { t.Fatal("authority_id must have a schema") }
    parent := contractJSON(t,root,".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json")
    ref := contractObject(t,contractObject(t,parent,"properties"),"Authority")
    if ref["$ref"] != authority["$id"] { t.Fatal("public policy schema refers to old Authority contract") }
}

func TestEveryFamilyRequiresObligationLevelRuleState(t *testing.T) {
    root := routingRepositoryRoot(t)
    common := contractJSON(t,root,".ptsip/contracts/schemas/Policy_Rules/Public_Structure_Rules.schema.json")
    defs := contractObject(t,common,"$defs")
    obligation := contractObject(t,defs,"ObligationState")
    if !containsString(contractArray(t,obligation,"required"),"Rule_state") { t.Fatal("shared obligation state not required") }
    if _, ok := contractObject(t,defs,"RuleState")["minLength"]; !ok { t.Fatal("RuleState must be a typed, nonempty string") }

    routes := contractJSON(t,root,policyRoutingData)
    developers := contractObject(t,contractObject(t,routes,"routes"),"PTSIP_DEVELOPER_POLICY")
    if len(developers)!=14 { t.Fatalf("unexpected root family count: %d",len(developers)) }
    for family, raw := range developers {
        route := raw.(map[string]any)
        schemaRef := contractObject(t,route,"schema_refs")["rules"].(string)
        schema := contractJSON(t,root,schemaRef)
        schemaDefs := contractObject(t,schema,"$defs")
        var group map[string]any
        if family=="GOV" {
            doc := contractObject(t,schemaDefs,"PolicyDocumentRules")
            layers := contractArray(t,doc,"allOf")
            group = layers[1].(map[string]any)
            identity := contractObject(t,contractObject(t,group,"properties"),"Identity")
            if containsString(contractArray(t,identity,"required"),"Rule_state") ||
               contractObject(t,identity,"properties")["Rule_state"] != nil {
                t.Fatal("GOV Identity incorrectly owns Rule_state")
            }
            if schema["$id"]!="urn:ptsip:policy:GOV_rules:v6" {t.Fatal("GOV rule contract version not migrated")}
            if containsString(contractArray(t,identity,"required"),"family") || containsString(contractArray(t,identity,"required"),"version") ||
                !containsString(contractArray(t,identity,"required"),"Rule_ver") ||
                contractObject(t,identity,"properties")["family"] != nil ||
                contractObject(t,identity,"properties")["version"] != nil {
                t.Fatal("GOV Rule Identity retains duplicate fields or is missing Rule_ver")
            }

        } else {
            doc := contractObject(t,schemaDefs,"IndependentRuleContract")
            group = contractArray(t,doc,"allOf")[1].(map[string]any)
            if schema["$id"]!="urn:ptsip:policy:"+family+"_rules:v3" { t.Fatalf("%s: candidate schema ID not migrated",family) }
        }
        constraints := contractObject(t,contractObject(t,group,"properties"),"Constraints")
        obligations := contractObject(t,contractObject(t,constraints,"properties"),"obligations")
        if obligations["type"]!="array" {t.Fatalf("%s: obligations not an array",family)}
        item := contractObject(t,obligations,"items")
        if family=="GOV" {
            if !containsString(contractArray(t,item,"required"),"id") ||
                !containsString(contractArray(t,item,"required"),"rule") {
                t.Fatal("GOV unified constraint record requires id and rule")
            }
            if contractObject(t,item,"properties")["condition"] != nil ||
                contractObject(t,contractObject(t,constraints,"properties"),"prohibitions") != nil {
                t.Fatal("GOV legacy constraint fields still defined")
            }
            rules := contractArray(t,item,"allOf")
            guarded := contractObject(t,rules[0].(map[string]any),"then")
            if !containsString(contractArray(t,guarded,"required"),"Rule_state") ||
                !containsString(contractArray(t,guarded,"required"),"applies_when") {
                t.Fatal("GOV obligations must retain applicability and status")
            }
        } else if item["$ref"]!="urn:ptsip:policy:Public_Structure_Rules:v1#/$defs/ObligationState" {
            t.Fatalf("%s: missing shared Rule_state contract",family)
        }
    }
}

func TestCurrentRootFamilyPolicyFields(t *testing.T) {
    root := routingRepositoryRoot(t)
    routing := contractJSON(t,root,policyRoutingData)
    developers := contractObject(t,contractObject(t,routing,"routes"),"PTSIP_DEVELOPER_POLICY")
    obligation := regexp.MustCompile(`(?m)^      - id: OBLIGATION/[A-Z0-9_]+`)
    prohibition := regexp.MustCompile(`(?m)^      - id: PROHIBITION/[A-Z0-9_]+`)
    appliesWhen := regexp.MustCompile(`(?m)^        applies_when: `)
    status := regexp.MustCompile(`(?m)^        Rule_state: `)
    direct := regexp.MustCompile(`(?m)^  authority_id: `)
    migrated := 0
    families := 0
    for family, raw := range developers {
        output := contractObject(t,raw.(map[string]any),"output")
        directory, ok := output["directory"].(string)
        if !ok || directory == "" { t.Fatalf("%s: output.directory must be a nonempty string",family) }
        entries, err := os.ReadDir(filepath.Join(root,filepath.FromSlash(directory)))
        if err != nil { t.Fatal(err) }
        modern := 0
        for _, file := range entries {
            if file.IsDir() || !strings.HasSuffix(file.Name(),".yaml") {continue}
            data, err := os.ReadFile(filepath.Join(root,filepath.FromSlash(directory),file.Name()))
            if err!=nil {t.Fatal(err)}
            source:=string(data)
            if !strings.HasPrefix(source,"Policy_Identity:\n") {continue}
            modern++
            migrated++
            if strings.Contains(source,"  authority_Identity:") || !direct.MatchString(source) {
                t.Errorf("%s: noncanonical Authority identifier",file.Name())
            }
            if family=="GOV" {
                if strings.Contains(source,"    family: GOV\n") || strings.Contains(source,"    version: \"1\"\n") || !strings.Contains(source,"    Rule_ver: \"1\"\n") {
                    t.Errorf("%s: duplicate Rule Identity metadata",file.Name())
                }
                if regexp.MustCompile(`(?m)^    Rule_state: `).MatchString(source) {t.Errorf("%s: legacy Identity.Rule_state",file.Name())}
).MatchString(source) {
                    t.Errorf("%s: GOV Status or rule identity migration incomplete",file.Name())
                }
            } else {
                expected:="candidate_schema_id: urn:ptsip:policy:"+family+"_rules:v3"
                if !strings.Contains(source,expected) {t.Errorf("%s: old candidate schema identifier",file.Name())}
            }
        }
        if modern==0 {t.Errorf("%s: no current policy with new structure",family)} else {families++}
    }
    if families!=14 || migrated<17 {t.Fatalf("coverage: families=%d modern_policy_docs=%d",families,migrated)}
}
