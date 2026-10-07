package binding_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func verifyConsistency(t *testing.T, store *binding.Store) object {
	t.Helper()
	result, err := store.VerifyConsistency()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func requireFailure(t *testing.T, result object, code string) {
	t.Helper()
	if result["status"] != "FAIL" {
		t.Fatalf("expected failure: %#v", result)
	}
	for _, raw := range result["failures"].([]any) {
		if raw.(object)["code"] == code {
			return
		}
	}
	t.Fatalf("failure %s absent: %#v", code, result)
}

func TestEmptyConsistencyRegistryDoesNotInventTargets(t *testing.T) {
	_, store := emptyFixture(t)
	result := verifyConsistency(t, store)
	if result["status"] != "PASS" || result["binding_count"] != 0 || result["checked_binding_count"] != 0 || len(result["failures"].([]any)) != 0 {
		t.Fatal(result)
	}
}

func TestNotCreatedBindingIsAValidRegisteredRelationship(t *testing.T) {
	_, store := emptyFixture(t)
	if _, err := store.CreateBinding(policyID); err != nil {
		t.Fatal(err)
	}
	result := verifyConsistency(t, store)
	if result["status"] != "PASS" || result["binding_count"] != 1 || result["checked_binding_count"] != 1 {
		t.Fatal(result)
	}
}

func TestCreatedBindingPassesExactConsistency(t *testing.T) {
	repo, store := emptyFixture(t)
	materialize(t, repo, store, currentPlan)
	before := registryBytes(t, repo)
	result := verifyConsistency(t, store)
	if result["status"] != "PASS" || result["binding_count"] != 1 || len(result["failures"].([]any)) != 0 {
		t.Fatal(result)
	}
	requireUnchanged(t, repo, before)
}

func TestConsistencyVerifierIsReadOnlyWhenPlanMovementIsDetected(t *testing.T) {
	repo, store := emptyFixture(t)
	materialize(t, repo, store, oldPlan)
	from, _ := repo.Path(oldPlan)
	to, _ := repo.Path(movedPlan)
	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
	before := registryBytes(t, repo)
	requireFailure(t, verifyConsistency(t, store), "PLAN_REF_RECONCILE_REQUIRED")
	requireUnchanged(t, repo, before)
}

func TestConsistencyVerifierFailsWhenFileIdentityCannotBeResolved(t *testing.T) {
	repo, store := emptyFixture(t)
	materialize(t, repo, store, currentPlan)
	path, _ := repo.Path(currentPlan)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	before := registryBytes(t, repo)
	requireFailure(t, verifyConsistency(t, store), "PLAN_REF_UNRESOLVED")
	requireUnchanged(t, repo, before)
}

func TestConsistencyVerifierRejectsVersionCurrentnessMismatch(t *testing.T) {
	repo, store := emptyFixture(t)
	materialize(t, repo, store, currentPlan)
	writePlan(t, repo, currentPlan, object{"version": "2.0"})
	before := registryBytes(t, repo)
	requireFailure(t, verifyConsistency(t, store), "PLAN_VERSION_MISMATCH")
	requireUnchanged(t, repo, before)
}

func TestConsistencyVerifierFailsClosedOnDuplicateFileIdentity(t *testing.T) {
	repo, store := emptyFixture(t)
	materialize(t, repo, store, currentPlan)
	writePlan(t, repo, "developer/planning/duplicate/plan.yaml", nil)
	before := registryBytes(t, repo)
	requireFailure(t, verifyConsistency(t, store), "PLAN_FILE_ID_AMBIGUOUS")
	requireUnchanged(t, repo, before)
}

func TestConsistencySchemaFailureIsReportedWithoutMutation(t *testing.T) {
	repo, store := emptyFixture(t)
	payload := registry()
	payload["schema_version"] = "ptsip-policy-plan-bindings/v1"
	testrepo.Write(t, repo, binding.BindingRegistryPath, payload)
	before := registryBytes(t, repo)
	result := verifyConsistency(t, store)
	requireFailure(t, result, "BINDING_REGISTRY_SCHEMA_INVALID")
	if result["binding_count"] != 0 || result["checked_binding_count"] != 0 {
		t.Fatal(result)
	}
	requireUnchanged(t, repo, before)
}

func TestConsistencyRelationsPreserveExactRootSections(t *testing.T) {
	repo, store := emptyFixture(t)
	writePlan(t, repo, currentPlan, nil)
	first, second := created(), created()
	first["plan_ref"], second["plan_ref"] = currentPlan, currentPlan
	first["policy_sections"], second["policy_sections"] = []any{firstSection}, []any{secondSection}
	second["binding_id"] = "PPB-0002"
	testrepo.Write(t, repo, binding.BindingRegistryPath, registry(first, second))
	result := verifyConsistency(t, store)
	if result["status"] != "PASS" || result["checked_binding_count"] != 2 {
		t.Fatal(result)
	}
	second["policy_sections"] = []any{firstSection}
	testrepo.Write(t, repo, binding.BindingRegistryPath, registry(first, second))
	requireFailure(t, verifyConsistency(t, store), "DUPLICATE_CREATED_RELATION")
}
