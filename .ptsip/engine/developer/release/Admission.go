// Package release evaluates non-normative, approval-bound design contracts.
// Release-specific rules live in .ptsip/contracts/release, not in Go.
package release

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "reflect"
    "strings"
)

// Only the bootstrap contract location is compiled into this engine.
const contractIndexPath = ".ptsip/contracts/release/index.json"

func repoPath(root, ref string) (string, error) {
    if ref == "" || filepath.IsAbs(ref) || strings.Contains(ref, "\\") ||
        filepath.ToSlash(filepath.Clean(ref)) != ref ||
        strings.HasPrefix(ref, "../") || strings.Contains(ref, "/../") {
        return "", fmt.Errorf("unsafe contract path %q", ref)
    }
    return filepath.Join(root, filepath.FromSlash(ref)), nil
}

func readText(root, ref string) (string, error) {
    p, err := repoPath(root, ref)
    if err != nil { return "", err }
    data, err := os.ReadFile(p)
    if err != nil { return "", fmt.Errorf("%s: %w", ref, err) }
    return strings.ReplaceAll(string(data), "\r\n", "\n"), nil
}

func readJSON(root, ref string) (map[string]any, error) {
    text, err := readText(root, ref)
    if err != nil { return nil, err }
    var m map[string]any
    if err := json.Unmarshal([]byte(text), &m); err != nil { return nil, fmt.Errorf("%s: %w", ref, err) }
    if m == nil { return nil, fmt.Errorf("%s: empty JSON object", ref) }
    return m, nil
}

func str(m map[string]any, k string) string {
    v, _ := m[k].(string)
    return v
}

func obj(m map[string]any, k string) map[string]any {
    v, _ := m[k].(map[string]any)
    return v
}

func array(m map[string]any, k string) []any {
    v, _ := m[k].([]any)
    return v
}

func requiredStrings(m map[string]any, k string) ([]string, error) {
    raw := array(m, k)
    if len(raw) == 0 { return nil, fmt.Errorf("missing required rule list %s", k) }
    out := make([]string, 0, len(raw))
    seen := make(map[string]bool)
    for _, elem := range raw {
        s, ok := elem.(string)
        if !ok || s == "" || seen[s] { return nil, fmt.Errorf("invalid or duplicate %s rule", k) }
        seen[s] = true
        out = append(out, s)
    }
    return out, nil
}

func compareFields(actual, expected map[string]any, label string) error {
    if len(expected) == 0 { return fmt.Errorf("%s has no rule assertions", label) }
    for k, want := range expected {
        got, exists := actual[k]
        if !exists || !reflect.DeepEqual(got, want) {
            return fmt.Errorf("%s: rule %q does not match contract", label, k)
        }
    }
    return nil
}

func verifyDocuments(root string, rules map[string]any) error {
    assertions := array(rules, "document_assertions")
    if len(assertions) < 2 { return fmt.Errorf("required document assertions absent") }
    paths := map[string]bool{}
    for _, entry := range assertions {
        check, ok := entry.(map[string]any)
        if !ok { return fmt.Errorf("invalid document assertion") }
        ref := str(check, "path")
        if ref == "" || paths[ref] { return fmt.Errorf("missing or duplicated asserted document") }
        paths[ref] = true
        source, err := readJSON(root, ref)
        if err != nil { return err }
        if err := compareFields(source, obj(check, "required_fields"), ref); err != nil { return err }
    }
    if !paths[contractIndexPath] || !paths[str(rules, "admission_ref")] {
        return fmt.Errorf("required indexed documents not covered")
    }
    return nil
}

func verifyRegistry(root string, index, rules map[string]any) error {
    reg := obj(rules, "registry")
    path := str(reg, "path")
    mode := str(reg, "mode_id")
    taskRef := str(reg, "mode_task_ref")
    if path == "" || path != str(index, "registry_ref") ||
       mode == "" || mode != str(index, "mode_id") ||
       str(rules, "admission_ref") != str(index, "contract_ref") ||
       str(rules, "approval_ref") != str(index, "approval_ref") {
        return fmt.Errorf("release index/registry/approval binding mismatch")
    }
    registry, err := readJSON(root, path)
    if err != nil { return err }
    matches := 0
    for _, entry := range array(registry, "modes") {
        m, ok := entry.(map[string]any)
        if !ok { return fmt.Errorf("invalid mode registry entry") }
        if str(m, "id") == mode {
            matches++
            if str(m, "task_ref") != taskRef { return fmt.Errorf("mode task binding differs from contract") }
        }
    }
    if matches != 1 { return fmt.Errorf("release mode must appear exactly once") }
    task := obj(obj(obj(registry, "tasks"), "execution"), taskRef)
    if err := compareFields(task, obj(reg, "task_expected"), "registered verifier task"); err != nil { return err }
    return nil
}

func verifyTextConstraints(root string, rule map[string]any, label string, forbid bool) error {
    path := str(rule, "path")
    if path == "" { return fmt.Errorf("%s source path not declared", label) }
    content, err := readText(root, path)
    if err != nil { return err }
    must, err := requiredStrings(rule, "must_contain")
    if err != nil { return fmt.Errorf("%s: %w", label, err) }
    for _, fragment := range must {
        if !strings.Contains(content, fragment) {
            return fmt.Errorf("%s missing required contract condition %q", label, fragment)
        }
    }
    if forbid {
        denied, err := requiredStrings(rule, "must_not_contain")
        if err != nil { return fmt.Errorf("%s: %w", label, err) }
        for _, fragment := range denied {
            if strings.Contains(content, fragment) {
                return fmt.Errorf("%s contains forbidden contract condition %q", label, fragment)
            }
        }
    }
    return nil
}

func verifyReleaseBlocked(root string, rule map[string]any) error {
    path := str(rule, "state_path")
    keys, err := requiredStrings(rule, "state_fields")
    if err != nil { return err }
    state, err := readJSON(root, path)
    if err != nil { return err }
    var value any = state
    for _, key := range keys {
        node, ok := value.(map[string]any)
        if !ok { return fmt.Errorf("publication state path is invalid") }
        next, found := node[key]
        if !found { return fmt.Errorf("publication state missing %s", key) }
        value = next
    }
    if str(rule, "expected") == "" || value != str(rule, "expected") {
        return fmt.Errorf("release publication is not in contract-declared blocked state")
    }
    return nil
}

func verifyProvenance(root string, item, rule map[string]any, isError bool) error {
    ref := str(item, "approval_ref")
    if !strings.HasPrefix(ref, "developer/policy/approvals/MPA-") ||
       !strings.HasSuffix(ref, ".yaml") {
        return fmt.Errorf("invalid approved provenance path %q", ref)
    }
    text, err := readText(root, ref)
    if err != nil { return err }
    required, err := requiredStrings(rule, "required_provenance_lines")
    if err != nil { return err }
    for _, line := range required {
        if !strings.Contains(text, line+"\n") { return fmt.Errorf("%s: missing approved provenance %q", ref, line) }
    }
    start := str(rule, "start_marker")
    end := str(rule, "end_marker")
    if start == "" || end == "" { return fmt.Errorf("direction boundaries missing") }
    a := strings.Index(text, start)
    if a < 0 { return fmt.Errorf("%s: selected direction missing", ref) }
    rest := text[a+len(start):]
    b := strings.Index(rest, end)
    if b < 0 { return fmt.Errorf("%s: unresolved boundary missing", ref) }
    direction := rest[:b]
    if direction == "" || direction != str(item, "selected_direction_yaml") {
        return fmt.Errorf("%s: approved direction does not match projection", ref)
    }
    modelField := str(rule, "model_field")
    if isError {
        modelField = str(rule, "error_model_field")
        categoryField := str(rule, "error_category_field")
        if categoryField == "" || str(item, "category") == "" ||
            !strings.Contains(direction, "    "+categoryField+": "+str(item, "category")+"\n") {
            return fmt.Errorf("%s: error category is not bound to approval", ref)
        }
    }
    if modelField == "" || str(item, "model") == "" ||
        !strings.Contains(direction, "    "+modelField+": "+str(item, "model")+"\n") {
        return fmt.Errorf("%s: model is not bound to approval", ref)
    }
    return nil
}

func verifyDirections(root string, contract, rules map[string]any) (int,int,error) {
    rule := obj(rules, "approval_direction_checks")
    topics, err := requiredStrings(rule, "common_topics")
    if err != nil { return 0,0,err }
    ids, err := requiredStrings(rule, "error_ids")
    if err != nil { return 0,0,err }
    common := array(contract, "common_directions")
    errors := array(contract, "error_directions")
    if len(common) != len(topics) || len(errors) != len(ids) {
        return 0,0,fmt.Errorf("approved direction coverage changed")
    }
    usedRefs := map[string]bool{}
    for i, raw := range common {
        item, ok := raw.(map[string]any)
        if !ok || str(item, "topic") != topics[i] {
            return 0,0,fmt.Errorf("common direction identity mismatch at %d", i)
        }
        ref := str(item, "approval_ref")
        if usedRefs[ref] { return 0,0,fmt.Errorf("duplicated provenance reference %s", ref) }
        usedRefs[ref]=true
        if err := verifyProvenance(root, item, rule, false); err != nil { return 0,0,err }
    }
    for i, raw := range errors {
        item, ok := raw.(map[string]any)
        if !ok || str(item, "error_id") != ids[i] ||
            !strings.HasPrefix(str(item,"category"),ids[i]+"_") {
            return 0,0,fmt.Errorf("error direction identity mismatch at %d",i)
        }
        ref := str(item,"approval_ref")
        if usedRefs[ref] { return 0,0,fmt.Errorf("duplicated provenance reference %s", ref) }
        usedRefs[ref]=true
        if err := verifyProvenance(root, item, rule, true); err != nil { return 0,0,err }
    }
    return len(common),len(errors),nil
}

// RunAdmission is a read-only verifier of *approved-direction projections*.
// Neither a successful design check nor edited contracts grant release authority.
func RunAdmission(root string) (any, int) {
    report := map[string]any{
        "status": "BLOCKED",
        "contract_validation": "UNVERIFIED",
        "release_verification": "NOT_ASSERTED",
        "publication_authorized": false,
        "contract_ref": contractIndexPath,
    }
    reject := func(err error) (any,int) {
        report["reason"] = err.Error()
        return report,2
    }
    index, err := readJSON(root, contractIndexPath)
    if err != nil { return reject(err) }
    rulesPath := str(index, "validation_rules_ref")
    if rulesPath == "" { return reject(fmt.Errorf("validation rule contract not registered")) }
    rules, err := readJSON(root, rulesPath)
    if err != nil { return reject(err) }
    // Structural safety floor: the runtime never treats a derived contract as
    // an active normative POLICY and always reports publication_authorized=false.
    if str(rules, "schema_version") != "ptsip-release-contract-check-rules/v1" ||
       str(rules, "artifact_class") != "DERIVED_NON_NORMATIVE_RELEASE_DESIGN_CHECKS" ||
       str(rules, "policy_authority") != "NONE" ||
       rules["publication_authorized"] != false ||
       str(rules, "contract_index_ref") != contractIndexPath ||
       str(rules, "admission_ref") != str(index, "contract_ref") ||
       str(rules, "approval_ref") != str(index, "approval_ref") {
        return reject(fmt.Errorf("invalid non-normative validation contract binding"))
    }
    if err := verifyDocuments(root, rules); err != nil { return reject(err) }
    if err := verifyRegistry(root,index,rules); err != nil { return reject(err) }
    if err := verifyTextConstraints(root,obj(rules,"cutover_approval"),"cutover approval",false); err != nil { return reject(err) }
    if str(obj(rules,"cutover_approval"),"path") != str(rules,"approval_ref") {
        return reject(fmt.Errorf("cutover approval reference mismatch"))
    }
    if err := verifyReleaseBlocked(root,obj(rules,"publication_block")); err != nil { return reject(err) }
    if err := verifyTextConstraints(root,obj(rules,"workflow"),"release workflow",true); err != nil { return reject(err) }
    contract, err := readJSON(root, str(rules,"admission_ref"))
    if err != nil { return reject(err) }
    if contract["publication_authorized"] != false ||
        index["publication_authorized"] != false ||
        index["fail_closed_on_unresolved"] != true ||
        str(index,"normative_policy_activation") != "PENDING" {
        return reject(fmt.Errorf("publication authority or normative POLICY activation is not permitted"))
    }
    if err := compareFields(obj(contract,"guardrails"),obj(rules,"admission_guardrails"),"release guardrails"); err != nil {
        return reject(err)
    }
    commonCount,errorCount,err := verifyDirections(root,contract,rules)
    if err != nil { return reject(err) }
    report["status"] = "DESIGN_CONTRACT_CONSISTENT"
    report["contract_validation"] = "APPROVED_PROVENANCE_BINDINGS_MATCH"
    report["contract_ref"] = str(rules,"admission_ref")
    report["approved_error_directions_checked"] = errorCount
    report["common_directions_checked"] = commonCount
    report["reason"] = "Design-only consistency; normative policy and publication are not authorized"
    return report,0
}
