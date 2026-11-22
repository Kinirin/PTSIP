package sourcecompat

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

func sourceIntegrityIssues(profile Profile, generation ObservedBinding) []Issue {
	var issues []Issue
	add := func(code, message, pointer string) {
		issues = append(issues, issue(generation, code, message, at(pointer)))
	}
	components, artifacts, relations := map[string]int{}, map[string]int{}, map[string]int{}
	for _, row := range profile.Components {
		components[row.ID]++
	}
	for _, row := range profile.Artifacts {
		artifacts[row.ID]++
	}
	for _, row := range profile.Relations {
		relations[row.ID]++
	}
	checkIDs := func(ids map[string]int, label, pointer string) {
		duplicates := []string{}
		for _, id := range slices.Sorted(maps.Keys(ids)) {
			if ids[id] > 1 {
				duplicates = append(duplicates, id)
			}
		}
		if len(duplicates) != 0 {
			add("SOURCE_DUPLICATE_ID", "duplicate "+label+" IDs: "+strings.Join(duplicates, ", "), pointer)
		}
	}
	if semantics, ok := profile.Semantics.(V034Semantics); ok {
		type boundaryRoot struct{ root, pointer string }
		var roots []boundaryRoot
		for _, boundary := range semantics.Boundaries {
			for _, value := range boundary.Roots {
				root := strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
				for _, previous := range roots {
					if root == previous.root || strings.HasPrefix(root, previous.root+"/") || strings.HasPrefix(previous.root, root+"/") {
						add("SOURCE_BOUNDARY_OVERLAP", fmt.Sprintf("historical boundary %s:%q overlaps %s:%q", boundary.Location.Pointer, root, previous.pointer, previous.root), boundary.Location.Pointer)
					}
				}
				roots = append(roots, boundaryRoot{root, boundary.Location.Pointer})
			}
		}
		checkIDs(components, "component", "/components")
		if semantics.DeclarationForm == "COMPONENTS" {
			dependencyPolicyIssues(profile.RawPayload, components, add)
		}
		return issues
	}
	semantics := profile.Semantics.(V036Semantics)
	if semantics.Mode == "hybrid" {
		for _, row := range []struct {
			label    string
			ids      map[string]int
			removals []string
		}{
			{"components", components, semantics.RemoveComponents}, {"associated_artifacts", artifacts, semantics.RemoveArtifacts}, {"relationships", relations, semantics.RemoveRelations},
		} {
			checkIDs(row.ids, row.label+" override", "/responsibility_map/overrides/"+row.label)
			conflicts := []string{}
			for _, id := range row.removals {
				if row.ids[id] != 0 {
					conflicts = append(conflicts, id)
				}
			}
			if len(conflicts) != 0 {
				slices.Sort(conflicts)
				add("SOURCE_OVERRIDE_REMOVE_CONFLICT", row.label+" IDs cannot be both overridden and removed: "+strings.Join(conflicts, ", "), "/responsibility_map/overrides")
			}
		}
		return issues
	}
	if semantics.Mode != "explicit" {
		return issues
	}
	checkIDs(components, "component", "/components")
	checkIDs(artifacts, "associated-artifact", "/associated_artifacts")
	checkIDs(relations, "relationship", "/relationships")
	collisions := []string{}
	for _, id := range slices.Sorted(maps.Keys(components)) {
		if artifacts[id] != 0 {
			collisions = append(collisions, id)
		}
	}
	if len(collisions) != 0 {
		add("SOURCE_ENDPOINT_ID_COLLISION", "component and associated-artifact IDs share one endpoint namespace: "+strings.Join(collisions, ", "), "")
	}
	seen := make(map[[3]string]bool)
	for _, relation := range profile.Relations {
		for _, endpoint := range []struct{ label, id string }{{"from", relation.From}, {"to", relation.To}} {
			if components[endpoint.id] == 0 && artifacts[endpoint.id] == 0 {
				add("SOURCE_REFERENCE_ERROR", fmt.Sprintf("relationship endpoint %q is not declared by the source Profile", endpoint.id), relation.Location.Pointer+"/"+endpoint.label)
			}
		}
		key := [3]string{relation.From, relation.To, relation.Type}
		if seen[key] {
			add("SOURCE_DUPLICATE_RELATION", fmt.Sprintf("duplicate semantic relationship %q --%s--> %q", relation.From, relation.Type, relation.To), relation.Location.Pointer)
		}
		seen[key] = true
	}
	for _, artifact := range profile.Artifacts {
		if components[artifact.Anchor] == 0 {
			add("SOURCE_REFERENCE_ERROR", fmt.Sprintf("associated-artifact anchor %q is not a declared component", artifact.Anchor), artifact.Location.Pointer+"/anchor")
		}
	}
	dependencyPolicyIssues(profile.RawPayload, components, add)
	return issues
}

func dependencyPolicyIssues(doc source.Document, componentIDs map[string]int, add func(code, message, pointer string)) {
	const policyPointer = "/component_dependency_policy"
	policy := source.Object(doc["component_dependency_policy"])
	allow, deny := map[[2]string]bool{}, map[[2]string]bool{}
	for _, rule := range []struct {
		kind string
		seen map[[2]string]bool
	}{{"allow", allow}, {"deny", deny}} {
		for i, entry := range source.Array(policy[rule.kind]) {
			relation := source.Object(entry)
			pointer := fmt.Sprintf("%s/%s/%d", policyPointer, rule.kind, i)
			from, to := source.String(relation["from"]), source.String(relation["to"])
			for _, endpoint := range []struct{ label, id string }{{"from", from}, {"to", to}} {
				if componentIDs[endpoint.id] == 0 {
					add("SOURCE_REFERENCE_ERROR", fmt.Sprintf("component dependency policy references undeclared component %q", endpoint.id), pointer+"/"+endpoint.label)
				}
			}
			pair := [2]string{from, to}
			if rule.seen[pair] {
				add("SOURCE_DUPLICATE_RELATION", fmt.Sprintf("duplicate component dependency relation %q -> %q", from, to), pointer)
			}
			rule.seen[pair] = true
		}
	}
	var conflicts [][2]string
	for pair := range allow {
		if deny[pair] {
			conflicts = append(conflicts, pair)
		}
	}
	slices.SortFunc(conflicts, func(a, b [2]string) int { return slices.Compare(a[:], b[:]) })
	for _, pair := range conflicts {
		add("SOURCE_POLICY_CONFLICT", fmt.Sprintf("component dependency relation %q -> %q appears in both allow and deny", pair[0], pair[1]), policyPointer)
	}
}
