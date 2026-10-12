package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	wheel "github.com/Kinirin/PTSIP/.github/checks/release/engine-verification"
)

func main() {
	mode := "build"
	if len(os.Args) > 1 && (os.Args[1] == "admit" || os.Args[1] == "gate") {
		mode = os.Args[1]
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	root := flag.String("root", ".", "repository root")
	kind := flag.String("kind", "", "one registered engine")
	target := flag.String("target", "", "native host target")
	binary := flag.String("binary", "", "already-built native executable")
	output := flag.String("output", "", "local artifact directory")
	source := flag.String("source-sha", "", "exact source revision")
	repository := flag.String("repository", "", "GitHub repository owner/name")
	runID := flag.Int64("run-id", 0, "exact GitHub Actions run ID")
	flag.Parse()
	if mode == "gate" {
		result, err := wheel.Gate(*root, wheel.GateOptions{Repository: *repository, SourceSHA: *source, Kind: *kind, RunID: *runID}, os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "ENGINE_RELEASE_GATE_FAILED:", err)
			os.Exit(2)
		}
		json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	if mode == "admit" {
		if err := wheel.Admit(*root); err != nil {
			fmt.Fprintln(os.Stderr, "ENGINE_VERIFICATION_BLOCKED:", err)
			os.Exit(2)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ENGINE_VERIFICATION_ADMITTED", "release_gate": "ENABLED", "release_gate_scope": "SELECTED_ENGINE_VERIFICATION", "publication": "BLOCKED"})
		return
	}
	if flag.NArg() != 0 || *kind == "" || *binary == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "ENGINE_WHEEL_BUILD_BLOCKED: required named arguments are missing")
		os.Exit(2)
	}
	result, err := wheel.Build(*root, wheel.Options{Kind: *kind, Target: *target, Binary: *binary, Output: *output, SourceSHA: *source})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ENGINE_WHEEL_BUILD_BLOCKED:", err)
		os.Exit(2)
	}
	if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
