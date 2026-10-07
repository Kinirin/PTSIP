package planning_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/planning"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const mergePlan = "developer/planning/0.4.0/index.yaml"

func mergeFixture(t *testing.T, wu02, extension, wu04 string, evidence bool, leafState string) *readOnlyRepository {
	t.Helper()
	repo := testrepo.Open(t.TempDir())
	files := testrepo.ReadJSON(t, testrepo.Open(testrepo.Root(t)), "developer/tests/planning/testdata/merge_fixture.json")
	for ref, value := range files {
		testrepo.Write(t, repo, ref, value)
	}
	adapter := &readOnlyRepository{Repository: repo}
	index := readPlanning(t, adapter, mergePlan)
	for _, change := range []struct {
		id, status string
		complete   bool
	}{{"WU-02", wu02, wu02 == "COMPLETE"}, {"WU-04", wu04, evidence}} {
		ref := "developer/planning/0.4.0/" + change.id + "/" + change.id + ".yaml"
		payload := readPlanning(t, adapter, ref)
		row := payload["work_unit"].(object)
		row["lifecycle"].(object)["status"] = change.status
		authorization := "AUTHORIZED"
		if change.status == "COMPLETE" {
			authorization = "COMPLETE"
		}
		row["implementation_authorization"].(object)["status"] = authorization
		completion := []any{object{"type": "MACHINE_VALIDATION", "id": change.id + "-focused", "result": "PASS"}}
		if change.complete {
			payload["completion_evidence"] = completion
		} else {
			delete(payload, "completion_evidence")
		}
		testrepo.Write(t, repo, ref, payload)
		for _, raw := range index["work_units"].([]any) {
			indexed := raw.(object)
			if indexed["id"] != change.id {
				continue
			}
			indexed["lifecycle"].(object)["status"] = change.status
			indexed["implementation_authorization"].(object)["status"] = authorization
			if change.complete {
				indexed["completion_evidence"] = completion
			} else {
				delete(indexed, "completion_evidence")
			}
		}
	}
	testrepo.Write(t, repo, mergePlan, index)
	extensionRef := "developer/planning/0.4.0/WU-02/WU-02-P01.yaml"
	payload := readPlanning(t, adapter, extensionRef)
	payload["extension"].(object)["lifecycle"].(object)["status"] = extension
	testrepo.Write(t, repo, extensionRef, payload)
	root := readPlanning(t, adapter, "developer/planning/index.yaml")
	for _, raw := range root["plans"].([]any)[0].(object)["entry_routing"].(object)["branch_entrypoints"].([]any) {
		row := raw.(object)
		if row["work_unit"] == "WU-04" {
			row["state"] = leafState
			if leafState == "MERGED" {
				row["merged_into"] = "dev/0.4.0"
			}
		}
	}
	testrepo.Write(t, repo, "developer/planning/index.yaml", root)
	return adapter
}

func TestPlanningMergeGatePriorityPreservesIndependentWork(t *testing.T) {
	for _, test := range []struct {
		name, wu02, extension, wu04, leafState, merged, gate string
		evidence                                             bool
	}{
		{"completed_wu04_does_not_preempt_wu02", "ACTIVE", "ACTIVE", "COMPLETE", "ACTIVE", "dev/0.4.0-WU-04", "WU-02-P01", true},
		{"both_complete_select_wu05", "COMPLETE", "COMPLETE", "COMPLETE", "ACTIVE", "dev/0.4.0-WU-04", "WU-05", true},
		{"unfinished_merge_resumes_after_wu02", "COMPLETE", "COMPLETE", "ACTIVE", "ACTIVE", "dev/0.4.0-WU-04", "WU-04", false},
		{"active_wu02_preserves_gate", "ACTIVE", "ACTIVE", "ACTIVE", "ACTIVE", "dev/0.4.0-WU-04", "WU-02-P01", false},
		{"later_completion_resumes_merged_wu04", "COMPLETE", "COMPLETE", "ACTIVE", "MERGED", "", "WU-04", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := mergeFixture(t, test.wu02, test.extension, test.wu04, test.evidence, test.leafState)
			path, err := repo.Path("developer/planning/index.yaml")
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := planning.ReconcilePlanning(repo, "dev/0.4.0", test.merged, false)
			if err != nil || result["current_gate_after"] != test.gate {
				t.Fatalf("%#v %v", result, err)
			}
			if test.merged != "" && result["merged_work_unit"] != "WU-04" {
				t.Fatal(result)
			}
			if test.gate == "WU-05" && !strings.HasSuffix(result["current_gate_document"].(string), "/WU-05/WU-05.yaml") {
				t.Fatal(result)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("preview mutated shared planning index", err)
			}
			if test.leafState == "MERGED" {
				root := readPlanning(t, repo, "developer/planning/index.yaml")
				rootPlan := root["plans"].([]any)[0].(object)
				index := readPlanning(t, repo, mergePlan)
				docs, err := planning.Documents(repo, index)
				if err != nil {
					t.Fatal(err)
				}
				state, err := planning.BuildPlanningMaterializedState(repo, rootPlan, index, docs)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, raw := range state["work_units"].([]any) {
					row := raw.(object)
					if row["id"] == "WU-04" {
						found = true
						if row["execution_location"] != "INTEGRATION_BRANCH" || row["branch"] != "dev/0.4.0" {
							t.Fatal(row)
						}
					}
				}
				if !found {
					t.Fatal("missing merged work unit projection")
				}
			}
		})
	}
}

func TestMergedPlanningLeafIsNotAnActiveResolverEntry(t *testing.T) {
	repo := mergeFixture(t, "ACTIVE", "ACTIVE", "ACTIVE", false, "MERGED")
	if _, err := planning.ResolvePlanningEntry(repo, "dev/0.4.0-WU-04"); err == nil || !strings.Contains(err.Error(), "no active exact planning entry") {
		t.Fatal(err)
	}
}

func TestPlanningCompletionRequiresMachineCompletionEvidence(t *testing.T) {
	repo := mergeFixture(t, "ACTIVE", "ACTIVE", "COMPLETE", false, "ACTIVE")
	if _, err := planning.ReconcilePlanning(repo, "dev/0.4.0", "dev/0.4.0-WU-04", false); err == nil || !strings.Contains(err.Error(), "MISSING_COMPLETION_EVIDENCE") {
		t.Fatal(err)
	}
}

func TestPendingBranchRenameAliasReconcilesToCanonicalBranch(t *testing.T) {
	repo := mergeFixture(t, "ACTIVE", "ACTIVE", "ACTIVE", false, "MERGED")
	root := readPlanning(t, repo, "developer/planning/index.yaml")
	plan := root["plans"].([]any)[0].(object)
	plan["integration_branch"] = "dev/0.3.8"
	plan["branch_identity_migration"] = object{"status": "RENAME_PENDING", "canonical_branch": "dev/0.3.8", "legacy_branch": "dev/0.4.0", "scope": "EXECUTION_BRANCH_IDENTITY_ONLY"}
	routing := plan["entry_routing"].(object)
	routing["merge_reconciliation"].(object)["target_branch"] = "dev/0.3.8"
	for _, raw := range routing["branch_entrypoints"].([]any) {
		entry := raw.(object)
		if entry["branch"] == "dev/0.4.0" {
			entry["role"], entry["canonical_branch"] = "BRANCH_RENAME_SOURCE_ALIAS", "dev/0.3.8"
		}
	}
	routing["branch_entrypoints"] = append([]any{object{"branch": "dev/0.3.8", "entry_document": mergePlan, "role": "INTEGRATION_CONTROL_PLANE", "state": "ACTIVE"}}, routing["branch_entrypoints"].([]any)...)
	testrepo.Write(t, repo.Repository, "developer/planning/index.yaml", root)
	canonical, err := planning.ResolvePlanningEntry(repo, "dev/0.3.8")
	if err != nil || canonical["role"] != "INTEGRATION_CONTROL_PLANE" {
		t.Fatalf("%#v %v", canonical, err)
	}
	legacy, err := planning.ResolvePlanningEntry(repo, "dev/0.4.0")
	if err != nil || legacy["role"] != "BRANCH_RENAME_SOURCE_ALIAS" {
		t.Fatalf("%#v %v", legacy, err)
	}
	result, err := planning.ReconcilePlanning(repo, "dev/0.4.0", "", false)
	if err != nil || result["integration_branch"] != "dev/0.3.8" {
		t.Fatalf("%#v %v", result, err)
	}
}

func TestPlanningReconciliationRequiresExplicitApprovalAndAuthorizationSources(t *testing.T) {
	for _, test := range []struct{ section, field, code string }{{"approval", "approval_source", "INVALID_WORK_UNIT_APPROVAL_SOURCE"}, {"implementation_authorization", "authorization_source", "INVALID_WORK_UNIT_AUTHORIZATION_SOURCE"}} {
		t.Run(test.field, func(t *testing.T) {
			repo := mergeFixture(t, "ACTIVE", "ACTIVE", "ACTIVE", false, "ACTIVE")
			ref := "developer/planning/0.4.0/WU-03/WU-03.yaml"
			payload := readPlanning(t, repo, ref)
			delete(payload["work_unit"].(object)[test.section].(object), test.field)
			testrepo.Write(t, repo.Repository, ref, payload)
			if _, err := planning.ReconcilePlanning(repo, "dev/0.4.0", "", false); err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatal(err)
			}
		})
	}
}
