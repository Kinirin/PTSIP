package v2

import (
    "fmt"
    "path"
    "regexp"
    "strings"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

// checkIndexedReferences is the referential-integrity stage after the selected
// registered JSON Schema check. It never infers semantic approval or ownership.
func checkIndexedReferences(document authoring.Document) error {
    root, ok := document["Policy_Rules"].(map[string]any)
    if !ok { return nil } // The registered structural schema reports this error.
    contracts, ok := root["contracts"].([]any)
    if !ok { return nil }
    for contractNumber, raw := range contracts {
        contract, ok := raw.(map[string]any)
        if !ok || contract["serialization"] != "INDEXED_SOURCE_BINDINGS" { continue }
        where := fmt.Sprintf("contracts[%d]", contractNumber)
        docs, err := indexedRecords(contract["source_documents"], "id", where+".source_documents")
        if err != nil { return err }
        for id, entry := range docs {
            name, _ := entry["source_ref"].(string)
            if name == "" || strings.HasPrefix(name,"/") || path.Clean(name) != name ||
                strings.ContainsAny(name, "\\\x00") || strings.Contains(name, "../") {
                return fmt.Errorf("INDEXED_INVALID_SOURCE_PATH: %s %s", where,id)
            }
            revision, _ := entry["revision_ref"].(string)
            if revision == "" { return fmt.Errorf("INDEXED_MISSING_SOURCE_REVISION: %s %s",where,id) }
            if strings.HasPrefix(revision,"git:") && !regexp.MustCompile(`^git:[0-9a-f]{40}$`).MatchString(revision) {
                return fmt.Errorf("INDEXED_INVALID_GIT_REVISION: %s %s",where,id)
            }
            if strings.HasPrefix(revision,"sha256:") && !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(revision) {
                return fmt.Errorf("INDEXED_INVALID_DIGEST_REVISION: %s %s",where,id)
            }
        }
        items, err := indexedRecords(contract["source_items"],"id",where+".source_items")
        if err != nil { return err }
        for id, entry := range items {
            documentID, _ := entry["document_id"].(string)
            if _, ok := docs[documentID]; !ok {
                return fmt.Errorf("INDEXED_UNDEFINED_DOCUMENT: %s %s -> %s",where,id,documentID)
            }
        }
        rules, err := indexedRecords(contract["rules"],"id",where+".rules")
        if err != nil { return err }
        for id, rule := range rules {
            src, _ := rule["source_item_id"].(string)
            if err := indexedItemRef(items,src,"rule",where+".rules."+id); err != nil { return err }
            if err := indexedBindingRefs(items,rule["authority_binding"],where+".rules."+id+".authority_binding"); err != nil { return err }
        }
        requirements, err := indexedRecords(contract["requirements"],"requirement_id",where+".requirements")
        if err != nil { return err }
        for id, req := range requirements {
            uses, ok := req["source_bindings"].([]any)
            if !ok { return fmt.Errorf("INDEXED_MISSING_SOURCE_BINDINGS: %s %s",where,id) }
            purposes := map[string]bool{}
            for _, rawUse := range uses {
                use, ok := rawUse.(map[string]any)
                if !ok { return fmt.Errorf("INDEXED_INVALID_SOURCE_BINDING: %s %s",where,id) }
                purpose, _ := use["purpose"].(string)
                if purposes[purpose] { return fmt.Errorf("INDEXED_DUPLICATE_PURPOSE: %s %s %s",where,id,purpose) }
                purposes[purpose] = true
                if err := indexedBindingRefs(items,use,where+".requirements."+id+"."+purpose); err != nil { return err }
                expectedRole := ""
                switch purpose {
                case "SCOPE": expectedRole = "scope"
                case "CONSTRAINT": expectedRole = "constraint"
                case "CRITERION": expectedRole = "criterion"
                }
                refs, _ := use["ref_ids"].([]any)
                for _, raw := range refs {
                    ref, _ := raw.(string)
                    if err := indexedItemRef(items,ref,expectedRole,where+".requirements."+id+"."+purpose); err != nil { return err }
                }
            }
            for _, key := range []string{"responsibility_binding","evidence_requirement_binding","relation_binding","judgement_binding"} {
                if err := indexedBindingRefs(items,req[key],where+".requirements."+id+"."+key); err != nil { return err }
            }
        }
        refs, _ := contract["historical_result_ref_ids"].([]any)
        for _, raw := range refs {
            ref, _ := raw.(string)
            if err := indexedItemRef(items,ref,"record",where+".historical_result_ref_ids"); err != nil { return err }
        }
    }
    return nil
}

func indexedRecords(raw any, key, where string) (map[string]map[string]any,error) {
    values, ok := raw.([]any)
    if !ok { return nil,fmt.Errorf("INDEXED_RECORDS_NOT_ARRAY: %s",where) }
    output := map[string]map[string]any{}
    for _, entry := range values {
        obj, ok := entry.(map[string]any)
        if !ok { return nil,fmt.Errorf("INDEXED_RECORD_NOT_OBJECT: %s",where) }
        id, _ := obj[key].(string)
        if id == "" { return nil,fmt.Errorf("INDEXED_EMPTY_ID: %s",where) }
        if _, exists := output[id]; exists { return nil,fmt.Errorf("INDEXED_DUPLICATE_ID: %s %s",where,id) }
        output[id]=obj
    }
    return output,nil
}
func indexedItemRef(items map[string]map[string]any, id, expectedRole, where string) error {
    item, ok := items[id]
    if !ok { return fmt.Errorf("INDEXED_UNDEFINED_SOURCE_ID: %s %s",where,id) }
    if expectedRole != "" && item["original_role"] != expectedRole {
        return fmt.Errorf("INDEXED_ROLE_MISMATCH: %s %s want %s",where,id,expectedRole)
    }
    return nil
}
func indexedBindingRefs(items map[string]map[string]any, raw any, where string) error {
    binding, ok := raw.(map[string]any)
    if !ok { return fmt.Errorf("INDEXED_BINDING_NOT_OBJECT: %s",where) }
    for _, name := range []string{"ref_ids","basis_ref_ids","value_ref_ids"} {
        values, ok := binding[name].([]any)
        if !ok { continue }
        for _, rawID := range values {
            id, ok := rawID.(string)
            if !ok { return fmt.Errorf("INDEXED_REF_NOT_STRING: %s %s",where,name) }
            if err := indexedItemRef(items,id,"",where+"."+name); err != nil { return err }
        }
    }
    if inputs, ok := binding["input_bindings"].([]any); ok {
        for _, input := range inputs {
            if err := indexedBindingRefs(items,input,where+".input_bindings"); err != nil { return err }
        }
    }
    return nil
}
