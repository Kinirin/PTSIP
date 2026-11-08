package applicability

// Local runner: deterministic snapshot, read-only assessment and grouped triage.
// No policy mutation, contract activation or Task Engine registration occurs.
import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
)

type TriageExample struct {
    PolicyID string `json:"policy_id"`
    RuleID string `json:"rule_id"`
    PolicyPath string `json:"policy_path"`
    // SourceItemID locates the current Policy_Content item; SourcePointer
    // addresses the restored source document.
    SourceItemID string `json:"source_item_id"`
    SourcePointer string `json:"source_pointer"`
}

type TriageGroup struct {
    Plane string `json:"plane"`
    ContractKind string `json:"contract_kind"`
    State string `json:"state"`
    Reason string `json:"reason"`
    Count int `json:"count"`
    Examples []TriageExample `json:"examples"`
}

type Triage struct {
    Format string `json:"format"`
    SourceHead string `json:"source_head"`
    EvidenceRole string `json:"evidence_role"`
    PolicyMutationAuthorized bool `json:"policy_mutation_authorized"`
    EvaluationAuthorized bool `json:"evaluation_authorized"`
    ExecutionAuthorized bool `json:"execution_authorized"`
    SourceAssessment string `json:"source_assessment"`
    BindingCount int `json:"binding_count"`
    Counts map[string]int `json:"counts"`
    Groups []TriageGroup `json:"groups"`
}

type LocalReport struct {
    Status string `json:"status"`
    SourceHead string `json:"source_head"`
    ArtifactDirectory string `json:"artifact_directory"`
    LogDirectory string `json:"log_directory"`
    PolicyCount int `json:"policy_count"`
    RuleCount int `json:"rule_count"`
    BindingCount int `json:"binding_count"`
    Counts map[string]int `json:"counts"`
    Tests string `json:"tests"`
    PolicyMutationAuthorized bool `json:"policy_mutation_authorized"`
}

func BuildTriage(input Input, result Output) (Triage, error) {
    tr := Triage{
        Format:"ptsip-applicability-triage/v1", SourceHead:result.SourceHead,
        EvidenceRole:"NON_NORMATIVE_DERIVED", SourceAssessment:"assessment.json",
        BindingCount:result.BindingCount, Counts:result.Counts,
        Groups:[]TriageGroup{},
    }
    if input.SourceHead != result.SourceHead {
        return tr, fmt.Errorf("SOURCE_HEAD_MISMATCH")
    }
    planeByID := map[string]string{}
    sourceByRule := map[[2]string]TriageExample{}
    for _, p := range input.Policies {
        planeByID[p.ID] = p.Plane
        pointers := map[string]string{}
        for _, item := range p.Items {
            pointers[item.ID] = item.SourcePointer
        }
        for _, rule := range p.Rules {
            sourceByRule[[2]string{p.ID, rule.ID}] = TriageExample{
                PolicyID:p.ID, RuleID:rule.ID, PolicyPath:p.Path,
                SourceItemID:rule.SourceRef, SourcePointer:pointers[rule.SourceRef],
            }
        }
    }
    groups := map[string]*TriageGroup{}
    for _, row := range result.Decisions {
        plane, found := planeByID[row.PolicyID]
        if !found { return tr, fmt.Errorf("UNKNOWN_POLICY_ID: %s", row.PolicyID) }
        source, found := sourceByRule[[2]string{row.PolicyID, row.RuleID}]
        if !found || source.SourceItemID == "" || source.SourcePointer == "" ||
            source.PolicyPath != row.PolicyPath || source.SourcePointer != row.SourcePointer {
            return tr, fmt.Errorf("TRIAGE_RULE_SOURCE_MISMATCH: %s/%s", row.PolicyID, row.RuleID)
        }
        key := strings.Join([]string{plane,row.ContractKind,row.State,row.Reason}, "\x00")
        group, exists := groups[key]
        if !exists {
            group = &TriageGroup{
                Plane:plane, ContractKind:row.ContractKind, State:row.State,
                Reason:row.Reason, Examples:[]TriageExample{},
            }
            groups[key] = group
        }
        group.Count++
        if len(group.Examples) < 3 {
            group.Examples = append(group.Examples, source)
        }
    }
    actual := map[string]int{}
    keys := make([]string, 0, len(groups))
    for key, group := range groups {
        keys = append(keys, key)
        actual[group.State] += group.Count
    }
    if len(result.Decisions) != result.BindingCount {
        return tr, fmt.Errorf("BINDING_COUNT_MISMATCH")
    }
    if len(actual) != len(result.Counts) {
        return tr, fmt.Errorf("STATE_COUNT_MISMATCH")
    }
    for state, want := range result.Counts {
        if actual[state] != want {
            return tr, fmt.Errorf("STATE_COUNT_MISMATCH: %s", state)
        }
    }
    sort.Strings(keys)
    for _, key := range keys { tr.Groups = append(tr.Groups, *groups[key]) }
    return tr, nil
}

func writeSnapshot(path string, value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    return os.WriteFile(path, append(data, '\n'), 0600)
}

func runLogged(root, stdoutPath, stderrPath, command string, args ...string) error {
    out, err := os.Create(stdoutPath)
    if err != nil { return err }
    defer out.Close()
    errors, err := os.Create(stderrPath)
    if err != nil { return err }
    defer errors.Close()
    cmd := exec.Command(command, args...)
    cmd.Dir = root
    cmd.Stdout = out
    cmd.Stderr = errors
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("COMMAND_FAILED: %s %v: %w; stdout=%s stderr=%s",
            command, args, err, stdoutPath, stderrPath)
    }
    return nil
}

// RunLocal produces the same four machine outputs as the former run_local.py.
// It calls the Go assessor in-process. Existing registered conformance checks
// are run as tests rather than reimplemented as an inferior validator.
func RunLocal(root string, skipTests bool) (LocalReport, error) {
    var report LocalReport
    var err error
    if root == "" {
        root, err = defaultRepoRoot()
        if err != nil { return report, err }
    }
    root, err = filepath.Abs(root)
    if err != nil { return report, err }
    snapshot, err := CaptureInput(root)
    if err != nil { return report, err }
    result, err := Assess(snapshot)
    if err != nil { return report, err }
    triage, err := BuildTriage(snapshot, result)
    if err != nil { return report, err }

    artifactDir := filepath.Join(root, ".agent", "policy-applicability", snapshot.SourceHead)
    logDir := filepath.Join(os.TempDir(), "ptsip-policy-applicability", snapshot.SourceHead)
    if err := os.MkdirAll(artifactDir, 0700); err != nil { return report, err }
    if err := os.MkdirAll(logDir, 0700); err != nil { return report, err }
    if err := writeSnapshot(filepath.Join(artifactDir, "snapshot.json"), snapshot); err != nil { return report, err }
    if err := writeSnapshot(filepath.Join(artifactDir, "assessment.json"), result); err != nil { return report, err }
    if err := writeSnapshot(filepath.Join(artifactDir, "triage.json"), triage); err != nil { return report, err }

    summary := map[string]any{
        "schema_version":result.SchemaVersion, "source_head":result.SourceHead,
        "policy_count":result.PolicyCount, "rule_count":result.RuleCount,
        "binding_count":result.BindingCount, "counts":result.Counts,
        "evaluation_authorized":false, "execution_authorized":false,
    }
    if err := writeSnapshot(filepath.Join(artifactDir, "summary.json"), summary); err != nil { return report, err }

    report = LocalReport{
        Status:"PASS", SourceHead:snapshot.SourceHead,
        ArtifactDirectory:artifactDir, LogDirectory:logDir,
        PolicyCount:result.PolicyCount, RuleCount:result.RuleCount,
        BindingCount:result.BindingCount, Counts:result.Counts,
        Tests:"SKIPPED", PolicyMutationAuthorized:false,
    }

    if !skipTests {
        moduleRoot := filepath.Join(root, ".ptsip", "engine", "developer")
        if err := runLogged(root, filepath.Join(logDir,"go-test.stdout.log"),
            filepath.Join(logDir,"go-test.stderr.log"), "go",
            "-C", moduleRoot, "test", "./policy/applicability", "-count=1"); err != nil {
            report.Status = "FAIL"
            return report, err
        }
        if err := runLogged(root, filepath.Join(logDir,"pytest.stdout.log"),
            filepath.Join(logDir,"pytest.stderr.log"), "python", "-m", "pytest", "-q",
            "src/tests/ptsip/contracts/test_unified_policy.py"); err != nil {
            report.Status = "FAIL"
            return report, err
        }
        report.Tests = "PASSED"
    }
    current, err := gitOutput(root, "rev-parse", "HEAD")
    if err != nil { return report, err }
    if strings.TrimSpace(string(current)) != snapshot.SourceHead {
        return report, fmt.Errorf("SOURCE_HEAD_CHANGED_AFTER_TESTS")
    }
    return report, nil
}
