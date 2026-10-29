package schema_check

import (
    "encoding/json"
    "path/filepath"
    "strings"
    "testing"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

func TestApprovedExplicitEffectsAndUnregisteredBlockContract(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err != nil { t.Fatal(err) }
    catalog, err := authoring.Load(root)
    if err != nil { t.Fatal(err) }
    validate, err := InstalledValidator()
    if err != nil { t.Skipf("installed validator unavailable: %v", err) }
    if err := catalog.CheckContracts(validate); err != nil {
        t.Fatalf("approved decision contract rejected: %v", err)
    }
    if len(catalog.Authorizations["authorizations"].([]any)) != 0 {
        t.Fatal("approved decision model must not automatically register grants or denials")
    }
    registration := catalog.AuthorizationDecision["registration"].(map[string]any)
    effects := catalog.AuthorizationDecision["explicit_effects"].(map[string]any)
    binding := catalog.AuthorizationDecision["binding"].(map[string]any)
    if registration["missing"] != "BLOCK_UNRESOLVED" ||
        registration["implicit_allow"] != false ||
        registration["implicit_deny"] != false ||
        effects["ALLOW"] != "ALLOW_CANDIDATE" ||
        effects["DENY"] != "BLOCK_EXPLICIT" ||
        binding["missing_or_invalid"] != "BLOCK_UNRESOLVED" {
        t.Fatal("approved C policy machine outcomes changed")
    }
    unresolved := catalog.AuthorizationDecision["unresolved"].(map[string]any)
    for _, field := range []string{"resource_identity","condition_evaluation","subject_binding","conflict_precedence"} {
        if unresolved[field] != "UNRESOLVED" {
            t.Fatalf("unapproved decision was silently resolved: %s", field)
        }
    }
    if _, err := catalog.LookupAuthorizationIntention(validate,"Auth-UNKNOWN"); err == nil ||
        !strings.Contains(err.Error(),"UNRESOLVED_AUTHORIZATION_INTENTION") {
        t.Fatalf("missing authorization was not blocked: %v", err)
    }

    // Any silent change to the approved machine decision must invalidate
    // existing contracts, rather than gaining authority by implementation.
    original, err := json.Marshal(catalog.AuthorizationDecision)
    if err != nil { t.Fatal(err) }
    tests := []struct{
        name string
        mutate func(map[string]any)
    }{
        {"implicit-allow",func(v map[string]any) {
            v["registration"].(map[string]any)["implicit_allow"]=true
        }},
        {"implicit-deny",func(v map[string]any) {
            v["registration"].(map[string]any)["implicit_deny"]=true
        }},
        {"allow-runtime-grant",func(v map[string]any) {
            v["explicit_effects"].(map[string]any)["ALLOW"]="EXECUTION_AUTHORIZED"
        }},
        {"deny-rewritten",func(v map[string]any) {
            v["explicit_effects"].(map[string]any)["DENY"]="ALLOW_CANDIDATE"
        }},
        {"conflict-precedence-inferred",func(v map[string]any) {
            v["unresolved"].(map[string]any)["conflict_precedence"]="DENY_OVERRIDES"
        }},
        {"missing-registration-becomes-deny",func(v map[string]any) {
            v["registration"].(map[string]any)["missing"]="BLOCK_EXPLICIT"
        }},
    }
    for _, tc := range tests {
        t.Run(tc.name,func(t *testing.T) {
            var mutated map[string]any
            if err := json.Unmarshal(original,&mutated);err!=nil {t.Fatal(err)}
            tc.mutate(mutated)
            catalog.AuthorizationDecision=mutated
            if err:=catalog.CheckContracts(validate);err==nil {
                t.Fatal("unapproved decision-contract mutation accepted")
            }
        })
    }
}
