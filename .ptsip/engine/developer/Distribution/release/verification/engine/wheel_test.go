package enginewheel

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root, ref string, value any) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(ref))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T, target string) (string, Options) {
	t.Helper()
	root := t.TempDir()
	entries := []map[string]any{}
	components := []map[string]any{}
	defaultIDs := []string{}
	for _, kind := range kinds {
		id := "fixture-" + kind
		manifest := "src/ptsip/engines/" + kind + "/manifest.json"
		binding := "src/ptsip/engines/" + kind + "/execution-bindings.json"
		entries = append(entries, map[string]any{"kind": strings.ToUpper(kind), "engine_id": id, "version": "0.2.0", "manifest_ref": manifest, "execution_binding_ref": binding})
		components = append(components, map[string]any{"engine_id": id, "version": "0.2.0", "entrypoint": "ptsip-engine-" + kind, "build_targets": []string{target}, "permission_from_availability": false, "installed_is_execution_authorization": false})
		defaultIDs = append(defaultIDs, id)
		writeFixture(t, root, manifest, map[string]any{"engine_id": id, "implementation_version": "0.2.0"})
		writeFixture(t, root, "src/ptsip/engines/"+kind+"/version.yaml", map[string]any{"schema_version": "ptsip-engine-version/v1", "engine": kind, "version": "0.2.0"})
		writeFixture(t, root, binding, map[string]any{"bindings": []map[string]any{{"engine_id": id, "engine_version": "0.2.0", "entrypoint": "ptsip-engine-" + kind}}})
	}
	writeFixture(t, root, "src/ptsip/engines/index.json", map[string]any{"official_engine_entries": entries})
	writeFixture(t, root, "src/ptsip/engines/contracts/installation-plan.json", map[string]any{
		"schema_version": "ptsip-engine-installation-plan/v2",
		"selection": "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL",
		"default_available_engine_ids": defaultIDs,
		"components": components,
	})
	license := filepath.Join(root, "License-Authority", "LICENSE.md")
	if err := os.MkdirAll(filepath.Dir(license), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(license, []byte("fixture license"), 0644); err != nil {
		t.Fatal(err)
	}
	h := hosts[target]
	binary := filepath.Join(root, "ptsip-engine-context"+h.Suffix)
	if err := os.WriteFile(binary, append(append([]byte{}, h.Magic[0]...), bytes.Repeat([]byte("X"), 300)...), 0755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "--all"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=", "commit", "-q", "-m", "Fixture"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if data, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fixture source commit failed: %v: %s", err, data)
		}
	}
	source, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return root, Options{Kind: "context", Target: target, Binary: binary, Output: filepath.Join(root, "wheelhouse"), SourceSHA: strings.TrimSpace(string(source))}
}
func TestNativeComponentArtifactForEveryDeclaredPlatform(t *testing.T) {
	for target := range hosts {
		t.Run(target, func(t *testing.T) {
			root, opts := fixture(t, target)
			result, err := Build(root, opts)
			if err != nil {
				t.Fatal(err)
			}
			archive, err := zip.OpenReader(result.Wheel)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			if len(archive.File) != 6 {
				t.Fatal("wheel must contain one binary and its five metadata files")
			}
			names := map[string]bool{}
			scripts := 0
			var record []byte
			for _, file := range archive.File {
				names[file.Name] = true
				if strings.Contains(file.Name, "/scripts/") {
					scripts++
					if !strings.HasSuffix(file.Name, "ptsip-engine-context"+hosts[target].Suffix) || file.Mode()&0111 == 0 {
						t.Fatal("wrong selected executable or mode")
					}
				}
				if strings.HasSuffix(file.Name, "/RECORD") {
					r, err := file.Open()
					if err != nil {
						t.Fatal(err)
					}
					record, err = io.ReadAll(r)
					r.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			rows, err := csv.NewReader(bytes.NewReader(record)).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != len(names) || scripts != 1 {
				t.Fatal("native component archive membership mismatch")
			}
			for _, row := range rows {
				if !names[row[0]] {
					t.Fatal("RECORD names an unregistered member")
				}
			}
			data, err := os.ReadFile(result.Receipt)
			if err != nil {
				t.Fatal(err)
			}
			var receipt map[string]any
			if err = json.Unmarshal(data, &receipt); err != nil {
				t.Fatal(err)
			}
			if receipt["install_authorized"] != false || receipt["runtime_authorized"] != false || receipt["release_verification"] != "NOT_PERFORMED_BY_ASSEMBLER" {
				t.Fatal("assembler created authority")
			}
			first, err := os.ReadFile(result.Wheel)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Build(root, opts); err != nil {
				t.Fatal(err)
			}
			second, _ := os.ReadFile(result.Wheel)
			if !bytes.Equal(first, second) {
				t.Fatal("same source composition produced unstable wheel bytes")
			}
		})
	}
}

// Intermediate producers are not consumer installers. Packaging policy
// fields belong to the complete PTSIP Wheel join, not native preflight.
func TestNativeArtifactBuildDoesNotRequireConsumerDistributionMetadata(t *testing.T) {
	root, opts := fixture(t, "x86_64-unknown-linux-gnu")
	var plan map[string]any
	if err := document(root, "src/ptsip/engines/contracts/installation-plan.json", &plan); err != nil {
		t.Fatal(err)
	}
	if _, exists := plan["distribution"]; exists {
		t.Fatal("native-only test fixture must not declare consumer distribution")
	}
	if _, exists := plan["default_installation"]; exists {
		t.Fatal("native-only test fixture must not declare pip installation")
	}
	if _, err := Build(root, opts); err != nil {
		t.Fatalf("native artifact build improperly depends on consumer distribution: %v", err)
	}
}

func TestWheelUsesCanonicalLicenseBlobDespiteWorktreeLineEndings(t *testing.T) {
	root, opts := fixture(t, "x86_64-pc-windows-msvc")
	if err := os.WriteFile(filepath.Join(root, "License-Authority/LICENSE.md"), []byte("fixture license\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := Build(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(result.Wheel)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	found := false
	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/licenses/LICENSE.md") {
			r, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil || string(data) != "fixture license" {
				t.Fatalf("worktree replaced canonical license blob: %q, %v", data, err)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("canonical license missing from wheel")
	}
}
func TestAssemblerRejectsUnregisteredSourcesBeforeWriting(t *testing.T) {
	for _, defect := range []string{"sha", "target", "kind", "binary", "contract", "reference", "duplicate", "missing_boolean", "wrong_version", "missing_permission", "incomplete_components", "wrong_schema", "duplicate_defaults"} {
		t.Run(defect, func(t *testing.T) {
			root, opts := fixture(t, "x86_64-unknown-linux-gnu")
			switch defect {
			case "sha":
				opts.SourceSHA = "unknown"
			case "target":
				opts.Target = "aarch64-unknown-linux-gnu"
			case "kind":
				opts.Kind = "unregistered"
			case "binary":
				if err := os.WriteFile(opts.Binary, bytes.Repeat([]byte("X"), 300), 0755); err != nil {
					t.Fatal(err)
				}
			case "contract", "missing_boolean", "wrong_version", "missing_permission", "incomplete_components", "wrong_schema", "duplicate_defaults":
				var p map[string]any
				if err := document(root, "src/ptsip/engines/contracts/installation-plan.json", &p); err != nil {
					t.Fatal(err)
				}
				row := p["components"].([]any)[0].(map[string]any)
				switch defect {
				case "contract":
					row["entrypoint"] = "ptsip-engine-wrong"
				case "missing_boolean":
					delete(row, "installed_is_execution_authorization")
				case "wrong_version":
					row["version"] = "9.0.0"
				case "missing_permission":
					delete(row, "permission_from_availability")
				case "incomplete_components":
					p["components"] = p["components"].([]any)[:6]
				case "wrong_schema":
					p["schema_version"] = "ptsip-engine-installation-plan/v1"
				case "duplicate_defaults":
					p["default_available_engine_ids"].([]any)[1] = p["default_available_engine_ids"].([]any)[0]
				}
				writeFixture(t, root, "src/ptsip/engines/contracts/installation-plan.json", p)
			case "reference", "duplicate":
				var index map[string]any
				if err := document(root, "src/ptsip/engines/index.json", &index); err != nil {
					t.Fatal(err)
				}
				rows := index["official_engine_entries"].([]any)
				if defect == "reference" {
					rows[0].(map[string]any)["manifest_ref"] = "../foreign.json"
				} else {
					rows[1] = rows[0]
				}
				writeFixture(t, root, "src/ptsip/engines/index.json", index)
			}
			if _, err := Build(root, opts); err == nil {
				t.Fatal("untrusted assembly accepted", defect)
			}
			if _, err := os.Stat(opts.Output); !os.IsNotExist(err) {
				t.Fatal("rejected input wrote an artifact", err)
			}
		})
	}
}
