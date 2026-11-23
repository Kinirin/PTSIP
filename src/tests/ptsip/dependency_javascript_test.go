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

func observeJavaScriptFixture(t *testing.T, files map[string]string, excluded string) inspection.DependencyScan {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for file, content := range files {
		filename := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if file != excluded {
			paths = append(paths, file)
		}
	}
	data, err := json.Marshal(map[string]any{"root": root, "files": paths, "observation_errors": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspection.ObserveDependencies(root, data)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestDependencyJavaScriptAndNPMEvidence(t *testing.T) {
	source := strings.Join([]string{
		`// require('hidden-comment')`,
		`const text = "require('hidden-string')";`,
		`const pattern = /require('hidden-regex')/;`,
		`if (flag) /require('hidden-condition-regex')/.test('');`,
		`import {`,
		`  Utility,`,
		`} from './util';`,
		`export * from './paths';`,
		`import 'ext';`,
		`const tool = require('dev');`,
		`const load = import(name);`,
		"const template = `require('hidden-template') ${import('./template')}`;",
		`const view = () => <div>require('hidden-jsx'){import('./jsx')}</div>;`,
		`const local = require('@scope/one/subpath');`,
		`require('./ambiguous');`,
		`require('./excluded');`,
		`object.require('hidden-property');`,
		`require('../../outside');`,
	}, "\n")
	report := observeJavaScriptFixture(t, map[string]string{
		"app/package.json": "\ufeff" + `{"name":"app","dependencies":{"ext":"^1.0","@scope/one":"workspace:*"},"devDependencies":{"dev":"^1.0"}}`,
		"app/main.tsx":     source,
		"app/util.ts":      "", "app/paths/index.ts": "", "app/template.ts": "", "app/jsx.ts": "",
		"app/ambiguous.js": "", "app/ambiguous.ts": "", "app/excluded.ts": "",
		"app/__tests__/smoke.ts":    `import 'ext';`,
		"packages/one/package.json": `{"name":"@scope/one"}`,
	}, "app/excluded.ts")
	if !report.CoverageComplete || !slices.Equal(report.Adapters, []string{"javascript-typescript", "npm-manifest"}) {
		t.Fatalf("unexpected scan issues or adapters: %v, %v", report.Issues, report.Adapters)
	}
	type expectation struct {
		kind, resolution, phase, path string
		line                          int
	}
	want := map[string]expectation{
		"./util":             {"IMPORTS", "RESOLVED", "UNKNOWN", "app/util.ts", 5},
		"./paths":            {"IMPORTS", "RESOLVED", "UNKNOWN", "app/paths/index.ts", 8},
		"ext":                {"IMPORTS", "EXTERNAL", "RUNTIME", "", 9},
		"dev":                {"IMPORTS", "EXTERNAL", "BUILD", "", 10},
		"<dynamic-import>":   {"LOADS", "DYNAMIC", "UNKNOWN", "", 11},
		"./template":         {"LOADS", "RESOLVED", "UNKNOWN", "app/template.ts", 12},
		"./jsx":              {"LOADS", "RESOLVED", "UNKNOWN", "app/jsx.ts", 13},
		"@scope/one/subpath": {"IMPORTS", "RESOLVED", "UNKNOWN", "packages/one/package.json", 14},
		"./ambiguous":        {"IMPORTS", "UNRESOLVED", "UNKNOWN", "", 15},
		"./excluded":         {"IMPORTS", "UNRESOLVED", "UNKNOWN", "", 16},
		"../../outside":      {"IMPORTS", "UNRESOLVED", "UNKNOWN", "", 18},
	}
	declarations, testImports := 0, 0
	for _, edge := range report.Edges {
		if edge.Adapter == "npm-manifest" {
			declarations++
			if edge.Provenance != "DECLARED" || edge.Line != nil {
				t.Errorf("manifest declaration treated as source observation: %+v", edge)
			}
			continue
		}
		if edge.Source == "app/__tests__/smoke.ts" {
			testImports++
			if edge.Phase != "TEST" || edge.Resolution != "EXTERNAL" {
				t.Errorf("test import classification: %+v", edge)
			}
			continue
		}
		path := ""
		if edge.ResolvedPath != nil {
			path = *edge.ResolvedPath
		}
		expected, ok := want[edge.Target]
		if !ok || edge.Provenance != "OBSERVED" || edge.EvidenceID == "" || edge.Line == nil ||
			(expectation{edge.EdgeType, edge.Resolution, edge.Phase, path, *edge.Line}) != expected {
			t.Errorf("unexpected source observation %q: %+v", edge.Target, edge)
		}
		delete(want, edge.Target)
	}
	if len(want) != 0 || declarations != 3 || testImports != 1 {
		t.Fatalf("missing observations: dependencies %v, declarations %d, test imports %d", want, declarations, testImports)
	}
}

func TestDependencyJavaScriptAmbiguousPackagesAndMalformedInputs(t *testing.T) {
	report := observeJavaScriptFixture(t, map[string]string{
		"package.json":              `{"dependencies":{"outer":"1","shared":"1"}}`,
		"packages/one/package.json": `{"name":"shared"}`,
		"packages/two/package.json": `{"name":"shared"}`,
		"broken/package.json":       `{"dependencies":{},"dependencies":{}}`,
		"broken/import.ts":          `import 'outer'; require('shared');`,
		"unknown.js":                "/* unterminated",
	}, "")
	if report.CoverageComplete || len(report.Issues) != 4 {
		t.Fatalf("want duplicate manifests, invalid package JSON and unsupported source diagnostics: %v", report.Issues)
	}
	var unresolved int
	for _, edge := range report.Edges {
		if edge.Source == "broken/import.ts" || edge.Target == "shared" {
			unresolved++
			if edge.Resolution != "UNRESOLVED" || edge.ResolvedPath != nil {
				t.Errorf("malformed or ambiguous dependency was resolved: %+v", edge)
			}
		}
	}
	if unresolved != 3 {
		t.Fatalf("missing unresolved dependency evidence: %v", report.Edges)
	}
}
