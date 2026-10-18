package test_mode

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestSafeTargetRejectsTraversal(t *testing.T) {
    root:=t.TempDir()
    for _,v:=range []string{"../outside","a/../b","/absolute","C:\\Windows\\system","a//b",""} {
        if _,err:=safeTarget(root,v);err==nil {t.Errorf("unsafe path accepted: %q",v)}
    }
}
func TestDeclaredVerificationOnly(t *testing.T) {
    root:=t.TempDir()
    location:=filepath.Join(root,"developer/tests/example")
    if err:=os.MkdirAll(location,0755);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(location,"test_contract.py"),[]byte("pass"),0644);err!=nil{t.Fatal(err)}
    component:=map[string]any{"id":"example-verification","roles":[]any{"VERIFICATION"},"include":[]any{"developer/tests/example/**"}}
    profile:=map[string]any{"components":[]any{component}}
    mode:=map[string]any{"id":"example","component_ref":"example-verification","execution":map[string]any{"pytest":[]any{"developer/tests/example"}}}
    registry:=map[string]any{"groups":map[string]any{"DEVELOPER":[]any{"example"}},"modes":[]any{mode},"defaults":map[string]any{"workflow_mode":"example"}}
    if found:=contractErrors(root,registry,profile);len(found)>0{t.Fatalf("valid declared contract rejected: %v",found)}
    registry["groups"]=map[string]any{"DEVELOPER":[]any{"example","example"}}
    found:=contractErrors(root,registry,profile)
    if len(found)==0||!strings.Contains(strings.Join(found,"|"),"MODE_GROUP_MEMBERSHIP"){t.Fatalf("duplicate accepted: %v",found)}
    registry["groups"]=map[string]any{"DEVELOPER":[]any{"example"}}
    component["roles"]=[]any{"IMPLEMENTATION"}
    found=contractErrors(root,registry,profile)
    if len(found)==0||!strings.Contains(strings.Join(found,"|"),"MISSING_VERIFICATION_OWNER"){t.Fatalf("unverified owner accepted: %v",found)}
}
