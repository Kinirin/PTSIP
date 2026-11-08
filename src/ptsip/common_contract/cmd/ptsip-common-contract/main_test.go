package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/compatibility"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/registry"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
)

const lifecycle = "urn:ptsip:common_policy:lifecycle:v1"
const identity = "urn:ptsip:common_policy:identity:v1"
const module = "../.."

func TestMachineProtocol(t *testing.T) {
	raw, err := os.ReadFile("protocol.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsondoc.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	const id = "urn:ptsip:common-contract:cli-protocol:v1"
	compile := func(fragment string) func(any) error {
		schema, err := validator.Compile(resolver.Graph{Root: id + fragment, Docs: map[string]map[string]any{id: doc.(record)}})
		if err != nil {
			t.Fatal(err)
		}
		return schema.Validate
	}
	request := record{"format": protocol, "operation": "validate", "urn": lifecycle, "instance": record{"status": "APPROVED.Default"}}
	if err := compile("#/$defs/request")(request); err != nil {
		t.Fatal(err)
	}
	response := dispatch(request, module)
	response["format"] = protocol
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	value, err := jsondoc.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := compile("#/$defs/response")(value); err != nil {
		t.Fatal(err)
	}
}

func TestR02R03R05R07CustomerResources(t *testing.T) {
	ids := []string{identity, lifecycle, "urn:ptsip:common_policy:rules:v1",
		"urn:ptsip:common:plan:unit:v1", "urn:ptsip:common:plan:lifecycle:v1", "urn:ptsip:common:plan:identity:v1"}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			resolved := contractOperation(record{"urn": id}, module, "lookup")
			if resolved["status"] != "RESOLVED" || resolved["contract"].(registry.Entry).URN != id {
				t.Fatalf("lookup = %#v", resolved)
			}
			valid := contractOperation(record{"urn": id, "instance": record{"extension": true}}, module, "validate")
			if valid["status"] != "VALID" {
				t.Fatalf("extension fields = %#v", valid)
			}
		})
	}
	statuses := []string{"DRAFT.Default", "DRAFT.Reviewed", "DRAFT.Denied", "DRAFT.Pass",
		"APPROVED.Default", "APPROVED.Applying", "APPROVED.Application_Problems", "APPROVED.Unresolved",
		"APPROVED.Activate", "APPROVED.provisional", "RETIRED.Default", "RETIRED.Removable", "RETIRED.Waiting"}
	for _, status := range statuses {
		got := contractOperation(record{"urn": lifecycle, "instance": record{"status": status}}, module, "validate")
		if got["status"] != "VALID" {
			t.Fatalf("official status %q = %#v", status, got)
		}
	}
	for _, status := range []any{"APPROVED", "APPROVED.Provisional", "APPROVED.Superseded", false} {
		got := contractOperation(record{"urn": lifecycle, "instance": record{"status": status}}, module, "validate")
		if got["status"] != "INVALID" {
			t.Fatalf("foreign status %v = %#v", status, got)
		}
	}
	for _, id := range []string{"urn:ptsip:common_policy:lifecycle", "urn:ptsip:common_policy:lifecycle:v2"} {
		got := contractOperation(record{"urn": id}, module, "lookup")
		if got["status"] != "UNRESOLVED" {
			t.Fatalf("unregistered version %s = %#v", id, got)
		}
	}
}

func TestR01R03R04ExtensionProofs(t *testing.T) {
	base := contractOperation(record{"urn": lifecycle}, module, "lookup")["contract"].(registry.Entry)
	makeSchema := func(status record) record {
		return record{"type": "object", "properties": record{"status": status, "owner": record{"type": "string"}}}
	}
	stronger := makeSchema(record{"const": "DRAFT.Default"})
	cases := []struct {
		name    string
		schema  record
		meaning record
		status  string
		code    string
	}{
		{"strengthening", stronger, nil, "SUPPORTED", ""},
		{"composed strengthening", record{"allOf": []any{record{"$ref": lifecycle}, stronger}}, nil, "SUPPORTED", ""},
		{"weakening", makeSchema(record{"type": "string"}), nil, "INCOMPATIBLE", "WEAKENED_CONSTRAINT"},
		{"contradiction", makeSchema(record{"const": "DEVELOPER.ONLY"}), nil, "INCOMPATIBLE", "CONTRADICTORY_CONSTRAINT"},
		{"same name different meaning", stronger, record{"/properties/status": "urn:example:different-status"}, "INCOMPATIBLE", "MEANING_CONFLICT"},
		{"unknown meaning", stronger, record{"/properties/status": nil}, "UNRESOLVED", "MEANING_EVIDENCE_MISSING"},
		{"unknown schema implication", makeSchema(record{"type": "string", "pattern": "^DRAFT"}), nil, "UNRESOLVED", "UNSUPPORTED_SCHEMA_COMPARISON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			meanings := record{}
			for pointer, meaning := range base.Semantics {
				meanings[pointer] = meaning
			}
			for pointer, meaning := range tc.meaning {
				if meaning == nil {
					delete(meanings, pointer)
				} else {
					meanings[pointer] = meaning
				}
			}
			got := contractOperation(record{"urn": lifecycle, "schema": tc.schema, "meanings": meanings}, module, "compatibility")
			if got["status"] != tc.status || got["structure"].(record)["status"] != "VALID" {
				t.Fatalf("report = %#v", got)
			}
			findings := got["semantics"].(compatibility.Result).Findings
			if tc.code != "" && (len(findings) != 1 || findings[0].Code != tc.code) {
				t.Fatalf("findings = %#v", findings)
			}
			if affected := got["change_impact"].(record); affected["scope"] != "AFFECTED_EXTENSION_ONLY" {
				t.Fatalf("impact = %#v", affected)
			}
			if independent := contractOperation(record{"urn": identity}, module, "lookup"); independent["status"] != "RESOLVED" {
				t.Fatalf("independent contract = %#v", independent)
			}
		})
	}
}

func TestR02ResourceAndReferenceFailures(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"schemas", "registry"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
		files, err := os.ReadDir(filepath.Join(module, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(module, dir, f.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, dir, f.Name()), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct{ ref, code string }{
		{"#", "REFERENCE_CYCLE"}, {"#/$defs/missing", "MISSING_SCHEMA_POINTER"},
		{"urn:ptsip:contract:constant:plan_status_draft:v1", "UNREGISTERED_URN"},
	} {
		result := contractOperation(record{"urn": lifecycle, "schema": record{"$ref": tc.ref}}, root, "compatibility")
		assertUnresolved(t, result, tc.code)
	}
	asset := filepath.Join(root, "schemas/policy_lifecycle.schema.json")
	if err := os.WriteFile(asset, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	assertUnresolved(t, contractOperation(record{"urn": lifecycle}, root, "lookup"), "STALE_SCHEMA")
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	assertUnresolved(t, contractOperation(record{"urn": lifecycle}, root, "lookup"), "RESOURCE_UNAVAILABLE")
	if independent := contractOperation(record{"urn": identity}, root, "lookup"); independent["status"] != "RESOLVED" {
		t.Fatalf("independent contract = %#v", independent)
	}
	manifest := filepath.Join(root, registry.Manifest)
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var data record
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	entries := data["contracts"].([]any)
	data["contracts"] = append(entries, entries[0])
	raw, err = json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, raw, 0600); err != nil {
		t.Fatal(err)
	}
	assertUnresolved(t, contractOperation(record{"urn": identity}, root, "lookup"), "AMBIGUOUS_REGISTRY_ENTRY")
}

func assertUnresolved(t *testing.T, report record, code string) {
	t.Helper()
	if report["status"] != "UNRESOLVED" || !strings.Contains(report["reference_resolution"].(record)["problem"].(string), code) {
		t.Fatalf("expected %s, got %#v", code, report)
	}
}
