// Command test-modes is a reusable Go execution adapter for registered
// PTSIP verification modes. The Project Profile owns verification responsibility;
// this command only checks references and runs declared execution targets.
package main

import (
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "reflect"
 "regexp"
 "sort"
 "strings"

 "go.yaml.in/yaml/v3"
)

type registry struct {
 Version string `yaml:"version"`
 Groups map[string][]string `yaml:"groups"`
 Modes []mode `yaml:"modes"`
}
type mode struct {
 ID string `yaml:"id"`
 ComponentRef string `yaml:"component_ref"`
 Execution struct {
  Pytest []string `yaml:"pytest"`
  Go []string `yaml:"go"`
 } `yaml:"execution"`
}
type profile struct {
 Components []struct {
  ID string `yaml:"id"`
  Classification string `yaml:"classification"`
  Roles []string `yaml:"roles"`
  Include []string `yaml:"include"`
  AnalysisInputs []string `yaml:"analysis_inputs"`
 } `yaml:"components"`
}
type plannedMode struct {
 ID string `json:"id"`
 ComponentRef string `json:"component_ref"`
 Pytest []string `json:"pytest"`
 Go []string `json:"go"`
}
type selection struct {
 Resolution string `json:"resolution"`
 Plan []plannedMode `json:"plan"`
}
var validID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func load(root string) (registry, profile, error) {
 var r registry
 var p profile
 b, err := os.ReadFile(filepath.Join(root, ".github/test_modes.yaml"))
 if err != nil { return r,p,err }
 if err=yaml.Unmarshal(b,&r);err!=nil{return r,p,err}
 b,err=os.ReadFile(filepath.Join(root, ".ptsip/profiles/main.ptsip.yaml"))
 if err!=nil{return r,p,err}
 if err=yaml.Unmarshal(b,&p);err!=nil{return r,p,err}
 return r,p,nil
}
func owns(pattern, target string) bool {
 if pattern==target{return true}
 if strings.HasSuffix(pattern,"/**") {
  prefix:=strings.TrimSuffix(pattern,"/**")
  if target==prefix || strings.HasPrefix(target,prefix+"/"){return true}
 }
 if strings.HasSuffix(pattern,"/*.go") && strings.TrimSuffix(pattern,"/*.go")==target {return true}
 if strings.HasSuffix(pattern,"/*") {
  prefix:=strings.TrimSuffix(pattern,"/*")+"/"
  if strings.HasPrefix(target,prefix)&&!strings.Contains(strings.TrimPrefix(target,prefix),"/"){return true}
 }
 matched,err:=filepath.Match(pattern,target)
 return err==nil&&matched
}
func validateTarget(root, target string) error {
 if target=="" || strings.ContainsRune(target, 0) || strings.ContainsRune(target, 92) || strings.HasPrefix(target, "/") {return fmt.Errorf("invalid target %q",target)}
 for _,part:=range strings.Split(target,"/"){if part==".."||part=="."||part=="" {return fmt.Errorf("unsafe target %q",target)}}
 stat,err:=os.Stat(filepath.Join(root,filepath.FromSlash(target)))
 if err!=nil{return fmt.Errorf("target %q: %w",target,err)}
 if !stat.IsDir()&& !stat.Mode().IsRegular(){return fmt.Errorf("non-file target: %q",target)}
 return nil
}
func validate(root string,r registry,p profile) error {
 if r.Version!="3.0" {return fmt.Errorf("Test Mode Registry version requires 3.0, got %q",r.Version)}
 if len(r.Groups)!=2 {return errors.New("only SUPPLY and DEVELOPER groups are permitted")}
 components:=map[string]struct{classification string;roles []string;include []string;analysis []string}{}
 for _,c:=range p.Components {
  if _,exists:=components[c.ID];exists{return fmt.Errorf("duplicate Profile component %q",c.ID)}
  components[c.ID]=struct{classification string;roles []string;include []string;analysis []string}{c.Classification,c.Roles,c.Include,c.AnalysisInputs}
 }
 declared:=map[string]string{}
 for _,group:=range []string{"SUPPLY","DEVELOPER"}{
  members,ok:=r.Groups[group];if !ok {return fmt.Errorf("missing group %q",group)}
  if !sort.StringsAreSorted(members){return fmt.Errorf("%s IDs not sorted",group)}
  for _,id:=range members{
   if !validID.MatchString(id){return fmt.Errorf("invalid mode ID %q",id)}
   if _,exists:=declared[id];exists{return fmt.Errorf("duplicate group member %q",id)}
   declared[id]=group
  }
 }
 if len(declared)!=len(r.Modes){return fmt.Errorf("group membership count %d differs from mode count %d",len(declared),len(r.Modes))}
 seen:=map[string]bool{}
 usedComponents:=map[string]bool{}
 for _,m:=range r.Modes {
  group,ok:=declared[m.ID]
  if !ok||seen[m.ID]{return fmt.Errorf("unregistered/duplicate mode %q",m.ID)}
  seen[m.ID]=true
  c,ok:=components[m.ComponentRef]
  if !ok {return fmt.Errorf("mode %q references missing component %q",m.ID,m.ComponentRef)}
  if usedComponents[m.ComponentRef]{return fmt.Errorf("component %q has multiple modes",m.ComponentRef)}
  usedComponents[m.ComponentRef]=true
  verification:=false
  for _,role:=range c.roles {if role=="VERIFICATION"{verification=true}}
  if !verification||len(c.analysis)==0{return fmt.Errorf("mode %q lacks VERIFICATION responsibility or analysis inputs",m.ID)}
  expected:="DEVELOPER"
  if c.classification=="PRODUCT"{expected="SUPPLY"} else if c.classification!="DEVELOPMENT_TOOLING"&&c.classification!="DELIVERY" {return fmt.Errorf("unsupported verification classification %q",c.classification)}
  if group!=expected{return fmt.Errorf("mode %q group %q conflicts with Profile classification %q",m.ID,group,c.classification)}
  if len(m.Execution.Pytest)+len(m.Execution.Go)==0{return fmt.Errorf("mode %q declares no execution targets",m.ID)}
  for _,target:=range append(append([]string{},m.Execution.Pytest...),m.Execution.Go...) {
   if err:=validateTarget(root,target);err!=nil{return err}
   owned:=false
   for _,pattern:=range c.include{if owns(pattern,target){owned=true;break}}
   if !owned {return fmt.Errorf("mode %q target %q outside Profile ownership",m.ID,target)}
  }
 }
 return nil
}
func toPlan(m mode) plannedMode {
 return plannedMode{ID:m.ID,ComponentRef:m.ComponentRef,Pytest:append([]string{},m.Execution.Pytest...),Go:append([]string{},m.Execution.Go...)}
}
func resolve(r registry, csv string, planFile string) (selection,error) {
 modes:=map[string]mode{}
 for _,m:=range r.Modes {modes[m.ID]=m}
 requested:=map[string]bool{}
 if planFile!="" {
  b,err:=os.ReadFile(planFile);if err!=nil{return selection{},err}
  var raw selection
  if err=json.Unmarshal(b,&raw);err!=nil{return selection{},err}
  if len(raw.Plan)==0 {
   if raw.Resolution!="NO_TEST_MODE_REQUIRED" {return selection{},errors.New("empty plan without explicit no-verification resolution")}
   return selection{Resolution:raw.Resolution,Plan:[]plannedMode{}},nil
  }
  for _,item:=range raw.Plan {
   canonical,ok:=modes[item.ID]
   if !ok||requested[item.ID]||item.ComponentRef!=canonical.ComponentRef||!reflect.DeepEqual(item.Pytest,canonical.Execution.Pytest)||!reflect.DeepEqual(item.Go,canonical.Execution.Go) {
    return selection{},fmt.Errorf("automatic plan is not an exact registered mode: %q",item.ID)
   }
   requested[item.ID]=true
  }
 } else {
  for _,item:=range strings.Split(csv,",") {
   id:=strings.TrimSpace(item)
   if id==""||requested[id]{return selection{},fmt.Errorf("empty/duplicate requested mode %q",id)}
   if _,ok:=modes[id];!ok{return selection{},fmt.Errorf("unregistered requested mode %q",id)}
   requested[id]=true
  }
 }
 result:=selection{Resolution:"SELECTED",Plan:[]plannedMode{}}
 for _,m:=range r.Modes {if requested[m.ID]{result.Plan=append(result.Plan,toPlan(m))}}
 return result,nil
}
func execute(root string, plan selection) error {
 for _,m:=range plan.Plan {
  fmt.Fprintf(os.Stderr,"=== Registered Test Mode: %s / %s ===
",m.ID,m.ComponentRef)
  for _,dir:=range m.Go {
   args:=[]string{"-C",filepath.Join(root,filepath.FromSlash(dir)),"test","-count=1","-v","./..."}
   cmd:=exec.Command("go",args...)
   cmd.Dir=root;cmd.Stdout=os.Stdout;cmd.Stderr=os.Stderr
   if err:=cmd.Run();err!=nil{return fmt.Errorf("%s go verification: %w",m.ID,err)}
  }
  if len(m.Pytest)>0 {
   args:=append([]string{"-m","pytest","-q"},m.Pytest...)
   cmd:=exec.Command("python",args...);cmd.Dir=root;cmd.Stdout=os.Stdout;cmd.Stderr=os.Stderr
   if err:=cmd.Run();err!=nil{return fmt.Errorf("%s pytest verification: %w",m.ID,err)}
  }
 }
 return nil
}
func main(){
 root:=flag.String("root","../..","repository root relative to developer/automation")
 modeIDs:=flag.String("mode","","comma-separated registered mode IDs")
 planFile:=flag.String("plan","","exact JSON plan from automatic resolver")
 runTests:=flag.Bool("execute",false,"execute selected tests")
 flag.Parse()
 if *modeIDs!=""&&*planFile!=""{fmt.Fprintln(os.Stderr,"cannot mix manual and automatic selection");os.Exit(2)}
 abs,err:=filepath.Abs(*root);if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 reg,prof,err:=load(abs)
 if err==nil{err=validate(abs,reg,prof)}
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 if *modeIDs==""&&*planFile=="" {
  fmt.Printf("{"status":"PASS","registry_version":"3.0","mode_count":%d}
",len(reg.Modes));return
 }
 selection,err:=resolve(reg,*modeIDs,*planFile)
 if err==nil&&*runTests{err=execute(abs,selection)}
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 _=json.NewEncoder(os.Stdout).Encode(selection)
}
