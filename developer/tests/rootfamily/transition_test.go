package rootfamily

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The admitted JSON contract owns expectations. Go is an independent verifier;
// Python commands below exercise the existing implementation's public boundary.
type plane struct {
	Path, PolicyClass, Prefix, ValueField, SubjectPrefix string
	SourceCount                                          int
}

func (p *plane) UnmarshalJSON(data []byte) error {
	var row struct {
		Path    string `json:"path"`
		Class   string `json:"policy_class"`
		Prefix  string `json:"prefix"`
		Field   string `json:"value_field"`
		Subject string `json:"subject_prefix"`
		Count   int    `json:"source_count"`
	}
	if err := json.Unmarshal(data, &row); err != nil {
		return err
	}
	*p = plane{row.Path, row.Class, row.Prefix, row.Field, row.Subject, row.Count}
	return nil
}

type verificationContract struct {
	ID       string   `json:"contract_id"`
	Families []string `json:"root_family_vocabulary"`
	Planes   []plane  `json:"planes"`
	Cases    []struct {
		ID    string `json:"id"`
		Error string `json:"expected_error"`
	} `json:"fail_closed_cases"`
	GoCases []struct {
		Module string `json:"module_path"`
		Error  string `json:"expected_error"`
	} `json:"go_execution_fail_closed_cases"`
	Neutral struct {
		ProgramLanguage   string `json:"program_language"`
		ModuleRef         string `json:"module_ref"`
		CreationPolicyRef string `json:"creation_policy_ref"`
		Cases             []struct {
			ID       string `json:"id"`
			Accepted bool   `json:"accepted"`
		} `json:"acceptance_cases"`
	} `json:"neutral_module_verification"`
	SmokeCases []struct {
		ID               string   `json:"id"`
		Command          []string `json:"command"`
		ExpectedMutation bool     `json:"expected_mutation"`
	} `json:"execution_smoke_cases"`
}

func repository(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err != nil {
		t.Fatal(err)
	}
	return root
}

func contract(t *testing.T) verificationContract {
	t.Helper()
	var result verificationContract
	data, err := os.ReadFile(filepath.Join(repository(t), "developer/policy/contracts/root-family-transition-verification.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != "urn:ptsip:developer:root-family-transition-verification:v1" {
		t.Fatal("unsupported contract identity")
	}
	return result
}

func mapping(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected mapping, got %T", value)
	}
	return result
}

func sequence(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("expected sequence, got %T", value)
	}
	return result
}

func read(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &result)
	} else {
		err = yaml.Unmarshal(data, &result)
	}
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return result
}

func write(t *testing.T, path string, value any) {
	t.Helper()
	var data []byte
	var err error
	if strings.HasSuffix(path, ".json") {
		data, err = json.Marshal(value)
	} else {
		data, err = yaml.Marshal(value)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func canonicalJSON(t *testing.T, value any) []byte {
	t.Helper()
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func pointer(t *testing.T, value map[string]any, path string) any {
	t.Helper()
	var node any = value
	for _, token := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		m := mapping(t, node)
		var ok bool
		node, ok = m[token]
		if !ok {
			t.Fatalf("missing source pointer %s", path)
		}
	}
	return node
}

func python(t *testing.T, args ...string) map[string]any {
	t.Helper()
	root := repository(t)
	executable := os.Getenv("PTSIP_TEST_PYTHON")
	if executable == "" {
		for _, candidate := range []string{filepath.Join(root, ".venv/Scripts/python.exe"), filepath.Join(root, ".venv/bin/python")} {
			if _, err := os.Stat(candidate); err == nil {
				executable = candidate
				break
			}
		}
	}
	if executable == "" {
		executable = "python"
	}
	command := exec.Command(executable, args...)
	command.Dir = root
	command.Env = append(os.Environ(), "PYTHONUTF8=1", "PYTHONPATH="+root+string(os.PathListSeparator)+filepath.Join(root, "src"))
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("consumer command failed: %v\n%s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("consumer result is not JSON: %v\n%s", err, out)
	}
	return result
}

func TestSourceResponsibilityAndLifecyclePreservation(t *testing.T) {
	c := contract(t)
	for _, p := range c.Planes {
		t.Run(p.PolicyClass, func(t *testing.T) {
			root := filepath.Join(repository(t), p.Path)
			index := read(t, filepath.Join(root, "index.yaml"))
			if index["migration_registry_ref"] != "registries/root-family-migration.json" {
				t.Fatal("graph is not explicitly admitted")
			}
			graph := read(t, filepath.Join(root, index["migration_registry_ref"].(string)))
			if graph["policy_class"] != p.PolicyClass {
				t.Fatal("foreign graph authority")
			}
			if len(sequence(t, graph["sources"])) != p.SourceCount {
				t.Fatal("source coverage changed")
			}
			entries := map[string]map[string]any{}
			for _, raw := range sequence(t, index["policies"]) {
				row := mapping(t, raw)
				id := row["id"].(string)
				if _, exists := entries[id]; exists {
					t.Fatalf("duplicate index identity %s", id)
				}
				entries[id] = row
			}
			used := map[string]bool{}
			for _, raw := range sequence(t, graph["sources"]) {
				source := mapping(t, raw)
				id := source["source_policy_id"].(string)
				archive := filepath.Join(root, source["archive_path"].(string))
				data, err := os.ReadFile(archive)
				if err != nil {
					t.Fatal(err)
				}
				if digest(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))) != source["source_sha256"] {
					t.Fatalf("source bytes changed: %s", id)
				}
				original := read(t, archive)
				if digest(canonicalJSON(t, original)) != source["record_sha256"] {
					t.Fatalf("source record changed: %s", id)
				}
				if original["policy_class"] != p.PolicyClass || mapping(t, original["policy"])["status"] != source["source_status"] {
					t.Fatalf("source identity/state changed: %s", id)
				}
				if _, err := os.Stat(filepath.Join(root, source["original_path"].(string))); !os.IsNotExist(err) {
					t.Fatalf("old physical authority reappeared: %s", id)
				}
				if entries[id]["authority_role"] != "MIGRATION_SOURCE" {
					t.Fatal("legacy source is current authority")
				}
				var reconstructed map[string]any
				if err := json.Unmarshal(canonicalJSON(t, source["header"]), &reconstructed); err != nil {
					t.Fatal(err)
				}
				seen := []string{}
				for _, rawUnit := range sequence(t, source["units"]) {
					unit := mapping(t, rawUnit)
					path := unit["source_pointer"].(string)
					for _, prior := range seen {
						if path == prior || strings.HasPrefix(path, prior+"/") || strings.HasPrefix(prior, path+"/") {
							t.Fatal("overlapping source pointer")
						}
					}
					seen = append(seen, path)
					pid, family, section := unit["policy_id"].(string), unit["family"].(string), unit["section"].(string)
					if unit["policy_path"] != family+"/"+pid+".yaml" || !strings.HasPrefix(pid, p.Prefix+"-"+family+"-") {
						t.Fatal("owner routing changed")
					}
					owner := read(t, filepath.Join(root, unit["policy_path"].(string)))
					policy := mapping(t, owner["policy"])
					if owner["policy_class"] != p.PolicyClass || owner["responsibility_family"] != family || policy["id"] != pid || policy["status"] != source["source_status"] {
						t.Fatal("owner identity or source lifecycle changed")
					}
					value, exists := mapping(t, owner[p.ValueField])[section]
					if !exists || !bytes.Equal(canonicalJSON(t, value), canonicalJSON(t, pointer(t, original, path))) {
						t.Fatalf("source responsibility value changed: %s%s", id, path)
					}
					key := pid + "/" + section
					if used[key] {
						t.Fatal("unit assigned more than once")
					}
					used[key] = true
					tokens := strings.Split(path[1:], "/")
					node := reconstructed
					for i, rawToken := range tokens {
						token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
						if i == len(tokens)-1 {
							node[token] = value
							break
						}
						if node[token] == nil {
							node[token] = map[string]any{}
						}
						node = mapping(t, node[token])
					}
				}
				if !bytes.Equal(canonicalJSON(t, reconstructed), canonicalJSON(t, original)) {
					t.Fatalf("incomplete reconstruction: %s", id)
				}
			}
			families := map[string]bool{}
			declared := map[string]bool{}
			for _, raw := range sequence(t, graph["materializations"]) {
				row := mapping(t, raw)
				pid := row["policy_id"].(string)
				owner := read(t, filepath.Join(root, row["path"].(string)))
				if mapping(t, owner["policy"])["status"] != row["source_status"] || entries[pid]["authority_role"] != "CANONICAL_AUTHORITY" {
					t.Fatal("materialization state/authority mismatch")
				}
				if !strings.HasPrefix(mapping(t, owner["authority_subject"])["id"].(string), p.SubjectPrefix) {
					t.Fatal("cross-plane subject authority")
				}
				values := mapping(t, owner[p.ValueField])
				kernels := sequence(t, mapping(t, values["family_definition"])["kernel_definitions"])
				ids := []any{}
				for _, k := range kernels {
					ids = append(ids, mapping(t, k)["id"])
				}
				if !reflect.DeepEqual(ids, owner["exclusive_kernel"]) {
					t.Fatal("kernel declaration mismatch")
				}
				count := 0
				for section := range values {
					if strings.HasPrefix(section, "unit_") {
						declared[pid+"/"+section] = true
						count++
					}
				}
				if float64(count) != row["unit_count"] {
					t.Fatal("declared unit count mismatch")
				}
				if row["definition_only"] == true && (count != 0 || row["source_status"] != "DRAFT") {
					t.Fatal("definition-only family promoted")
				}
				families[row["family"].(string)] = true
			}
			if !reflect.DeepEqual(used, declared) {
				t.Fatal("unit coverage mismatch")
			}
			if len(families) != len(c.Families) {
				t.Fatal("family coverage mismatch")
			}
			for _, family := range c.Families {
				if !families[family] {
					t.Fatal("missing family " + family)
				}
			}
			t.Logf("%d frozen policies, %d exact responsibility units, %d families", p.SourceCount, len(used), len(families))
		})
	}
}

func TestResolverAndSupportAuthorityBoundaries(t *testing.T) {
	c := contract(t)
	result := python(t, "-m", "developer.automation.policy_resolver", "resolve", "--scope", "developer/policy", "--operation", "MODIFY", "--json")
	for _, raw := range sequence(t, result["policies"]) {
		row := mapping(t, raw)
		pid := row["policy_id"].(string)
		found := false
		for _, family := range c.Families {
			if strings.HasPrefix(pid, "MPD-"+family+"-") {
				found = true
			}
		}
		if !found {
			t.Fatal("developer resolver selected legacy authority")
		}
	}
	legacy := python(t, "-m", "developer.automation.policy_resolver", "get", "MPD-SPEC-0006", "--section", "identity_and_resolution", "--json")
	if legacy["projection_authority"] != false || len(sequence(t, legacy["canonical_owners"])) == 0 {
		t.Fatal("legacy projection claims authority or lacks exact owners")
	}
	data := python(t, "-c", `import json
from pathlib import Path
from developer.automation.root_family_policy_entry import resolve_entry
from ptsip.governance.authority import AuthorityCatalog
root=Path.cwd(); catalog=AuthorityCatalog(root)
families=json.loads(Path('developer/policy/contracts/root-family-transition-verification.v1.json').read_text(encoding='utf-8'))['root_family_vocabulary']
rows=[]
for family in families:
 for _,route,record in catalog.resolve_family(family):
  rows.append({'family':family,'id':route['id'],'class':record['policy_class'],'subject':record['authority_subject']['id'],'role':catalog._validate_role(record)['projection_role']})
entries=[resolve_entry(cls,family,root=root) for cls in ['PTSIP_DEVELOPER_POLICY','PTSIP_SUPPORT_FEATURE'] for family in families]
print(json.dumps({'support':rows,'entries':entries}))`)
	for _, raw := range sequence(t, data["support"]) {
		row := mapping(t, raw)
		if row["class"] != "PTSIP_SUPPORT_FEATURE" || !strings.HasPrefix(row["subject"].(string), "SUP_") || row["role"] != "ROOT_FAMILY_CONTRACT_AUTHORITY" {
			t.Fatal("support inherited foreign/project authority")
		}
	}
	for _, raw := range sequence(t, data["entries"]) {
		row := mapping(t, raw)
		if row["family_state_before_materialization"] == "RESERVED" || len(sequence(t, row["registered_policies"])) == 0 {
			t.Fatal("family-aware entry remains unresolved")
		}
	}
}

func copyPolicy(t *testing.T) string {
	t.Helper()
	source := filepath.Join(repository(t), "src/policy")
	target := filepath.Join(t.TempDir(), "policy")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestActualConsumerFailsClosed(t *testing.T) {
	for _, failure := range contract(t).Cases {
		t.Run(failure.ID, func(t *testing.T) {
			root := copyPolicy(t)
			registry := filepath.Join(root, "registries/root-family-migration.json")
			graph := read(t, registry)
			var source map[string]any
			for _, raw := range sequence(t, graph["sources"]) {
				candidate := mapping(t, raw)
				if candidate["source_status"] == "ACTIVE" {
					source = candidate
					break
				}
			}
			unit := mapping(t, sequence(t, source["units"])[0])
			ownerPath := filepath.Join(root, unit["policy_path"].(string))
			switch failure.ID {
			case "registry_missing":
				if err := os.Remove(registry); err != nil {
					t.Fatal(err)
				}
			case "owner_missing":
				if err := os.Remove(ownerPath); err != nil {
					t.Fatal(err)
				}
			case "foreign_class", "source_state_changed", "unit_changed":
				owner := read(t, ownerPath)
				if failure.ID == "foreign_class" {
					owner["policy_class"] = "PTSIP_DEVELOPER_POLICY"
				}
				if failure.ID == "source_state_changed" {
					mapping(t, owner["policy"])["status"] = "DRAFT"
				}
				if failure.ID == "unit_changed" {
					mapping(t, owner["authority_semantics"])[unit["section"].(string)] = "UNAUTHORIZED_REPLACEMENT"
				}
				write(t, ownerPath, owner)
			case "duplicate_pointer":
				source["units"] = append(sequence(t, source["units"]), unit)
				write(t, registry, graph)
			case "path_escape":
				unit["policy_path"] = "../outside.yaml"
				write(t, registry, graph)
			case "archive_changed":
				path := filepath.Join(root, source["archive_path"].(string))
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(data, []byte("\n# drift\n")...), 0600); err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatalf("unimplemented admitted case %s", failure.ID)
			}
			result := python(t, "-c", `import json,sys
from pathlib import Path
from ptsip.governance.authority import validate_migration
try:
 validate_migration(Path(sys.argv[1]),'PTSIP_SUPPORT_FEATURE')
 print(json.dumps({'error':None}))
except (ValueError,OSError) as exc:
 print(json.dumps({'error':type(exc).__name__+':'+str(exc)}))`, root)
			errorText, _ := result["error"].(string)
			if !strings.Contains(errorText, failure.Error) {
				t.Fatalf("expected %s, got %v", failure.Error, result)
			}
		})
	}
}

func TestGoExecutionIsAdmittedAlongsidePythonRegression(t *testing.T) {
	result := python(t, ".github/scripts/resolve_test_modes.py", "manual", "--mode", "repository-architecture")
	plan := sequence(t, result["plan"])
	if len(plan) != 1 {
		t.Fatal("unexpected execution selection")
	}
	mode := mapping(t, plan[0])
	if !reflect.DeepEqual(mode["go"], []any{"developer/tests"}) || len(sequence(t, mode["pytest"])) == 0 {
		t.Fatal("Go execution missing or existing regression removed")
	}
	bindings := python(t, "-m", "developer.automation.policy_resolver", "resolve", "--scope", "developer/tests/rootfamily", "--operation", "VERIFY", "--json")
	owners := map[string]bool{}
	for _, raw := range sequence(t, bindings["policies"]) {
		owners[mapping(t, raw)["policy_id"].(string)] = true
	}
	for _, id := range []string{"MPD-ASSURE-0003", "MPD-REAL-0003", "MPD-NORM-0001"} {
		if !owners[id] {
			t.Fatal("missing exact verification authority " + id)
		}
	}
}

func TestGoExecutionRejectsUnownedOrEscapingModules(t *testing.T) {
	for _, failure := range contract(t).GoCases {
		t.Run(failure.Module, func(t *testing.T) {
			registry := read(t, filepath.Join(repository(t), ".github/test_modes.yaml"))
			for _, raw := range sequence(t, registry["modes"]) {
				mode := mapping(t, raw)
				if mode["id"] == "repository-architecture" {
					mapping(t, mode["execution"])["go"] = []any{failure.Module}
				}
			}
			path := filepath.Join(t.TempDir(), "registry.yaml")
			write(t, path, registry)
			result := python(t, "-c", `import json,runpy,sys
from pathlib import Path
validator=runpy.run_path('.github/scripts/validate_test_modes.py')['validate_registry']
print(json.dumps({'errors':validator(Path(sys.argv[1]),Path('developer/profiles/ptsip-repository.yaml'),Path.cwd())}))`, path)
			if !strings.Contains(string(canonicalJSON(t, result["errors"])), failure.Error) {
				t.Fatalf("expected %s, got %v", failure.Error, result)
			}
		})
	}
}
