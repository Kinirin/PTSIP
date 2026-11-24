package releasetest_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

func TestCanonicalCompositionHasOneToolValueAndNoLegacyFallback(t *testing.T) {
	root := repoRoot(t)
	c, err := release.ReadComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	source, err := release.ReadSourceVersions(root)
	if err != nil {
		t.Fatal(err)
	}
	if c.Tool != source.ToolVersion || c.Profile != source.ProfileVersion || len(c.Engines) != len(source.Engines) {
		t.Fatalf("canonical declaration diverges: %+v", c)
	}
	contract, err := release.ReadContract(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := release.VerifyCompositionProjections(root, c, contract); err != nil {
		t.Fatal(err)
	}
	wrongTool := c
	wrongTool.Tool = "T.99.0.0"
	if err := release.VerifyCompositionProjections(root, wrongTool, contract); err == nil || !strings.Contains(err.Error(), "Tool projection differs from src/VERSION") {
		t.Fatalf("Tool projection from a different source version was accepted: %v", err)
	}
	c.Engines["context"] = "99.0.0"
	if err := release.VerifyCompositionProjections(root, c, contract); err == nil {
		t.Fatal("stale distributed engine projection admitted")
	}
}

func TestCurrentCompositionPreservesOriginalPublishedState(t *testing.T) {
	root := repoRoot(t)
	contract, err := release.ReadContract(root)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(root, filepath.FromSlash(contract.Publication.Registry))
	before, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	var registry release.Registry
	if err := json.Unmarshal(before, &registry); err != nil {
		t.Fatal(err)
	}
	if registry.Latest == nil {
		t.Fatal("registered Package history has no latest snapshot")
	}
	snapshot, err := release.ResolvePackage(registry, *registry.Latest)
	if err != nil {
		t.Fatal(err)
	}
	c, err := release.ReadComposition(root)
	if err != nil || c.Published == nil || c.Published.Source != snapshot.SourceSHA ||
		c.Published.Identity != snapshot.Version || c.Published.Digest != snapshot.Digest ||
		c.Published.Tag != "ptsip-release-"+snapshot.SourceSHA {
		t.Fatalf("original source publication receipt differs: %+v (%v)", c.Published, err)
	}
	after, err := os.ReadFile(state)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("viewing composition altered original published evidence")
	}
}

func TestHistoricalCompositionIsExplicitReadOnly(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := release.ParseComposition(data); err != nil {
		t.Fatal(err)
	}
	duplicate := strings.Replace(string(data), "profile_version =", "tool_version = \"0.3.8a5\"\nprofile_version =", 1)
	if _, err := release.ParseComposition([]byte(duplicate)); err == nil {
		t.Fatal("historical parser admitted an ambiguous Tool declaration")
	}
	fixture := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixture, "pyproject.toml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	copyCompositionContract(t, fixture)
	if _, err := release.ReadComposition(fixture); err == nil || !strings.Contains(err.Error(), "src/VERSION") {
		t.Fatalf("missing src/VERSION used historical fallback: %v", err)
	}
}
