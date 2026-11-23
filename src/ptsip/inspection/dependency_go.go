package inspection

import (
	"cmp"
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/modfile"
)

type observedGoModule struct {
	name      string
	directory string
	ambiguous bool
}

func scanGo(c dependencyContext) ([]DependencyEdge, []DependencyIssue, []string) {
	var modules []observedGoModule
	byName := map[string]int{}
	packages := map[string]string{}
	var edges []DependencyEdge
	var issues []DependencyIssue
	active := false
	for _, file := range c.paths {
		if strings.EqualFold(path.Ext(file), ".go") {
			active = true
			if dir := path.Dir(file); packages[dir] == "" && c.exists(file) {
				packages[dir] = file
			}
		}
		if path.Base(file) != "go.mod" {
			continue
		}
		active = true
		raw, err := c.read(file)
		if err != nil {
			issues = append(issues, DependencyIssue{"go-source", file, err.Error()})
			continue
		}
		if !utf8.Valid(raw) {
			issues = append(issues, DependencyIssue{"go-source", file, "go.mod must be UTF-8"})
			continue
		}
		name := modfile.ModulePath(raw)
		if name == "" {
			issues = append(issues, DependencyIssue{"go-source", file, "go.mod does not declare a module path"})
			continue
		}
		if index, exists := byName[name]; exists {
			modules[index].ambiguous = true
			issues = append(issues, DependencyIssue{"go-source", file, fmt.Sprintf("duplicate repository Go module %q", name)})
			continue
		}
		byName[name] = len(modules)
		modules = append(modules, observedGoModule{name: name, directory: path.Dir(file)})
	}
	if !active {
		return nil, nil, nil
	}
	slices.SortFunc(modules, func(a, b observedGoModule) int {
		return cmp.Or(cmp.Compare(len(b.name), len(a.name)), cmp.Compare(a.name, b.name))
	})
	for _, file := range c.paths {
		if !strings.EqualFold(path.Ext(file), ".go") {
			continue
		}
		raw, err := c.read(file)
		if err != nil {
			issues = append(issues, DependencyIssue{"go-source", file, err.Error()})
			continue
		}
		set := token.NewFileSet()
		parsed, err := parser.ParseFile(set, file, raw, parser.ImportsOnly|parser.SkipObjectResolution)
		if err != nil {
			issues = append(issues, DependencyIssue{"go-source", file, err.Error()})
			continue
		}
		phase := "RUNTIME"
		if strings.HasSuffix(file, "_test.go") {
			phase = "TEST"
		}
		for _, spec := range parsed.Imports {
			name, err := strconv.Unquote(spec.Path.Value)
			if err != nil || name == "" {
				issues = append(issues, DependencyIssue{"go-source", file, "invalid Go import path"})
				continue
			}
			line := set.PositionFor(spec.Pos(), false).Line
			edge := DependencyEdge{
				EvidenceID: fmt.Sprintf("go:%s:%d:%s", file, line, name), Source: file, Target: name,
				EdgeType: "IMPORTS", Phase: phase, Provenance: "OBSERVED", Line: &line, Adapter: "go-source",
			}
			resolveGoImport(&edge, modules, packages)
			edges = append(edges, edge)
		}
	}
	return edges, issues, []string{"go-source"}
}

func resolveGoImport(edge *DependencyEdge, modules []observedGoModule, packages map[string]string) {
	for _, module := range modules {
		if edge.Target != module.name && !strings.HasPrefix(edge.Target, module.name+"/") {
			continue
		}
		suffix := strings.TrimPrefix(strings.TrimPrefix(edge.Target, module.name), "/")
		if !module.ambiguous && (suffix == "" || (path.Clean(suffix) == suffix &&
			!strings.HasPrefix(suffix, "../") && suffix != "..")) {
			if representative := packages[path.Join(module.directory, suffix)]; representative != "" {
				note := "Import resolves within a repository Go module"
				edge.Resolution, edge.TargetScope, edge.ResolvedPath, edge.Note = "RESOLVED", "PROJECT_COMPONENT", &representative, &note
				return
			}
		}
		note := "Import matches a repository Go module without one observed package path"
		edge.Resolution, edge.TargetScope, edge.Note = "UNRESOLVED", "UNRESOLVED_TARGET", &note
		return
	}
	first, _, _ := strings.Cut(edge.Target, "/")
	if !strings.Contains(first, ".") {
		note := "Go standard-library import path"
		edge.Resolution, edge.TargetScope, edge.Note = "EXTERNAL", "PLATFORM", &note
		return
	}
	note := "Import is outside repository Go module paths"
	edge.Resolution, edge.TargetScope, edge.Note = "EXTERNAL", "EXTERNAL_DEPENDENCY", &note
}
