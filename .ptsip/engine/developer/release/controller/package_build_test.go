package releasecontroller

import (
	"strings"
	"testing"
)

func sourceBoundJoinFixture() (Selection, []PublishedComponent, map[string]string) {
	source := strings.Repeat("a", 40)
	versions := map[string]string{"context": "0.2.0", "projection": "0.2.0"}
	entries := []Component{
		{ID: "tool", Kind: "tool", Version: "T.0.3.8.b5"},
		{ID: "profiles", Kind: "profiles", Version: "pp.1.02"},
		{ID: "context", Kind: "engine", Version: versions["context"]},
		{ID: "projection", Kind: "engine", Version: versions["projection"]},
	}
	selection := Selection{
		Schema: "ptsip-release-selection/v2",
		Type: "PTSIP_PACKAGE", SourceSHA: source, PackageBase: "0.3.8",
		Components: entries, Platforms: []string{"linux-x64", "windows-x64"},
	}
	components := make([]PublishedComponent, 0, len(entries))
	for _, entry := range entries {
		components = append(components, PublishedComponent{
			ID: entry.ID, Kind: entry.Kind, Version: entry.Version,
			SourceSHA: source,
			Files: []File{{
				Name: entry.ID + ".artifact", SHA256: strings.Repeat("b", 64),
				Platform: "any", URL: "https://example.invalid/" + entry.ID,
			}},
		})
	}
	return selection, components, versions
}

func TestSourceBoundVerificationSnapshotAllowsChangedSourceForNonpublishingTests(t *testing.T) {
	s, components, registered := sourceBoundJoinFixture()
	snapshot, err := sourceBoundVerificationSnapshot(s, components, registered)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SourceSHA != s.SourceSHA || snapshot.Sequence != 1 ||
		len(snapshot.Components) != len(s.Components) ||
		!strings.HasPrefix(snapshot.Version, s.PackageBase+".") {
		t.Fatalf("test-only source-bound snapshot differs: %#v", snapshot)
	}
	for _, component := range snapshot.Components {
		if component.SourceSHA != s.SourceSHA {
			t.Fatalf("test artifact mixed with published historical source: %#v", component)
		}
	}
	// Published versions are preserved in test evidence. Production publication
	// still applies EngineReuse and ReusePublishedComponents independently.
	if snapshot.Components[2].Version != "0.2.0" {
		t.Fatal("verification silently changed the engine public version")
	}
}

func TestSourceBoundVerificationSnapshotRejectsForgedOrIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name string
		mutate func(*Selection, *[]PublishedComponent, map[string]string)
	}{
		{"stale-source", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			(*c)[2].SourceSHA = strings.Repeat("c", 40)
		}},
		{"wrong-version", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			(*c)[2].Version = "0.9.9"
		}},
		{"wrong-kind", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			(*c)[2].Kind = "profiles"
		}},
		{"missing-component", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			*c = (*c)[:len(*c)-1]
		}},
		{"duplicate-component", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			(*c)[3] = (*c)[2]
		}},
		{"missing-files", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			(*c)[2].Files = nil
		}},
		{"malformed-file-digest", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			(*c)[2].Files[0].SHA256 = "incorrect"
		}},
		{"unknown-engine", func(s *Selection, c *[]PublishedComponent, versions map[string]string) {
			delete(versions, "projection")
		}},
		{"not-package", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			s.Type = "TOOL_ONLY"
		}},
		{"wrong-base", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			s.PackageBase = "invalid"
		}},
		{"missing-selected-entry", func(s *Selection, c *[]PublishedComponent, _ map[string]string) {
			s.Components = s.Components[:len(s.Components)-1]
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, components, registered := sourceBoundJoinFixture()
			tc.mutate(&s, &components, registered)
			if _, err := sourceBoundVerificationSnapshot(s, components, registered); err == nil {
				t.Fatal("unverified Package source evidence passed")
			}
		})
	}
}
