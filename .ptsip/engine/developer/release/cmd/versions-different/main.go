package main

import (
 "encoding/json"
 "flag"
 "fmt"
 "os"
 "path/filepath"
 release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release"
)
func main(){
 cmd:=flag.String("command","check","preview, check, or apply (explicit only)")
 root:=flag.String("root",".","repository root")
 base:=flag.String("base","","exact base SHA")
 head:=flag.String("head","","exact checked-out head SHA")
 component:=flag.String("component","","component for explicit apply")
 target:=flag.String("target","","caller-selected target version")
 digest:=flag.String("version-sha256","","verified src/VERSION digest")
 explicit:=flag.Bool("explicit-user-decision",false,"explicit caller-selected version; never automatic")
 flag.Parse()
 abs,err:=filepath.Abs(*root)
 if err!=nil{fatal(err)}
 var output any
 switch *cmd {
 case "preview":output,err=release.PreviewVersionsDifferent(abs,*base,*head)
 case "check":output,err=release.CheckVersionsDifferent(abs,*base,*head)
 case "apply":
  err=release.ApplySelectedVersion(abs,*base,*head,*component,*target,*digest,*explicit)
  output=map[string]any{"status":"SELECTED_VERSION_APPLIED","component":*component,"target":*target,"publication_authorized":false}
 default:err=fmt.Errorf("unregistered command")
 }
 if err!=nil{fatal(err)}
 json.NewEncoder(os.Stdout).Encode(output)
}
func fatal(err error){fmt.Fprintln(os.Stderr,err);os.Exit(2)}
