package releasetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func TestPackageRequestExpandsAllAndCannotDowngradeForCredentials(t *testing.T) {
	root := repoRoot(t)
	s, err := release.ReleaseRequest(root, "PTSIP_PACKAGE", "", "")
	if err != nil {
		t.Fatal(err)
	}
	c, err := release.ReadComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Components) != len(c.Engines)+2 || s.Type != "PTSIP_PACKAGE" {
		t.Fatal("Package request omitted canonical inventory")
	}
	if err := release.AdmitPublisherScope(root, s, "github-pypi"); err == nil {
		t.Fatal("Package component vector claimed an unrequested standalone Tool PyPI project")
	}
	if s.Type != "PTSIP_PACKAGE" || len(s.Components) != len(c.Engines)+2 {
		t.Fatal("credential failure changed the request")
	}
	if _, err := release.ReleaseRequest(root, "PTSIP_PACKAGE", "context", ""); err == nil {
		t.Fatal("Package accepted an engine subset")
	}
	for kind, names := range map[string]string{"TOOL_ONLY": "", "ENGINE_ONLY": "context", "PROFILES_ONLY": "", "TOOL_ENGINES": "context,projection", "TOOL_PROFILES": "", "ENGINES_PROFILES": "context", "MULTIPLE_ENGINES": "context,projection"} {
		if selected, err := release.ReleaseRequest(root, kind, names, ""); err != nil || selected.Type != kind {
			t.Fatalf("explicit component release changed: %s %v", kind, err)
		}
	}
	if _, err := release.ReleaseRequest(root, "ENGINE_ONLY", "context,projection", ""); err == nil {
		t.Fatal("single engine request admitted multiple engines")
	}
}

func TestUnchangedToolReuseIgnoresPublicationMetadataButRejectsRuntimeChange(t *testing.T) {
	root := t.TempDir()
	write := func(ref, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(ref))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	contract, err := os.ReadFile(filepath.Join(repoRoot(t), release.ContractPath))
	if err != nil {
		t.Fatal(err)
	}
	write(release.ContractPath, string(contract))
	project := "[project]\nversion=\"0.3.8b1\"\n[tool.ptsip.release]\npackage_base=\"0.3.8\"\n"
	write("pyproject.toml", project)
	write("src/runtime.go", "original")
	write(".github/workflows/tooling-release.yml", "old transport")
	run := func(args ...string) string {
		t.Helper()
		b, err := release.Run(root, args...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(b))
	}
	run("git", "init", "--quiet")
	run("git", "config", "user.name", "fixture")
	run("git", "config", "user.email", "fixture@example.invalid")
	commit := func() string {
		run("git", "add", ".")
		run("git", "commit", "--quiet", "-m", "fixture")
		return run("git", "rev-parse", "HEAD")
	}
	origin := commit()
	write(".github/workflows/tooling-release.yml", "new transport")
	write("pyproject.toml", project+"[tool.ptsip.release.published]\nidentity=\"0.3.8.abcdef\"\n")
	commit()
	if delta, err := release.ComponentSourceChanges(root, "tool", origin, "HEAD"); err != nil || delta != "" {
		t.Fatalf("unchanged Tool requires a spurious new version: %s %v", delta, err)
	}
	write("src/runtime.go", "changed")
	commit()
	if delta, err := release.ComponentSourceChanges(root, "tool", origin, "HEAD"); err != nil || !strings.Contains(delta, "src/runtime.go") {
		t.Fatal("changed Tool reused immutable files")
	}
}

func TestPackageExactInventoryRejectsMissingAndSubstituteEngine(t *testing.T) {
	registered := map[string]string{"first": "1.0.0", "second": "2.0.0", "newly-registered": "1.0.0"}
	components := []release.PublishedComponent{}
	for id, kind := range map[string]string{"tool": "tool", "profiles": "profiles", "first": "engine", "second": "engine", "newly-registered": "engine"} {
		components = append(components, release.PublishedComponent{ID: id, Kind: kind, Version: "1.0.0", SourceSHA: strings.Repeat("a", 40), Files: []release.File{{Name: id + ".whl", SHA256: strings.Repeat("b", 64), URL: "https://example.invalid/" + id + ".whl", Platform: "any"}}})
	}
	makeSnapshot := func(c []release.PublishedComponent) release.Snapshot {
		t.Helper()
		r := release.Registry{NumericBase: "0.3.8"}
		s, err := release.RefreshPackage(&r, strings.Repeat("a", 40), c)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	if err := release.CompletePackage(makeSnapshot(components), registered); err != nil {
		t.Fatal("registered inventory used a fixed seven-engine count")
	}
	for index, c := range components {
		missing := append([]release.PublishedComponent{}, components[:index]...)
		missing = append(missing, components[index+1:]...)
		if err := release.CompletePackage(makeSnapshot(missing), registered); err == nil {
			t.Fatalf("Package omitted %s", c.ID)
		}
	}
	substitute := append([]release.PublishedComponent{}, components...)
	for i := range substitute {
		if substitute[i].ID == "first" {
			substitute[i].ID = "unknown-engine"
		}
	}
	if err := release.CompletePackage(makeSnapshot(substitute), registered); err == nil {
		t.Fatal("same component count admitted an unregistered substitute")
	}
	if err := release.InstallPackage(makeSnapshot(substitute), "windows-x64", "must-never-execute", registered); err == nil {
		t.Fatal("incomplete Package reached installation adapter")
	}
}
