package lifecycle_test

import (
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

type object = map[string]any

func versionPolicy(version, status string) object {
	payload := object{
		"schema_version": "ptsip-developer-root-family-policy/v1", "policy_class": "PTSIP_DEVELOPER_POLICY", "responsibility_family": "GOV",
		"authority_subject": object{"id": "FIXTURE_SUBJECT", "statement": "Fixture version contract"},
		"exclusive_kernel":  []any{"FIXTURE_KERNEL"},
		"policy":            object{"id": "MPD-GOV-9999", "version": version, "title": "Fixture", "status": status, "substatus": "Default"},
		"rules":             object{"fixture": object{"enabled": true}},
		"relations":         object{"supersedes": []any{}, "amends": []any{}, "extends": []any{}, "depends_on": []any{}},
	}
	if status == "APPROVED" {
		payload["transition"] = object{"target_status": "APPROVED", "target_substatus": "Applying", "state": "PENDING", "requirements": []any{object{"id": "TR-001", "type": "FIXTURE", "state": "OPEN", "next_action": object{"action": "FIXTURE", "execution": "PARALLEL_ALLOWED", "completion_check": "FIXTURE_PASS"}}}}
	}
	return payload
}

func TestPolicyVersionSchemaPreservesLifecycleAndNumericSyntax(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, test := range []struct {
		version, status string
		valid           bool
	}{
		{"0.0", "DRAFT", true}, {"0.137", "DRAFT", true}, {"1.0", "APPROVED", true}, {"1.137", "APPROVED", true},
		{"2.0", "APPROVED", true}, {"2.105", "APPROVED", true}, {"12.248", "APPROVED", true}, {"3.7", "SUPERSEDED", false}, {"4.0", "RETIRED", true},
		{"1.0", "DRAFT", false}, {"0.1", "APPROVED", false}, {"1.9", "ACTIVE", false}, {"0.9", "RETIRED", true},
		{"01.2", "DRAFT", false}, {"1.02", "APPROVED", false}, {"1.0-draft", "APPROVED", false}, {"1.0.0", "APPROVED", false}, {"draft-1", "DRAFT", false},
	} {
		t.Run(test.status+"/"+test.version, func(t *testing.T) {
			err := repo.Validate("developer/policy/schemas/root-family-policy.schema.json", versionPolicy(test.version, test.status))
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, error=%v", test.valid, err)
			}
		})
	}
}

func TestPolicyVersionSemanticMismatchAndInitialValue(t *testing.T) {
	if got := lifecycle.InitialPolicyVersion(); got != "0.0" {
		t.Fatalf("initial version=%s", got)
	}
	errors := lifecycle.ValidatePolicyVersionSemantics("MPD-REAL-0005", versionPolicy("1.4", "ACTIVE"))
	if len(errors) != 1 || !strings.Contains(errors[0], "POLICY_VERSION_STATUS_MISMATCH") {
		t.Fatalf("mismatch not detected: %v", errors)
	}
}

func TestPolicyVersionTransitionsPreserveApprovedChangeClasses(t *testing.T) {
	for _, test := range []struct{ version, status, change, target, want string }{
		{"0.12", "DRAFT", "DRAFT_NORMATIVE", "DRAFT", "0.13"},
		{"0.12", "DRAFT", "LIFECYCLE_TRANSITION", "APPROVED", "1.12"},
		{"2.99", "APPROVED", "COMPATIBLE_NORMATIVE", "APPROVED", "2.100"},
		{"2.99", "APPROVED", "INCOMPATIBLE_NORMATIVE", "APPROVED", "3.0"},
		{"3.7", "APPROVED", "LIFECYCLE_TRANSITION", "RETIRED", "3.7"},
		{"4.2", "APPROVED", "NON_NORMATIVE", "APPROVED", "4.2"},
	} {
		t.Run(test.change+"/"+test.status+"/"+test.version, func(t *testing.T) {
			result, err := lifecycle.ResolvePolicyVersionTransition(test.version, test.status, test.change, test.target)
			if err != nil || result["next_version"] != test.want {
				t.Fatalf("want %s, result=%#v, error=%v", test.want, result, err)
			}
		})
	}
	if _, err := lifecycle.ResolvePolicyVersionTransition("1.4", "APPROVED", "LIFECYCLE_TRANSITION", "ACTIVE"); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_POLICY_VERSION_TRANSITION") {
		t.Fatalf("unapproved change admitted: %v", err)
	}
}
