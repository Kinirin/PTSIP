package contracts_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/inspection"
)

func observePythonFixture(t *testing.T, fixtures map[string]string, omitted string) inspection.DependencyScan {
	t.Helper()
	root := repositoryFixture(t)
	files := []string{}
	for path, text := range fixtures {
		writeRepositoryFixture(t, root, path, text)
		if path != omitted {
			files = append(files, path)
		}
	}
	data, err := json.Marshal(object{"root": root, "files": files, "observation_errors": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := inspection.ObserveDependencies(root, data)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestDependencyPythonUsesObservedImportsAndPackageRoots(t *testing.T) {
	result := observePythonFixture(t, map[string]string{
		"pyproject.toml": `[project]
dependencies = ["vendor-package>=1", "ambiguous", "requests"]
[tool.setuptools.packages.find]
where = ["tools/generated"]
`,
		"src/pkg/__init__.py":        "",
		"src/pkg/alpha.py":           "",
		"src/pkg/beta.py":            "",
		"src/pkg/hidden.py":          "",
		"src/requests/__init__.py":   "",
		"src/ambiguous.py":           "",
		"src/ambiguous/__init__.py":  "",
		"tools/generated/helpers.py": "",
		"src/pkg/main.py": `from __future__ import annotations
import os, pkg.alpha as first
from . import beta as other
from .alpha import exported
from . import hidden
from .. import unavailable
import vendor_package
import helpers
from requests.unobserved import target
import ambiguous
quoted = "import fabricated"
match reading:
    case 1:
        import sys
`,
	}, "src/pkg/hidden.py")
	if !result.CoverageComplete || !slices.Contains(result.Adapters, "python") || len(result.Edges) != 12 {
		t.Fatalf("unexpected Python dependency coverage: %+v", result)
	}
	for _, edge := range result.Edges {
		if edge.Adapter != "python" || edge.EdgeType != "IMPORTS" || edge.Phase != "UNKNOWN" || edge.Provenance != "OBSERVED" || edge.Line == nil {
			t.Fatalf("unexpected Python import evidence fields: %+v", edge)
		}
		var resolved string
		switch edge.Target {
		case "pkg.alpha":
			resolved = "src/pkg/alpha.py"
		case "pkg.beta":
			resolved = "src/pkg/beta.py"
		case "pkg":
			resolved = "src/pkg/__init__.py"
		case "helpers":
			resolved = "tools/generated/helpers.py"
		case "os", "sys", "__future__":
			if edge.TargetScope != "PLATFORM" || edge.Resolution != "EXTERNAL" {
				t.Fatalf("stdlib target was not classified: %+v", edge)
			}
		case "vendor_package":
			if edge.TargetScope != "EXTERNAL_DEPENDENCY" || edge.Resolution != "EXTERNAL" {
				t.Fatalf("declared dependency was not classified: %+v", edge)
			}
		case "..", "requests.unobserved", "ambiguous":
			if edge.ResolvedPath != nil || edge.Resolution != "UNRESOLVED" {
				t.Fatalf("ambiguous or missing local target was inferred: %+v", edge)
			}
		default:
			t.Fatalf("unexpected Python dependency edge: %+v", edge)
		}
		if resolved != "" && (edge.ResolvedPath == nil || *edge.ResolvedPath != resolved || edge.Resolution != "RESOLVED") {
			t.Fatalf("Python module did not resolve to %q: %+v", resolved, edge)
		}
	}
}

func TestDependencyPythonSeparatesLiteralFiniteAndUnboundedDynamicImports(t *testing.T) {
	result := observePythonFixture(t, map[string]string{
		"src/pkg/__init__.py": "",
		"src/pkg/alpha.py":    "",
		"src/pkg/beta.py":     "",
		"src/pkg/main.py": `import importlib as loader
from importlib import import_module as load
load("pkg.alpha")
__import__("pkg.b\x65ta", level=0)
for name in ["alpha", "beta"]:
    loader.import_module("pkg." + name)
for name in ["alpha"]:
    name = configured()
    loader.import_module("pkg." + name)
load(f"pkg.{runtime}")
__import__("pkg.beta", level=1)
__import__(*args)
`,
		"src/pkg/shadowed.py": `from importlib import import_module as load
load = replacement
load("pkg.alpha")
`,
		"src/pkg/invalid.py": "import pkg.alpha\ndef invalid(\n",
	}, "")
	if result.CoverageComplete || len(result.Issues) != 1 || result.Issues[0].Path != "src/pkg/invalid.py" {
		t.Fatalf("malformed Python file did not remain an incomplete observation: %+v", result)
	}
	found := map[string]int{}
	for _, edge := range result.Edges {
		if edge.EdgeType != "LOADS" {
			if edge.Source == "src/pkg/invalid.py" {
				t.Fatalf("partial syntax recovery emitted confident imports: %+v", edge)
			}
			continue
		}
		if edge.Note == nil {
			t.Fatalf("dynamic import lacks its evidence basis: %+v", edge)
		}
		kind := strings.SplitN(*edge.Note, ": ", 2)[0]
		found[kind]++
		switch kind {
		case "STATIC_LITERAL_DYNAMIC_IMPORT", "BOUNDED_DYNAMIC_IMPORT":
			if edge.Resolution != "RESOLVED" || edge.ResolvedPath == nil || !slices.Contains([]string{"pkg.alpha", "pkg.beta"}, edge.Target) {
				t.Fatalf("bounded import did not bind exact observed module: %+v", edge)
			}
		case "UNBOUNDED_DYNAMIC_IMPORT":
			if edge.Resolution != "DYNAMIC" || edge.ResolvedPath != nil || edge.Target != "<dynamic-import>" {
				t.Fatalf("unbounded import inferred a target: %+v", edge)
			}
		default:
			t.Fatalf("unrecognized dynamic import result: %+v", edge)
		}
	}
	if found["STATIC_LITERAL_DYNAMIC_IMPORT"] != 2 || found["BOUNDED_DYNAMIC_IMPORT"] != 2 || found["UNBOUNDED_DYNAMIC_IMPORT"] != 5 {
		t.Fatalf("unexpected observations per dynamic import basis: %v", found)
	}
}
