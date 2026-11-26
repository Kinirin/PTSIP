package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
)

func TestDiagnosticResourceSources(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, field := range []string{"Plan_Document", "Plan_Identity", "Plan_Lifecycle", "Plan_Unit",
		"Policy_Document", "Policy_Identity", "Policy_Lifecycle", "Policy_Rules", "Policy_Content"} {
		t.Run(field, func(t *testing.T) {
			report := dispatch(record{"format": protocol, "operation": "inspect-resource", "field": field}, root)
			stage, _ := report["reference_resolution"].(record)
			if report["binding"] != "DIAGNOSTIC_ONLY" || stage["status"] != "RESOLVED" || report["status"] != "RESOLVED" {
				t.Fatalf("source resolution %s: %#v", field, report)
			}
		})
	}
	valid := dispatch(record{"format": protocol, "operation": "inspect-resource", "field": "Plan_Lifecycle",
		"instance": record{"status": "Halt.Paused"}}, root)
	if valid["status"] != "VALID" {
		t.Fatalf("valid status %#v", valid)
	}
	invalid := dispatch(record{"format": protocol, "operation": "inspect-resource", "field": "Plan_Lifecycle",
		"instance": record{"status": "DRAFT.Default"}}, root)
	if invalid["status"] != "INVALID" {
		t.Fatalf("invalid status %#v", invalid)
	}
	unknown := dispatch(record{"format": protocol, "operation": "inspect-resource", "field": "Unknown"}, root)
	stage, _ := unknown["reference_resolution"].(record)
	if unknown["status"] != "UNRESOLVED" || !strings.Contains(stage["problem"].(string), "UNREGISTERED_DIAGNOSTIC_FIELD") {
		t.Fatalf("unknown schema %#v", unknown)
	}
}

func TestCanonicalPlanContractValidatesIndependentFields(t *testing.T) {
	root := filepath.Join("..", "..")
	lookup := dispatch(record{"format": protocol, "operation": "lookup", "urn": registry.PlanDocumentURN}, root)
	entry, _ := lookup["contract"].(registry.Entry)
	if lookup["status"] != "RESOLVED" || entry.Version != "SCH.1.00" {
		t.Fatalf("canonical Plan registration: %#v", lookup)
	}
	for _, child := range registry.PlanDependencies() {
		result := dispatch(record{"format": protocol, "operation": "lookup", "urn": child.URN}, root)
		if result["status"] != "UNRESOLVED" {
			t.Errorf("Plan child acquired an independent public registration: %#v", result)
		}
	}
	compatibility := dispatch(record{
		"format": protocol, "operation": "compatibility", "urn": registry.PlanDocumentURN,
		"schema": record{"allOf": []any{record{"$ref": registry.PlanDocumentURN}}},
	}, root)
	if compatibility["status"] != "UNRESOLVED" {
		t.Errorf("unregistered Plan semantic compatibility asserted: %#v", compatibility)
	}

	for _, tc := range []struct {
		name   string
		change func(record, record, record, record)
	}{
		{name: "approved structure"},
		{"missing schema version", func(doc, _, _, _ record) { delete(doc, "SchemaVersion") }},
		{"wrong schema version", func(doc, _, _, _ record) { doc["SchemaVersion"] = "SCH.1.01" }},
		{"superseded combined version", func(doc, _, _, _ record) { doc["contract_vision"] = "ver.1.00/Rev.0001" }},
		{"invalid Plan ID", func(_, identity, _, _ record) { identity["plan_id"] = "plan-abc*" }},
		{"missing Plan ID", func(_, identity, _, _ record) { delete(identity, "plan_id") }},
		{"superseded lifecycle status", func(_, _, lifecycle, _ record) { lifecycle["status"] = "Implementation.In_progress" }},
		{"misplaced schema version", func(_, _, lifecycle, _ record) { lifecycle["SchemaVersion"] = "SCH.1.00" }},
		{"empty completion date", func(_, _, lifecycle, _ record) { lifecycle["completion_date"] = "" }},
		{"non-string completion date", func(_, _, lifecycle, _ record) { lifecycle["completion_date"] = 12 }},
		{"timezone missing", func(_, _, lifecycle, _ record) { lifecycle["Creation_date"] = "2026-11-26T09:00:00" }},
		{"invalid calendar date", func(_, _, lifecycle, _ record) { lifecycle["last_modified_date"] = "2026-02-30T00:00:00Z" }},
		{"independent approval field", func(_, _, lifecycle, _ record) { lifecycle["approval_ref"] = "example" }},
		{"invalid work unit ID", func(_, _, _, unit record) { unit["id"] = "WU-aa-007" }},
		{"mixed work unit prefix", func(_, _, _, unit record) { unit["id"] = "WU-CD-007" }},
		{"superseded work unit item ID", func(_, _, _, unit record) { unit["work_unit_id"] = unit["id"] }},
		{"superseded work unit container", func(doc, _, _, _ record) { doc["work_unit_item"] = doc["work_unit"] }},
		{"non-array work unit value", func(_, _, _, unit record) { unit["value"] = "user data" }},
		{"work unit session", func(_, _, _, unit record) { unit["session_id"] = "example" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity := record{"plan_id": "plan-Ab90", "revision": "consumer-owned revision"}
			lifecycle := record{"status": "Approval.In_progress", "Creation_date": "2026-11-26T09:00:00+09:00",
				"last_modified_date": "2026-11-26T00:00:01Z", "completion_date": nil}
			unit := record{"id": "WU-AB-007", "task_id": []any{}, "value": []any{nil, "free-form", true}}
			doc := record{
				"SchemaVersion": "SCH.1.00", "Plan_Identity": identity,
				"Plan_lifecycle": lifecycle,
				"work_unit":      []any{record{"id": "WU-AB-000", "task_id": []any{}}, unit},
			}
			expected := "VALID"
			if tc.change != nil {
				tc.change(doc, identity, lifecycle, unit)
				expected = "INVALID"
			}
			result := dispatch(record{
				"format": protocol, "operation": "validate", "urn": registry.PlanDocumentURN, "instance": doc,
			}, root)
			if result["status"] != expected {
				t.Fatalf("expected %s: %#v", expected, result)
			}
		})
	}
}

func TestCanonicalPlanRuleReferences(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, tc := range []struct {
		name, status string
		ref          any
	}{
		{"owned exact rule", "VALID", "SFP-CNTR-0009#SFP-CNTR-0009/R-02"},
		{"rule in another policy", "UNRESOLVED", "SFP-CNTR-0001#SFP-CNTR-0009/R-02"},
		{"unknown policy", "UNRESOLVED", "no-owner#SFP-CNTR-0009/R-02"},
		{"unknown rule", "UNRESOLVED", "SFP-CNTR-0009#no-rule"},
		{"case differs", "UNRESOLVED", "SFP-CNTR-0009#SFP-CNTR-0009/r-02"},
		{"old tuple", "INVALID", []any{"SFP-CNTR-0009", "SFP-CNTR-0009/R-02"}},
		{"missing rule", "INVALID", "SFP-CNTR-0009#"},
		{"ambiguous separator", "INVALID", "SFP-CNTR-0009#SFP-CNTR-0009/R-02#extra"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := dispatch(record{
				"format": protocol, "operation": "validate", "urn": registry.PlanDocumentURN,
				"instance": record{"SchemaVersion": "SCH.1.00", "policy_refs": []any{tc.ref}},
			}, root)
			if result["status"] != tc.status {
				t.Fatalf("expected %s: %#v", tc.status, result)
			}
			if tc.status == "UNRESOLVED" && !strings.Contains(result["reference_resolution"].(record)["problem"].(string), "policy_refs[0]") {
				t.Fatalf("missing affected reference index: %#v", result)
			}
		})
	}
}

func TestCanonicalPolicyContractValidatesReferencedFields(t *testing.T) {
	root := filepath.Join("..", "..")
	const urn = "urn:ptsip:contract:public-structure-policy:v4"
	lookup := dispatch(record{"format": protocol, "operation": "lookup", "urn": urn}, root)
	provenance, _ := lookup["provenance"].(record)
	entry, _ := lookup["contract"].(registry.Entry)
	if lookup["status"] != "RESOLVED" || provenance["registry_revision"] != "v2" || entry.Version != "SCH.1.00" {
		t.Fatalf("canonical Policy lookup: %#v", lookup)
	}
	lifecycle := record{"status": "APPROVED.Default"}
	instance := record{
		"Policy_Identity":  record{"id": "example", "SchemaVersion": "SCH.1.00", "Version": "v1.2.3"},
		"Policy_Lifecycle": lifecycle,
		"Policy_Rules":     record{"rules": []any{record{"id": "example"}}},
		"Policy_Content":   record{"items": []any{record{"id": "example", "value": true}}},
	}
	request := record{"format": protocol, "operation": "validate", "urn": urn, "instance": instance}
	if report := dispatch(request, root); report["status"] != "VALID" {
		t.Fatalf("valid canonical Policy: %#v", report)
	}
	identity := instance["Policy_Identity"].(record)
	identity["SchemaVersion"] = "v4"
	if report := dispatch(request, root); report["status"] != "INVALID" {
		t.Fatalf("Policy URN suffix accepted as a SchemaVersion instance value: %#v", report)
	}
	identity["SchemaVersion"] = "SCH.1.00"
	identity["version"] = "v1.2.3"
	if report := dispatch(request, root); report["status"] != "INVALID" {
		t.Fatalf("superseded Policy version field accepted: %#v", report)
	}
	delete(identity, "version")
	lifecycle["status"] = "APPROVED"
	if report := dispatch(request, root); report["status"] != "INVALID" {
		t.Fatalf("unknown lifecycle status accepted: %#v", report)
	}
}
