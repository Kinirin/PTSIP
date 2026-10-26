// Approved Policy_Status/Policy_version resources for native PTSIP VALIDATION.
// This adapter does not resolve $ref itself, infer schema authority, or depend
// on developer/automation's retired Go validator code.
package schema_check

import (
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
)

type StatusProbe struct {
 Status string `json:"status"`
 SchemaPath string `json:"schema_path"`
 PositiveValid bool `json:"positive_valid"`
 MissingVersionRejected bool `json:"missing_version_rejected"`
 LegacyNestedRejected bool `json:"legacy_nested_rejected"`
 InvalidStatusRejected bool `json:"invalid_status_rejected"`
 Problem string `json:"problem,omitempty"`
}

const publicPolicyStatusPath = ".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json"
const policyVersionPath = ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json"

// approvedResource reads exactly a file already in the approved-schema set.
// It does not search arbitrary paths or fetch URIs from the network.
func approvedResource(root,relative string) (map[string]any,error) {
 allowed:=false
 for _,path:=range approvedSchemaPaths {if path==relative{allowed=true;break}}
 if !allowed{return nil,fmt.Errorf("UNAPPROVED_SCHEMA_RESOURCE: %s",relative)}
 raw,err:=os.ReadFile(filepath.Join(root,filepath.FromSlash(relative)))
 if err!=nil{return nil,fmt.Errorf("APPROVED_SCHEMA_RESOURCE_UNAVAILABLE: %s: %w",relative,err)}
 var doc map[string]any
 if err:=json.Unmarshal(raw,&doc);err!=nil{return nil,fmt.Errorf("INVALID_APPROVED_SCHEMA_RESOURCE: %s: %w",relative,err)}
 id,ok:=doc["$id"].(string)
 if !ok||id==""{return nil,fmt.Errorf("APPROVED_SCHEMA_RESOURCE_ID_REQUIRED: %s",relative)}
 return doc,nil
}

// probeApprovedFlatPolicyStatus checks the current shared status record shape
// through the installed VALIDATION engine. It never grants policy conformance.
func probeApprovedFlatPolicyStatus(root string, e Engine) StatusProbe {
 report := StatusProbe{Status: "UNRESOLVED", SchemaPath: publicPolicyStatusPath}
 fail := func(err error) StatusProbe { report.Problem = err.Error(); return report }
 schema, err := approvedResource(root, publicPolicyStatusPath)
 if err != nil { return fail(err) }
 if schema["$id"] != "urn:ptsip:policy:Status:v2" {
  return fail(fmt.Errorf("SHARED_STATUS_SCHEMA_ID_MISMATCH"))
 }
 positive := map[string]any{
  "Policy_Status_Schema_ver": "2",
  "status": "APPROVED.Applying",
  "contract": "urn:ptsip:contract:policy:gov:responsibility:orchestration",
 }
 valid, err := nativeValidate(e, schema, positive, nil)
 if err != nil { return fail(err) }
 report.PositiveValid = valid

 missingVersion := map[string]any{
  "status": "APPROVED.Applying",
  "contract": "urn:ptsip:contract:policy:gov:responsibility:orchestration",
 }
 accepted, err := nativeValidate(e, schema, missingVersion, nil)
 if err != nil { return fail(err) }
 report.MissingVersionRejected = !accepted

 legacy := map[string]any{
  "Schema": map[string]any{"Policy_Status_version": map[string]any{"version":"1"}},
  "constant": map[string]any{
   "status": "APPROVED",
   "substatus": "Applying",
   "contract": "urn:ptsip:contract:policy:gov:responsibility:orchestration",
  },
 }
 accepted, err = nativeValidate(e, schema, legacy, nil)
 if err != nil { return fail(err) }
 report.LegacyNestedRejected = !accepted

 invalidStatus := map[string]any{
  "Policy_Status_Schema_ver": "2",
  "status": "APPROVED.NotDeclared",
  "contract": "urn:ptsip:contract:policy:gov:responsibility:orchestration",
 }
 accepted, err = nativeValidate(e, schema, invalidStatus, nil)
 if err != nil { return fail(err) }
 report.InvalidStatusRejected = !accepted

 if !report.PositiveValid || !report.MissingVersionRejected ||
  !report.LegacyNestedRejected || !report.InvalidStatusRejected {
  return fail(fmt.Errorf("SHARED_FLAT_STATUS_SCHEMA_UNEXPECTED_VERDICT"))
 }
 report.Status = "PASS"
 return report
}
