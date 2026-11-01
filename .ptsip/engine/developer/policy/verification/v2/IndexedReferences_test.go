package v2_test

import (
	"encoding/json"
	"strings"
	"testing"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	entry "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractentry"
	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
	v2 "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/v2"
)

const changeTarget = "developer/policy/CHANGE/MPD-CHANGE-0001.yaml"

func indexedCopy(t *testing.T, d authoring.Document) authoring.Document {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var clone authoring.Document
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
func indexedContract(t *testing.T, doc authoring.Document) map[string]any {
	t.Helper()
	rules := doc["Policy_Rules"].(map[string]any)
	return rules["contracts"].([]any)[0].(map[string]any)
}

func TestChange0001RegisteredV2SelectionAndStructure(t *testing.T) {
	root := repoRoot(t)
	selected, err := entry.Resolve(root, changeTarget)
	if err != nil {
		t.Fatalf("policy identity selection: %v", err)
	}
	if selected.PolicyKind != "PTSIP_DEVELOPER_POLICY" || selected.RootFamily != "CHANGE" ||
		selected.ContractRefs["public_schema"] != ".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json" ||
		selected.ContractRefs["selection_schema"] != ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json" {
		t.Fatalf("wrong selected contract: %+v", selected)
	}
	doc, err := authoring.ReadDocument(root, changeTarget)
	if err != nil {
		t.Fatalf("block YAML parse: %v", err)
	}
	c := indexedContract(t, doc)
	if c["serialization"] != "INDEXED_SOURCE_BINDINGS" {
		t.Fatal("indexed contract not selected")
	}
	if doc["Policy_Lifecycle"].(map[string]any)["status"] != "DRAFT.Reviewed" {
		t.Fatal("approval state changed")
	}
	if c["Rule_ver"] != "1" {
		t.Fatal("source rule version changed")
	}
	if len(c["source_items"].([]any)) != 19 {
		t.Fatal("19 source items were not retained")
	}
	if len(c["requirements"].([]any)) != 4 {
		t.Fatal("trial requirement grouping changed unexpectedly")
	}
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatalf("installed official PTSIP validator missing: %v", err)
	}
	if err := v2.Check(root, doc, validator); err != nil {
		t.Fatalf("registered v2 verification failed: %v", err)
	}
}

func TestChange0001IndexedReferenceCounterexamples(t *testing.T) {
	root := repoRoot(t)
	original, err := authoring.ReadDocument(root, changeTarget)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatalf("official native validator missing: %v", err)
	}
	for _, tc := range []struct {
		name     string
		update   func(map[string]any)
		expected string
	}{
		{"duplicate-source-id", func(c map[string]any) {
			xs := c["source_items"].([]any)
			c["source_items"] = append(xs, xs[0])
		}, ""},
		{"missing-document-id", func(c map[string]any) {
			c["source_items"].([]any)[0].(map[string]any)["document_id"] = "MISSING_DOC"
		}, "INDEXED_UNDEFINED_DOCUMENT"},
		{"unresolved-reference", func(c map[string]any) {
			req := c["requirements"].([]any)[0].(map[string]any)
			req["source_bindings"].([]any)[0].(map[string]any)["ref_ids"] = []any{"NO_SUCH_SOURCE"}
		}, "INDEXED_UNDEFINED_SOURCE_ID"},
		{"wrong-source-purpose", func(c map[string]any) {
			req := c["requirements"].([]any)[0].(map[string]any)
			req["source_bindings"].([]any)[2].(map[string]any)["ref_ids"] = []any{"I11"}
		}, "INDEXED_ROLE_MISMATCH"},
		{"invalid-revision", func(c map[string]any) {
			c["source_documents"].([]any)[0].(map[string]any)["revision_ref"] = "git:bad"
		}, "INDEXED_INVALID_GIT_REVISION"},
		{"duplicate-purpose", func(c map[string]any) {
			req := c["requirements"].([]any)[0].(map[string]any)
			list := req["source_bindings"].([]any)
			req["source_bindings"] = append(list, list[0])
		}, ""},
		{"empty-without-basis", func(c map[string]any) {
			req := c["requirements"].([]any)[0].(map[string]any)
			req["source_bindings"].([]any)[0] = map[string]any{"purpose": "MEANING", "state": "CONFIRMED_EMPTY"}
		}, ""},
		{"not-applicable-without-basis", func(c map[string]any) {
			req := c["requirements"].([]any)[0].(map[string]any)
			req["source_bindings"].([]any)[0] = map[string]any{"purpose": "MEANING", "state": "NOT_APPLICABLE"}
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := indexedCopy(t, original)
			tc.update(indexedContract(t, doc))
			err := v2.Check(root, doc, validator)
			if err == nil {
				t.Fatal("invalid reference contract accepted")
			}
			if tc.expected != "" && !strings.Contains(err.Error(), tc.expected) {
				t.Fatalf("expected %s, got %v", tc.expected, err)
			}
		})
	}
}

func TestRequirementIDsReplaceResponsibilityReferences(t *testing.T) {
	root := repoRoot(t)
	original, err := authoring.ReadDocument(root, changeTarget)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatal(err)
	}
	doc := indexedCopy(t, original)
	reqs := indexedContract(t, doc)["requirements"].([]any)
	reqs[0].(map[string]any)["responsibility_refs"] = []any{"R-ID-C615A4E8BF5F4A0597CD28E6A5B79011"}
	if err := v2.Check(root, doc, validator); err == nil {
		t.Fatal("removed responsibility_refs accepted")
	}
	doc = indexedCopy(t, original)
	reqs = indexedContract(t, doc)["requirements"].([]any)
	reqs[1].(map[string]any)["requirement_id"] = reqs[0].(map[string]any)["requirement_id"]
	if err := v2.Check(root, doc, validator); err == nil || !strings.Contains(err.Error(), "INDEXED_DUPLICATE_ID") {
		t.Fatalf("duplicate requirement identity accepted: %v", err)
	}
}

func TestGovRequiredFieldsAndLocalRuleSources(t *testing.T) {
	root := repoRoot(t)
	original, err := authoring.ReadDocument(root, "developer/policy/GOV/MPD-GOV-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		update func(map[string]any)
	}{
		{"missing-requirements", func(c map[string]any) { delete(c, "requirements") }},
		{"inline-rule-source", func(c map[string]any) {
			c["rules"].([]any)[0].(map[string]any)["source_ref"] = map[string]any{"source_ref": "x", "item_ref": "/x", "revision_ref": "1"}
		}},
		{"undefined-rule-source", func(c map[string]any) { c["rules"].([]any)[0].(map[string]any)["source_ref"] = []any{"MISSING_SOURCE"} }},
		{"pending-without-unresolved", func(c map[string]any) { delete(c["requirements"].([]any)[0].(map[string]any), "unresolved_fields") }},
		{"empty-optional-field", func(c map[string]any) { c["requirements"].([]any)[0].(map[string]any)["target_refs"] = []any{} }},
		{"duplicate-item-id", func(c map[string]any) {
			items := c["source_items"].([]any)
			items[1].(map[string]any)["item_id"] = items[0].(map[string]any)["item_id"]
		}},
		{"undefined-authority-id", func(c map[string]any) {
			c["rules"].([]any)[0].(map[string]any)["authority_ref"] = []any{"MISSING_AUTHORITY"}
		}},
		{"inline-authority-coordinates", func(c map[string]any) {
			c["rules"].([]any)[0].(map[string]any)["authority_ref"] = map[string]any{"source_ref": "x", "item_ref": "/x", "revision_ref": "1"}
		}},
		{"duplicate-authority-id", func(c map[string]any) {
			entries := c["authority_ref"].([]any)
			c["authority_ref"] = append(entries, entries[0])
		}},
		{"missing-authority-catalog", func(c map[string]any) { delete(c, "authority_ref") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := indexedCopy(t, original)
			tc.update(indexedContract(t, doc))
			if err := v2.Check(root, doc, validator); err == nil {
				t.Fatal("invalid GOV field contract accepted")
			}
		})
	}
	// No pending binding is declared in this structural fixture. The absence
	// of an unresolved_fields value is allowed and grants no fulfillment.
	doc := indexedCopy(t, original)
	c := indexedContract(t, doc)
	originalRequirement := c["requirements"].([]any)[0].(map[string]any)
	source := originalRequirement["meaning_refs"]
	c["requirements"] = []any{map[string]any{
		"requirement_id": "REQ-FIXTURE", "meaning_refs": source, "scope_refs": source,
		"target_refs": source, "criteria_refs": source,
		"empty_field_basis": []any{map[string]any{"field": "evidence_requirement_refs", "basis_refs": source}},
	}}
	if err := v2.Check(root, doc, validator); err != nil {
		t.Fatalf("omitted empty optional metadata rejected: %v", err)
	}
}

func TestIndexedV2CrossPolicyRegression(t *testing.T) {
	root := repoRoot(t)
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatalf("official native validator missing: %v", err)
	}
	for _, target := range []string{
		"developer/policy/GOV/MPD-GOV-0001.yaml",
		"developer/policy/ARCH/MPD-ARCH-0001.yaml",
		"developer/policy/ARCH/MPD-ARCH-0002.yaml",
		"developer/policy/CHANGE/MPD-CHANGE-0007.yaml",
		"developer/policy/RECORD/MPD-RECORD-0001.yaml",
	} {
		t.Run(target, func(t *testing.T) {
			selected, err := entry.Resolve(root, target)
			if err != nil {
				t.Fatalf("selection regression: %v", err)
			}
			if selected.ContractRefs["selection_schema"] != ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json" {
				t.Fatal("unexpected schema selection")
			}
			doc, err := authoring.ReadDocument(root, target)
			if err != nil {
				t.Fatalf("YAML parse regression: %v", err)
			}
			if err := v2.Check(root, doc, validator); err != nil {
				t.Fatalf("selected v2 structure regression: %v", err)
			}
		})
	}
}

func TestAuthorityReferencesStayInOwningContract(t *testing.T) {
	root := repoRoot(t)
	doc, err := authoring.ReadDocument(root, "developer/policy/GOV/MPD-GOV-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatal(err)
	}
	first := indexedContract(t, doc)
	second := indexedCopy(t, first)
	second["contract_id"] = "urn:ptsip:policy:test:other-contract"
	delete(second, "authority_ref")
	doc["Policy_Rules"].(map[string]any)["contracts"] = []any{first, second}
	if err := v2.Check(root, doc, validator); err == nil || !strings.Contains(err.Error(), "UNDEFINED_AUTHORITY_ID") {
		t.Fatalf("authority catalog from another contract was consumed: %v", err)
	}
}

func TestAllFamilyReviewDimensions(t *testing.T) {
	root := repoRoot(t)
	original, err := authoring.ReadDocument(root, "developer/policy/GOV/MPD-GOV-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	validator, err := verification.InstalledValidator()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{"NORM", "GOV", "INTENT", "ARCH", "INFO", "CNTR", "RISK", "SUPPLY", "REAL", "ASSURE", "CTRL", "CHANGE", "OPS", "RECORD"} {
		t.Run(family, func(t *testing.T) {
			// Synthetic review-slot fixture only. It makes no claim that a GOV
			// policy's substantive meaning applies to another Family.
			doc := indexedCopy(t, original)
			doc["Policy_Identity"].(map[string]any)["root_family"] = family
			c := indexedContract(t, doc)
			meaning := c["requirements"].([]any)[0].(map[string]any)["meaning_refs"]
			req := map[string]any{"requirement_id": "REQ-REVIEW", "meaning_refs": meaning,
				"unresolved_fields": []any{"scope_refs", "target_refs", "constraint_refs", "criteria_refs", "evidence_requirement_refs"}}
			c["requirements"] = []any{req}
			if err := v2.Check(root, doc, validator); err != nil {
				t.Fatalf("pending review rejected: %v", err)
			}
			for field, required := range map[string]bool{
				"scope_refs": true, "target_refs": family != "INTENT", "constraint_refs": family == "INTENT",
				"criteria_refs": family != "NORM", "evidence_requirement_refs": family == "GOV" || family == "ASSURE" || family == "SUPPLY" || family == "RECORD",
			} {
				candidate := indexedCopy(t, doc)
				r := indexedContract(t, candidate)["requirements"].([]any)[0].(map[string]any)
				remaining := []any{}
				for _, item := range r["unresolved_fields"].([]any) {
					if item != field {
						remaining = append(remaining, item)
					}
				}
				r["unresolved_fields"] = remaining
				if err := v2.Check(root, candidate, validator); (err != nil) != required {
					t.Fatalf("%s required=%v: %v", field, required, err)
				}
			}
		})
	}
}
