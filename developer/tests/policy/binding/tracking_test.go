package binding_test

import (
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func trackingFixture(t *testing.T, ref string) (*testrepo.Repository, *binding.Store) {
	t.Helper()
	repo, store := emptyFixture(t)
	row := created()
	row["plan_ref"] = ref
	testrepo.Write(t, repo, binding.BindingRegistryPath, registry(row))
	return repo, store
}

func TestResolverUsesExactV2IdentityKeys(t *testing.T) {
	_, store := trackingFixture(t, oldPlan)
	for _, query := range []object{
		{"resolved_plan_id": created()["resolved_plan_id"]},
		{"resolved_plan_id": created()["resolved_plan_id"], "plan_file_id": created()["plan_file_id"]},
		{"binding_id": "PPB-0001", "policy_ref": policyID, "plan_ref": oldPlan},
	} {
		result, err := store.ResolveBindings(query)
		if err != nil || result["status"] != "BOUND" || len(result["bindings"].([]any)) != 1 {
			t.Fatalf("exact resolution: query=%#v result=%#v error=%v", query, result, err)
		}
		for key, value := range query {
			if result[key] != value {
				t.Fatalf("query identity %s not preserved", key)
			}
		}
	}
}

func TestResolverRequiresAtLeastOneExactKey(t *testing.T) {
	_, store := trackingFixture(t, oldPlan)
	_, err := store.ResolveBindings(object{})
	requireCode(t, err, "BINDING_QUERY_EMPTY")
}

func TestTrackerReportsCurrentForExactRegisteredLocation(t *testing.T) {
	repo, store := trackingFixture(t, currentPlan)
	writePlan(t, repo, currentPlan, object{"version": "1.1", "revision": "Rev.0002"})
	before := registryBytes(t, repo)
	result, err := store.TrackPlanRef("PPB-0001", false)
	if err != nil || result["status"] != "CURRENT" || result["discovered_plan_ref"] != currentPlan || result["changed"] != false {
		t.Fatalf("current tracking: %#v, error=%v", result, err)
	}
	requireUnchanged(t, repo, before)
}

func TestTrackerReportsUnresolvedWhenPlanFileIdentityIsMissing(t *testing.T) {
	repo, store := trackingFixture(t, oldPlan)
	before := registryBytes(t, repo)
	result, err := store.TrackPlanRef("PPB-0001", false)
	if err != nil || result["status"] != "UNRESOLVED" || result["discovered_plan_ref"] != nil || len(result["candidates"].([]any)) != 0 {
		t.Fatalf("unresolved tracking: %#v, error=%v", result, err)
	}
	requireUnchanged(t, repo, before)
}

func TestTrackerDetectsMovedPlanWithoutInferringItsPath(t *testing.T) {
	repo, store := trackingFixture(t, oldPlan)
	writePlan(t, repo, movedPlan, nil)
	before := registryBytes(t, repo)
	result, err := store.TrackPlanRef("PPB-0001", false)
	if err != nil || result["status"] != "RECONCILE_REQUIRED" || result["current_plan_ref"] != oldPlan || result["discovered_plan_ref"] != movedPlan || !reflect.DeepEqual(result["candidates"], []any{movedPlan}) {
		t.Fatalf("moved tracking: %#v, error=%v", result, err)
	}
	requireUnchanged(t, repo, before)
}

func TestTrackerApplyUpdatesOnlyPlanReference(t *testing.T) {
	repo, store := trackingFixture(t, oldPlan)
	writePlan(t, repo, movedPlan, object{"version": "1.1", "revision": "Rev.0002"})
	result, err := store.TrackPlanRef("PPB-0001", true)
	if err != nil || result["status"] != "RECONCILED" || result["applied"] != true {
		t.Fatalf("applied tracking: %#v, error=%v", result, err)
	}
	resolved, err := store.ResolveBindings(object{"binding_id": "PPB-0001"})
	if err != nil {
		t.Fatal(err)
	}
	want := created()
	want["plan_ref"] = movedPlan
	if !reflect.DeepEqual(resolved["bindings"].([]any)[0], want) {
		t.Fatalf("tracking changed identity/version/revision: %#v", resolved)
	}
}

func TestTrackerFailsClosedOnDuplicatePlanFileIdentity(t *testing.T) {
	repo, store := trackingFixture(t, oldPlan)
	writePlan(t, repo, "developer/planning/a/plan.yaml", nil)
	writePlan(t, repo, "developer/planning/b/plan.yaml", nil)
	before := registryBytes(t, repo)
	_, err := store.TrackPlanRef("PPB-0001", false)
	requireCode(t, err, "PLAN_FILE_ID_AMBIGUOUS")
	requireUnchanged(t, repo, before)
}

func TestTrackerFailsClosedWhenFileIdentityBindsAnotherLogicalPlan(t *testing.T) {
	repo, store := trackingFixture(t, oldPlan)
	writePlan(t, repo, movedPlan, object{"resolved_plan_id": "PLN.VERI.BIND.Q8m2Za1K"})
	before := registryBytes(t, repo)
	_, err := store.TrackPlanRef("PPB-0001", false)
	requireCode(t, err, "PLAN_FILE_ID_CONFLICT")
	requireUnchanged(t, repo, before)
}

func TestTrackerIgnoresYAMLWithoutExplicitPlanIdentity(t *testing.T) {
	repo, store := trackingFixture(t, oldPlan)
	testrepo.Write(t, repo, "developer/planning/index.yaml", object{"schema_version": "unrelated"})
	writePlan(t, repo, movedPlan, nil)
	result, err := store.TrackPlanRef("PPB-0001", false)
	if err != nil || result["status"] != "RECONCILE_REQUIRED" || !reflect.DeepEqual(result["candidates"], []any{movedPlan}) {
		t.Fatalf("implicit YAML identity inferred: %#v error=%v", result, err)
	}
}
