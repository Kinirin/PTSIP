package release

import (
  "encoding/json"
  "os"
  "path/filepath"
  "strings"
  "testing"
)

func rootOfRepository(t *testing.T) string {
  t.Helper()
  root, err := filepath.Abs("../../../..")
  if err != nil { t.Fatal(err) }
  return root
}

func copyFixture(t *testing.T, source, dest, path string) {
  t.Helper()
  b, err := os.ReadFile(filepath.Join(source,filepath.FromSlash(path)))
  if err != nil { t.Fatal(err) }
  out := filepath.Join(dest,filepath.FromSlash(path))
  if err := os.MkdirAll(filepath.Dir(out),0755); err != nil { t.Fatal(err) }
  if err := os.WriteFile(out,b,0644); err != nil { t.Fatal(err) }
}

func writeFixture(t *testing.T, root,path string, data []byte) {
  t.Helper()
  if err := os.WriteFile(filepath.Join(root,filepath.FromSlash(path)),data,0644); err != nil { t.Fatal(err) }
}

func setupFixture(t *testing.T) string {
  t.Helper()
  from := rootOfRepository(t)
  root := t.TempDir()
  for _, path := range []string{
    contractIndexPath,contractPath,modeRegistryPath,cutoverApprovalPath,verificationGapPath,
    ".github/workflows/tooling-release.yml",
  } { copyFixture(t,from,root,path) }
  contract, err := readJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  for _, group := range []string{"common_directions","error_directions"} {
    for _, raw := range array(contract,group) {
      approval := str(raw.(map[string]any),"approval_ref")
      copyFixture(t,from,root,approval)
    }
  }
  return root
}

func TestAdmissionApprovedProjectionIsNonPublishing(t *testing.T) {
  root := setupFixture(t)
  result,code := RunAdmission(root)
  report := result.(map[string]any)
  if code != 0 || report["status"] != "DESIGN_CONTRACT_CONSISTENT" {
    t.Fatalf("expected validated design projection: code=%d result=%v",code,report)
  }
  if report["publication_authorized"] != false ||
    report["release_verification"] != "NOT_ASSERTED" {
    t.Fatalf("validator must not authorize publication: %v",report)
  }
}

func TestAdmissionFailsClosedOnMissingOrInvalidContract(t *testing.T) {
  t.Run("missing",func(t *testing.T) {
    root := setupFixture(t)
    if err := os.Remove(filepath.Join(root,filepath.FromSlash(contractPath))); err != nil { t.Fatal(err) }
    _,code := RunAdmission(root)
    if code == 0 { t.Fatal("missing contract was accepted") }
  })
  t.Run("malformed",func(t *testing.T) {
    root := setupFixture(t)
    writeFixture(t,root,contractPath,[]byte("{"))
    _,code := RunAdmission(root)
    if code == 0 { t.Fatal("malformed contract was accepted") }
  })
}

func TestAdmissionRejectsAlteredApproval(t *testing.T) {
  root := setupFixture(t)
  contract,err := readJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  first := array(contract,"error_directions")[0].(map[string]any)
  path := str(first,"approval_ref")
  data,err := os.ReadFile(filepath.Join(root,filepath.FromSlash(path)))
  if err != nil { t.Fatal(err) }
  changed := strings.Replace(string(data),"response_model: B_CONDITIONAL_AUTOMATIC_RECOVERY","response_model: UNAUTHORIZED_REPLACEMENT",1)
  writeFixture(t,root,path,[]byte(changed))
  _,code := RunAdmission(root)
  if code == 0 { t.Fatal("approval/source mismatch was accepted") }
}

func TestAdmissionRejectsAlteredProjection(t *testing.T) {
  root := setupFixture(t)
  contract,err := readJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  record := array(contract,"common_directions")[1].(map[string]any)
  record["model"] = "UNAPPROVED_MODEL"
  data,err := json.Marshal(contract)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,contractPath,data)
  _,code := RunAdmission(root)
  if code == 0 { t.Fatal("unapproved model was accepted") }
}

func TestAdmissionRejectsUnregisteredMode(t *testing.T) {
  root := setupFixture(t)
  registry,err := readJSON(root,modeRegistryPath)
  if err != nil { t.Fatal(err) }
  registry["modes"] = []any{}
  data,err := json.Marshal(registry)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,modeRegistryPath,data)
  _,code := RunAdmission(root)
  if code == 0 { t.Fatal("unregistered mode was accepted") }
}

func TestAdmissionRejectsUnblockedRelease(t *testing.T) {
  root := setupFixture(t)
  gap,err := readJSON(root,verificationGapPath)
  if err != nil { t.Fatal(err) }
  obj(gap,"release_verification_gap")["release_execution"] = "ENABLED"
  data,err := json.Marshal(gap)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,verificationGapPath,data)
  _,code := RunAdmission(root)
  if code == 0 { t.Fatal("unblocked publication was accepted") }
}

func TestAdmissionRejectsSafetyBoundaryChanges(t *testing.T) {
  root := setupFixture(t)
  contract,err := readJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  obj(contract,"guardrails")["security_recovery_requires_owner_approval"] = false
  data,err := json.Marshal(contract)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,contractPath,data)
  _,code := RunAdmission(root)
  if code == 0 { t.Fatal("modified security constraint was accepted") }
}

func TestAdmissionRejectsPublisherWorkflowMutation(t *testing.T) {
  root := setupFixture(t)
  path := ".github/workflows/tooling-release.yml"
  data,err := os.ReadFile(filepath.Join(root,filepath.FromSlash(path)))
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,path,append(data,[]byte("\n# dangerous addition\n# gh release create\n")...))
  _,code := RunAdmission(root)
  if code == 0 { t.Fatal("publisher operation was accepted") }
}
