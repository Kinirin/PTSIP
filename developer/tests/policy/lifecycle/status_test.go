package lifecycle_test

import (
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestCatalogNoLongerOwnsPolicyStatusVocabulary(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	catalog, err := repo.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	if slot, ok := catalog["$defs"].(object)["policy_status"].(object); !ok || len(slot) != 0 || catalog["policy_status_contract_ref"] != lifecycle.PolicyStatusContract {
		t.Fatal("catalog must keep an empty slot and the exact external contract reference")
	}
	contract, err := repo.Read(lifecycle.PolicyStatusContract)
	if err != nil {
		t.Fatal(err)
	}
	if contract["authority"].(object)["primary_owner_family"] != "GOV" {
		t.Fatal("policy lifecycle status belongs to GOV")
	}
}

func TestGOVPolicyStatusContractRejectsOldAndCrossStatusValues(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for status, subs := range map[string][]string{
		"DRAFT":    {"Default", "Reviewed", "Denied", "Pass"},
		"APPROVED": {"Default", "Applying", "Application_Problems", "Unresolved", "Activate"},
		"RETIRED":  {"Default", "Removable", "Waiting"},
	} {
		for _, sub := range subs {
			if err := repo.Validate(lifecycle.PolicyStatusContract, object{"status": status, "substatus": sub}); err != nil {
				t.Fatalf("%s.%s: %v", status, sub, err)
			}
		}
	}
	for _, state := range []object{
		{"status": "ACTIVE", "substatus": "Default"},
		{"status": "SUPERSEDED", "substatus": "Default"},
		{"status": "DRAFT", "substatus": "Activate"},
		{"status": "APPROVED", "substatus": "Removable"},
		{"status": "RETIRED", "substatus": "Applying"},
		{"status": "APPROVED"},
	} {
		if err := repo.Validate(lifecycle.PolicyStatusContract, state); err == nil {
			t.Fatalf("invalid policy state admitted: %v", state)
		}
	}
}

func TestPolicyDraftPassAutomaticallyNormalizesWithoutMutatingInput(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	input := object{"status": "DRAFT", "substatus": "Pass"}
	got, err := lifecycle.NormalizePolicyStatus(repo, input)
	if err != nil {
		t.Fatal(err)
	}
	want := object{"status": "APPROVED", "substatus": "Default"}
	if !reflect.DeepEqual(got, want) || input["status"] != "DRAFT" || input["substatus"] != "Pass" {
		t.Fatalf("normalization result %v, input %v", got, input)
	}
	if again, err := lifecycle.NormalizePolicyStatus(repo, got); err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("normalization must be idempotent: %v %v", again, err)
	}
	for _, sub := range []string{"Default", "Reviewed", "Denied"} {
		state := object{"status": "DRAFT", "substatus": sub}
		if got, err := lifecycle.NormalizePolicyStatus(repo, state); err != nil || !reflect.DeepEqual(got, state) {
			t.Fatalf("unapproved state changed: %v %v", got, err)
		}
	}
}
