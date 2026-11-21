package contracts_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type profileRuntime struct{ cli, scripts, packageRoot string }

var profileRuntimes = struct {
	sync.Mutex
	paths map[string]profileRuntime
}{paths: map[string]profileRuntime{}}
var profileTestTemporary string

func TestMain(m *testing.M) {
	var err error
	profileTestTemporary, err = os.MkdirTemp("", "ptsip-profile-native-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	// This directory is created exclusively for this test process.
	if relative, err := filepath.Rel(os.TempDir(), profileTestTemporary); err == nil && relative != "." && !strings.HasPrefix(relative, "..") {
		_ = os.RemoveAll(profileTestTemporary)
	}
	os.Exit(code)
}

func installedProfileRuntime(t *testing.T, kind string) profileRuntime {
	t.Helper()
	profileRuntimes.Lock()
	defer profileRuntimes.Unlock()
	if installed, ok := profileRuntimes.paths[kind]; ok {
		return installed
	}
	product := productRoot(t)
	engine := filepath.Join(product, "engines", kind)
	if _, err := os.Stat(filepath.Join(engine, "Cargo.toml")); err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	cargo, err := exec.LookPath("cargo")
	if err != nil {
		home, e := os.UserHomeDir()
		if e != nil {
			t.Fatal(e)
		}
		cargo = filepath.Join(home, ".cargo", "bin", "cargo"+suffix)
	}
	// Restored caches can carry future mtimes. Rebuild this package so native
	// evidence always reflects the current included contracts and schemas.
	buildCommand(t, cargo, "clean", "--manifest-path", filepath.Join(engine, "Cargo.toml"), "-p", "ptsip-engine-"+kind)
	buildCommand(t, cargo, "build", "--locked", "--manifest-path", filepath.Join(engine, "Cargo.toml"))
	root := filepath.Join(profileTestTemporary, kind)
	installed := profileRuntime{cli: filepath.Join(root, "scripts", "ptsip-go"+suffix), scripts: filepath.Join(root, "scripts"), packageRoot: filepath.Join(root, "package")}
	if err := os.MkdirAll(installed.scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	buildCommand(t, "go", "-C", product, "build", "-o", installed.cli, ".")
	data, err := os.ReadFile(filepath.Join(engine, "target", "debug", "ptsip-engine-"+kind+suffix))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installed.scripts, "ptsip-engine-"+kind+suffix), data, 0o755); err != nil {
		t.Fatal(err)
	}
	copyJSONTree(t, filepath.Join(product, "schemas"), filepath.Join(installed.packageRoot, "schemas"))
	copyJSONTree(t, filepath.Join(product, "engines"), filepath.Join(installed.packageRoot, "engines"))
	profileRuntimes.paths[kind] = installed
	return installed
}

func profileEnvelope(t *testing.T, kind, operation string, input object) object {
	t.Helper()
	engine := filepath.Join(productRoot(t), "engines", kind)
	c := readObject(t, filepath.Join(engine, "contract.json"))
	m := readObject(t, filepath.Join(engine, "manifest.json"))
	return object{"engine_id": m["engine_id"], "engine_version": m["implementation_version"], "contract_id": c["contract_id"], "contract_version": c["version"], "operation_id": operation, "input": input}
}

// Invoke installed copies through the real Go router, never a test-only worker.
func runProfileOperation(t *testing.T, kind, operation string, input object) (object, error) {
	t.Helper()
	installed := installedProfileRuntime(t, kind)
	request, err := json.Marshal(profileEnvelope(t, kind, operation, input))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(installed.cli, "engine", installed.packageRoot, installed.scripts, operation)
	cmd.Stdin = bytes.NewReader(request)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var response object
	if parseErr := json.Unmarshal(stdout.Bytes(), &response); parseErr != nil {
		return nil, fmt.Errorf("native response: %w (%v): %s %s", parseErr, err, stdout.String(), stderr.String())
	}
	return response, err
}

func TestDiscoveryProfileNative(t *testing.T) {
	installed := installedProfileRuntime(t, "discovery")
	for _, version := range []string{"pp.1.01", "pp.1.02", "pp.2.00", "pp.3.00"} {
		t.Run(version, func(t *testing.T) {
			response, err := runProfileOperation(t, "discovery", "discovery.profile.contract", object{"root": installed.packageRoot, "contract_version": version})
			if err != nil || response["status"] != "PASS" {
				t.Fatalf("selection failed: %v %v", response, err)
			}
			selected := response["output"].(object)
			if selected["version"] != version || selected["allowed"] != true {
				t.Fatalf("incorrect selection: %v", selected)
			}
			if version == "pp.3.00" {
				if selected["lifecycle"] != "ACTIVE" || len(selected["notifications"].(array)) != 0 {
					t.Fatal(selected)
				}
			} else {
				if selected["lifecycle"] != "DEPRECATED" || len(selected["notifications"].(array)) != 1 || selected["notifications"].(array)[0].(object)["mandatory"] != true {
					t.Fatal(selected)
				}
			}
			identity, e := runProfileOperation(t, "validation", "validation.profile.identity", object{"version": version, "operation": "VALIDATE", "selected_contract": selected})
			if e != nil || identity["status"] != "PASS" || identity["output"].(object)["status"] != "SUPPORTED" {
				t.Fatalf("native handoff failed: %v %v", identity, e)
			}
		})
	}
	for _, input := range []object{{"root": installed.packageRoot, "contract_version": "pp.1.00"}, {"root": installed.packageRoot, "source_revision": strings.Repeat("0", 64)}, {"root": installed.packageRoot, "registry_path": "../schemas/config.json"}} {
		response, err := runProfileOperation(t, "discovery", "discovery.profile.contract", input)
		if err == nil || response["status"] != "FAIL" {
			t.Fatalf("invalid selection accepted: %v %v", response, err)
		}
	}
	response, err := runProfileOperation(t, "discovery", "discovery.profile.contract", object{"root": installed.packageRoot})
	if err != nil || response["output"].(object)["version"] != "pp.3.00" {
		t.Fatalf("current selector failed: %v %v", response, err)
	}
	root := t.TempDir()
	write := func(name string, value any) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(value)
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	response, err = runProfileOperation(t, "discovery", "discovery.profile.catalog", object{"repository_root": root})
	if err != nil || response["output"].(object)["catalog_valid"] != false {
		t.Fatalf("missing catalog result: %v %v", response, err)
	}
	catalog := object{"schema_version": "ptsip-local-profile-catalog/v1", "default_profile": "main", "profiles": array{object{"id": "main", "resource": "main.ptsip.yaml"}}}
	write(".ptsip/profiles/index.json", catalog)
	write(".ptsip/profiles/main.ptsip.yaml", object{"ptsip": object{"version": "pp.2.00"}})
	response, err = runProfileOperation(t, "discovery", "discovery.profile.catalog", object{"repository_root": root})
	if err != nil || response["output"].(object)["selected_id"] != "main" || response["output"].(object)["catalog_valid"] != true {
		t.Fatalf("catalog failed: %v %v", response, err)
	}
	catalog["profiles"] = append(catalog["profiles"].(array), object{"id": "main", "resource": "other.ptsip.yaml"})
	write(".ptsip/profiles/index.json", catalog)
	response, err = runProfileOperation(t, "discovery", "discovery.profile.catalog", object{"repository_root": root})
	if err == nil || response["status"] != "FAIL" {
		t.Fatalf("ambiguous catalog accepted: %v %v", response, err)
	}
	records := array{object{"ID": "child with spaces", "parent_id": "ROOT", "rule_value": array{1, "opaque"}}, object{"ID": "ROOT", "rule_value": object{"root_id": "child with spaces"}}}
	response, err = runProfileOperation(t, "discovery", "discovery.profile.rules", object{"records": records, "rule_id": "child with spaces"})
	if err != nil || response["output"].(object)["status"] != "RESOLVED" || response["output"].(object)["validation_verdict"] != "NOT_ASSERTED" {
		t.Fatalf("opaque exact resolution failed: %v %v", response, err)
	}
	response, err = runProfileOperation(t, "discovery", "discovery.profile.rules", object{"records": records, "rule_id": "child"})
	if err != nil || response["output"].(object)["status"] != "UNRESOLVED" {
		t.Fatalf("prefix matching was used: %v %v", response, err)
	}
}

func TestProfileBackgroundSessions(t *testing.T) {
	cases := []struct {
		kind, operation string
		input           object
	}{
		{"discovery", "discovery.profile.rules", object{"records": array{object{"ID": "arbitrary-id"}}}},
		{"validation", "validation.profile.rules", object{"records": array{object{"ID": "arbitrary-id"}}}},
		{"projection", "projection.profile.metadata", object{"profile_contract_version": "pp.2.00", "specification_source": "https://github.com/Kinirin/PTSIP", "specification_revision": strings.Repeat("1", 40), "schema_ref": "urn:selected:profile"}},
	}
	for _, item := range cases {
		t.Run(item.kind, func(t *testing.T) {
			installed := installedProfileRuntime(t, item.kind)
			cmd := exec.Command(installed.cli, "engine-serve", installed.packageRoot, installed.scripts, item.operation)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := false
			defer func() {
				if !done {
					_ = stdin.Close()
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			reader := bufio.NewReader(stdout)
			request := profileEnvelope(t, item.kind, item.operation, item.input)
			for index := 0; index < 3; index++ {
				if index == 1 {
					request["input"] = object{"undeclared": true}
				} else {
					request["input"] = item.input
				}
				raw, _ := json.Marshal(request)
				if _, err := stdin.Write(append(raw, '\n')); err != nil {
					t.Fatal(err)
				}
				result := make(chan []byte, 1)
				go func() { line, _ := reader.ReadBytes('\n'); result <- line }()
				var line []byte
				select {
				case line = <-result:
				case <-time.After(20 * time.Second):
					t.Fatal("session did not respond before EOF")
				}
				var response object
				if err := json.Unmarshal(line, &response); err != nil {
					t.Fatalf("response %q: %v %s", line, err, stderr.String())
				}
				expected := "PASS"
				if index == 1 {
					expected = "FAIL"
				}
				if response["status"] != expected {
					t.Fatalf("request %d: %v", index, response)
				}
				if index == 0 {
					time.Sleep(50 * time.Millisecond)
				}
			}
			_ = stdin.Close()
			if err := cmd.Wait(); err != nil {
				t.Fatalf("EOF shutdown: %v %s", err, stderr.String())
			}
			done = true
		})
	}
}

func TestProfileOperationRegistrationAndSourceClosure(t *testing.T) {
	root := repositoryRoot(t)
	index := readObject(t, filepath.Join(productRoot(t), "engines", "index.json"))
	wanted := map[string]string{
		"discovery.profile.contract":          "DISCOVERY",
		"discovery.profile.catalog":           "DISCOVERY",
		"discovery.profile.rules":             "DISCOVERY",
		"validation.profile.identity":         "VALIDATION",
		"validation.profile.transition":       "VALIDATION",
		"validation.profile.rules":            "VALIDATION",
		"projection.profile.metadata":         "PROJECTION",
		"projection.migration.source":         "PROJECTION",
		"projection.migration.coverage":       "PROJECTION",
		"projection.migration.compatibility":  "PROJECTION",
		"validation.profile.identity-rewrite": "VALIDATION",
		"context.repository.snapshot":         "CONTEXT",
		"discovery.repository.inspect":        "DISCOVERY",
		"discovery.profile.candidates":        "DISCOVERY",
		"validation.repository.snapshot":      "VALIDATION",
		"validation.profile.path":             "VALIDATION",
		"validation.profile.transition-state": "VALIDATION",
		"validation.profile.convergence":      "VALIDATION",
	}
	seen := map[string]string{}
	for _, raw := range index["official_engine_entries"].(array) {
		entry := raw.(object)
		kind := entry["kind"].(string)
		for _, id := range entry["operations"].(array) {
			operation := id.(string)
			if prior, exists := seen[operation]; exists {
				t.Fatalf("ambiguous operation %s: %s/%s", operation, prior, kind)
			}
			seen[operation] = kind
		}
		contract := readObject(t, filepath.Join(root, filepath.FromSlash(entry["contract_ref"].(string))))
		for _, raw := range contract["operations"].(array) {
			op := raw.(object)
			id := op["operation_id"].(string)
			owner, selected := wanted[id]
			if !selected {
				continue
			}
			if owner != kind {
				t.Fatalf("%s owner = %s", id, kind)
			}
			evidence := op["verification_evidence"].(object)
			ref := evidence["input_schema_ref"].(object)
			path, fragment, ok := strings.Cut(ref["path"].(string), "#/$defs/")
			if !ok {
				t.Fatal(ref)
			}
			bytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(bytes)
			if hex.EncodeToString(digest[:]) != ref["sha256"] {
				t.Fatalf("stale %s native schema digest", id)
			}
			schema := readObject(t, filepath.Join(root, filepath.FromSlash(path)))
			if schema["$id"] != ref["id"] {
				t.Fatalf("%s native schema identity mismatch", id)
			}
			defs := schema["$defs"].(object)
			if !reflect.DeepEqual(expandNativeDefinition(t, defs, defs[fragment]), op["typed_input"]) {
				t.Fatalf("%s input differs from its registered engine-owned definition", id)
			}
			outPath, outDef, ok := strings.Cut(evidence["output_schema_ref"].(string), "#/$defs/")
			if !ok || outPath != path {
				t.Fatal(evidence)
			}
			if !reflect.DeepEqual(expandNativeDefinition(t, defs, defs[outDef]), op["typed_output"]) {
				t.Fatalf("%s output differs from its registered engine-owned definition", id)
			}
		}
	}
	for id, kind := range wanted {
		if seen[id] != kind {
			t.Fatalf("%s has no exact %s registration", id, kind)
		}
	}
	snapshot := readObject(t, filepath.Join(productRoot(t), "engines", "contracts", "engines_registries", "derived-snapshot.json"))
	for _, raw := range snapshot["source_refs"].(array) {
		ref := raw.(object)
		bytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref["path"].(string))))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(bytes)
		if hex.EncodeToString(digest[:]) != ref["sha256"] {
			t.Fatalf("stale native source pin: %s", ref["path"])
		}
	}
	canonical, err := json.Marshal(snapshot["source_refs"])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	if hex.EncodeToString(digest[:]) != snapshot["snapshot_revision"] {
		t.Fatal("snapshot revision does not identify its current registered source references")
	}
}

func expandNativeDefinition(t *testing.T, defs object, value any) any {
	t.Helper()
	switch value := value.(type) {
	case object:
		if raw, ok := value["$ref"].(string); ok {
			name, local := strings.CutPrefix(raw, "#/$defs/")
			definition, exists := defs[name]
			if !local || !exists || len(value) != 1 {
				t.Fatalf("invalid native definition reference %v", value)
			}
			return expandNativeDefinition(t, defs, definition)
		}
		expanded := make(object, len(value))
		for name, child := range value {
			expanded[name] = expandNativeDefinition(t, defs, child)
		}
		return expanded
	case array:
		expanded := make(array, len(value))
		for index, child := range value {
			expanded[index] = expandNativeDefinition(t, defs, child)
		}
		return expanded
	default:
		return value
	}
}
