package schema_validation

import (
 "bytes"
 "encoding/json"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "testing"
)

type nativeTrialEngine struct {
 Executable string `json:"executable"`
 EngineID string `json:"engine_id"`
 EngineVersion string `json:"engine_version"`
 ContractID string `json:"contract_id"`
 ContractVersion string `json:"contract_version"`
 PackageVersion string `json:"package_version"`
}

// Test-only discovery: the installed official PTSIP engine registry is the source.
// This file never calls or imports developer/automation or a retired validator.
const nativeTrialDiscovery = `import json, os, sysconfig
from pathlib import Path
from importlib import metadata, resources
base=resources.files('ptsip').joinpath('engines')
idx=json.loads(base.joinpath('index.json').read_text(encoding='utf-8'))
entries=[e for e in idx['official_engine_entries'] if e.get('kind')=='VALIDATION']
if len(entries)!=1: raise RuntimeError('ambiguous installed VALIDATION engine')
e=entries[0]
ref=e['contract_ref']
if not ref.startswith('src/ptsip/engines/'): raise RuntimeError('unexpected contract reference')
contract=json.loads(base.joinpath(*ref.removeprefix('src/ptsip/engines/').split('/')).read_text(encoding='utf-8'))
name='ptsip-engine-validation'+('.exe' if os.name=='nt' else '')
exe=Path(sysconfig.get_path('scripts'))/name
if not exe.is_file(): raise RuntimeError('installed native engine missing')
print(json.dumps({'executable':str(exe),'engine_id':e['engine_id'],
 'engine_version':e['version'],'contract_id':contract['contract_id'],
 'contract_version':contract['version'],'package_version':metadata.version('PTSIP')}))
`

type nativeTrialResult struct {
 Status string `json:"status"`
 Error any `json:"error"`
 Output struct {
  Valid bool `json:"valid"`
  Diagnostics []string `json:"diagnostics"`
  Verdict string `json:"ptsip_conformance_verdict"`
 } `json:"output"`
}

func nativeTrialCall(e nativeTrialEngine, schema any, instance any) (nativeTrialResult,string,error) {
 request:=map[string]any{
  "engine_id":e.EngineID,"engine_version":e.EngineVersion,
  "contract_id":e.ContractID,"contract_version":e.ContractVersion,
  "operation_id":"validation.schema",
  "input":map[string]any{"schema":schema,"instance":instance},
 }
 data,err:=json.Marshal(request)
 if err!=nil{return nativeTrialResult{},"",err}
 command:=exec.Command(e.Executable)
 command.Stdin=bytes.NewReader(data)
 combined,err:=command.CombinedOutput()
 var response nativeTrialResult
 parseErr:=json.Unmarshal(combined,&response)
 if parseErr!=nil {return response,string(combined),fmt.Errorf("engine exit=%v JSON=%v",err,parseErr)}
 return response,string(combined),err
}

// The inline tests must match the approved version-string contract.
// External refs are exploratory and never assert policy conformance.
func TestOfficialNativePolicySchemaTrial(t *testing.T) {
 python:=os.Getenv("PTSIP_PYTHON")
 if python==""{python="python"}
 data,err:=exec.Command(python,"-c",nativeTrialDiscovery).CombinedOutput()
 if err!=nil{t.Fatalf("NATIVE_ENGINE_UNAVAILABLE: %v: %s",err,data)}
 var engine nativeTrialEngine
 if err=json.Unmarshal(data,&engine);err!=nil{t.Fatal(err)}
 root,err:=filepath.Abs(filepath.Join("..","..",".."))
 if err!=nil{t.Fatal(err)}
 load:=func(relative string) map[string]any{
  raw,err:=os.ReadFile(filepath.Join(root,filepath.FromSlash(relative)))
  if err!=nil{t.Fatal(err)}
  var document map[string]any
  if err=json.Unmarshal(raw,&document);err!=nil{t.Fatal(err)}
  return document
 }
 version:=load(".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json")
 for _,test:=range []struct{name string;instance map[string]any;expected bool}{
  {"valid-version",map[string]any{"version":"1.0"},true},
  {"undecided-format-string",map[string]any{"version":"preview-beta"},true},
  {"missing-version",map[string]any{},false},
  {"numeric-version",map[string]any{"version":7},false},
 }{
  t.Run(test.name,func(t *testing.T){
   response,raw,err:=nativeTrialCall(engine,version,test.instance)
   if err!=nil || response.Status!="PASS" ||
     response.Output.Verdict!="NOT_ASSERTED" || response.Output.Valid!=test.expected {
    t.Fatalf("installed official validation.schema mismatch: error=%v result=%s",err,raw)
   }
  })
 }
 status:=map[string]any{
  "status":"DRAFT","substatus":"Default",
  "contract":"urn:ptsip:contract:policy:arch:identity:status",
 }
 for _,probe:=range []struct{name string;schema any;instance any}{
  {"status-external-ref",
   load(".ptsip/contracts/schemas/Policy_Status/policy_status_constant.schema.json"),status},
  {"public-status-external-ref",
   load(".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json"),
   map[string]any{
    "Schema":map[string]any{"Policy_Status_version":map[string]any{"version":"1.0"}},
    "constant":status,
   }},
 }{
  t.Run(probe.name,func(t *testing.T){
   response,raw,err:=nativeTrialCall(engine,probe.schema,probe.instance)
   report:=map[string]any{
    "probe":probe.name,"package_version":engine.PackageVersion,
    "engine_version":engine.EngineVersion,"status":response.Status,
    "valid":response.Output.Valid,"diagnostics":response.Output.Diagnostics,
    "external_ref_conformance":"NOT_ASSERTED",
   }
   if err!=nil{report["native_error"]=err.Error()}
   if response.Error!=nil{report["engine_error"]=response.Error}
   summary,_:=json.Marshal(report)
   t.Logf("NATIVE_EXTERNAL_REF_TRIAL %s raw=%s",summary,raw)
  })
 }
}
