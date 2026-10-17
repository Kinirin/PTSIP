package releasetest

import (
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

// Exercise the real historical producer delta that the hosted Package join
// consumes, including the frozen container runner added before this migration.
func TestSingleProjectMigrationAdmitsHistoricalToolDeltaOnlyForPackage(t *testing.T) {
	root := repoRoot(t)
	head, err := release.Run(root, "git", "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.TrimSpace(string(head))
	old := release.PublishedComponent{ID: "tool", Kind: "tool", Version: "T.0.3.8.b3", SourceSHA: "916333f9aa0f9589272911818d8352fb0cae09e1"}
	current := old
	current.SourceSHA = source
	registry := release.Registry{Snapshots: []release.Snapshot{{Components: []release.PublishedComponent{old}}}}
	s := release.Selection{Type: "PTSIP_PACKAGE", SourceSHA: source}
	components, err := release.ReusePublishedComponents(root, s, registry, []release.PublishedComponent{current}, map[string]string{})
	if err != nil || len(components) != 1 || components[0].SourceSHA != source {
		t.Fatalf("first single-project Package must retain the rebuilt Tool: %v", err)
	}
	s.Type = "TOOL_ONLY"
	if _, err := release.ReusePublishedComponents(root, s, registry, []release.PublishedComponent{current}, map[string]string{}); err == nil {
		t.Fatal("Package migration exception admitted a standalone Tool replacement")
	}
	s.Type = "PTSIP_PACKAGE"
	registry.Snapshots[0].Components[0].SourceSHA = "04f2170d7e240efde8df7179df690de0ee9a10a1"
	if _, err := release.ReusePublishedComponents(root, s, registry, []release.PublishedComponent{current}, map[string]string{}); err == nil {
		t.Fatal("Package migration exception admitted a different prior Tool source")
	}
}
