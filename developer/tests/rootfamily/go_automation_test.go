package rootfamily

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// These independent vectors were captured from the Python implementation before
// retirement. They are protocol regression data and never policy authority.
func frozenAutomationVectors(t *testing.T, key string) map[string]any {
	t.Helper()
	path := filepath.Join(repository(t), "developer/tests/rootfamily/testdata/go_automation_protocol_vectors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture["projection_authority"] != false || fixture["fixture_role"] != "FROZEN_PRE_RETIREMENT_PROTOCOL_EQUIVALENCE" {
		t.Fatal("equivalence vectors must remain non-authoritative")
	}
	result := mapping(t, fixture[key])
	if key == "entry" {
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		encoded = []byte(strings.ReplaceAll(string(encoded), "developer/profiles/ptsip-repository.yaml", ".ptsip/profiles/main.ptsip.yaml"))
		if err := json.Unmarshal(encoded, &result); err != nil {
			t.Fatal(err)
		}
	}
	if key == "resolver" {
		families := mapping(t, result["families"])
		for _, raw := range families {
			row := mapping(t, raw)
			policies, ok := row["registered_policies"].([]any)
			if !ok {
				continue
			}
			for _, item := range policies {
				policy := mapping(t, item)
				if policy["policy_id"] == "MPD-INFO-0003" || policy["policy_id"] == "MPD-CHANGE-0003" {
					policy["status"] = "APPROVED"
				}
			}
		}
	}
	return result
}

func buildAutomation(t *testing.T) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "ptsip-dev")
	if os.PathSeparator == '\\' {
		name += ".exe"
	}
	command := exec.Command("go", "-C", filepath.Join(repository(t), "developer/automation"), "build", "-o", name, "./cmd/ptsip-dev")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Go automation build: %v\n%s", err, output)
	}
	return name
}

func automation(t *testing.T, binary, root string, args ...string) (map[string]any, error, string) {
	t.Helper()
	command := exec.Command(binary, append([]string{"--repository", root}, args...)...)
	// The compiled runtime must execute without Python or any executable search path.
	for _, value := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(value, "=", 2)[0])
		if key != "PATH" && key != "PYTHONPATH" && key != "PYTHONHOME" {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "PATH=", "PYTHONPATH=", "PYTHONHOME=")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, err, string(output)
	}
	var value map[string]any
	if err := json.Unmarshal(output, &value); err != nil {
		t.Fatalf("Go automation JSON: %v\n%s", err, output)
	}
	return value, nil, string(output)
}

func copyDeveloperWithoutLegacy(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "pyproject.toml"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, plane := range []string{"developer/policy", "src/policy", "registry"} {
		origin := filepath.Join(repository(t), plane)
		err := filepath.WalkDir(origin, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && entry.Name() == "legacy" {
				return filepath.SkipDir
			}
			relative, err := filepath.Rel(repository(t), name)
			if err != nil {
				return err
			}
			destination := filepath.Join(target, relative)
			if entry.IsDir() {
				return os.MkdirAll(destination, 0700)
			}
			data, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			return os.WriteFile(destination, data, 0600)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return target
}

func TestGoDirectRootResolverParityAndNoPythonDependency(t *testing.T) {
	binary := buildAutomation(t)
	root := repository(t)
	expected := frozenAutomationVectors(t, "resolver")
	for key, want := range mapping(t, expected["cases"]) {
		t.Run(key, func(t *testing.T) {
			parts := strings.Split(key, "|")

			got, err, output := automation(t, binary, root, "policy-resolver", "resolve", "--scope", parts[0], "--operation", parts[1])
			if err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Go/Python contract parity mismatch\nGo: %#v\nPython: %#v", got, want)
			}
		})
	}
	for key, want := range mapping(t, expected["families"]) {
		t.Run(key, func(t *testing.T) {
			parts := strings.Split(key, "|")
			got, err, output := automation(t, binary, root, "root-family-entry", "resolve", "--policy-class", parts[0], "--family", parts[1])
			if err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Root Family identity parity mismatch: %s\nGo: %#v\nPython: %#v", key, got, want)
			}
		})
	}
	for _, test := range []struct {
		Name string
		Args []string
	}{
		{"get", []string{"policy-resolver", "get", "MPD-REAL-0005", "--section", "go_automation_implementation"}},
		{"rule", []string{"policy-resolver", "rule", "PTSIP-CLS-001"}},
	} {
		got, err, output := automation(t, binary, root, test.Args...)
		if err != nil {
			t.Fatalf("%v\n%s", err, output)
		}
		if !reflect.DeepEqual(got, expected[test.Name]) {
			t.Fatalf("%s protocol parity mismatch", test.Name)
		}
	}
}

func TestGoResolverNeedsNoLegacyFilesAndRejectsLegacyIdentity(t *testing.T) {
	binary := buildAutomation(t)
	root := copyDeveloperWithoutLegacy(t)
	for _, args := range [][]string{
		{"policy-resolver", "validate"},
		{"policy-resolver", "get", "MPD-CNTR-0004", "--section", "direct_root_automation_contract"},
		{"policy-resolver", "resolve", "--scope", "developer/automation", "--operation", "MODIFY"},
		{"root-family-entry", "resolve", "--policy-class", "PTSIP_DEVELOPER_POLICY", "--family", "CNTR"},
	} {
		if _, err, output := automation(t, binary, root, args...); err != nil {
			t.Fatalf("legacy-free Go resolver: %v\n%s", err, output)
		}
	}
	for _, id := range []string{"MPD-0010", "MPD-0015", "MPD-SPEC-0021"} {
		if _, err, output := automation(t, binary, root, "policy-resolver", "get", id); err == nil || !strings.Contains(output, "audit-only") {
			t.Fatalf("legacy identity accepted: %s\n%s", id, output)
		}
	}
}

func TestGoResolverRejectsTamperedAuthorityAndRouting(t *testing.T) {
	binary := buildAutomation(t)
	tests := []struct {
		Name, Path string
		Mutate     func(map[string]any)
	}{
		{"owner_inactive", "developer/policy/CNTR/MPD-CNTR-0004.yaml", func(v map[string]any) { mapping(t, v["policy"])["status"] = "DRAFT" }},
		{"owner_wrong_contract", "developer/policy/CNTR/MPD-CNTR-0004.yaml", func(v map[string]any) {
			mapping(t, mapping(t, v["rules"])["direct_root_automation_contract"])["resolver_contract_ref"] = "src/policy/index.yaml"
		}},
		{"foreign_policy_class", "developer/policy/REAL/MPD-REAL-0005.yaml", func(v map[string]any) { v["policy_class"] = "PTSIP_SUPPORT_FEATURE" }},
		{"changed_family", "developer/policy/REAL/MPD-REAL-0005.yaml", func(v map[string]any) { v["responsibility_family"] = "NORM" }},
		{"invented_operation", "developer/policy/policy-resolver-bindings/registry.yaml", func(v map[string]any) {
			v["operation_vocabulary"] = append(sequence(t, v["operation_vocabulary"]), "MIGRATE")
		}},
		{"unregistered_context", "developer/policy/contracts/go-policy-resolver.v1.yaml", func(v map[string]any) { mapping(t, v["resolver"])["task_context_ref"] = "AGENTS.md" }},
		{"narrative_fallback", "developer/policy/contracts/go-policy-resolver.v1.yaml", func(v map[string]any) { mapping(t, v["lookup"])["natural_language_fallback"] = "ALLOWED" }},
		{"escaped_binding_path", "developer/policy/policy-resolver-bindings/registry.yaml", func(v map[string]any) { v["binding_records_ref"] = "../outside.jsonl" }},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			root := copyDeveloperWithoutLegacy(t)
			target := filepath.Join(root, test.Path)
			value := read(t, target)
			test.Mutate(value)
			write(t, target, value)
			if _, err, output := automation(t, binary, root, "policy-resolver", "validate"); err == nil {
				t.Fatalf("tampered authority accepted\n%s", output)
			}
		})
	}
	root := repository(t)
	for _, args := range [][]string{
		{"policy-resolver", "resolve", "--scope", "../outside", "--operation", "MODIFY"},
		{"policy-resolver", "resolve", "--scope", "developer", "--operation", "MIGRATE"},
		{"root-family-entry", "resolve", "--policy-class", "PTSIP_DEVELOPER_POLICY", "--family", "SPEC"},
		{"root-family-entry", "resolve", "--policy-class", "UNKNOWN", "--family", "NORM"},
		{"policy-resolver", "get", "MPD-REAL-0005", "--section", "guessed_section"},
	} {
		if _, err, output := automation(t, binary, root, args...); err == nil {
			t.Fatalf("unregistered request accepted\n%s", output)
		}
	}
}

func TestGoCutoverReportsRemainingWorkWithoutClaimingCompletion(t *testing.T) {
	binary := buildAutomation(t)
	got, err, output := automation(t, binary, repository(t), "automation-migration", "inspect")
	if err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	if got["implementation_complete"] != false || got["legacy_removal_ready"] != false {
		t.Fatal("partial implementation claimed complete")
	}
	root := repository(t)
	raw, readErr := os.ReadFile(filepath.Join(root, "developer/policy/registries/go-automation-migration.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	var inventory map[string]any
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{}
	for _, rawModule := range sequence(t, inventory["modules"]) {
		module := rawModule.(map[string]any)
		path := module["python_path"].(string)
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err == nil && info.Mode().IsRegular() {
			expected[path] = true
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	remaining := sequence(t, got["remaining_python_files"])
	expectedStatus := "IN_PROGRESS"
	if len(expected) == 0 && len(sequence(t, got["pending_modules"])) == 0 {
		expectedStatus = "GO_SOURCE_CUTOVER_READY_FOR_FINAL_VERIFICATION"
	}
	if got["status"] != expectedStatus {
		t.Fatal("source readiness does not reflect the registered remaining work", got)
	}
	if len(remaining) != len(expected) || len(sequence(t, got["go_implemented_modules"])) == 0 {
		t.Fatal("cutover projection differs from registered physical sources")
	}
	for _, rawPath := range remaining {
		if !expected[rawPath.(string)] {
			t.Fatalf("unregistered or retired Python source reported: %s", rawPath)
		}
	}
}

func TestGoStateAndPlanningExactEntryParity(t *testing.T) {
	binary := buildAutomation(t)
	expected := frozenAutomationVectors(t, "entry")
	for category, values := range expected {
		for identity, raw := range mapping(t, values) {
			t.Run(category+"/"+identity, func(t *testing.T) {
				want := mapping(t, raw)
				args := []string{"repository-state", "resolve", "--domain", identity}
				if category == "planning" {
					args = []string{"planning-entry", "resolve", "--branch", identity}
				}
				got, err, output := automation(t, binary, repository(t), args...)
				if (err == nil) != (want["accepted"] == true) {
					t.Fatalf("entry acceptance differs: %v\n%s", err, output)
				}
				if err == nil && !reflect.DeepEqual(got, want["record"]) {
					t.Fatalf("state/planning contract differs: %#v versus %#v", got, want["record"])
				}
			})
		}
	}
}

func TestGoRootIdentityInspectionAndUnregisteredAllocationGuard(t *testing.T) {
	binary := buildAutomation(t)
	expected := frozenAutomationVectors(t, "inspection")
	for id, want := range expected {
		got, err, output := automation(t, binary, repository(t), "root-family-entry", "inspect", id)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Root identity inspection mismatch: %s, %v\n%s", id, err, output)
		}
	}
	root := copyDeveloperWithoutLegacy(t)
	source := read(t, filepath.Join(root, "developer/policy/CNTR/MPD-CNTR-0004.yaml"))
	mapping(t, source["policy"])["id"] = "MPD-CNTR-9999"
	write(t, filepath.Join(root, "developer/policy/CNTR/MPD-CNTR-9999.yaml"), source)
	if _, err, output := automation(t, binary, root, "root-family-entry", "resolve", "--policy-class", "PTSIP_DEVELOPER_POLICY", "--family", "CNTR"); err == nil || !strings.Contains(output, "unregistered Root") {
		t.Fatalf("allocation ignored unregistered Root policy: %v\n%s", err, output)
	}
	for _, id := range []string{"MPD-0010", "MPD-SPEC-0021", "MPD-CNTR-1", "SFP-FAKE-0001"} {
		if _, err, _ := automation(t, binary, repository(t), "root-family-entry", "inspect", id); err == nil {
			t.Fatalf("invalid Root identity accepted: %s", id)
		}
	}
}
