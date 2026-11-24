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
		{"independent approval field", func(_, _, lifecycle, _ record) { lifecycle["approval_ref"] = "example" }},
		{"invalid work unit ID", func(_, _, _, unit record) { unit["work_unit_id"] = "WU-aa-007" }},
		{"non-array work unit value", func(_, _, _, unit record) { unit["value"] = "user data" }},
		{"work unit session", func(_, _, _, unit record) { unit["session_id"] = "example" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity := record{"plan_id": "plan-Ab90", "revision": "consumer-owned revision"}
			lifecycle := record{"status": "Approval.In_progress", "completion_date": nil}
			unit := record{"work_unit_id": "WU-AB-007", "task_id": []any{}, "value": []any{nil, "free-form", true}}
			doc := record{
				"SchemaVersion": "SCH.1.00", "Plan_Identity": identity,
				"Plan_lifecycle": lifecycle, "work_unit_item": []any{unit},
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

func TestCanonicalPolicyContractValidatesReferencedFields(t *testing.T) {
	root := filepath.Join("..", "..")
	const urn = "urn:ptsip:contract:public-structure-policy:v4"
	lookup := dispatch(record{"format": protocol, "operation": "lookup", "urn": urn}, root)
	provenance, _ := lookup["provenance"].(record)
	if lookup["status"] != "RESOLVED" || provenance["registry_revision"] != "v2" {
		t.Fatalf("canonical Policy lookup: %#v", lookup)
	}
	lifecycle := record{"status": "APPROVED.Default"}
	instance := record{
		"Policy_Identity":  record{"id": "example"},
		"Policy_Lifecycle": lifecycle,
		"Policy_Rules":     record{"rules": []any{record{"id": "example"}}},
		"Policy_Content":   record{"items": []any{record{"id": "example", "value": true}}},
	}
	request := record{"format": protocol, "operation": "validate", "urn": urn, "instance": instance}
	if report := dispatch(request, root); report["status"] != "VALID" {
		t.Fatalf("valid canonical Policy: %#v", report)
	}
	lifecycle["status"] = "APPROVED"
	if report := dispatch(request, root); report["status"] != "INVALID" {
		t.Fatalf("unknown lifecycle status accepted: %#v", report)
	}
}
