package contracts_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

var registeredOperations = []string{
	"validation.policy.identity", "validation.policy.rules", "validation.rbac.ra",
	"validation.rbac.rac", "validation.rbac.aec", "validation.rbac.references",
}

type installedRuntime struct {
	packageRoot string
	scripts     string
	cli         string
	engine      string
}

func copyJSONTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "target" {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func buildCommand(t *testing.T, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func installRuntime(t *testing.T) installedRuntime {
	t.Helper()
	root := t.TempDir()
	installed := installedRuntime{packageRoot: filepath.Join(root, "package"), scripts: filepath.Join(root, "scripts")}
	if err := os.Mkdir(installed.scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	installed.cli = filepath.Join(installed.scripts, "ptsip-go"+suffix)
	installed.engine = filepath.Join(installed.scripts, "ptsip-engine-validation"+suffix)
	product := productRoot(t)
	buildCommand(t, "go", "-C", product, "build", "-o", installed.cli, ".")
	cargo, err := exec.LookPath("cargo")
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			t.Fatal(homeErr)
		}
		cargo = filepath.Join(home, ".cargo", "bin", "cargo"+suffix)
		if _, err := os.Stat(cargo); err != nil {
			t.Fatal("Rust toolchain required for native operation conformance")
		}
	}
	engine := filepath.Join(product, "engines", "validation")
	buildCommand(t, cargo, "build", "--locked", "--manifest-path", filepath.Join(engine, "Cargo.toml"))
	data, err := os.ReadFile(filepath.Join(engine, "target", "debug", "ptsip-engine-validation"+suffix))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed.engine, data, 0o755); err != nil {
		t.Fatal(err)
	}
	copyJSONTree(t, filepath.Join(product, "schemas"), filepath.Join(installed.packageRoot, "schemas"))
	copyJSONTree(t, filepath.Join(product, "engines"), filepath.Join(installed.packageRoot, "engines"))
	return installed
}

func nativeRequest(t *testing.T, operation string, input object) object {
	t.Helper()
	engine := filepath.Join(productRoot(t), "engines", "validation")
	contract := readObject(t, filepath.Join(engine, "contract.json"))
	manifest := readObject(t, filepath.Join(engine, "manifest.json"))
	return object{
		"engine_id": manifest["engine_id"], "engine_version": manifest["implementation_version"],
		"contract_id": contract["contract_id"], "contract_version": contract["version"],
		"operation_id": operation, "input": input,
	}
}

func runJSON(t *testing.T, executable string, args []string, request any, directory string, extraEnv ...string) (object, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), extraEnv...)
	if request != nil {
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stdin = bytes.NewReader(data)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if child, ok := err.(*exec.ExitError); ok {
			code = child.ExitCode()
		} else {
			t.Fatalf("could not run command: %v", err)
		}
	}
	var response object
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("native output is not a single JSON object: %v\n%s\n%s", err, stdout.String(), stderr.String())
		}
	}
	return response, stderr.String(), code
}

func invoke(t *testing.T, installed installedRuntime, operation string, input object, extraEnv ...string) (object, string, int) {
	t.Helper()
	return runJSON(t, installed.cli, []string{"engine", installed.packageRoot, installed.scripts, operation}, nativeRequest(t, operation, input), "", extraEnv...)
}

func operationInput(t *testing.T, operation string) object {
	t.Helper()
	documents := rbacDocuments()
	policy := rulePolicy()
	policy["Policy_Identity"] = object{"id": "policy", "title": "Policy", "SchemaVersion": "SCH.1.00", "Version": "v1.2.3"}
	documents["policy"] = policy
	if strings.HasPrefix(operation, "validation.policy.") {
		return object{"policy": policy}
	}
	if operation == "validation.rbac.references" {
		return documents
	}
	kind := operation[strings.LastIndex(operation, ".")+1:]
	return object{"document": documents[kind]}
}

func outputOf(t *testing.T, response object, valid bool) object {
	t.Helper()
	if response["status"] != "PASS" || response["execution_authorized"] != true || response["normative_authority"] != false {
		t.Fatalf("unexpected successful result envelope: %v", response)
	}
	output, ok := response["output"].(object)
	if !ok || output["valid"] != valid || output["execution_authorized"] != true || output["ptsip_conformance_verdict"] != "NOT_ASSERTED" {
		t.Fatalf("unexpected operation result: %v", response)
	}
	for _, field := range []string{"diagnostics", "user_contracts", "ra_usage"} {
		if _, ok := output[field].(array); !ok {
			t.Fatalf("result %s must be a JSON array: %v", field, output)
		}
	}
	return output
}

func TestRegisteredOperationsThroughGoRustAndSameGoWorker(t *testing.T) {
	installed := installRuntime(t)
	for _, operation := range registeredOperations {
		t.Run(operation, func(t *testing.T) {
			response, stderr, code := invoke(t, installed, operation, operationInput(t, operation))
			if code != 0 {
				t.Fatalf("native invocation failed %d: %s, %v", code, stderr, response)
			}
			outputOf(t, response, true)
			input := operationInput(t, operation)
			input["unexpected"] = true
			response, _, code = invoke(t, installed, operation, input)
			if code != 2 || response["status"] != "FAIL" || response["output"] != nil || response["execution_authorized"] != false {
				t.Fatalf("typed input was accepted: %v", response)
			}
			input = operationInput(t, operation)
			switch operation {
			case "validation.policy.identity":
				input["policy"].(object)["Policy_Identity"].(object)["Version"] = "not-a-version"
			case "validation.policy.rules":
				input["policy"].(object)["Policy_Content"] = object{"items": array{}}
			case "validation.rbac.references":
				input["rac"].(object)["RAC_list"].(array)[0].(object)["RAC_name"] = "Different canonical name"
			default:
				input["document"].(object)["field_outside_contract"] = true
			}
			response, stderr, code = invoke(t, installed, operation, input)
			if code != 0 {
				t.Fatalf("owned data diagnostic became an adapter failure: %s, %v", stderr, response)
			}
			output := outputOf(t, response, false)
			if len(output["diagnostics"].(array)) == 0 {
				t.Fatal("invalid owned data has no diagnostic")
			}
		})
	}
	t.Run("old-operation-authorization", func(t *testing.T) {
		response, stderr, code := invoke(t, installed, "validation.schema", object{"schema": object{"type": "integer"}, "instance": "bad"})
		if code != 0 || response["status"] != "PASS" || response["execution_authorized"] != false {
			t.Fatalf("existing operation authorization changed: %s, %v", stderr, response)
		}
	})
	t.Run("scoped-subdomains", func(t *testing.T) {
		input := operationInput(t, "validation.policy.rules")
		policy := input["policy"].(object)
		policy["Policy_Identity"] = object{"Version": "invalid"}
		policy["Policy_Lifecycle"] = object{"future_field": true}
		response, _, code := invoke(t, installed, "validation.policy.rules", input)
		if code != 0 {
			t.Fatal(response)
		}
		outputOf(t, response, true)
	})
	t.Run("bootstrap-and-preserve", func(t *testing.T) {
		project := t.TempDir()
		input := operationInput(t, "validation.policy.rules")
		input["project_root"] = project
		response, _, code := invoke(t, installed, "validation.policy.rules", input)
		if code != 0 {
			t.Fatal(response)
		}
		outputOf(t, response, true)
		if _, err := os.Stat(contractRoot(project)); !os.IsNotExist(err) {
			t.Fatal("read-only invocation created user contracts")
		}
		input["bootstrap_user_contracts"] = true
		response, _, code = invoke(t, installed, "validation.policy.rules", input)
		if code != 0 {
			t.Fatal(response)
		}
		outputOf(t, response, true)
		entries, err := os.ReadDir(contractRoot(project))
		if err != nil || len(entries) != 3 {
			t.Fatalf("expected three requested templates: %v, %v", entries, err)
		}
		path := filepath.Join(contractRoot(project), "evaluation_state.json")
		authored := []byte(`{"const":"different"}`)
		if err := os.WriteFile(path, authored, 0o644); err != nil {
			t.Fatal(err)
		}
		response, _, code = invoke(t, installed, "validation.policy.rules", input)
		if code != 0 {
			t.Fatal(response)
		}
		outputOf(t, response, false)
		actual, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(actual, authored) {
			t.Fatal("authored contract was overwritten")
		}
	})
	t.Run("bootstrap-shared-version-settings-before-first-version", func(t *testing.T) {
		project := t.TempDir()
		input := object{
			"policy":       object{"Policy_Identity": object{"title": "New policy"}},
			"project_root": project, "bootstrap_user_contracts": true,
		}
		response, stderr, code := invoke(t, installed, "validation.policy.identity", input)
		if code != 0 {
			t.Fatalf("explicit version settings bootstrap failed: %s, %v", stderr, response)
		}
		outputOf(t, response, true)
		settings := readObject(t, filepath.Join(contractRoot(project), "Version_Contract_Structure.json"))
		for _, section := range []string{"policy", "plan"} {
			if value, present := settings[section]; !present || value != nil {
				t.Fatalf("expected an unconfigured %s section: %v", section, settings)
			}
		}
	})
	t.Run("explicit-same-go-cli-binding", func(t *testing.T) {
		response, stderr, code := invoke(t, installed, registeredOperations[0], operationInput(t, registeredOperations[0]),
			"PTSIP_GO_CLI="+filepath.Join(t.TempDir(), "missing-cli"), "PTSIP_USER_CONTRACT_ROOT="+t.TempDir(),
			"PYTHONPATH="+t.TempDir(), "PYTHONHOME="+t.TempDir())
		if code != 0 {
			t.Fatalf("explicit CLI binding inherited ambient values: %s, %v", stderr, response)
		}
		outputOf(t, response, true)
	})
	t.Run("stale-manifest-contract-pin", func(t *testing.T) {
		root := t.TempDir()
		copyJSONTree(t, installed.packageRoot, root)
		path := filepath.Join(root, "engines", "validation", "contract.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		modified := installed
		modified.packageRoot = root
		response, stderr, code := invoke(t, modified, registeredOperations[0], operationInput(t, registeredOperations[0]))
		if code != 2 || response != nil || !strings.Contains(stderr, "manifest") {
			t.Fatalf("stale contract bytes were executed: %s, %v", stderr, response)
		}
	})
	t.Run("namespace-and-uniqueness", func(t *testing.T) {
		for _, duplicate := range []bool{false, true} {
			root := t.TempDir()
			index := readObject(t, filepath.Join(productRoot(t), "engines", "index.json"))
			entries := index["official_engine_entries"].(array)
			for _, raw := range entries {
				entry := raw.(object)
				if entry["kind"] == "VALIDATION" {
					if duplicate {
						entry["operations"] = append(entry["operations"].(array), registeredOperations[0])
					} else {
						entry["kind"] = "STORAGE"
					}
				}
			}
			data, _ := json.Marshal(index)
			if err := os.MkdirAll(filepath.Join(root, "engines"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "engines", "index.json"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			response, stderr, code := runJSON(t, installed.cli, []string{"operations", root}, nil, "")
			if code != 2 || response != nil || !strings.Contains(stderr, "operation") {
				t.Fatalf("invalid owner index was accepted: %s, %v", stderr, response)
			}
		}
	})
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func TestRegisteredOperationAndSourcePinClosure(t *testing.T) {
	product := productRoot(t)
	engine := filepath.Join(product, "engines", "validation")
	index := readObject(t, filepath.Join(product, "engines", "index.json"))
	owners := readObject(t, filepath.Join(product, "engines", "contracts", "capability-owner-map.json"))
	contract := readObject(t, filepath.Join(engine, "contract.json"))
	manifest := readObject(t, filepath.Join(engine, "manifest.json"))
	var declared, indexed, mapped []string
	for _, row := range contract["operations"].(array) {
		declared = append(declared, row.(object)["operation_id"].(string))
	}
	for _, row := range index["official_engine_entries"].(array) {
		entry := row.(object)
		for _, id := range entry["operations"].(array) {
			if strings.HasPrefix(id.(string), "validation.") && entry["kind"] != "VALIDATION" {
				t.Fatal("validation namespace has a non-Validation owner")
			}
		}
		if entry["kind"] == "VALIDATION" {
			for _, id := range entry["operations"].(array) {
				indexed = append(indexed, id.(string))
			}
		}
	}
	for _, row := range owners["entries"].(array) {
		entry := row.(object)
		if entry["semantic_owner"] == "VALIDATION" {
			for _, id := range entry["operation_ids"].(array) {
				mapped = append(mapped, id.(string))
			}
		}
	}
	if !reflect.DeepEqual(declared, indexed) || !reflect.DeepEqual(declared, mapped) {
		t.Fatalf("registered operation sets differ: %v, %v, %v", declared, indexed, mapped)
	}
	seen := map[string]bool{}
	for _, operation := range declared {
		if seen[operation] {
			t.Fatalf("duplicate operation %s", operation)
		}
		seen[operation] = true
	}
	for _, operation := range registeredOperations {
		if !seen[operation] {
			t.Errorf("approved operation missing: %s", operation)
		}
	}
	capability := manifest["capabilities"].(array)[0].(object)
	if capability["sha256"] != fileHash(t, filepath.Join(engine, "contract.json")) {
		t.Fatal("manifest capability hash differs from current contract bytes")
	}
	for _, location := range []struct{ path, field string }{
		{"engines/contracts/engines_registries/derived-snapshot.json", "source_refs"},
		{"engines/contracts/dependency-graph.json", "nodes"},
	} {
		for _, row := range readObject(t, filepath.Join(product, filepath.FromSlash(location.path)))[location.field].(array) {
			ref := row.(object)
			if ref["sha256"] != fileHash(t, filepath.Join(repositoryRoot(t), filepath.FromSlash(ref["path"].(string)))) {
				t.Errorf("stale source hash %s", ref["path"])
			}
		}
	}
	transport := filepath.Join(product, "engines", "contracts", "stdio-transport.json")
	transportHash := fileHash(t, transport)
	for _, name := range []string{"context", "storage", "validation", "discovery", "execution", "projection"} {
		binding := readObject(t, filepath.Join(product, "engines", name, "execution-bindings.json"))["bindings"].(array)[0].(object)
		if binding["transport_ref"].(object)["sha256"] != transportHash || fileHash(t, filepath.Join(product, "engines", name, "transport.json")) != transportHash {
			t.Errorf("stale transport projection for %s", name)
		}
	}
}
