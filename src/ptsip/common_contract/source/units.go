package source

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
)

type Unit struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
}

func RuleUnits(doc Document) (map[string]Unit, error) {
	units := map[string]Unit{}
	add := func(parent Document, field, path, kind string) error {
		if raw, exists := parent[field]; exists {
			items, ok := raw.([]any)
			if !ok {
				return fmt.Errorf("SOURCE_RULE_LIST_MUST_BE_ARRAY: %s", path)
			}
			for i, item := range items {
				units[fmt.Sprintf("%s/%d", path, i)] = Unit{kind, item}
			}
		}
		return nil
	}
	rules := Object(doc["Policy_Rules"])
	if raw, exists := doc["Policy_Rules"]; exists && Object(raw) == nil {
		return nil, fmt.Errorf("NATIVE_POLICY_RULES_MUST_BE_OBJECT")
	}
	if raw, exists := rules["contracts"]; exists && Array(raw) == nil {
		return nil, fmt.Errorf("NATIVE_POLICY_CONTRACTS_MUST_BE_ARRAY")
	}
	for i, raw := range Array(rules["contracts"]) {
		contract := Object(raw)
		if contract == nil {
			return nil, fmt.Errorf("NATIVE_POLICY_CONTRACT_MUST_BE_OBJECT")
		}
		base := fmt.Sprintf("/Policy_Rules/contracts/%d", i)
		if err := add(contract, "rules", base+"/rules", "RULE"); err != nil {
			return nil, err
		}
		if err := add(contract, "requirements", base+"/requirements", "REQUIREMENT"); err != nil {
			return nil, err
		}
	}
	if err := add(rules, "Definition", "/Policy_Rules/Definition", "RULE"); err != nil {
		return nil, err
	}
	boundary := Object(rules["Boundary"])
	if raw, exists := rules["Boundary"]; exists && Object(raw) == nil {
		return nil, fmt.Errorf("NATIVE_POLICY_BOUNDARY_MUST_BE_OBJECT")
	}
	if err := add(boundary, "subjects", "/Policy_Rules/Boundary/subjects", "REQUIREMENT"); err != nil {
		return nil, err
	}
	for _, section := range []string{"rules", "authority_semantics"} {
		semantics := Object(doc[section])
		if raw, exists := doc[section]; exists && Object(raw) == nil {
			return nil, fmt.Errorf("NATIVE_SEMANTICS_MUST_BE_OBJECT: %s", section)
		}
		for _, name := range slices.Sorted(maps.Keys(semantics)) {
			value := semantics[name]
			pointer := "/" + section + "/" + resolver.Escape(name)
			if name == "family_definition" {
				definition := Object(value)
				if definition == nil {
					return nil, fmt.Errorf("FAMILY_DEFINITION_MUST_BE_OBJECT: %s", pointer)
				}
				if err := add(definition, "kernel_definitions", pointer+"/kernel_definitions", "RULE"); err != nil {
					return nil, err
				}
			} else {
				units[pointer] = Unit{"SEMANTIC_UNIT", value}
			}
		}
	}
	return units, nil
}
