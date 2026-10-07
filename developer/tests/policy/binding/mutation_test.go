package binding_test

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const policyID = "MPD-INFO-0001"
const currentPlan = "developer/planning/current/plan.yaml"
const oldPlan = "developer/planning/old/plan.yaml"
const movedPlan = "developer/planning/new/location.yaml"

func emptyFixture(t *testing.T) (*testrepo.Repository, *binding.Store) {
	t.Helper()
	repo, store := storeFixture(t)
	path, err := repo.Path(created()["plan_ref"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	testrepo.Write(t, repo, binding.BindingRegistryPath, registry())
	return repo, store
}

func writePlan(t *testing.T, repo *testrepo.Repository, ref string, changes object) {
	t.Helper()
	identity := object{}
	for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
		identity[key] = created()[key]
	}
	for key, value := range changes {
		identity[key] = value
	}
	testrepo.Write(t, repo, ref, object{"plan_identity": identity})
}

func linkValues(id, ref string) object {
	values := created()
	values["binding_id"], values["plan_ref"] = id, ref
	delete(values, "planning_state")
	return values
}

func materialize(t *testing.T, repo *testrepo.Repository, store *binding.Store, ref string) object {
	t.Helper()
	result, err := store.CreateBinding(policyID)
	if err != nil {
		t.Fatal(err)
	}
	writePlan(t, repo, ref, nil)
	result, err = store.LinkPlan(linkValues(result["binding"].(object)["binding_id"].(string), ref))
	if err != nil {
		t.Fatal(err)
	}
	return result["binding"].(object)
}

func moveValues(id, from, to string) object {
	return object{"binding_id": id, "policy_ref": policyID, "resolved_plan_id": created()["resolved_plan_id"], "plan_file_id": created()["plan_file_id"], "from_plan_ref": from, "to_plan_ref": to}
}

func registryBytes(t *testing.T, repo *testrepo.Repository) []byte {
	t.Helper()
	path, err := repo.Path(binding.BindingRegistryPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func requireUnchanged(t *testing.T, repo *testrepo.Repository, before []byte) {
	t.Helper()
	if !bytes.Equal(before, registryBytes(t, repo)) {
		t.Fatal("read-only or failed operation changed registry bytes")
	}
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !(err.Error() == code || strings.HasPrefix(err.Error(), code+":")) {
		t.Fatalf("want %s, got %v", code, err)
	}
}

func TestBindingIDsAreAllocatedWithoutInventingPlanFields(t *testing.T) {
	_, store := emptyFixture(t)
	for _, id := range []string{"PPB-0001", "PPB-0002"} {
		result, err := store.CreateBinding(policyID)
		want := object{"binding_id": id, "policy_ref": policyID, "planning_state": "NOT_CREATED"}
		if err != nil || !reflect.DeepEqual(result["binding"], want) {
			t.Fatalf("allocation: result=%#v, error=%v", result, err)
		}
	}
}

func TestLinkPlanUsesOnlyExactV2IdentityFields(t *testing.T) {
	repo, store := emptyFixture(t)
	allocated, err := store.CreateBinding(policyID)
	if err != nil {
		t.Fatal(err)
	}
	writePlan(t, repo, currentPlan, nil)
	values := linkValues("", currentPlan)
	delete(values, "binding_id")
	result, err := store.LinkPlan(values)
	if err != nil {
		t.Fatal(err)
	}
	want := created()
	want["binding_id"], want["plan_ref"] = allocated["binding"].(object)["binding_id"], currentPlan
	if !reflect.DeepEqual(result["binding"], want) {
		t.Fatalf("identity fields changed: %#v", result)
	}
	for _, retired := range []string{"plan_id", "registered_plan_id"} {
		if _, exists := result["binding"].(object)[retired]; exists {
			t.Fatalf("retired identity %s recreated", retired)
		}
	}
}

func TestLinkPlanRejectsEveryPlanDocumentIdentityMismatchWithoutMutation(t *testing.T) {
	for _, test := range []struct{ field, value, code string }{
		{"plan_file_id", "PLANFILE.MAIN.Z9ap7Kx21QmB", "PLAN_FILE_ID_MISMATCH"},
		{"resolved_plan_id", "PLN.VERI.BIND.Q8m2Za1K", "RESOLVED_PLAN_ID_MISMATCH"},
		{"version", "2.0", "PLAN_VERSION_MISMATCH"}, {"revision", "Rev.0002", "PLAN_REVISION_MISMATCH"},
	} {
		t.Run(test.field, func(t *testing.T) {
			repo, store := emptyFixture(t)
			if _, err := store.CreateBinding(policyID); err != nil {
				t.Fatal(err)
			}
			writePlan(t, repo, currentPlan, object{test.field: test.value})
			before := registryBytes(t, repo)
			_, err := store.LinkPlan(linkValues("PPB-0001", currentPlan))
			requireCode(t, err, test.code)
			requireUnchanged(t, repo, before)
		})
	}
}

func TestLinkPlanFailsClosedWhenUncreatedBindingIsAmbiguous(t *testing.T) {
	repo, store := emptyFixture(t)
	for range 2 {
		if _, err := store.CreateBinding(policyID); err != nil {
			t.Fatal(err)
		}
	}
	writePlan(t, repo, currentPlan, nil)
	values := linkValues("", currentPlan)
	delete(values, "binding_id")
	before := registryBytes(t, repo)
	_, err := store.LinkPlan(values)
	requireCode(t, err, "AMBIGUOUS_UNCREATED_BINDING")
	requireUnchanged(t, repo, before)
}

func TestMoveUpdatesOnlyPhysicalReferenceWithExactIdentityGuards(t *testing.T) {
	repo, store := emptyFixture(t)
	row := materialize(t, repo, store, oldPlan)
	writePlan(t, repo, movedPlan, object{"version": "1.1", "revision": "Rev.0002"})
	result, err := store.MovePlanRef(moveValues(row["binding_id"].(string), oldPlan, movedPlan))
	if err != nil {
		t.Fatal(err)
	}
	want := clone(row)
	want["plan_ref"] = movedPlan
	if !reflect.DeepEqual(result["binding"], want) {
		t.Fatalf("movement changed logical identity/currentness: %#v", result)
	}
	snapshot, err := store.LoadBindingRegistry(true)
	if err != nil || !reflect.DeepEqual(snapshot.Payload["bindings"].([]any)[0], want) {
		t.Fatalf("stored movement changed extra fields: %v", err)
	}
}

func TestMoveFailsClosedOnStaleFromReference(t *testing.T) {
	repo, store := emptyFixture(t)
	row := materialize(t, repo, store, oldPlan)
	writePlan(t, repo, movedPlan, nil)
	before := registryBytes(t, repo)
	_, err := store.MovePlanRef(moveValues(row["binding_id"].(string), "developer/planning/stale/plan.yaml", movedPlan))
	requireCode(t, err, "STALE_PLAN_REF")
	requireUnchanged(t, repo, before)
}

func TestMoveFailsClosedOnDestinationLogicalIdentityMismatch(t *testing.T) {
	repo, store := emptyFixture(t)
	row := materialize(t, repo, store, oldPlan)
	writePlan(t, repo, movedPlan, object{"resolved_plan_id": "PLN.VERI.BIND.Q8m2Za1K"})
	before := registryBytes(t, repo)
	_, err := store.MovePlanRef(moveValues(row["binding_id"].(string), oldPlan, movedPlan))
	requireCode(t, err, "RESOLVED_PLAN_ID_MISMATCH")
	requireUnchanged(t, repo, before)
}
