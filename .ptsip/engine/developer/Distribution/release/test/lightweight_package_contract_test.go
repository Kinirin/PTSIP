package releasetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLightweightSDistManifestTracksCurrentProfile(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "MANIFEST.in"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "include schemas/ptsip-profile-pp-2.00.schema.json") {
		t.Fatal("current canonical pp.2.00 schema is not in the exact-source sdist manifest")
	}
	p, err := os.ReadFile(filepath.Join(root, "src/ptsip/specdata/ptsip-profile-pp-2.00.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	q, err := os.ReadFile(filepath.Join(root, ".ptsip/contracts/schemas/release/profile/ptsip-profile-pp-2.00.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != string(q) {
		t.Fatal("current canonical and installed schema projections diverge")
	}
}
