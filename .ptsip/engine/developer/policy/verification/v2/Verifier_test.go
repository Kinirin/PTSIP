package v2_test

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"

    authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
    verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
    v2 "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/v2"
)

func repoRoot(t *testing.T) string {
    t.Helper()
    root, err := filepath.Abs("../../../../../..")
    if err != nil { t.Fatal(err) }
    return root
}

func TestV2RejectsInactiveRegistry(t *testing.T) {
    root := t.TempDir()
    p := filepath.Join(root, filepath.FromSlash(".ptsip/contracts/policy/candidate.index.json"))
    if err := os.MkdirAll(filepath.Dir(p),0755); err != nil { t.Fatal(err) }
    raw, _ := json.Marshal(map[string]any{"active_registration":false, "active_policy_cutover":"DEFERRED"})
    if err := os.WriteFile(p,raw,0644); err != nil { t.Fatal(err) }
    if err := v2.Check(root, authoring.Document{}, func(any,any,[]any)(bool,error){return true,nil}); err == nil || !strings.Contains(err.Error(),"NOT_ACTIVE") {
        t.Fatalf("inactive candidate was accepted: %v",err)
    }
}

func TestV2RegisteredPolicyAndLegacyRejection(t *testing.T) {
    root := repoRoot(t)
    validate, err := verification.InstalledValidator()
    if err != nil { t.Skipf("PTSIP native validation unavailable: %v",err) }
    doc, err := authoring.ReadDocument(root,"developer/policy/ARCH/MPD-ARCH-0001.yaml")
    if err != nil { t.Fatal(err) }
    if err := v2.Check(root,doc,validate); err != nil { t.Fatal(err) }
    delete(doc,"Policy_Rules")
    if err := v2.Check(root,doc,validate); err == nil { t.Fatal("missing v2 rules accepted") }
    legacy, err := authoring.ReadDocument(root,"developer/policy/GOV/MPD-GOV-0006.yaml")
    if err != nil { t.Fatal(err) }
    if err := v2.Check(root,legacy,validate); err == nil { t.Fatal("v7 legacy shape accepted") }
}
