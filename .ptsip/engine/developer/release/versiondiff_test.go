package release

import (
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
)
func gitForVersionTest(t *testing.T,root string,args ...string)string{
 t.Helper()
 cmd:=exec.Command("git",append([]string{"-C",root},args...)...)
 b,err:=cmd.CombinedOutput()
 if err!=nil{t.Fatalf("git %v: %v: %s",args,err,b)}
 return strings.TrimSpace(string(b))
}
func fixtureWrite(t *testing.T,root,path,content string){
 t.Helper()
 dest:=filepath.Join(root,filepath.FromSlash(path))
 if err:=os.MkdirAll(filepath.Dir(dest),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(dest,[]byte(content),0644);err!=nil{t.Fatal(err)}
}
func versionFixture(sectioned bool)string{
 current:=map[string]string{
  "context":"0.2.2","storage":"0.3.2","validation":"0.3.4",
  "discovery":"0.3.3","planning":"0.3.2","execution":"0.3.2","projection":"0.2.2",
 }
 var b strings.Builder
 if sectioned {b.WriteString("[package]\npackage_version = \"0.3.8-b7\"\n[tool]\ntool_version = \"T.0.3.8.b2\"\n[profile]\nprofile_version = \"pp.2.00\"\n")}
 if !sectioned {b.WriteString("package_version = \"0.3.8-b7\"\ntool_version = \"T.0.3.8.b2\"\nprofile_version = \"pp.2.00\"\n")}
 b.WriteString("[engines]\n")
 for _,id:=range engineIDs {b.WriteString(id+" = \""+current[id]+"\"\n")}
 return b.String()
}
func versionTestRepo(t *testing.T,sectioned bool)(string,string,string){
 t.Helper()
 root:=t.TempDir()
 gitForVersionTest(t,root,"init","-q")
 gitForVersionTest(t,root,"config","user.email","test@example.com")
 gitForVersionTest(t,root,"config","user.name","Test")
 sourceContract,err:=os.ReadFile(filepath.Join("..","..","..","..",filepath.FromSlash(versionsContract)))
 if err!=nil{t.Fatal(err)}
 fixtureWrite(t,root,versionsContract,string(sourceContract))
 fixtureWrite(t,root,versionsSource,versionFixture(sectioned))
 fixtureWrite(t,root,"src/ptsip/engines/validation/existing.rs","pub fn old() {}\n")
 gitForVersionTest(t,root,"add",".")
 gitForVersionTest(t,root,"commit","-qm","baseline")
 base:=gitForVersionTest(t,root,"rev-parse","HEAD")
 fixtureWrite(t,root,"src/ptsip/engines/validation/new.rs","pub fn added_feature() {}\n")
 gitForVersionTest(t,root,"add",".")
 gitForVersionTest(t,root,"commit","-qm","add declaration")
 head:=gitForVersionTest(t,root,"rev-parse","HEAD")
 return root,base,head
}
func TestVersionsDifferentPreviewDestination(t *testing.T){
 if versionsPreview!=".ptsip/engine/developer/release/versions_different.ini"{
  t.Fatalf("unexpected version preview destination: %s",versionsPreview)
 }
 root,base,head:=versionTestRepo(t,true)
 if _,err:=PreviewVersionsDifferent(root,base,head);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,filepath.FromSlash(versionsPreview)));err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,".ptsip","engines","release","versions_different.ini"));!os.IsNotExist(err){
  t.Fatalf("obsolete preview destination created: %v",err)
 }
}

func TestVersionsDifferentPreviewAndTamper(t *testing.T){
 root,base,head:=versionTestRepo(t,true)
 report,err:=PreviewVersionsDifferent(root,base,head)
 if err!=nil{t.Fatal(err)}
 if report.Legacy || len(report.Entries)!=1 || report.Entries[0].Component!="engine.validation" ||
    !report.Entries[0].DeclarationCandidate || report.Entries[0].Current!="0.3.4" {
  t.Fatalf("unexpected diff report: %#v",report)
 }
 if _,err=CheckVersionsDifferent(root,base,head);err!=nil{t.Fatal(err)}
 path:=filepath.Join(root,filepath.FromSlash(versionsPreview))
 f,err:=os.OpenFile(path,os.O_APPEND|os.O_WRONLY,0)
 if err!=nil{t.Fatal(err)}
 if _,err=f.WriteString("proposed_version=0.3.5\n");err!=nil{t.Fatal(err)}
 f.Close()
 if _,err=CheckVersionsDifferent(root,base,head);err==nil{t.Fatal("modified preview was accepted")}
}
func TestVersionsDifferentPolicyOnlyNoBump(t *testing.T){
 root,base,head:=versionTestRepo(t,true)
 _=base
 fixtureWrite(t,root,"developer/policy/unrelated.yaml","approval: example\n")
 gitForVersionTest(t,root,"add",".")
 gitForVersionTest(t,root,"commit","-qm","policy only")
 policyHead:=gitForVersionTest(t,root,"rev-parse","HEAD")
 report,err:=AnalyzeVersionsDifferent(root,head,policyHead)
 if err!=nil{t.Fatal(err)}
 if len(report.Entries)!=0{t.Fatalf("policy-only commit changed versions: %#v",report)}
}
func TestVersionsDifferentApplyRequiresExplicitSelection(t *testing.T){
 root,base,head:=versionTestRepo(t,true)
 report,err:=PreviewVersionsDifferent(root,base,head)
 if err!=nil{t.Fatal(err)}
 args:=[]string{root,base,head,"engine.validation","0.3.5",report.VersionDigest}
 if err=ApplySelectedVersion(args[0],args[1],args[2],args[3],args[4],args[5],false);err==nil{
  t.Fatal("unapproved version update accepted")
 }
 if err=ApplySelectedVersion(root,base,head,"engine.validation","0.3.5",strings.Repeat("0",64),true);err==nil{
  t.Fatal("wrong source digest accepted")
 }
 if err=ApplySelectedVersion(root,base,head,"engine.validation","0.3.5",report.VersionDigest,true);err!=nil{t.Fatal(err)}
 data,err:=os.ReadFile(filepath.Join(root,"src","VERSION"))
 if err!=nil{t.Fatal(err)}
 fields,legacy,err:=sourceVersions(data)
 if err!=nil || legacy || fields["engine.validation"]!="0.3.5" {
  t.Fatalf("not updated: fields=%v legacy=%v err=%v",fields,legacy,err)
 }
 if _,err=CheckVersionsDifferent(root,base,head);err==nil{t.Fatal("stale preview remains valid after editing VERSION")}
}
func TestVersionsDifferentLegacyReadOnly(t *testing.T){
 root,base,head:=versionTestRepo(t,false)
 report,err:=PreviewVersionsDifferent(root,base,head)
 if err!=nil{t.Fatal(err)}
 if !report.Legacy{t.Fatal("legacy format not flagged")}
 if err=ApplySelectedVersion(root,base,head,"engine.validation","0.3.5",report.VersionDigest,true);err==nil{
  t.Fatal("legacy source was modified")
 }
}
func TestVersionsDifferentRejectsFakeHeadAndNoSourceChange(t *testing.T){
 root,base,head:=versionTestRepo(t,true)
 if _,err:=AnalyzeVersionsDifferent(root,head,base);err==nil{t.Fatal("non-ancestor accepted")}
 if _,err:=AnalyzeVersionsDifferent(root,base,strings.Repeat("a",40));err==nil{t.Fatal("unverified HEAD accepted")}
 if _,err:=AnalyzeVersionsDifferent(root,head,head);err==nil{t.Fatal("identical commits accepted")}
}
