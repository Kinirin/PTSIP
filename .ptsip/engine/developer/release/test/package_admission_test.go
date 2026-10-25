package releasetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
)

func TestPackageAdmissionRequiresEachNativeHostAndVerifiedSource(t *testing.T) {
	for _, scenario := range []string{"complete", "missing source", "failed host", "changed wheel", "wrong snapshot", "wrong host"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			f := makePackageBundleFixture(t, "linux-x64")
			s := release.Selection{SourceSHA: bundleSource, SelectionSHA: strings.Repeat("b", 64), Tag: "ptsip-release-" + bundleSource, Platforms: []string{"windows-x64", "linux-x64"}}
			for _, host := range s.Platforms {
				dir := filepath.Join(root, "verified", host)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				name, err := release.BundledPackageWheelName("0.3.8b3", host)
				if err != nil {
					t.Fatal(err)
				}
				data := []byte("verified complete wheel for " + host)
				r := release.PackageReceipt{Schema: "ptsip-pip-package-test-receipt/v2", Source: s.SourceSHA, Selection: s.SelectionSHA, Platform: host, PyPI: "0.3.8b3", Snapshot: f.snapshot, Wheel: release.File{Name: name, SHA256: release.Digest(data), Platform: host}, Status: "PASS"}
				if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
					t.Fatal(err)
				}
				if host == "linux-x64" {
					source := []byte("verified complete source archive")
					r.SourceDist = &release.File{Name: "ptsip-0.3.8b3.tar.gz", SHA256: release.Digest(source), Platform: "any"}
					if err := os.WriteFile(filepath.Join(dir, r.SourceDist.Name), source, 0644); err != nil {
						t.Fatal(err)
					}
					switch scenario {
					case "missing source":
						r.SourceDist = nil
					case "failed host":
						r.Status = "FAIL"
					case "changed wheel":
						os.WriteFile(filepath.Join(dir, name), []byte("changed"), 0644)
					case "wrong snapshot":
						r.Snapshot.Digest = strings.Repeat("c", 64)
					case "wrong host":
						r.Wheel.Platform = "windows-x64"
					}
				}
				if err := release.WriteJSON(filepath.Join(dir, "package-receipt.json"), r); err != nil {
					t.Fatal(err)
				}
			}
			component, err := release.ConsumePackageReceipt(root, s, f.snapshot, "verified", "shipping", "0.3.8b3")
			if scenario != "complete" {
				if err == nil {
					t.Fatal("unverified Package was admitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(component.Files) != 3 {
				t.Fatal("Package is missing a host wheel or source distribution")
			}
		})
	}
}

func TestCompletePackagePublicationTargetsOnlyPTSIPDespiteIntermediateEngineFiles(t *testing.T) {
	root := t.TempDir()
	route, err := os.ReadFile(filepath.Join(repoRoot(t), ".github/checks/release/pypi-publishers.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, ".github/checks/release")
	os.MkdirAll(p, 0755)
	os.WriteFile(filepath.Join(p, "pypi-publishers.json"), route, 0644)
	m := release.ShippingManifest{PackageDistribution: &release.PublishedComponent{ID: "package", Kind: "package", Version: "0.3.8b3"}}
	for _, kind := range bundleKinds {
		m.Selection.Components = append(m.Selection.Components, release.Component{ID: kind, Kind: "engine"})
		dir := filepath.Join(root, "shipping/pypi", kind)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "engine.whl"), []byte("intermediate"), 0644)
	}
	dir := filepath.Join(root, "shipping/pypi/package")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "ptsip.whl"), []byte("complete"), 0644)
	targets, err := release.PublicationTargets(root, m, "shipping", "github-pypi")
	if err != nil || len(targets) != 1 || targets[0]["component"] != "package" {
		t.Fatalf("unexpected PyPI project targets: %v %v", targets, err)
	}
}
