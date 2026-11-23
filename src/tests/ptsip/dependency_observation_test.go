package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/inspection"
)

func TestDependencyObservationUsesDiscoveryAndDoesNotFillCoverage(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                  "module example.test/tool\ngo 1.23\n",
		"cmd/main.go":             "package main\nimport (\n  \"fmt\"\n  _ \"example.test/tool/internal/model\"\n  _ \"example.test/tool/not-observed\"\n  _ \"external.test/modules/foo\"\n)\n",
		"cmd/main_test.go":        "package main\nimport \"testing\"\n",
		"internal/model/model.go": "package model\n",
		"not-observed/file.go":    "package notobserved\n",
		"old.py":                  "import invisible\n",
	}
	var paths []string
	for name, source := range files {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		if name != "not-observed/file.go" {
			paths = append(paths, name)
		}
	}
	raw, err := json.Marshal(map[string]any{"root": root, "files": paths, "observation_errors": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspection.ObserveDependencies(root, raw)
	if err != nil {
		t.Fatal(err)
	}
	if report.CoverageComplete || len(report.Issues) != 1 || report.Issues[0].Path != "old.py" ||
		!slices.Equal(report.Adapters, []string{"go-source"}) || report.EdgeCount != 5 {
		t.Fatalf("incomplete or unexpected evidence: %+v", report)
	}
	byTarget := map[string]inspection.DependencyEdge{}
	for _, edge := range report.Edges {
		byTarget[edge.Target] = edge
	}
	local := byTarget["example.test/tool/internal/model"]
	unobserved := byTarget["example.test/tool/not-observed"]
	if local.Resolution != "RESOLVED" || local.ResolvedPath == nil || *local.ResolvedPath != "internal/model/model.go" ||
		unobserved.Resolution != "UNRESOLVED" || unobserved.ResolvedPath != nil ||
		byTarget["fmt"].TargetScope != "PLATFORM" || byTarget["testing"].Phase != "TEST" ||
		byTarget["external.test/modules/foo"].TargetScope != "EXTERNAL_DEPENDENCY" {
		t.Fatalf("dependency classification diverged from observed files: %+v", report.Edges)
	}
}
