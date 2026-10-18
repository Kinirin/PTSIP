// PTSIP developer-only validator orchestrator.
// Register future validator modules here; implementation remains in their own packages.
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "sort"

    schema_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/schema_check"
    test_mode "github.com/Kinirin/PTSIP/.ptsip/engine/developer/test_mode"
)

type Validator func(string) (any, int)

func runSchemaCheck(root string) (any, int) {
    result, code := schema_check.Run(root)
    return result, code
}

func runTestMode(root string) (any, int) {
    result, code := test_mode.Run(root)
    return result, code
}

var validators = map[string]Validator{
    "schema-check": runSchemaCheck,
    "test-mode-registry": runTestMode,
}

func main() {
    root := flag.String("root", ".", "repository root (read-only)")
    selected := flag.String("validator", "schema-check", "registered developer validator")
    flag.Parse()

    names := make([]string, 0, len(validators))
    for name := range validators {
        names = append(names, name)
    }
    sort.Strings(names)
    runner, registered := validators[*selected]
    if !registered {
        emit(map[string]any{
            "status": "UNRESOLVED",
            "reason": "UNREGISTERED_DEVELOPER_VALIDATOR",
            "available": names,
        })
        os.Exit(2)
    }
    absolute, err := filepath.Abs(*root)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }
    report, code := runner(absolute)
    emit(report)
    os.Exit(code)
}

func emit(report any) {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(report); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }
}
