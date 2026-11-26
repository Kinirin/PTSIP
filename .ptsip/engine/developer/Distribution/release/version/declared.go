package version

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

// DeclaredVersionChange is a comparison of existing version declarations.
// It never proposes an increment or infers a semantic or published release.
type DeclaredVersionChange struct {
	Component string   `json:"component"`
	Previous  string   `json:"previous_version"`
	Version   string   `json:"version"`
	Files     []string `json:"source_files"`
}

func VersionsAtCommit(root, revision string) (map[string]string, error) {
	if !sha40.MatchString(revision) {
		return nil, fmt.Errorf("EXACT_VERSION_REVISION_REQUIRED")
	}
	raw, err := gitVersion(root, "show", revision+":"+versionsSource)
	if err != nil {
		return nil, err
	}
	values, currentErr := currentVersions(raw)
	if currentErr == nil {
		return values, nil
	}
	checkout, err := gitVersion(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	head := strings.TrimSpace(string(checkout))
	if revision == head {
		return nil, currentErr
	}
	if _, err := gitVersion(root, "merge-base", "--is-ancestor", revision, head); err != nil {
		return nil, fmt.Errorf("HISTORICAL_VERSION_ANCESTOR_REQUIRED: %w", err)
	}
	if original, err := versionsource.ParseHistoricalSectioned(raw); err == nil {
		return releaseNoteVersions(original), nil
	}
	return historicalVersions(raw)
}

func currentVersions(raw []byte) (map[string]string, error) {
	versions, err := versionsource.Parse(raw)
	if err != nil {
		return nil, err
	}
	return releaseNoteVersions(versions), nil
}

func releaseNoteVersions(versions versionsource.Versions) map[string]string {
	// These are the component routes registered for the current release notes.
	values := map[string]string{"package": versions.PackageVersion, "tool": versions.ToolVersion, "profile": versions.ProfileVersion}
	for id, value := range versions.Engines {
		values["engine."+id] = value
	}
	return values
}

func DeclaredChanges(root, base, head string) (VersionDiffReport, []DeclaredVersionChange, error) {
	report, err := AnalyzeVersionsDifferent(root, base, head)
	if err != nil {
		return report, nil, err
	}
	previous, err := VersionsAtCommit(root, base)
	if err != nil {
		return report, nil, err
	}
	current, err := VersionsAtCommit(root, head)
	if err != nil {
		return report, nil, err
	}
	files := map[string][]string{}
	for _, entry := range report.Entries {
		files[entry.Component] = entry.Paths
	}
	components := make([]string, 0, len(current))
	for component := range current {
		components = append(components, component)
	}
	sort.Strings(components)
	changes := make([]DeclaredVersionChange, 0)
	for _, component := range components {
		prior, exists := previous[component]
		if !exists || current[component] == prior {
			continue
		}
		paths := append([]string{versionsSource}, files[component]...)
		changes = append(changes, DeclaredVersionChange{Component: component, Previous: prior, Version: current[component], Files: paths})
	}
	return report, changes, nil
}
