package contracts_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
)

func rulePolicy() object {
	return object{
		"Policy_Rules": object{
			"rules":      array{object{"id": "r.first", "content": object{"content_id": array{"field-name"}}, "RA_id": raID}},
			"evaluation": object{"state": "requested"},
		},
		"Policy_Content": object{"items": array{object{"id": "field-name", "value": object{"kind": "note"}}}},
	}
}

func TestCentralRegistryRegistersPlanRootWithoutItsChildren(t *testing.T) {
	config := readObject(t, filepath.Join(productRoot(t), "schemas", "config.json"))
	if len(config) != 3 || config["Policy"] == nil || config["plan"] == nil || config["storage"] == nil {
		t.Fatalf("unexpected central registry groups: %v", config)
	}
	policies := config["Policy"].(object)
	plans := config["plan"].(object)
	storage := config["storage"].(object)["Responsibility-Authority"].(object)
	if len(policies) != 1 || policies["ptsip.policy.v4"] == nil || len(storage) != 3 ||
		len(plans) != 1 || plans[registry.PlanDocumentURN] == nil {
		t.Fatalf("unexpected current contract registrations: %v", config)
	}
	for _, key := range []string{"RA_list", "RAC_list", "AEC_list"} {
		entry := storage[key].(object)
		schema := readObject(t, filepath.Join(productRoot(t), "schemas", filepath.FromSlash(entry["path"].(string))))
		if schema["$id"] != entry["$id"] {
			t.Errorf("resource identity differs for %s", key)
		}
	}
	plan := plans[registry.PlanDocumentURN].(object)
	if plan["$id"] != registry.PlanDocumentURN || plan["version"] != "SCH.1.00" ||
		plan["path"] != "plan/public-structure-plan.schema.json" {
		t.Fatalf("unexpected Plan root: %v", plan)
	}
	v := validator(t)
	identity := object{"plan_id": "plan-A123"}
	instance := object{
		"SchemaVersion": "SCH.1.00", "Plan_Identity": identity,
		"Plan_lifecycle": object{"status": "Draft.Proposed", "completion_date": nil},
	}
	if issues := v.StructureIssues(registry.PlanDocumentURN, instance); len(issues) != 0 {
		t.Fatalf("valid Plan rejected: %+v", issues)
	}
	identity["plan_id"] = "plan-AA"
	if issues := v.StructureIssues(registry.PlanDocumentURN, instance); len(issues) == 0 {
		t.Fatal("Plan registration failed to validate the Identity child")
	}
}

func TestUserIDAndStateContractsAreIndependent(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	writeContract(t, project, "rules_patterns.json", object{"$schema": dialect, "pattern": `^r\.`})
	writeContract(t, project, "policy_content_id.json", object{"$schema": dialect, "pattern": "^[a-z-]+$"})
	writeContract(t, project, "evaluation_state.json", object{"$schema": dialect, "const": "requested"})
	policy := rulePolicy()
	results, err := v.ValidateUserContracts(policy, project, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Status != "VALID" {
			t.Fatalf("independent valid constraints failed: %+v", result)
		}
	}
	rule := policy["Policy_Rules"].(object)["rules"].(array)[0].(object)
	rule["content"].(object)["content_id"] = array{"r.first"}
	results, err = v.ValidateUserContracts(policy, project, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexed := contractsByName(results)
	if indexed["rules_patterns.json"].Status != "VALID" || indexed["policy_content_id.json"].Status != "INVALID" {
		t.Fatalf("rule and content patterns were mixed: %v", indexed)
	}
	want := []any{"Policy_Rules", "rules", 0, "content", "content_id", 0}
	if !reflect.DeepEqual(indexed["policy_content_id.json"].Issues[0].Path, want) {
		t.Fatalf("wrong constraint target path: %v", indexed["policy_content_id.json"].Issues)
	}
}

func TestBootstrapCreatesOnlyNeededNeutralContracts(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	policy := object{"Policy_Rules": object{"evaluation": object{"state": false}}}
	results, err := v.ValidateUserContracts(policy, project, false, nil)
	if err != nil || len(results) != 1 || results[0].Status != "UNCONFIGURED" {
		t.Fatalf("missing contract result: %v, %v", results, err)
	}
	if _, err := os.Stat(contractRoot(project)); !os.IsNotExist(err) {
		t.Fatalf("read-only validation created contracts: %v", err)
	}
	results, err = v.ValidateUserContracts(policy, project, true, nil)
	if err != nil || len(results) != 1 || results[0].Status != "UNCONFIGURED" {
		t.Fatalf("neutral bootstrap result: %v, %v", results, err)
	}
	entries, err := os.ReadDir(contractRoot(project))
	if err != nil || len(entries) != 1 || entries[0].Name() != "evaluation_state.json" {
		t.Fatalf("bootstrap created unrelated contracts: %v, %v", entries, err)
	}
	if got := readObject(t, filepath.Join(contractRoot(project), "evaluation_state.json")); !reflect.DeepEqual(got, object{"$schema": dialect}) {
		t.Fatalf("neutral template imposed constraints: %v", got)
	}
	results, err = v.ValidateUserContracts(object{}, project, true, nil)
	if err != nil || len(results) != 0 {
		t.Fatalf("unused fields acquired a contract: %v, %v", results, err)
	}
}

func TestBootstrapIsAtomicAndIdempotentAcrossConcurrentCalls(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	var tasks sync.WaitGroup
	failures := make(chan error, 12)
	for index := 0; index < 12; index++ {
		tasks.Add(1)
		go func() {
			defer tasks.Done()
			results, err := v.ValidateUserContracts(rulePolicy(), project, true, nil)
			if err != nil {
				failures <- err
				return
			}
			if len(results) != 3 {
				t.Errorf("concurrent bootstrap result contains %d contracts", len(results))
			}
			for _, result := range results {
				if result.Status != "UNCONFIGURED" {
					t.Errorf("partial template became visible: %+v", result)
				}
			}
		}()
	}
	tasks.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	entries, err := os.ReadDir(contractRoot(project))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
		if got := readObject(t, filepath.Join(contractRoot(project), entry.Name())); !reflect.DeepEqual(got, object{"$schema": dialect}) {
			t.Fatalf("incomplete neutral template %s: %v", entry.Name(), got)
		}
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"evaluation_state.json", "policy_content_id.json", "rules_patterns.json"}) {
		t.Fatalf("unexpected bootstrap files: %v", names)
	}
}

func TestBadExistingUserContractsAreReportedAndPreserved(t *testing.T) {
	v := validator(t)
	cases := []struct{ payload, status, code string }{
		{"", "INVALID", "CONTRACT_SYNTAX"},
		{"[", "INVALID", "CONTRACT_SYNTAX"},
		{`{"enum":[1],"enum":[2]}`, "INVALID", "CONTRACT_SYNTAX"},
		{`{"pattern":4}`, "INVALID", "CONTRACT_SCHEMA"},
		{`{"properties":[]}`, "INVALID", "CONTRACT_SCHEMA"},
		{`{"pattern":"a{99999999999999999999999}"}`, "INVALID", "CONTRACT_SCHEMA"},
		{`{"$schema":"https://json-schema.org/draft-07/schema"}`, "UNSUPPORTED", "CONTRACT_DIALECT"},
		{`{"$defs":{"legacy":{"$id":"urn:legacy","$schema":"http://json-schema.org/draft-07/schema#"}}}`, "UNSUPPORTED", "CONTRACT_DIALECT"},
	}
	for _, test := range cases {
		t.Run(test.code+"_"+test.payload, func(t *testing.T) {
			project := t.TempDir()
			path := writeContract(t, project, "evaluation_state.json", object{})
			if err := os.WriteFile(path, []byte(test.payload), 0o644); err != nil {
				t.Fatal(err)
			}
			results, err := v.ValidateUserContracts(rulePolicy(), project, true, nil)
			if err != nil {
				t.Fatal(err)
			}
			result := contractsByName(results)["evaluation_state.json"]
			if result.Status != test.status || !hasCode(result.Issues, test.code) {
				t.Fatalf("unexpected contract error: %+v", result)
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, []byte(test.payload)) {
				t.Fatal("existing user contract was overwritten")
			}
		})
	}
}

func TestExplicitUserBindingsAndLocalSchemaResources(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	writeContract(t, project, "unspecified.json", object{"const": 8})
	writeContract(t, project, "wrong-target.json", object{"x-ptsip-target": "Policy_Identity.missing", "const": 8})
	writeContract(t, project, "vocabulary.json", object{"$defs": object{"state": object{"enum": array{"requested"}}}})
	writeContract(t, project, "evaluation_state.json", object{"$ref": "vocabulary.json#/$defs/state"})
	writeContract(t, project, "require-rules.json", object{"x-ptsip-target": "Policy_Rules", "required": array{"rules"}})
	results, err := v.ValidateUserContracts(rulePolicy(), project, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexed := contractsByName(results)
	for _, name := range []string{"evaluation_state.json", "require-rules.json"} {
		if indexed[name].Status != "VALID" {
			t.Fatalf("local reference or explicit binding failed: %+v", indexed[name])
		}
	}
	for _, name := range []string{"unspecified.json", "wrong-target.json", "vocabulary.json"} {
		if indexed[name].Status != "UNBOUND" {
			t.Fatalf("undeclared binding must be reported: %+v", indexed[name])
		}
	}
}

func TestUserContractsRejectMissingExternalAndUnvisitedReferences(t *testing.T) {
	v := validator(t)
	cases := []any{
		object{"$ref": "missing.json"}, object{"$ref": "../outside.json"},
		object{"$ref": "https://example.invalid/contract.json"},
		object{"anyOf": array{true, object{"$ref": "missing.json"}}},
	}
	for index, schema := range cases {
		project := t.TempDir()
		writeContract(t, project, "evaluation_state.json", schema)
		if err := os.WriteFile(filepath.Join(filepath.Dir(contractRoot(project)), "outside.json"), []byte(`{"const":"requested"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		results, err := v.ValidateUserContracts(rulePolicy(), project, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		result := contractsByName(results)["evaluation_state.json"]
		if result.Status != "INVALID" || !hasCode(result.Issues, "CONTRACT_REFERENCE") {
			t.Fatalf("reference case %d passed: %+v", index, result)
		}
	}
}

func TestUserCanonicalResourceIDsAndAmbiguousEmbeddedIDs(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	base := "https://schema.example.invalid/contracts/"
	writeContract(t, project, "evaluation_state.json", object{"$id": base + "state", "$ref": "vocabulary"})
	writeContract(t, project, "vocabulary.json", object{"$id": base + "vocabulary", "const": "requested"})
	results, err := v.ValidateUserContracts(rulePolicy(), project, false, nil)
	if err != nil || contractsByName(results)["evaluation_state.json"].Status != "VALID" {
		t.Fatalf("registered canonical resource IDs should resolve locally: %v, %v", results, err)
	}
	project = t.TempDir()
	writeContract(t, project, "evaluation_state.json", object{"$ref": "urn:test:shared"})
	for name, value := range map[string]string{"one.json": "requested", "two.json": "other"} {
		writeContract(t, project, name, object{"$defs": object{"state": object{"$id": "urn:test:shared", "const": value}}})
	}
	results, err = v.ValidateUserContracts(rulePolicy(), project, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexed := contractsByName(results)
	for _, name := range []string{"evaluation_state.json", "one.json", "two.json"} {
		if indexed[name].Status != "INVALID" || !hasCode(indexed[name].Issues, "CONTRACT_REFERENCE") {
			t.Fatalf("ambiguous nested resource was silently selected: %+v", indexed[name])
		}
	}
}

func TestUserBindingsReachTupleFields(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	writeContract(t, project, "date.json", object{"x-ptsip-target": "Policy_Lifecycle.evaluation[].last_changed_date", "pattern": "^2026"})
	policy := object{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "20251117"}, object{}}}}
	results, err := v.ValidateUserContracts(policy, project, false, nil)
	if err != nil || len(results) != 1 || results[0].Status != "INVALID" {
		t.Fatalf("tuple field constraint failed to apply: %v, %v", results, err)
	}
	want := []any{"Policy_Lifecycle", "evaluation", 0, "last_changed_date"}
	if !reflect.DeepEqual(results[0].Issues[0].Path, want) {
		t.Fatalf("unexpected tuple path: %v", results[0].Issues)
	}
}

func TestContentReferencesAreExactAndDocumentLocal(t *testing.T) {
	v := validator(t)
	policy := rulePolicy()
	assertValid(t, evaluate(t, v, "validation.policy.rules", object{"policy": policy}))
	policy["Policy_Content"].(object)["items"] = array{object{"id": "field-name"}, object{"id": "field-name"}}
	result := evaluate(t, v, "validation.policy.rules", object{"policy": policy})
	if result.Valid || !hasCode(result.Diagnostics, "DUPLICATE_ID") || !hasCode(result.Diagnostics, "UNKNOWN_ID") {
		t.Fatalf("ambiguous document-local content should fail: %+v", result)
	}
	policy["Policy_Content"].(object)["items"] = array{object{"id": "Field-name"}}
	result = evaluate(t, v, "validation.policy.rules", object{"policy": policy})
	if result.Valid || !hasCode(result.Diagnostics, "UNKNOWN_ID") {
		t.Fatalf("case-folded content reference accepted: %+v", result)
	}
}

func TestScopedUnboundContractsRemainVisibleOnlyToTheirOwner(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	writeContract(t, project, "missing-target.json", object{"x-ptsip-target": "Policy_Identity.not_existing", "const": "rejected"})
	identity := evaluate(t, v, "validation.policy.identity", object{"policy": object{"Policy_Identity": object{}}, "project_root": project})
	if identity.Valid || len(identity.UserContracts) != 1 || identity.UserContracts[0].Status != "UNBOUND" {
		t.Fatalf("owned unbound contract disappeared: %+v", identity)
	}
	rules := evaluate(t, v, "validation.policy.rules", object{"policy": object{"Policy_Rules": object{}}, "project_root": project})
	assertValid(t, rules)
	if len(rules.UserContracts) != 0 {
		t.Fatalf("unrelated identity contract entered Rules output: %v", rules.UserContracts)
	}
}

func TestBootstrapRejectsUserRootAliases(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(project, ".ptsip")); err != nil {
		t.Skipf("host cannot create a directory symlink: %v", err)
	}
	_, err := v.ValidateUserContracts(rulePolicy(), project, true, nil)
	if err == nil {
		t.Fatal("user contract root escaped the selected project")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("outside target was mutated: %v, %v", entries, err)
	}
}

func TestUserContractBooleanSchemasAndArbitraryStates(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	policy := object{"Policy_Rules": object{"evaluation": object{"state": object{"custom": array{false, 2, "unknown meaning"}}}}}
	path := writeContract(t, project, "evaluation_state.json", false)
	results, err := v.ValidateUserContracts(policy, project, false, nil)
	if err != nil || contractsByName(results)["evaluation_state.json"].Status != "INVALID" {
		t.Fatalf("false schema did not reject the value: %v, %v", results, err)
	}
	payload, _ := json.Marshal(object{"const": policy["Policy_Rules"].(object)["evaluation"].(object)["state"]})
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	results, err = v.ValidateUserContracts(policy, project, false, nil)
	if err != nil || contractsByName(results)["evaluation_state.json"].Status != "VALID" {
		t.Fatalf("user-defined state constant was rejected: %v, %v", results, err)
	}
}

func TestUserPatternsSupportLookaheadAndBackreferences(t *testing.T) {
	v := validator(t)
	cases := []struct{ name, pattern, accepted, rejected string }{
		{"lookahead", `^(?!forbidden$)[a-z]+$`, "ready", "forbidden"},
		{"backreference", `^([a-z]+)-\1$`, "ready-ready", "ready-other"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			project := t.TempDir()
			writeContract(t, project, "evaluation_state.json", object{"$schema": dialect, "pattern": test.pattern})
			policy := object{"Policy_Rules": object{"evaluation": object{"state": test.accepted}}}
			results, err := v.ValidateUserContracts(policy, project, false, nil)
			if err != nil || contractsByName(results)["evaluation_state.json"].Status != "VALID" {
				t.Fatalf("user pattern rejected supported syntax/value: %v, %v", results, err)
			}
			policy["Policy_Rules"].(object)["evaluation"].(object)["state"] = test.rejected
			results, err = v.ValidateUserContracts(policy, project, false, nil)
			if err != nil || contractsByName(results)["evaluation_state.json"].Status != "INVALID" {
				t.Fatalf("user pattern failed to constrain the value: %v, %v", results, err)
			}
		})
	}
}
