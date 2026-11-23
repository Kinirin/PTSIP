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
	schema := filepath.Join(root, CompositionSchema)
	raw, err := os.ReadFile(filepath.Join(source, CompositionSchema))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(schema), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schema, raw, 0644); err != nil {
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

func TestSourceCompositionRejectsAmbiguousOrRetiredAuthority(t *testing.T) {
	tests := []struct {
		name string
		file string
		from string
		to   string
	}{
		{"old-route", releaseIndexPath, CompositionPath, historicalCompositionPath},
		{"duplicate-route", releaseIndexPath, "\"platform\": \"Common\",", "\"platform\": \"Common\", \"platform\": \"Common\","},
		{"fallback-source", CompositionPath, "\"file\": \"src/VERSION\"", "\"file\": \"pyproject.toml\""},
		{"publication-grant", CompositionPath, "\"publication_authorized\": false", "\"publication_authorized\": true"},
		{"stale-source-binding", CompositionPath, "EXACT_FINAL_COMMITTED_SHA", "ANY_CHECKOUT"},
		{"missing-receipt-digest", CompositionPath, "\"wheel_sha256\"", "\"filename\""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture(t)
			file := filepath.Join(root, test.file)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			mutated := strings.Replace(string(raw), test.from, test.to, 1)
			if mutated == string(raw) {
				t.Fatal("fixture was not mutated")
			}
			if err := os.WriteFile(file, []byte(mutated), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadSourceComposition(root); err == nil {
				t.Fatal("invalid source composition was admitted")
			}
		})
	}
}

func TestSourceCompositionValidatesNativeWheelReceipt(t *testing.T) {
	root := fixture(t)
	digest := strings.Repeat("a", 64)
	receipt := map[string]any{
		"source_sha": strings.Repeat("a", 40), "verified_snapshot_sha256": digest, "wheel_sha256": digest,
		"filename": "ptsip-0.3.8-py3-none-linux_x86_64.whl", "platform": "linux-x64",
		"components": map[string]any{
			"tool":     map[string]string{"version": "T.0.3.8", "sha256": digest},
			"profiles": map[string]string{"version": "pp.3.00", "sha256": digest},
			"context":  map[string]string{"version": "2.2.0", "sha256": digest},
		},
	}
	encode := func() []byte {
		t.Helper()
		raw, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	if err := ValidateNativeWheelReceipt(root, encode()); err != nil {
		t.Fatal(err)
	}
	receipt["publication_authorized"] = false
	if err := ValidateNativeWheelReceipt(root, encode()); err == nil {
		t.Fatal("receipt schema admitted an unregistered publication field")
	}
	delete(receipt, "publication_authorized")
	receipt["verified_snapshot_sha256"] = "present"
	if err := ValidateNativeWheelReceipt(root, encode()); err == nil {
		t.Fatal("receipt schema did not enforce the SHA-256 format")
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
