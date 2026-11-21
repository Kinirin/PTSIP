package main

import (
  "path/filepath"
  "strings"
  "testing"
)

func TestDiagnosticResourceSources(t *testing.T) {
    root:=filepath.Join("..","..")
    for _,field:=range []string{"Plan_Document","Plan_Identity","Plan_Lifecycle","Plan_Unit",
        "Policy_Document","Policy_Identity","Policy_Lifecycle","Policy_Rules","Policy_Content"} {
        t.Run(field,func(t *testing.T) {
            report:=dispatch(record{"format":protocol,"operation":"inspect-resource","field":field},root)
            if report["binding"]!="DIAGNOSTIC_ONLY" || report["status"]!="RESOLVED" {
                t.Fatalf("resource %s: %#v",field,report)
            }
        })
    }
    valid:=dispatch(record{"format":protocol,"operation":"inspect-resource","field":"Plan_Lifecycle",
        "instance":record{"status":"Halt.Paused"}},root)
    if valid["status"]!="VALID" {t.Fatalf("valid status %#v",valid)}
    invalid:=dispatch(record{"format":protocol,"operation":"inspect-resource","field":"Plan_Lifecycle",
        "instance":record{"status":"DRAFT.Default"}},root)
    if invalid["status"]!="INVALID" {t.Fatalf("invalid status %#v",invalid)}
    unknown:=dispatch(record{"format":protocol,"operation":"inspect-resource","field":"Unknown"},root)
    stage,_:=unknown["reference_resolution"].(record)
    if unknown["status"]!="UNRESOLVED" || !strings.Contains(stage["problem"].(string),"UNREGISTERED_DIAGNOSTIC_FIELD") {
        t.Fatalf("unknown schema %#v",unknown)
    }
}
