// policy-applicability consumes an exact JSON snapshot from input.py.
// Assessment results are provisional and cannot authorize Rego or policy writes.
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
    flag.Parse()
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
func fail(err error) {
    fmt.Fprintln(os.Stderr, "POLICY_APPLICABILITY_FAILED:", err)
    os.Exit(2)
}
