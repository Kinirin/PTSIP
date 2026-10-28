package release

// Versions-different is a non-normative release preview tool.
// A source diff or new declaration is never proof of a functional release.
import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "sort"
 "strings"
)

const versionsContract = ".ptsip/contracts/release/versions_different.json"
const versionsPreview = ".ptsip/engines/release/versions_different.ini"
const versionsSource = "src/VERSION"

var sha40 = regexp.MustCompile("^[a-f0-9]{40}$")
var sha64 = regexp.MustCompile("^[a-f0-9]{64}$")
var assignment = regexp.MustCompile("^([a-z][a-z0-9_]*)[ \t]*=[ \t]*\"([^\"]+)\"[ \t]*$")
var header = regexp.MustCompile("^\\[([a-z]+)\\]$")
var declaration = regexp.MustCompile("^\\+[ \t]*((pub(\\([^)]*\\))?[ \t]+)?(async[ \t]+)?fn[ \t]+[a-zA-Z_]|(async[ \t]+)?def[ \t]+[a-zA-Z_]|func[ \t]+(\\([^)]*\\)[ \t]*)?[a-zA-Z_])")
var engineIDs = []string{"context","discovery","execution","planning","projection","storage","validation"}
var engineVersionRE = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$")
var packageVersionRE = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-(a|b|rc)[1-9][0-9]*)?$")
var toolVersionRE = regexp.MustCompile("^T\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\.(a|b|rc)[1-9][0-9]*)?$")
var profileVersionRE = regexp.MustCompile("^pp\\.[0-9]+\\.[0-9]+$")

type VersionDiffEntry struct {
 Component string
 Current string
 Paths []string
 DeclarationCandidate bool
}
type VersionDiffReport struct {
 Base string
 Head string
 VersionDigest string
 Legacy bool
 Entries []VersionDiffEntry
}

func contractObject(m map[string]any,k string) map[string]any {
 v,_:=m[k].(map[string]any)
 return v
}
func contractString(m map[string]any,k string) string {
 v,_:=m[k].(string)
 return v
}
func checkVersionsContract(root string) error {
 raw,err:=os.ReadFile(filepath.Join(root,filepath.FromSlash(versionsContract)))
 if err!=nil{return err}
 var m map[string]any
 d:=json.NewDecoder(bytes.NewReader(raw))
 d.DisallowUnknownFields()
 if err=d.Decode(&m);err!=nil{return err}
 var rest any
 if err=d.Decode(&rest);err!=io.EOF{return fmt.Errorf("trailing contract JSON")}
 detection:=contractObject(m,"detection")
 update:=contractObject(m,"update")
 bounds:=contractObject(m,"boundaries")
 if contractString(m,"schema_version")!="ptsip-versions-different/v1" ||
    contractString(m,"state")!="DRAFT_NON_NORMATIVE" ||
    contractString(m,"source")!=versionsSource || contractString(m,"preview")!=versionsPreview ||
    contractString(detection,"basis")!="EXACT_GIT_DIFF" ||
    contractString(detection,"function_addition")!="SYNTAX_CANDIDATE_ONLY" ||
    contractString(detection,"functional_verification")!="UNRESOLVED" ||
    contractString(detection,"policy_only")!="NO_BUMP" ||
    contractString(update,"automatic_bump")!="BLOCKED_PENDING_POLICY" ||
    contractString(update,"increment_rule")!="UNRESOLVED" ||
    contractString(update,"explicit_apply")!="REQUIRES_CALLER_DECISION_AND_EXACT_DIGEST" ||
    contractString(update,"publication")!="NOT_AUTHORIZED" ||
    contractString(bounds,"source_root")!="src/ptsip/" ||
    contractString(bounds,"engine_prefix")!="src/ptsip/engines/" ||
    contractString(bounds,"profile_prefix")!="src/ptsip/profiles/" ||
    contractString(bounds,"other_source")!="TOOL_CANDIDATE" {
   return fmt.Errorf("unresolved contract must not grant version or publication authority")
 }
 ids,ok:=bounds["engine_ids"].([]any)
 if !ok || len(ids)!=len(engineIDs){return fmt.Errorf("invalid engine inventory")}
 registered:=map[string]bool{}
 for _,id:=range ids {s,ok:=id.(string);if !ok || registered[s] {return fmt.Errorf("duplicate engine")};registered[s]=true}
 for _,id:=range engineIDs {if !registered[id] {return fmt.Errorf("missing engine %s",id)}}
 return nil
}
func gitVersion(root string,args ...string)([]byte,error){
 cmd:=exec.Command("git",append([]string{"-C",root},args...)...)
 b,err:=cmd.Output()
 if err!=nil {return nil,fmt.Errorf("git %s: %w",strings.Join(args," "),err)}
 return b,nil
}
func sourceVersions(data []byte)(map[string]string,bool,error) {
 result:=map[string]string{}
 section:=""
 sectioned:=false
 for _,line:=range strings.Split(strings.ReplaceAll(string(data),"\r\n","\n"),"\n") {
  l:=strings.TrimSpace(line)
  if l=="" || strings.HasPrefix(l,"#"){continue}
  if m:=header.FindStringSubmatch(l);m!=nil{
   section=m[1];sectioned=true
   if section!="engines" && section!="package" && section!="tool" && section!="profile" {return nil,false,fmt.Errorf("unregistered version section")}
   continue
  }
  m:=assignment.FindStringSubmatch(l)
  if m==nil {return nil,false,fmt.Errorf("unsupported version assignment")}
  key:=m[1]
  switch {
  case section=="engines":key="engine."+key
  case (section=="package"||section=="")&&key=="package_version":key="package"
  case (section=="tool"||section=="")&&key=="tool_version":key="tool"
  case (section=="profile"||section=="")&&key=="profile_version":key="profile"
  default:return nil,false,fmt.Errorf("unknown source VERSION key %s",key)
  }
  if _,present:=result[key];present{return nil,false,fmt.Errorf("duplicate VERSION key")}
  result[key]=m[2]
 }
 if len(result)!=3+len(engineIDs) || !packageVersionRE.MatchString(result["package"]) ||
    !toolVersionRE.MatchString(result["tool"]) || !profileVersionRE.MatchString(result["profile"]){
  return nil,false,fmt.Errorf("invalid or incomplete VERSION")
 }
 for _,id:=range engineIDs {if !engineVersionRE.MatchString(result["engine."+id]){return nil,false,fmt.Errorf("invalid engine version %s",id)}}
 return result,!sectioned,nil
}
func diffComponent(path string)string{
 if strings.HasPrefix(path,"src/ptsip/profiles/"){return "profile"}
 if strings.HasPrefix(path,"src/ptsip/engines/"){
  remain:=strings.TrimPrefix(path,"src/ptsip/engines/")
  for _,id:=range engineIDs {if strings.HasPrefix(remain,id+"/"){return "engine."+id}}
 }
 return "tool"
}

// AnalyzeVersionsDifferent detects affected source boundaries and syntactic additions;
// semantic functionality and version increment remain unresolved.
func AnalyzeVersionsDifferent(root,base,head string)(VersionDiffReport,error){
 var report VersionDiffReport
 if err:=checkVersionsContract(root);err!=nil{return report,err}
 if !sha40.MatchString(base)||!sha40.MatchString(head)||base==head{return report,fmt.Errorf("two different exact commit SHAs required")}
 now,err:=gitVersion(root,"rev-parse","HEAD")
 if err!=nil{return report,err}
 if strings.TrimSpace(string(now))!=head{return report,fmt.Errorf("head is not checked-out HEAD")}
 dirty,err:=gitVersion(root,"status","--porcelain","--","src/ptsip","src/VERSION")
 if err!=nil{return report,err}
 if len(bytes.TrimSpace(dirty))!=0{return report,fmt.Errorf("dirty source tree or VERSION")}
 if _,err=gitVersion(root,"merge-base","--is-ancestor",base,head);err!=nil{return report,fmt.Errorf("base must be an ancestor")}
 version,err:=gitVersion(root,"show",head+":"+versionsSource)
 if err!=nil{return report,err}
 values,legacy,err:=sourceVersions(version)
 if err!=nil{return report,err}
 digest:=sha256.Sum256(version)
 report=VersionDiffReport{Base:base,Head:head,VersionDigest:hex.EncodeToString(digest[:]),Legacy:legacy}
 files,err:=gitVersion(root,"diff","--no-renames","--name-only","-z",base,head,"--","src/ptsip")
 if err!=nil{return report,err}
 entries:=map[string]*VersionDiffEntry{}
 for _,raw:=range bytes.Split(files,[]byte{0}){
  if len(raw)==0{continue}
  path:=string(raw)
  if !strings.HasPrefix(path,"src/ptsip/")||strings.ContainsAny(path,"\r\n"){return report,fmt.Errorf("invalid changed path")}
  component:=diffComponent(path)
  entry:=entries[component]
  if entry==nil {entry=&VersionDiffEntry{Component:component,Current:values[component]};entries[component]=entry}
  entry.Paths=append(entry.Paths,path)
  if strings.HasSuffix(path,".go")||strings.HasSuffix(path,".rs")||strings.HasSuffix(path,".py"){
   patch,e:=gitVersion(root,"diff","--no-ext-diff","--no-renames","--unified=0",base,head,"--",path)
   if e!=nil{return report,e}
   for _,line:=range strings.Split(string(patch),"\n"){
    if !strings.HasPrefix(line,"+++")&&declaration.MatchString(line){entry.DeclarationCandidate=true}
   }
  }
 }
 for _,entry:=range entries {sort.Strings(entry.Paths);report.Entries=append(report.Entries,*entry)}
 sort.Slice(report.Entries,func(i,j int)bool{return report.Entries[i].Component<report.Entries[j].Component})
 return report,nil
}
func FormatVersionsDifferent(r VersionDiffReport)[]byte{
 var b strings.Builder
 fmt.Fprintf(&b,"[metadata]\nschema_version=ptsip-versions-different-preview/v1\nbase_sha=%s\nhead_sha=%s\nversion_sha256=%s\nstatus=REVIEW_REQUIRED\nsource_layout=",r.Base,r.Head,r.VersionDigest)
 if r.Legacy{b.WriteString("LEGACY_READ_ONLY\n")}else{b.WriteString("SECTIONED_TOML_V1\n")}
 for _,e:=range r.Entries {
  fmt.Fprintf(&b,"\n[%s]\ncurrent_version=%s\nsource_change=DETECTED\ndeclaration_addition=",e.Component,e.Current)
  if e.DeclarationCandidate{b.WriteString("CANDIDATE_ONLY\n")}else{b.WriteString("NOT_DETECTED\n")}
  b.WriteString("functional_change=UNRESOLVED\nproposed_version=UNRESOLVED\n")
  for i,p:=range e.Paths {fmt.Fprintf(&b,"source_file_%d=%s\n",i+1,p)}
 }
 return []byte(b.String())
}
func PreviewVersionsDifferent(root,base,head string)(VersionDiffReport,error){
 r,err:=AnalyzeVersionsDifferent(root,base,head)
 if err!=nil{return r,err}
 path:=filepath.Join(root,filepath.FromSlash(versionsPreview))
 if err=os.MkdirAll(filepath.Dir(path),0755);err!=nil{return r,err}
 if info,e:=os.Lstat(path);e==nil && !info.Mode().IsRegular(){return r,fmt.Errorf("unsafe preview destination")} else if e!=nil&&!os.IsNotExist(e){return r,e}
 temp,err:=os.CreateTemp(filepath.Dir(path),".version-diff-")
 if err!=nil{return r,err}
 defer os.Remove(temp.Name())
 if _,err=temp.Write(FormatVersionsDifferent(r));err!=nil{temp.Close();return r,err}
 if err=temp.Close();err!=nil{return r,err}
 return r,os.Rename(temp.Name(),path)
}
func CheckVersionsDifferent(root,base,head string)(VersionDiffReport,error){
 r,err:=AnalyzeVersionsDifferent(root,base,head)
 if err!=nil{return r,err}
 path:=filepath.Join(root,filepath.FromSlash(versionsPreview))
 info,err:=os.Lstat(path)
 if err!=nil{return r,err}
 if !info.Mode().IsRegular(){return r,fmt.Errorf("unsafe preview")}
 b,err:=os.ReadFile(path)
 if err!=nil{return r,err}
 if !bytes.Equal(b,FormatVersionsDifferent(r)){return r,fmt.Errorf("stale or modified preview")}
 return r,nil
}

// ApplySelectedVersion changes only a caller-selected version after an exact preview
// and exact source digest check. It does not implement an automatic increment rule.
func ApplySelectedVersion(root,base,head,component,target,digest string,explicit bool)error{
 if !explicit{return fmt.Errorf("explicit version choice is required")}
 if !sha64.MatchString(digest){return fmt.Errorf("exact version source sha256 required")}
 r,err:=CheckVersionsDifferent(root,base,head)
 if err!=nil{return err}
 if r.Legacy{return fmt.Errorf("legacy VERSION layout is read only")}
 if r.VersionDigest!=digest{return fmt.Errorf("stale version source digest")}
 allowed:=false
 for _,entry:=range r.Entries{if entry.Component==component{allowed=true}}
 if !allowed{return fmt.Errorf("component has no source diff")}
 section,key:="",""
 switch{
 case component=="tool"&&toolVersionRE.MatchString(target):section,key="tool","tool_version"
 case component=="profile"&&profileVersionRE.MatchString(target):section,key="profile","profile_version"
 case component=="package"&&packageVersionRE.MatchString(target):section,key="package","package_version"
 case strings.HasPrefix(component,"engine.")&&engineVersionRE.MatchString(target):
  id:=strings.TrimPrefix(component,"engine.")
  for _,x:=range engineIDs{if x==id{section,key="engines",id}}
 }
 if section==""{return fmt.Errorf("invalid component or target version")}
 path:=filepath.Join(root,filepath.FromSlash(versionsSource))
 info,err:=os.Lstat(path)
 if err!=nil{return err}
 if !info.Mode().IsRegular(){return fmt.Errorf("unsafe VERSION source")}
 b,err:=os.ReadFile(path)
 if err!=nil{return err}
 sum:=sha256.Sum256(b)
 if hex.EncodeToString(sum[:])!=digest{return fmt.Errorf("VERSION changed after preview")}
 lines:=strings.SplitAfter(string(b),"\n")
 active:=""
 count:=0
 for i,line:=range lines{
  short:=strings.TrimSpace(line)
  if m:=header.FindStringSubmatch(short);m!=nil{active=m[1]}
  if active!=section{continue}
  m:=assignment.FindStringSubmatch(short)
  if m==nil || m[1]!=key{continue}
  if m[2]==target{return fmt.Errorf("version unchanged")}
  count++
  end:=""
  if strings.HasSuffix(line,"\r\n"){end="\r\n"}else if strings.HasSuffix(line,"\n"){end="\n"}
  lines[i]=key+" = "+fmt.Sprintf("%q",target)+end
 }
 if count!=1{return fmt.Errorf("ambiguous VERSION field")}
 after:=[]byte(strings.Join(lines,""))
 if _,legacy,err:=sourceVersions(after);err!=nil||legacy{return fmt.Errorf("invalid resulting VERSION")}
 tmp,err:=os.CreateTemp(filepath.Dir(path),".version-update-")
 if err!=nil{return err}
 defer os.Remove(tmp.Name())
 if _,err=tmp.Write(after);err!=nil{tmp.Close();return err}
 if err=tmp.Chmod(info.Mode().Perm());err!=nil{tmp.Close();return err}
 if err=tmp.Close();err!=nil{return err}
 return os.Rename(tmp.Name(),path)
}
