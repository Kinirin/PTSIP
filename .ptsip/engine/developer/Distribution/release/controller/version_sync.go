package releasecontroller

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
	"github.com/pelletier/go-toml/v2"
)

var tomlSectionPattern = regexp.MustCompile(`^[ \t]*\[([^\[\]\r\n]+)\][ \t]*(?:#.*)?$`)
var sourceProfileVersionPattern = regexp.MustCompile(`^pp\.[0-9]+\.[0-9]+$`)

var officialEngineIDs = versionsource.EngineIDs()

type VersionSource = versionsource.Versions

type PackageVersionSyncResult struct {
	Source           string             `json:"source"`
	Target           string             `json:"target"`
	Version          string             `json:"version"`
	Previous         string             `json:"previous_version"`
	ToolVersion      string             `json:"tool_version"`
	ProfileVersion   string             `json:"profile_version"`
	Engines          map[string]string  `json:"engines"`
	ChangedFiles     []string           `json:"changed_files"`
	CheckedFiles     []string           `json:"checked_files"`
	Changed          bool               `json:"changed"`
	Status           string             `json:"status"`
	ChangedManifests []string           `json:"changed_manifests"`
	ChangedBindings  []string           `json:"changed_bindings"`
	Notice           *VersionSyncNotice `json:"notice,omitempty"`
}

type VersionSyncNotice struct {
	ChangedManifestCount int      `json:"changed_manifest_count"`
	ChangedBindingCount  int      `json:"changed_binding_count"`
	AffectedEngineIDs    []string `json:"affected_engine_ids"`
	PayloadKind          string   `json:"payload_kind"`
}

func replaceProjection(path string, data []byte, mode os.FileMode) error {
	next, err := os.CreateTemp(filepath.Dir(path), ".version-projection-*")
	if err != nil {
		return err
	}
	defer os.Remove(next.Name())
	_, writeErr := next.Write(data)
	if err := errors.Join(writeErr, next.Chmod(mode), next.Close()); err != nil {
		return err
	}
	return os.Rename(next.Name(), path)
}

func applyVersionProjections(root string, projections []versionProjection) error {
	var written []versionProjection
	rollback := func(err error) error {
		for i := len(written) - 1; i >= 0; i-- {
			entry := written[i]
			current, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.path)))
			if readErr != nil || !bytes.Equal(current, entry.after) {
				err = errors.Join(err, fmt.Errorf("rollback would overwrite a concurrent edit to %s", entry.path))
				continue
			}
			err = errors.Join(err, replaceProjection(filepath.Join(root, filepath.FromSlash(entry.path)), entry.before, entry.mode))
		}
		return err
	}
	for _, entry := range projections {
		if bytes.Equal(entry.before, entry.after) {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(entry.path))
		current, err := os.ReadFile(path)
		if err != nil {
			return rollback(err)
		}
		if !bytes.Equal(current, entry.before) {
			return rollback(fmt.Errorf("projection changed concurrently: %s", entry.path))
		}
		if err := replaceProjection(path, entry.after, entry.mode); err != nil {
			return rollback(fmt.Errorf("replace %s: %w", entry.path, err))
		}
		written = append(written, entry)
	}
	return nil
}

func ReadSourceVersions(root string) (VersionSource, error) {
	return versionsource.Read(root)
}

// Kept as a read-only compatibility helper. All canonical values now live in
// the same parsed src/VERSION document.
func ReadSourcePackageVersion(root string) (string, string, string, error) {
	version, err := ReadSourceVersions(root)
	if err != nil {
		return "", "", "", err
	}
	base, state, _ := versionsource.PackageParts(version.PackageVersion)
	return version.PackageVersion, base, state, nil
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
	base, state, valid := versionsource.PackageParts(source.PackageVersion)
	if !valid {
		return nil, "", fmt.Errorf("invalid canonical Package version")
	}
	expected := map[string]map[string]string{
		"project": {"version": fmt.Sprintf("%q", source.DistributionVersion())},
		"tool.ptsip.release": {
			"package_base":     fmt.Sprintf("%q", base),
			"package_revision": "0",
			"package_state":    fmt.Sprintf("%q", state),
			"tool_version":     fmt.Sprintf("%q", source.ToolVersion),
			"profile_version":  fmt.Sprintf("%q", source.ProfileVersion),
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
					identity := section + "." + key
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
	path   string
	before []byte
	after  []byte
	mode   os.FileMode
}

func SyncPackageVersion(root string, checkOnly bool) (PackageVersionSyncResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return PackageVersionSyncResult{}, err
	}
	rawSource, _, err := readProjection(root, versionsource.Path)
	if err != nil {
		return PackageVersionSyncResult{}, err
	}
	source, err := versionsource.Parse(rawSource)
	if err != nil {
		return PackageVersionSyncResult{}, err
	}
	result := PackageVersionSyncResult{
		Source: "src/VERSION", Target: "pyproject.toml",
		Version: source.PackageVersion, ToolVersion: source.ToolVersion,
		ProfileVersion: source.ProfileVersion, Engines: source.Engines,
		CheckedFiles: []string{versionsource.Path}, ChangedFiles: []string{},
		ChangedManifests: []string{}, ChangedBindings: []string{},
	}
	projections := []versionProjection{}
	stage := func(relative string, transform func([]byte) ([]byte, error)) error {
		before, mode, err := readProjection(root, relative)
		if err != nil {
			return fmt.Errorf("read %s: %w", relative, err)
		}
		after, err := transform(before)
		if err != nil {
			return fmt.Errorf("project %s: %w", relative, err)
		}
		projections = append(projections, versionProjection{path: relative, before: before, after: after, mode: mode})
		result.CheckedFiles = append(result.CheckedFiles, relative)
		if !bytes.Equal(before, after) {
			result.ChangedFiles = append(result.ChangedFiles, relative)
		}
		return nil
	}
	if err := stage("pyproject.toml", func(before []byte) ([]byte, error) {
		after, previous, err := ProjectVersionComposition(before, source)
		result.Previous = previous
		return after, err
	}); err != nil {
		return result, err
	}
	const registryPath = "src/ptsip/common_contract/registry/manifest.json"
	config, _, err := readProjection(root, ProfileConfigPath)
	if err != nil {
		return result, err
	}
	registry, _, err := readProjection(root, registryPath)
	if err != nil {
		return result, err
	}
	if err := source.VerifyOwners(config, registry); err != nil {
		return result, err
	}
	result.CheckedFiles = append(result.CheckedFiles, ProfileConfigPath, registryPath)
	var entries []syncEngineEntry
	if err := stage(currentEngineIndex, func(data []byte) ([]byte, error) {
		_, declared, err := engineEntries(data, source)
		entries = declared
		if err != nil {
			return nil, err
		}
		return projectIndexedEngines(data, source)
	}); err != nil {
		return result, err
	}
	for _, entry := range entries {
		id := strings.ToLower(entry.Kind)
		if err := stageEngine(root, stage, entry, source.Engines[id]); err != nil {
			return result, fmt.Errorf("project %s: %w", id, err)
		}
		result.CheckedFiles = append(result.CheckedFiles, entry.Contract)
	}
	for _, changed := range result.ChangedFiles {
		if strings.HasSuffix(changed, "/manifest.json") {
			result.ChangedManifests = append(result.ChangedManifests, changed)
		}
		if strings.HasSuffix(changed, "/execution-bindings.json") {
			result.ChangedBindings = append(result.ChangedBindings, changed)
		}
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
		if bytes.Equal(entry.before, entry.after) {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(entry.path))
		current, err := os.ReadFile(p)
		if err != nil {
			return result, err
		}
		if !bytes.Equal(current, entry.before) {
			return result, fmt.Errorf("projection changed concurrently: %s", entry.path)
		}
	}
	if err := applyVersionProjections(root, projections); err != nil {
		return result, err
	}
	result.Status = "SYNCED"
	if len(result.ChangedManifests)+len(result.ChangedBindings) != 0 {
		notice := &VersionSyncNotice{
			ChangedManifestCount: len(result.ChangedManifests),
			ChangedBindingCount:  len(result.ChangedBindings),
			PayloadKind:          "SOURCE_VERSION_IDENTITY_MIGRATION_NOTIFICATION",
		}
		for _, entry := range entries {
			if slices.Contains(result.ChangedManifests, entry.Manifest) ||
				slices.Contains(result.ChangedBindings, entry.Bindings) {
				notice.AffectedEngineIDs = append(notice.AffectedEngineIDs, strings.ToLower(entry.Kind))
			}
		}
		result.Notice = notice
	}
	return result, nil
}

// Deterministic readable target order for external tests and diagnostics.
func CanonicalEngineIDs() []string {
	ids := append([]string(nil), officialEngineIDs...)
	sort.Strings(ids)
	return ids
}
