package lifecycle

import (
	"reflect"
	"sort"
)

func uniqueCatalogFunctions(values []string) []string {
	result := []string{}
	for _, value := range values {
		if !policyContains(result, value) {
			result = append(result, value)
		}
	}
	return result
}

// Historical change scopes retain their original identities and selectors.
// Only an explicit registered binding selects a Go implementation for validation.
func CatalogImplementationTargets(record Object) ([]any, []string) {
	scope := Map(record["change_scope"])
	declared := append(append([]any{}, List(scope["materialization_targets"])...), List(scope["deferred_application_targets"])...)
	sourceFunctions := map[string][]string{}
	for _, raw := range declared {
		row := Map(raw)
		ref := Text(row["path"])
		sourceFunctions[ref] = uniqueCatalogFunctions(append(sourceFunctions[ref], policyStrings(row["python_functions"])...))
	}
	bindings := map[string][]any{}
	errors := []string{}
	for _, raw := range List(record["go_implementation_bindings"]) {
		binding := Map(raw)
		ref := Text(binding["source_path"])
		expected, registered := sourceFunctions[ref]
		if !registered {
			errors = append(errors, "neutral catalog Go binding source is outside the registered scope: "+ref)
			continue
		}
		if _, duplicate := bindings[ref]; duplicate {
			errors = append(errors, "neutral catalog duplicate Go implementation binding: "+ref)
			continue
		}
		actual := policyStrings(binding["source_functions"])
		sort.Strings(expected)
		sort.Strings(actual)
		if !reflect.DeepEqual(actual, expected) {
			errors = append(errors, "neutral catalog Go binding must cover every declared source selector exactly: "+ref)
		}
		bindings[ref] = List(binding["targets"])
	}
	result := []any{}
	used := map[string]bool{}
	for _, raw := range declared {
		ref := Text(Map(raw)["path"])
		if targets, bound := bindings[ref]; bound {
			if !used[ref] {
				result = append(result, targets...)
				used[ref] = true
			}
		} else {
			result = append(result, raw)
		}
	}
	return result, errors
}
