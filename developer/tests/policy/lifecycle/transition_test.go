package lifecycle_test

import (
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func transitionPolicy(state, requirementState string) object {
	payload := versionPolicy("1.0", "APPROVED")
	payload["policy"].(object)["id"] = "MPD-GOV-9999"
	payload["transition"] = object{"target_status": "APPROVED", "target_substatus": "Activate", "state": state, "requirements": []any{object{
		"id": "TR-001", "type": "STRUCTURAL_COMPATIBILITY", "state": requirementState,
		"refs": []any{"MPD-INFO-0001"}, "next_action": object{"action": "ALIGN_PLAN_IDENTITY_CONTRACT", "execution": "PARALLEL_ALLOWED", "target_refs": []any{"MPD-INFO-0001"}, "completion_check": "PLAN_IDENTITY_CONTRACT_ALIGNED"},
	}}}
	return payload
}

func transitionRequirement(payload object) object {
	return payload["transition"].(object)["requirements"].([]any)[0].(object)
}

func requireTransitionError(t *testing.T, payload object, expected string) {
	t.Helper()
	errors := lifecycle.ValidatePolicyTransitionSemantics("MPD-ASSURE-0002", payload)
	for _, message := range errors {
		if strings.Contains(message, expected) {
			return
		}
	}
	t.Fatalf("expected %q, got %v", expected, errors)
}

func TestApprovedTransitionContractPassesCurrentRootSchema(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	if err := repo.Validate("developer/policy/schemas/root-family-policy.schema.json", transitionPolicy("PENDING", "OPEN")); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionRequirementVocabularyAndBlockingFieldAreClosed(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, field := range []string{"state", "blocking"} {
		t.Run(field, func(t *testing.T) {
			payload := transitionPolicy("PENDING", "OPEN")
			if field == "state" {
				transitionRequirement(payload)[field] = "DEFERRED"
			} else {
				transitionRequirement(payload)[field] = true
			}
			if err := repo.Validate("developer/policy/schemas/root-family-policy.schema.json", payload); err == nil {
				t.Fatal("unregistered requirement field or state admitted")
			}
		})
	}
}

func TestApprovedReadyTransitionRequiresEveryRequirementSatisfied(t *testing.T) {
	requireTransitionError(t, transitionPolicy("READY", "OPEN"), "must be PENDING")
	if errors := lifecycle.ValidatePolicyTransitionSemantics("MPD-ASSURE-0002", transitionPolicy("READY", "SATISFIED")); len(errors) != 0 {
		t.Fatal(errors)
	}
}

func TestTransitionDependenciesResolveExactlyAndRejectCycles(t *testing.T) {
	payload := transitionPolicy("PENDING", "OPEN")
	transitionRequirement(payload)["next_action"].(object)["after"] = []any{"TR-999"}
	requireTransitionError(t, payload, "unknown requirement TR-999")
	payload = transitionPolicy("PENDING", "OPEN")
	transitionRequirement(payload)["next_action"].(object)["after"] = []any{"TR-002"}
	transition := payload["transition"].(object)
	transition["requirements"] = append(transition["requirements"].([]any), object{"id": "TR-002", "type": "VERIFICATION", "state": "OPEN", "next_action": object{"action": "RUN_VERIFICATION", "execution": "SERIAL_REQUIRED", "after": []any{"TR-001"}, "completion_check": "VERIFICATION_PASS"}})
	requireTransitionError(t, payload, "transition dependency cycle")
}

func TestApprovedActivateRequiresCompleteSatisfiedRequirements(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	payload := transitionPolicy("COMPLETE", "SATISFIED")
	payload["policy"].(object)["substatus"], payload["policy"].(object)["version"] = "Activate", "2.0"
	if err := repo.Validate("developer/policy/schemas/root-family-policy.schema.json", payload); err != nil {
		t.Fatal(err)
	}
	if errors := lifecycle.ValidatePolicyTransitionSemantics("MPD-ASSURE-0002", payload); len(errors) != 0 {
		t.Fatal(errors)
	}
	payload["transition"].(object)["state"] = "READY"
	requireTransitionError(t, payload, "requires COMPLETE")
	payload["transition"].(object)["state"] = "COMPLETE"
	transitionRequirement(payload)["state"] = "OPEN"
	requireTransitionError(t, payload, "all requirements SATISFIED")
}

func TestRelatedStatusSchemasIncludeApproved(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, test := range []struct {
		file string
		path []string
	}{
		{"developer/policy/schemas/policy-approval-provenance.schema.json", []string{"properties", "approval", "properties", "target_status", "enum"}},
		{"developer/policy/schemas/source-application-review.schema.json", []string{"properties", "policy_query_list", "items", "properties", "expected_status", "enum"}},
	} {
		t.Run(test.file, func(t *testing.T) {
			value, err := repo.Read(test.file)
			if err != nil {
				t.Fatal(err)
			}
			var current any = value
			for _, key := range test.path {
				current = current.(object)[key]
			}
			for _, status := range current.([]any) {
				if status == "APPROVED" {
					return
				}
			}
			t.Fatal("APPROVED status missing")
		})
	}
}
