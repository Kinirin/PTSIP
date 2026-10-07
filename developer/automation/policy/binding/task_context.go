package binding

import (
	"fmt"
	"os"
)

// TaskContextRepository supplies capabilities owned outside policy binding.
// Joining their exact registered references does not transfer their authority.
type TaskContextRepository interface {
	Repository
	CurrentBranch() (string, error)
	ValidateImplementationReference(Object) (Object, error)
}

func taskStrings(value any, label string) ([]string, error) {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("task context %s must be non-empty", label)
	}
	result := []string{}
	seen := map[string]bool{}
	for _, raw := range values {
		item, ok := raw.(string)
		if !ok || item == "" {
			return nil, fmt.Errorf("task context %s must contain non-empty strings", label)
		}
		if seen[item] {
			return nil, fmt.Errorf("task context %s must be unique", label)
		}
		seen[item] = true
		result = append(result, item)
	}
	return result, nil
}

func (r *Resolver) taskReference(value any, label string) (string, error) {
	ref := Text(value)
	if ref == "" {
		return "", fmt.Errorf("%s must be a non-empty repository reference", label)
	}
	scope, err := r.Repo.Scope(ref)
	if err != nil {
		return "", err
	}
	path, err := r.Repo.Path(scope)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s does not resolve to a repository file: %s", label, ref)
	}
	return scope, nil
}

func (r *Resolver) taskEntry() (Object, error) {
	ref := Map(r.Contract["resolver"])["task_context_ref"]
	if ref == nil {
		return nil, nil
	}
	path, err := r.taskReference(ref, "task_context_ref")
	if err != nil {
		return nil, err
	}
	payload, err := r.Repo.Read(path)
	if err != nil {
		return nil, err
	}
	entry := Map(payload["coding_agent_entry"])
	if entry == nil || Map(entry["task_bindings"]) == nil {
		return nil, fmt.Errorf("coding_agent_entry.task_bindings must be a mapping")
	}
	return entry, nil
}

func (r *Resolver) taskContext(scope, operation string, enforceBranch bool) (Object, error) {
	entry, err := r.taskEntry()
	if err != nil || entry == nil {
		return nil, err
	}
	bindings := Map(entry["task_bindings"])
	rawScope := bindings[scope]
	if rawScope == nil {
		return nil, nil
	}
	operations := Map(rawScope)
	if operations == nil {
		return nil, fmt.Errorf("task context operations must be a mapping: %s", scope)
	}
	rawContext := operations[operation]
	if rawContext == nil {
		return nil, nil
	}
	context := Map(rawContext)
	if context == nil {
		return nil, fmt.Errorf("task context must be a mapping: %s:%s", scope, operation)
	}
	declared := Text(entry["branch"])
	if declared == "" {
		return nil, fmt.Errorf("coding_agent_entry.branch must be non-empty")
	}
	capabilities, ok := r.Repo.(TaskContextRepository)
	if !ok {
		return nil, fmt.Errorf("task context verification capabilities are unavailable")
	}
	var actual, match any
	if enforceBranch {
		branch, err := capabilities.CurrentBranch()
		if err != nil {
			return nil, err
		}
		if branch != declared {
			return nil, fmt.Errorf("task context branch mismatch: declared %q, actual %q", declared, branch)
		}
		actual, match = branch, true
	}
	planning, err := r.taskReference(entry["planning_entry"], "planning_entry")
	if err != nil {
		return nil, err
	}
	ruleRefs, err := taskStrings(context["normative_rule_refs"], "normative_rule_refs")
	if err != nil {
		return nil, err
	}
	rules := []any{}
	for _, id := range ruleRefs {
		rule, err := r.Rule(id)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	implementations, ok := context["implementation_refs"].([]any)
	if !ok || len(implementations) == 0 {
		return nil, fmt.Errorf("task context implementation_refs must be non-empty")
	}
	implementationRefs := []any{}
	for _, raw := range implementations {
		reference := Map(raw)
		if reference == nil {
			return nil, fmt.Errorf("implementation reference must be a mapping")
		}
		validated, err := capabilities.ValidateImplementationReference(reference)
		if err != nil {
			return nil, err
		}
		implementationRefs = append(implementationRefs, validated)
	}
	testRefs, err := taskStrings(context["test_refs"], "test_refs")
	if err != nil {
		return nil, err
	}
	for index, ref := range testRefs {
		testRefs[index], err = r.taskReference(ref, "test_refs item")
		if err != nil {
			return nil, err
		}
	}
	seenTests := map[string]bool{}
	for _, ref := range testRefs {
		if seenTests[ref] {
			return nil, fmt.Errorf("task context test_refs must be unique")
		}
		seenTests[ref] = true
	}
	constraints, err := taskStrings(context["constraints"], "constraints")
	if err != nil {
		return nil, err
	}
	ruleSource, err := r.taskReference(Map(r.Contract["resolver"])["normative_rule_registry_ref"], "normative_rule_registry_ref")
	if err != nil {
		return nil, err
	}
	return Object{"branch": declared, "branch_context": Object{"declared": declared, "actual": actual, "match": match}, "planning_entry": planning, "normative_rule_refs": ruleRefs, "normative_rule_source": ruleSource, "normative_rules": rules, "implementation_refs": implementationRefs, "test_refs": testRefs, "constraints": constraints}, nil
}

func (r *Resolver) validateTaskBindings() error {
	entry, err := r.taskEntry()
	if err != nil || entry == nil {
		return err
	}
	for scope, raw := range Map(entry["task_bindings"]) {
		if _, exists := r.Bindings[scope]; !exists {
			return fmt.Errorf("task context scope %q has no exact policy binding", scope)
		}
		selected := Map(raw)
		if selected == nil {
			return fmt.Errorf("task context operations must be a mapping: %s", scope)
		}
		for operation := range selected {
			admitted := false
			for _, candidate := range operations {
				admitted = admitted || candidate == operation
			}
			if !admitted {
				return fmt.Errorf("task context uses unsupported operation %q", operation)
			}
			context, err := r.taskContext(scope, operation, false)
			if err != nil {
				return err
			}
			if context == nil {
				return fmt.Errorf("task context did not resolve for %s:%s", scope, operation)
			}
		}
	}
	return nil
}
