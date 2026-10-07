package machine

import (
	"reflect"
	"strings"
	"testing"
)

func TestSelectionDocumentSchemaCoverageUsesNativeValidationWithoutExecution(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, vector := range []struct {
		kind    string
		payload Object
	}{
		{"request", Object{"kind": "CASE_IDS", "case_ids": []any{"b", "a"}}},
		{"request", Object{"kind": "RULE_REF", "rule_ref": "explicit.selection.release"}},
		{"rule", Object{"id": "explicit.selection.release", "kind": "CASE_ID_SET", "case_ids": []any{"b", "a"}}},
		{"result", Object{"state": "RESOLVED", "case_ids": []any{"a", "b"}, "diagnostics": []any{}}},
		{"result", Object{"state": "REJECTED", "case_ids": []any{}, "diagnostics": []any{Object{"code": "UNKNOWN_CASE", "location": "$.case_ids[0]", "reference": "missing"}}}},
	} {
		result, err := r.ValidateSelectionDocument(vector.kind, vector.payload)
		if err != nil || !reflect.DeepEqual(result, Object{"status": "PASS", "kind": vector.kind, "execution_performed": false}) {
			t.Fatal(result, err)
		}
	}
	for _, payload := range []Object{{}, {"kind": "CASE_IDS", "case_ids": []any{}}, {"kind": "CASE_IDS", "case_ids": []any{"a", "a"}}, {"kind": "CASE_IDS", "case_ids": []any{" a"}}, {"kind": "CASE_IDS", "case_ids": []any{""}}, {"kind": "RULE_REF", "rule_ref": ""}, {"kind": "RULE_REF", "rule_ref": "release", "case_ids": []any{"a"}}, {"kind": "CASE_IDS", "case_ids": []any{"a"}, "unknown": true}, {"kind": "AUTO", "case_ids": []any{"a"}}} {
		if _, err := r.ValidateSelectionDocument("request", payload); err == nil {
			t.Fatal("invalid request accepted", payload)
		}
	}
	for _, payload := range []Object{{"state": "RESOLVED", "case_ids": []any{}, "diagnostics": []any{}}, {"state": "RESOLVED", "case_ids": []any{"a", "a"}, "diagnostics": []any{}}, {"state": "REJECTED", "case_ids": []any{"a"}, "diagnostics": []any{Object{"code": "UNKNOWN_CASE", "location": "$"}}}, {"state": "REJECTED", "case_ids": []any{}, "diagnostics": []any{}}, {"state": "REJECTED", "case_ids": []any{}, "diagnostics": []any{Object{"code": "UNKNOWN_ERROR", "location": "$"}}}} {
		if _, err := r.ValidateSelectionDocument("result", payload); err == nil {
			t.Fatal("invalid or partially usable result accepted", payload)
		}
	}
	for _, payload := range []Object{{"id": "release", "kind": "CASE_ID_SET", "case_ids": []any{"a", "a"}}, {"id": "release", "kind": "PATH_GLOB", "case_ids": []any{"a"}}} {
		if _, err := r.ValidateSelectionDocument("rule", payload); err == nil {
			t.Fatal("invalid rule accepted", payload)
		}
	}
	if _, err := r.ValidateSelectionDocument("other", Object{}); err == nil || !strings.HasPrefix(err.Error(), "UNKNOWN_SELECTION_DOCUMENT") {
		t.Fatal("unknown document kind fell back", err)
	}
	for _, vector := range []struct {
		payload Object
		code    string
	}{
		{Object{"state": "RESOLVED", "case_ids": []any{"b", "a"}, "diagnostics": []any{}}, "NONDETERMINISTIC_SELECTION_ORDER"},
		{Object{"state": "REJECTED", "case_ids": []any{}, "diagnostics": []any{Object{"code": "UNKNOWN_CASE", "location": "$.z"}, Object{"code": "UNKNOWN_CASE", "location": "$.a"}}}, "NONDETERMINISTIC_DIAGNOSTIC_ORDER"},
	} {
		if _, err := r.ValidateSelectionDocument("result", vector.payload); err == nil || !strings.HasPrefix(err.Error(), vector.code) {
			t.Fatal("ordering contract changed", err)
		}
	}
}
