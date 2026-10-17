// Findings and limited schema-corpus inspection for the developer tool.
// A limited check is never a JSON Schema metaschema or policy-conformance PASS.
package schema_check

import (
    "encoding/json"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

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

func Run(root string)(Report,int){
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
