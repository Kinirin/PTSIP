package schema_check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovedResourceReadsOnlyExplicitApprovalPaths(t *testing.T) {
	root := t.TempDir()
	relative := policyVersionPath
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"$id":"urn:example:approved","type":"object"}`), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := approvedResource(root, relative)
	if err != nil || doc["$id"] != "urn:example:approved" {
		t.Fatalf("approved schema not read: %v %v", doc, err)
	}
	if _, err = approvedResource(root, "developer/automation/policy/lifecycle/version_contract.go"); err == nil || !strings.Contains(err.Error(), "UNAPPROVED_SCHEMA_RESOURCE") {
		t.Fatalf("unapproved read should fail: %v", err)
	}
}

func TestApprovedResourceRejectsMissingIDAndMalformed(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, filepath.FromSlash(policyVersionPath))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ raw, problem string }{
		{`{"type":"object"}`, "APPROVED_SCHEMA_RESOURCE_ID_REQUIRED"},
		{`{"$id":`, "INVALID_APPROVED_SCHEMA_RESOURCE"},
	} {
		if err := os.WriteFile(target, []byte(tc.raw), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := approvedResource(root, policyVersionPath)
		if err == nil || !strings.Contains(err.Error(), tc.problem) {
			t.Fatalf("expected %s: %v", tc.problem, err)
		}
	}
}

func TestApprovedFlatStatusProbeMissingResourcesIsUnresolved(t *testing.T) {
	report := probeApprovedFlatPolicyStatus(t.TempDir(), Engine{})
	if report.Status != "UNRESOLVED" || !strings.Contains(report.Problem, "APPROVED_SCHEMA_RESOURCE_UNAVAILABLE") {
		t.Fatalf("fail-closed resource probe required: %+v", report)
	}
}

func statusTestRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func statusTestSchema(t *testing.T, root string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(publicPolicyStatusPath)))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}
func statusTestSources(t *testing.T, root string) string {
	t.Helper()
	dest := t.TempDir()
	for _, family := range []string{"APPROVED", "DRAFT", "RETIRED"} {
		rel := ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_" + family + ".json"
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dest
}
func TestSharedStatusReferencesMatchLifecycleContracts(t *testing.T) {
	root := statusTestRoot(t)
	schema := statusTestSchema(t, root)
	resources, err := verifySharedStatusLifecycleRefs(root, schema)
	if err != nil {
		t.Fatalf("status source references invalid: %v", err)
	}
	if len(resources) != 3 {
		t.Fatalf("expected three native schema resources, got %d", len(resources))
	}
	props := schema["properties"].(map[string]any)
	status := props["status"].(map[string]any)
	if _, ok := status["enum"]; ok {
		t.Fatal("public schema must not duplicate lifecycle enum")
	}
	if _, ok := status["type"]; ok {
		t.Fatal("status type must come from source references")
	}
	oneOf := status["oneOf"].([]any)
	original := append([]any(nil), oneOf...)
	status["oneOf"] = original[:2]
	if _, err := verifySharedStatusLifecycleRefs(root, schema); err == nil {
		t.Fatal("missing source reference must fail")
	}
	status["oneOf"] = original
	status["oneOf"] = []any{original[0], original[0], original[2]}
	if _, err := verifySharedStatusLifecycleRefs(root, schema); err == nil {
		t.Fatal("duplicate source reference must fail")
	}
	status["oneOf"] = original
	status["enum"] = []any{"APPROVED.Default"}
	if _, err := verifySharedStatusLifecycleRefs(root, schema); err == nil {
		t.Fatal("duplicate public enum must fail")
	}
	delete(status, "enum")
	status["oneOf"] = []any{map[string]any{"$ref": "urn:ptsip:unknown"}, original[1], original[2]}
	if _, err := verifySharedStatusLifecycleRefs(root, schema); err == nil {
		t.Fatal("unknown source reference must fail")
	}
}

func TestStatusSourceUpdatesNeedNoPublicSchemaEdit(t *testing.T) {
	root := statusTestRoot(t)
	schema := statusTestSchema(t, root)
	fixture := statusTestSources(t, root)
	approved := ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json"
	file := filepath.Join(fixture, filepath.FromSlash(approved))
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	primary := doc["properties"].(map[string]any)["status"].(map[string]any)
	values := primary["enum"].([]any)
	primary["enum"] = append(values, "APPROVED.FutureState")
	changed, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, changed, 0644); err != nil {
		t.Fatal(err)
	}
	resources, err := verifySharedStatusLifecycleRefs(fixture, schema)
	if err != nil {
		t.Fatalf("adding a state only to source must be valid: %v", err)
	}
	approvedResource := resources[0].(map[string]any)
	got := approvedResource["properties"].(map[string]any)["status"].(map[string]any)["enum"].([]any)
	if len(got) != len(values)+1 {
		t.Fatalf("new state lost: %v", got)
	}
	primary["enum"] = append(values, values[0])
	duplicate, _ := json.Marshal(doc)
	if err := os.WriteFile(file, duplicate, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "LIFECYCLE_SOURCE_ENUM_DUPLICATE") {
		t.Fatalf("duplicate state must fail closed: %v", err)
	}
	primary["enum"] = append(values, "DRAFT.NotApproved")
	wrong, _ := json.Marshal(doc)
	if err := os.WriteFile(file, wrong, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "LIFECYCLE_SOURCE_ENUM_INVALID") {
		t.Fatalf("wrong family prefix must fail closed: %v", err)
	}
}

func TestStatusSourceMissingAndLegacyFormatFailClosed(t *testing.T) {
	root := statusTestRoot(t)
	schema := statusTestSchema(t, root)
	fixture := statusTestSources(t, root)
	rel := ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json"
	path := filepath.Join(fixture, filepath.FromSlash(rel))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	props := doc["properties"].(map[string]any)
	props["substatus"] = map[string]any{"enum": []any{"Default"}}
	changed, _ := json.Marshal(doc)
	if err := os.WriteFile(path, changed, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "LEGACY_SUBSTATUS") {
		t.Fatalf("legacy split schema must fail closed: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "SOURCE_UNAVAILABLE") {
		t.Fatalf("missing source must fail closed: %v", err)
	}
}
