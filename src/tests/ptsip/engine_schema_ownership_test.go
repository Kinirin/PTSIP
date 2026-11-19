package contracts_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type engineSchemaAssignment struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	Owner        string `json:"owner_engine"`
	ID           string `json:"schema_id"`
	SourceBlob   string `json:"source_blob_sha"`
	SourceSHA256 string `json:"source_sha256"`
	Version      string `json:"version"`
}

func engineSchemaAssignments(t *testing.T) []engineSchemaAssignment {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "src", "planning", "0.3.8", "ptsip-schema-centralization.plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var plan struct {
		Migration struct {
			Assignments []engineSchemaAssignment `json:"ownership_assignments"`
		} `json:"engine_schema_migration"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Migration.Assignments) != 21 {
		t.Fatalf("approved native schema ownership count = %d, want 21", len(plan.Migration.Assignments))
	}
	return plan.Migration.Assignments
}

func nativeEngineSchemas(t *testing.T) map[string]*jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(offlineProfileLoader{})
	ids := map[string]string{}
	for _, assignment := range engineSchemaAssignments(t) {
		doc := readObject(t, filepath.Join(repositoryRoot(t), filepath.FromSlash(assignment.Target)))
		id, ok := doc["$id"].(string)
		if !ok || id != assignment.ID {
			t.Fatalf("%s resource identity = %v, want %s", assignment.Target, doc["$id"], assignment.ID)
		}
		if prior, exists := ids[id]; exists {
			t.Fatalf("duplicate native schema identity %s: %s and %s", id, prior, assignment.Target)
		}
		ids[id] = strings.TrimSuffix(filepath.Base(assignment.Target), ".schema.json")
		if err := compiler.AddResource(id, doc); err != nil {
			t.Fatalf("add owner-local schema %s: %v", assignment.Target, err)
		}
	}
	compiled := map[string]*jsonschema.Schema{}
	for id, name := range ids {
		schema, err := compiler.Compile(id)
		if err != nil {
			t.Fatalf("compile offline native schema %s: %v", id, err)
		}
		compiled[name] = schema
	}
	return compiled
}

func TestNativeEngineSchemaOwnershipAndImmutableSources(t *testing.T) {
	root := repositoryRoot(t)
	index := readObject(t, filepath.Join(productRoot(t), "engines", "index.json"))
	routes := index["engine_schema_refs"].(object)
	owners := map[string]int{}
	seenTargets := map[string]bool{}
	for _, assignment := range engineSchemaAssignments(t) {
		assignment := assignment
		name := strings.TrimSuffix(filepath.Base(assignment.Target), ".schema.json")
		t.Run(name, func(t *testing.T) {
			prefix := "src/ptsip/engines/" + strings.ToLower(assignment.Owner) + "/src/schemas/"
			if !strings.HasPrefix(assignment.Target, prefix) || strings.Contains(assignment.Target, "..") {
				t.Fatalf("target is outside its recorded engine owner: %s", assignment.Target)
			}
			if seenTargets[assignment.Target] {
				t.Fatalf("duplicate schema target %s", assignment.Target)
			}
			seenTargets[assignment.Target] = true
			owners[assignment.Owner]++
			if routes[name] != assignment.Target {
				t.Fatalf("registered route %s = %v, want %s", name, routes[name], assignment.Target)
			}
			if blob := gitWorkingBlobSHA(t, assignment.Target); blob != assignment.SourceBlob {
				t.Fatalf("owner-local schema differs from approved source blob: got %s want %s", blob, assignment.SourceBlob)
			}
			command := exec.Command("git", "cat-file", "blob", assignment.SourceBlob)
			command.Dir = root
			original, err := command.Output()
			if err != nil {
				t.Fatalf("read immutable source blob: %v", err)
			}
			digest := sha256.Sum256(original)
			if hex.EncodeToString(digest[:]) != assignment.SourceSHA256 {
				t.Fatal("approved original source SHA256 does not match Git blob")
			}
			canonical, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(assignment.Target)))
			if err != nil {
				t.Fatal(err)
			}
			// Git's clean filter is checked above; permit checkout line endings.
			if !bytes.Equal(bytes.ReplaceAll(canonical, []byte("\r\n"), []byte("\n")), original) {
				t.Fatal("migrated schema changed bytes beyond checkout line endings")
			}
		})
	}
	for owner, count := range map[string]int{"DISCOVERY": 10, "EXECUTION": 6, "VALIDATION": 4, "STORAGE": 1} {
		if owners[owner] != count {
			t.Errorf("owner %s schema count = %d, want %d", owner, owners[owner], count)
		}
	}
	for key, route := range routes {
		text, ok := route.(string)
		if !ok || !strings.HasPrefix(text, "src/ptsip/engines/") || !strings.Contains(text, "/src/schemas/") {
			t.Errorf("native schema route %s does not select an owner-local source: %v", key, route)
		}
	}
	for _, relative := range []string{"engines/index.json", "engines/contracts/work-item-bindings.json", "engines/contracts/native-runtime.json"} {
		data, err := os.ReadFile(filepath.Join(productRoot(t), filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("engines/contracts/schemas/")) || bytes.Contains(data, []byte("engines/discovery/tpbg-f3.schema.json")) {
			t.Errorf("current registry retains retired native schema reference: %s", relative)
		}
		if bytes.Contains(data, []byte(`engines\\contracts\\schemas\\`)) {
			t.Errorf("current registry retains escaped Windows retired schema reference: %s", relative)
		}
	}
	_ = nativeEngineSchemas(t)
}

func TestOwnerLocalNativeSchemasValidateRegisteredEngines(t *testing.T) {
	schemas := nativeEngineSchemas(t)
	index := readObject(t, filepath.Join(productRoot(t), "engines", "index.json"))
	for _, raw := range index["official_engine_entries"].([]any) {
		entry := raw.(object)
		kind := entry["kind"].(string)
		t.Run(kind, func(t *testing.T) {
			for _, item := range []struct{ reference, schema string }{
				{"manifest_ref", "core-manifest"}, {"contract_ref", "capability-contract"},
			} {
				path := entry[item.reference].(string)
				doc := readObject(t, filepath.Join(repositoryRoot(t), filepath.FromSlash(path)))
				if err := schemas[item.schema].Validate(doc); err != nil {
					t.Fatalf("current %s %s invalid after schema migration: %v", kind, item.schema, err)
				}
				if item.schema == "capability-contract" {
					invalid := clone(t, doc)
					invalid["execution_permission_from_declaration"] = true
					if err := schemas[item.schema].Validate(invalid); err == nil {
						t.Fatal("schema relocation allowed execution permission from declaration")
					}
				}
			}
			bindings := readObject(t, filepath.Join(repositoryRoot(t), filepath.FromSlash(entry["execution_binding_ref"].(string))))
			for i, rawBinding := range bindings["bindings"].([]any) {
				binding := rawBinding.(object)
				if err := schemas["execution-binding"].Validate(binding); err != nil {
					t.Fatalf("binding %d: %v", i, err)
				}
				invalid := clone(t, binding)
				invalid["permission_grant"] = true
				if err := schemas["execution-binding"].Validate(invalid); err == nil {
					t.Fatal("relocated binding schema granted permission")
				}
			}
		})
	}
	for name, schema := range schemas {
		if err := schema.Validate(object{}); err == nil {
			t.Error(fmt.Sprintf("relocated %s schema accepted an empty object", name))
		}
	}
}
