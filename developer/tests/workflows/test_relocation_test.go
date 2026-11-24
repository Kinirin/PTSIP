package workflows_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestDeveloperGoTestLocation(t *testing.T) {
	root := testrepo.Root(t)
	for _, file := range trackedFiles(t, root, "ls-files", "--", ".ptsip") {
		if strings.HasSuffix(file, "_test.go") {
			t.Errorf("Developer Go tests belong under developer/tests: %s", file)
		}
	}
}

func TestDeveloperGoTestRelocationLedger(t *testing.T) {
	root := testrepo.Root(t)
	raw, err := os.ReadFile(filepath.Join(root, "developer", "tests", "testdata", "dot-ptsip-go-test-relocation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ledger struct {
		Baseline string `json:"baseline_head"`
		Files    []struct {
			Source       string   `json:"source"`
			Target       string   `json:"target"`
			Identifiers  []string `json:"test_identifiers"`
			BuildTags    []string `json:"build_constraints"`
			Replacements map[string]struct {
				Target string `json:"target"`
				Test   string `json:"test"`
				Reason string `json:"reason"`
			} `json:"retired_test_replacements"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		t.Fatal(err)
	}
	if _, err := hex.DecodeString(ledger.Baseline); err != nil || len(ledger.Baseline) != 40 || len(ledger.Files) == 0 {
		t.Fatal("relocation ledger requires an exact baseline and file inventory")
	}
	tracked := make(map[string]bool)
	for _, file := range trackedFiles(t, root, "ls-files", "--", "developer/tests") {
		tracked[file] = true
	}
	sources := make(map[string]bool, len(ledger.Files))
	targets := make(map[string]bool, len(ledger.Files))
	for _, file := range ledger.Files {
		if sources[file.Source] || targets[strings.ToLower(file.Target)] {
			t.Fatalf("duplicate source or colliding target for %s", file.Source)
		}
		if !strings.HasPrefix(file.Source, ".ptsip/") || !strings.HasSuffix(file.Source, "_test.go") ||
			path.Clean(file.Source) != file.Source {
			t.Fatalf("invalid relocation source: %s", file.Source)
		}
		sources[file.Source] = true
		targets[strings.ToLower(file.Target)] = true
		if !tracked[file.Target] || !strings.HasPrefix(file.Target, "developer/tests/") ||
			!strings.HasSuffix(file.Target, "_test.go") || path.Clean(file.Target) != file.Target {
			t.Fatalf("missing tracked relocation destination: %s", file.Target)
		}
		before := gitOutput(t, root, "show", ledger.Baseline+":"+file.Source)
		after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Target)))
		if err != nil {
			t.Fatal(err)
		}
		if got := topLevelTests(t, file.Source, before); !slices.Equal(got, file.Identifiers) {
			t.Errorf("inaccurate baseline identifiers for %s: %v", file.Source, got)
		}
		current := topLevelTests(t, file.Target, after)
		for _, name := range file.Identifiers {
			if !slices.Contains(current, name) && file.Replacements[name].Test == "" {
				t.Errorf("test %s missing from %s", name, file.Target)
			}
		}
		for name, replacement := range file.Replacements {
			if !slices.Contains(file.Identifiers, name) || replacement.Reason == "" || !tracked[replacement.Target] ||
				!strings.HasPrefix(replacement.Target, "developer/tests/") || !strings.HasSuffix(replacement.Target, "_test.go") {
				t.Fatalf("invalid retired test replacement for %s in %s", name, file.Source)
			}
			code, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(replacement.Target)))
			if err != nil || !slices.Contains(topLevelTests(t, replacement.Target, code), replacement.Test) {
				t.Fatalf("replacement test %s missing from %s: %v", replacement.Test, replacement.Target, err)
			}
		}
		if !slices.Equal(buildConstraints(before), file.BuildTags) || !slices.Equal(buildConstraints(after), file.BuildTags) {
			t.Errorf("build constraints differ for %s", file.Target)
		}
	}
	for _, file := range trackedFiles(t, root, "ls-tree", "-r", "--name-only", ledger.Baseline, "--", ".ptsip") {
		if strings.HasSuffix(file, "_test.go") && !sources[file] {
			t.Errorf("missing baseline test in relocation ledger: %s", file)
		}
	}
}

func trackedFiles(t *testing.T, root, command string, args ...string) []string {
	t.Helper()
	output := gitOutput(t, root, append([]string{command, "-z"}, args...)...)
	return strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
}

func gitOutput(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

func topLevelTests(t *testing.T, name string, raw []byte) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		name := function.Name.Name
		for _, prefix := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
			rest, matches := strings.CutPrefix(name, prefix)
			initial, _ := utf8.DecodeRuneInString(rest)
			if matches && !unicode.IsLower(initial) {
				names = append(names, name)
			}
		}
	}
	return names
}

func buildConstraints(raw []byte) []string {
	var constraints []string
	for line := range bytes.SplitSeq(raw, []byte("\n")) {
		text := strings.TrimSpace(string(line))
		if strings.HasPrefix(text, "package ") {
			break
		}
		if strings.HasPrefix(text, "//go:build ") || strings.HasPrefix(text, "// +build ") {
			constraints = append(constraints, text)
		}
	}
	return constraints
}
