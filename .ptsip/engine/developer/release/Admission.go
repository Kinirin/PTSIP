// Package release validates approved release directions against their source
// approval provenance. This design check never grants publication authority.
package release

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strings"
)

const (
  contractIndexPath = ".ptsip/contracts/release/index.json"
  contractPath = ".ptsip/contracts/release/admission.json"
  modeRegistryPath = ".github/test_modes.json"
  cutoverApprovalPath = "developer/policy/approvals/MPA-20261020-RELEASE-VERIFICATION-CUTOVER-KEEP-LATEST.yaml"
  verificationGapPath = ".github/verification-retirement.json"
)

func readJSON(root, path string) (map[string]any, error) {
  b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
  if err != nil { return nil, fmt.Errorf("%s: %w", path, err) }
  var obj map[string]any
  if err := json.Unmarshal(b, &obj); err != nil { return nil, fmt.Errorf("%s: %w", path, err) }
  if obj == nil { return nil, fmt.Errorf("%s: expected JSON object", path) }
  return obj, nil
}

func str(m map[string]any, key string) string {
  x, _ := m[key].(string)
  return x
}

func obj(m map[string]any, key string) map[string]any {
  v, _ := m[key].(map[string]any)
  return v
}

func array(m map[string]any, key string) []any {
  v, _ := m[key].([]any)
  return v
}

func safeApprovalPath(path string) bool {
  if !strings.HasPrefix(path, "developer/policy/approvals/MPA-") || !strings.HasSuffix(path, ".yaml") { return false }
  cleaned := filepath.ToSlash(filepath.Clean(path))
  return cleaned == path && !strings.Contains(path, "..")
}

func approvalDirection(text string) (string, error) {
  text = strings.ReplaceAll(text, "\r\n", "\n")
  marker := "  selected_direction:\n"
  pos := strings.Index(text, marker)
  if pos < 0 { return "", fmt.Errorf("selected_direction missing") }
  after := text[pos + len(marker):]
  end := strings.Index(after, "  unresolved:\n")
  if end < 0 { return "", fmt.Errorf("unresolved boundary missing") }
  selected := after[:end]
  if selected == "" { return "", fmt.Errorf("selected_direction empty") }
  return selected, nil
}

// verifyApprovedProjection guarantees that every projected rule is byte-for-byte
// bound to a recorded approved direction. A provenance record is not an active
// normative POLICY, and this function never evaluates publication readiness.
func verifyApprovedProjection(root string, item map[string]any, expectedKind string) error {
  path := str(item, "approval_ref")
  if !safeApprovalPath(path) { return fmt.Errorf("invalid approval path %q", path) }
  raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
  if err != nil { return fmt.Errorf("approval %s: %w", path, err) }
  text := strings.ReplaceAll(string(raw), "\r\n", "\n")
  for _, requirement := range []string{
    "  decision: APPROVED\n",
    "  policy_content_review_scope: DIRECTION\n",
    "  this_record: APPROVAL_PROVENANCE_NOT_ACTIVE_POLICY\n",
  } {
    if !strings.Contains(text, requirement) { return fmt.Errorf("%s: missing approved direction boundary %q", path, strings.TrimSpace(requirement)) }
  }
  projected := str(item, "selected_direction_yaml")
  source, err := approvalDirection(text)
  if err != nil { return fmt.Errorf("%s: %w", path, err) }
  if projected != source { return fmt.Errorf("%s: approved direction differs from projection", path) }
  model := str(item, "model")
  if model == "" { return fmt.Errorf("%s: projected model missing", path) }
  field := "model"
  if expectedKind == "error" {
    field = "response_model"
    category := str(item, "category")
    if category == "" || !strings.Contains(source, "    error_category: "+category+"\n") {
      return fmt.Errorf("%s: error category not bound to approved source", path)
    }
  }
  if !strings.Contains(source, "    "+field+": "+model+"\n") {
    return fmt.Errorf("%s: model not bound to approved source", path)
  }
  return nil
}

func verifyRegistry(root string, index map[string]any) error {
  if str(index, "schema_version") != "ptsip-release-contract-index/v1" ||
    str(index, "contract_ref") != contractPath ||
    str(index, "registry_ref") != modeRegistryPath ||
    str(index, "approval_ref") != cutoverApprovalPath ||
    str(index, "engine_entry") != ".ptsip/engine/developer/main.go" ||
    str(index, "verifier_root") != ".ptsip/engine/developer/release/" ||
    str(index, "validator_name") != "release-admission-check" ||
    str(index, "mode_id") != "release-admission-design-check" ||
    str(index, "normative_policy_activation") != "PENDING" ||
    index["publication_authorized"] != false ||
    index["fail_closed_on_unresolved"] != true ||
    index["legacy_release_verifier_fallback"] != false {
    return fmt.Errorf("release index violates approved fail-closed design route")
  }
  registry, err := readJSON(root, modeRegistryPath)
  if err != nil { return err }
  found := 0
  for _, raw := range array(registry, "modes") {
    entry, ok := raw.(map[string]any)
    if !ok { return fmt.Errorf("invalid mode entry") }
    if str(entry, "id") == "release-admission-design-check" {
      found++
      if str(entry, "task_ref") != "release-admission-contract-check" { return fmt.Errorf("incorrect release task binding") }
    }
  }
  if found != 1 { return fmt.Errorf("release verification mode must be registered exactly once") }
  task := obj(obj(obj(registry,"tasks"),"execution"),"release-admission-contract-check")
  if str(task, "module_root") != ".ptsip/engine/developer" ||
    str(task, "package") != "." ||
    str(task, "validator") != "release-admission-check" ||
    str(task, "contract_ref") != contractPath ||
    task["publication_authorized"] != false {
    return fmt.Errorf("registered developer Go verifier differs from approved path")
  }
  return nil
}

func verifyReleaseGap(root string) error {
  gap, err := readJSON(root, verificationGapPath)
  if err != nil { return err }
  if str(obj(gap, "release_verification_gap"), "release_execution") != "BLOCKED" {
    return fmt.Errorf("release publication must remain blocked")
  }
  return nil
}

func verifyGuardrails(contract map[string]any) error {
  guards := obj(contract,"guardrails")
  required := map[string]string{
    "independent_logical_release_id":"REQUIRED",
    "publisher_specific_outcome":"INDEPENDENT",
    "multiple_error_classification":"INDEPENDENT",
    "verification_evidence_reuse":"SCOPED_CONDITIONAL",
    "admission_after_publication_error":"REEVALUATE_BEFORE_NEXT_ATTEMPT",
    "unresolved_policy_or_verification":"FAIL_CLOSED",
    "actual_github_or_pypi_publication":"BLOCKED",
  }
  for k, want := range required {
    if str(guards,k) != want { return fmt.Errorf("guardrail %s changed",k) }
  }
  if guards["admission_success_is_blanket_recovery_authorization"] != false ||
    guards["security_recovery_requires_owner_approval"] != true {
    return fmt.Errorf("mandatory authorization boundaries missing")
  }
  return nil
}

// RunAdmission validates machine-readable approval-derived contract integrity.
// A successful contract check reports DESIGN_CONTRACT_CONSISTENT, never PASS
// for normative policy validation or publication admission.
func RunAdmission(root string) (any, int) {
  report := map[string]any{
    "status":"BLOCKED",
    "contract_validation":"UNVERIFIED",
    "release_verification":"NOT_ASSERTED",
    "publication_authorized":false,
    "contract_ref":contractPath,
  }
  reject := func(err error) (any,int) {
    report["reason"] = err.Error()
    return report,2
  }
  index, err := readJSON(root, contractIndexPath)
  if err != nil { return reject(err) }
  if err := verifyRegistry(root,index); err != nil { return reject(err) }
  if err := verifyReleaseGap(root); err != nil { return reject(err) }
  contract, err := readJSON(root, contractPath)
  if err != nil { return reject(err) }
  if str(contract,"schema_version") != "ptsip-release-admission-direction-projection/v1" ||
    str(contract,"artifact_class") != "DERIVED_MACHINE_CONTRACT_FROM_APPROVED_PROVENANCE" ||
    str(contract,"normative_policy_status") != "NOT_ACTIVE_NORMATIVE_POLICY" ||
    str(contract,"conflict_resolution_approval_ref") != cutoverApprovalPath ||
    str(contract,"old_root_policy_fallback") != "FORBIDDEN" ||
    str(contract,"mode_registry") != modeRegistryPath ||
    str(contract,"developer_validator") != ".ptsip/engine/developer/main.go" ||
    contract["publication_authorized"] != false {
    return reject(fmt.Errorf("contract metadata violates approved non-normative authority boundary"))
  }
  if err := verifyGuardrails(contract); err != nil { return reject(err) }
  cutover, err := os.ReadFile(filepath.Join(root,filepath.FromSlash(cutoverApprovalPath)))
  if err != nil { return reject(err) }
  if !strings.Contains(string(cutover),"  conflict_resolution: KEEP_LATEST\n") ||
    !strings.Contains(string(cutover),"  decision: APPROVED\n") ||
    !strings.Contains(string(cutover),"    publication_execution: BLOCKED_UNTIL_SEPARATE_POLICY_AND_VALIDATION_APPROVAL\n") {
    return reject(fmt.Errorf("missing approved KEEP_LATEST fail-closed cutover record"))
  }
  expectedTopics := []string{"baseline","release-identity","multiple-errors","evidence-validity","conditional-resumption"}
  common := array(contract,"common_directions")
  if len(common) != len(expectedTopics) { return reject(fmt.Errorf("expected five common admission directions")) }
  for i, raw := range common {
    item, ok := raw.(map[string]any)
    if !ok || str(item,"topic") != expectedTopics[i] { return reject(fmt.Errorf("incorrect common admission direction %d",i)) }
    if err := verifyApprovedProjection(root,item,"common"); err != nil { return reject(err) }
  }
  errors := array(contract,"error_directions")
  if len(errors) != 14 { return reject(fmt.Errorf("expected all fourteen error directions")) }
  for i, raw := range errors {
    item, ok := raw.(map[string]any)
    if !ok || str(item,"error_id") != fmt.Sprintf("E%02d",i+1) ||
      !strings.HasPrefix(str(item,"category"),fmt.Sprintf("E%02d_",i+1)) {
      return reject(fmt.Errorf("error sequence, category or identity mismatch at %d",i))
    }
    if err := verifyApprovedProjection(root,item,"error"); err != nil { return reject(err) }
  }
  report["status"] = "DESIGN_CONTRACT_CONSISTENT"
  report["contract_validation"] = "APPROVED_PROVENANCE_BINDINGS_MATCH"
  report["approved_error_directions_checked"] = 14
  report["common_directions_checked"] = 5
  report["reason"] = "Success does not certify normative policy or authorize release publication"
  return report,0
}
