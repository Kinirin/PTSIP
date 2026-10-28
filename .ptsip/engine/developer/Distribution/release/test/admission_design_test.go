package releasetest

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "testing"
  releaseversion "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release"
)

// Fixture locations are not runtime policy authority; production resolves from contracts.
const (
  contractIndexPath = ".ptsip/contracts/release/index.json"
  contractPath = ".ptsip/contracts/release/admission.json"
  rulesPath = ".ptsip/contracts/release/validation-rules.json"
  modeRegistryPath = ".github/test_modes.json"
  cutoverApprovalPath = "developer/policy/approvals/MPA-20261020-RELEASE-VERIFICATION-CUTOVER-KEEP-LATEST.yaml"
  verificationGapPath = ".github/verification-retirement.json"
)


func admissionJSON(root,path string)(map[string]any,error){
    b,err:=os.ReadFile(filepath.Join(root,filepath.FromSlash(path)))
    if err!=nil{return nil,err}
    var doc map[string]any
    if err=json.Unmarshal(b,&doc);err!=nil{return nil,err}
    if doc==nil{return nil,fmt.Errorf("empty JSON: %s",path)}
    return doc,nil
}
func admissionArray(m map[string]any,key string)[]any{
    a,_:=m[key].([]any)
    return a
}
func admissionStr(m map[string]any,key string)string{
    s,_:=m[key].(string)
    return s
}
func admissionObj(m map[string]any,key string)map[string]any{
    o,_:=m[key].(map[string]any)
    return o
}

func rootOfRepository(t *testing.T) string {
  t.Helper()
  root, err := filepath.Abs("../../../../../..")
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
    contractIndexPath,contractPath,rulesPath,modeRegistryPath,cutoverApprovalPath,verificationGapPath,
    ".github/workflows/tooling-release.yml",
  } { copyFixture(t,from,root,path) }
  contract, err := admissionJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  for _, group := range []string{"common_directions","error_directions"} {
    for _, raw := range admissionArray(contract,group) {
      approval := admissionStr(raw.(map[string]any),"approval_ref")
      copyFixture(t,from,root,approval)
    }
  }
  return root
}

func TestAdmissionApprovedProjectionIsNonPublishing(t *testing.T) {
  root := setupFixture(t)
  result,code := releaseversion.RunAdmission(root)
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
    _,code := releaseversion.RunAdmission(root)
    if code == 0 { t.Fatal("missing contract was accepted") }
  })
  t.Run("malformed",func(t *testing.T) {
    root := setupFixture(t)
    writeFixture(t,root,contractPath,[]byte("{"))
    _,code := releaseversion.RunAdmission(root)
    if code == 0 { t.Fatal("malformed contract was accepted") }
  })
}

func TestAdmissionRejectsAlteredApproval(t *testing.T) {
  root := setupFixture(t)
  contract,err := admissionJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  first := admissionArray(contract,"error_directions")[0].(map[string]any)
  path := admissionStr(first,"approval_ref")
  data,err := os.ReadFile(filepath.Join(root,filepath.FromSlash(path)))
  if err != nil { t.Fatal(err) }
  changed := strings.Replace(string(data),"response_model: B_CONDITIONAL_AUTOMATIC_RECOVERY","response_model: UNAUTHORIZED_REPLACEMENT",1)
  writeFixture(t,root,path,[]byte(changed))
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("approval/source mismatch was accepted") }
}

func TestAdmissionRejectsAlteredProjection(t *testing.T) {
  root := setupFixture(t)
  contract,err := admissionJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  record := admissionArray(contract,"common_directions")[1].(map[string]any)
  record["model"] = "UNAPPROVED_MODEL"
  data,err := json.Marshal(contract)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,contractPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("unapproved model was accepted") }
}

func TestAdmissionRejectsUnregisteredMode(t *testing.T) {
  root := setupFixture(t)
  registry,err := admissionJSON(root,modeRegistryPath)
  if err != nil { t.Fatal(err) }
  registry["modes"] = []any{}
  data,err := json.Marshal(registry)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,modeRegistryPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("unregistered mode was accepted") }
}

func TestAdmissionRejectsUnblockedRelease(t *testing.T) {
  root := setupFixture(t)
  gap,err := admissionJSON(root,verificationGapPath)
  if err != nil { t.Fatal(err) }
  admissionObj(gap,"release_verification_gap")["release_execution"] = "ENABLED"
  data,err := json.Marshal(gap)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,verificationGapPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("unblocked publication was accepted") }
}

func TestAdmissionRejectsSafetyBoundaryChanges(t *testing.T) {
  root := setupFixture(t)
  contract,err := admissionJSON(root,contractPath)
  if err != nil { t.Fatal(err) }
  admissionObj(contract,"guardrails")["security_recovery_requires_owner_approval"] = false
  data,err := json.Marshal(contract)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,contractPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("modified security constraint was accepted") }
}

func TestAdmissionRejectsPublisherWorkflowMutation(t *testing.T) {
  root := setupFixture(t)
  path := ".github/workflows/tooling-release.yml"
  data,err := os.ReadFile(filepath.Join(root,filepath.FromSlash(path)))
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,path,append(data,[]byte("\n# dangerous addition\n# gh release create\n")...))
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("publisher operation was accepted") }
}

func TestAdmissionRejectsMissingValidationRules(t *testing.T) {
  root := setupFixture(t)
  if err := os.Remove(filepath.Join(root,filepath.FromSlash(rulesPath))); err != nil { t.Fatal(err) }
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("missing rules contract was accepted") }
}

func TestAdmissionRejectsWeakenedRuleAssertions(t *testing.T) {
  root := setupFixture(t)
  rules,err := admissionJSON(root,rulesPath)
  if err != nil { t.Fatal(err) }
  admissionObj(rules,"admission_guardrails")["security_recovery_requires_owner_approval"] = false
  data,err := json.Marshal(rules)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,rulesPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("rules contract diverged from approval projection and was accepted") }
}

func TestAdmissionRejectsUnindexedOrInvalidRules(t *testing.T) {
  root := setupFixture(t)
  index,err := admissionJSON(root,contractIndexPath)
  if err != nil { t.Fatal(err) }
  index["validation_rules_ref"] = "../untrusted.json"
  data,err := json.Marshal(index)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,contractIndexPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("untrusted rules path accepted") }
}

func TestAdmissionRejectsRemovedContractAssertion(t *testing.T) {
  root := setupFixture(t)
  rules,err := admissionJSON(root,rulesPath)
  if err != nil { t.Fatal(err) }
  rules["document_assertions"] = []any{}
  data,err := json.Marshal(rules)
  if err != nil { t.Fatal(err) }
  writeFixture(t,root,rulesPath,data)
  _,code := releaseversion.RunAdmission(root)
  if code == 0 { t.Fatal("missing contract assertions accepted") }
}
