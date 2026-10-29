package boundary

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	base := filepath.Join(source, ".ptsip/contracts/release")
	if err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCurrentBoundaryNeedsNoRetiredRoots(t *testing.T) {
	root := fixture(t)
	if _, code := Run(root); code != 0 {
		t.Fatal("current structure rejected")
	}
	for _, op := range []string{"RELEASE_STRUCTURE_CHECK", "ENGINE_LOCAL_VERIFICATION", "SOURCE_BOUND_TEST_EVIDENCE"} {
		if err := CheckOperation(root, op); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckOperation(root, "PUBLISH"); err == nil {
		t.Fatal("unregistered publication operation accepted")
	}
	if err := RequirePublication(root); err == nil || !strings.Contains(err.Error(), "SUCCESSOR_AUTHORITY_UNRESOLVED") {
		t.Fatalf("publication gate: %v", err)
	}
	for _, retired := range []string{"developer", "registry", "schemas", ".github"} {
		if _, err := os.Stat(filepath.Join(root, retired)); !os.IsNotExist(err) {
			t.Fatalf("fixture contains retired root: %s", retired)
		}
	}
}

func TestBoundaryRejectsAuthorityWaiversAndInventoryDrift(t *testing.T) {
	mutations := map[string]any{
		"publication_authorized": true, "runtime_activation_authorized": true, "legacy_fallback": true,
		"policy_authority": "APPROVED", "publication_admission": "PASS", "replacement_required": false,
		"verification_waivers": []string{"legacy-completion"},
		"operations":           []string{"RELEASE_STRUCTURE_CHECK,ENGINE_LOCAL_VERIFICATION,SOURCE_BOUND_TEST_EVIDENCE"},
		"engine_inventory":     []string{"context,discovery,execution,planning,projection,storage,validation"},
		"artifact_digest":      "OPTIONAL", "source_sha": "ANY_PREVIOUS_RUN", "unknown_field": false,
	}
	for key, value := range mutations {
		t.Run(key, func(t *testing.T) {
			root := fixture(t)
			file := filepath.Join(root, Path)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			data[key] = value
			raw, err = json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, raw, 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil {
				t.Fatalf("forged %s accepted", key)
			}
		})
	}
}

func TestBoundaryRejectsMissingAndTrailingData(t *testing.T) {
	root := fixture(t)
	file := filepath.Join(root, Path)
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"{}", "invalid"} {
		if err := os.WriteFile(file, append(append([]byte{}, raw...), []byte(suffix)...), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil {
			t.Fatal("trailing data accepted")
		}
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("missing boundary accepted")
	}
}

// Scan production source literals only; negative fixtures may mention retired
// paths. A reintroduced authority input must fail before any external tool runs.
func TestReleaseRuntimeHasNoRetiredAuthorityInput(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			for _, prefix := range []string{"developer/automation", "developer/policy/approvals", "developer/policy/contracts", "developer/policy/registries", ".github/checks", ".github/scripts", ".github/release", "registry/"} {
				if strings.HasPrefix(value, prefix) {
					t.Errorf("retired runtime input %q in %s", value, path)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStructureCheckRejectsReroutedAndPublishingIndexes(t *testing.T) {
	for _, key := range []string{"validator_name", "engine_entry", "verifier_root", "normative_policy_activation", "contract_ref", "validation_rules_ref"} {
		t.Run(key, func(t *testing.T) {
			root := fixture(t)
			file := filepath.Join(root, ".ptsip/contracts/release/index.json")
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			data[key] = "unregistered"
			raw, err = json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, raw, 0644); err != nil {
				t.Fatal(err)
			}
			if _, code := Run(root); code == 0 {
				t.Fatalf("rerouted %s accepted", key)
			}
		})
	}
}
