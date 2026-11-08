// policy-applicability is a Go entry for snapshot, assessment and local triage.
// The assessment remains read-only and cannot authorize policy or Rego writes.
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"

    "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/applicability"
)

func main() {
    input := flag.String("input", "-", "input JSON path, or - for stdin")
    output := flag.String("output", "-", "report JSON path, or - for stdout")
    summary := flag.Bool("summary", false, "print counts instead of full report")
    snapshot := flag.Bool("snapshot", false, "capture source-bound applicability input as JSON")
    local := flag.Bool("local", false, "run snapshot + assessment + grouped triage + checks")
    root := flag.String("root", "", "exact repository root (default: current Git repository)")
    skipTests := flag.Bool("skip-tests", false, "skip test commands in --local")
    flag.Parse()

    if (*snapshot && *local) || (*skipTests && !*local) {
        fail(fmt.Errorf("INCOMPATIBLE_OPERATION_FLAGS"))
    }
    if *snapshot || *local {
        if *input != "-" || *output != "-" || *summary {
            fail(fmt.Errorf("SNAPSHOT_LOCAL_DO_NOT_ACCEPT_ASSESSOR_INPUT_OUTPUT_FLAGS"))
        }
        if *local {
            report, err := applicability.RunLocal(*root, *skipTests)
            if err != nil {
                if report.Status != "" {
                    encoded, _ := json.MarshalIndent(report, "", "  ")
                    _, _ = os.Stderr.Write(append(encoded, '\n'))
                }
                fail(err)
            }
            printJSON(report)
            return
        }
        base := *root
        if base == "" { base = "." }
        request, err := applicability.CaptureInput(base)
        if err != nil { fail(err) }
        printJSON(request)
        return
    }

    var raw []byte
    var err error
    if *input == "-" {
        raw, err = io.ReadAll(io.LimitReader(os.Stdin, 32<<20))
    } else {
        raw, err = os.ReadFile(*input)
    }
    if err != nil { fail(err) }
    var request applicability.Input
    if err := json.Unmarshal(raw, &request); err != nil { fail(err) }
    result, err := applicability.Assess(request)
    if err != nil { fail(err) }
    encoded, err := json.MarshalIndent(result, "", "  ")
    if err != nil { fail(err) }
    encoded = append(encoded, '\n')
    if *output != "-" {
        if err := os.WriteFile(*output, encoded, 0600); err != nil { fail(err) }
    }
    if *summary {
        item := map[string]any{
            "schema_version": result.SchemaVersion, "source_head": result.SourceHead,
            "policy_count": result.PolicyCount, "rule_count": result.RuleCount,
            "binding_count": result.BindingCount, "counts": result.Counts,
            "evaluation_authorized": false, "execution_authorized": false,
        }
        encoded, err = json.Marshal(item)
        if err != nil { fail(err) }
        encoded = append(encoded, '\n')
    }
    if *summary || *output == "-" { _, _ = os.Stdout.Write(encoded) }
}

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { fail(err) }
    _, _ = os.Stdout.Write(append(encoded, '\n'))
}

func fail(err error) {
    fmt.Fprintln(os.Stderr, "POLICY_APPLICABILITY_FAILED:", err)
    os.Exit(2)
}
