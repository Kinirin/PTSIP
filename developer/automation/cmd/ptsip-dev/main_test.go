package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCleanupCLIUsesRegisteredRepeatedScopes(t *testing.T) {
	result, err := run([]string{"markdown-cleanup", "inspect", "--scope", "developer/tests/policy", "--scope", "developer/tests/planning"})
	if err != nil {
		t.Fatal(err)
	}
	scopes, ok := result.(map[string]any)["scopes"].([]string)
	if !ok || !reflect.DeepEqual(scopes, []string{"developer/tests/policy", "developer/tests/planning"}) {
		t.Fatalf("scope arguments lost: %#v", result)
	}
}

func TestCLIRejectsUnregisteredRepetitionAndMissingRepeatedValues(t *testing.T) {
	for _, request := range [][]string{
		{"policy-resolver", "resolve", "--scope", ".", "--scope", "developer/automation", "--operation", "READ"},
		{"markdown-cleanup", "inspect", "--scope", "--scope", "developer/tests/policy"},
	} {
		if _, err := run(request); err == nil || !(strings.Contains(err.Error(), "duplicate option") || strings.Contains(err.Error(), "requires a value")) {
			t.Fatalf("invalid named arguments admitted: %v, error=%v", request, err)
		}
	}
}
