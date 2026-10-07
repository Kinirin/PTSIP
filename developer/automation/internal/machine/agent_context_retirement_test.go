package machine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func agentSelectorRetirementFixture(t *testing.T) *Repository {
	t.Helper()
	r := policyTestRepo(t)
	source, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"developer/state/index.yaml", "developer/state/repository-state-index.schema.json", ".ptsip/index.json", ".ptsip/profiles/index.json", "src/ptsip/repository/schemas/repository-index.schema.json"} {
		raw, err := source.ReadSource(ref)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.AtomicWrite(ref, raw, nil); err != nil {
			t.Fatal(err)
		}
	}
	profile := Object{"components": []any{Object{"id": "current", "include": []any{"src/agent_contracts/**"}}}}
	policyTestWrite(t, r, "developer/profiles/ptsip-repository.yaml", profile)
	policyTestWrite(t, r, ".ptsip/profiles/main.ptsip.yaml", profile)
	return r
}

func TestAgentProfileSelectorsPreserveRetirementAndNormalizationWithoutPython(t *testing.T) {
	for _, field := range []string{"include", "analysis_inputs", "associated_artifacts"} {
		for _, retired := range agentRetiredProfileRoots {
			t.Run(field+"/"+retired, func(t *testing.T) {
				r := agentSelectorRetirementFixture(t)
				selector := retired
				if !strings.HasSuffix(selector, ".md") && !strings.HasSuffix(selector, ".yaml") {
					selector += "/**"
				}
				returned := retired
				if strings.HasSuffix(selector, "/**") {
					returned += "/returned.yaml"
				}
				if err := r.AtomicWrite(returned, []byte("returned legacy content\n"), nil); err != nil {
					t.Fatal(err)
				}
				profile := Object{"components": []any{Object{"id": "current", "include": []any{"src/agent_contracts/**"}}}}
				if field == "associated_artifacts" {
					profile[field] = []any{Object{"id": "legacy", "include": []any{selector}}}
				} else {
					Map(List(profile["components"])[0])[field] = []any{selector}
				}
				policyTestWrite(t, r, "developer/profiles/ptsip-repository.yaml", profile)
				result := VerifyCurrentAgentProfileSelectors(r)
				violations := List(Map(result["detail"])["retired_selectors"])
				if result["status"] != "FAIL" || len(violations) != 1 || Map(violations[0])["selector"] != selector || Map(violations[0])["retired_root"] != retired {
					t.Fatal(result)
				}
			})
		}
	}
	for _, selector := range []string{"./spec/**", `spec\**`, "./docs/planning/**"} {
		t.Run(selector, func(t *testing.T) {
			r := agentSelectorRetirementFixture(t)
			policyTestWrite(t, r, ".ptsip/profiles/main.ptsip.yaml", Object{"components": []any{Object{"id": "legacy", "include": []any{selector}}}})
			if result := VerifyCurrentAgentProfileSelectors(r); result["status"] != "FAIL" {
				t.Fatal(result)
			}
		})
	}
}

func TestAgentProfileContractsFailClosedAndNonDefaultProfilesAreRevalidated(t *testing.T) {
	for _, ref := range []string{"developer/profiles/ptsip-repository.yaml", ".ptsip/index.json", ".ptsip/profiles/index.json", ".ptsip/profiles/main.ptsip.yaml"} {
		t.Run(ref, func(t *testing.T) {
			r := agentSelectorRetirementFixture(t)
			path, _ := r.Path(ref)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if result := VerifyCurrentAgentProfileSelectors(r); result["status"] != "FAIL" || len(Strings(Map(result["detail"])["errors"])) == 0 {
				t.Fatal(result)
			}
		})
	}
	for _, payload := range []any{[]any{}, Object{}, Object{"components": "src/**"}, Object{"components": []any{Object{}}}, Object{"components": []any{Object{"id": "current", "include": "src/**"}}}, Object{"components": []any{Object{"id": "current", "include": []any{"src/**"}, "analysis_inputs": []any{nil}}}}, Object{"components": []any{Object{"id": "current", "include": []any{"src/**"}}}, "associated_artifacts": Object{}}} {
		r := agentSelectorRetirementFixture(t)
		if err := r.WriteYAML(".ptsip/profiles/main.ptsip.yaml", payload, nil); err != nil {
			t.Fatal(err)
		}
		if result := VerifyCurrentAgentProfileSelectors(r); result["status"] != "FAIL" {
			t.Fatal(result)
		}
	}
	for _, defect := range []string{"retired_review", "missing_review", "unsafe_namespace", "unsafe_catalog", "history_is_prose"} {
		t.Run(defect, func(t *testing.T) {
			r := agentSelectorRetirementFixture(t)
			switch defect {
			case "retired_review", "missing_review":
				catalog, _ := r.Read(".ptsip/profiles/index.json")
				catalog["profiles"] = append(List(catalog["profiles"]), Object{"id": "review", "resource": "review.ptsip.yaml"})
				policyTestWrite(t, r, ".ptsip/profiles/index.json", catalog)
				if defect == "retired_review" {
					policyTestWrite(t, r, ".ptsip/profiles/review.ptsip.yaml", Object{"components": []any{Object{"id": "legacy", "include": []any{"spec/**"}}}})
				}
			case "unsafe_namespace":
				index, _ := r.Read(".ptsip/index.json")
				Map(Map(index["namespaces"])["profiles"])["index"] = "../outside.json"
				policyTestWrite(t, r, ".ptsip/index.json", index)
			case "unsafe_catalog":
				catalog, _ := r.Read(".ptsip/profiles/index.json")
				catalog["profiles"] = append(List(catalog["profiles"]), Object{"id": "outside", "resource": "../outside.ptsip.yaml"})
				policyTestWrite(t, r, ".ptsip/profiles/index.json", catalog)
			case "history_is_prose":
				policyTestWrite(t, r, "developer/profiles/ptsip-repository.yaml", Object{"components": []any{Object{"id": "current", "include": []any{"src/agent_contracts/**"}, "purpose": "Replaces spec/** and MEMORY.md.", "analysis_inputs": []any{"developer/planning/**", ".ptsip/context/**"}}}})
			}
			result := VerifyCurrentAgentProfileSelectors(r)
			want := "FAIL"
			if defect == "history_is_prose" {
				want = "PASS"
			}
			if result["status"] != want {
				t.Fatal(result)
			}
		})
	}
}

func agentOperationRetirementFixture(t *testing.T) *Repository {
	t.Helper()
	r := &Repository{Root: t.TempDir()}
	policyTestWrite(t, r, "src/ptsip/agent_contracts/index.yaml", Object{"operations": []any{Object{"id": "migrate-profile", "ref": "operations/migrate-profile.yaml"}}})
	policyTestWrite(t, r, "src/ptsip/agent_contracts/operations/migrate-profile.yaml", Object{"operation_id": "PTSIP-OP-MIGRATE-PROFILE-001", "implementation_refs": []any{"src/ptsip/migration/analyzer.py"}})
	if err := r.AtomicWrite("src/ptsip/migration/analyzer.py", []byte("raise RuntimeError('must not be imported')\n"), nil); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAgentOperationReferencesAreInspectedWithoutExecutionAndFailClosed(t *testing.T) {
	for _, refs := range []any{[]any{"../outside.py"}, []any{"/tmp/outside.py"}, []any{"C:/outside.py"}, []any{"C:outside.py"}, []any{`src\ptsip\migration\analyzer.py`}, []any{"src/ptsip/migration/*.py"}, []any{"https://example.com/code.py"}, nil, []any{}, "src/ptsip/migration/analyzer.py", []any{nil}, []any{""}, []any{"src/ptsip/migration"}} {
		r := agentOperationRetirementFixture(t)
		policyTestWrite(t, r, "src/ptsip/agent_contracts/operations/migrate-profile.yaml", Object{"operation_id": "PTSIP-OP-MIGRATE-PROFILE-001", "implementation_refs": refs})
		if result := VerifyAgentOperationImplementationRefs(r); result["status"] != "FAIL" {
			t.Fatal(result)
		}
	}
	for _, entries := range []any{[]any{}, nil, []any{Object{"ref": "../outside.yaml"}}, []any{Object{"ref": nil}}} {
		r := agentOperationRetirementFixture(t)
		policyTestWrite(t, r, "src/ptsip/agent_contracts/index.yaml", Object{"operations": entries})
		if result := VerifyAgentOperationImplementationRefs(r); result["status"] != "FAIL" {
			t.Fatal(result)
		}
	}
	r := agentOperationRetirementFixture(t)
	if result := VerifyAgentOperationImplementationRefs(r); result["status"] != "PASS" {
		t.Fatal(result)
	}
	path, _ := r.Path("src/ptsip/migration/analyzer.py")
	if err := os.Rename(path, filepath.Join(filepath.Dir(path), "analyzer_new.py")); err != nil {
		t.Fatal(err)
	}
	if result := VerifyAgentOperationImplementationRefs(r); result["status"] != "FAIL" || !strings.Contains(strings.Join(Strings(Map(result["detail"])["errors"]), "\n"), "analyzer.py") {
		t.Fatal(result)
	}
	policyTestWrite(t, r, "src/ptsip/agent_contracts/operations/migrate-profile.yaml", Object{"operation_id": "PTSIP-OP-MIGRATE-PROFILE-001", "implementation_refs": []any{"src/ptsip/migration/analyzer_new.py"}})
	if result := VerifyAgentOperationImplementationRefs(r); result["status"] != "PASS" {
		t.Fatal(result)
	}
	policyTestWrite(t, r, "src/ptsip/agent_contracts/operations/unindexed.yaml", Object{"implementation_refs": []any{"missing.py"}})
	if result := VerifyAgentOperationImplementationRefs(r); result["status"] != "FAIL" {
		t.Fatal(result)
	}
}

func TestAgentOperationReferencesRejectExternalSymlinks(t *testing.T) {
	for _, target := range []string{"implementation", "index"} {
		t.Run(target, func(t *testing.T) {
			r := agentOperationRetirementFixture(t)
			outside := filepath.Join(t.TempDir(), "outside.yaml")
			if err := os.WriteFile(outside, []byte("operations: []\n"), 0644); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(r.Root, "src/ptsip/migration/outside.py")
			if target == "index" {
				link = filepath.Join(r.Root, "src/ptsip/agent_contracts/index.yaml")
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
			} else {
				policyTestWrite(t, r, "src/ptsip/agent_contracts/operations/migrate-profile.yaml", Object{"operation_id": "PTSIP-OP-MIGRATE-PROFILE-001", "implementation_refs": []any{"src/ptsip/migration/outside.py"}})
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Skipf("symlink creation unavailable on this platform: %v", err)
			}
			result := VerifyAgentOperationImplementationRefs(r)
			if result["status"] != "FAIL" {
				t.Fatal("external implementation or index was accepted", result)
			}
		})
	}
}

func agentContextRetirementFixture(t *testing.T) *Repository {
	t.Helper()
	r := policyTestRepo(t)
	source, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"src", "developer/state", "developer/profiles", ".ptsip"} {
		root, err := source.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "__pycache__" || entry.Name() == ".cache" || entry.Name() == "legacy" {
					return filepath.SkipDir
				}
				return nil
			}
			relative, err := filepath.Rel(source.Root, path)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return r.AtomicWrite(filepath.ToSlash(relative), raw, nil)
		}); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestAgentContextNativeVerificationRevalidatesProfileAndImplementationInputs(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []string{"M5", "AUTO"} {
		result, err := VerifyAgentContextMigration(r, stage)
		if err != nil || result["status"] != "PASS" {
			t.Fatal(result, err)
		}
	}
	for _, defect := range []string{"retired_profile_selector", "missing_implementation"} {
		t.Run(defect, func(t *testing.T) {
			r := agentContextRetirementFixture(t)
			checkID := "CURRENT_PROJECT_PROFILE_SELECTORS_REVALIDATED"
			if defect == "retired_profile_selector" {
				ref := "developer/profiles/ptsip-repository.yaml"
				profile, err := r.Read(ref)
				if err != nil {
					t.Fatal(err)
				}
				Map(List(profile["components"])[0])["analysis_inputs"] = []any{"docs/planning/**"}
				policyTestWrite(t, r, ref, profile)
			} else {
				ref := "src/ptsip/agent_contracts/operations/migrate-profile.yaml"
				operation, err := r.Read(ref)
				if err != nil {
					t.Fatal(err)
				}
				operation["implementation_refs"] = []any{"src/ptsip/migration/deleted_implementation.py"}
				policyTestWrite(t, r, ref, operation)
				checkID = "OPERATION_IMPLEMENTATION_REFS_EXIST"
			}
			result, err := VerifyAgentContextMigration(r, "AUTO")
			if err != nil || result["status"] != "FAIL" || result["stage"] != "M8" {
				t.Fatal(result, err)
			}
			found := false
			for _, raw := range List(result["checks"]) {
				check := Map(raw)
				if check["id"] == checkID {
					found = check["status"] == "FAIL"
				}
			}
			if !found {
				t.Fatal("current input drift was not revalidated", result)
			}
		})
	}
}
