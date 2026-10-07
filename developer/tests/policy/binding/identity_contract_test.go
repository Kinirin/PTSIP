package binding_test

import (
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func contractRule(t *testing.T, repo *testrepo.Repository, policyID, section string) object {
	t.Helper()
	path := "developer/policy/" + policyID[4:len(policyID)-5] + "/" + policyID + ".yaml"
	policy, err := repo.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := policy["rules"].(object)[section].(object)
	if !ok {
		t.Fatalf("missing exact rule %s#%s", policyID, section)
	}
	return value
}

func TestPolicyPlanIdentityContractPreservesExactLogicalAndPhysicalSemantics(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	identity := contractRule(t, repo, "MPD-INFO-0001", "unit_mpd_spec_0023_759e9dad168d")
	if _, exists := identity["plan_id"]; exists {
		t.Fatal("retired plan_id field reintroduced")
	}
	for _, test := range []struct {
		key, role, pattern string
		stable             []string
	}{
		{"resolved_plan_id", "LOGICAL_PLAN_IDENTITY", `^PLN\.[A-Z]{4}\.[A-Z]{4}\.[A-Za-z0-9]{8}$`, []string{"stable_across_version_change", "stable_across_revision_change", "repository_unique"}},
		{"plan_file_id", "STABLE_PLAN_FILE_IDENTITY", `^PLANFILE\.[A-Z]{4}\.[A-Za-z0-9]{12}$`, []string{"stable_across_path_move", "stable_across_file_rename", "stable_across_content_change", "repository_unique"}},
	} {
		value := identity[test.key].(object)
		if value["role"] != test.role || value["current_identity_pattern"] != test.pattern {
			t.Fatal(test.key, value)
		}
		for _, field := range test.stable {
			if value[field] != true {
				t.Fatal(test.key, field, value)
			}
		}
	}
	if !reflect.DeepEqual(identity["registered_plan_id"], object{"separate_identity_supported": false, "responsibility_merged_into": "plan_file_id"}) {
		t.Fatal(identity["registered_plan_id"])
	}
	created := contractRule(t, repo, "MPD-INFO-0001", "unit_mpd_spec_0023_ba39fe03d76a")["CREATED"].(object)
	for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision", "plan_ref"} {
		if created[key] != "REQUIRED" {
			t.Fatal(key, created)
		}
	}
	management := contractRule(t, repo, "MPD-CTRL-0001", "unit_mpd_0013_55660a13d7b1")
	if !reflect.DeepEqual(management["relationship_resolution"].(object)["exact_lookup_keys"], []any{"binding_id", "policy_ref", "resolved_plan_id", "plan_file_id", "plan_ref"}) {
		t.Fatal(management)
	}
	if management["plan_materialization"].(object)["separate_registered_plan_id_allocation"] != "FORBIDDEN" {
		t.Fatal(management)
	}
	movement := contractRule(t, repo, "MPD-CHANGE-0001", "unit_mpd_0013_ca5439755018")
	for name, want := range map[string]object{
		"same_plan_path_move":         {"resolved_plan_id": "SAME", "plan_file_id": "SAME", "plan_ref": "DIFFERENT", "outcome": "MOVE"},
		"plan_file_replacement":       {"resolved_plan_id": "SAME", "plan_file_id": "DIFFERENT", "outcome": "FILE_REPLACEMENT"},
		"plan_file_identity_conflict": {"resolved_plan_id": "DIFFERENT", "plan_file_id": "SAME", "outcome": "PLAN_FILE_ID_CONFLICT"},
	} {
		if !reflect.DeepEqual(movement[name].(object)["classification"], want) {
			t.Fatal(name, movement[name])
		}
	}
	mutation := contractRule(t, repo, "MPD-CTRL-0001", "unit_mpd_0013_ece0f737d29d")["movement_mutation_contract"].(object)
	if !reflect.DeepEqual(mutation["required_inputs"], []any{"binding_id", "policy_ref", "resolved_plan_id", "plan_file_id", "from_plan_ref", "to_plan_ref"}) || !reflect.DeepEqual(mutation["forbidden_inputs"], []any{"version", "revision"}) || !reflect.DeepEqual(mutation["update_scope"], []any{"plan_ref"}) || mutation["stale_state_protection"] != "EXPECTED_REGISTRY_DIGEST_CAS" {
		t.Fatal(mutation)
	}
	scope := contractRule(t, repo, "MPD-ASSURE-0002", "unit_mpd_veri_0001_85b7db19eee4")["verification_scope"].([]any)
	for _, key := range []string{"RESOLVED_PLAN_ID_BINDING", "PLAN_FILE_ID_BINDING", "PLAN_REF_BINDING"} {
		found := false
		for _, item := range scope {
			if item == key {
				found = true
			}
			if item == "PLAN_ID_BINDING" {
				t.Fatal("retired PLAN_ID_BINDING scope")
			}
		}
		if !found {
			t.Fatal("missing verification scope", key)
		}
	}
}

const outputSchemaPath = "developer/policy/schemas/management/verification/policy-plan-consistency-report.schema.json"
const outputSchemaVersion = "ptsip-management-verification-policy-plan-consistency-report/v1"
const outputSchemaID = "urn:ptsip:management-verification:policy-plan-consistency-report:v1"

func TestConsistencyOutputSchemaIsExactAndOwnsOnlyInstanceIdentity(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	schema, err := repo.Read(outputSchemaPath)
	if err != nil {
		t.Fatal(err)
	}
	if schema["$id"] != outputSchemaID || schema["additionalProperties"] != true || !reflect.DeepEqual(schema["required"], []any{"schema_version"}) || !reflect.DeepEqual(schema["properties"], object{"schema_version": object{"const": outputSchemaVersion}}) {
		t.Fatal(schema)
	}
	if err := repo.Validate(outputSchemaPath, object{"schema_version": outputSchemaVersion}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Validate(outputSchemaPath, object{"schema_version": "wrong/v1"}); err == nil {
		t.Fatal("wrong output identity admitted")
	}
}

func TestConsistencyOutputPolicyKeepsVerificationAndOutputOwnershipSeparate(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	rule := contractRule(t, repo, "MPD-CNTR-0002", "unit_mpd_veri_0006_6beb693dae2a")
	if rule["verification_semantics_ref"] != "MPD-VERI-0001" {
		t.Fatal(rule)
	}
	artifact := rule["schema_artifact"].(object)
	if artifact["path"] != outputSchemaPath || artifact["instance_schema_version"] != outputSchemaVersion || artifact["schema_document_id"] != outputSchemaID || artifact["schema_document_is_management_policy"] != false {
		t.Fatal(artifact)
	}
	boundary := rule["authority_boundary"].(object)
	if boundary["verification_semantics_owner"] != "MPD-VERI-0001" || boundary["verification_output_semantics_owner"] != "MPD-VERI-0006" {
		t.Fatal(boundary)
	}
	// Historical labels are audited through the registered migration graph; current
	// normative values above are consumed from their exact Root owner and section.
	graph, err := repo.Read("developer/policy/registries/root-family-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range graph["sources"].([]any) {
		source := raw.(object)
		if source["source_policy_id"] != "MPD-VERI-0006" {
			continue
		}
		want := object{"id": "MPD-VERI-0006", "version": "0.1", "title": "Policy-Plan Consistency Verification Output Contract", "status": "DRAFT"}
		if !reflect.DeepEqual(source["header"].(object)["policy"], want) {
			t.Fatal(source["header"])
		}
		return
	}
	t.Fatal("missing historical output contract identity")
}
