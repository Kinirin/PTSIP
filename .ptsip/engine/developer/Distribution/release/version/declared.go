package version

import (
	"fmt"
	"sort"
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
	values, _, err := sourceVersions(raw)
	return values, err
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
		if current[component] == previous[component] {
			continue
		}
		paths := append([]string{"src/VERSION"}, files[component]...)
		changes = append(changes, DeclaredVersionChange{Component: component, Previous: previous[component], Version: current[component], Files: paths})
	}
	return report, changes, nil
}
