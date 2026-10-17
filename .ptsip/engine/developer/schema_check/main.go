// Developer-only adapter. Delegates limited schema-shape checks to installed PTSIP VALIDATION.
// Never grants policy conformance, execution authority, or a replacement JSON Schema engine.
package main

import (
 "bytes"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "io/fs"
 "os"
 "os/exec"
 "path/filepath"
 "sort"
 "strings"
 "time"
)

const discover=`import json, os, sysconfig
from importlib import metadata, resources
from pathlib import Path
version = metadata.version('PTSIP')
expected = os.getenv('PTSIP_TEST_PACKAGE_VERSION')
if expected and version != expected: raise RuntimeError(f'PTSIP version mismatch: {version} != {expected}')
base = resources.files('ptsip').joinpath('engines')
index = json.loads(base.joinpath('index.json').read_text(encoding='utf-8'))
entries = [x for x in index['official_engine_entries'] if x.get('kind')=='VALIDATION']
if len(entries)!=1: raise RuntimeError('VALIDATION engine not unambiguously registered')
e=entries[0]
def read(ref):
    prefix='src/ptsip/engines/'
    if not isinstance(ref,str) or not ref.startswith(prefix) or '..' in ref.split('/'):
        raise RuntimeError('invalid registered resource reference')
    return json.loads(base.joinpath(*ref[len(prefix):].split('/')).read_text(encoding='utf-8'))
manifest=read(e['manifest_ref'])
contract=read(e['contract_ref'])
bindings=read(e['execution_binding_ref'])['bindings']
if len(bindings)!=1: raise RuntimeError('ambiguous execution binding')
binding=bindings[0]
if not (e['engine_id']==manifest['engine_id']==binding['engine_id'] and e['version']==manifest['implementation_version']==binding['engine_version'] and e['capability_id']==contract['contract_id']==binding['capability_id'] and contract['version']==binding['contract_version']):
    raise RuntimeError('installed identity/version mismatch')
if binding.get('permission_grant') is not False: raise RuntimeError('binding cannot grant permission')
if contract.get('execution_permission_from_declaration') is not False: raise RuntimeError('declaration cannot grant permission')
if 'validation.schema' not in e['operations'] or not any(op['operation_id']=='validation.schema' for op in contract['operations']):
    raise RuntimeError('validation.schema missing')
name=binding['entrypoint']
if name!='ptsip-engine-validation': raise RuntimeError('unrecognized registered entrypoint')
exe=Path(sysconfig.get_path('scripts'))/(name+('.exe' if os.name=='nt' else ''))
if not exe.is_file(): raise RuntimeError('native engine executable missing')
print(json.dumps({'executable':str(exe),'engine_id':e['engine_id'],'engine_version':e['version'],'contract_id':contract['contract_id'],'contract_version':contract['version'],'package_version':version}))
`

type Engine struct {
 Executable string `json:"executable"`
 EngineID string `json:"engine_id"`
 EngineVersion string `json:"engine_version"`
 ContractID string `json:"contract_id"`
 ContractVersion string `json:"contract_version"`
 PackageVersion string `json:"package_version"`
}
type Finding struct {
 File string `json:"file"`
 Status string `json:"status"`
 Problem string `json:"problem,omitempty"`
 UnverifiedRefs []string `json:"unverified_refs,omitempty"`
 Warnings []string `json:"warnings,omitempty"`
}
type Report struct {
 Status string `json:"status"`
 Scope string `json:"scope"`
 Engine string `json:"engine"`
 PackageVersion string `json:"package_version,omitempty"`
 Total int `json:"total"`
 Failed int `json:"failed"`
 UnverifiedReferences int `json:"unverified_references"`
 PolicyConformance string `json:"policy_conformance"`
 FullJSONSchemaConformance string `json:"full_json_schema_conformance"`
 Findings []Finding `json:"findings"`
}

func installed() (Engine,error) {
 var e Engine
 python:=os.Getenv("PTSIP_PYTHON"); if python=="" {python="python"}
 raw,err:=exec.Command(python,"-c",discover).CombinedOutput()
 if err!=nil {return e,fmt.Errorf("installed engine discovery failed: %v: %s",err,strings.TrimSpace(string(raw)))}
 if err=json.Unmarshal(raw,&e);err!=nil{return e,err}
 out,err:=exec.Command(e.Executable,"--manifest").Output()
 if err!=nil{return e,fmt.Errorf("native manifest failed: %w",err)}
 var m struct {EngineID string `json:"engine_id"`; Version string `json:"implementation_version"`}
 if err=json.Unmarshal(out,&m);err!=nil{return e,err}
 if m.EngineID!=e.EngineID || m.Version!=e.EngineVersion {return e,errors.New("native manifest differs from installed registry")}
 return e,nil
}

// A deliberately LIMITED, inline shape check. It is not the JSON Schema metaschema.
var inlineShape=map[string]any{
 "type":"object", "required":[]string{"$id"},
 "properties":map[string]any{
   "$id":map[string]any{"type":"string","minLength":1},
   "$schema":map[string]any{"type":"string","minLength":1},
   "properties":map[string]any{"type":"object"},
   "$defs":map[string]any{"type":"object"},
   "required":map[string]any{"type":"array","items":map[string]any{"type":"string"}},
 },
 "additionalProperties":true,
}

func nativeCheck(e Engine,doc any) error {
 request:=map[string]any{
  "engine_id":e.EngineID,"engine_version":e.EngineVersion,
  "contract_id":e.ContractID,"contract_version":e.ContractVersion,
  "operation_id":"validation.schema",
  "input":map[string]any{"schema":inlineShape,"instance":doc},
 }
 data,_:=json.Marshal(request)
 cmd:=exec.Command(e.Executable);cmd.Stdin=bytes.NewReader(data)
 var stdout,stderr bytes.Buffer;cmd.Stdout=&stdout;cmd.Stderr=&stderr
 done:=make(chan error,1);go func(){done<-cmd.Run()}()
 select {
 case err:= <-done:
  if err!=nil{return fmt.Errorf("native validation execution failed: %v: %s %s",err,stdout.String(),stderr.String())}
 case <-time.After(20*time.Second):
  if cmd.Process!=nil{_ =cmd.Process.Kill()};<-done
  return errors.New("native invocation timeout")
 }
 var result struct {
  Status string `json:"status"`
  Error any `json:"error"`
  NormativeAuthority bool `json:"normative_authority"`
  ExecutionAuthorized bool `json:"execution_authorized"`
  Output struct {
   Valid bool `json:"valid"`
   Diagnostics []string `json:"diagnostics"`
   Verdict string `json:"ptsip_conformance_verdict"`
  } `json:"output"`
 }
 if err:=json.Unmarshal(stdout.Bytes(),&result);err!=nil{return fmt.Errorf("native response not JSON: %w",err)}
 if result.Status!="PASS" || result.Error!=nil || result.NormativeAuthority || result.ExecutionAuthorized || result.Output.Verdict!="NOT_ASSERTED"{
  return fmt.Errorf("native contract rejected limited check: %s",strings.TrimSpace(stdout.String()))
 }
 if !result.Output.Valid{return fmt.Errorf("engine rejected limited schema shape: %v",result.Output.Diagnostics)}
 return nil
}

func collectRefs(v any,refs *[]string){
 switch x:=v.(type){
 case map[string]any:
  for key,value:=range x{
   if key=="$ref"{if s,ok:=value.(string);ok{*refs=append(*refs,s)}}
   collectRefs(value,refs)
  }
 case []any:
  for _,item:=range x{collectRefs(item,refs)}
 }
}

func check(root string)(Report,int){
 report:=Report{Status:"FAIL",Scope:"LIMITED_SCHEMA_JSON_AND_ENGINE_SHAPE_CHECK",
  Engine:"INSTALLED_PTSIP_VALIDATION",PolicyConformance:"NOT_ASSERTED",
  FullJSONSchemaConformance:"NOT_ASSERTED",Findings:[]Finding{}}
 e,err:=installed()
 if err!=nil{report.Failed++;report.Findings=append(report.Findings,Finding{File:"<installed-VALIDATION>",Status:"FAIL",Problem:err.Error()});return report,2}
 report.PackageVersion=e.PackageVersion
 folder:=filepath.Join(root,"developer","contracts","schemas")
 var paths []string
 err=filepath.WalkDir(folder,func(path string,item fs.DirEntry,walkError error)error{
  if walkError!=nil{return walkError}
  if !item.IsDir() && strings.HasSuffix(item.Name(),".json"){paths=append(paths,path)}
  return nil
 })
 if err!=nil||len(paths)==0{
  report.Failed++;report.Findings=append(report.Findings,Finding{File:"developer/contracts/schemas",Status:"FAIL",Problem:"schema directory missing or empty"})
  return report,2
 }
 sort.Strings(paths);ids:=map[string]string{}
 for _,path:=range paths{
  relative,_:=filepath.Rel(root,path);relative=filepath.ToSlash(relative)
  f:=Finding{File:relative,Status:"LIMITED_CHECK_PASS"};report.Total++
  data,readError:=os.ReadFile(path)
  var doc any
  if readError==nil{readError=json.Unmarshal(data,&doc)}
  if readError!=nil{f.Status="FAIL";f.Problem="INVALID_JSON: "+readError.Error()
  }else if obj,ok:=doc.(map[string]any);!ok{
   f.Status="FAIL";f.Problem="JSON_SCHEMA_ROOT_NOT_OBJECT"
  }else{
   if id,valid:=obj["$id"].(string);valid&&id!=""{
    if prior,found:=ids[id];found{f.Status="FAIL";f.Problem="DUPLICATE_SCHEMA_ID_WITH: "+prior
    }else{ids[id]=relative}
   }
   if len(obj)<=2{f.Warnings=append(f.Warnings,"UNCONSTRAINED_OR_PLACEHOLDER_SCHEMA")}
   collectRefs(obj,&f.UnverifiedRefs)
   sort.Strings(f.UnverifiedRefs)
   report.UnverifiedReferences+=len(f.UnverifiedRefs)
   if f.Status!="FAIL"{
    if error:=nativeCheck(e,doc);error!=nil{f.Status="FAIL";f.Problem=error.Error()}
   }
  }
  if f.Status=="FAIL"{report.Failed++}
  report.Findings=append(report.Findings,f)
 }
 if report.Failed==0{report.Status="LIMITED_CHECK_PASS"}
 if report.Failed>0{return report,2}
 return report,0
}

func main(){
 root:=flag.String("root",".","repository root; read-only")
 flag.Parse()
 absolute,err:=filepath.Abs(*root)
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 report,code:=check(absolute)
 encoder:=json.NewEncoder(os.Stdout);encoder.SetIndent("","  ")
 if err:=encoder.Encode(report);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 os.Exit(code)
}
