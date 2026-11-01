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
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(policyStatusVocabularyPath)))
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dest, filepath.FromSlash(policyStatusVocabularyPath))
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, raw, 0644); err != nil {
		t.Fatal(err)
	}
	return dest
}
func TestSharedStatusReferencesMatchLifecycleContract(t *testing.T) {
	root := statusTestRoot(t)
	schema := statusTestSchema(t, root)
	resources, err := verifySharedStatusLifecycleRefs(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected one native schema resource, got %d", len(resources))
	}
	status := schema["properties"].(map[string]any)["status"].(map[string]any)
	if len(status) != 1 || status["$ref"] != policyStatusVocabularyID {
		t.Fatal("public status must refer to the single vocabulary without copying its enum")
	}
	original := status["$ref"]
	status["$ref"] = "urn:ptsip:unknown"
	if _, err := verifySharedStatusLifecycleRefs(root, schema); err == nil {
		t.Fatal("unknown source reference accepted")
	}
	status["$ref"] = original
	status["enum"] = []any{"APPROVED.Default"}
	if _, err := verifySharedStatusLifecycleRefs(root, schema); err == nil {
		t.Fatal("duplicate public vocabulary accepted")
	}
}
func TestStatusSourceUpdatesNeedNoPublicSchemaEdit(t *testing.T) {
	root := statusTestRoot(t)
	schema := statusTestSchema(t, root)
	fixture := statusTestSources(t, root)
	file := filepath.Join(fixture, filepath.FromSlash(policyStatusVocabularyPath))
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	values := doc["enum"].([]any)
	write := func(values []any) {
		t.Helper()
		doc["enum"] = values
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(append(append([]any(nil), values...), "APPROVED.FutureState"))
	resources, err := verifySharedStatusLifecycleRefs(fixture, schema)
	if err != nil {
		t.Fatalf("source-only state update rejected: %v", err)
	}
	if len(resources[0].(map[string]any)["enum"].([]any)) != len(values)+1 {
		t.Fatal("new source value lost")
	}
	write(append(append([]any(nil), values...), values[0]))
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "ENUM_DUPLICATE") {
		t.Fatalf("duplicate status must fail closed: %v", err)
	}
	write(append(append([]any(nil), values...), "UNKNOWN.NotAdmitted"))
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "ENUM_INVALID") {
		t.Fatalf("undeclared status family must fail closed: %v", err)
	}
	write([]any{"APPROVED.Default"})
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "FAMILY_MISSING") {
		t.Fatalf("dropping lifecycle families must fail closed: %v", err)
	}
}
func TestStatusSourceMissingAndLegacyFormatFailClosed(t *testing.T) {
	root := statusTestRoot(t)
	schema := statusTestSchema(t, root)
	fixture := statusTestSources(t, root)
	file := filepath.Join(fixture, filepath.FromSlash(policyStatusVocabularyPath))
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["properties"] = map[string]any{"substatus": map[string]any{"enum": []any{"Default"}}}
	data, _ := json.Marshal(doc)
	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "LEGACY_SUBSTATUS") {
		t.Fatalf("legacy status layout must fail closed: %v", err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySharedStatusLifecycleRefs(fixture, schema); err == nil || !strings.Contains(err.Error(), "SOURCE_UNAVAILABLE") {
		t.Fatalf("missing unified source must fail closed: %v", err)
	}
}
