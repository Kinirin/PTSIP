package engines_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func installedEnvironment(t *testing.T) (object, string, string) {
	t.Helper()
	registered := copyRegisteredSource(t)
	root := t.TempDir()
	packageRoot := filepath.Join(root, "site", "ptsip")
	distribution := filepath.Join(root, "site", "ptsip-0.3.8b6.dist-info")
	scripts := filepath.Join(root, "scripts")
	for _, dir := range []string{packageRoot, distribution, scripts} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	sourceRoot := filepath.Join(registered, "src", "ptsip", "engines")
	if err := filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		relative, e := filepath.Rel(sourceRoot, path)
		if e != nil {
			return e
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		target := filepath.Join(packageRoot, "engines", relative)
		if e := os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			return e
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	platform, target, suffix := "linux-x64", "x86_64-unknown-linux-gnu", ""
	if runtime.GOOS == "windows" {
		platform, target, suffix = "windows-x64", "x86_64-pc-windows-msvc", ".exe"
	}
	if runtime.GOOS == "darwin" {
		platform, target = "macos-arm64", "aarch64-apple-darwin"
	}
	origins := object{}
	for _, raw := range entries(t) {
		e := raw.(object)
		name := strings.ToLower(e["kind"].(string))
		// Hashable inert artifacts: diagnostics must never execute inspected engines.
		filename := "ptsip-engine-" + name + suffix
		bytes := []byte("inert-native-artifact-" + name)
		if err := os.WriteFile(filepath.Join(scripts, filename), bytes, 0755); err != nil {
			t.Fatal(err)
		}
		origins[name] = object{"schema_version": "ptsip-independent-wheel-origin/v1", "engine_id": e["engine_id"], "engine_version": e["version"], "host_target": target, "entrypoint": filename, "binary_sha256": fileHash(t, scripts, filename), "source_sha": strings.Repeat("a", 40), "execution_authorized": false}
		transport, err := os.ReadFile(filepath.Join(testrepo.Root(t), "src", "ptsip", "engines", name, "transport.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(packageRoot, "engines", name, "transport.json"), transport, 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON(t, distribution, "ptsip-engines.json", object{"schema_version": "ptsip-bundled-engine-origins/v1", "package_version": "0.3.8b6", "platform": platform, "engines": origins, "execution_authorized": false})
	if err := os.WriteFile(filepath.Join(distribution, "METADATA"), []byte("Metadata-Version: 2.4\r\nName: PTSIP\r\nVersion: 0.3.8b6\r\n\r\nDescription\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return object{"package_root": packageRoot, "distribution_root": distribution, "scripts_root": scripts}, packageRoot, scripts
}

func TestOfficialEngineCompatibilityCLIEncoding(t *testing.T) {
	root := testrepo.Root(t)
	code := `import contextlib, io, json
from ptsip import cli
report={'schema_version':'ptsip-official-engine-compatibility-error/v1','status':'UNVERIFIED','reason':'파일 누락 🚀','execution_authorized':False}
cli.official_engine_compatibility=lambda: (report,4)
buffer=io.BytesIO()
stream=io.TextIOWrapper(buffer,encoding='cp949')
with contextlib.redirect_stdout(stream):
 assert cli.main(['engines','compatibility','--json'])==4
stream.flush()
raw=buffer.getvalue()
assert json.loads(raw.decode('utf-8'))==report
print('Legacy-console JSON roundtrip PASS')
`
	cmd := exec.Command("python", "-c", code)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(root, "src"), "PYTHONIOENCODING=cp949")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("diagnostic JSON must survive Windows legacy encodings: %v\n%s", err, output)
	}
}

func environmentRow(t *testing.T, output object, kind string) object {
	t.Helper()
	for _, raw := range output["entries"].([]any) {
		if row := raw.(object); row["engine"] == kind {
			return row
		}
	}
	t.Fatal("missing diagnostic row", kind)
	return nil
}

func addEnvironmentDependency(t *testing.T, packageRoot string, version string, stale bool) {
	t.Helper()
	c := entry(t, "CONTEXT")
	p := entry(t, "STORAGE")
	cc := document(t, c["contract_ref"].(string))
	pc := document(t, p["contract_ref"].(string))
	refs := []any{object{"id": cc["contract_id"], "version": cc["version"], "path": c["contract_ref"], "sha256": digest(t, c["contract_ref"].(string))}, object{"id": pc["contract_id"], "version": pc["version"], "path": p["contract_ref"], "sha256": digest(t, p["contract_ref"].(string))}}
	if stale {
		refs[0].(object)["sha256"] = strings.Repeat("0", 64)
	}
	evidence := object{"id": "test-compatibility-evidence", "version": "1.0.0", "consumer_engine_id": c["engine_id"], "provider_engine_id": p["engine_id"], "outcome": "PASS", "source_refs": refs}
	hash := writeJSON(t, packageRoot, "engines/context/compatibility-evidence.json", evidence)
	m := document(t, c["manifest_ref"].(string))
	m["dependencies"] = []any{object{"provider_engine_id": p["engine_id"], "capability_id": pc["contract_id"], "contract_version": version, "compatibility_ref": object{"id": evidence["id"], "version": evidence["version"], "path": "src/ptsip/engines/context/compatibility-evidence.json", "sha256": hash}}}
	writeJSON(t, packageRoot, "engines/context/manifest.json", m)
}

func TestOfficialEngineEnvironmentDiagnostics(t *testing.T) {
	binary := os.Getenv("PTSIP_TEST_DISCOVERY_BINARY")
	if binary == "" {
		binary = native(t, "DISCOVERY")
	}
	t.Run("installed catalog does not imply compatibility", func(t *testing.T) {
		input, _, _ := installedEnvironment(t)
		out := invoke(t, binary, "DISCOVERY", "discovery.check-environment", input, true)
		if out["installation_integrity"] != "VERIFIED" || out["status"] != "UNVERIFIED" || out["pairwise_compatibility_guaranteed"] != false || out["engine_count"] != float64(7) {
			t.Fatal(out)
		}
		row := environmentRow(t, out, "PLANNING")
		if len(row["operations"].([]any)) == 0 || row["transport_agreement"] != "VERIFIED" {
			t.Fatal(row)
		}
		operation := row["operations"].([]any)[0].(object)
		if operation["request_template"] == nil || operation["input_schema"] == nil || operation["template_is_executable_request"] != false {
			t.Fatal(operation)
		}
	})
	t.Run("current two sided contract evidence", func(t *testing.T) {
		input, pkg, _ := installedEnvironment(t)
		version := document(t, entry(t, "STORAGE")["contract_ref"].(string))["version"].(string)
		addEnvironmentDependency(t, pkg, version, false)
		out := invoke(t, binary, "DISCOVERY", "discovery.check-environment", input, true)
		if environmentRow(t, out, "CONTEXT")["dependency_compatibility"] != "VERIFIED" || out["status"] != "UNVERIFIED" {
			t.Fatal(out)
		}
	})
	t.Run("stale evidence is not compatible", func(t *testing.T) {
		input, pkg, _ := installedEnvironment(t)
		version := document(t, entry(t, "STORAGE")["contract_ref"].(string))["version"].(string)
		addEnvironmentDependency(t, pkg, version, true)
		out := invoke(t, binary, "DISCOVERY", "discovery.check-environment", input, true)
		if environmentRow(t, out, "CONTEXT")["dependency_compatibility"] != "UNVERIFIED" {
			t.Fatal(out)
		}
	})
	t.Run("known incompatible dependency version", func(t *testing.T) {
		input, pkg, _ := installedEnvironment(t)
		addEnvironmentDependency(t, pkg, "999.0.0", false)
		out := invoke(t, binary, "DISCOVERY", "discovery.check-environment", input, true)
		if out["status"] != "INCOMPATIBLE" {
			t.Fatal(out)
		}
	})
	for _, tc := range []string{"tampered binary", "wrong origin version", "missing engine", "untrusted transport", "ambiguous registration", "escaped contract path"} {
		t.Run(tc, func(t *testing.T) {
			input, pkg, scripts := installedEnvironment(t)
			distribution := input["distribution_root"].(string)
			readFixture := func(ref string) object {
				d, err := testrepo.Open(pkg).Read(ref)
				if err != nil {
					t.Fatal(err)
				}
				return d
			}
			switch tc {
			case "tampered binary":
				filename := "ptsip-engine-context"
				if runtime.GOOS == "windows" {
					filename += ".exe"
				}
				if err := os.WriteFile(filepath.Join(scripts, filename), []byte("tampered"), 0755); err != nil {
					t.Fatal(err)
				}
			case "wrong origin version":
				d, err := testrepo.Open(distribution).Read("ptsip-engines.json")
				if err != nil {
					t.Fatal(err)
				}
				d["engines"].(object)["context"].(object)["engine_version"] = "999.0.0"
				writeJSON(t, distribution, "ptsip-engines.json", d)
			case "missing engine":
				if err := os.Remove(filepath.Join(pkg, "engines/context/manifest.json")); err != nil {
					t.Fatal(err)
				}
			case "untrusted transport":
				v := readFixture("engines/context/transport.json")
				v["version"] = "999.0.0"
				writeJSON(t, pkg, "engines/context/transport.json", v)
			case "ambiguous registration":
				v := readFixture("engines/index.json")
				es := v["official_engine_entries"].([]any)
				es[1] = es[0]
				writeJSON(t, pkg, "engines/index.json", v)
			case "escaped contract path":
				v := readFixture("engines/index.json")
				v["official_engine_entries"].([]any)[0].(object)["contract_ref"] = "src/ptsip/engines/../../escape.json"
				writeJSON(t, pkg, "engines/index.json", v)
			}
			out := invoke(t, binary, "DISCOVERY", "discovery.check-environment", input, true)
			if out["status"] != "INCOMPATIBLE" || len(out["entries"].([]any)) != 7 {
				t.Fatal(out)
			}
			if tc == "tampered binary" {
				row := environmentRow(t, out, "CONTEXT")
				if row["installation_integrity"] != "FAILED" || row["contract_agreement"] != "VERIFIED" || row["transport_agreement"] != "VERIFIED" {
					t.Fatal(row)
				}
			}
		})
	}
	t.Run("caller cannot grant authority", func(t *testing.T) {
		input, _, _ := installedEnvironment(t)
		input["execution_authorized"] = true
		invoke(t, binary, "DISCOVERY", "discovery.check-environment", input, false)
	})
}
