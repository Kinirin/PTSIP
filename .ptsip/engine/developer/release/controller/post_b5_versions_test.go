package releasecontroller

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"

    "go.yaml.in/yaml/v3"
)

// TestPublishedB5EngineVersionsAndProjections binds the owner's explicit
// version decision to the published 0.3.8b5 source without treating the
// historical publication as a new authorization to publish.
func TestPublishedB5EngineVersionsAndProjections(t *testing.T) {
    root, err := filepath.Abs("../../../../..")
    if err != nil { t.Fatal(err) }
    file, err := os.ReadFile(filepath.Join(root, "developer/policy/approvals/MPA-20261026-SEVEN-ENGINE-POST-B5-VERSIONS.yaml"))
    if err != nil { t.Fatal(err) }
    var decision struct {
        Approval struct {
            Published struct {
                Source string `yaml:"source_sha"`
                Package string `yaml:"package"`
            } `yaml:"previously_published_reference"`
            Versions map[string]struct {
                Previous string `yaml:"published_b5"`
                Next string `yaml:"approved_next"`
            } `yaml:"version_selection"`
        } `yaml:"approval"`
    }
    if err := yaml.Unmarshal(file, &decision); err != nil { t.Fatal(err) }
    if decision.Approval.Published.Source != "912a2b4956e856bec709cadc322cbd68588cdd5b" ||
        decision.Approval.Published.Package != "0.3.8b5" ||
        len(decision.Approval.Versions) != 7 {
        t.Fatal("published 0.3.8b5 engine version decision is incomplete")
    }
    // A historical b5-to-b6 decision is a baseline, not a permanent upper bound.
    // Consume the explicitly approved successor for the scoped Discovery change.
    successorFile, err := os.ReadFile(filepath.Join(root, "developer/policy/approvals/MPA-20261026-OFFICIAL-ENGINE-COMPATIBILITY-DIAGNOSTICS.yaml"))
    if err != nil { t.Fatal(err) }
    var successor struct { Approval struct {
        Decision string `yaml:"decision"`
        Operation string `yaml:"operation_id"`
        Versions struct { Discovery string `yaml:"discovery_engine"` } `yaml:"source_version_projections"`
    } `yaml:"approval"` }
    if err := yaml.Unmarshal(successorFile, &successor); err != nil { t.Fatal(err) }
    if successor.Approval.Decision != "APPROVED" || successor.Approval.Operation != "discovery.check-environment" || successor.Approval.Versions.Discovery == "" {
        t.Fatal("scoped Discovery successor approval is missing")
    }
    discovery := decision.Approval.Versions["discovery"]
    discovery.Next = successor.Approval.Versions.Discovery
    decision.Approval.Versions["discovery"] = discovery
    composition, err := ReadComposition(root)
    if err != nil { t.Fatal(err) }
    if len(composition.Engines) != 7 { t.Fatalf("unexpected official engine count: %d",len(composition.Engines)) }
    var contract Contract
    if err := ReadJSON(root, ContractPath, &contract); err != nil { t.Fatal(err) }
    if err := VerifyCompositionProjections(root, composition, contract); err != nil { t.Fatal(err) }

    var plan struct {
        Components []struct {
            Version string `json:"version"`
            CargoManifest string `json:"cargo_manifest_ref"`
        } `json:"components"`
    }
    if err := ReadJSON(root, "src/ptsip/engines/contracts/installation-plan.json", &plan); err != nil { t.Fatal(err) }
    if len(plan.Components) != 7 { t.Fatal("Package native installation plan inventory is incomplete") }
    matched := map[string]bool{}
    for _, entry := range plan.Components {
        id := strings.TrimSuffix(strings.TrimPrefix(entry.CargoManifest,"src/ptsip/engines/"),"/Cargo.toml")
        desired, found := decision.Approval.Versions[id]
        if !found || matched[id] || entry.Version != desired.Next { t.Fatalf("installation plan engine %s differs", id) }
        matched[id] = true
    }
    for id, desired := range decision.Approval.Versions {
        parse := func(v string) ([3]int,error) {
            out := [3]int{}
            if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(v) { return out, fmt.Errorf("invalid engine version %s",v) }
            _,err := fmt.Sscanf(v,"%d.%d.%d",&out[0],&out[1],&out[2])
            return out,err
        }
        prior,err := parse(desired.Previous)
        if err != nil { t.Fatal(err) }
        next,err := parse(desired.Next)
        if err != nil { t.Fatal(err) }
        greater := false
        for j:=0; j<3; j++ {
            if next[j] > prior[j] { greater=true; break }
            if next[j] < prior[j] { break }
        }
        if !greater { t.Fatalf("engine %s %s must exceed published b5 version %s",id,desired.Next,desired.Previous) }
        if composition.Engines[id] != desired.Next { t.Fatalf("composition mismatch: %s",id) }
        if !matched[id] { t.Fatalf("installation plan missing engine %s",id) }
        if err := ValidateEngineVersion(root,id,desired.Next); err != nil { t.Fatal(err) }
        lock,err := os.ReadFile(filepath.Join(root,"src/ptsip/engines",id,"Cargo.lock"))
        if err != nil { t.Fatal(err) }
        marker:=fmt.Sprintf("name = %q\nversion = %q","ptsip-engine-"+id,desired.Next)
        if !strings.Contains(string(lock),marker) { t.Fatalf("Cargo.lock projection mismatch: %s",id) }
        note:=fmt.Sprintf("docs/releasenote/engine/%s/%s-engine-%s.md",id,id,desired.Next)
        if _,err := os.Stat(filepath.Join(root,note)); err != nil { t.Fatalf("versioned engine release note missing: %s: %v",note,err) }
    }
}
