package machine

import (
	"encoding/json"
	"testing"

	policybinding "github.com/Kinirin/PTSIP/developer/automation/policy/binding"
)

type taskContextTestRepository struct {
	*Repository
	context Object
}

func (r *taskContextTestRepository) Read(ref string) (Object, error) {
	if ref == "developer/automation/implementation_workflows.yaml" {
		return r.context, nil
	}
	payload, err := r.Repository.Read(ref)
	if err != nil || ref != ResolverContract {
		return payload, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var copied Object
	if err := json.Unmarshal(raw, &copied); err != nil {
		return nil, err
	}
	Map(copied["resolver"])["task_context_ref"] = "developer/automation/implementation_workflows.yaml"
	return copied, nil
}

func TestPolicyTaskContextUsesNativeSelectorVerificationWithoutLiveRegistration(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	branch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	context := Object{"normative_rule_refs": []any{"PTSIP-AUT-007"}, "implementation_refs": []any{Object{"path": "developer/automation/policy/binding/resolver.go", "selector": Object{"kind": "GO_FUNCTION", "name": "NewResolver"}}}, "test_refs": []any{"developer/automation/internal/machine/task_context_native_test.go"}, "constraints": []any{"UNIT_TEST_ONLY_NO_LIVE_REGISTRATION"}}
	entry := Object{"branch": branch, "planning_entry": "developer/automation/implementation_workflows.yaml", "task_bindings": Object{"developer/automation": Object{"MODIFY": context}}}
	adapter := &taskContextTestRepository{Repository: repo, context: Object{"coding_agent_entry": entry}}
	resolver, err := policybinding.NewResolver(adapter)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolver.ValidateBindings(); err != nil {
		t.Fatal(err)
	}
	result, err := resolver.Resolve("developer/automation", "MODIFY")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Map(result["task_context"])
	refs := List(resolved["implementation_refs"])
	if len(refs) != 1 || Map(Map(refs[0])["selector"])["kind"] != "GO_FUNCTION" || Map(Map(refs[0])["resolved_location"]) == nil {
		t.Fatalf("native selector evidence absent: %#v", refs)
	}
	canonical, err := repo.Read(ResolverContract)
	if err != nil || Map(canonical["resolver"])["task_context_ref"] != nil {
		t.Fatalf("unit context changed live registration: %v", err)
	}
	Map(Map(List(context["implementation_refs"])[0])["selector"])["name"] = "NotARegisteredFunction"
	if _, err := resolver.Resolve("developer/automation", "MODIFY"); err == nil {
		t.Fatal("invalid native selector accepted through task context")
	}
}
