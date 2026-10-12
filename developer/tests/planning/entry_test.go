package planning_test

import (
	"os"
	"reflect"
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
	for _, check := range []struct{ branch, path, version string }{
		{"main", "developer/planning/0.3.8/index.yaml", "0.3.8"},
		{"dev/0.3.8a3", prereleasePlan, "0.3.8a3"},
	} {
		result, err := planning.ResolvePlanningEntry(repo, check.branch)
		if err != nil || result["entry_document"] != check.path || result["plan_version"] != check.version || result["role"] != "INTEGRATION_CONTROL_PLANE" {
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
	if _, err := planning.ResolvePlanningEntry(repo, "dev/0.3.8a3"); err == nil || !strings.Contains(err.Error(), "ambiguous planning entry") {
		t.Fatal(err)
	}
}

func TestPlanningEntryCLIProvidesExactDocumentAndRejectsMissingOrUnknownEntries(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	repo := prereleaseFixture(t)
	testrepo.CopyTree(t, repo.Repository, "developer/policy")
	testrepo.CopyFiles(t, repo.Repository, "pyproject.toml")
	result, err, output := testrepo.CLI(t, binary, repo.Root, "planning-entry", "resolve", "--branch", "dev/0.3.8a3")
	if err != nil || result["entry_document"] != prereleasePlan || result["branch"] != "dev/0.3.8a3" {
		t.Fatalf("machine entry path must be exact: %v\n%s", err, output)
	}
	if _, err, _ := testrepo.CLI(t, binary, repo.Root, "planning-entry", "resolve", "--branch", "dev/0.3.8a30"); err == nil {
		t.Fatal("similar unknown branch accepted by native CLI")
	}
	path, err := repo.Path(prereleasePlan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err, output := testrepo.CLI(t, binary, repo.Root, "planning-entry", "resolve", "--branch", "dev/0.3.8a3"); err == nil || !strings.Contains(output, prereleasePlan) {
		t.Fatalf("missing registered entry document must fail closed: %v\n%s", err, output)
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
	// This test changes the formal identity contract for the prerelease only.
	// Keep the independent 0.3.8 release plan out of this isolated fixture.
	rootRef := "developer/planning/index.yaml"
	root := readPlanning(t, repo, rootRef)
	root["plans"] = root["plans"].([]any)[:1]
	testrepo.Write(t, repo.Repository, rootRef, root)
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

func TestApprovedReleaseMaintainsIndependentCurrentAuthority(t *testing.T) {
	repo := prereleaseFixture(t)
	root := readPlanning(t, repo, "developer/planning/0.3.8/index.yaml")
	units := root["work_units"].([]any)
	var unit object
	for _, raw := range units {
		candidate := raw.(object)
		if candidate["id"] == "WU-01" {
			if unit != nil {
				t.Fatal("duplicate current candidate work unit")
			}
			unit = candidate
		}
	}
	if unit == nil {
		t.Fatal("approved candidate work unit missing")
	}
	if root["plan"].(object)["status"] != "ACTIVE" || root["plan"].(object)["current_gate"] != "WU-03" ||
		unit["id"] != "WU-01" ||
		unit["approval"].(object)["status"] != "APPROVED" ||
		unit["approval"].(object)["approval_source"] != "USER_EXPLICIT" ||
		unit["lifecycle"].(object)["status"] != "COMPLETE" ||
		unit["implementation_authorization"].(object)["status"] != "COMPLETE" ||
		unit["implementation_authorization"].(object)["authorization_source"] != "USER_EXPLICIT" {
		t.Fatal("verified foundation completion must preserve explicit approval and execution authority sources")
	}
	parent := readPlanning(t, repo, "developer/planning/0.3.8/WU-01/WU-01.yaml")
	if parent["work_unit"].(object)["schema_version"] != nil {
		t.Fatal("schema_version belongs to the work unit document, not work_unit metadata")
	}
	if parent["schema_version"] != "ptsip-work-unit/v1" {
		t.Fatal("current release work unit schema version mismatch")
	}
	if !reflect.DeepEqual(parent["completion_evidence"], unit["completion_evidence"]) || len(unit["completion_evidence"].([]any)) == 0 {
		t.Fatal("completed work unit and index must carry matching verification evidence")
	}
	extensions := parent["extensions"].([]any)
	if len(extensions) != 1 {
		t.Fatalf("expected one approved current release extension, got %d", len(extensions))
	}
	row := extensions[0].(object)
	ref := "developer/planning/0.3.8/WU-01/WU-01-P01.yaml"
	if row["id"] != "WU-01-P01" || row["path"] != ref || row["status"] != "COMPLETE" {
		t.Fatal("verified extension must preserve its exact registration at completion")
	}
	review := readPlanning(t, repo, ref)
	if review["approval"].(object)["status"] != "APPROVED" ||
		review["approval"].(object)["approval_source"] != "USER_EXPLICIT" ||
		review["implementation_authorization"].(object)["status"] != "COMPLETE" ||
		review["implementation_authorization"].(object)["authorization_source"] != "USER_EXPLICIT" ||
		review["extension"].(object)["lifecycle"].(object)["status"] != "COMPLETE" ||
		review["objective"].(object)["runtime_activation"] != "FORBIDDEN" {
		t.Fatal("verification completion cannot authorize runtime activation")
	}
	closeout := review["foundation_closeout"].(object)
	if closeout["approval_source"] != "USER_EXPLICIT" || closeout["runtime_activation"] != false || closeout["policy_status_mutation"] != false {
		t.Fatal("closeout changed policy or runtime authority")
	}
}

func TestUnapprovedReleaseWorkUnitRegistryPlaceholderIsRejected(t *testing.T) {
	repo := prereleaseFixture(t)
	planRef := "developer/planning/0.3.8/index.yaml"
	plan := readPlanning(t, repo, planRef)
	plan["work_units"] = []any{object{
		"id":                           "WU-01",
		"lifecycle":                    object{"status": "NOT_CREATED"},
		"approval":                     object{"status": "PENDING", "inherited_from": []any{}},
		"implementation_authorization": object{"status": "NOT_AUTHORIZED"},
		"depends_on":                   []any{},
	}}
	testrepo.Write(t, repo.Repository, planRef, plan)
	if failures := planning.ValidatePlanning(repo); len(failures) == 0 {
		t.Fatal("unapproved work unit registry placeholder must not be accepted")
	}
}

func TestReleaseSingleRouteRejectsMismatchedEntry(t *testing.T) {
	repo := prereleaseFixture(t)
	root := readPlanning(t, repo, "developer/planning/index.yaml")
	plans := root["plans"].([]any)
	entry := plans[len(plans)-1].(object)
	route := entry["entry_routing"].(object)["branch_entrypoints"].([]any)[0].(object)
	route["entry_document"] = prereleasePlan
	testrepo.Write(t, repo.Repository, "developer/planning/index.yaml", root)
	if len(planning.ValidatePlanning(repo)) == 0 {
		t.Fatal("mismatched release route must be rejected")
	}
}
