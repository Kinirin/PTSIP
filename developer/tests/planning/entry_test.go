package planning_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/planning"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const prereleasePlan = "developer/planning/0.3.8/0.3.8a3/index.yaml"
const prereleaseWU08 = "developer/planning/0.3.8/0.3.8a3/WU-08/WU-08.yaml"

func prereleaseFixture(t *testing.T) *readOnlyRepository {
	t.Helper()
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyCatalogContracts(t, repo)
	testrepo.CopyTree(t, repo, "developer/planning")
	testrepo.CopyFiles(t, repo, "developer/policy/registries/governance-source-registry.yaml", "developer/bindings/schemas/policy-plan-bindings.schema.json")
	return &readOnlyRepository{Repository: repo}
}

func readPlanning(t *testing.T, repo *readOnlyRepository, ref string) object {
	t.Helper()
	value, err := repo.Read(ref)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func requirePlanningError(t *testing.T, failures []string, expected string) {
	t.Helper()
	for _, failure := range failures {
		if strings.Contains(failure, expected) {
			return
		}
	}
	t.Fatalf("expected %q, got %v", expected, failures)
}

func TestRegisteredPrereleaseAndContinuationResolveExactly(t *testing.T) {
	repo := prereleaseFixture(t)
	if failures := planning.ValidatePlanning(repo); len(failures) != 0 {
		t.Fatal(failures)
	}
	for _, branch := range []string{"dev/0.3.8", "dev/0.3.8a3"} {
		result, err := planning.ResolvePlanningEntry(repo, branch)
		if err != nil || result["entry_document"] != prereleasePlan || result["plan_version"] != "0.3.8a3" || result["role"] != "INTEGRATION_CONTROL_PLANE" {
			t.Fatalf("%#v %v", result, err)
		}
	}
}

func TestSimilarUnregisteredPrereleaseBranchStaysUnresolved(t *testing.T) {
	repo := prereleaseFixture(t)
	if _, err := planning.ResolvePlanningEntry(repo, "dev/0.3.8a30"); err == nil || !strings.Contains(err.Error(), "no active exact planning entry") {
		t.Fatal(err)
	}
}

func TestDuplicateBranchRegistrationIsAmbiguous(t *testing.T) {
	repo := prereleaseFixture(t)
	index := readPlanning(t, repo, "developer/planning/index.yaml")
	index["plans"] = append(index["plans"].([]any), planning.Clone(index["plans"].([]any)[0].(object)))
	testrepo.Write(t, repo.Repository, "developer/planning/index.yaml", index)
	if failures := planning.ValidatePlanning(repo); len(failures) == 0 {
		t.Fatal("duplicate branch registration passed validation")
	}
	if _, err := planning.ResolvePlanningEntry(repo, "dev/0.3.8"); err == nil || !strings.Contains(err.Error(), "ambiguous planning entry") {
		t.Fatal(err)
	}
}

func TestInvalidRegisteredWorkUnitBlocksPlanningValidation(t *testing.T) {
	for _, tamper := range []string{"missing", "identity", "dependencies", "completion"} {
		t.Run(tamper, func(t *testing.T) {
			repo := prereleaseFixture(t)
			if tamper == "missing" {
				path, err := repo.Path(prereleaseWU08)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else {
				payload := readPlanning(t, repo, prereleaseWU08)
				switch tamper {
				case "identity":
					payload["plan_id"] = "OTHER_PLAN"
				case "dependencies":
					payload["work_unit"].(object)["depends_on"] = []any{}
				case "completion":
					payload["work_unit"].(object)["lifecycle"].(object)["status"] = "COMPLETE"
					delete(payload, "completion_evidence")
				}
				testrepo.Write(t, repo.Repository, prereleaseWU08, payload)
			}
			if failures := planning.ValidatePlanning(repo); len(failures) == 0 {
				t.Fatal("invalid work unit admitted")
			}
		})
	}
}

func TestIncompleteGateDependencyBlocksPlanning(t *testing.T) {
	repo := prereleaseFixture(t)
	ref := "developer/planning/0.3.8/0.3.8a3/WU-06/WU-06.yaml"
	payload := readPlanning(t, repo, ref)
	payload["work_unit"].(object)["lifecycle"].(object)["status"] = "BLOCKED"
	testrepo.Write(t, repo.Repository, ref, payload)
	requirePlanningError(t, planning.ValidatePlanning(repo), "current gate dependency incomplete: WU-06")
}

func TestPrereleaseCanonicalPathCannotBeReboundToDifferentIdentity(t *testing.T) {
	repo := prereleaseFixture(t)
	payload := readPlanning(t, repo, prereleasePlan)
	payload["plan"].(object)["prerelease"] = "0.3.8a4"
	testrepo.Write(t, repo.Repository, prereleasePlan, payload)
	requirePlanningError(t, planning.ValidatePlanning(repo), "prerelease canonical location")
}

func TestDependencyOrderCannotOmitOrRepeatWorkUnits(t *testing.T) {
	repo := prereleaseFixture(t)
	payload := readPlanning(t, repo, prereleasePlan)
	order := payload["execution_model"].(object)["dependency_order"].([]any)
	order[len(order)-1] = []any{"WU-07"}
	testrepo.Write(t, repo.Repository, prereleasePlan, payload)
	requirePlanningError(t, planning.ValidatePlanning(repo), "each registered work unit exactly once")
}

func TestPrereleaseFormalIdentityUsesCanonicalBindingGrammar(t *testing.T) {
	for _, test := range []struct{ field, value string }{{"resolved_plan_id", "PLAN_FROM_BRANCH_NAME"}, {"plan_file_id", "developer/planning/0.3.8/index.yaml"}, {"version", ""}, {"revision", ""}} {
		t.Run(test.field, func(t *testing.T) {
			repo := prereleaseFixture(t)
			payload := readPlanning(t, repo, prereleasePlan)
			payload["plan_identity"].(object)[test.field] = test.value
			testrepo.Write(t, repo.Repository, prereleasePlan, payload)
			if failures := planning.ValidatePlanning(repo); len(failures) == 0 {
				t.Fatal("invalid formal identity admitted")
			}
		})
	}
}

func TestRegisteredPrereleaseRequiresFormalPlanIdentity(t *testing.T) {
	repo := prereleaseFixture(t)
	payload := readPlanning(t, repo, prereleasePlan)
	delete(payload, "plan_identity")
	testrepo.Write(t, repo.Repository, prereleasePlan, payload)
	requirePlanningError(t, planning.ValidatePlanning(repo), "plan_identity")
}

func TestPlanningIdentityGrammarComesFromRegisteredBindingContract(t *testing.T) {
	repo := prereleaseFixture(t)
	ref := "developer/bindings/schemas/policy-plan-bindings.schema.json"
	schema := readPlanning(t, repo, ref)
	schema["$defs"].(object)["binding"].(object)["properties"].(object)["resolved_plan_id"].(object)["pattern"] = "^REGISTERED_ID$"
	testrepo.Write(t, repo.Repository, ref, schema)
	payload := readPlanning(t, repo, prereleasePlan)
	payload["plan_identity"].(object)["resolved_plan_id"] = "REGISTERED_ID"
	testrepo.Write(t, repo.Repository, prereleasePlan, payload)
	if failures := planning.ValidatePlanning(repo); len(failures) != 0 {
		t.Fatal(failures)
	}
}

func TestEmergencyPrereleaseResolvesExactOverlay(t *testing.T) {
	repo := &readOnlyRepository{Repository: testrepo.Open(t.TempDir())}
	ref := "developer/planning/0.3.8a1/emergency-implementation-overlay.yaml"
	testrepo.Write(t, repo.Repository, ref, object{"branch": object{"name": "dev/0.3.8a1"}, "plan_version": "0.3.8a1"})
	result, err := planning.ResolvePlanningEntry(repo, "dev/0.3.8a1")
	if err != nil || result["status"] != "RESOLVED" || result["branch"] != "dev/0.3.8a1" || result["plan_version"] != "0.3.8a1" || result["entry_document"] != ref || result["role"] != "EMERGENCY_RELEASE_OVERLAY" || result["work_unit"] != nil {
		t.Fatalf("%#v %v", result, err)
	}
}
