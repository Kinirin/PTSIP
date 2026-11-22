package contracts_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const pp200ToPP300 = "projection.migration.pp200-to-pp300"

func TestPP200ToPP300Projection(t *testing.T) {
	packageRoot := installedProfileRuntime(t, "projection").packageRoot
	_, targetSchema := officialPP300Schema(t)
	project := func(t *testing.T, profile object) object {
		t.Helper()
		return repositoryOperation(t, "projection", pp200ToPP300, object{"package_root": packageRoot, "profile": profile})
	}
	checkTarget := func(t *testing.T, result object, expected object) {
		t.Helper()
		if result["status"] != "PROJECTED" || len(result["unmapped"].(array)) != 0 || !reflect.DeepEqual(result["target_profile"], expected) {
			t.Fatalf("expected complete semantic projection, got %v", result)
		}
		if err := targetSchema.Validate(result["target_profile"]); err != nil {
			t.Fatalf("projected Profile did not validate in independent Go validator: %v", err)
		}
	}

	t.Run("explicit_default_and_outgoing_pairs", func(t *testing.T) {
		profile := explicitPP200WithDependencies()
		original := clone(t, profile)
		expected := clone(t, profile)
		expected["ptsip"].(object)["version"] = "pp.3.00"
		expected["responsibility_map"].(object)["mode"] = "DEFAULT"
		delete(expected, "component_dependency_policy")
		parts := expected["components"].(array)
		parts[0].(object)["dependency"] = object{"default": "deny", "allow": array{object{"from": "api", "to": "lib"}}}
		parts[1].(object)["dependency"] = object{"default": "deny", "allow": array{object{"from": "lib", "to": "api"}}, "deny": array{object{"from": "lib", "to": "worker"}}}
		parts[2].(object)["dependency"] = object{"default": "deny"}

		result := project(t, profile)
		checkTarget(t, result, expected)
		if !reflect.DeepEqual(profile, original) || !reflect.DeepEqual(result, project(t, profile)) {
			t.Fatal("projection changed its source or produced a nondeterministic result")
		}
		envelope, err := runProfileOperation(t, "projection", pp200ToPP300, object{"package_root": packageRoot, "profile": profile})
		if err != nil || envelope["execution_authorized"] != false || envelope["normative_authority"] != false {
			t.Fatalf("semantic projection assigned mutation or normative authority: %v, %v", envelope, err)
		}
		changes := map[string]object{}
		for _, item := range result["changes"].(array) {
			entry := item.(object)
			path := entry["target_path"].(string)
			if _, duplicate := changes[path]; duplicate {
				t.Fatalf("ambiguous target pointer %s", path)
			}
			changes[path] = entry
		}
		for path, expected := range map[string]object{
			"/ptsip/version":                   {"source_path": "/ptsip/version", "before": "pp.2.00", "after": "pp.3.00"},
			"/responsibility_map/mode":         {"source_path": "/responsibility_map/mode", "before": "explicit", "after": "DEFAULT"},
			"/components/2/dependency/default": {"source_path": "/component_dependency_policy/default", "before": "deny", "after": "deny"},
			"/components/0/dependency/allow/0": {"source_path": "/component_dependency_policy/allow/1", "before": object{"from": "api", "to": "lib"}, "after": object{"from": "api", "to": "lib"}},
			"/components/1/dependency/allow/0": {"source_path": "/component_dependency_policy/allow/0", "before": object{"from": "lib", "to": "api"}, "after": object{"from": "lib", "to": "api"}},
			"/components/1/dependency/deny/0":  {"source_path": "/component_dependency_policy/deny/0", "before": object{"from": "lib", "to": "worker"}, "after": object{"from": "lib", "to": "worker"}},
		} {
			expected["target_path"] = path
			if !reflect.DeepEqual(changes[path], expected) {
				t.Errorf("wrong change evidence for %s: got %v, want %v", path, changes[path], expected)
			}
		}
	})

	t.Run("untouched_historical_example_has_no_additional_dependency_restriction", func(t *testing.T) {
		profile := historicalPP200MigrationExample(t, "example.ptsip.yaml")
		if _, exists := profile["component_dependency_policy"]; exists {
			t.Fatal("recorded historical example no longer exercises an absent policy")
		}
		expected := clone(t, profile)
		expected["ptsip"].(object)["version"] = "pp.3.00"
		expected["responsibility_map"].(object)["mode"] = "DEFAULT"
		for _, component := range expected["components"].(array) {
			component.(object)["dependency"] = object{"default": "allow"}
		}
		result := project(t, profile)
		checkTarget(t, result, expected)
		for _, entry := range result["changes"].(array) {
			row := entry.(object)
			if strings.HasSuffix(row["target_path"].(string), "/dependency/default") && (row["source_path"] != "/component_dependency_policy" || row["before"] != nil || row["after"] != "allow") {
				t.Errorf("absent source policy was not represented in the trace: %v", row)
			}
		}
	})

	for _, mode := range []string{"template", "hybrid"} {
		t.Run("recorded_"+mode+"_exact_materialization", func(t *testing.T) {
			profile := historicalPP200MigrationExample(t, mode+"-python-package.ptsip.yaml")
			binding := profile["responsibility_map"].(object)
			mapFromCatalog := recordedTemplateMap(t, binding["template"].(object))
			expected := clone(t, profile)
			expected["ptsip"].(object)["version"] = "pp.3.00"
			expected["responsibility_map"] = object{"mode": "DEFAULT"}
			for key, field := range mapFromCatalog {
				expected[key] = field
			}
			if mode == "hybrid" {
				// Both fixture overrides are full rows for the exact template IDs.
				expected["components"] = clone(t, binding["overrides"].(object))["components"]
				expected["associated_artifacts"] = array{}
			}
			for _, part := range expected["components"].(array) {
				part.(object)["dependency"] = object{"default": "allow"}
			}
			result := project(t, profile)
			checkTarget(t, result, expected)
			for _, entry := range result["changes"].(array) {
				row := entry.(object)
				if row["source_path"] == "/responsibility_map" && row["target_path"] == "/components" && reflect.DeepEqual(row["before"], binding) && reflect.DeepEqual(row["after"], expected["components"]) {
					return
				}
			}
			t.Fatal("template identity, overrides, and materialized rows are not bound in the change trace")
		})
	}

	t.Run("hybrid_keeps_template_order_and_appends_new_ids_in_source_order", func(t *testing.T) {
		profile := historicalPP200MigrationExample(t, "hybrid-python-package.ptsip.yaml")
		overrides := profile["responsibility_map"].(object)["overrides"].(object)
		rows := overrides["components"].(array)
		for _, id := range []string{"z-added", "a-added"} {
			rows = append(rows, object{"id": id, "classification": "PRODUCT", "purpose": "New project component", "include": array{id + "/**"}})
		}
		overrides["components"] = append(array{rows[1], rows[0]}, rows[2:]...)
		result := project(t, profile)
		if result["status"] != "PROJECTED" {
			t.Fatalf("exact materialization failed: %v", result)
		}
		var ids []string
		for _, item := range result["target_profile"].(object)["components"].(array) {
			ids = append(ids, item.(object)["id"].(string))
		}
		if !reflect.DeepEqual(ids, []string{"package", "package-tests", "z-added", "a-added"}) {
			t.Fatalf("effective component order changed: %v", ids)
		}
	})

	for _, tc := range []struct {
		name   string
		amend  func(object)
		issues map[string]string
	}{
		{"custom_fields_and_lineage", func(profile object) {
			profile["ptsip"].(object)["revision"] = "Rev.0002"
			profile["components"].(array)[0].(object)["extra/name~"] = object{"value": true}
			profile["responsibility_map"].(object)["user_data"] = "unmapped responsibility-map extension"
		}, map[string]string{"/ptsip/revision": "LINEAGE_UNMAPPED", "/components/0/extra~1name~0": "UNSUPPORTED_SOURCE_FIELD", "/responsibility_map/user_data": "UNSUPPORTED_SOURCE_FIELD"}},
		{"conflicting_and_missing_dependency_pairs", func(profile object) {
			profile["component_dependency_policy"].(object)["deny"] = array{object{"from": "api", "to": "lib"}, object{"from": "api", "to": "unknown"}}
		}, map[string]string{"/component_dependency_policy/deny/0": "DEPENDENCY_RELATION_CONFLICT", "/component_dependency_policy/deny/1/to": "REFERENCE_UNRESOLVED"}},
		{"ambiguous_component_and_missing_artifact_anchor", func(profile object) {
			profile["components"].(array)[2].(object)["id"] = "lib"
			profile["associated_artifacts"].(array)[0].(object)["anchor"] = "unknown"
		}, map[string]string{"/components/2/id": "IDENTITY_AMBIGUOUS", "/associated_artifacts/0/anchor": "REFERENCE_UNRESOLVED"}},
		{"incomplete_dependency_policy", func(profile object) {
			delete(profile["component_dependency_policy"].(object), "default")
		}, map[string]string{"/component_dependency_policy/default": "DEPENDENCY_DEFAULT_UNMAPPED"}},
		{"template_without_materialization", func(profile object) {
			profile["responsibility_map"] = object{"mode": "template", "template": object{"id": "python", "revision": "template-revision"}}
			for _, field := range []string{"components", "associated_artifacts", "relationships", "component_dependency_policy"} {
				delete(profile, field)
			}
		}, map[string]string{"/responsibility_map/template": "RESPONSIBILITY_MAP_UNMAPPED"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := explicitPP200WithDependencies()
			tc.amend(profile)
			result := project(t, profile)
			if result["status"] != "UNMAPPED" || result["target_profile"] != nil || len(result["changes"].(array)) != 0 {
				t.Fatalf("unsupported source produced a partial or apparent candidate: %v", result)
			}
			issues := map[string]string{}
			for _, row := range result["unmapped"].(array) {
				diagnostic := row.(object)
				issues[diagnostic["source_path"].(string)] = diagnostic["code"].(string)
			}
			for path, code := range tc.issues {
				if issues[path] != code {
					t.Errorf("missing %s at %s in %v", code, path, issues)
				}
			}
		})
	}

	t.Run("tampered_registered_target_is_rejected", func(t *testing.T) {
		candidateRoot := t.TempDir()
		for _, rel := range []string{"schemas/config.json", "schemas/history/profile/project-profile-pp.2.00.schema.json", "schemas/storage/profile/project-profile.schema.json"} {
			raw, err := os.ReadFile(filepath.Join(packageRoot, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatal(err)
			}
			if rel == "schemas/storage/profile/project-profile.schema.json" {
				raw = append(raw, '\n')
			}
			writeRepositoryFixture(t, candidateRoot, rel, string(raw))
		}
		result, err := runProfileOperation(t, "projection", pp200ToPP300, object{"package_root": candidateRoot, "profile": explicitPP200WithDependencies()})
		if err == nil || result["status"] != "FAIL" || !strings.Contains(fmt.Sprint(result["error"]), "SHA-256 mismatch") {
			t.Fatalf("tampered target did not fail its registered identity check: %v, %v", result, err)
		}
	})
}

func explicitPP200WithDependencies() object {
	profile := historicalProfileInstance("pp.2.00")
	profile["ptsip"].(object)["rule_profile"] = "project-rules.ptsip.yaml"
	components := array{}
	for _, id := range []string{"api", "lib", "worker"} {
		components = append(components, object{"id": id, "classification": "PRODUCT", "include": array{"src/" + id + "/**"},
			"purpose": id + " package", "roles": array{"IMPLEMENTATION"}, "consumers": array{"kept-consumer"}, "analysis_inputs": array{"assets/**"}})
	}
	profile["components"] = components
	profile["associated_artifacts"] = array{object{"id": "api-docs", "anchor": "api", "include": array{"docs/**"}, "purpose": "Reference material"}}
	profile["relationships"] = array{object{"id": "lib-api-docs", "from": "lib", "to": "api-docs", "type": "READS"}}
	profile["component_dependency_policy"] = object{"default": "deny", "allow": array{object{"from": "lib", "to": "api"}, object{"from": "api", "to": "lib"}},
		"deny": array{object{"from": "lib", "to": "worker"}}}
	return profile
}

func recordedTemplateMap(t *testing.T, binding object) object {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(productRoot(t), "engines", "projection", "src", "historical-profile-templates.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog []object
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	for _, row := range catalog {
		if row["id"] == binding["id"] && row["revision"] == binding["revision"] {
			return clone(t, row["map"].(object))
		}
	}
	t.Fatalf("recorded template binding is missing: %v", binding)
	return nil
}

func historicalPP200MigrationExample(t *testing.T, fixture string) object {
	t.Helper()
	const fixtureCommit = "0cb9745113ca87de4c59924b9ac8d4f62d1fc21d"
	const fixtureDirectory = "src/ptsip/profiles/history/pp.2.00/"
	root := repositoryRoot(t)
	cmd := exec.Command("git", "cat-file", "blob", fixtureCommit+":"+fixtureDirectory+fixture)
	cmd.Dir = root
	data, err := cmd.CombinedOutput()
	if err != nil {
		shallow := exec.Command("git", "rev-parse", "--is-shallow-repository")
		shallow.Dir = root
		commit := exec.Command("git", "cat-file", "-e", fixtureCommit+"^{commit}")
		commit.Dir = root
		if repoType, shallowErr := shallow.Output(); shallowErr == nil && strings.TrimSpace(string(repoType)) == "true" && commit.Run() != nil {
			t.Skip("recorded historical Profile example is outside the shallow checkout")
		}
		t.Fatalf("read recorded Profile example: %v: %s", err, data)
	}
	var profile object
	if err := yaml.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	return profile
}
