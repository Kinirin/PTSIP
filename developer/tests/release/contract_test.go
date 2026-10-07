package release_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

var releaseContractPairs = [][2]string{
	{"schemas/ptsip-profile.schema.json", "src/ptsip/specdata/ptsip-profile.schema.json"},
	{"schemas/ptsip-profile-pp-1.01.schema.json", "src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json"},
	{"schemas/ptsip-artifact-evidence.schema.json", "src/ptsip/specdata/ptsip-artifact-evidence.schema.json"},
	{"schemas/ptsip-agent-classification.schema.json", "src/ptsip/specdata/ptsip-agent-classification.schema.json"},
	{"schemas/ptsip-diagnostic.schema.json", "src/ptsip/specdata/ptsip-diagnostic.schema.json"},
	{"schemas/ptsip-normalized-evidence.schema.json", "src/ptsip/specdata/ptsip-normalized-evidence.schema.json"},
	{"registry/ptsip-registry.yaml", "src/ptsip/specdata/ptsip-registry.yaml"},
}

func TestReleaseContractCanonicalAndEmbeddedAssetsAreIdentical(t *testing.T) {
	root := testrepo.Root(t)
	for _, pair := range releaseContractPairs {
		canonical, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pair[0])))
		if err != nil {
			t.Fatal(err)
		}
		embedded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pair[1])))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(canonical, embedded) {
			t.Fatalf("release contract projection drift: %s != %s", pair[0], pair[1])
		}
	}
}

func TestReleaseContractAssetsRemainBoundToSpecificationRevision(t *testing.T) {
	root := testrepo.Root(t)
	constants, err := os.ReadFile(filepath.Join(root, "src", "ptsip", "constants.py"))
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile("(?m)^SPEC_REVISION = \"([0-9a-f]{40})\"$").FindSubmatch(constants)
	if len(match) != 2 {
		t.Fatal("SPEC_REVISION is not an exact 40-character lowercase SHA")
	}
	revision := string(match[1])

	for _, pair := range releaseContractPairs {
		for _, path := range pair {
			headObject := releaseGit(t, root, "rev-parse", "--verify", "HEAD:"+path)
			boundObject := releaseGit(t, root, "rev-parse", "--verify", revision+":"+path)
			if headObject != boundObject {
				t.Fatalf("release-bound asset drifted from %s: %s", revision, path)
			}
		}
	}
}
