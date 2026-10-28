// Package ctrl_admission validates the effective, machine-readable CTRL
// Release Admission policy. It never authorizes a particular publication run:
// exact-source ASSURE evidence and immutable artifacts are checked separately.
package ctrl_admission

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "reflect"
    "sort"
    "strings"
)
const policyRef = "developer/policy/CTRL/release-admission.policy.json"
const schemaRef = ".ptsip/contracts/schemas/release/ctrl-release-admission.schema.json"
const ownerRef = "developer/policy/registries/release-admission-ctrl-ownership.json"
const activationRef = "developer/policy/registries/release-admission-publication-activation.json"
const approvalRef = "developer/policy/approvals/MPA-20261026-CTRL-RELEASE-ADMISSION-COMPLETION.yaml"

func read(root, path string) ([]byte,error) {
    if path=="" || filepath.IsAbs(path) || strings.Contains(path,"\\") ||
       filepath.Clean(path)!=path || strings.HasPrefix(path,"..") { return nil,fmt.Errorf("unsafe CTRL source path %q",path) }
    base,err:=filepath.Abs(root); if err!=nil{return nil,err}
    actual,err:=filepath.EvalSymlinks(filepath.Join(base,path)); if err!=nil{return nil,err}
    rel,err:=filepath.Rel(base,actual)
    if err!=nil || rel==".." || strings.HasPrefix(rel,".."+string(os.PathSeparator)) {return nil,fmt.Errorf("CTRL source escapes repository: %s",path)}
    info,err:=os.Stat(actual)
    if err!=nil || !info.Mode().IsRegular(){return nil,fmt.Errorf("missing regular CTRL contract file: %s",path)}
    return os.ReadFile(actual)
}
func data(root,path string)(map[string]any,error){
    b,err:=read(root,path); if err!=nil{return nil,err}
    decoder:=json.NewDecoder(bytes.NewReader(b));decoder.DisallowUnknownFields()
    var obj map[string]any
    if err=decoder.Decode(&obj);err!=nil{return nil,fmt.Errorf("invalid CTRL JSON %s: %w",path,err)}
    if obj==nil{return nil,fmt.Errorf("empty CTRL object %s",path)}
    if decoder.Decode(new(any))==nil{return nil,fmt.Errorf("trailing CTRL JSON %s",path)}
    return obj,nil
}
func object(value any) map[string]any {m,_:=value.(map[string]any);return m}
func field(m map[string]any,k string) string {v,_:=m[k].(string);return v}
func equal(a,b any) bool {return reflect.DeepEqual(a,b)}
func must(m map[string]any,k string,want any) error {
    if m==nil || !equal(m[k],want) {return fmt.Errorf("CTRL policy mismatch: %s",k)}
    return nil
}
func keys(m map[string]any)[]string{out:=make([]string,0,len(m));for k:=range m{out=append(out,k)};sort.Strings(out);return out}
func validateSchema(node map[string]any,value any,path string)error{
    // The committed schema intentionally uses only standard, closed Draft 2020-12
    // keywords: type, const, required, properties, additionalProperties.
    for k:=range node{switch k{case "$id","$schema","type","const","required","properties","additionalProperties":default:return fmt.Errorf("unsupported schema keyword at %s: %s",path,k)}}
    if field(node,"type")!="object"{return fmt.Errorf("schema is not a closed object: %s",path)}
    actual,ok:=value.(map[string]any);if !ok{return fmt.Errorf("expected JSON object: %s",path)}
    if node["additionalProperties"]!=false{return fmt.Errorf("open schema forbidden: %s",path)}
    specs:=object(node["properties"]);if len(specs)==0{return fmt.Errorf("empty schema fields: %s",path)}
    req,ok:=node["required"].([]any);if !ok || len(req)!=len(specs){return fmt.Errorf("required-field closure absent: %s",path)}
    seen:=map[string]bool{}
    for _,entry:=range req{
        name,ok:=entry.(string)
        if !ok || seen[name] {return fmt.Errorf("invalid required schema member: %s",path)}
        seen[name]=true
        if _,present:=specs[name]; !present{return fmt.Errorf("required field outside schema: %s",name)}
        if _,present:=actual[name]; !present{return fmt.Errorf("CTRL policy field missing: %s.%s",path,name)}
    }
    for k,v:=range actual{
        child,ok:=specs[k];if !ok{return fmt.Errorf("unknown CTRL policy field: %s.%s",path,k)}
        def:=object(child)
        if _,has:=def["const"];has{
            if !equal(def["const"],v){return fmt.Errorf("CTRL policy const mismatch: %s.%s",path,k)}
            if len(def)!=1{return fmt.Errorf("constant schema has extra keywords: %s.%s",path,k)}
        }else if err:=validateSchema(def,v,path+"."+k);err!=nil{return err}
    }
    return nil
}
func approved(root,ref string,markers ...string)error{
    b,err:=read(root,ref);if err!=nil{return err}
    s:="\n"+strings.ReplaceAll(string(b),"\r\n","\n")+"\n"
    for _,v:=range append([]string{"  decision: APPROVED","  decision_source: USER_EXPLICIT"},markers...){
        if !strings.Contains(s,"\n"+v+"\n") {return fmt.Errorf("missing owner approval evidence %s: %s",ref,v)}
    }
    return nil
}
// Verify enforces a validated normative policy without ever inferring
// publication authorization from the non-normative release design report.
func Verify(root string)error{
    policy,err:=data(root,policyRef);if err!=nil{return err}
    schema,err:=data(root,schemaRef);if err!=nil{return err}
    if field(schema,"$schema")!="https://json-schema.org/draft/2020-12/schema" ||
       field(schema,"$id")!="urn:ptsip:policy:ctrl:release-admission:v1" {
        return fmt.Errorf("unregistered CTRL Release Admission schema")
    }
    if err=validateSchema(schema,policy,"release_admission");err!=nil{return err}
    identity:=object(policy["policy_identity"])
    if err=must(identity,"id","MPD-CTRL-RELEASE-ADMISSION");err!=nil{return err}
    for _,pair:=range [][2]string{{"status","APPROVED"},{"substatus","Applying"},{"root_family","CTRL"},{"conflict_resolution","KEEP_LATEST"},{"approval_ref",approvalRef}}{
        if err=must(identity,pair[0],pair[1]);err!=nil{return err}
    }
    responsibility:=object(policy["responsibility"])
    for _,pair:=range [][2]string{{"admission","CTRL"},{"verification_evidence","ASSURE"},{"publication_execution","RELEASE_CONTROLLER"}}{
        if err=must(responsibility,pair[0],pair[1]);err!=nil{return err}
    }
    authorization:=object(policy["authorization"])
    for _,pair:=range [][2]string{{"mode","PER_RELEASE_CONDITIONAL"},{"automation","ENABLED"},{"credential_scope","NO_AUTOMATIC_EXPANSION"}}{
        if err=must(authorization,pair[0],pair[1]);err!=nil{return err}
    }
    if err=must(authorization,"pypi_projects",[]any{"PTSIP"});err!=nil{return err}
    if err=must(authorization,"destinations",[]any{"github","github-pypi"});err!=nil{return err}
    if err=must(authorization,"test_pass_grants_publication",false);err!=nil{return err}
    evidence:=object(policy["evidence"])
    for _,pair:=range [][2]string{{"source_sha","EXACT_HEAD"},{"selection","EXACT_SELECTION_SHA256"},{"workflow",".github/workflows/tooling-test.yml"},{"all_selected_component_host_jobs","SUCCESS_REQUIRED"},{"verified_original_artifact_digests","REQUIRED"},{"stale_or_missing","FAIL_CLOSED"}}{
        if err=must(evidence,pair[0],pair[1]);err!=nil{return err}
    }
    if err=must(evidence,"required_platforms",[]any{"windows-x64","linux-x64"});err!=nil{return err}
    publication:=object(policy["publication"])
    for _,pair:=range [][2]string{{"github_release_identity","EXACT_SOURCE_SHA"},{"already_published_assets","IMMUTABLE"},{"engine_producer_same_version_changed","REJECT"},{"package_completeness","TOOL_PROFILE_AND_ALL_SEVEN_ENGINES"},{"developer_policy_fallback","FORBIDDEN"},{"verification_workflow_publication","FORBIDDEN"},{"tool_only_fallback","FORBIDDEN"}}{
        if err=must(publication,pair[0],pair[1]);err!=nil{return err}
    }
    recovery:=object(policy["recovery"])
    for _,pair:=range [][2]string{{"security_anomaly","SUSPEND_AND_PRESERVE_EVIDENCE"},{"resume","EXPLICIT_OWNER_APPROVAL"},{"partial_publication","RE_EVALUATE_ADMISSION"}}{
        if err=must(recovery,pair[0],pair[1]);err!=nil{return err}
    }
    refs:=object(policy["references"])
    for k,want:=range map[string]string{
        "owner":ownerRef,"activation":activationRef,
        "verification":".ptsip/contracts/release/verification-contract.json",
        "composition":".ptsip/contracts/release/composition-contract.json",
        "admission_directions":".ptsip/contracts/release/admission.json",
        "publisher_scope":".ptsip/contracts/release/pypi-publishers.json",
        "schema":schemaRef,"publication_workflow":".github/workflows/tooling-release.yml",
    } {if err=must(refs,k,want);err!=nil{return err}}
    if err=approved(root,approvalRef,"  responsibility_scope: CTRL_PER_RELEASE_PUBLICATION_ADMISSION");err!=nil{return err}
    if err=approved(root,"developer/policy/approvals/MPA-20261026-RELEASE-ADMISSION-PUBLICATION-KEEP-LATEST.yaml","  conflict_resolution: KEEP_LATEST");err!=nil{return err}
    if err=approved(root,"developer/policy/approvals/MPA-20261026-RELEASE-ADMISSION-CTRL-OWNERSHIP.yaml","  responsibility_scope: RELEASE_ADMISSION_OWNER_SELECTION");err!=nil{return err}
    owner,err:=data(root,ownerRef);if err!=nil{return err}
    if err=must(owner,"owner_decision","CTRL_CENTERED");err!=nil{return err}
    if err=must(object(owner["responsibility"]),"admission_decision","CTRL");err!=nil{return err}
    if err=must(object(owner["responsibility"]),"exact_source_verification_evidence","ASSURE");err!=nil{return err}
    if err=must(object(owner["machine_contract"]),"runtime_policy_ref",policyRef);err!=nil{return err}
    actOwner:=object(owner["activation"])
    for _,key:=range []string{"direction_approved","runtime_policy_document_complete","required_policy_schema_and_validator_complete","automation_implementation_authorized"}{
        if err=must(actOwner,key,true);err!=nil{return err}
    }
    activation,err:=data(root,activationRef);if err!=nil{return err}
    if err=must(activation,"resolution","KEEP_LATEST");err!=nil{return err}
    if err=must(object(activation["effective_decision"]),"admission_authority_owner","CTRL");err!=nil{return err}
    if err=must(object(activation["effective_decision"]),"verification_evidence_owner","ASSURE");err!=nil{return err}
    gates:=object(activation["machine_gates"])
    if err=must(gates,"normative_successor_validated",true);err!=nil{return err}
    if err=must(gates,"publish_workflow_enabled",true);err!=nil{return err}
    // No repository-level grant for an individual source SHA.
    if err=must(gates,"actual_publication_admitted",false);err!=nil{return err}
    publisher,err:=data(root,field(refs,"publisher_scope"));if err!=nil{return err}
    auth:=object(publisher["authentication"])
    if err=must(auth,"declared_project_scope",[]any{"PTSIP"});err!=nil{return err}
    if err=must(auth,"mode","API_TOKEN");err!=nil{return err}
    if err=must(object(publisher["package"]),"project","PTSIP");err!=nil{return err}
    if err=must(auth,"oidc_fallback",false);err!=nil{return err}
    verification,err:=data(root,field(refs,"verification"));if err!=nil{return err}
    if err=must(verification,"workflow",field(evidence,"workflow"));err!=nil{return err}
    if err=must(verification,"required_platforms",[]any{"windows-x64","linux-x64"});err!=nil{return err}
    if err=must(verification,"publication_operations",false);err!=nil{return err}
    waiver:=object(evidence["temporary_join_waiver"])
    if err=must(waiver,"ref","developer/policy/registries/release-20261026-package-join-waiver.json");err!=nil{return err}
    if err=must(waiver,"package_version","0.3.8b6");err!=nil{return err}
    if err=must(waiver,"profile_version","pp.2.00");err!=nil{return err}
    if err=must(waiver,"scope","FIRST_PUBLICATION_ONLY");err!=nil{return err}
    if err=must(waiver,"full_join","WAIVED");err!=nil{return err}
    if err=must(waiver,"lightweight_package_install","REQUIRED");err!=nil{return err}
    if err=must(waiver,"lightweight_package_version","REQUIRED");err!=nil{return err}
    if err=must(waiver,"required_original_digests","REQUIRED");err!=nil{return err}
    if err=must(verification,"temporary_join_waiver_ref",field(waiver,"ref"));err!=nil{return err}
    if err=must(object(verification["joined_verification"]),"required","TEMPORARY_RECONCILED_WAIVER_FOR_PTSIP_0_3_8_B6");err!=nil{return err}
    w,err:=data(root,field(waiver,"ref"));if err!=nil{return err}
    if err=must(w,"decision","RECONCILE");err!=nil{return err}
    if err=must(w,"state","ACTIVE_UNTIL_FIRST_SUCCESSFUL_PUBLICATION_RECORD");err!=nil{return err}
    if err=approved(root,"developer/policy/approvals/MPA-20261026-RELEASE-JOIN-RECONCILE.yaml","  conflict_resolution: RECONCILE");err!=nil{return err}
    if err=approved(root,"developer/policy/approvals/MPA-20261026-PACKAGE-0-3-8-B6-KEEP-LATEST.yaml","  conflict_resolution: KEEP_LATEST");err!=nil{return err}
    if err=approved(root,"developer/policy/approvals/MPA-20261026-PROJECT-PROFILE-2-00.yaml","  selected_profile_version: 'pp.2.00'");err!=nil{return err}

    composition,err:=data(root,field(refs,"composition"));if err!=nil{return err}
    if err=must(object(composition["github_release"]),"asset_overwrite",false);err!=nil{return err}
    if err=must(object(composition["engine_reuse"]),"changed_source_same_version","FAIL_REQUIRES_EXPLICIT_NEW_VERSION");err!=nil{return err}
    if err=must(object(composition["package_request"]),"component_subset_or_exclusion","FORBIDDEN");err!=nil{return err}
    directions,err:=data(root,field(refs,"admission_directions"));if err!=nil{return err}
    common,ok1:=directions["common_directions"].([]any);issues,ok2:=directions["error_directions"].([]any)
    if !ok1 || !ok2 || len(common)!=5 || len(issues)!=14 {return fmt.Errorf("release error/admission direction coverage incomplete")}
    workflow,err:=read(root,field(refs,"publication_workflow"));if err!=nil{return err}
    for _,token:=range []string{"admit-policy","prepare-publication","publish-carrier","pypa/gh-action-pypi-publish@release/v1","PYPI_API_TOKEN"}{
        if !strings.Contains(string(workflow),token){return fmt.Errorf("publication workflow lacks %s",token)}
    }
    if strings.Contains(string(workflow),".github/checks/release/controller"){return fmt.Errorf("retired publication controller reactivated")}
    return nil
}
func Run(root string)(any,int){
    if err:=Verify(root);err!=nil{return map[string]any{"status":"BLOCKED","reason":err.Error(),"publication_authorized":false,"per_release_evidence":"NOT_ASSERTED"},2}
    return map[string]any{"status":"ACTIVE_POLICY_VERIFIED","normative_policy":policyRef,"admission_owner":"CTRL","evidence_owner":"ASSURE","per_release_evidence":"NOT_ASSERTED","publication_authorized":false,"permission_model":"CONDITIONAL_EXACT_SOURCE"},0
}
