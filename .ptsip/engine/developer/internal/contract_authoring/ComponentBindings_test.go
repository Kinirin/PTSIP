package contract_authoring_test

import (
    "encoding/json"
    "strings"
    "testing"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

func TestRIdKeyedRuleBindingsAndNoComponentMetadata(t *testing.T) {
    catalog, validate := nativeCatalog(t)
    profile, err := authoring.ReadDocument(catalog.Root, authoring.ProfilePath)
    if err != nil { t.Fatal(err) }
    if err := catalog.CheckLocalProfile(validate, profile); err != nil { t.Fatal(err) }
    bindings, ok := profile["component_bindings"].(map[string]any)
    if !ok || len(bindings) < 70 { t.Fatal("R-ID rule registry not populated") }
    if _, ok := profile["component_map"]; ok { t.Fatal("retired component_map remains") }
    for id, raw := range bindings {
        if !strings.HasPrefix(id, "R-ID-") { t.Fatal("non-R-ID binding", id) }
        item := raw.(map[string]any)
        if item["rule"] == "" { t.Fatal("missing rule constant", id) }
        if _, present := item["status"]; !present { t.Fatal("missing responsibility status", id) }
        for _, old := range []string{"component_id", "release_owner", "compatibility_owner", "roles", "source_text", "source_policy", "source_rule_id", "Rule_state", "live_policy"} {
            if _, present := item[old]; present { t.Fatal("retired component owner field", id, old) }
        }
    }
}

func TestInvalidRIdBindingsAreRejected(t *testing.T) {
    catalog, validate := nativeCatalog(t)
    source, err := authoring.ReadDocument(catalog.Root, authoring.ProfilePath)
    if err != nil { t.Fatal(err) }
    for _, item := range []struct {
        name string
        mutate func(authoring.Document)
    }{
        {"unknown-condition", func(p authoring.Document) {
            p["component_bindings"].(map[string]any)["R-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"] = map[string]any{"rule":"EXAMPLE-RULE","status":"OBLIGATION","condition_name":"MISSING","condition_id":[]any{"C-ID-BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"}}
        }},
        {"invalid-key", func(p authoring.Document) {
            p["component_bindings"].(map[string]any)["not-an-R-ID"] = map[string]any{"rule":"EXAMPLE-RULE","status":"OBLIGATION"}
        }},
        {"missing-state", func(p authoring.Document) {
            p["component_bindings"].(map[string]any)["R-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"] = map[string]any{"rule":"EXAMPLE-RULE"}
        }},
        {"legacy-component-metadata", func(p authoring.Document) {
            p["component_bindings"].(map[string]any)["R-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"] = map[string]any{"rule":"EXAMPLE-RULE","status":"OBLIGATION","component_id":"official-context-engine"}
        }},
        {"legacy-live-policy", func(p authoring.Document) { p["component_bindings"].(map[string]any)["R-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"] = map[string]any{"rule":"EXAMPLE-RULE","status":"OBLIGATION","live_policy":[]any{"MPD-GOV-0001"}} }},
        {"retired-component-map", func(p authoring.Document) { p["component_map"] = map[string]any{"mode":"explicit"} }},
    } {
        t.Run(item.name, func(t *testing.T) {
            raw, err := json.Marshal(source)
            if err != nil { t.Fatal(err) }
            var p authoring.Document
            if err := json.Unmarshal(raw, &p); err != nil { t.Fatal(err) }
            item.mutate(p)
            if err := catalog.CheckLocalProfile(validate, p); err == nil { t.Fatal("invalid binding accepted") }
        })
    }
}

func TestRetiredComponentProjectionDoesNotGrantAuthority(t *testing.T) {
    catalog, validate := nativeCatalog(t)
    profile, err := authoring.ReadDocument(catalog.Root, authoring.ProfilePath)
    if err != nil { t.Fatal(err) }
    if result, err := catalog.MaterializeMain(validate, profile); err == nil || result != nil {
        t.Fatal("retired component projection accepted")
    }
}
