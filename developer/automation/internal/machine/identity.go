package machine

import "fmt"

func (r *Repository) InspectFamilyID(id string) (Object, error) {
	match := familyPolicyID.FindStringSubmatch(id)
	if match == nil {
		return nil, fmt.Errorf("not an exact Root Family policy identity: %s", id)
	}
	registry, err := r.Read("developer/policy/registries/root-family-entry-registry.json")
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/root-family-entry-registry.schema.json", registry); err != nil {
		return nil, err
	}
	matches := []string{}
	for class, raw := range Map(registry["policy_classes"]) {
		if Map(raw)["id_prefix"] == match[1] {
			matches = append(matches, class)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("unregistered or ambiguous Root identity prefix")
	}
	class := matches[0]
	route := Map(Map(registry["policy_classes"])[class])
	recognized := false
	for _, family := range List(route["recognized_root_families"]) {
		if family == match[2] {
			recognized = true
		}
	}
	if !recognized {
		return nil, fmt.Errorf("Root Family not registered for class")
	}
	return Object{"status": "ROOT_FAMILY_ID", "policy_id": id, "policy_class": class, "responsibility_family": match[2],
		"canonical_path": Text(route["canonical_root"]) + "/" + match[2] + "/" + id + ".yaml",
		"schema_ref":     route["schema_ref"], "semantic_inheritance": "FORBIDDEN"}, nil
}
