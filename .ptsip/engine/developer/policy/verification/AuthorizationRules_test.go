package schema_check

import (
    "encoding/json"
    "path/filepath"
    "strings"
    "testing"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

func TestAuthorizationRulesRequireAllFiveFields(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err != nil { t.Fatal(err) }
    catalog, err := authoring.Load(root)
    if err != nil { t.Fatal(err) }
    validate, err := InstalledValidator()
    if err != nil { t.Skipf("installed validator unavailable: %v", err) }
    full := map[string]any{
        "ID":"Auth-READ",
        "resource":"EXAMPLE-RESOURCE",
        "action":"REA",
        "effect":"DENY",
        "condition_name":"POLICY-STATUS",
    }
    makeRegistry := func(rule map[string]any) authoring.Document {
        raw,err:=json.Marshal(rule)
        if err!=nil {t.Fatal(err)}
        var item map[string]any
        if err:=json.Unmarshal(raw,&item);err!=nil{t.Fatal(err)}
        return authoring.Document{
            "$schema":"./Authorization_intention.schema.json",
            "schema_version":"ptsip-authorization-intention/v1",
            "authorizations":[]any{item},
        }
    }
    catalog.Authorizations=makeRegistry(full)
    if err:=catalog.CheckContracts(validate);err!=nil{t.Fatal("valid five-field authorization rejected",err)}
    profile,err:=authoring.ReadDocument(root,authoring.ProfilePath)
    if err!=nil{t.Fatal(err)}
    var linked bool
    for _,raw:=range profile["component_bindings"].(map[string]any) {
        record:=raw.(map[string]any)
        if record["condition_name"]=="POLICY-STATUS" {
            record["Authorization_ID"]="Auth-READ"
            linked=true
            break
        }
    }
    if !linked {t.Fatal("expected registered condition label not found")}
    if err:=catalog.CheckProfile(validate,profile);err!=nil{t.Fatal("registered rule/condition binding rejected",err)}
    for _,raw:=range profile["component_bindings"].(map[string]any) {
        item:=raw.(map[string]any)
        if item["Authorization_ID"]=="Auth-READ" {
            item["Authorization_ID"]="Auth-UNKNOWN"
            break
        }
    }
    if err:=catalog.CheckProfile(validate,profile);err==nil{t.Fatal("unregistered authorization was accepted")}
    if _,err:=catalog.LookupAuthorizationIntention(validate,"Auth-READ");err==nil||!strings.Contains(err.Error(),"UNBOUND_AUTHORIZATION_INTENTION"){
        t.Fatalf("authorization not bound in on-disk profile must fail closed: %v",err)
    }

    cases := []struct{name string; change func(map[string]any)}{
        {"missing-ID",func(v map[string]any){delete(v,"ID")}},
        {"missing-resource",func(v map[string]any){delete(v,"resource")}},
        {"missing-action",func(v map[string]any){delete(v,"action")}},
        {"missing-effect",func(v map[string]any){delete(v,"effect")}},
        {"missing-condition-name",func(v map[string]any){delete(v,"condition_name")}},
        {"invalid-action",func(v map[string]any){v["action"]="DELETE"}},
        {"invalid-effect",func(v map[string]any){v["effect"]="APPROVED"}},
        {"invalid-ID",func(v map[string]any){v["ID"]="REA-ABCDEFGHIJKLMNO"}},
        {"unknown-condition",func(v map[string]any){v["condition_name"]="UNKNOWN-CONDITION"}},
        {"extra-field",func(v map[string]any){v["intention"]="legacy"}},
    }
    for _,tc:=range cases{
        t.Run(tc.name,func(t *testing.T){
            candidate:=makeRegistry(full)
            row:=candidate["authorizations"].([]any)[0].(map[string]any)
            tc.change(row)
            catalog.Authorizations=candidate
            if err:=catalog.CheckContracts(validate);err==nil{t.Fatal("invalid authorization rule was accepted")}
        })
    }
}

func TestNamedRuleConstantRejectsRandomHashesAndLongNames(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err!=nil{t.Fatal(err)}
    catalog,err:=authoring.Load(root)
    if err!=nil{t.Fatal(err)}
    validate,err:=InstalledValidator()
    if err!=nil{t.Skipf("installed validator unavailable: %v",err)}
    original,err:=authoring.ReadDocument(root,"developer/policy/GOV/MPD-GOV-0006.yaml")
    if err!=nil{t.Fatal(err)}
    for _,tc:=range []string{"RULE-FD125E8425C39349304D0BC566F5701E","THIS-RULE-NAME-IS-TOO-LONG","lowercase"}{
        raw,_:=json.Marshal(original)
        var doc authoring.Document
        if err:=json.Unmarshal(raw,&doc);err!=nil{t.Fatal(err)}
        row:=doc["Policy_Rules"].(map[string]any)["Definition"].([]any)[0].(map[string]any)
        row["rule"]=tc
        if err:=catalog.CheckRules(validate,doc);err==nil{t.Fatalf("invalid rule constant accepted: %s",tc)}
    }
}
