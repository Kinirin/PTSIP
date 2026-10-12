package lifecycle_test

import (
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestGOVLifecycleStructureResolvesEachDeclaredFieldContract(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	gov, err := repo.Read("developer/policy/GOV/MPD-GOV-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rules := gov["rules"].(object)
	if _, stale := rules["policy_status_lifecycle"]; stale {
		t.Fatal("standalone status section survived lifecycle consolidation")
	}
	fields := rules["policy_lifecycle"].(object)
	if len(fields) != 4 {
		t.Fatal("lifecycle requires id, title, status and version fields")
	}
	for _, field := range []string{"policy_id", "policy_title", "policy_status", "policy_version"} {
		if _, ok := fields[field].(object); !ok {
			t.Fatalf("lifecycle field missing: %s", field)
		}
	}
	for field, ref := range map[string]string{"policy_status": lifecycle.PolicyStatusContract, "policy_version": lifecycle.PolicyVersionContract} {
		if fields[field].(object)["contract_ref"] != ref {
			t.Fatalf("%s has a stale canonical contract", field)
		}
		contract, err := repo.Read(ref)
		if err != nil {
			t.Fatal(err)
		}
		authority := contract["authority"].(object)
		if authority["primary_owner_family"] != "GOV" || authority["policy_section"] != "rules.policy_lifecycle."+field {
			t.Fatalf("%s authority points outside its lifecycle field", field)
		}
	}
}

func TestGOVVersionContractJoinsNewStatusWithoutLegacyPolicyReuse(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, identity := range []object{
		{"status": "DRAFT", "substatus": "Default", "version": "0.0"},
		{"status": "DRAFT", "substatus": "Pass", "version": "0.137"},
		{"status": "APPROVED", "substatus": "Default", "version": "1.0"},
		{"status": "APPROVED", "substatus": "Applying", "version": "1.137"},
		{"status": "APPROVED", "substatus": "Activate", "version": "1.8"},
		{"status": "APPROVED", "substatus": "Activate", "version": "2.0"},
		{"status": "APPROVED", "substatus": "Applying", "version": "12.248"},
		{"status": "RETIRED", "substatus": "Default", "version": "0.7"},
		{"status": "RETIRED", "substatus": "Waiting", "version": "1.4"},
		{"status": "RETIRED", "substatus": "Removable", "version": "4.0"},
	} {
		if err := lifecycle.ValidatePolicyLifecycleVersion(repo, identity); err != nil {
			t.Fatalf("valid identity %v: %v", identity, err)
		}
	}
	for _, identity := range []object{
		{"status": "DRAFT", "substatus": "Default", "version": "1.0"},
		{"status": "APPROVED", "substatus": "Activate", "version": "0.8"},
		{"status": "ACTIVE", "version": "1.9"},
		{"status": "ACTIVE", "version": "2.0"},
		{"status": "ACTIVE", "version": "02.0"},
		{"status": "ACTIVE", "version": "2.01"},
		{"status": "ACTIVE", "version": "2.0.0"},
		{"status": "ACTIVE", "version": 2},
		{"status": "ACTIVE", "substatus": "Default", "version": "2.0"},
		{"status": "SUPERSEDED", "version": "3.1"},
		{"status": "APPROVED", "version": "1.0"},
		{"status": "APPROVED", "substatus": "Activate", "version": "01.2"},
		{"status": "APPROVED", "substatus": "Activate", "version": "1.02"},
		{"status": "APPROVED", "substatus": "Activate", "version": "1.0.0"},
		{"status": "APPROVED", "substatus": "Activate", "version": 1},
	} {
		if err := lifecycle.ValidatePolicyLifecycleVersion(repo, identity); err == nil {
			t.Fatalf("invalid identity admitted: %v", identity)
		}
	}
	contract, err := repo.Read(lifecycle.PolicyVersionContract)
	if err != nil {
		t.Fatal(err)
	}
	boundary := contract["source_boundary"].(object)
	if boundary["legacy_policy_reuse"] != false || boundary["legacy_source_fallback"] != false {
		t.Fatal("new version contract must not restore legacy policy authority")
	}
}
