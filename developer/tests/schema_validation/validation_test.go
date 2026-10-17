package schema_validation

import (
 "encoding/json"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
)

func schemaCheck(t *testing.T, docs map[string]string) (map[string]any, error) {
 return schemaCheckWithMissing(t,docs,"")
}

func schemaCheckWithMissing(t *testing.T, docs map[string]string, missing string) (map[string]any, error) {
 t.Helper()
 root,err:=filepath.Abs(filepath.Join("..","..",".."))
 if err!=nil{t.Fatal(err)}
 fixture:=t.TempDir()
 dir:=filepath.Join(fixture,"developer","contracts","schemas")
 if err:=os.MkdirAll(dir,0755);err!=nil{t.Fatal(err)}
 for file,content:=range docs{
  dest:=filepath.Join(dir,file)
  if err:=os.MkdirAll(filepath.Dir(dest),0755);err!=nil{t.Fatal(err)}
  if err:=os.WriteFile(dest,[]byte(content),0644);err!=nil{t.Fatal(err)}
 }
 // Copy current approved contracts, not substitute legacy schemas or invented rules.
 for _,relative:=range []string{
  "developer/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json",
  "developer/contracts/schemas/Policy_Status/policy_status_constant.schema.json",
  "developer/contracts/lifecycle/Policy_version/Policy_version.schema.json",
 }{
  if relative==missing {continue}
  raw,err:=os.ReadFile(filepath.Join(root,filepath.FromSlash(relative)))
  if err!=nil{t.Fatal(err)}
  target:=filepath.Join(fixture,filepath.FromSlash(relative))
  if err:=os.MkdirAll(filepath.Dir(target),0755);err!=nil{t.Fatal(err)}
  if err:=os.WriteFile(target,raw,0644);err!=nil{t.Fatal(err)}
 }
 moduleDir:=filepath.Join(root,".ptsip","engine","developer")
 cmd:=exec.Command("go","-C",moduleDir,"run",".","--validator","schema-check","--root",fixture)
 stdout,err:=cmd.Output()
 if err!=nil{
  if _,ok:=err.(*exec.ExitError);!ok{t.Fatalf("validator could not run: %v",err)}
 }
 var result map[string]any
 if parseError:=json.Unmarshal(stdout,&result);parseError!=nil{
  t.Fatalf("validator must emit structured JSON even on failure: %v; stdout=%s",parseError,string(stdout))
 }
 return result,err
}

func TestInstalledValidationEngineAcceptsLimitedSchemaShape(t *testing.T){
 result,err:=schemaCheck(t,map[string]string{
  "valid.schema.json":`{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:valid","type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`,
 })
 if err!=nil{t.Fatalf("unexpected validation error: %v",err)}
 if result["status"]!="LIMITED_CHECK_PASS"{t.Fatalf("unexpected status: %v",result)}
 if result["policy_conformance"]!="NOT_ASSERTED" || result["full_json_schema_conformance"]!="NOT_ASSERTED"{
  t.Fatalf("limited engine check must not claim policy or full JSON Schema conformance: %v",result)
 }
}

func TestMalformedSchemaFailsClosed(t *testing.T){
 result,err:=schemaCheck(t,map[string]string{"malformed.schema.json":`{"$id":"urn:example:broken",}`})
 if err==nil || result["status"]!="FAIL"{t.Fatalf("malformed JSON must fail closed: %v, %v",err,result)}
 findings:=result["findings"].([]any)
 detected:=false
 for _,item:=range findings{
  row:=item.(map[string]any)
  if row["file"]=="developer/contracts/schemas/malformed.schema.json" && strings.Contains(row["problem"].(string),"INVALID_JSON") {detected=true}
 }
 if !detected { t.Fatalf("malformed fixture not rejected: %v",findings) }
}

func TestDuplicateSchemaIDFailsClosed(t *testing.T){
 result,err:=schemaCheck(t,map[string]string{
  "a.json":`{"$id":"urn:example:duplicate"}`,
  "b.json":`{"$id":"urn:example:duplicate"}`,
 })
 if err==nil || result["status"]!="FAIL"{t.Fatalf("duplicate identity must fail closed: %v, %v",err,result)}
}

func TestExternalRefsAreNotClaimedVerified(t *testing.T){
 result,err:=schemaCheck(t,map[string]string{
  "ref.json":`{"$id":"urn:example:reference","properties":{"linked":{"$ref":"urn:example:external"}}}`,
 })
 if err!=nil{t.Fatalf("external reference reporting should not pretend to resolve or reject it: %v",err)}
 if result["unverified_references"].(float64)<1{
  t.Fatalf("unresolved references must be explicitly reported: %v",result)
 }
 detected:=false
 for _,item:=range result["findings"].([]any){
  row:=item.(map[string]any)
  if row["file"]=="developer/contracts/schemas/ref.json" {
   refs:=row["unverified_refs"].([]any)
   if len(refs)==1 && refs[0]=="urn:example:external" {detected=true}
  }
 }
 if !detected {t.Fatalf("fixture external reference not explicitly reported: %v",result)}
}

func TestOrchestratorRejectsUnknownValidator(t *testing.T) {
 root,err:=filepath.Abs(filepath.Join("..","..",".."))
 if err!=nil {t.Fatal(err)}
 moduleDir:=filepath.Join(root,".ptsip","engine","developer")
 cmd:=exec.Command("go","-C",moduleDir,"run",".","--validator","unknown-validator","--root",t.TempDir())
 stdout,err:=cmd.Output()
 if err==nil {t.Fatalf("unregistered validator must fail closed; output=%s",stdout)}
 var result map[string]any
 if err:=json.Unmarshal(stdout,&result);err!=nil {
  t.Fatalf("unregistered validator must emit structured error: %v, output=%s",err,stdout)
 }
 if result["status"]!="UNRESOLVED" || result["reason"]!="UNREGISTERED_DEVELOPER_VALIDATOR" {
  t.Fatalf("unexpected failure semantics: %v",result)
 }
}


func TestPolicyStatusAndVersionContractsIncludedInEnginePath(t *testing.T) {
 result,err:=schemaCheck(t,map[string]string{"fixture.json":`{"$id":"urn:example:fixture"}`})
 if err!=nil || result["status"]!="LIMITED_CHECK_PASS" { t.Fatalf("engine path failed: %v %v",err,result) }
 required:=map[string]bool{
  "developer/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json":false,
  "developer/contracts/schemas/Policy_Status/policy_status_constant.schema.json":false,
  "developer/contracts/lifecycle/Policy_version/Policy_version.schema.json":false,
 }
 for _,item:=range result["findings"].([]any){
  row:=item.(map[string]any)
  name:=row["file"].(string)
  if _,ok:=required[name];ok {required[name]=row["status"]=="LIMITED_CHECK_PASS"}
 }
 for name,ok:=range required {if !ok{t.Errorf("approved contract not processed: %s (%v)",name,result)}}
 if result["policy_conformance"]!="NOT_ASSERTED" || result["full_json_schema_conformance"]!="NOT_ASSERTED"{
  t.Fatalf("structural check cannot assert policy conformance: %v",result)
 }
}

func TestMissingPolicyVersionContractFailsClosed(t *testing.T){
 result,err:=schemaCheckWithMissing(t,map[string]string{"fixture.json":`{"$id":"urn:example:fixture"}`},"developer/contracts/lifecycle/Policy_version/Policy_version.schema.json")
 if err==nil || result["status"]!="FAIL"{t.Fatalf("missing mandatory schema must fail: %v %v",err,result)}
 detected:=false
 for _,item:=range result["findings"].([]any){
  row:=item.(map[string]any)
  if row["file"]=="developer/contracts/lifecycle/Policy_version/Policy_version.schema.json" && row["problem"]=="MISSING_APPROVED_SCHEMA"{detected=true}
 }
 if !detected{t.Fatalf("missing contract not reported: %v",result)}
}

func TestDuplicateSchemaIDAcrossRootsFailsClosed(t *testing.T){
 result,err:=schemaCheck(t,map[string]string{
  "duplicate.json":`{"$id":"urn:ptsip:policy:Status-version:v1"}`,
 })
 if err==nil || result["status"]!="FAIL"{t.Fatalf("duplicate logical schema id must fail: %v %v",err,result)}
 detected:=false
 for _,item:=range result["findings"].([]any){
  row:=item.(map[string]any)
  if problem,ok:=row["problem"].(string);ok&&strings.Contains(problem,"DUPLICATE_SCHEMA_ID_WITH"){detected=true}
 }
 if !detected{t.Fatalf("cross-root duplicate not detected: %v",result)}
}
