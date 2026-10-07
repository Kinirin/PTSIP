package binding_test

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func rootResolver(t *testing.T) *binding.Resolver {
	t.Helper()
	resolver, err := binding.NewResolver(testrepo.Open(testrepo.Root(t)))
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

// The pre-transition binding evidence and the admitted migration graph form an
// independent coverage oracle. This helper never reads retired policy bodies and
// never becomes a runtime policy lookup or a substitute for canonical routing.
func expectedMigratedRefs(t *testing.T, scope, operation string) []any {
	t.Helper()
	repo := testrepo.Open(testrepo.Root(t))
	graph, err := repo.Read("developer/policy/registries/root-family-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]object{}
	for _, raw := range graph["sources"].([]any) {
		source := raw.(object)
		sources[source["source_policy_id"].(string)] = source
	}
	path, err := repo.Path("developer/policy/analysis/root-family-transition-bindings.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	var selected object
	for scanner.Scan() {
		var record object
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		if record["scope"] == scope {
			if selected != nil {
				t.Fatal("duplicate audit binding", scope)
			}
			selected = record
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if selected == nil {
		t.Fatal("missing exact audit scope", scope)
	}
	refs := selected["default_refs"].([]any)
	if operations, ok := selected["operations"].(object); ok {
		if override, exists := operations[operation]; exists {
			refs = override.([]any)
		}
	}
	result := []any{}
	byID := map[string]object{}
	for _, raw := range refs {
		ref := raw.(object)
		source := sources[ref["policy_id"].(string)]
		if source == nil || source["source_status"] != "ACTIVE" {
			t.Fatal("unadmitted audit source", ref)
		}
		for _, section := range ref["sections"].([]any) {
			pointer := "/rules/" + section.(string)
			matched := false
			for _, raw := range source["units"].([]any) {
				unit := raw.(object)
				sourcePointer := unit["source_pointer"].(string)
				if sourcePointer != pointer && !strings.HasPrefix(sourcePointer, pointer+"/") {
					continue
				}
				matched = true
				id := unit["policy_id"].(string)
				entry := byID[id]
				if entry == nil {
					entry = object{"policy_id": id, "path": "developer/policy/" + unit["policy_path"].(string), "status": "ACTIVE", "sections": []any{}}
					byID[id] = entry
					result = append(result, entry)
				}
				found := false
				for _, existing := range entry["sections"].([]any) {
					if existing == unit["section"] {
						found = true
					}
				}
				if !found {
					entry["sections"] = append(entry["sections"].([]any), unit["section"])
				}
			}
			if !matched {
				t.Fatal("audit responsibility lacks Root unit", pointer)
			}
		}
	}
	// Neutral module admission was added after the archived transition bindings.
	// Consume only its two admitted exact references from the canonical binding
	// source; every original responsibility above remains independently checked.
	if operation == "MODIFY" {
		path, err := repo.Path("developer/policy/policy-resolver-bindings/bindings.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 4*1024*1024)
		for scanner.Scan() {
			var record object
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if record["scope"] != scope {
				continue
			}
			refs := record["default_refs"].([]any)
			if operations, ok := record["operations"].(object); ok {
				if override, exists := operations[operation]; exists {
					refs = override.([]any)
				}
			}
			for _, raw := range refs {
				ref := raw.(object)
				id := ref["policy_id"]
				section, family := "", ""
				if id == "MPD-REAL-0004" {
					section, family = "neutral_module_creation", "REAL"
				} else if id == "MPD-CNTR-0003" {
					section, family = "root_family_projection_module", "CNTR"
				} else {
					continue
				}
				if !reflect.DeepEqual(ref["sections"], []any{section}) {
					t.Fatal("neutral module contract routing drift", ref)
				}
				result = append(result, object{"policy_id": id, "path": "developer/policy/" + family + "/" + id.(string) + ".yaml", "status": "ACTIVE", "sections": []any{section}})
			}
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func TestPolicyResolverBindingPlaneIsMachineValid(t *testing.T) {
	if err := rootResolver(t).ValidateBindings(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverRoutingPreservesEveryRegisteredSourceResponsibility(t *testing.T) {
	resolver := rootResolver(t)
	cases := []struct{ name, scope, bindingScope, operation string }{
		{"migration_plan", "src/ptsip/migration", "src/ptsip/migration", "PLAN"}, {"migration_modify", "src/ptsip/migration", "src/ptsip/migration", "MODIFY"}, {"migration_verify", "src/ptsip/migration", "src/ptsip/migration", "VERIFY"},
		{"exact_ancestor", "src/ptsip/migration/future_engine.py", "src/ptsip/migration", "READ"},
		{"github_authority", "src/ptsip/app/github_authority.py", "src/ptsip/app/github_authority.py", "MODIFY"},
		{"release_override", "README.md", ".", "RELEASE"},
		{"public_profile", "profiles/example.ptsip.yaml", "profiles", "MODIFY"},
		{"current_schema", "schemas/ptsip-profile-pp-1.01.schema.json", "schemas", "MODIFY"},
		{"future_registry", "registry/project-profile-contracts.yaml", "registry/project-profile-contracts.yaml", "MODIFY"},
		{"unrelated_modify", "README.md", ".", "MODIFY"},
		{"pyproject_override", "pyproject.toml", "pyproject.toml", "MODIFY"},
		{"tooling_test", ".github/workflows/tooling-test.yml", ".github/workflows/tooling-test.yml", "MODIFY"},
		{"remote_verify", "developer/automation/pp/pp_remote_verify.py", "developer/automation/pp/pp_remote_verify.py", "VERIFY"},
		{"release_verify", "developer/automation/pp/pp_release_verify.py", "developer/automation/pp/pp_release_verify.py", "RELEASE"},
		{"tooling_release", ".github/workflows/tooling-release.yml", ".github/workflows/tooling-release.yml", "RELEASE"},
		{"seed_transition", "developer/automation/seed_pp_102_transition.py", "developer/automation/seed_pp_102_transition.py", "MODIFY"},
		{"public_catalog_schema", "developer/policy/schemas/public-profile-catalog.schema.json", "developer/policy/schemas/public-profile-catalog.schema.json", "MODIFY"},
		{"deferred_agent_plan", "src/agent_contracts/example.yaml", "src/agent_contracts", "PLAN"},
	}
	for _, group := range []struct {
		name, operation string
		scopes          []string
	}{
		{"pp_automation", "MODIFY", []string{"developer/automation/project_profile_registry.py", "developer/automation/pp/pp_transition_delta.py", "developer/automation/pp/pp_transition_reconciler.py"}},
		{"runtime_registry", "MODIFY", []string{"src/ptsip/project_profile_contracts.py", "src/ptsip/profile_identity.py", "src/ptsip/profile_compatibility.py", "src/ptsip/specdata/project-profile-contracts.yaml"}},
		{"distribution", "MODIFY", []string{"setup.py", "MANIFEST.in", ".github/scripts/verify_distribution_contracts.py"}},
		{"h3_hooks", "MODIFY", []string{"developer/automation/dev_setup.py", "developer/automation/pp/pp_pre_commit.py", ".githooks/pre-commit", "setup_dev.bat", "bootstrap_repo.ps1"}},
		{"release_surface", "RELEASE", []string{".github/workflows/release.yml", ".github/scripts/verify_release_contract.py"}},
		{"local_profile", "MODIFY", []string{"src/ptsip/local_profile_catalog.py", "src/ptsip/profile_metadata.py"}},
	} {
		for _, scope := range group.scopes {
			cases = append(cases, struct{ name, scope, bindingScope, operation string }{group.name + "/" + scope, scope, scope, group.operation})
		}
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := resolver.Resolve(test.scope, test.operation)
			if err != nil {
				t.Fatal(err)
			}
			if result["binding_scope"] != test.bindingScope || result["projection_authority"] != false {
				t.Fatal(result)
			}
			want := expectedMigratedRefs(t, test.bindingScope, test.operation)
			if !reflect.DeepEqual(result["policies"], want) {
				t.Fatalf("Root responsibility selection differs\nactual=%#v\nexpected=%#v", result["policies"], want)
			}
		})
	}
}

func TestDuplicatePolicyIdentityIsABlockingBindingError(t *testing.T) {
	resolver := rootResolver(t)
	record := resolver.Bindings["src/ptsip/migration"]
	refs := record["operations"].(object)["PLAN"].([]any)
	record["operations"].(object)["PLAN"] = append(refs, refs[0])
	if err := resolver.ValidateBindings(); err == nil || !strings.Contains(err.Error(), "duplicate policy") {
		t.Fatal(err)
	}
}

func TestResolverInheritanceRequiresExplicitEligibility(t *testing.T) {
	for _, test := range []struct {
		name, scope, want string
		parent, child     bool
	}{{"skip_non_inheritable_ancestor", "developer/foo/bar.py", "developer", true, false}, {"exact_non_inheritable_scope", "developer/foo", "developer/foo", true, false}, {"all_non_inheritable", "developer/foo/bar.py", "", false, false}} {
		t.Run(test.name, func(t *testing.T) {
			resolver := rootResolver(t)
			refs := []any{object{"policy_id": "MPD-INFO-0001", "sections": []any{firstSection}}}
			resolver.Bindings = map[string]object{"developer": {"inherit_to_descendants": test.parent, "default_refs": refs}, "developer/foo": {"inherit_to_descendants": test.child, "default_refs": refs}}
			result, err := resolver.Resolve(test.scope, "READ")
			if test.want == "" {
				if err == nil || !strings.Contains(err.Error(), "no eligible policy binding") {
					t.Fatal(err)
				}
			} else if err != nil || result["binding_scope"] != test.want {
				t.Fatalf("%#v %v", result, err)
			}
		})
	}
}

func TestBindingSchemaRequiresBooleanInheritance(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, value := range []any{nil, "true"} {
		record := object{"scope": "synthetic/scope", "default_refs": []any{object{"policy_id": "MPD-INFO-0001", "sections": []any{firstSection}}}}
		if value != nil {
			record["inherit_to_descendants"] = value
		}
		if err := repo.Validate("developer/policy/schemas/policy-resolver-binding.schema.json", record); err == nil || !strings.Contains(err.Error(), "inherit_to_descendants") {
			t.Fatal(err)
		}
	}
}

func TestSimilarScopeNamesDoNotMatchOrAttachTaskContext(t *testing.T) {
	resolver := rootResolver(t)
	for _, scope := range []string{"src/ptsip/app/github_authority_extra.py", "src/ptsip-migrations/future_engine.py"} {
		result, err := resolver.Resolve(scope, "MODIFY")
		if err != nil {
			t.Fatal(err)
		}
		if result["binding_scope"] != "src" {
			t.Fatal(result)
		}
		if _, exists := result["task_context"]; exists {
			t.Fatal("unregistered task context attached")
		}
		if scope == "src/ptsip-migrations/future_engine.py" {
			ids := []any{}
			for _, raw := range result["policies"].([]any) {
				ids = append(ids, raw.(object)["policy_id"])
			}
			if !reflect.DeepEqual(ids, []any{"MPD-NORM-0001", "MPD-REAL-0004", "MPD-CNTR-0003"}) {
				t.Fatal(ids)
			}
		}
	}
}

func TestPolicyGetProjectsOnlyOneExactRootSection(t *testing.T) {
	result, err := rootResolver(t).Get("MPD-INFO-0001", firstSection)
	if err != nil {
		t.Fatal(err)
	}
	if result["fragment"] != "rules."+firstSection || result["canonical_path"] != "developer/policy/INFO/MPD-INFO-0001.yaml" {
		t.Fatal(result)
	}
	record := result["record"].(object)
	if _, exists := record["registry_resolution_budget"]; !exists {
		t.Fatal(record)
	}
	if _, exists := record["ptsip_design_priority"]; exists {
		t.Fatal("unselected section leaked")
	}
	if _, err := rootResolver(t).Get("MPD-SPEC-0006", "identity_and_resolution"); err == nil {
		t.Fatal("historical identity used as runtime fallback")
	}
}

func TestPolicyExplainReturnsCompactMetadataWithoutPolicyBody(t *testing.T) {
	result, err := rootResolver(t).Explain("MPD-INFO-0001")
	if err != nil {
		t.Fatal(err)
	}
	if result["policy_id"] != "MPD-INFO-0001" || result["status"] != "ACTIVE" {
		t.Fatal(result)
	}
	if _, exists := result["record"]; exists {
		t.Fatal("policy body leaked")
	}
	found := false
	for _, section := range result["rule_sections"].([]string) {
		if section == firstSection {
			found = true
		}
	}
	if !found {
		t.Fatal(result)
	}
}

func TestResolverRejectsUnknownOperationsEscapingScopesAndUnknownPolicies(t *testing.T) {
	resolver := rootResolver(t)
	for _, operation := range []string{"SEARCH", "GUESS", ""} {
		if _, err := resolver.Resolve("src/ptsip/migration/future_engine.py", operation); err == nil {
			t.Fatal("unknown operation admitted", operation)
		}
	}
	if _, err := resolver.Resolve("../outside-repository", "READ"); err == nil {
		t.Fatal("scope escape admitted")
	}
	if _, err := resolver.Get("MPD-9999", ""); err == nil {
		t.Fatal("unknown policy identity admitted")
	}
}

func TestNormativeRuleProjectionReadsOneMachineRegistryRecord(t *testing.T) {
	result, err := rootResolver(t).Rule("PTSIP-AUT-007")
	if err != nil {
		t.Fatal(err)
	}
	if result["schema_version"] != "ptsip-normative-rule-projection/v2" || result["canonical_source"] != "registry/ptsip-registry.yaml" || result["projection_authority"] != false || result["registry_record"].(object)["id"] != "PTSIP-AUT-007" {
		t.Fatal(result)
	}
	for _, field := range []string{"section_text", "line_start"} {
		if _, exists := result[field]; exists {
			t.Fatal("narrative projection leaked", field)
		}
	}
}

func TestResolverCLIProjectsOneRuleAndFailsCleanlyOnMalformedJSONL(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	result, err, output := testrepo.CLI(t, binary, testrepo.Root(t), "policy-resolver", "rule", "PTSIP-AUT-007")
	if err != nil || result["rule_id"] != "PTSIP-AUT-007" || strings.Contains(output, "## 10. Action-time synchronization") {
		t.Fatalf("%v\n%s", err, output)
	}
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyTree(t, repo, "developer/policy")
	testrepo.CopyFiles(t, repo, "pyproject.toml")
	if err := repo.AtomicWrite("developer/policy/policy-resolver-bindings/bindings.jsonl", []byte("{malformed}\n"), nil); err != nil {
		t.Fatal(err)
	}
	_, err, output = testrepo.CLI(t, binary, repo.Root, "policy-resolver", "resolve", "--scope", "src/ptsip/migration/engine.py", "--operation", "MODIFY")
	if err == nil || !strings.Contains(output, "binding line") || strings.Contains(output, "Traceback") {
		t.Fatalf("%v\n%s", err, output)
	}
}
