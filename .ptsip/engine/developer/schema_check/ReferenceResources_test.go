package schema_check

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestApprovedResourceReadsOnlyExplicitApprovalPaths(t *testing.T){
 root:=t.TempDir()
 relative:=policyVersionPath
 target:=filepath.Join(root,filepath.FromSlash(relative))
 if err:=os.MkdirAll(filepath.Dir(target),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(target,[]byte(`{"$id":"urn:example:approved","type":"object"}`),0644);err!=nil{t.Fatal(err)}
 doc,err:=approvedResource(root,relative)
 if err!=nil||doc["$id"]!="urn:example:approved"{t.Fatalf("approved schema not read: %v %v",doc,err)}
 if _,err=approvedResource(root,"developer/automation/policy/lifecycle/version_contract.go");err==nil||!strings.Contains(err.Error(),"UNAPPROVED_SCHEMA_RESOURCE"){t.Fatalf("unapproved read should fail: %v",err)}
}

func TestApprovedResourceRejectsMissingIDAndMalformed(t *testing.T){
 root:=t.TempDir()
 target:=filepath.Join(root,filepath.FromSlash(policyVersionPath))
 if err:=os.MkdirAll(filepath.Dir(target),0755);err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{raw,problem string}{
  {`{"type":"object"}`,"APPROVED_SCHEMA_RESOURCE_ID_REQUIRED"},
  {`{"$id":`,"INVALID_APPROVED_SCHEMA_RESOURCE"},
 }{
  if err:=os.WriteFile(target,[]byte(tc.raw),0644);err!=nil{t.Fatal(err)}
  _,err:=approvedResource(root,policyVersionPath)
  if err==nil||!strings.Contains(err.Error(),tc.problem){t.Fatalf("expected %s: %v",tc.problem,err)}
 }
}

func TestApprovedReferenceProbeMissingResourcesIsUnresolved(t *testing.T){
 report:=probeApprovedPolicyVersionReference(t.TempDir(),Engine{})
 if report.Status!="UNRESOLVED"||!strings.Contains(report.Problem,"APPROVED_SCHEMA_RESOURCE_UNAVAILABLE"){
  t.Fatalf("fail-closed resource probe required: %+v",report)
 }
}
