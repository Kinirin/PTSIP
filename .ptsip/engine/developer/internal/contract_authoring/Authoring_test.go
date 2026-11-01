package contract_authoring_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

func repository(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func nativeCatalog(t *testing.T) (*authoring.Catalog, authoring.Validator) {
	t.Helper()
	catalog, err := authoring.Load(repository(t))
	if err != nil {
		t.Fatal(err)
	}
	validate, err := schema_check.InstalledValidator()
	if err != nil {
		t.Skipf("installed PTSIP validation engine unavailable: %v", err)
	}
	return catalog, validate
}

func doc(raw string) authoring.Document {
	var result authoring.Document
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		panic(err)
	}
	return result
}

func policyFixture(t *testing.T, catalog *authoring.Catalog, path string) authoring.Document {
	t.Helper()
	policy, err := authoring.ReadDocument(catalog.Root, path)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func copyDocument(input authoring.Document) authoring.Document {
	raw, _ := json.Marshal(input)
	return doc(string(raw))
}

func profileFixture(t *testing.T, catalog *authoring.Catalog) authoring.Document {
	t.Helper()
	profile, err := authoring.ReadDocument(catalog.Root, authoring.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	delete(profile, "component_map")
	delete(profile, "component_bindings")
	catalog.Definitions = doc(`{"schema_version":"ptsip-responsibility-authority/v1","condition_list":[{"ID":"C-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","name":"REVIEW-COMPLETE"}],"authority_list":[{"ID":"OA-ID-edit","name":"Edit policy","action":"edit","scope":"example"}]}`)
	profile["responsibility"] = doc(`{"items":[{"name":"Policy editor","ID":"R-ID-BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB","condition_id":["C-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"],"Operational_Authority":["OA-ID-edit"],"role_name":"Editor","runtime_required":false}]}`)["items"]
	profile["component_bindings"] = authoring.Document{}
	return profile
}

func TestRegisteredContractsAndAllFourteenSelections(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	if err := catalog.CheckContracts(validate); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Families) != 14 {
		t.Fatalf("expected 14 families, got %d", len(catalog.Families))
	}
	if catalog.Common["$id"] != "urn:ptsip:candidate:contract:policy:public-structure:v2" {
		t.Fatal("authoring did not load the registered public v2 schema")
	}
	for _, family := range catalog.Families {
		t.Run(family, func(t *testing.T) {
			// This probe checks explicit Family dispatch and preservation. Actual
			// schema validation uses the source policy fixtures in the regression
			// below; it does not assume one Family's semantics fit another.
			input := policyFixture(t, catalog, "developer/policy/ARCH/MPD-ARCH-0001.yaml")
			input["Policy_Identity"].(map[string]any)["root_family"] = family
			before := copyDocument(input)
			calls := 0
			probe := func(schema, instance any, resources []any) (bool, error) {
				calls++
				if !reflect.DeepEqual(schema, catalog.Common) || !reflect.DeepEqual(instance, before) {
					t.Fatal("authoring passed an inferred shape or changed the supplied document")
				}
				return true, nil
			}
			generated, status, err := catalog.GenerateRules(probe, input, family)
			if err != nil || status != "GENERATED" || calls != 1 {
				t.Fatalf("explicit selection failed: %s %d %v", status, calls, err)
			}
			if !reflect.DeepEqual(input, before) || !reflect.DeepEqual(generated, before) {
				t.Fatal("generation synthesized or changed policy content")
			}
		})
	}
}

func TestConditionalDefaultsAndInvalidSelections(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	empty, status, err := catalog.GenerateRules(validate, authoring.Document{}, "")
	if err != nil || status != "OMITTED" || len(empty) != 0 {
		t.Fatalf("default must omit: %v %s %v", empty, status, err)
	}
	for _, input := range []authoring.Document{
		authoring.Document{},
		doc(`{"Policy_Rules":{}}`),
		doc(`{"Policy_Identity":false}`),
		doc(`{"Policy_Identity":{"root_family":"UNKNOWN"}}`),
		doc(`{"Policy_Identity":{"root_family":false}}`),
		doc(`{"Policy_Identity":{"root_family":"GOV"}}`),
		doc(`{"Policy_Identity":{"root_family":"GOV"},"Policy_Rules":{}}`),
	} {
		if err := catalog.CheckRules(validate, input); err == nil {
			t.Fatalf("invalid or incomplete policy accepted: %v", input)
		}
	}
	conflict := doc(`{"Policy_Identity":{"root_family":"INFO"}}`)
	if _, _, err := catalog.GenerateRules(validate, conflict, "GOV"); err == nil || !strings.Contains(err.Error(), "ROOT_FAMILY_CONFLICT") {
		t.Fatal("conflicting selection accepted", err)
	}
	if _, _, err := catalog.GenerateRules(validate, authoring.Document{}, "GOV"); err == nil || !strings.Contains(err.Error(), "DECLARED_ROOT_FAMILY_REQUIRED") {
		t.Fatal("Family declaration or required values were invented", err)
	}
	if _, _, err := catalog.GenerateRules(validate, authoring.Document{}, "UNKNOWN"); err == nil || !strings.Contains(err.Error(), "UNKNOWN_ROOT_FAMILY") {
		t.Fatal("unregistered requested Family accepted", err)
	}
}

func TestProfileReferencesTypesAndExtensibleFields(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	valid := profileFixture(t, catalog)
	valid["custom_profile_note"] = "allowed"
	item := valid["responsibility"].([]any)[0].(map[string]any)
	item["custom_field"] = "allowed"
	if err := catalog.CheckProfile(validate, valid); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(authoring.Document)
	}{
		{"unknown_condition", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["condition_id"] = []any{"C-ID-CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"}
		}},
		{"unknown_authority", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["Operational_Authority"] = []any{"OA-ID-missing"}
		}},
		{"duplicate_responsibility_id", func(p authoring.Document) {
			entries := p["responsibility"].([]any)
			p["responsibility"] = append(entries, entries[0])
		}},
		{"duplicate_condition_ref", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["condition_id"] = []any{"C-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "C-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
		}},
		{"missing_role", func(p authoring.Document) { delete(p["responsibility"].([]any)[0].(map[string]any), "role_name") }},
		{"draft_boolean", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["runtime_required"] = "true/false"
		}},
		{"schema_instead_of_id", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["ID"] = doc(`{"type":"string"}`)
		}},
		{"invalid_id", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["ID"] = "R-ID-invalid/segment"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, _ := json.Marshal(valid)
			candidate := doc(string(encoded))
			tc.mutate(candidate)
			if err := catalog.CheckProfile(validate, candidate); err == nil {
				t.Fatal("invalid profile accepted")
			}
		})
	}
	conditions := catalog.Definitions["condition_list"].([]any)
	catalog.Definitions["condition_list"] = append(conditions, conditions[0])
	if err := catalog.CheckProfile(validate, valid); err == nil || !strings.Contains(err.Error(), "DUPLICATE_OR_EMPTY_ID") {
		t.Fatal("duplicate condition ID accepted", err)
	}
}

func TestPolicyRulesAuthoringRejectsLegacyFourBlocks(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	input := policyFixture(t, catalog, "developer/policy/GOV/MPD-GOV-0001.yaml")
	input["Policy_Rules"] = doc(`{"Identity":{"contract_id":"urn:ptsip:policy:rule:gov:example","Rule_ver":"1"},"Definition":[{"id":"R-ID-8D95436233BE6DCD5E90B2AAA986052C","rule":"FLATTEN-POLICY-STATU"}],"Boundary":{"scope":"Example scope"},"Evaluation":{"criteria":{"evidence":["Legacy evidence"]}}}`)
	before := copyDocument(input)
	if _, _, err := catalog.GenerateRules(validate, input, "GOV"); err == nil {
		t.Fatal("retired four-block policy rules were accepted")
	}
	if !reflect.DeepEqual(input, before) {
		t.Fatal("failed generation altered the source document")
	}
}

func TestPolicyRulesAuthoringPropagatesPublicSchemaRejection(t *testing.T) {
	catalog, _ := nativeCatalog(t)
	input := policyFixture(t, catalog, "developer/policy/ARCH/MPD-ARCH-0001.yaml")
	before := copyDocument(input)
	reject := func(schema, instance any, resources []any) (bool, error) {
		if !reflect.DeepEqual(schema, catalog.Common) || !reflect.DeepEqual(instance, before) {
			t.Fatal("policy authoring did not validate the complete supplied policy")
		}
		return false, nil
	}
	if _, _, err := catalog.GenerateRules(reject, input, "ARCH"); err == nil || !strings.Contains(err.Error(), "SCHEMA_REJECTED") {
		t.Fatal("public schema rejection was ignored", err)
	}
	if !reflect.DeepEqual(input, before) {
		t.Fatal("failed generation changed the input")
	}
	catalog.Common = nil
	if err := catalog.CheckRules(reject, input); err == nil || !strings.Contains(err.Error(), "REGISTERED_POLICY_SCHEMA_REQUIRED") {
		t.Fatal("missing registered public schema accepted", err)
	}
}

func TestNoOverwriteAndInvalidPaths(t *testing.T) {
	root := t.TempDir()
	if err := authoring.WriteDocument(root, "output.json", authoring.Document{}); err != nil {
		t.Fatal(err)
	}
	if err := authoring.WriteDocument(root, "output.json", doc(`{"changed":true}`)); err == nil {
		t.Fatal("existing output overwritten")
	}
	for _, path := range []string{"../outside.json", "a/../b.json", "a\\b.json", "a:b.json", "/absolute.json"} {
		if err := authoring.WriteDocument(root, path, authoring.Document{}); err == nil {
			t.Fatalf("unsafe output accepted: %s", path)
		}
		if _, err := authoring.LocalFile(root, path); err == nil {
			t.Fatalf("unsafe input accepted: %s", path)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "duplicate.yaml"), []byte("responsibility: []\nresponsibility: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := authoring.ReadDocument(root, "duplicate.yaml"); err == nil || !strings.Contains(err.Error(), "duplicate mapping key") {
		t.Fatal("duplicate YAML key accepted", err)
	}
	for _, extension := range []string{".json", ".yaml", ".yml"} {
		name := "roundtrip" + extension
		value := doc(`{"runtime_required":false,"ID":"R-ID-example","conditions":["C-ID-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"],"name":"책임 매칭 🧭"}`)
		if err := authoring.WriteDocument(root, name, value); err != nil {
			t.Fatal(err)
		}
		roundtrip, err := authoring.ReadDocument(root, name)
		if err != nil || roundtrip["runtime_required"] != false || roundtrip["ID"] != "R-ID-example" || roundtrip["name"] != "책임 매칭 🧭" {
			t.Fatalf("serialization changed values: %v %v", roundtrip, err)
		}
	}
}

func TestCurrentPoliciesValidateThroughRegisteredPublicSchema(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	for _, path := range []string{
		"developer/policy/GOV/MPD-GOV-0001.yaml",
		"developer/policy/ARCH/MPD-ARCH-0001.yaml",
		"developer/policy/ARCH/MPD-ARCH-0002.yaml",
		"developer/policy/CHANGE/MPD-CHANGE-0001.yaml",
		"developer/policy/CHANGE/MPD-CHANGE-0007.yaml",
		"developer/policy/RECORD/MPD-RECORD-0001.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			input := policyFixture(t, catalog, path)
			before := copyDocument(input)
			family := input["Policy_Identity"].(map[string]any)["root_family"].(string)
			generated, status, err := catalog.GenerateRules(validate, input, family)
			if err != nil || status != "GENERATED" {
				t.Fatalf("registered public schema rejected source policy: %s %v", status, err)
			}
			if !reflect.DeepEqual(input, before) || !reflect.DeepEqual(generated, before) {
				t.Fatal("generation changed source values or synthesized optional fields")
			}
		})
	}
}

func TestGenerationCLILeavesDefaultAndFailureUnwritten(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	output := "must-not-exist-authoring-test.json"
	result, code := authoring.Run(catalog.Root, "policy-rules", "generate", "", "", output, validate)
	if code != 0 || result.(map[string]any)["status"] != "OMITTED" {
		t.Fatalf("default generation: %v", result)
	}
	if _, err := os.Stat(filepath.Join(catalog.Root, output)); !os.IsNotExist(err) {
		t.Fatal("default created an output file")
	}
	result, code = authoring.Run(catalog.Root, "policy-rules", "generate", "GOV", "", output, validate)
	if code == 0 || result.(map[string]any)["execution_authorized"] != false {
		t.Fatalf("missing input accepted or execution authorized: %v", result)
	}
	if _, err := os.Stat(filepath.Join(catalog.Root, output)); !os.IsNotExist(err) {
		t.Fatal("failed generation created output")
	}
	result, code = authoring.Run(catalog.Root, "responsibility-authority", "generate", "", "", "", validate)
	if code != 0 {
		t.Fatalf("empty profile generation failed: %v", result)
	}
	generated := result.(map[string]any)["document"].(map[string]any)
	if len(generated["responsibility"].([]any)) != 0 {
		t.Fatal("unspecified mappings generated")
	}
	if _, copied := generated["component_map"]; copied {
		t.Fatal("retired component map generated")
	}
	if bindings, present := generated["component_bindings"]; present && len(bindings.(map[string]any)) != 0 {
		t.Fatal("unspecified component assignments generated")
	}
}
