package releasetest

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

// Ported from b5: current developer release controller owns these tests.
const bundleSource = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

var bundleKinds = []string{"context", "storage", "validation", "discovery", "planning", "execution", "projection"}
var bundleTags = map[string]string{"linux-x64": "manylinux_2_39_x86_64", "windows-x64": "win_amd64", "macos-arm64": "macosx_11_0_arm64"}
var bundleTargets = map[string]string{"linux-x64": "x86_64-unknown-linux-gnu", "windows-x64": "x86_64-pc-windows-msvc", "macos-arm64": "aarch64-apple-darwin"}

func bundleTestArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	names := []string{}
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func bundleTestContents(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
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
		files[f.Name] = b
	}
	return files
}

type packageBundleFixture struct {
	tool, profiles []byte
	snapshot       release.Snapshot
	registered     map[string]string
	native         release.PackageNativePayload
}

func makePackageBundleFixture(t *testing.T, platform string) packageBundleFixture {
	t.Helper()
	f := packageBundleFixture{registered: map[string]string{}, native: release.PackageNativePayload{Platform: platform, Wheels: map[string][]byte{}}}
	entries := []map[string]any{}
	components := []release.PublishedComponent{}
	for _, kind := range bundleKinds {
		version := "0.3.1"
		if kind == "context" || kind == "projection" {
			version = "0.2.1"
		}
		identity := "urn:uuid:" + kind
		project := "ptsip-engine-" + kind
		entrypoint := project
		if platform == "windows-x64" {
			entrypoint += ".exe"
		}
		binary := append([]byte("\x7fELF"), []byte(kind)...)
		if platform == "windows-x64" {
			binary = append([]byte("MZ"), []byte(kind)...)
		}
		if platform == "macos-arm64" {
			binary = append([]byte{0xcf, 0xfa, 0xed, 0xfe}, []byte(kind)...)
		}
		origin := map[string]any{"schema_version": "ptsip-independent-wheel-origin/v1", "source_sha": bundleSource, "engine_id": identity, "engine_version": version, "host_target": bundleTargets[platform], "entrypoint": entrypoint, "binary_sha256": release.Digest(binary), "execution_authorized": false}
		base := strings.ReplaceAll(project, "-", "_") + "-" + version
		var wheel []byte
		raw, _ := json.Marshal(origin)
		wheel = bundleTestArchive(t, map[string][]byte{base + ".dist-info/METADATA": []byte("Metadata-Version: 2.4\nName: " + project + "\nVersion: " + version + "\n\nengine\n"), base + ".dist-info/WHEEL": []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: false\nTag: py3-none-" + bundleTags[platform] + "\n"), base + ".dist-info/ptsip-origin.json": raw, base + ".data/scripts/" + entrypoint: binary})
		name := base + "-py3-none-" + bundleTags[platform] + ".whl"
		f.native.Wheels[kind] = wheel
		f.registered[kind] = version
		entries = append(entries, map[string]any{"kind": strings.ToUpper(kind), "engine_id": identity, "version": version})
		components = append(components, release.PublishedComponent{ID: kind, Kind: "engine", Version: version, SourceSHA: origin["source_sha"].(string), Files: []release.File{{Name: name, SHA256: release.Digest(wheel), Platform: platform, URL: "https://example.invalid/" + name}}})
	}
	index, _ := json.Marshal(map[string]any{"official_engine_entries": entries})
	f.tool = bundleTestArchive(t, map[string][]byte{"ptsip/__init__.py": []byte(""), "ptsip/engines/index.json": index, "ptsip_tool-0.3.8b3.dist-info/METADATA": []byte("Metadata-Version: 2.4\nName: ptsip-tool\nVersion: 0.3.8b3\nRequires-Dist: PyYAML>=6\nRequires-Dist: ptsip-engine-context==0.2.1\n\nTool\n"), "ptsip_tool-0.3.8b3.dist-info/entry_points.txt": []byte("[console_scripts]\nptsip=ptsip.cli:main\n")})
	f.profiles = bundleTestArchive(t, map[string][]byte{"src/ptsip/profiles/index.yaml": []byte("catalog"), "src/ptsip/profiles/main.ptsip.yaml": []byte("example"), ".ptsip/contracts/release/profile/project-profile-contracts.yaml": []byte("current: pp.1.02\n")})
	for _, c := range []struct {
		id, kind, version string
		data              []byte
	}{{"tool", "tool", "T.0.3.8.b3", f.tool}, {"profiles", "profiles", "pp.1.02", f.profiles}} {
		components = append(components, release.PublishedComponent{ID: c.id, Kind: c.kind, Version: c.version, SourceSHA: bundleSource, Files: []release.File{{Name: c.id + ".zip", SHA256: release.Digest(c.data), Platform: "any", URL: "https://example.invalid/" + c.id}}})
	}
	registry := release.Registry{NumericBase: "0.3.8"}
	var err error
	f.snapshot, err = release.RefreshPackage(&registry, bundleSource, components)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f packageBundleFixture) build() ([]byte, error) {
	return release.BundledPackageWheel(f.tool, f.profiles, f.snapshot, f.registered, "0.3.8b3", f.native)
}
func TestCompleteHostBundles(t *testing.T) {
	for _, host := range []string{"linux-x64", "windows-x64", "macos-arm64"} {
		t.Run(host, func(t *testing.T) {
			f := makePackageBundleFixture(t, host)
			data, err := f.build()
			if err != nil {
				t.Fatal(err)
			}
			files := bundleTestContents(t, data)
			di := "ptsip-0.3.8b3.dist-info/"
			meta := string(files[di+"METADATA"])
			if strings.Contains(meta, "Requires-Dist: ptsip-engine-") || !strings.Contains(meta, "Requires-Dist: PyYAML>=6") {
				t.Fatal("dependencies not projected correctly")
			}
			if !strings.Contains(string(files[di+"WHEEL"]), "Root-Is-Purelib: false\nTag: py3-none-"+bundleTags[host]) {
				t.Fatal("wrong compatibility tag")
			}
			for _, kind := range bundleKinds {
				name := "ptsip-0.3.8b3.data/scripts/ptsip-engine-" + kind
				if host == "windows-x64" {
					name += ".exe"
				}
				if len(files[name]) == 0 {
					t.Fatal("native executable omitted")
				}
			}
			rows, err := csv.NewReader(bytes.NewReader(files[di+"RECORD"])).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != len(files) {
				t.Fatal("incomplete RECORD")
			}
			for _, row := range rows {
				if row[0] == di+"RECORD" {
					continue
				}
				hash := release.Digest(files[row[0]])
				raw := make([]byte, 32)
				for i := range raw {
					var value byte
					for _, c := range hash[i*2 : i*2+2] {
						value *= 16
						if c >= 'a' {
							value += byte(c - 'a' + 10)
						} else {
							value += byte(c - '0')
						}
					}
					raw[i] = value
				}
				if row[1] != "sha256="+base64.RawURLEncoding.EncodeToString(raw) {
					t.Fatal("invalid RECORD hash")
				}
			}
			z, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			for _, file := range z.File {
				if file.Flags&8 != 0 {
					t.Fatal("streaming descriptor retained")
				}
				if strings.Contains(file.Name, ".data/scripts/") && file.Mode().Perm()&0111 == 0 {
					t.Fatal("executable permission lost")
				}
			}
			name, err := release.BundledPackageWheelName("0.3.8b3", host)
			if err != nil {
				t.Fatal(err)
			}
			if name != "ptsip-0.3.8b3-py3-none-"+bundleTags[host]+".whl" {
				t.Fatal("wrong bundle filename")
			}
		})
	}
}
func TestRejectsIncompleteAndUnadmittedInputs(t *testing.T) {
	cases := map[string]func(*packageBundleFixture){
		"missing engine":     func(f *packageBundleFixture) { delete(f.native.Wheels, "storage") },
		"extra engine":       func(f *packageBundleFixture) { f.native.Wheels["foreign"] = []byte("bad") },
		"substituted engine": func(f *packageBundleFixture) { f.native.Wheels["storage"] = f.native.Wheels["context"] },
		"changed digest":     func(f *packageBundleFixture) { f.native.Wheels["storage"] = append(f.native.Wheels["storage"], 0) },
		"unknown host":       func(f *packageBundleFixture) { f.native.Platform = "unknown" },
		"other host":         func(f *packageBundleFixture) { f.native.Platform = "windows-x64" },
		"registry version":   func(f *packageBundleFixture) { f.registered["storage"] = "9.9.9" },
		"normalized engine dependency": func(f *packageBundleFixture) {
			files := bundleTestContents(t, f.tool)
			files["ptsip_tool-0.3.8b3.dist-info/METADATA"] = []byte("Name: ptsip-tool\nVersion: 0.3.8b3\nRequires-Dist: PTSIP_Engine_Context==0.2.1\n\nTool\n")
			f.tool = bundleTestArchive(t, files)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := makePackageBundleFixture(t, "linux-x64")
			mutate(&f)
			if _, err := f.build(); err == nil {
				t.Fatal("unadmitted payload accepted")
			}
		})
	}
}
func TestRejectsCorruptNativeMetadataEvenWithMatchingArchiveDigest(t *testing.T) {
	for _, scenario := range []string{"engine id", "bundleSource", "host", "binary", "grant", "duplicate", "traversal", "metadata version", "wheel tag"} {
		t.Run(scenario, func(t *testing.T) {
			f := makePackageBundleFixture(t, "linux-x64")
			files := bundleTestContents(t, f.native.Wheels["storage"])
			base := "ptsip_engine_storage-0.3.1"
			var origin map[string]any
			json.Unmarshal(files[base+".dist-info/ptsip-origin.json"], &origin)
			switch scenario {
			case "engine id":
				origin["engine_id"] = "foreign"
			case "bundleSource":
				origin["source_sha"] = strings.Repeat("b", 40)
			case "host":
				origin["host_target"] = bundleTargets["windows-x64"]
			case "binary":
				files[base+".data/scripts/ptsip-engine-storage"] = []byte("tampered")
			case "grant":
				origin["execution_authorized"] = true
			case "traversal":
				files["../escape"] = []byte("bad")
			case "metadata version":
				files[base+".dist-info/METADATA"] = []byte("Name: ptsip-engine-storage\nVersion: 9.9.9\n\n")
			case "wheel tag":
				files[base+".dist-info/WHEEL"] = []byte("Root-Is-Purelib: false\nTag: py3-none-win_amd64\n")
			}
			files[base+".dist-info/ptsip-origin.json"], _ = json.Marshal(origin)
			data := bundleTestArchive(t, files)
			if scenario == "duplicate" {
				var b bytes.Buffer
				z := zip.NewWriter(&b)
				for name, value := range files {
					w, _ := z.Create(name)
					w.Write(value)
				}
				w, _ := z.Create(base + ".dist-info/ptsip-origin.json")
				w.Write(files[base+".dist-info/ptsip-origin.json"])
				z.Close()
				data = b.Bytes()
			}
			f.native.Wheels["storage"] = data
			for i, c := range f.snapshot.Components {
				if c.ID == "storage" {
					f.snapshot.Components[i].Files[0].SHA256 = release.Digest(data)
				}
			}
			reg := release.Registry{NumericBase: "0.3.8"}
			var err error
			f.snapshot, err = release.RefreshPackage(&reg, bundleSource, f.snapshot.Components)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.build(); err == nil {
				t.Fatal("corrupt native origin accepted")
			}
		})
	}
}
