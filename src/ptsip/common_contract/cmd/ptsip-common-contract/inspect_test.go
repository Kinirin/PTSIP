package main

import (
	"path/filepath"
	"strings"
	"testing"
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
