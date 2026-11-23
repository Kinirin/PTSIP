package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/inspection"
)

func TestDependencyWorkflowTracksLiteralScriptReferences(t *testing.T) {
	root := t.TempDir()
	const workflow = `.github/workflows/build.yml`
	files := map[string]string{
		workflow: `on: workflow_dispatch
defaults:
  run:
    working-directory: tools
jobs:
  setup:
    steps:
      - run: ENV=dev bash -e './script; setup.sh'
  unit-test:
    defaults:
      run:
        working-directory: jobs
    steps:
      - run: cmd /c .\check.cmd && sh ../tools/script.sh
      - working-directory: .
        run: |
          pwsh -File './tools/Test File.ps1'; bash ./untracked.sh
          python3 -O ./tools/fixture.py
          python3 -m chosen ./not-a-script.py
          bash C:\not-local\external.sh
          bash bare-missing.sh
          # bash ./comment.sh
  publish:
    steps:
      - run: ./script.sh
`,
		"tools/script; setup.sh":        "not executed",
		"tools/script.sh":               "not executed",
		"jobs/check.cmd":                "not executed",
		"tools/Test File.ps1":           "not executed",
		"tools/fixture.py":              "not executed",
		"untracked.sh":                  "not observed",
		".github/workflows/broken.yaml": "jobs: []\n",
	}
	observed := []string{}
	for name, data := range files {
		filename := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if name != "untracked.sh" {
			observed = append(observed, name)
		}
	}
	input, err := json.Marshal(map[string]any{"root": root, "files": observed, "observation_errors": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := inspection.ObserveDependencies(root, input)
	if err != nil || !slices.Contains(result.Adapters, "github-actions") {
		t.Fatalf("workflow adapter unavailable: %#v, %v", result, err)
	}
	if result.CoverageComplete || !slices.ContainsFunc(result.Issues, func(issue inspection.DependencyIssue) bool {
		return issue.Adapter == "github-actions" && issue.Path == ".github/workflows/broken.yaml" && strings.Contains(issue.Message, "jobs is not a mapping")
	}) {
		t.Fatalf("invalid workflow did not report incomplete coverage: %#v", result.Issues)
	}
	want := map[string]struct{ path, phase, cwd string }{
		"setup:0:./script; setup.sh":        {"tools/script; setup.sh", "BUILD", "tools"},
		"unit-test:0:./check.cmd":           {"jobs/check.cmd", "TEST", "jobs"},
		"unit-test:0:../tools/script.sh":    {"tools/script.sh", "TEST", "jobs"},
		"unit-test:1:./tools/Test File.ps1": {"tools/Test File.ps1", "TEST", "."},
		"unit-test:1:./tools/fixture.py":    {"tools/fixture.py", "TEST", "."},
		"unit-test:1:./untracked.sh":        {"", "TEST", "."},
		"publish:0:./script.sh":             {"tools/script.sh", "RELEASE", "tools"},
	}
	for _, edge := range result.Edges {
		if edge.Adapter != "github-actions" {
			continue
		}
		id := strings.TrimPrefix(edge.EvidenceID, "github-actions:"+workflow+":")
		expected, exists := want[id]
		if !exists || edge.Source != workflow || edge.Phase != expected.phase || edge.EdgeType != "INVOKES" ||
			edge.Provenance != "DECLARED" || edge.WorkingDirectory == nil || *edge.WorkingDirectory != expected.cwd || edge.Line != nil {
			t.Fatalf("unexpected workflow evidence: %#v", edge)
		}
		if expected.path == "" {
			if edge.Resolution != "UNRESOLVED" || edge.TargetScope != "UNRESOLVED_TARGET" || edge.ResolvedPath != nil || edge.Note == nil {
				t.Fatalf("unobserved path should remain unresolved: %#v", edge)
			}
		} else if edge.Resolution != "RESOLVED" || edge.TargetScope != "PROJECT_COMPONENT" || edge.ResolvedPath == nil || *edge.ResolvedPath != expected.path {
			t.Fatalf("workflow invocation did not resolve through the inventory: %#v", edge)
		}
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("missing workflow dependencies: %#v", want)
	}
}
