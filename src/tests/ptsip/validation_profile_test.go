package contracts_test

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func validationSelectedProfile() object {
	return object{
		"contract_id": "urn:profile:test", "version": "pp.2.00",
		"operations":      []any{"IDENTIFY", "VALIDATE", "ANALYZE", "CREATE_TARGET"},
		"schema_resource": "schemas/storage/profile/history/project-profile-pp.2.00.schema.json",
		"source_sha256":   strings.Repeat("a", 64), "lifecycle": "ACTIVE", "allowed": true,
		"notifications": []any{},
	}
}

func nativeProfileOutput(t *testing.T, operation string, input object) object {
	t.Helper()
	result, err := runProfileOperation(t, "validation", operation, input)
	if err != nil || result["status"] != "PASS" {
		t.Fatalf("native Profile operation failed: %v: %#v", err, result)
	}
	output, ok := result["output"].(map[string]any)
	if !ok {
		t.Fatalf("native Profile operation omitted object output: %#v", result)
	}
	return output
}

func nativeProfileFailure(t *testing.T, operation string, input object) {
	t.Helper()
	result, err := runProfileOperation(t, "validation", operation, input)
	if err == nil && result["status"] != "FAIL" {
		t.Fatalf("invalid native input was accepted: %#v", result)
	}
}

func TestNativeProfileValidationResponsibilities(t *testing.T) {
	t.Run("typed_identity_and_immutable_instance", func(t *testing.T) {
		content := "Profile 선언\n"
		input := object{
			"version": "pp.2.00", "filename_token": "pp2.00", "revision": "Rev.0042",
			"content": content, "profile_role": "PROJECT", "operation": "VALIDATE",
			"selected_contract": validationSelectedProfile(),
		}
		output := nativeProfileOutput(t, "validation.profile.identity", input)
		if output["status"] != "SUPPORTED" || output["contract_version"] != "pp.2.00" ||
			output["filename_token"] != "pp2.00" || output["parsed_revision"] != "Rev.0042" ||
			output["instance_revision"] != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content))) {
			t.Fatalf("identity/revision/content binding drifted: %#v", output)
		}
	})
	t.Run("operation_support_is_explicit", func(t *testing.T) {
		for _, input := range []object{
			{"version": "pp.1.02", "selected_contract": validationSelectedProfile()},
			{"version": "pp.2.00", "operation": "MIGRATE_SOURCE", "selected_contract": validationSelectedProfile()},
		} {
			if output := nativeProfileOutput(t, "validation.profile.identity", input); output["status"] != "UNSUPPORTED" {
				t.Fatalf("unsupported request was accepted: %#v", output)
			}
		}
	})
	t.Run("lifecycle_and_selected_admission", func(t *testing.T) {
		for _, tc := range []struct {
			lifecycle string
			newUse    bool
			allowed   bool
			status    string
		}{
			{"ACTIVE", true, true, "SUPPORTED"},
			{"LEGACY", false, true, "SUPPORTED"},
			{"LEGACY", true, true, "UNSUPPORTED"},
			{"ACTIVE", false, false, "UNSUPPORTED"},
			{"DEPRECATED", true, true, "SUPPORTED"},
		} {
			selected := validationSelectedProfile()
			selected["lifecycle"], selected["allowed"] = tc.lifecycle, tc.allowed
			output := nativeProfileOutput(t, "validation.profile.identity", object{
				"version": "pp.2.00", "new_registration": tc.newUse, "selected_contract": selected,
			})
			if output["status"] != tc.status {
				t.Fatalf("lifecycle %s/new=%v: %#v", tc.lifecycle, tc.newUse, output)
			}
			if tc.lifecycle == "DEPRECATED" {
				notifications := output["notifications"].([]any)
				if len(notifications) != 1 || notifications[0].(map[string]any)["code"] != "UPGRADE_REQUIRED" ||
					notifications[0].(map[string]any)["mandatory"] != true {
					t.Fatalf("mandatory transition notice is missing: %#v", notifications)
				}
			}
		}
	})
	t.Run("malformed_identity_rejected", func(t *testing.T) {
		for _, fields := range []object{
			{"version": "pp.2.0"}, {"version": "v2.00"}, {"version": "pp.02.00"},
			{"version": "pp.2.00", "revision": "Rev.0000"},
			{"version": "pp.2.00", "revision": "Rev.42"},
			{"version": "pp.2.00", "filename_token": "pp1.02"},
			{"version": "pp.2.00", "execute": true},
		} {
			fields["selected_contract"] = validationSelectedProfile()
			nativeProfileFailure(t, "validation.profile.identity", fields)
		}
	})
	t.Run("direct_transitions_do_not_create_transitive_edges", func(t *testing.T) {
		input := object{
			"source_contract_version": "pp.1.01", "target_contract_version": "pp.1.02",
			"transitions": []any{
				object{"from": "pp.1.01", "to": "pp.1.02", "kind": "SEMANTIC_MIGRATION"},
				object{"from": "pp.1.02", "to": "pp.2.00", "kind": "IDENTITY_ONLY"},
			},
		}
		output := nativeProfileOutput(t, "validation.profile.transition", input)
		if output["allowed"] != true || output["transition_kind"] != "SEMANTIC_MIGRATION" ||
			output["intermediate_materialization_allowed"] != false {
			t.Fatalf("exact registered edge failed: %#v", output)
		}
		input["target_contract_version"] = "pp.2.00"
		if output := nativeProfileOutput(t, "validation.profile.transition", input); output["allowed"] != false {
			t.Fatalf("unregistered transitive edge was inferred: %#v", output)
		}
		input["target_contract_version"], input["declared_transition"] = "pp.1.02", "IDENTITY_ONLY"
		if output := nativeProfileOutput(t, "validation.profile.transition", input); output["allowed"] != false {
			t.Fatalf("declared transition mismatch was accepted: %#v", output)
		}
	})
	t.Run("historical_transition_requires_exact_revision", func(t *testing.T) {
		revision := strings.Repeat("b", 40)
		input := object{
			"source_contract_version": "0.3.4-draft", "source_revision": revision,
			"target_contract_version": "pp.1.01", "transitions": []any{},
			"historical_bridges": []any{object{
				"declared_version": "0.3.4-draft", "specification_revision": revision,
				"target_contract": "pp.1.01", "transition_kind": "SEMANTIC_MIGRATION",
				"legacy_target_filename": "ptsip_0.3.6.yaml",
			}},
		}
		output := nativeProfileOutput(t, "validation.profile.transition", input)
		if output["allowed"] != true || output["legacy_target_filename"] != "ptsip_0.3.6.yaml" ||
			!reflect.DeepEqual(output["evidence_refs"], []any{revision}) {
			t.Fatalf("historical source evidence was lost: %#v", output)
		}
		input["source_revision"] = strings.Repeat("c", 40)
		if output := nativeProfileOutput(t, "validation.profile.transition", input); output["allowed"] != false {
			t.Fatalf("historical revision mismatch was accepted: %#v", output)
		}
	})
	t.Run("duplicate_transition_binding_rejected", func(t *testing.T) {
		edge := object{"from": "pp.1.01", "to": "pp.1.02", "kind": "SEMANTIC_MIGRATION"}
		nativeProfileFailure(t, "validation.profile.transition", object{
			"source_contract_version": "pp.1.01", "target_contract_version": "pp.1.02", "transitions": []any{edge, edge},
		})
	})
	t.Run("opaque_rule_ids_and_arbitrary_depth", func(t *testing.T) {
		records := []any{
			object{"ID": "임의 ID", "rule_value": object{"root_id": "child", "constant": "opaque"}},
			object{"ID": "child", "parent_id": "임의 ID", "rule_value": []any{false, "opaque"}},
			object{"ID": "third", "parent_id": "child"},
			object{"ID": "fourth", "parent_id": "third"},
		}
		output := nativeProfileOutput(t, "validation.profile.rules", object{"records": records})
		if output["valid"] != true || !reflect.DeepEqual(output["records"], records) {
			t.Fatalf("opaque data or explicit deeper graph was rejected/rewritten: %#v", output)
		}
	})
	t.Run("invalid_rule_relations_report_precise_codes", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			records []any
			code    string
		}{
			{"duplicate", []any{object{"ID": "same"}, object{"ID": "same"}}, "DUPLICATE_ID"},
			{"missing_parent", []any{object{"ID": "child", "parent_id": "absent"}}, "MISSING_PARENT"},
			{"cycle", []any{object{"ID": "a", "parent_id": "b"}, object{"ID": "b", "parent_id": "a"}}, "PARENT_CYCLE"},
			{"self_cycle", []any{object{"ID": "a", "parent_id": "a"}}, "PARENT_CYCLE"},
			{"nonchild_pointer", []any{object{"ID": "a", "rule_value": object{"root_id": "b"}}, object{"ID": "b"}}, "INVALID_CHILD_POINTER"},
			{"nonstring_pointer", []any{object{"ID": "a", "rule_value": object{"root_id": 42}}}, "INVALID_CHILD_POINTER"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				output := nativeProfileOutput(t, "validation.profile.rules", object{"records": tc.records})
				if output["valid"] != false {
					t.Fatalf("invalid graph accepted: %#v", output)
				}
				found := false
				for _, diagnostic := range output["diagnostics"].([]any) {
					found = found || diagnostic.(map[string]any)["code"] == tc.code
				}
				if !found {
					t.Fatalf("missing expected code %s: %#v", tc.code, output)
				}
			})
		}
	})
}
