package releasetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
)

func TestCanonicalCompositionHasOneToolValueAndNoLegacyFallback(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := release.ParseComposition(data)
	if err != nil || c.Tool == "" || c.Profile != "pp.1.02" || len(c.Engines) != 7 {
		t.Fatalf("canonical declaration: %v", err)
	}
	noDeclaration := strings.Split(string(data), "[tool.ptsip.release]")[0]
	if _, err := release.ParseComposition([]byte(noDeclaration)); err == nil {
		t.Fatal("missing canonical declaration used legacy fallback")
	}
	duplicate := strings.Replace(string(data), "profile_version =", "tool_version = \"0.3.8a5\"\nprofile_version =", 1)
	if _, err := release.ParseComposition([]byte(duplicate)); err == nil {
		t.Fatal("duplicate Tool declaration authority admitted")
	}
	var contract release.Contract
	if err := release.ReadJSON(root, release.ContractPath, &contract); err != nil {
		t.Fatal(err)
	}
	c.Engines["context"] = "99.0.0"
	if err := release.VerifyCompositionProjections(root, c, contract); err == nil {
		t.Fatal("stale distributed engine projection admitted")
	}
}

func TestPublishedDeclarationDoesNotRewriteCurrentVersions(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := release.ParseComposition(data)
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixture, "pyproject.toml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	s := release.Snapshot{Version: "0.3.8.abcdef", Digest: "abcdef" + strings.Repeat("a", 58), SourceSHA: strings.Repeat("b", 40)}
	if err := release.RecordPublishedDeclaration(fixture, s); err != nil {
		t.Fatal(err)
	}
	c, err := release.ReadComposition(fixture)
	if err != nil || c.Tool != before.Tool || c.Engines["context"] != before.Engines["context"] || c.Published == nil || c.Published.Source != s.SourceSHA {
		t.Fatalf("publication record replaced current composition: %v", err)
	}
	if err := release.RecordPublishedDeclaration(fixture, s); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(fixture, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(after), "[tool.ptsip.release.published]") != 1 {
		t.Fatal("publication retry duplicated declaration")
	}
}
