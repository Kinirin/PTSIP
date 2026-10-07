package repository_test

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const contextSource = ".ptsip/context/source/context.source.json"
const contextJSON = ".ptsip/context/context.json"
const contextJSONL = ".ptsip/context/context.jsonl"
const contextSchema = ".ptsip/context/context.schema.json"

func contextRecords(t *testing.T, repo *testrepo.Repository) []testrepo.Object {
	t.Helper()
	path, err := repo.Path(contextJSONL)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	records := []testrepo.Object{}
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	scanner.Buffer(make([]byte, 4096), 8*1024*1024)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var record testrepo.Object
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}

func contextFixture(t *testing.T) *testrepo.Repository {
	t.Helper()
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyTree(t, repo, "developer/policy")
	testrepo.CopyFiles(t, repo, ".ptsip/index.json", "pyproject.toml")
	testrepo.WriteJSON(t, repo, "semantic-input.json", testrepo.ReadJSON(t, testrepo.Open(testrepo.Root(t)), contextSource))
	return repo
}

func TestNativeContextProjectionPreservesPythonRegressionCoverage(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	repo := testrepo.Open(testrepo.Root(t))
	t.Run("repository_context_plane_is_in_sync", func(t *testing.T) {
		result, err, output := testrepo.CLI(t, binary, repo.Root, "context-projection", "check")
		if err != nil || result["status"] != "PASS" {
			t.Fatalf("%v\n%s", err, output)
		}
	})
	t.Run("json_and_jsonl_are_semantically_equivalent", func(t *testing.T) {
		bundle := testrepo.ReadJSON(t, repo, contextJSON)
		records := contextRecords(t, repo)
		if len(records) < 2 || records[0]["record_type"] != "manifest" || records[1]["record_type"] != "state" {
			t.Fatal("missing exact manifest/state record framing")
		}
		memory := []any{}
		for i, record := range records[2:] {
			if record["record_type"] != "memory" || record["ordinal"] != float64(i) {
				t.Fatal("non-contiguous memory records", record)
			}
			memory = append(memory, record["payload"])
		}
		decoded := testrepo.Object{"format": "ptsip-context/v1", "source_binding": records[0]["source_binding"], "projection_policy": records[0]["projection_policy"], "state": records[1]["payload"], "memory": memory}
		if !reflect.DeepEqual(decoded, bundle) {
			t.Fatal("JSONL differs from JSON bundle")
		}
	})
	t.Run("schema_accepts_bundle_and_every_record", func(t *testing.T) {
		schema := testrepo.ReadJSON(t, repo, contextSchema)
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource(schema["$id"].(string), schema); err != nil {
			t.Fatal(err)
		}
		compiled, err := compiler.Compile(schema["$id"].(string))
		if err != nil {
			t.Fatal(err)
		}
		if err := compiled.Validate(testrepo.ReadJSON(t, repo, contextJSON)); err != nil {
			t.Fatal(err)
		}
		for _, record := range contextRecords(t, repo) {
			if err := compiled.Validate(record); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("provider_neutral_extensible_projection_policy", func(t *testing.T) {
		policy := testrepo.ReadJSON(t, repo, contextSource)["projection_policy"]
		want := testrepo.Object{"authority": "CANONICAL_SEMANTIC_MODEL", "extension_policy": "EXTENSIBLE", "generated_only": true, "mandatory_formats": []any{"json", "jsonl", "schema-json"}, "provider_scope": []any{"OPENAI", "ANTHROPIC", "GOOGLE", "XAI"}, "semantic_equivalence": "REQUIRED"}
		if !reflect.DeepEqual(policy, want) {
			t.Fatal(policy)
		}
	})
	t.Run("single_write_generates_all_mandatory_projections", func(t *testing.T) {
		fixture := contextFixture(t)
		result, err, output := testrepo.CLI(t, binary, fixture.Root, "context-projection", "write", "--input", "semantic-input.json")
		if err != nil || result["status"] != "PASS" {
			t.Fatalf("%v\n%s", err, output)
		}
		for _, ref := range []string{contextSource, contextJSON, contextJSONL, contextSchema} {
			path, err := fixture.Path(ref)
			if err != nil {
				t.Fatal(err)
			}
			if info, err := os.Stat(path); err != nil || info.IsDir() {
				t.Fatalf("missing projection %s: %v", ref, err)
			}
		}
		_, err, output = testrepo.CLI(t, binary, fixture.Root, "context-projection", "check")
		if err != nil {
			t.Fatalf("%v\n%s", err, output)
		}
	})
	t.Run("projection_drift_fails_closed", func(t *testing.T) {
		fixture := contextFixture(t)
		_, err, output := testrepo.CLI(t, binary, fixture.Root, "context-projection", "write", "--input", "semantic-input.json")
		if err != nil {
			t.Fatalf("%v\n%s", err, output)
		}
		testrepo.WriteJSON(t, fixture, contextJSON, testrepo.Object{})
		_, err, output = testrepo.CLI(t, binary, fixture.Root, "context-projection", "check")
		if err == nil || !strings.Contains(output, "DRIFT") {
			t.Fatalf("%v\n%s", err, output)
		}
	})
}
