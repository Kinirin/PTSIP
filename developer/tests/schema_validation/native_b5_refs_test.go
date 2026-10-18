// Diagnostic policy-schema tests use only installed official PTSIP b5 VALIDATION.
// They do not import or replicate legacy developer/automation validators.
package schema_validation

import (
  "bytes"
  "encoding/json"
  "os"
  "os/exec"
  "path/filepath"
  "testing"
)

type b5NativeResult struct {
 Status string `json:"status"`
 Error any `json:"error"`
 Output *struct {
  Valid bool `json:"valid"`
  Diagnostics []string `json:"diagnostics"`
  Verdict string `json:"ptsip_conformance_verdict"`
  ResourcesUsed []map[string]any `json:"schema_resources_used"`
 } `json:"output"`
}

func b5ReadApproved(t *testing.T, root, relative string) map[string]any {
 t.Helper()
 raw,err:=os.ReadFile(filepath.Join(root,filepath.FromSlash(relative)))
 if err!=nil{t.Fatal(err)}
 var schema map[string]any
 if err:=json.Unmarshal(raw,&schema);err!=nil{t.Fatal(err)}
 if _,ok:=schema["$id"].(string);!ok{t.Fatalf("approved schema %s missing $id",relative)}
 return schema
}

func b5NativeCall(t *testing.T, engine nativeTrialEngine, schema, instance any, resources []any) b5NativeResult {
 t.Helper()
 request:=map[string]any{
  "engine_id":engine.EngineID,
  "engine_version":engine.EngineVersion,
  "contract_id":engine.ContractID,
  "contract_version":engine.ContractVersion,
  "operation_id":"validation.schema",
  "input":map[string]any{"schema":schema,"instance":instance,"schema_resources":resources},
 }
 raw,err:=json.Marshal(request);if err!=nil{t.Fatal(err)}
 cmd:=exec.Command(engine.Executable);cmd.Stdin=bytes.NewReader(raw)
 output,runErr:=cmd.CombinedOutput()
 var response b5NativeResult
 if err:=json.Unmarshal(output,&response);err!=nil{t.Fatalf("official native response invalid JSON: %v: %s",err,output)}
 t.Logf("B5_NATIVE_REF status=%s valid=%v engine_error=%v diagnostics=%v used_resources=%d exit=%v",response.Status,response.Output!=nil&&response.Output.Valid,response.Error,func()[]string{if response.Output==nil{return nil};return response.Output.Diagnostics}(),func()int{if response.Output==nil{return 0};return len(response.Output.ResourcesUsed)}(),runErr)
 if runErr!=nil||response.Status!="PASS"||response.Error!=nil||response.Output==nil||response.Output.Verdict!="NOT_ASSERTED" {
  t.Fatalf("official b5 validation.schema failed: exit=%v response=%s",runErr,output)
 }
 return response
}

func b5InstalledEngine(t *testing.T) nativeTrialEngine {
 t.Helper()
 python:=os.Getenv("PTSIP_PYTHON");if python==""{python="python"}
 data,err:=exec.Command(python,"-c",nativeTrialDiscovery).CombinedOutput()
 if err!=nil{t.Fatalf("installed official b5 VALIDATION missing: %v: %s",err,data)}
 var engine nativeTrialEngine
 if err:=json.Unmarshal(data,&engine);err!=nil{t.Fatal(err)}
 if engine.PackageVersion!="0.3.8b5"||engine.EngineVersion!="0.3.3"||engine.ContractVersion!="4.0.0"{
  t.Fatalf("expected exact b5 engine and capability, got package=%s engine=%s capability=%s",engine.PackageVersion,engine.EngineVersion,engine.ContractVersion)
 }
 return engine
}

func TestB5ApprovedInternalPolicyStatusReferences(t *testing.T){
 engine:=b5InstalledEngine(t)
 root,err:=filepath.Abs(filepath.Join("..","..",".."));if err!=nil{t.Fatal(err)}
 status:=b5ReadApproved(t,root,"developer/contracts/schemas/Policy_Status/policy_status_constant.schema.json")
 resources:=[]any{
  b5ReadApproved(t,root,"developer/contracts/lifecycle/Policy_status/Policy_status_DRAFT.json"),
  b5ReadApproved(t,root,"developer/contracts/lifecycle/Policy_status/Policy_status_APPROVED.json"),
  b5ReadApproved(t,root,"developer/contracts/lifecycle/Policy_status/Policy_status_RETIRED.json"),
 }
 // Input values are test instances, not changes to policy authority.
 goodContract:="urn:ptsip:contract:policy:arch:identity:status"
 tests:=[]struct{name string;record map[string]any;want bool}{
  {"draft-default",map[string]any{"status":"DRAFT","substatus":"Default","contract":goodContract},true},
  {"approved-activate",map[string]any{"status":"APPROVED","substatus":"Activate","contract":goodContract},true},
  {"mismatched-status-substatus",map[string]any{"status":"APPROVED","substatus":"Reviewed","contract":goodContract},false},
  {"missing-required-contract",map[string]any{"status":"DRAFT","substatus":"Default"},false},
  {"invalid-contract-urn",map[string]any{"status":"DRAFT","substatus":"Default","contract":"unexpected"},false},
 }
 for _,test:=range tests{
  t.Run(test.name,func(t *testing.T){
   result:=b5NativeCall(t,engine,status,test.record,resources)
   if result.Output.Valid!=test.want{t.Errorf("validation decision mismatch: wanted=%v got=%v",test.want,result.Output.Valid)}
  })
 }
}

func TestB5ApprovedPublicPolicyStatusExternalAndInternalRefs(t *testing.T){
 engine:=b5InstalledEngine(t)
 root,err:=filepath.Abs(filepath.Join("..","..",".."));if err!=nil{t.Fatal(err)}
 parent:=b5ReadApproved(t,root,"developer/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json")
 var resources []any
 for _,p:=range []string{
  "developer/contracts/lifecycle/Policy_version/Policy_version.schema.json",
  "developer/contracts/schemas/Policy_Status/policy_status_constant.schema.json",
  "developer/contracts/lifecycle/Policy_status/Policy_status_DRAFT.json",
  "developer/contracts/lifecycle/Policy_status/Policy_status_APPROVED.json",
  "developer/contracts/lifecycle/Policy_status/Policy_status_RETIRED.json",
  "developer/contracts/schemas/Policy_Identity/Policy_Identity_version.schema.json",
  "developer/contracts/schemas/Authority/Authority_version.schema.json",
  "developer/contracts/schemas/Policy_Rules/Policy_Rules_version.schema.json",
 }{resources=append(resources,b5ReadApproved(t,root,p))}
 constant:=map[string]any{"status":"DRAFT","substatus":"Default","contract":"urn:ptsip:contract:policy:arch:identity:status"}
 valid:=map[string]any{"Schema":map[string]any{"Policy_Status_version":map[string]any{"version":"preview"}},"constant":constant}
 missingVersion:=map[string]any{"Schema":map[string]any{"Policy_Status_version":map[string]any{}},"constant":constant}
 missingConstant:=map[string]any{"Schema":map[string]any{"Policy_Status_version":map[string]any{"version":"preview"}}}
 for _,test:=range []struct{name string;input map[string]any;want bool}{
  {"valid-full-envelope",valid,true},
  {"missing-nested-version",missingVersion,false},
  {"missing-constant",missingConstant,false},
 }{
  t.Run(test.name,func(t *testing.T){
   result:=b5NativeCall(t,engine,parent,test.input,resources)
   if result.Output.Valid!=test.want{t.Errorf("approved full status envelope mismatch: wanted=%v got=%v",test.want,result.Output.Valid)}
  })
 }
}

func TestB5InternalRefWithNoNetworkOrFilesystemDiscovery(t *testing.T){
 engine:=b5InstalledEngine(t)
 // Only the closed inline schema is provided. Internal #/$defs resolution
 // must work through the b5 engine without external network or file access.
 schema:=map[string]any{
  "type":"object","required":[]string{"field"},
  "properties":map[string]any{"field":map[string]any{"$ref":"#/$defs/field"}},
  "$defs":map[string]any{"field":map[string]any{"type":"string"}},
 }
 for _,test:=range []struct{name string;value any;want bool}{
  {"internal-valid",map[string]any{"field":"text"},true},
  {"internal-type-violation",map[string]any{"field":42},false},
 }{
  t.Run(test.name,func(t *testing.T){
   result:=b5NativeCall(t,engine,schema,test.value,[]any{})
   if result.Output.Valid!=test.want{t.Errorf("inline internal ref mismatch: wanted=%v got=%v",test.want,result.Output.Valid)}
  })
 }
}

