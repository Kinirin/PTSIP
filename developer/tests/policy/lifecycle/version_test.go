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
		"schema_version": "ptsip-developer-policy/v1", "policy_class": "PTSIP_DEVELOPER_POLICY",
		"policy":    object{"id": "MPD-0010", "version": version, "title": "Fixture", "status": status},
		"rules":     object{"fixture": object{"enabled": true}},
		"relations": object{"supersedes": []any{}, "amends": []any{}, "extends": []any{}, "depends_on": []any{}},
	}
	if status == "APPROVED" {
		payload["transition"] = object{"target_status": "ACTIVE", "state": "PENDING", "requirements": []any{object{"id": "TR-001", "type": "FIXTURE", "state": "OPEN", "next_action": object{"action": "FIXTURE", "execution": "PARALLEL_ALLOWED", "completion_check": "FIXTURE_PASS"}}}}
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
		{"2.0", "ACTIVE", true}, {"2.105", "ACTIVE", true}, {"12.248", "ACTIVE", true}, {"3.7", "SUPERSEDED", true}, {"4.0", "RETIRED", true},
		{"1.0", "DRAFT", false}, {"0.1", "APPROVED", false}, {"1.9", "ACTIVE", false}, {"0.9", "RETIRED", false},
		{"01.2", "DRAFT", false}, {"1.02", "APPROVED", false}, {"1.0-draft", "APPROVED", false}, {"1.0.0", "APPROVED", false}, {"draft-1", "DRAFT", false},
	} {
		t.Run(test.status+"/"+test.version, func(t *testing.T) {
			err := repo.Validate("developer/policy/schemas/management-policy.schema.json", versionPolicy(test.version, test.status))
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
		{"1.12", "APPROVED", "LIFECYCLE_TRANSITION", "ACTIVE", "2.12"},
		{"2.99", "ACTIVE", "COMPATIBLE_NORMATIVE", "ACTIVE", "2.100"},
		{"2.99", "ACTIVE", "INCOMPATIBLE_NORMATIVE", "ACTIVE", "3.0"},
		{"3.7", "ACTIVE", "LIFECYCLE_TRANSITION", "RETIRED", "3.7"},
		{"4.2", "ACTIVE", "NON_NORMATIVE", "ACTIVE", "4.2"},
	} {
		t.Run(test.change+"/"+test.status+"/"+test.version, func(t *testing.T) {
			result, err := lifecycle.ResolvePolicyVersionTransition(test.version, test.status, test.change, test.target)
			if err != nil || result["next_version"] != test.want {
				t.Fatalf("want %s, result=%#v, error=%v", test.want, result, err)
			}
		})
	}
	if _, err := lifecycle.ResolvePolicyVersionTransition("1.4", "APPROVED", "COMPATIBLE_NORMATIVE", "APPROVED"); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_POLICY_VERSION_TRANSITION") {
		t.Fatalf("unapproved change admitted: %v", err)
	}
}
