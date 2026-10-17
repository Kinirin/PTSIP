package lifecycle_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestCurrentGOVLifecycleApplicationPreservesVersionsAndExcludesLegacy(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	audit, err := repo.Read("developer/policy/registries/current-policy-lifecycle-application.json")
	if err != nil {
		t.Fatal(err)
	}
	index, err := repo.Read(lifecycle.PolicyIndex)
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]object{}
	for _, raw := range index["policies"].([]any) {
		entry := raw.(object)
		if !strings.HasPrefix(entry["path"].(string), "developer/policy/legacy/") {
			entries[entry["id"].(string)] = entry
		}
	}
	states := map[string]int{}
	for _, raw := range audit["policies"].([]any) {
		row := raw.(object)
		ref := row["path"].(string)
		if strings.HasPrefix(ref, "developer/policy/legacy/") {
			t.Fatal("excluded legacy policy entered current application", ref)
		}
		payload, err := repo.Read(ref)
		if err != nil {
			t.Fatal(err)
		}
		identity := payload["policy"].(object)
		before, after := row["from"].(object), row["to"].(object)
		if row["version_changed"] != false || before["version"] != after["version"] || identity["version"] != before["version"] {
			t.Fatal("application changed an existing policy version", row)
		}
		entry := entries[row["policy_id"].(string)]
		if entry == nil || identity["status"] != after["status"] || identity["substatus"] != after["substatus"] || entry["status"] != identity["status"] || entry["substatus"] != identity["substatus"] {
			t.Fatal("policy, application evidence and index disagree", row)
		}
		if err := repo.Validate("developer/contracts/lifecycle/Policy_version/Policy_version.schema.json", object{"version": identity["version"]}); err != nil {
			t.Fatalf("current version field %s: %v", ref, err)
		}
		states[identity["status"].(string)+"."+identity["substatus"].(string)]++
		delete(entries, row["policy_id"].(string))
	}
	if len(entries) != 0 || !reflect.DeepEqual(states, map[string]int{"APPROVED.Applying": 24, "APPROVED.Default": 8, "DRAFT.Default": 4}) {
		t.Fatal("application must cover the current policy inventory exactly", entries, states)
	}
}
