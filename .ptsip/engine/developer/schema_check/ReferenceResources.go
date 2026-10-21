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

type ReferenceProbe struct {
 Status string `json:"status"`
 SchemaPath string `json:"schema_path"`
 Reference string `json:"reference,omitempty"`
 ResourceIDs []string `json:"resource_ids,omitempty"`
 PositiveValid bool `json:"positive_valid"`
 MissingVersionRejected bool `json:"missing_version_rejected"`
 Problem string `json:"problem,omitempty"`
}

const publicPolicyStatusPath = ".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json"
const policyVersionPath = ".ptsip/contracts/lifecycle/Policy_version/Policy_version.schema.json"

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

// probeApprovedPolicyVersionReference uses an existing $ref field from the
// approved Public_Policy_Status schema, not a synthesized policy constraint.
// It passes only the Policy_version schema to schema_resources: passing the
// Policy_Status constant would fail on its currently unsupported internal $ref.
func probeApprovedPolicyVersionReference(root string,e Engine) ReferenceProbe {
 report:=ReferenceProbe{Status:"UNRESOLVED",SchemaPath:publicPolicyStatusPath}
 fail:=func(err error) ReferenceProbe {report.Problem=err.Error();return report}
 parent,err:=approvedResource(root,publicPolicyStatusPath)
 if err!=nil{return fail(err)}
 version,err:=approvedResource(root,policyVersionPath)
 if err!=nil{return fail(err)}
 props,ok:=parent["properties"].(map[string]any)
 if !ok{return fail(fmt.Errorf("POLICY_STATUS_PROPERTIES_MISSING"))}
 schema,ok:=props["Schema"].(map[string]any)
 if !ok{return fail(fmt.Errorf("POLICY_STATUS_SCHEMA_FIELD_MISSING"))}
 schemaProps,ok:=schema["properties"].(map[string]any)
 if !ok{return fail(fmt.Errorf("POLICY_STATUS_SCHEMA_PROPERTIES_MISSING"))}
 versionRef,ok:=schemaProps["Policy_Status_version"].(map[string]any)
 if !ok{return fail(fmt.Errorf("POLICY_STATUS_VERSION_REF_MISSING"))}
 ref,ok:=versionRef["$ref"].(string)
 if !ok||ref=="" {return fail(fmt.Errorf("POLICY_STATUS_VERSION_REF_INVALID"))}
 id:=version["$id"].(string)
 if ref!=id{return fail(fmt.Errorf("POLICY_STATUS_VERSION_REF_ID_MISMATCH: %s != %s",ref,id))}
 report.Reference=ref
 report.ResourceIDs=[]string{id}
 resources:=[]any{version}
 // A generic string is sufficient; the policy version format is unresolved.
 positive,err:=nativeValidate(e,versionRef,map[string]any{"version":"sample"},resources)
 if err!=nil{return fail(err)}
 missing,err:=nativeValidate(e,versionRef,map[string]any{},resources)
 if err!=nil{return fail(err)}
 report.PositiveValid=positive
 report.MissingVersionRejected=!missing
 if !positive||missing{return fail(fmt.Errorf("APPROVED_VERSION_REF_PROBE_UNEXPECTED_VERDICT"))}
 report.Status="PASS"
 return report
}
