package binding_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

type taskRepository struct {
	*testrepo.Repository
	branch                            string
	branchReads, implementationChecks int
}

func (r *taskRepository) CurrentBranch() (string, error) { r.branchReads++; return r.branch, nil }
func (r *taskRepository) ValidateImplementationReference(ref object) (object, error) {
	r.implementationChecks++
	if ref["selector"].(object)["kind"] != "GO_FUNCTION" {
		return nil, fmt.Errorf("selector rejected by owning capability")
	}
	return ref, nil
}

func taskFixture(t *testing.T) (*taskRepository, *binding.Resolver, object, object) {
	t.Helper()
	base, _ := storeFixture(t)
	repo := &taskRepository{Repository: base, branch: "dev/fixture"}
	testrepo.CopyFiles(t, base, "registry/ptsip-registry.yaml")
	testrepo.Write(t, base, "developer/planning/fixture.yaml", object{"fixture": true})
	testrepo.Write(t, base, "developer/tests/fixture.yaml", object{"fixture": true})
	resolver, err := binding.NewResolver(repo)
	if err != nil {
		t.Fatal(err)
	}
	resolver.Bindings = map[string]object{"fixture": {"scope": "fixture", "inherit_to_descendants": true, "default_refs": []any{object{"policy_id": "MPD-INFO-0001", "sections": []any{firstSection}}}, "operations": object{}}}
	resolver.Contract["resolver"].(object)["task_context_ref"] = "developer/policy/fixture-task.yaml"
	context := object{"normative_rule_refs": []any{"PTSIP-AUT-007"}, "implementation_refs": []any{object{"path": "fixture.go", "selector": object{"kind": "GO_FUNCTION", "name": "Run"}}}, "test_refs": []any{"developer/tests/fixture.yaml"}, "constraints": []any{"fixture only"}}
	entry := object{"branch": "dev/fixture", "planning_entry": "developer/planning/fixture.yaml", "task_bindings": object{"fixture": object{"MODIFY": context}}}
	testrepo.Write(t, base, "developer/policy/fixture-task.yaml", object{"coding_agent_entry": entry})
	return repo, resolver, entry, context
}

func TestTaskContextJoinsOnlyExactScopeAndOperation(t *testing.T) {
	repo, resolver, _, _ := taskFixture(t)
	result, err := resolver.Resolve("fixture", "MODIFY")
	if err != nil {
		t.Fatal(err)
	}
	context := result["task_context"].(object)
	branch := context["branch_context"].(object)
	if branch["actual"] != "dev/fixture" || branch["match"] != true || context["planning_entry"] != "developer/planning/fixture.yaml" || repo.implementationChecks != 1 {
		t.Fatalf("incomplete exact context: %#v", context)
	}
	for _, request := range [][2]string{{"fixture/child.go", "MODIFY"}, {"fixture", "READ"}} {
		result, err := resolver.Resolve(request[0], request[1])
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := result["task_context"]; exists {
			t.Fatalf("context leaked to %v", request)
		}
	}
	if repo.branchReads != 1 || repo.implementationChecks != 1 {
		t.Fatal("unrelated operation executed task capabilities")
	}
}

func TestTaskContextValidationChecksRegistrationWithoutRequiringCurrentBranch(t *testing.T) {
	repo, resolver, _, _ := taskFixture(t)
	repo.branch = "different-branch"
	if err := resolver.ValidateBindings(); err != nil {
		t.Fatal(err)
	}
	if repo.branchReads != 0 || repo.implementationChecks != 1 {
		t.Fatal("registration validation enforced runtime branch")
	}
	if _, err := resolver.Resolve("fixture", "MODIFY"); err == nil || !strings.Contains(err.Error(), "branch mismatch") {
		t.Fatalf("runtime branch mismatch admitted: %v", err)
	}
}

func TestTaskContextRejectsUnresolvedOrAmbiguousInputs(t *testing.T) {
	for _, test := range []struct {
		name, errorText string
		change          func(object, object)
	}{
		{"missing planning file", "planning_entry", func(entry, context object) { entry["planning_entry"] = "missing.yaml" }},
		{"escaped planning file", "escapes root", func(entry, context object) { entry["planning_entry"] = "../outside.yaml" }},
		{"empty constraints", "constraints", func(entry, context object) { context["constraints"] = []any{} }},
		{"duplicate constraints", "unique", func(entry, context object) { context["constraints"] = []any{"same", "same"} }},
		{"duplicate rules", "unique", func(entry, context object) { context["normative_rule_refs"] = []any{"PTSIP-AUT-007", "PTSIP-AUT-007"} }},
		{"unknown rule", "unknown normative rule", func(entry, context object) { context["normative_rule_refs"] = []any{"UNKNOWN"} }},
		{"missing test", "test_refs item", func(entry, context object) { context["test_refs"] = []any{"missing_test.go"} }},
		{"duplicate tests", "unique", func(entry, context object) {
			context["test_refs"] = []any{"developer/tests/fixture.yaml", "developer/tests/fixture.yaml"}
		}},
		{"empty implementation refs", "implementation_refs", func(entry, context object) { context["implementation_refs"] = []any{} }},
		{"rejected selector", "owning capability", func(entry, context object) {
			context["implementation_refs"] = []any{object{"path": "fixture.go", "selector": object{"kind": "UNKNOWN"}}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, resolver, entry, context := taskFixture(t)
			test.change(entry, context)
			testrepo.Write(t, repo.Repository, "developer/policy/fixture-task.yaml", object{"coding_agent_entry": entry})
			if _, err := resolver.Resolve("fixture", "MODIFY"); err == nil || !strings.Contains(err.Error(), test.errorText) {
				t.Fatalf("want %q, got %v", test.errorText, err)
			}
		})
	}
}

func TestTaskContextRequiresExactPolicyRegistrationAndClosedOperation(t *testing.T) {
	for _, test := range []struct{ scope, operation, want string }{{"fixture/child", "MODIFY", "no exact policy binding"}, {"fixture", "GUESS", "unsupported operation"}} {
		t.Run(test.scope+"/"+test.operation, func(t *testing.T) {
			repo, resolver, entry, context := taskFixture(t)
			entry["task_bindings"] = object{test.scope: object{test.operation: context}}
			testrepo.Write(t, repo.Repository, "developer/policy/fixture-task.yaml", object{"coding_agent_entry": entry})
			if err := resolver.ValidateBindings(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("want %q, got %v", test.want, err)
			}
		})
	}
}
