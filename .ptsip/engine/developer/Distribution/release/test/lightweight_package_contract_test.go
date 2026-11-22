package releasetest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

func TestLightweightSDistManifestTracksRegisteredProfiles(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "MANIFEST.in"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "include src/ptsip/schemas/config.json") ||
		!strings.Contains(string(b), "recursive-include src/ptsip/schemas/storage/profile *.schema.json") ||
		!strings.Contains(string(b), "recursive-include src/ptsip/schemas/history/profile *.schema.json") {
		t.Fatal("the canonical Profile catalog and schemas must be included in the sdist")
	}
	registrations, _, err := release.ReadProfileRegistrations(root)
	if err != nil {
		t.Fatal(err)
	}
	if registrations["ptsip.project-profile"].Lifecycle != "ACTIVE" {
		t.Fatal("current Profile has no active central registration")
	}
	schemas := filepath.Join(root, "src", "ptsip", "schemas")
	for _, registered := range registrations {
		body, err := os.ReadFile(filepath.Join(schemas, filepath.FromSlash(registered.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if release.Digest(bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))) != registered.Digest {
			t.Fatalf("Profile schema differs from its registered digest: %s", registered.Path)
		}
	}
}
