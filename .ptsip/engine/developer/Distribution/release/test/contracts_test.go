package releasetest

import (
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

func TestEveryAdmittedReleaseType(t *testing.T) {
	cases := []struct {
		tool, profiles bool
		engines        int
		full           bool
		want           string
	}{
		{true, true, 7, true, "PTSIP_PACKAGE"}, {true, false, 0, false, "TOOL_ONLY"},
		{false, false, 1, false, "ENGINE_ONLY"}, {false, true, 0, false, "PROFILES_ONLY"},
		{true, false, 2, false, "TOOL_ENGINES"}, {true, true, 0, false, "TOOL_PROFILES"},
		{false, true, 2, false, "ENGINES_PROFILES"}, {false, false, 2, false, "MULTIPLE_ENGINES"},
	}
	for _, c := range cases {
		got, err := release.Classify(c.tool, c.profiles, c.engines, c.full)
		if err != nil || got != c.want {
			t.Fatalf("release type mismatch: %s, %v", got, err)
		}
	}
	if _, err := release.Classify(false, false, 0, false); err == nil {
		t.Fatal("empty release accepted")
	}
	if _, err := release.Classify(true, true, 1, false); err == nil {
		t.Fatal("partial selection misreported as full Package")
	}
}

func TestRequiredWindowsLinuxAndOptionalMacMatrix(t *testing.T) {
	s := release.Selection{Components: []release.Component{{ID: "context", Kind: "engine", Version: "0.2.0"}, {ID: "projection", Kind: "engine", Version: "0.2.0"}}, Platforms: []string{"windows-x64", "linux-x64"}}
	runners := map[string]string{"windows-x64": "windows-2025", "linux-x64": "ubuntu-24.04", "macos-arm64": "macos-15"}
	matrix, err := release.Matrix(s, runners)
	if err != nil || len(matrix) != 4 {
		t.Fatalf("default matrix: %v", err)
	}
	for _, row := range matrix {
		if row.Platform == "macos-arm64" {
			t.Fatal("macOS scheduled without request")
		}
	}
	s.Platforms = append(s.Platforms, "macos-arm64")
	matrix, err = release.Matrix(s, runners)
	if err != nil || len(matrix) != 6 {
		t.Fatalf("optional matrix: %v", err)
	}
}

func TestPackageRefreshLatestPinAndCollision(t *testing.T) {
	registry := release.Registry{NumericBase: "0.3.8"}
	component := release.PublishedComponent{ID: "context", Kind: "engine", Version: "0.2.0", SourceSHA: strings.Repeat("a", 40), Files: []release.File{{Name: "context.whl", SHA256: strings.Repeat("a", 64), URL: "https://example.invalid/context.whl", Platform: "linux-x64"}}}
	first, err := release.RefreshPackage(&registry, strings.Repeat("a", 40), []release.PublishedComponent{component})
	if err != nil {
		t.Fatal(err)
	}
	component.Files[0].SHA256 = strings.Repeat("b", 64)
	second, err := release.RefreshPackage(&registry, strings.Repeat("b", 40), []release.PublishedComponent{component})
	if err != nil {
		t.Fatal(err)
	}
	if first.Version == second.Version || !strings.HasPrefix(second.Version, "0.3.8.") {
		t.Fatal("Package base changed or artifact change lost")
	}
	latest, err := release.ResolvePackage(registry, "0.3.8")
	if err != nil || latest.Version != second.Version {
		t.Fatal("base did not select latest snapshot")
	}
	pinned, err := release.ResolvePackage(registry, first.Version)
	if err != nil || pinned.Digest != first.Digest {
		t.Fatal("exact selector did not preserve old snapshot")
	}
	if _, err := release.ResolvePackage(registry, "0.3.8.ffffff"); err == nil {
		t.Fatal("unknown SHA was accepted")
	}
	before := len(registry.Snapshots)
	again, err := release.RefreshPackage(&registry, strings.Repeat("b", 40), []release.PublishedComponent{component})
	if err != nil || again.Version != second.Version || len(registry.Snapshots) != before {
		t.Fatal("repeat publication was not idempotent")
	}
	registry.Snapshots[len(registry.Snapshots)-1].Digest = strings.Repeat("f", 64)
	if _, err := release.RefreshPackage(&registry, strings.Repeat("b", 40), []release.PublishedComponent{component}); err == nil {
		t.Fatal("SHA6 collision did not fail closed")
	}
}
