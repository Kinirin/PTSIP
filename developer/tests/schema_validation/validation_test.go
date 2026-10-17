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
 t.Helper()
 root,err:=filepath.Abs(filepath.Join("..","..",".."))
 if err!=nil{t.Fatal(err)}
 fixture:=t.TempDir()
 dir:=filepath.Join(fixture,"developer","contracts","schemas")
 if err:=os.MkdirAll(dir,0755);err!=nil{t.Fatal(err)}
 for file,content:=range docs{
  if err:=os.WriteFile(filepath.Join(dir,file),[]byte(content),0644);err!=nil{t.Fatal(err)}
 }
 mainFile:=filepath.Join(root,".ptsip","engine","developer","schema_check","main.go")
 cmd:=exec.Command("go","run",mainFile,"--root",fixture)
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
 if !strings.Contains(findings[0].(map[string]any)["problem"].(string),"INVALID_JSON"){
  t.Fatalf("unexpected error: %v",findings)
 }
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
 if result["unverified_references"].(float64)!=1{
  t.Fatalf("unresolved references must be explicitly reported: %v",result)
 }
}
