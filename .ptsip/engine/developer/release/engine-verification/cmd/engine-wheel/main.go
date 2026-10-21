package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	wheel "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/engine-verification"
)

func main() {
	mode := "build"
	if len(os.Args) > 1 && (os.Args[1] == "admit" || os.Args[1] == "admit-publication" || os.Args[1] == "gate" || os.Args[1] == "abi-check" || os.Args[1] == "publish-github" || os.Args[1] == "verify-pypi") {
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
	input := flag.String("input", "", "original verification artifacts inside the repository")
	verifiedEngines := flag.String("verified-engines", "all", "engine selection used by the original verification run")
	flag.Parse()
	for _, forbidden := range []string{"admit", "admit-publication", "publish-github", "verify-pypi"} {
		if mode == forbidden {
			fmt.Fprintln(os.Stderr, "ENGINE_PUBLICATION_BLOCKED: migrated engine verifier has no active admission authority")
			os.Exit(2)
		}
	}
	if mode == "admit-publication" {
		if err := wheel.AdmitPublication(*root); err != nil {
			fmt.Fprintln(os.Stderr, "ENGINE_PUBLICATION_BLOCKED:", err)
			os.Exit(2)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ENGINE_PUBLICATION_ADMITTED", "scope": "SELECTED_REGISTERED_ENGINES_ONLY", "tool": "PRESERVE_PUBLISHED_0.3.8a5"})
		return
	}
	if mode == "abi-check" {
		if err := wheel.VerifyLinuxABI(*binary); err != nil {
			fmt.Fprintln(os.Stderr, "LINUX_WHEEL_ABI_FAILED:", err)
			os.Exit(2)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "PASS", "wheel_tag": "manylinux_2_39_x86_64"})
		return
	}
	if mode == "publish-github" || mode == "verify-pypi" {
		opts := wheel.PublicationOptions{GateOptions: wheel.GateOptions{Repository: *repository, SourceSHA: *source, Kind: *kind, RunID: *runID}, VerifiedEngines: *verifiedEngines, Input: *input, Output: *output}
		var result wheel.PublicationBundle
		var err error
		if mode == "publish-github" {
			result, err = wheel.PublishGitHub(*root, opts, os.Getenv("GITHUB_TOKEN"))
		} else {
			result, err = wheel.VerifyPublishedPyPI(*root, opts, os.Getenv("GITHUB_TOKEN"))
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "ENGINE_PUBLICATION_FAILED:", err)
			os.Exit(2)
		}
		if mode == "publish-github" && os.Getenv("GITHUB_OUTPUT") != "" {
			f, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			fmt.Fprintf(f, "pypi_pending=%t\n", result.PyPIPending)
			f.Close()
		}
		json.NewEncoder(os.Stdout).Encode(result)
		return
	}
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
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ENGINE_VERIFICATION_ADMITTED", "release_gate": "ENABLED", "release_gate_scope": "SELECTED_ENGINE_VERIFICATION", "publication_authority": "SEPARATE_ENGINE_PUBLICATION_CONTRACT"})
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
