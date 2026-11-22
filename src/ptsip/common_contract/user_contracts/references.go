package usercontracts

import "fmt"

type RAUsage struct {
	RAID  string  `json:"RA_id"`
	Paths [][]any `json:"paths"`
}

func object(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func array(value any) []any {
	result, _ := value.([]any)
	return result
}

func indexRecords(records []any, key string, path []any) (map[string]map[string]any, []Issue) {
	indexed := make(map[string]map[string]any)
	duplicates := make(map[string]bool)
	issues := []Issue{}
	for position, value := range records {
		record := object(value)
		identifier, exists := record[key].(string)
		if !exists {
			continue
		}
		if _, found := indexed[identifier]; found {
			duplicates[identifier] = true
			location := append(append([]any{}, path...), position, key)
			issues = append(issues, Issue{Code: "DUPLICATE_ID", Path: location, Message: fmt.Sprintf("Duplicate ID %q", identifier)})
		} else {
			indexed[identifier] = record
		}
	}
	for identifier := range duplicates {
		delete(indexed, identifier)
	}
	return indexed, issues
}

func listRecords(kind string, document map[string]any) []any {
	if kind == "RA" {
		return array(object(document["RA_list"])["entries"])
	}
	return array(document[kind+"_list"])
}

func conditionReferences(records []any, kind string, targets map[string]map[string]any) []Issue {
	issues := []Issue{}
	idsField, namesField := kind+"_ids", kind+"_name"
	for position, value := range records {
		record := object(value)
		links := array(record[idsField])
		names := array(record[namesField])
		_, namesPresent := record[namesField]
		pairingValid := len(names) == len(links)
		seen := make(map[string]bool)
		for linkIndex, link := range links {
			for name, rawID := range object(link) {
				identifier := rawID.(string)
				path := []any{"RA_list", "entries", position, idsField, linkIndex}
				if namesPresent && (linkIndex >= len(names) || names[linkIndex] != name) {
					pairingValid = false
				}
				if seen[identifier] {
					issues = append(issues, Issue{Code: "DUPLICATE_ID", Path: path, Message: fmt.Sprintf("Duplicate %s reference %q", kind, identifier)})
				}
				seen[identifier] = true
				target, found := targets[identifier]
				if !found {
					issues = append(issues, Issue{Code: "UNKNOWN_ID", Path: path, Message: fmt.Sprintf("No unique %s target for %q", kind, identifier)})
				} else if target[namesField] != name {
					issues = append(issues, Issue{Code: "NAME_MISMATCH", Path: path, Message: fmt.Sprintf("Display name differs from the owning %s record for %q", kind, identifier)})
				}
			}
		}
		if namesPresent && !pairingValid {
			issues = append(issues, Issue{
				Code: "NAME_PAIRING", Path: []any{"RA_list", "entries", position, namesField},
				Message: fmt.Sprintf("%s display names must match the linked ID names in order", kind),
			})
		}
	}
	return issues
}

func (v *Validator) rbacIndex(ra, rac, aec map[string]any) (map[string]map[string]any, []Issue) {
	issues := []Issue{}
	documents := []struct {
		kind     string
		document map[string]any
	}{{"RA", ra}, {"RAC", rac}, {"AEC", aec}}
	for _, item := range documents {
		if item.document != nil {
			issues = append(issues, v.StructureIssues(item.kind+"_list", item.document)...)
		}
	}
	if len(issues) != 0 {
		return nil, issues
	}
	records := listRecords("RA", ra)
	indexed, duplicateIssues := indexRecords(records, "RA_id", []any{"RA_list", "entries"})
	issues = append(issues, duplicateIssues...)
	for _, item := range documents[1:] {
		targets, duplicates := indexRecords(listRecords(item.kind, item.document), item.kind+"_id", []any{item.kind + "_list"})
		issues = append(issues, duplicates...)
		issues = append(issues, conditionReferences(records, item.kind, targets)...)
	}
	return indexed, issues
}

func policyReferences(policy map[string]any, ra map[string]map[string]any, contentLinks bool) ([]Issue, []RAUsage) {
	issues := []Issue{}
	usage := []RAUsage{}
	usageIndex := make(map[string]int)
	content := make(map[string]map[string]any)
	if contentLinks {
		var duplicates []Issue
		content, duplicates = indexRecords(array(object(policy["Policy_Content"])["items"]), "id", []any{"Policy_Content", "items"})
		issues = append(issues, duplicates...)
	}
	for position, entry := range array(object(policy["Policy_Rules"])["rules"]) {
		rule := object(entry)
		if identifier, exists := rule["RA_id"].(string); exists && ra != nil {
			path := []any{"Policy_Rules", "rules", position, "RA_id"}
			if _, found := ra[identifier]; !found {
				issues = append(issues, Issue{Code: "UNKNOWN_ID", Path: path, Message: fmt.Sprintf("No unique RA target for %q", identifier)})
			} else {
				index, seen := usageIndex[identifier]
				if !seen {
					index = len(usage)
					usageIndex[identifier] = index
					usage = append(usage, RAUsage{RAID: identifier, Paths: [][]any{}})
				}
				usage[index].Paths = append(usage[index].Paths, path)
			}
		}
		if contentLinks {
			for index, value := range array(object(rule["content"])["content_id"]) {
				identifier := value.(string)
				if _, found := content[identifier]; !found {
					issues = append(issues, Issue{
						Code: "UNKNOWN_ID", Path: []any{"Policy_Rules", "rules", position, "content", "content_id", index},
						Message: fmt.Sprintf("No unique Content ID %q in this policy document", identifier),
					})
				}
			}
		}
	}
	return issues, usage
}
