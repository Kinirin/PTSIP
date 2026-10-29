package contract_authoring_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func govInput() authoring.Document {
	return doc(`{"Policy_Rules":{"Identity":{"contract_id":"urn:ptsip:policy:rule:gov:example","Rule_ver":"1"},"Definition":{"meaning":"Example policy"},"Boundary":{"scope":"Example scope"}}}`)
}

func profileFixture(t *testing.T, catalog *authoring.Catalog) authoring.Document {
	t.Helper()
	profile, err := authoring.ReadDocument(catalog.Root, authoring.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	delete(profile, "component_map")
	delete(profile, "component_bindings")
	catalog.Definitions = doc(`{"schema_version":"ptsip-responsibility-authority/v1","condition_list":[{"ID":"C-ID-review","name":"Review is complete"}],"authority_list":[{"ID":"OA-ID-edit","name":"Edit policy","action":"edit","scope":"example"}]}`)
	profile["responsibility"] = doc(`{"items":[{"name":"Policy editor","ID":"R-ID-editor","condition_id":["C-ID-review"],"Operational_Authority":["OA-ID-edit"],"role_name":"Editor","runtime_required":false}]}`)["items"]
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
	for _, family := range catalog.Families {
		t.Run(family, func(t *testing.T) {
			input := govInput()
			if family != "GOV" {
				input = doc(`{"Policy_Rules":{"Identity":{"approval":{"state":"DRAFT.Default"}},"Definition":{"meaning":"Example"},"Boundary":{"scope":"Example"},"state":"APPROVED.provisional","title":"Explicitly supplied legacy state"}}`)
			}
			generated, status, err := catalog.GenerateRules(validate, input, family)
			if err != nil || status != "GENERATED" {
				t.Fatalf("generation failed: %s %v", status, err)
			}
			if err := catalog.CheckRules(validate, generated); err != nil {
				t.Fatal(err)
			}
			rules := generated["Policy_Rules"].(map[string]any)
			if _, present := rules["Responsibility_Refs"]; present {
				t.Fatal("optional responsibility reference was generated")
			}
			if family == "GOV" {
				if _, present := rules["state"]; present {
					t.Fatal("approval state was invented")
				}
				if _, present := rules["title"]; present {
					t.Fatal("optional title was generated")
				}
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
	if err := catalog.CheckRules(validate, doc(`{"Policy_Rules":{}}`)); err != nil {
		t.Fatal("shared optional fields unexpectedly required", err)
	}
	for _, input := range []authoring.Document{
		doc(`{"Policy_Identity":{"root_family":"UNKNOWN"}}`),
		doc(`{"Policy_Identity":{"root_family":false}}`),
		doc(`{"Policy_Identity":{"root_family":"GOV"}}`),
		doc(`{"Policy_Identity":{"root_family":"GOV"},"Policy_Rules":{}}`),
	} {
		if err := catalog.CheckRules(validate, input); err == nil {
			t.Fatalf("invalid selection accepted: %v", input)
		}
	}
	conflict := doc(`{"Policy_Identity":{"root_family":"INFO"}}`)
	if _, _, err := catalog.GenerateRules(validate, conflict, "GOV"); err == nil || !strings.Contains(err.Error(), "ROOT_FAMILY_CONFLICT") {
		t.Fatal("conflicting selection accepted", err)
	}
	if _, _, err := catalog.GenerateRules(validate, authoring.Document{}, "GOV"); err == nil || !strings.Contains(err.Error(), "REQUIRED_INPUT_VALUES") {
		t.Fatal("required values were invented", err)
	}
	if _, _, err := catalog.GenerateRules(validate, govInput(), "ARCH"); err == nil || !strings.Contains(err.Error(), ".state") {
		t.Fatal("legacy approval value must be supplied explicitly", err)
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
			p["responsibility"].([]any)[0].(map[string]any)["condition_id"] = []any{"C-ID-missing"}
		}},
		{"unknown_authority", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["Operational_Authority"] = []any{"OA-ID-missing"}
		}},
		{"duplicate_responsibility_id", func(p authoring.Document) {
			entries := p["responsibility"].([]any)
			p["responsibility"] = append(entries, entries[0])
		}},
		{"duplicate_condition_ref", func(p authoring.Document) {
			p["responsibility"].([]any)[0].(map[string]any)["condition_id"] = []any{"C-ID-review", "C-ID-review"}
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

func TestPolicyResponsibilityReferencesResolveThroughProfile(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	profile := profileFixture(t, catalog)
	temporaryRoot := t.TempDir()
	profilePath := filepath.Join(temporaryRoot, filepath.FromSlash(authoring.ProfilePath))
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(profile)
	if err := os.WriteFile(profilePath, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	catalog.Root = temporaryRoot
	input := govInput()
	input["Policy_Rules"].(map[string]any)["Responsibility_Refs"] = []any{"R-ID-editor"}
	if _, _, err := catalog.GenerateRules(validate, input, "GOV"); err != nil {
		t.Fatal(err)
	}
	input["Policy_Rules"].(map[string]any)["Responsibility_Refs"] = []any{"R-ID-missing"}
	if _, _, err := catalog.GenerateRules(validate, input, "GOV"); err == nil || !strings.Contains(err.Error(), "UNRESOLVED_RESPONSIBILITY") {
		t.Fatal("unresolved responsibility accepted", err)
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
		value := doc(`{"runtime_required":false,"ID":"R-ID-example","conditions":["C-ID-review"],"name":"책임 매칭 🧭"}`)
		if err := authoring.WriteDocument(root, name, value); err != nil {
			t.Fatal(err)
		}
		roundtrip, err := authoring.ReadDocument(root, name)
		if err != nil || roundtrip["runtime_required"] != false || roundtrip["ID"] != "R-ID-example" || roundtrip["name"] != "책임 매칭 🧭" {
			t.Fatalf("serialization changed values: %v %v", roundtrip, err)
		}
	}
}

func TestCurrentGOVPoliciesValidateThroughSharedPublicSelection(t *testing.T) {
	catalog, validate := nativeCatalog(t)
	public, err := authoring.ReadDocument(catalog.Root, ".ptsip/contracts/schemas/Policy/PTSIP_Public_Structure.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	resources := append([]any(nil), catalog.Resources...)
	for _, relative := range []string{
		".ptsip/contracts/schemas/Policy/Policy_Identity/PTSIP_Policy_Identity.schema.json",
		".ptsip/contracts/policy/identity/policy_kind.json",
		".ptsip/contracts/schemas/Policy/Policy_Status/Public_Policy_Status.schema.json",
		".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json",
	} {
		resource, err := authoring.ReadDocument(catalog.Root, relative)
		if err != nil {
			t.Fatal(err)
		}
		resources = append(resources, resource)
	}
	for _, suffix := range []string{"0001", "0004", "0005", "0006"} {
		policy, err := authoring.ReadDocument(catalog.Root, "developer/policy/GOV/MPD-GOV-"+suffix+".yaml")
		if err != nil {
			t.Fatal(err)
		}
		valid, err := validate(public, policy, resources)
		if err != nil || !valid {
			t.Fatalf("GOV-%s rejected by shared public schema: %v", suffix, err)
		}
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
	for _, field := range []string{"component_map", "component_bindings"} {
		if _, copied := generated[field]; copied {
			t.Fatal("unspecified component assignments generated")
		}
	}
}
