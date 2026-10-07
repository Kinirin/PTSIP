package planning_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/planning"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"go.yaml.in/yaml/v3"
)

type object = map[string]any

func readyExtension() object {
	return object{
		"schema_version": "ptsip-plan-extension/v1", "plan_version": "0.4.0",
		"extension":                    object{"id": "WU-02-P01", "parent": "WU-02", "lifecycle": object{"status": "ACTIVE"}},
		"implementation_authorization": object{"status": "AUTHORIZED", "authorization_source": "USER_EXPLICIT"},
		"current_known_blockers":       []any{},
		"migration_stages":             object{"P01_A": object{"status": "COMPLETE"}, "P01_B": object{"status": "COMPLETE"}},
		"p01_f_execution_plan":         object{"execution_order": []any{object{"id": "P01_F_FINAL", "status": "COMPLETE", "validation": object{"status": "PASS"}}}},
	}
}

func TestExtensionReadinessRequiresEveryMachineCompletionCondition(t *testing.T) {
	if !planning.ExtensionMachineReady(readyExtension()) {
		t.Fatal("ready extension rejected")
	}
	for _, test := range []struct {
		name   string
		change func(object)
	}{
		{"blocker", func(value object) { value["current_known_blockers"] = []any{"WAITING_FOR_VALIDATION"} }},
		{"pending validation", func(value object) {
			value["p01_f_execution_plan"].(object)["execution_order"].([]any)[0].(object)["validation"].(object)["status"] = "PENDING"
		}},
		{"incomplete migration stage", func(value object) { value["migration_stages"].(object)["P01_B"].(object)["status"] = "ACTIVE" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := readyExtension()
			test.change(value)
			if planning.ExtensionMachineReady(value) {
				t.Fatal("incomplete extension admitted")
			}
		})
	}
}

func TestExtensionTextUpdatesPreserveAuthorizationAndSiblingStructure(t *testing.T) {
	extension := "schema_version: ptsip-plan-extension/v1\nextension:\n  id: WU-02-P01\n  parent: WU-02\n  lifecycle:\n    status: ACTIVE\napproval:\n  status: APPROVED\n  approval_source: USER_EXPLICIT\nimplementation_authorization:\n  status: AUTHORIZED\n  authorization_source: USER_EXPLICIT\ndepends_on:\n  - WU-02\n"
	parent := "extensions:\n  - id: WU-02-P01\n    path: developer/planning/0.4.0/WU-02/WU-02-P01.yaml\n    status: ACTIVE\nscope_contract:\n  planning_contract: ptsip-planning/v1\n"
	got, err := planning.ReplaceExtensionStatuses(extension, readyExtension())
	if err != nil {
		t.Fatal(err)
	}
	parent, err = planning.ReplaceStageStatus(parent, "WU-02-P01", "ACTIVE", "COMPLETE")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"lifecycle:\n    status: COMPLETE", "implementation_authorization:\n  status: COMPLETE", "  authorization_source: USER_EXPLICIT", "approval:\n  status: APPROVED", "depends_on:\n  - WU-02"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing preserved text %q:\n%s", expected, got)
		}
	}
	if !strings.Contains(parent, "- id: WU-02-P01\n    path:") || !strings.Contains(parent, "    status: COMPLETE\nscope_contract:") {
		t.Fatal(parent)
	}
}

func TestExtensionParentLifecycleMismatchIsExplicit(t *testing.T) {
	parent := object{"work_unit": object{"id": "WU-02"}, "extensions": []any{object{"id": "WU-02-P01", "path": "developer/planning/0.4.0/WU-02/WU-02-P01.yaml", "status": "COMPLETE"}}}
	failures := planning.ExtensionParentConsistency(parent, readyExtension(), "WU-02-P01", "developer/planning/0.4.0/WU-02/WU-02-P01.yaml")
	if len(failures) != 1 || !strings.Contains(failures[0], "parent extension status mismatch") {
		t.Fatalf("mismatch not reported: %v", failures)
	}
}

func TestStagePromotionSynchronizesDocumentAndFollowingStage(t *testing.T) {
	text := "migration_stages:\n  P01_E:\n    status: IMPLEMENTED_VALIDATION_PENDING\n    migration_only_artifacts: RETIRED_PENDING_MACHINE_VALIDATION\ncurrent_known_blockers:\n  - STAGE_A_MACHINE_VALIDATION_PENDING\n  - OTHER_BLOCKER\nexecution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n      validation:\n        status: PENDING\n    - id: STAGE_B\n      status: BLOCKED_BY_STAGE_A_VALIDATION\n"
	automatic := object{
		"next_stage": object{"id": "STAGE_B", "from_status": "BLOCKED_BY_STAGE_A_VALIDATION", "to_status": "READY"},
		"document_updates": object{
			"mapping_scalars": []any{object{"section": "migration_stages", "key": "P01_E", "field": "status", "from_value": "IMPLEMENTED_VALIDATION_PENDING", "to_value": "COMPLETE_READY_FOR_NEXT_GATE"}, object{"section": "migration_stages", "key": "P01_E", "field": "migration_only_artifacts", "from_value": "RETIRED_PENDING_MACHINE_VALIDATION", "to_value": "RETIRED"}},
			"list_removals":   []any{object{"section": "current_known_blockers", "value": "STAGE_A_MACHINE_VALIDATION_PENDING"}},
		},
	}
	got, err := planning.PlanningPromoteStageText(text, "STAGE_A", automatic)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"P01_E:\n    status: COMPLETE_READY_FOR_NEXT_GATE", "migration_only_artifacts: RETIRED", "OTHER_BLOCKER", "- id: STAGE_A\n      status: COMPLETE", "validation:\n        status: PASS", "status: PASS\n    - id: STAGE_B", "- id: STAGE_B\n      status: READY"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "STAGE_A_MACHINE_VALIDATION_PENDING") {
		t.Fatal("completed blocker remains")
	}
}

func TestStagePromotionNormalizesEmptyBlockerList(t *testing.T) {
	text := "migration_stages:\n  P01_F:\n    status: IMPLEMENTED_VALIDATION_PENDING\ncurrent_known_blockers:\n  - P01_F_MACHINE_VALIDATION_PENDING\nexecution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n      validation:\n        status: PENDING\n"
	automatic := object{"document_updates": object{"mapping_scalars": []any{object{"section": "migration_stages", "key": "P01_F", "field": "status", "from_value": "IMPLEMENTED_VALIDATION_PENDING", "to_value": "COMPLETE"}}, "list_removals": []any{object{"section": "current_known_blockers", "value": "P01_F_MACHINE_VALIDATION_PENDING"}}}}
	got, err := planning.PlanningPromoteStageText(text, "STAGE_A", automatic)
	if err != nil {
		t.Fatal(err)
	}
	var payload object
	if err := yaml.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatal(err)
	}
	blockers, ok := payload["current_known_blockers"].([]any)
	if !ok || len(blockers) != 0 || !strings.Contains(got, "current_known_blockers: []") {
		t.Fatalf("empty list not preserved:\n%s", got)
	}
}

func TestStagePromotionPreservesBoundariesWithTrailingWhitespace(t *testing.T) {
	text := "execution_order:\n    - id: STAGE_A   \n      status: IMPLEMENTED_VALIDATION_PENDING   \n      validation:   \n        status: PENDING   \n    - id: STAGE_B\n      status: BLOCKED_BY_STAGE_A_VALIDATION\n"
	automatic := object{"next_stage": object{"id": "STAGE_B", "from_status": "BLOCKED_BY_STAGE_A_VALIDATION", "to_status": "READY"}}
	got, err := planning.PlanningPromoteStageText(text, "STAGE_A", automatic)
	if err != nil || !strings.Contains(got, "status: PASS\n    - id: STAGE_B") || !strings.Contains(got, "- id: STAGE_B\n      status: READY") {
		t.Fatalf("boundary damaged: %v\n%s", err, got)
	}
}

func TestStagePromotionRejectsStaleScalarsAndUnreadyStages(t *testing.T) {
	text := "migration_stages:\n  P01_E:\n    status: READY\nexecution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n"
	automatic := object{"document_updates": object{"mapping_scalars": []any{object{"section": "migration_stages", "key": "P01_E", "field": "status", "from_value": "IMPLEMENTED_VALIDATION_PENDING", "to_value": "COMPLETE_READY_FOR_NEXT_GATE"}}, "list_removals": []any{}}}
	if _, err := planning.PlanningPromoteStageText(text, "STAGE_A", automatic); err == nil {
		t.Fatal("stale scalar admitted")
	}
	if _, err := planning.PlanningPromoteStageText("execution_order:\n    - id: STAGE_A\n      status: READY\n", "STAGE_A", object{}); err == nil {
		t.Fatal("unready stage admitted")
	}
}

type readOnlyRepository struct {
	*testrepo.Repository
	reads int
}

func (r *readOnlyRepository) Read(ref string) (object, error) {
	r.reads++
	return r.Repository.Read(ref)
}
func (r *readOnlyRepository) AtomicWrite(string, []byte, *string) error {
	return fmt.Errorf("unexpected write")
}
func (r *readOnlyRepository) DispatchOperation(string, string, map[string]string, []string) (any, error) {
	return nil, fmt.Errorf("unexpected execution")
}

func TestCompletedStageRetriesExtensionInspectionWithoutRepromotion(t *testing.T) {
	repo := &readOnlyRepository{Repository: testrepo.Open(t.TempDir())}
	testrepo.Write(t, repo.Repository, "plan.yaml", object{"execution_order": []any{object{"id": "STAGE_A", "status": "COMPLETE"}}})
	got, err := planning.FinalizePlanningStage(repo, "plan.yaml", "STAGE_A")
	if err != nil || got["promoted"] != false || repo.reads != 2 || len(planning.Strings(got["failures"])) != 0 {
		t.Fatalf("extension retry: reads=%d, result=%#v, error=%v", repo.reads, got, err)
	}
}
