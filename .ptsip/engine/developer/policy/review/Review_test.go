package review

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExactPointer(t *testing.T) {
	doc := map[string]any{"a/b": map[string]any{"~name": []any{nil, "selected"}}, "other": "excluded"}
	got, err := Pointer(doc, "/a~1b/~0name/1")
	if err != nil || got != "selected" {
		t.Fatalf("%v, %v", got, err)
	}
	got, err = Pointer(doc, "/a~1b/~0name/0")
	if err != nil || got != nil {
		t.Fatalf("explicit null lost: %v, %v", got, err)
	}
	for _, pointer := range []string{"a", "/missing", "/a~2b", "/a~", "/a~1b/~0name/01", "/a~1b/~0name/-", "/a~1b/~0name/2"} {
		if _, err := Pointer(doc, pointer); err == nil {
			t.Errorf("accepted %q", pointer)
		}
	}
}

func TestInventoryFindsUndeclaredFieldsEvenWhenSchemaAllowsThem(t *testing.T) {
	schema := map[string]any{"additionalProperties": true, "required": []any{"Identity"}, "properties": map[string]any{
		"Identity":   map[string]any{"properties": map[string]any{}},
		"Boundary":   map[string]any{"properties": map[string]any{"scope": map[string]any{}}},
		"Definition": map[string]any{"items": map[string]any{"properties": map[string]any{"id": map[string]any{"$ref": "urn:existing:id"}}}},
	}}
	value := map[string]any{"Boundary": map[string]any{"scope": "x", "one_off": true},
		"Definition": []any{map[string]any{"id": "existing", "unregistered": true}}, "candidate": true}
	want := []Finding{{"/Policy_Rules/Identity", "MISSING_DECLARED_FIELD"},
		{"/Policy_Rules/Boundary/one_off", "UNDECLARED_FIELD"},
		{"/Policy_Rules/Definition/0/unregistered", "UNDECLARED_FIELD"}, {"/Policy_Rules/candidate", "UNDECLARED_FIELD"}}
	if got := Inventory(schema, value, "/Policy_Rules"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestInventoryDoesNotClaimToInspectOpenOrConditionalObjects(t *testing.T) {
	for _, schema := range []map[string]any{{}, {"$ref": "urn:unknown"}, {"allOf": []any{}}, {"patternProperties": map[string]any{}}} {
		if got := Inventory(schema, map[string]any{"field": true}, "/x"); len(got) != 1 || !strings.Contains(got[0].Kind, "REVIEW") && got[0].Kind != "UNINSPECTED_SCHEMA" {
			t.Fatalf("silently inspected unsupported shape: %#v", got)
		}
	}
}

func write(t *testing.T, root, path, content string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotRejectsChangedBytesAndEscapingPaths(t *testing.T) {
	root := t.TempDir()
	write(t, root, "source.yaml", "value: true\n")
	hash, err := digest(root, "source.yaml")
	if err != nil {
		t.Fatal(err)
	}
	packet := Packet{Format: Format, Files: map[string]string{"source.yaml": hash}}
	if err := packet.Verify(root); err != nil {
		t.Fatal(err)
	}
	write(t, root, "source.yaml", "value: true\n\n")
	if err := packet.Verify(root); err == nil || !strings.Contains(err.Error(), "REVIEW_INPUT_CHANGED") {
		t.Fatalf("%v", err)
	}
	for _, path := range []string{"../outside", "/absolute", "C:/outside", "folder/../source.yaml"} {
		packet.Files = map[string]string{path: hash}
		if err := packet.Verify(root); err == nil {
			t.Errorf("accepted %s", path)
		}
	}
	if err := (Packet{Format: Format}).Verify(root); err == nil {
		t.Fatal("empty snapshot accepted")
	}
}

func TestBuildPreservesUnresolvedTargetAndOnlySelectedSourceSection(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".ptsip/index.json", `{"namespaces":{"contracts":{"status":"ACTIVE","index":"contracts/index.json"}}}`)
	write(t, root, ".ptsip/contracts/index.json", `{"policy":"policy/boundary.json","authoring":{"entrypoint":".ptsip/engine/developer","validators":["policy-rules"],"execution_authorized":false}}`)
	write(t, root, ".ptsip/engine/developer/index.json", `{"policy_authority":"NONE","execution_authorized":false,"validators":{"policy-check":"policy/index.json","contract-entry":"policy/index.json"}}`)
	write(t, root, ".ptsip/contracts/policy/boundary.json", `{"domain":"policy","routing_binding":{"data":".ptsip/contracts/policy/routing/policy_routes.json"}}`)
	write(t, root, ".ptsip/contracts/policy/routing/policy_routes.json", `{}`)
	write(t, root, "developer/policy/RECORD/legacy.yaml", `{"policy":{"id":"OLD"},"responsibility_family":"RECORD"}`)
	write(t, root, "legacy.yaml", `{"rules":{"selected":{"keep":true},"unrelated":"must not be included"}}`)
	packet, err := Build(root, []string{"developer/policy/RECORD/legacy.yaml"}, []string{"legacy.yaml#/rules/selected"})
	if err != nil {
		t.Fatal(err)
	}
	if packet.Targets[0].Selection != nil || packet.Targets[0].SelectionProblem != "POLICY_IDENTITY_SELECTION_UNRESOLVED" {
		t.Fatalf("guessed a legacy route: %#v", packet.Targets[0])
	}
	if len(packet.Sources) != 1 || !reflect.DeepEqual(packet.Sources[0].Value, map[string]any{"keep": true}) {
		t.Fatalf("source scope expanded: %#v", packet.Sources)
	}
	if packet.MutationAuthorized || packet.SchemaValidation != "NOT_PERFORMED" {
		t.Fatal("helper claimed authority/validation")
	}
	for _, spec := range []string{"legacy.yaml", "legacy.yaml#", "legacy.yaml#/rules/missing"} {
		if _, err := Build(root, []string{"developer/policy/RECORD/legacy.yaml"}, []string{spec}); err == nil {
			t.Errorf("accepted ambiguous source %s", spec)
		}
	}
}

func TestCurrentUnifiedReviewResolvesCandidateAndKeepsHistoryExplicit(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	packet, err := Build(root, []string{"developer/policy/RECORD/MPD-RECORD-0001.yaml"},
		[]string{".ptsip/contracts/policy/Rules/RECORD_rules.json#/$id"})
	if err != nil {
		t.Fatal(err)
	}
	target := packet.Targets[0]
	if target.Selection == nil || target.SelectionProblem != "" {
		t.Fatalf("unified selection failed: %+v", target)
	}
	selection := target.Selection
	if selection.PolicyKind != "PTSIP_DEVELOPER_POLICY" || selection.RootFamily != "RECORD" ||
		selection.SchemaIDs["public"] != "urn:ptsip:candidate:policy:unified:v3" ||
		selection.ContractRefs["public_schema"] != ".ptsip/contracts/policy/unified/Unified_Policy.schema.json" ||
		selection.ContractRefs["unified_registry"] != ".ptsip/contracts/policy/unified/index.json" {
		t.Fatalf("wrong registered unified selection: %+v", selection)
	}
	if _, legacyFamily := selection.ContractRefs["family_fields"]; legacyFamily {
		t.Fatalf("legacy v2 Family contract was selected for a unified document: %+v", selection)
	}
	if selection.StructuralState != "CANDIDATE" || selection.PolicyConformance != "NOT_ASSERTED" ||
		selection.ExecutionAuthorized || selection.MutationAuthorized {
		t.Fatalf("candidate selection claimed normative authority: %+v", selection)
	}
	for _, ref := range target.Selection.ContractRefs {
		if strings.HasPrefix(ref, ".ptsip/contracts/policy/Rules/") {
			t.Fatal("historical source became selected contract")
		}
	}
	if len(packet.Sources) != 1 || packet.Sources[0].Value != "urn:ptsip:contract:policy:rules:RECORD:v2" {
		t.Fatal("explicit historical evidence lost")
	}
	if packet.SchemaValidation != "NOT_PERFORMED" || packet.MutationAuthorized {
		t.Fatal("review granted authority")
	}
	if packet.Files[".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json"] == "" {
		t.Fatal("selection input not captured")
	}
	if err := packet.Verify(root); err != nil {
		t.Fatal(err)
	}
}
