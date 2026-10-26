package releasecontroller

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sourcePackageVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))(?:-((?:a|b|rc)[1-9][0-9]*))?$`)
var tomlSectionPattern = regexp.MustCompile(`^[ \t]*\[([^\[\]\r\n]+)\][ \t]*(?:#.*)?$`)

// PackageVersionSyncResult reports a projection operation without making
// pyproject.toml an independent version authority.
type PackageVersionSyncResult struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Version  string `json:"version"`
	Previous string `json:"previous_version"`
	Changed  bool   `json:"changed"`
	Status   string `json:"status"`
}

func ReadSourcePackageVersion(root string) (string, string, string, error) {
	data, err := os.ReadFile(filepath.Join(root, "src", "VERSION"))
	if err != nil {
		return "", "", "", fmt.Errorf("read src/VERSION: %w", err)
	}
	version := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if version != string(data) && string(data) != version+"\n" && string(data) != version+"\r\n" {
		return "", "", "", fmt.Errorf("src/VERSION must contain exactly one version line")
	}
	parts := sourcePackageVersionPattern.FindStringSubmatch(version)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid src/VERSION %q: expected MAJOR.MINOR.PATCH[-aN|-bN|-rcN]", version)
	}
	return version, parts[1], parts[2], nil
}

func tomlProjectionLine(line, key, replacement string) (string, string, bool, error) {
	original := line
	newline := ""
	if strings.HasSuffix(line, "\r\n") {
		newline = "\r\n"
		line = strings.TrimSuffix(line, "\r\n")
	} else if strings.HasSuffix(line, "\n") {
		newline = "\n"
		line = strings.TrimSuffix(line, "\n")
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

// ProjectPackageVersion updates only four existing TOML declarations, preserving
// all unrelated composition, Tool, Engine and published snapshot fields.
func ProjectPackageVersion(data []byte, version, base, state string) ([]byte, string, error) {
	expected := map[string]map[string]string{
		"project": {
			"version": fmt.Sprintf("%q", version),
		},
		"tool.ptsip.release": {
			"package_base":     fmt.Sprintf("%q", base),
			"package_revision": "0",
			"package_state":    fmt.Sprintf("%q", state),
		},
	}
	seenSections := map[string]bool{}
	seenKeys := map[string]bool{}
	var previous string
	var out strings.Builder
	section := ""
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
	return []byte(out.String()), previous, nil
}

// SyncPackageVersion provides explicit mutation and read-only check modes.
// The source is always src/VERSION; pyproject.toml never flows back to it.
func SyncPackageVersion(root string, checkOnly bool) (PackageVersionSyncResult, error) {
	version, base, state, err := ReadSourcePackageVersion(root)
	if err != nil {
		return PackageVersionSyncResult{}, err
	}
	result := PackageVersionSyncResult{
		Source: "src/VERSION", Target: "pyproject.toml", Version: version,
	}
	path := filepath.Join(root, "pyproject.toml")
	before, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	after, previous, err := ProjectPackageVersion(before, version, base, state)
	if err != nil {
		return result, err
	}
	result.Previous = previous
	result.Changed = !bytes.Equal(before, after)
	if !result.Changed {
		result.Status = "IN_SYNC"
		return result, nil
	}
	if checkOnly {
		return result, fmt.Errorf("pyproject.toml Package version differs from src/VERSION (%s): run sync-version", version)
	}
	stat, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ptsip-version-*.toml")
	if err != nil {
		return result, err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(stat.Mode().Perm()); err == nil {
		_, err = tmp.Write(after)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return result, err
	}
	latest, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	if !bytes.Equal(latest, before) {
		return result, fmt.Errorf("pyproject.toml changed during version projection; retry")
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return result, err
	}
	result.Status = "SYNCED"
	return result, nil
}
