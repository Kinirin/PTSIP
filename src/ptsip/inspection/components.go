package inspection

import (
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
)

type DecisionStatus string

const (
	Resolved   DecisionStatus = "RESOLVED"
	Unknown    DecisionStatus = "UNKNOWN"
	Conflict   DecisionStatus = "CONFLICT"
	Incomplete DecisionStatus = "INCOMPLETE"
)

type ComponentCandidate struct {
	ID             string         `json:"id"`
	Include        []string       `json:"include"`
	Anchors        []string       `json:"anchors"`
	EvidenceIDs    []string       `json:"evidence_ids"`
	DecisionStatus DecisionStatus `json:"decision_status"`
	Classification *string        `json:"classification"`
}

type DependencyAnchor struct {
	Adapter      string
	ResolvedPath string
	EvidenceID   string
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// DiscoverComponentCandidates proposes groups of observed file selectors without
// deciding the components' classifications.
func DiscoverComponentCandidates(inventory Inventory, dependencies []DependencyAnchor) []ComponentCandidate {
	type observations struct{ anchors, evidence map[string]bool }
	groups := make(map[string]observations)
	add := func(selector, anchor, id string) {
		if _, exists := groups[selector]; !exists {
			groups[selector] = observations{make(map[string]bool), make(map[string]bool)}
		}
		groups[selector].anchors[anchor] = true
		groups[selector].evidence[id] = true
	}
	for _, manifest := range inventory.Manifests {
		selector := manifest
		if parent := path.Dir(manifest); parent != "." {
			selector = parent + "/**"
		}
		add(selector, "manifest", "manifest:"+manifest)
	}
	schemaParents := make(map[string]int)
	for _, schema := range inventory.SchemaCandidates {
		schemaParents[path.Dir(schema)]++
	}
	for parent, count := range schemaParents {
		selector := "*.schema.*"
		if parent != "." {
			selector = parent + "/**"
		}
		add(selector, "contract/schema-group", fmt.Sprintf("schema-group:%s:%d", parent, count))
	}
	for _, root := range inventory.ToolLikeRoots {
		add(root+"/**", "top-level-tool-root", "root:"+root)
	}
	for _, root := range inventory.TestRoots {
		add(root+"/**", "top-level-test-root", "root:"+root)
	}
	for _, edge := range dependencies {
		if edge.Adapter == "github-actions" && edge.ResolvedPath != "" {
			add(edge.ResolvedPath, "ci-invoked-script", edge.EvidenceID)
		}
	}
	result := make([]ComponentCandidate, 0, len(groups))
	for _, selector := range slices.Sorted(maps.Keys(groups)) {
		group := groups[selector]
		id := strings.ToLower(strings.Trim(nonAlphanumeric.ReplaceAllString(strings.TrimSuffix(selector, "/**"), "-"), "-"))
		if id == "" {
			id = "repository-root"
		}
		result = append(result, ComponentCandidate{ID: id, Include: []string{selector}, Anchors: slices.Sorted(maps.Keys(group.anchors)),
			EvidenceIDs: slices.Sorted(maps.Keys(group.evidence)), DecisionStatus: Unknown})
	}
	return result
}
