package engines_test

import (
    "testing"
)

func TestPostB5CapabilityVersions(t *testing.T) {
    cases := []struct{
        name string
        engine string
        contract string
        publishedEngine string
        publishedContract string
        algorithm string
    }{
        {"validation","0.3.4","4.0.1","0.3.3","4.0.0","INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION"},
        {"projection","0.2.2","2.0.1","0.2.1","2.0.0","EXPLICIT_JSON_POINTER_SELECTION_WITHOUT_SOURCE_MUTATION_OR_LEGACY_OUTPUT_PARITY"},
    }
    approval := document(t,"developer/policy/approvals/MPA-20261026-VALIDATION-PROJECTION-CAPABILITY-CONTRACT-BUMPS.yaml")
    if approval["approval"].(object)["decision"] != "APPROVED" {
        t.Fatal("explicit owner capability publication versions missing")
    }
    for _,tc := range cases {
        t.Run(tc.name,func(t *testing.T) {
            contractPath := "src/ptsip/engines/"+tc.name+"/contract.json"
            manifestPath := "src/ptsip/engines/"+tc.name+"/manifest.json"
            bindingsPath := "src/ptsip/engines/"+tc.name+"/execution-bindings.json"
            c:=document(t,contractPath)
            m:=document(t,manifestPath)
            b:=document(t,bindingsPath)["bindings"].([]any)[0].(object)
            if c["version"] != tc.contract || m["implementation_version"] != tc.engine ||
                b["engine_version"] != tc.engine || b["contract_version"] != tc.contract {
                t.Fatal("release identity not projected consistently",tc.name)
            }
            capability := m["capabilities"].([]any)[0].(object)
            if capability["version"] != tc.contract || capability["sha256"] != digest(t,contractPath) {
                t.Fatal("stale capability binding",tc.name)
            }
            original := approval["approval"].(object)["published_reference"].(object)[tc.name].(object)
            next := approval["approval"].(object)["approved_versions"].(object)[tc.name].(object)
            if original["engine"] != tc.publishedEngine || original["contract"] != tc.publishedContract ||
                next["engine"] != tc.engine || next["contract"] != tc.contract {
                t.Fatal("approved release version no longer matches published b5 baseline",tc.name)
            }
            operations := c["operations"].([]any)
            if len(operations)==0 || operations[0].(object)["machine_semantics"].(object)["algorithm"] != tc.algorithm {
                t.Fatal("capability semantics unexpectedly regressed",tc.name)
            }
            for _,raw:=range operations {
                operation:=raw.(object)
                if operation["compatibility"].(object)["contract_version"] != tc.contract {
                    t.Fatal("mixed contract version",tc.name)
                }
            }
        })
    }
}
