package v2_test

import (
    "encoding/json"
    "strings"
    "testing"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
    entry "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractentry"
    verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
    v2 "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/v2"
)

const changeTarget = "developer/policy/CHANGE/MPD-CHANGE-0001.yaml"

func indexedCopy(t *testing.T, d authoring.Document) authoring.Document {
    t.Helper()
    raw, err := json.Marshal(d)
    if err != nil { t.Fatal(err) }
    var clone authoring.Document
    if err := json.Unmarshal(raw, &clone); err != nil { t.Fatal(err) }
    return clone
}
func indexedContract(t *testing.T, doc authoring.Document) map[string]any {
    t.Helper()
    rules := doc["Policy_Rules"].(map[string]any)
    return rules["contracts"].([]any)[0].(map[string]any)
}

func TestChange0001RegisteredV2SelectionAndStructure(t *testing.T) {
    root := repoRoot(t)
    selected, err := entry.Resolve(root,changeTarget)
    if err != nil { t.Fatalf("policy identity selection: %v",err) }
    if selected.PolicyKind != "PTSIP_DEVELOPER_POLICY" || selected.RootFamily != "CHANGE" ||
        selected.ContractRefs["public_schema"] != ".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json" ||
        selected.ContractRefs["selection_schema"] != ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json" {
        t.Fatalf("wrong selected contract: %+v", selected)
    }
    doc, err := authoring.ReadDocument(root,changeTarget)
    if err != nil { t.Fatalf("block YAML parse: %v",err) }
    c := indexedContract(t,doc)
    if c["serialization"] != "INDEXED_SOURCE_BINDINGS" { t.Fatal("indexed contract not selected") }
    if doc["Policy_Status"].(map[string]any)["status"] != "DRAFT.Reviewed" { t.Fatal("approval state changed") }
    if c["Rule_ver"] != "1" { t.Fatal("source rule version changed") }
    if len(c["source_items"].([]any)) != 19 { t.Fatal("19 source items were not retained") }
    if len(c["requirements"].([]any)) != 4 { t.Fatal("trial requirement grouping changed unexpectedly") }
    validator, err := verification.InstalledValidator()
    if err != nil { t.Fatalf("installed official PTSIP validator missing: %v",err) }
    if err := v2.Check(root,doc,validator); err != nil { t.Fatalf("registered v2 verification failed: %v",err) }
}

func TestChange0001IndexedReferenceCounterexamples(t *testing.T) {
    root := repoRoot(t)
    original, err := authoring.ReadDocument(root,changeTarget)
    if err != nil { t.Fatal(err) }
    validator, err := verification.InstalledValidator()
    if err != nil { t.Fatalf("official native validator missing: %v",err) }
    for _, tc := range []struct{
        name string
        update func(map[string]any)
        expected string
    }{
        {"duplicate-source-id",func(c map[string]any){
            xs:=c["source_items"].([]any)
            c["source_items"]=append(xs,xs[0])
        },""},
        {"missing-document-id",func(c map[string]any){
            c["source_items"].([]any)[0].(map[string]any)["document_id"]="MISSING_DOC"
        },"INDEXED_UNDEFINED_DOCUMENT"},
        {"unresolved-reference",func(c map[string]any){
            req:=c["requirements"].([]any)[0].(map[string]any)
            req["source_bindings"].([]any)[0].(map[string]any)["ref_ids"]=[]any{"NO_SUCH_SOURCE"}
        },"INDEXED_UNDEFINED_SOURCE_ID"},
        {"wrong-source-purpose",func(c map[string]any){
            req:=c["requirements"].([]any)[0].(map[string]any)
            req["source_bindings"].([]any)[2].(map[string]any)["ref_ids"]=[]any{"I11"}
        },"INDEXED_ROLE_MISMATCH"},
        {"invalid-revision",func(c map[string]any){
            c["source_documents"].([]any)[0].(map[string]any)["revision_ref"]="git:bad"
        },"INDEXED_INVALID_GIT_REVISION"},
        {"duplicate-purpose",func(c map[string]any){
            req:=c["requirements"].([]any)[0].(map[string]any)
            list:=req["source_bindings"].([]any)
            req["source_bindings"]=append(list,list[0])
        },""},
        {"empty-without-basis",func(c map[string]any){
            req:=c["requirements"].([]any)[0].(map[string]any)
            req["source_bindings"].([]any)[0]=map[string]any{"purpose":"MEANING","state":"CONFIRMED_EMPTY"}
        },""},
        {"not-applicable-without-basis",func(c map[string]any){
            req:=c["requirements"].([]any)[0].(map[string]any)
            req["source_bindings"].([]any)[0]=map[string]any{"purpose":"MEANING","state":"NOT_APPLICABLE"}
        },""},
    } {
        t.Run(tc.name,func(t *testing.T){
            doc:=indexedCopy(t,original)
            tc.update(indexedContract(t,doc))
            err:=v2.Check(root,doc,validator)
            if err==nil { t.Fatal("invalid reference contract accepted") }
            if tc.expected!="" && !strings.Contains(err.Error(),tc.expected) {
                t.Fatalf("expected %s, got %v",tc.expected,err)
            }
        })
    }
}

func TestChange0001ManyToManyRIDsRemainStructuralOnly(t *testing.T) {
    root:=repoRoot(t)
    doc, err:=authoring.ReadDocument(root,changeTarget)
    if err!=nil { t.Fatal(err) }
    c:=indexedContract(t,doc)
    req:=c["requirements"].([]any)
    one:=req[0].(map[string]any)
    two:=req[1].(map[string]any)
    one["responsibility_binding"]=map[string]any{
        "state":"DECLARED", "basis_ref_ids":[]any{"I06"},
        "responsibility_refs":[]any{"R-ID-C615A4E8BF5F4A0597CD28E6A5B79011","R-ID-E29A03BD454B4BB2A4074F16C587AD62"},
    }
    two["responsibility_binding"]=map[string]any{
        "state":"DECLARED", "basis_ref_ids":[]any{"I07"},
        "responsibility_refs":[]any{"R-ID-C615A4E8BF5F4A0597CD28E6A5B79011"},
    }
    validator,err:=verification.InstalledValidator()
    if err!=nil { t.Fatalf("official validator missing: %v",err) }
    if err:=v2.Check(root,doc,validator); err!=nil { t.Fatalf("many-to-many binding structurally rejected: %v",err) }
}
