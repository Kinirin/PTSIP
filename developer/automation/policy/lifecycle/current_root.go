package lifecycle

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var supportCurrentRootID = regexp.MustCompile(strings.Replace(rootID.String(), "MPD-", "SFP-", 1))

type CurrentRootRepository interface {
	Repository
	ValidateDefinition(string, string, any) error
}

// Current Root authority is admitted by its own class-local catalog and schema.
// Historical migration sources do not constrain current status or rule content.
func ValidateCurrentRootContracts(r CurrentRootRepository, class string) (Object, error) {
	registry, err := r.Read("developer/policy/registries/root-family-entry-registry.json")
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/root-family-entry-registry.schema.json", registry); err != nil {
		return nil, err
	}
	route := Map(Map(registry["policy_classes"])[class])
	if route == nil {
		return nil, fmt.Errorf("ROOT_CURRENT_CLASS_UNREGISTERED: %s", class)
	}
	root := Text(route["canonical_root"])
	index, err := r.Read(root + "/index.yaml")
	if err != nil {
		return nil, err
	}
	indexSchema := "developer/policy/schemas/developer-policy-catalog.schema.json"
	if class == "PTSIP_SUPPORT_FEATURE" {
		indexSchema = root + "/schemas/ptsip-support-feature-policy-index.schema.json"
	}
	if err := r.Validate(indexSchema, index); err != nil {
		return nil, err
	}
	seen, families := map[string]bool{}, map[string]bool{}
	ids := []string{}
	var semanticRegistry Object
	if class == "PTSIP_SUPPORT_FEATURE" {
		semanticRegistry, err = r.Read(root + "/registries/ptsip-support-authority-schema-registry.yaml")
		if err != nil {
			return nil, err
		}
	}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		if seen[id] {
			return nil, fmt.Errorf("ROOT_CURRENT_DUPLICATE_ID: %s", id)
		}
		seen[id] = true
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			continue
		}
		if class == DeveloperClass && entry["policy_class"] != class {
			continue
		}
		match := rootID.FindStringSubmatch(id)
		if class == "PTSIP_SUPPORT_FEATURE" {
			match = supportCurrentRootID.FindStringSubmatch(id)
		}
		if match == nil || (entry["authority_role"] != nil && entry["authority_role"] != "CANONICAL_AUTHORITY") {
			return nil, fmt.Errorf("ROOT_CURRENT_AUTHORITY_UNREGISTERED: %s", id)
		}
		family := match[1]
		path := root + "/" + family + "/" + id + ".yaml"
		indexed := Text(entry["path"])
		if class == "PTSIP_SUPPORT_FEATURE" {
			indexed = root + "/" + indexed
		}
		if indexed != path {
			return nil, fmt.Errorf("ROOT_CURRENT_PATH_MISMATCH: %s", id)
		}
		owner, err := r.Read(path)
		if err != nil {
			return nil, err
		}
		policy := Map(owner["policy"])
		if owner["policy_class"] != class || owner["responsibility_family"] != family || policy["id"] != id || policy["status"] != entry["status"] {
			return nil, fmt.Errorf("ROOT_CURRENT_METADATA_MISMATCH: %s", id)
		}
		if err := r.Validate(Text(route["schema_ref"]), owner); err != nil {
			return nil, fmt.Errorf("ROOT_CURRENT_SCHEMA_INVALID: %s: %w", id, err)
		}
		field := "rules"
		if class == "PTSIP_SUPPORT_FEATURE" {
			field = "authority_semantics"
		}
		definition := Map(Map(owner[field])["family_definition"])
		kernels := []any{}
		for _, raw := range List(definition["kernel_definitions"]) {
			kernels = append(kernels, Map(raw)["id"])
		}
		if definition != nil && !reflect.DeepEqual(kernels, List(owner["exclusive_kernel"])) {
			return nil, fmt.Errorf("ROOT_CURRENT_KERNEL_MISMATCH: %s", id)
		}
		if class == DeveloperClass {
			failures := append(ValidatePolicyVersionSemantics(id, owner), ValidatePolicyTransitionSemantics(id, owner)...)
			if len(failures) > 0 {
				return nil, fmt.Errorf("ROOT_CURRENT_LIFECYCLE_INVALID: %s", strings.Join(failures, "; "))
			}
		} else {
			matches := []Object{}
			for _, raw := range List(semanticRegistry["entries"]) {
				row := Map(raw)
				if row["policy_id"] == id {
					matches = append(matches, row)
				}
			}
			if len(matches) != 1 {
				return nil, fmt.Errorf("ROOT_CURRENT_SEMANTIC_BINDING_UNRESOLVED: %s", id)
			}
			contract := Map(owner["authority_contract"])
			for _, key := range []string{"authority_type", "schema_id", "schema_version"} {
				if contract[key] != matches[0][key] {
					return nil, fmt.Errorf("ROOT_CURRENT_SEMANTIC_BINDING_MISMATCH: %s", id)
				}
			}
			if err := r.ValidateDefinition(root+"/"+Text(semanticRegistry["semantic_schema_document"]), Text(matches[0]["schema_definition"]), owner[field]); err != nil {
				return nil, fmt.Errorf("ROOT_CURRENT_SEMANTICS_INVALID: %s: %w", id, err)
			}
		}
		families[family] = true
		ids = append(ids, id)
	}
	for _, raw := range List(route["recognized_root_families"]) {
		if !families[Text(raw)] {
			return nil, fmt.Errorf("ROOT_CURRENT_FAMILY_UNRESOLVED: %s", Text(raw))
		}
	}
	sort.Strings(ids)
	return Object{"status": "PASS", "policy_class": class, "policy_ids": ids, "current_policy_count": len(ids), "historical_reconstruction_required": false}, nil
}
