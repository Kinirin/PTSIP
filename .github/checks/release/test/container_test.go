package releasetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func frozenContainer(t *testing.T) (release.ContainerContract, release.ShippingManifest) {
	t.Helper()
	c, err := release.ReadContainerContract(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/container-shipping.json")
	if err != nil {
		t.Fatal(err)
	}
	if release.Digest(b) != c.ManifestSHA {
		t.Fatal("frozen source fixture differs from original published bytes")
	}
	var m release.ShippingManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return c, m
}

func TestTemporaryContainerPreservesPublishedPackageAndAllSevenEngines(t *testing.T) {
	c, m := frozenContainer(t)
	files, err := release.ContainerWheels(c, m)
	if err != nil || len(files) != 8 {
		t.Fatalf("full container set: %d %v", len(files), err)
	}
	for _, f := range files {
		if strings.Contains(f.Name, "win_amd64") || strings.HasPrefix(f.Name, "ptsip_tool") {
			t.Fatal("wrong host or standalone Tool selected")
		}
	}
	m.PackageDistribution.Version = "0.3.8b4"
	if _, err := release.ContainerWheels(c, m); err == nil {
		t.Fatal("unapproved application version selected")
	}
	c, m = frozenContainer(t)
	m.Selection.SourceSHA = strings.Repeat("b", 40)
	if _, err := release.ContainerWheels(c, m); err == nil {
		t.Fatal("nearby source replaced original release")
	}
	c, m = frozenContainer(t)
	for i := range m.Snapshot.Components {
		if m.Snapshot.Components[i].Kind == "engine" {
			for j := range m.Snapshot.Components[i].Files {
				if m.Snapshot.Components[i].Files[j].Platform == "linux-x64" {
					m.Snapshot.Components[i].Files[j].Platform = "macos-arm64"
				}
			}
			break
		}
	}
	if _, err := release.ContainerWheels(c, m); err == nil {
		t.Fatal("Package with missing Linux engine admitted")
	}
}

func TestContainerModeNeedsNoCurrentPythonCompositionAuthority(t *testing.T) {
	root := t.TempDir()
	for _, ref := range []string{".github/test_modes.yaml", ".github/checks/release/test/container-verification.json", release.ContainerContractPath} {
		b, err := os.ReadFile(filepath.Join(repoRoot(t), ref))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, ref)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"git", "init", "--quiet"}, {"git", "config", "user.name", "fixture"}, {"git", "config", "user.email", "fixture@example.invalid"}, {"git", "add", "."}, {"git", "commit", "--quiet", "-m", "fixture"}} {
		if _, err := release.Run(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Plan(root, release.ContainerMode, "package", "")
	if err != nil || len(p.Matrix.Include) != 1 || len(p.Join.Include) != 0 || p.Selection.SourceSHA == release.ContainerSource {
		t.Fatalf("container recipe selection: %+v %v", p, err)
	}
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); !os.IsNotExist(err) {
		t.Fatal("test accidentally supplied a Python canonical source")
	}
	for _, target := range []string{"tool", "context", "context,projection"} {
		if _, err := Plan(root, release.ContainerMode, target, ""); err == nil {
			t.Fatal("partial temporary Package admitted")
		}
	}
	if _, err := Plan(root, release.ContainerMode, "package", "macos-arm64"); err == nil {
		t.Fatal("unbuilt container platform admitted")
	}
}

func TestContainerInstructionCannotEnablePyPIOrPolicyCutover(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), release.ContainerContractPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"pypi_publication", "canonical_current_composition_owner", "successor_info_implementation_cutover", "automated_policy_validation_available", "merge_to_main", "runtime_activation"} {
		root := t.TempDir()
		p := filepath.Join(root, release.ContainerContractPath)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		mutated := strings.Replace(string(b), `"`+field+`": false`, `"`+field+`": true`, 1)
		if err := os.WriteFile(p, []byte(mutated), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := release.ReadContainerContract(root); err == nil {
			t.Fatal("temporary publication expanded authority", field)
		}
	}
}

func TestContainerReceiptRejectsChangedArchiveAndRecipeSource(t *testing.T) {
	root := repoRoot(t)
	s, err := release.ContainerSelection(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := release.ReadContainerContract(root)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	data := []byte("verified-image-archive")
	p := filepath.Join(dir, "image.tar.gz")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
	r := release.ContainerReceipt{Schema: "ptsip-temporary-container-test/v1", Source: s.SourceSHA, OriginalSource: release.ContainerSource, Selection: s.SelectionSHA, ImageID: "sha256:" + strings.Repeat("a", 64), ImageTag: "ptsip-container-test:" + s.SourceSHA, Archive: release.File{Name: "image.tar.gz", SHA256: release.Digest(data)}, ManifestSHA: c.ManifestSHA, Status: "PASS", Runtime: map[string]any{"engine_count": float64(7), "all_installed": true, "execution_authorized": false}}
	if err := release.ValidateContainerReceipt(root, s, r, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("changed-image"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := release.ValidateContainerReceipt(root, s, r, dir); err == nil {
		t.Fatal("modified image archive admitted")
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
	r.Source = release.ContainerSource
	if err := release.ValidateContainerReceipt(root, s, r, dir); err == nil {
		t.Fatal("application SHA substituted for container recipe SHA")
	}
}
