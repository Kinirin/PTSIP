package ctrl_admission

import (
    "os"
    "path/filepath"
    "testing"
)
func TestCTRLNormativePolicyActiveAndSafe(t *testing.T) {
    root,err:=filepath.Abs("../../../..")
    if err!=nil{t.Fatal(err)}
    if err=Verify(root);err!=nil{t.Fatal(err)}
    report,code:=Run(root)
    if code!=0{t.Fatalf("independent policy check refused: %v",report)}
    m:=report.(map[string]any)
    if m["publication_authorized"]!=false || m["per_release_evidence"]!="NOT_ASSERTED"{t.Fatal("test pass cannot grant publication")}
}
func TestCTRLPolicySchemaClosedAndTamperBlocked(t *testing.T) {
    root,err:=filepath.Abs("../../../..");if err!=nil{t.Fatal(err)}
    schema,err:=data(root,schemaRef);if err!=nil{t.Fatal(err)}
    policy,err:=data(root,policyRef);if err!=nil{t.Fatal(err)}
    if err=validateSchema(schema,policy,"ctrl");err!=nil{t.Fatal(err)}
    copyPolicy:=map[string]any{};for k,v:=range policy{copyPolicy[k]=v}
    copyPolicy["authorization"]=map[string]any{"mode":"UNCONDITIONAL"}
    if err=validateSchema(schema,copyPolicy,"ctrl");err==nil{t.Fatal("unconditional publication approved")}
    copyPolicy["authorization"]=policy["authorization"]
    copyPolicy["unknown_implicit_allow"]=true
    if err=validateSchema(schema,copyPolicy,"ctrl");err==nil{t.Fatal("unknown policy field approved")}
    tmp:=t.TempDir()
    if err=Verify(tmp);err==nil{t.Fatal("missing policy/approval accepted")}
    if _,err=os.Stat(filepath.Join(root,policyRef));err!=nil{t.Fatal(err)}
}
