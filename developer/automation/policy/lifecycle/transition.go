package lifecycle

import "strings"

func ValidatePolicyTransitionSemantics(id string, payload Object) []string {
	transition := Map(payload["transition"])
	if transition == nil {
		return []string{}
	}
	errors := []string{}
	requirements := List(transition["requirements"])
	states := map[string]string{}
	dependencies := map[string][]string{}
	order := []string{}
	for _, raw := range requirements {
		unit := Map(raw)
		name := Text(unit["id"])
		if name == "" {
			continue
		}
		if _, found := states[name]; found {
			errors = append(errors, id+": transition requirement IDs must be unique")
		}
		states[name] = Text(unit["state"])
		order = append(order, name)
	}
	for _, raw := range requirements {
		unit := Map(raw)
		name := Text(unit["id"])
		after := Strings(Map(unit["next_action"])["after"])
		dependencies[name] = after
		for _, dependency := range after {
			if _, found := states[dependency]; !found {
				errors = append(errors, id+": "+name+" references unknown requirement "+dependency)
			}
			if dependency == name {
				errors = append(errors, id+": "+name+" must not depend on itself")
			}
			if Contains([]string{"IN_PROGRESS", "FAILED", "SATISFIED"}, states[name]) && states[dependency] != "SATISFIED" {
				errors = append(errors, id+": "+name+" cannot progress before dependency is SATISFIED: "+dependency)
			}
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string, []string)
	visit = func(name string, lineage []string) {
		if visited[name] {
			return
		}
		if visiting[name] {
			errors = append(errors, id+": transition dependency cycle: "+strings.Join(append(lineage, name), " -> "))
			return
		}
		visiting[name] = true
		for _, dependency := range dependencies[name] {
			if _, found := states[dependency]; found {
				visit(dependency, append(append([]string{}, lineage...), name))
			}
		}
		delete(visiting, name)
		visited[name] = true
	}
	for _, name := range order {
		visit(name, nil)
	}
	satisfied := len(requirements) > 0
	for _, raw := range requirements {
		if Map(raw)["state"] != "SATISFIED" {
			satisfied = false
		}
	}
	status := Map(payload["policy"])["status"]
	if status == "APPROVED" {
		expected := "PENDING"
		if satisfied {
			expected = "READY"
		}
		if transition["state"] != expected {
			errors = append(errors, id+": APPROVED transition state must be "+expected)
		}
	} else if status == "ACTIVE" {
		if !satisfied {
			errors = append(errors, id+": ACTIVE transition history requires all requirements SATISFIED")
		}
		if transition["state"] != "COMPLETE" {
			errors = append(errors, id+": ACTIVE transition history must be COMPLETE")
		}
	}
	return errors
}
