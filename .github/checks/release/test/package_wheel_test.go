package releasetest

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func TestPipPackageHasEveryMandatoryEngineAndProfilePayload(t *testing.T) {
	registered := map[string]string{"context": "0.2.0", "storage": "0.3.0", "new-engine": "1.0.0"}
	components := []release.PublishedComponent{}
	for id, kind := range map[string]string{"tool": "tool", "profiles": "profiles", "context": "engine", "storage": "engine", "new-engine": "engine"} {
		v := "0.3.8b1"
		if kind == "profiles" {
			v = "pp.1.02"
		}
		if kind == "engine" {
			v = registered[id]
		}
		components = append(components, release.PublishedComponent{ID: id, Kind: kind, Version: v, SourceSHA: strings.Repeat("a", 40), Files: []release.File{{Name: id + ".whl", SHA256: strings.Repeat("b", 64), URL: "https://example.invalid/" + id + ".whl", Platform: "any"}}})
	}
	r := release.Registry{NumericBase: "0.3.8"}
	snapshot, err := release.RefreshPackage(&r, strings.Repeat("a", 40), components)
	if err != nil {
		t.Fatal(err)
	}
	makeZip := func(files map[string]string) []byte {
		t.Helper()
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		for name, text := range files {
			f, err := w.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Write([]byte(text)); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	tool := makeZip(map[string]string{"ptsip/__init__.py": "existing Tool payload", "ptsip-0.3.8b1.dist-info/METADATA": "Metadata-Version: 2.4\nName: PTSIP\nVersion: 0.3.8b1\nRequires-Dist: PyYAML>=6\n\nTool description\n", "ptsip-0.3.8b1.dist-info/entry_points.txt": "[console_scripts]\nptsip=ptsip.cli:main\n"})
	profiles := makeZip(map[string]string{"src/ptsip/profiles/index.yaml": "catalog", "src/ptsip/profiles/example.ptsip.yaml": "example", "registry/project-profile-contracts.yaml": "current: pp.1.02\n"})
	wheel, err := release.PackageWheel(tool, profiles, snapshot, registered, "0.3.8.1b1")
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(wheel), int64(len(wheel)))
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		contents[f.Name] = string(b)
	}
	metadata := contents["ptsip-0.3.8.1b1.dist-info/METADATA"]
	for id, v := range registered {
		if !strings.Contains(metadata, "Requires-Dist: ptsip-engine-"+id+"=="+v+"\n") {
			t.Fatalf("pip Package omitted mandatory engine %s", id)
		}
	}
	if strings.Contains(metadata, "extra ==") || strings.Contains(metadata, "sys_platform") {
		t.Fatal("Package permits engine omission")
	}
	if contents["ptsip/profiles/index.yaml"] == "" || contents["ptsip/package-composition.json"] == "" || !strings.Contains(metadata, "Version: 0.3.8.1b1") {
		t.Fatal("Package payload or independent version projection absent")
	}
	if _, err := release.PackageWheel(tool, profiles, snapshot, map[string]string{"context": "0.2.0"}, "0.3.8.1b1"); err == nil {
		t.Fatal("Package wheel assembly accepted a subset of registered engines")
	}
}
