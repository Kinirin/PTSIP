package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	usercontracts "github.com/Kinirin/PTSIP/src/ptsip/common_contract/user_contracts"
)

type object = map[string]any
type array = []any

const dialect = "https://json-schema.org/draft/2020-12/schema"

var lifecycleStates = []string{
	"DRAFT.Default", "DRAFT.Reviewed", "DRAFT.Denied", "DRAFT.Pass",
	"APPROVED.Default", "APPROVED.Applying", "APPROVED.Application_Problems",
	"APPROVED.Unresolved", "APPROVED.Activate", "APPROVED.provisional",
	"RETIRED.Default", "RETIRED.Removable", "RETIRED.Waiting",
}

func productRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source location unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "ptsip"))
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(filepath.Dir(productRoot(t)))
}

func validator(t *testing.T) *usercontracts.Validator {
	t.Helper()
	v, err := usercontracts.New(productRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func readObject(t *testing.T, path string) object {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value object
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return value
}

func clone(t *testing.T, value object) object {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var cloned object
	if err := json.Unmarshal(data, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func contractRoot(project string) string {
	return filepath.Join(project, ".ptsip", "contract", "user")
}

func writeContract(t *testing.T, project, name string, schema any) string {
	t.Helper()
	root := contractRoot(project)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func evaluate(t *testing.T, v *usercontracts.Validator, operation string, input object) usercontracts.Result {
	t.Helper()
	result, err := v.EvaluateOperation(operation, input)
	if err != nil {
		t.Fatalf("%s: %v", operation, err)
	}
	if !result.ExecutionAuthorized || result.PTSIPConformanceVerdict != "NOT_ASSERTED" {
		t.Fatalf("unexpected operation authorization or conformance claim: %+v", result)
	}
	return result
}

func assertValid(t *testing.T, result usercontracts.Result) {
	t.Helper()
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("expected valid result, got %+v", result)
	}
}

func hasCode(issues []usercontracts.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func contractsByName(results []usercontracts.ContractResult) map[string]usercontracts.ContractResult {
	indexed := map[string]usercontracts.ContractResult{}
	for _, result := range results {
		if result.Name != nil {
			indexed[*result.Name] = result
		}
	}
	return indexed
}

func TestPolicyRootReferencesFourIndependentSchemas(t *testing.T) {
	root := filepath.Join(productRoot(t), "schemas", "policy")
	paths := map[string]string{
		"Policy_Identity":  "identity/PTSIP_policy_identity.schema.json",
		"Policy_Lifecycle": "lifecycle/PTSIP_policy_lifecycle.schema.json",
		"Policy_Rules":     "rules/PTSIP_policy_rules.schema.json",
		"Policy_Content":   "content/PTSIP_policy_content.schema.json",
	}
	schema := readObject(t, filepath.Join(root, "public-structure-policy.schema.json"))
	properties := schema["properties"].(object)
	if len(properties) != len(paths) {
		t.Fatalf("expected four root fields, got %v", properties)
	}
	ids := map[string]bool{schema["$id"].(string): true}
	for field, relative := range paths {
		component := readObject(t, filepath.Join(root, filepath.FromSlash(relative)))
		id := component["$id"].(string)
		if ids[id] {
			t.Fatalf("duplicate schema identity %s", id)
		}
		ids[id] = true
		if !reflect.DeepEqual(properties[field], object{"$ref": id}) {
			t.Fatalf("%s must reference its independent schema: %v", field, properties[field])
		}
	}
	v := validator(t)
	if issues := v.StructureIssues("ptsip.policy.v4", object{
		"Policy_Identity": object{}, "Policy_Lifecycle": object{}, "Policy_Rules": object{}, "Policy_Content": object{},
	}); len(issues) != 0 {
		t.Fatal(issues)
	}
}

func TestAllThirteenLifecycleStates(t *testing.T) {
	v := validator(t)
	for _, state := range lifecycleStates {
		t.Run(state, func(t *testing.T) {
			if issues := v.StructureIssues("ptsip.policy.v4", object{"Policy_Lifecycle": object{"status": state}}); len(issues) != 0 {
				t.Fatal(issues)
			}
		})
	}
}

func TestLifecycleEvaluationSecondDescriptionIsOptional(t *testing.T) {
	v := validator(t)
	for _, description := range []object{{}, {"description": ""}, {"description": strings.Repeat("a", 200)}} {
		if issues := v.StructureIssues("ptsip.policy.v4", object{"Policy_Lifecycle": object{
			"evaluation": array{object{"last_changed_date": "20261117"}, description},
		}}); len(issues) != 0 {
			t.Fatal(issues)
		}
	}
}

func TestPolicyRejectsRemovedFieldsAndInvalidShapes(t *testing.T) {
	v := validator(t)
	cases := []object{
		{"Policy_Authority": object{}}, {"Policy_Evaluation": object{}},
		{"Policy_Relationships": object{}}, {"Policy_Migration": object{}},
		{"Policy_Identity": object{"root_family": "ARCH"}},
		{"Policy_Identity": object{"policy_kind": "local"}},
		{"Policy_Identity": object{"revision": 2}},
		{"Policy_Lifecycle": object{"status": "APPROVED.Provisional"}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "20261117"}}}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"description": ""}, object{"last_changed_date": "20261117"}}}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "2026-11-17"}, object{}}}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "20261301"}, object{}}}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "20261117"}, object{"description": strings.Repeat("a", 201)}}}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "20261117"}, object{}, object{}}}},
		{"Policy_Lifecycle": object{"evaluation": array{object{"last_changed_date": "20261117", "approval_ref": "x"}, object{}}}},
		{"Policy_Rules": object{"rules": array{object{"id_RA": "R-ID-" + strings.Repeat("A", 32)}}}},
		{"Policy_Rules": object{"rules": array{object{"RA_id": "R-ID-" + strings.Repeat("a", 32)}}}},
		{"Policy_Rules": object{"evaluation": object{"source_refs": array{"source"}}}},
		{"Policy_Rules": object{"evaluation": object{"evidence_refs": array{"item"}}}},
		{"Policy_Rules": object{"evaluation": object{"evaluated_at": "20261117"}}},
		{"Policy_Rules": object{"evaluation": object{"evaluation_scope": "all"}}},
		{"Policy_Content": object{"source_document": object{}}},
		{"Policy_Content": object{"items": array{object{"source_pointer": "/source"}}}},
		{"Policy_Content": object{"items": array{object{"path": "item.txt"}}}},
		{"Policy_Content": object{"items": array{object{"sha256": strings.Repeat("a", 64)}}}},
		{"Policy_Content": object{"items": array{object{"revision_ref": "previous"}}}},
	}
	for index, payload := range cases {
		if len(v.StructureIssues("ptsip.policy.v4", payload)) == 0 {
			t.Errorf("case %d accepted removed field or malformed shape: %v", index, payload)
		}
	}
}

func TestContentHasNoDefaultRequirednessOrGlobalPattern(t *testing.T) {
	v := validator(t)
	policy := object{"Policy_Content": object{"items": array{object{}, object{"id": "project / label"}, object{"value": array{nil, object{"mixed": true}}}}}}
	if issues := v.StructureIssues("ptsip.policy.v4", policy); len(issues) != 0 {
		t.Fatal(issues)
	}
}

func TestOfficialAndUserVersionContracts(t *testing.T) {
	v := validator(t)
	for _, version := range []string{"1.2.3", "ver1.2.3", "v1.2.3alpha", "Ver.1.2.3.beta", "V.1.2.3-Hotfix", "1.2.3-rc+4.5", "1.2.3.4", "1.2.3.4.5"} {
		t.Run(version, func(t *testing.T) {
			if !v.OfficialVersionValid(version) {
				t.Fatalf("approved official version rejected: %s", version)
			}
			assertValid(t, evaluate(t, v, "validation.policy.identity", object{"policy": object{"Policy_Identity": object{"version": version}}}))
		})
	}
	for _, version := range []any{"1.2", "1.2.3-Beta", "1.2.3-unknown", "1.2.3\n", 123} {
		if v.OfficialVersionValid(version) {
			t.Errorf("unapproved official version accepted: %v", version)
		}
		if result := evaluate(t, v, "validation.policy.identity", object{"policy": object{"Policy_Identity": object{"version": version}}}); result.Valid {
			t.Errorf("invalid version passed without a user contract: %v", version)
		}
	}
	project := t.TempDir()
	custom := object{"release_train": "winter"}
	writeContract(t, project, "my-version.json", object{"$schema": dialect, "x-ptsip-target": "Policy_Identity.version", "const": custom})
	for _, version := range []any{"v1.2.3", custom} {
		assertValid(t, evaluate(t, v, "validation.policy.identity", object{"policy": object{"Policy_Identity": object{"version": version}}, "project_root": project}))
	}
	if result := evaluate(t, v, "validation.policy.identity", object{"policy": object{"Policy_Identity": object{"version": object{"release_train": "summer"}}}, "project_root": project}); result.Valid {
		t.Fatal("version outside both alternatives was accepted")
	}
}

func TestOmittedVersionAndCategoryDoNotCreateContracts(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	assertValid(t, evaluate(t, v, "validation.policy.identity", object{
		"policy": object{"Policy_Identity": object{"title": "Example"}}, "project_root": project, "bootstrap_user_contracts": true,
	}))
	if _, err := os.Stat(filepath.Join(project, ".ptsip")); !os.IsNotExist(err) {
		t.Fatalf("unused contracts must not be created: %v", err)
	}
	policy := object{"Policy_Identity": object{"Policy_Category": object{"team": "release", "scope": "Artifacts"}}}
	results, err := v.ValidateUserContracts(policy, project, true, nil)
	if err != nil || len(results) != 1 || results[0].Status != "UNCONFIGURED" {
		t.Fatalf("missing user category contract should be explicit: %v, %v", results, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".ptsip")); !os.IsNotExist(err) {
		t.Fatal("category must not acquire an invented default contract")
	}
}

func TestUserCategoryFieldNamesAndRequiredRoles(t *testing.T) {
	v := validator(t)
	project := t.TempDir()
	writeContract(t, project, "categories.json", object{
		"$schema": dialect, "x-ptsip-target": "Policy_Identity.Policy_Category",
		"x-ptsip-category-roles": object{"CATEGORY_NAMESPACE": "team", "NAMESPACE_DEFINITION": "scope_summary"},
		"type":                   "object", "properties": object{"team": object{"type": "string"}, "scope_summary": object{"type": "string"}}, "additionalProperties": false,
	})
	policy := object{"Policy_Identity": object{"Policy_Category": object{"team": "release", "scope_summary": "Artifact preparation"}}}
	assertValid(t, evaluate(t, v, "validation.policy.identity", object{"policy": policy, "project_root": project}))
	delete(policy["Policy_Identity"].(object)["Policy_Category"].(object), "scope_summary")
	result := evaluate(t, v, "validation.policy.identity", object{"policy": policy, "project_root": project})
	if result.Valid || !hasCode(result.Diagnostics, "CATEGORY_ROLE") {
		t.Fatalf("missing approved semantic role was accepted: %+v", result)
	}
}

func TestDevelopmentCheckoutCannotReceiveUserContractBootstrap(t *testing.T) {
	v := validator(t)
	checkout := t.TempDir()
	product := filepath.Join(checkout, "src", "ptsip")
	if err := os.MkdirAll(filepath.Join(product, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(product, "CLI.go"), filepath.Join(product, "schemas", "config.json")} {
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	project := filepath.Join(checkout, "contained-project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := v.ValidateUserContracts(object{"Policy_Content": object{"items": array{object{"id": "a"}}}}, project, true, nil)
	if err == nil {
		t.Fatal("development checkout accepted user contract creation")
	}
	for _, root := range []string{checkout, project} {
		if _, err := os.Stat(filepath.Join(root, ".ptsip")); !os.IsNotExist(err) {
			t.Fatalf("development instance was mutated: %s", root)
		}
	}
}
