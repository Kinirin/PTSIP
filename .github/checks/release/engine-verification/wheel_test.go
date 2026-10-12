package enginewheel

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
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
	packages := []map[string]any{}
	for _, kind := range kinds {
		id := "fixture-" + kind
		manifest := "src/ptsip/engines/" + kind + "/manifest.json"
		binding := "src/ptsip/engines/" + kind + "/execution-bindings.json"
		entries = append(entries, map[string]any{"kind": strings.ToUpper(kind), "engine_id": id, "version": "0.2.0", "manifest_ref": manifest, "execution_binding_ref": binding})
		packages = append(packages, map[string]any{"engine_id": id, "version": "0.2.0", "package_distribution": "ptsip-engine-" + kind, "distribution": "INDEPENDENT_VERSIONED_NATIVE_WHEEL", "installation": "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY", "build_targets": []string{target}, "permission_from_availability": false, "installed_is_execution_authorization": false})
		writeFixture(t, root, manifest, map[string]any{"engine_id": id, "implementation_version": "0.2.0"})
		writeFixture(t, root, binding, map[string]any{"bindings": []map[string]any{{"engine_id": id, "engine_version": "0.2.0", "entrypoint": "ptsip-engine-" + kind}}})
	}
	writeFixture(t, root, "src/ptsip/engines/index.json", map[string]any{"official_engine_entries": entries})
	writeFixture(t, root, "src/ptsip/engines/contracts/installation-plan.json", map[string]any{"packages": packages})
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
	return root, Options{Kind: "context", Target: target, Binary: binary, Output: filepath.Join(root, "wheelhouse"), SourceSHA: strings.Repeat("a", 40)}
}
func TestIndependentWheelForEveryDeclaredPlatform(t *testing.T) {
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
				t.Fatal("independent artifact membership mismatch")
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
func TestAssemblerRejectsUnregisteredSourcesBeforeWriting(t *testing.T) {
	for _, defect := range []string{"sha", "target", "kind", "binary", "contract", "reference", "duplicate", "missing_boolean"} {
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
			case "contract", "missing_boolean":
				var p map[string]any
				if err := document(root, "src/ptsip/engines/contracts/installation-plan.json", &p); err != nil {
					t.Fatal(err)
				}
				row := p["packages"].([]any)[0].(map[string]any)
				if defect == "contract" {
					row["installation"] = "OPERATOR_ONLY"
				} else {
					delete(row, "installed_is_execution_authorization")
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
