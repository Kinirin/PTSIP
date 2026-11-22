// Package inspection derives repository inventory from Discovery observations.
package inspection

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

type Inventory struct {
	ScanMode           string
	ObservedFiles      int
	Directories        int
	Manifests          []string
	RequirementsFiles  []string
	SchemaCandidates   []string
	TestRoots          []string
	ToolLikeRoots      []string
	SDKLikeRoots       []string
	TopLevelExtensions map[string]int
	ObservationErrors  []string
}

var manifestNames = []string{
	"pyproject.toml", "setup.py", "setup.cfg", "package.json", "Cargo.toml", "go.mod",
	"pom.xml", "build.gradle", "build.gradle.kts", "requirements.txt",
}

// ProjectInventory consumes the output of discovery.repository.inspect. It uses
// only the observed file paths and carries forward incomplete observations.
func ProjectInventory(output json.RawMessage) (Inventory, error) {
	var observed struct {
		Mode   string   `json:"file_mode"`
		Files  []string `json:"files"`
		Errors []string `json:"observation_errors"`
	}
	if err := json.Unmarshal(output, &observed); err != nil {
		return Inventory{}, fmt.Errorf("decode Discovery repository observation: %w", err)
	}
	if (observed.Mode != "git-tracked" && observed.Mode != "filesystem") || observed.Files == nil || observed.Errors == nil {
		return Inventory{}, fmt.Errorf("incomplete Discovery repository observation")
	}
	result := Inventory{
		ScanMode: observed.Mode, ObservedFiles: len(observed.Files), ObservationErrors: observed.Errors,
		TopLevelExtensions: make(map[string]int),
	}
	seen := make(map[string]bool, len(observed.Files))
	dirs := make(map[string]bool)
	extCounts := make(map[string]int)
	var extensions []string
	slices.Sort(observed.Files)
	for _, name := range observed.Files {
		if !fs.ValidPath(name) || name == "." || seen[name] {
			return Inventory{}, fmt.Errorf("invalid or repeated Discovery file path %q", name)
		}
		seen[name] = true
		for parent := path.Dir(name); parent != "." && !dirs[parent]; parent = path.Dir(parent) {
			dirs[parent] = true
		}
		base := path.Base(name)
		suffix := strings.ToLower(path.Ext(base))
		if suffix == "" || suffix == "." || suffix == strings.ToLower(base) {
			suffix = "<none>"
		}
		if extCounts[suffix] == 0 {
			extensions = append(extensions, suffix)
		}
		extCounts[suffix]++
		if slices.Contains(manifestNames, base) || slices.Contains([]string{".csproj", ".sln", ".slnx"}, suffix) {
			result.Manifests = append(result.Manifests, name)
		}
		if strings.HasPrefix(base, "requirements") && (suffix == ".txt" || suffix == ".in") {
			result.RequirementsFiles = append(result.RequirementsFiles, name)
		}
		parts := strings.Split(strings.ToLower(name), "/")
		stem := strings.TrimSuffix(strings.ToLower(base), suffix)
		if slices.Contains([]string{".json", ".yaml", ".yml", ".proto", ".avsc", ".xsd"}, suffix) &&
			(strings.Contains(stem, "schema") || slices.Contains(parts, "schemas") || slices.Contains(parts, "contracts")) {
			result.SchemaCandidates = append(result.SchemaCandidates, name)
		}
	}
	result.Directories = len(dirs)
	for dir := range dirs {
		if strings.Contains(dir, "/") {
			continue
		}
		name := strings.ToLower(dir)
		if slices.Contains([]string{"test", "tests", "testing"}, name) {
			result.TestRoots = append(result.TestRoots, dir)
		}
		if slices.Contains([]string{"tool", "tools", "devtools", "scripts", "build", "ci"}, name) {
			result.ToolLikeRoots = append(result.ToolLikeRoots, dir)
		}
		if strings.Contains(name, "sdk") || slices.Contains([]string{"src", "lib", "libs", "packages"}, name) {
			result.SDKLikeRoots = append(result.SDKLikeRoots, dir)
		}
	}
	slices.Sort(result.TestRoots)
	slices.Sort(result.ToolLikeRoots)
	slices.Sort(result.SDKLikeRoots)
	slices.SortStableFunc(extensions, func(a, b string) int { return extCounts[b] - extCounts[a] })
	for _, extension := range extensions[:min(20, len(extensions))] {
		result.TopLevelExtensions[extension] = extCounts[extension]
	}
	return result, nil
}
