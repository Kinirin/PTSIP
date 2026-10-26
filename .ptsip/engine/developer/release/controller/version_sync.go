package releasecontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var sourcePackageVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))(?:-((?:a|b|rc)[1-9][0-9]*))?$`)
var sourceEngineVersionPattern = regexp.MustCompile(`^(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)
var sourceToolVersionPattern = regexp.MustCompile(`^T\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:\.(?:a|b|rc)[1-9][0-9]*)?$`)
var sourceProfileVersionPattern = regexp.MustCompile(`^pp\.[0-9]+\.[0-9]+$`)
var tomlSectionPattern = regexp.MustCompile(`^[ \t]*\[([^\[\]\r\n]+)\][ \t]*(?:#.*)?$`)

var officialEngineIDs = []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"}

type VersionSource struct {
	PackageVersion string            `toml:"package_version" json:"package_version"`
	ToolVersion    string            `toml:"tool_version" json:"tool_version"`
	ProfileVersion string            `toml:"profile_version" json:"profile_version"`
	Engines        map[string]string `toml:"engines" json:"engines"`
}

type PackageVersionSyncResult struct {
	Source         string            `json:"source"`
	Target         string            `json:"target"`
	Version        string            `json:"version"`
	Previous       string            `json:"previous_version"`
	ToolVersion    string            `json:"tool_version"`
	ProfileVersion string            `json:"profile_version"`
	Engines        map[string]string `json:"engines"`
	ChangedFiles   []string          `json:"changed_files"`
	CheckedFiles   []string          `json:"checked_files"`
	Changed        bool              `json:"changed"`
	Status         string            `json:"status"`
}

func ReadSourceVersions(root string) (VersionSource, error) {
	var version VersionSource
	data, err := os.ReadFile(filepath.Join(root, "src", "VERSION"))
	if err != nil {
		return version, fmt.Errorf("read src/VERSION: %w", err)
	}
	if err := toml.Unmarshal(data, &version); err != nil {
		return version, fmt.Errorf("invalid src/VERSION TOML: %w", err)
	}
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return version, err
	}
	for key := range raw {
		if key != "package_version" && key != "tool_version" && key != "profile_version" && key != "engines" {
			return version, fmt.Errorf("unregistered src/VERSION field: %s", key)
		}
	}
	if len(raw) != 4 {
		return version, fmt.Errorf("src/VERSION requires Package, Tool, Profile and Engines")
	}
	if !sourcePackageVersionPattern.MatchString(version.PackageVersion) {
		return version, fmt.Errorf("invalid src/VERSION package_version: %q", version.PackageVersion)
	}
	if !sourceToolVersionPattern.MatchString(version.ToolVersion) || !sourceProfileVersionPattern.MatchString(version.ProfileVersion) {
		return version, fmt.Errorf("invalid src/VERSION Tool or Profile version")
	}
	if len(version.Engines) != len(officialEngineIDs) {
		return version, fmt.Errorf("src/VERSION must declare all seven official Engines exactly once")
	}
	for _, id := range officialEngineIDs {
		if !sourceEngineVersionPattern.MatchString(version.Engines[id]) {
			return version, fmt.Errorf("invalid or missing src/VERSION Engine version: %s", id)
		}
	}
	return version, nil
}

// Kept as a read-only compatibility helper. All canonical values now live in
// the same parsed src/VERSION document.
func ReadSourcePackageVersion(root string) (string, string, string, error) {
	version, err := ReadSourceVersions(root)
	if err != nil {
		return "", "", "", err
	}
	parts := sourcePackageVersionPattern.FindStringSubmatch(version.PackageVersion)
	return version.PackageVersion, parts[1], parts[2], nil
}

func tomlProjectionLine(original, key, replacement string) (string, string, bool, error) {
	line, newline := original, ""
	if strings.HasSuffix(line, "\r\n") {
		newline, line = "\r\n", strings.TrimSuffix(line, "\r\n")
	} else if strings.HasSuffix(line, "\n") {
		newline, line = "\n", strings.TrimSuffix(line, "\n")
	}
	start := regexp.MustCompile(`^[ \t]*` + regexp.QuoteMeta(key) + `[ \t]*=`)
	if !start.MatchString(line) {
		return original, "", false, nil
	}
	pattern := regexp.MustCompile(`^([ \t]*` + regexp.QuoteMeta(key) + `[ \t]*=[ \t]*)("[^"\r\n]*"|[0-9]+)([ \t]*(?:#.*)?)$`)
	parts := pattern.FindStringSubmatch(line)
	if len(parts) != 4 {
		return "", "", false, fmt.Errorf("unsupported TOML declaration for %s", key)
	}
	return parts[1] + replacement + parts[3] + newline, parts[2], true, nil
}

func ProjectVersionComposition(data []byte, source VersionSource) ([]byte, string, error) {
	match := sourcePackageVersionPattern.FindStringSubmatch(source.PackageVersion)
	if len(match) != 3 {
		return nil, "", fmt.Errorf("invalid canonical Package version")
	}
	expected := map[string]map[string]string{
		"project": {"version": fmt.Sprintf("%q", source.PackageVersion)},
		"tool.ptsip.release": {
			"package_base": fmt.Sprintf("%q", match[1]),
			"package_revision": "0",
			"package_state": fmt.Sprintf("%q", match[2]),
			"tool_version": fmt.Sprintf("%q", source.ToolVersion),
			"profile_version": fmt.Sprintf("%q", source.ProfileVersion),
		},
		"tool.ptsip.release.engines": {},
	}
	for id, version := range source.Engines {
		expected["tool.ptsip.release.engines"][id] = fmt.Sprintf("%q", version)
	}
	seenSections, seenKeys := map[string]bool{}, map[string]bool{}
	previous := ""
	section := ""
	var out strings.Builder
	for _, original := range strings.SplitAfter(string(data), "\n") {
		line := strings.TrimRight(original, "\r\n")
		if matches := tomlSectionPattern.FindStringSubmatch(line); len(matches) == 2 {
			section = strings.TrimSpace(matches[1])
			if expected[section] != nil {
				if seenSections[section] {
					return nil, "", fmt.Errorf("duplicate TOML table [%s]", section)
				}
				seenSections[section] = true
			}
		}
		updated := original
		if declarations := expected[section]; declarations != nil {
			for key, replacement := range declarations {
				var found bool
				var old string
				var err error
				updated, old, found, err = tomlProjectionLine(updated, key, replacement)
				if err != nil {
					return nil, "", err
				}
				if found {
					identity := section+"."+key
					if seenKeys[identity] {
						return nil, "", fmt.Errorf("duplicate TOML declaration: %s", identity)
					}
					seenKeys[identity] = true
					if identity == "project.version" {
						previous = strings.Trim(old, "\"")
					}
				}
			}
		}
		out.WriteString(updated)
	}
	for section, declarations := range expected {
		if !seenSections[section] {
			return nil, "", fmt.Errorf("missing TOML table [%s]", section)
		}
		for key := range declarations {
			if !seenKeys[section+"."+key] {
				return nil, "", fmt.Errorf("missing TOML declaration %s.%s", section, key)
			}
		}
	}
	// Require a valid TOML file after targeted field projection.
	var parsed map[string]any
	if err := toml.Unmarshal([]byte(out.String()), &parsed); err != nil {
		return nil, "", fmt.Errorf("invalid resulting pyproject TOML: %w", err)
	}
	return []byte(out.String()), previous, nil
}

func projectEngineIndex(data []byte, source VersionSource) ([]byte, error) {
	var doc struct {
		Entries []struct {
			Kind string `json:"kind"`
			Version string `json:"version"`
		} `json:"official_engine_entries"`
	}
	if err := json.Unmarshal(data, &doc); err != nil { return nil, err }
	if len(doc.Entries) != len(source.Engines) {
		return nil, fmt.Errorf("Engine index and src/VERSION inventory differ")
	}
	seen := map[string]bool{}
	changed := false
	for _, entry := range doc.Entries {
		id := strings.ToLower(entry.Kind)
		if seen[id] || source.Engines[id] == "" {
			return nil, fmt.Errorf("unknown or duplicate official Engine: %s", id)
		}
		seen[id] = true
		if entry.Version != source.Engines[id] { changed = true }
	}
	if !changed { return data, nil }
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil { return nil, err }
	entries, ok := parsed["official_engine_entries"].([]any)
	if !ok { return nil, fmt.Errorf("missing official Engine index entries") }
	for _, item := range entries {
		entry := item.(map[string]any)
		id := strings.ToLower(entry["kind"].(string))
		entry["version"] = source.Engines[id]
	}
	updated, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil { return nil, err }
	return append(updated, '\n'), nil
}

func projectSingleLine(data []byte, field, version string) ([]byte, error) {
	// Only an exact top-level scalar declaration can be projected; history
	// entries and other version semantics are intentionally untouched.
	re := regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(field) + `:[ \t]*)([^\r\n#]*?)([ \t]*(?:#[^\r\n]*)?)(\r?)$`)
	matches := re.FindAllSubmatchIndex(data, -1)
	if len(matches) != 1 {
		return nil, fmt.Errorf("missing or ambiguous %s YAML projection", field)
	}
	return re.ReplaceAll(data, []byte("${1}"+version+"${3}${4}")), nil
}

type versionProjection struct {
	path string
	before []byte
	after []byte
	mode os.FileMode
}

func SyncPackageVersion(root string, checkOnly bool) (PackageVersionSyncResult, error) {
	source, err := ReadSourceVersions(root)
	if err != nil { return PackageVersionSyncResult{}, err }
	result := PackageVersionSyncResult{
		Source: "src/VERSION", Target: "pyproject.toml",
		Version: source.PackageVersion, ToolVersion: source.ToolVersion,
		ProfileVersion: source.ProfileVersion, Engines: source.Engines,
		CheckedFiles: []string{}, ChangedFiles: []string{},
	}
	projections := []versionProjection{}
	stage := func(relative string, transform func([]byte) ([]byte, error)) error {
		if filepath.IsAbs(relative) || strings.Contains(relative, "..") { return fmt.Errorf("invalid projection path") }
		path := filepath.Join(root, filepath.FromSlash(relative))
		before, err := os.ReadFile(path)
		if err != nil { return fmt.Errorf("read %s: %w", relative, err) }
		stat, err := os.Stat(path)
		if err != nil { return err }
		after, err := transform(before)
		if err != nil { return fmt.Errorf("project %s: %w", relative, err) }
		projections = append(projections, versionProjection{path: relative, before: before, after: after, mode: stat.Mode().Perm()})
		result.CheckedFiles = append(result.CheckedFiles, relative)
		if !bytes.Equal(before, after) { result.ChangedFiles = append(result.ChangedFiles, relative) }
		return nil
	}
	if err := stage("pyproject.toml", func(before []byte) ([]byte, error) {
		after, previous, err := ProjectVersionComposition(before, source)
		result.Previous = previous
		return after, err
	}); err != nil { return result, err }
	if err := stage("src/ptsip/engines/index.json", func(b []byte) ([]byte, error) { return projectEngineIndex(b, source) }); err != nil { return result, err }
	if err := stage("src/ptsip/specdata/project-profile-contracts.yaml", func(b []byte) ([]byte, error) {
		return projectSingleLine(b, "current", source.ProfileVersion)
	}); err != nil { return result, err }
	for _, id := range officialEngineIDs {
		id := id
		err := stage("src/ptsip/engines/"+id+"/version.yaml", func(b []byte) ([]byte, error) {
			if !bytes.Contains(b, []byte("engine: "+id+"\n")) { return nil, fmt.Errorf("Engine identity mismatch for %s", id) }
			return projectSingleLine(b, "version", source.Engines[id])
		})
		if err != nil { return result, err }
	}
	result.Changed = len(result.ChangedFiles) != 0
	if !result.Changed {
		result.Status = "IN_SYNC"
		return result, nil
	}
	if checkOnly {
		return result, fmt.Errorf("version projections differ from src/VERSION: %s; run sync-version", strings.Join(result.ChangedFiles, ", "))
	}
	// Check every target first to avoid any mutations when a source or
	// projection is invalid; only approved version-bearing fields are edited.
	for _, entry := range projections {
		if bytes.Equal(entry.before, entry.after) { continue }
		p := filepath.Join(root, filepath.FromSlash(entry.path))
		current, err := os.ReadFile(p)
		if err != nil { return result, err }
		if !bytes.Equal(current, entry.before) { return result, fmt.Errorf("projection changed concurrently: %s", entry.path) }
		if err := os.WriteFile(p, entry.after, entry.mode); err != nil { return result, err }
	}
	result.Status = "SYNCED"
	return result, nil
}

// Deterministic readable target order for external tests and diagnostics.
func CanonicalEngineIDs() []string {
	ids := append([]string(nil), officialEngineIDs...)
	sort.Strings(ids)
	return ids
}
