package binding_test

import (
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

type object = map[string]any

func registry(rows ...object) object {
	entries := []any{}
	for _, row := range rows {
		entries = append(entries, row)
	}
	return object{"schema_version": "ptsip-policy-plan-bindings/v2", "registry_role": "POLICY_PLAN_BINDING", "schema_ref": binding.BindingSchemaPath, "bindings": entries}
}

func created() object {
	return object{"binding_id": "PPB-0001", "policy_ref": "MPD-INFO-0001", "planning_state": "CREATED", "resolved_plan_id": "PLN.MIGR.READ.A7k2Q9mX", "plan_file_id": "PLANFILE.MAIN.H7sP2kQ9mXa4", "version": "1.0", "revision": "Rev.0001", "plan_ref": "developer/planning/0.4.0/example.yaml"}
}

func clone(value object) object {
	result := object{}
	for key, item := range value {
		result[key] = item
	}
	return result
}

func TestBindingSchemaStatesPreserveRequiredAndForbiddenFields(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	if err := repo.Validate(binding.BindingSchemaPath, registry()); err != nil {
		t.Fatal(err)
	}
	notCreated := object{"binding_id": "PPB-0001", "policy_ref": "MPD-INFO-0001", "planning_state": "NOT_CREATED"}
	if err := repo.Validate(binding.BindingSchemaPath, registry(notCreated)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Validate(binding.BindingSchemaPath, registry(created())); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"resolved_plan_id", "plan_file_id", "version", "revision", "plan_ref"} {
		t.Run("forbidden before materialization/"+field, func(t *testing.T) {
			candidate := clone(notCreated)
			candidate[field] = created()[field]
			if err := repo.Validate(binding.BindingSchemaPath, registry(candidate)); err == nil {
				t.Fatalf("accepted %s in NOT_CREATED", field)
			}
		})
		t.Run("required after materialization/"+field, func(t *testing.T) {
			candidate := created()
			delete(candidate, field)
			if err := repo.Validate(binding.BindingSchemaPath, registry(candidate)); err == nil {
				t.Fatalf("accepted CREATED without %s", field)
			}
		})
	}
	for _, field := range []string{"plan_id", "registered_plan_id"} {
		t.Run("retired identity/"+field, func(t *testing.T) {
			candidate := created()
			candidate[field] = "legacy"
			if err := repo.Validate(binding.BindingSchemaPath, registry(candidate)); err == nil {
				t.Fatalf("accepted retired field %s", field)
			}
		})
	}
}

func TestBindingSchemaUsesExactIdentitiesAndDoesNotInventVersionFormats(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, test := range []struct{ field, value string }{
		{"resolved_plan_id", "PLN.MIGRATION.READ.A7k2Q9mX"}, {"plan_file_id", "PLANFILE.MAIN.short"}, {"version", ""}, {"revision", ""}, {"policy_ref", "MPD-0013"},
	} {
		t.Run(test.field+"/"+test.value, func(t *testing.T) {
			candidate := created()
			candidate[test.field] = test.value
			if err := repo.Validate(binding.BindingSchemaPath, registry(candidate)); err == nil {
				t.Fatalf("accepted invalid %s: %q", test.field, test.value)
			}
		})
	}
	candidate := created()
	candidate["version"], candidate["revision"] = "custom-version", "custom-revision"
	if err := repo.Validate(binding.BindingSchemaPath, registry(candidate)); err != nil {
		t.Fatal(err)
	}
}

func TestBindingSchemaPreservesManyToManyPlanIdentityReuse(t *testing.T) {
	first, second := created(), created()
	second["binding_id"], second["policy_ref"] = "PPB-0002", "MPD-ASSURE-0001"
	if err := testrepo.Open(testrepo.Root(t)).Validate(binding.BindingSchemaPath, registry(first, second)); err != nil {
		t.Fatal(err)
	}
}
