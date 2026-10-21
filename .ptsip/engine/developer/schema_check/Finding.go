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
 ReferenceProbe *ReferenceProbe `json:"reference_probe,omitempty"`
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

// approvedSchemaPaths are the current owner-approved Policy_Status and
// Policy_version schema files. Requiring them detects missing contracts rather
// than accidentally reporting a successful partial directory scan.
var approvedSchemaPaths = []string{
    ".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json",
    ".ptsip/contracts/schemas/Policy_Status/policy_status_constant.schema.json",
    ".ptsip/contracts/lifecycle/Policy_version/Policy_version.schema.json",
}

var schemaScanRoots = []string{
    ".ptsip/contracts/schemas",
    ".ptsip/contracts/lifecycle/Policy_version",
}

func schemaFiles(root string) ([]string, []Finding) {
    paths := []string{}
    findings := []Finding{}
    found := map[string]bool{}
    for _, relativeRoot := range schemaScanRoots {
        folder := filepath.Join(root, filepath.FromSlash(relativeRoot))
        count := 0
        err := filepath.WalkDir(folder, func(path string, item fs.DirEntry, walkError error) error {
            if walkError != nil {
                return walkError
            }
            if !item.IsDir() && strings.HasSuffix(item.Name(), ".json") {
                relative, relError := filepath.Rel(root, path)
                if relError != nil {
                    return relError
                }
                found[filepath.ToSlash(relative)] = true
                paths = append(paths, path)
                count++
            }
            return nil
        })
        if err != nil {
            findings = append(findings, Finding{File: relativeRoot, Status: "FAIL", Problem: "SCHEMA_ROOT_UNAVAILABLE: " + err.Error()})
        } else if count == 0 {
            findings = append(findings, Finding{File: relativeRoot, Status: "FAIL", Problem: "SCHEMA_ROOT_EMPTY"})
        }
    }
    for _, relative := range approvedSchemaPaths {
        if !found[relative] {
            findings = append(findings, Finding{File: relative, Status: "FAIL", Problem: "MISSING_APPROVED_SCHEMA"})
        }
    }
    sort.Strings(paths)
    return paths, findings
}

func Run(root string)(Report,int){
 report:=Report{Status:"FAIL",Scope:"LIMITED_SCHEMA_JSON_AND_ENGINE_SHAPE_CHECK",
  Engine:"INSTALLED_PTSIP_VALIDATION",PolicyConformance:"NOT_ASSERTED",
  FullJSONSchemaConformance:"NOT_ASSERTED",Findings:[]Finding{}}
 e,err:=installed()
 if err!=nil{report.Failed++;report.Findings=append(report.Findings,Finding{File:"<installed-VALIDATION>",Status:"FAIL",Problem:err.Error()});return report,2}
 report.PackageVersion=e.PackageVersion
 paths, problems := schemaFiles(root)
 report.Failed += len(problems)
 report.Findings = append(report.Findings, problems...)
 ids:=map[string]string{}
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
 // Reference probe is independent of the limited document-shape scan.
 // It proves only the approved Policy_Status_version -> Policy_version link.
 // Full policy status reference verification is separately exercised by the official b5 engine integration tests.
 if len(problems)==0 {
  probe:=probeApprovedPolicyVersionReference(root,e)
  report.ReferenceProbe=&probe
  if probe.Status!="PASS" {report.Failed++}
 }
 if report.Failed==0{report.Status="LIMITED_CHECK_PASS"}
 if report.Failed>0{return report,2}
 return report,0
}
