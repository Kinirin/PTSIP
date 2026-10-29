package contract_authoring

import "fmt"

// CheckLocalProfile validates only explicit R-ID rule bindings and their references.
// The retired component-to-owner projection is intentionally not reintroduced.
func (c *Catalog) CheckLocalProfile(v Validator, profile Document) error {
	return c.CheckProfile(v, profile)
}

func (c *Catalog) checkComponentBindings(v Validator, profile Document) error {
	bindings := object(profile["component_bindings"])
	if bindings == nil { return fmt.Errorf("RULE_BINDINGS_REQUIRED") }
	conditions, _, err := c.definitionIDs()
	if err != nil { return err }
	conditionNames := map[string]string{}
	for _, raw := range c.Definitions["condition_list"].([]any) {
		row := object(raw)
		conditionNames[stringValue(row["ID"])] = stringValue(row["name"])
	}
	authorizations := c.authorizationIDs()
	for rid, raw := range bindings {
		binding := object(raw)
		if binding == nil { return fmt.Errorf("INVALID_RULE_BINDING: %s", rid) }
		if _, legacy := binding["Rule_state"]; legacy { return fmt.Errorf("RETIRED_POLICY_RULE_STATE: %s", rid) }
		if _, present := binding["status"]; !present { return fmt.Errorf("RESPONSIBILITY_STATUS_REQUIRED: %s", rid) }
		if ref := stringValue(binding["Authorization_ID"]); ref != "" {
			if _, exists := authorizations[ref]; !exists {
				return fmt.Errorf("UNRESOLVED_AUTHORIZATION_INTENTION: %s/%s", rid, ref)
			}
		}
		if policies, exists := binding["live_policy"]; exists {
			items, ok := policies.([]any)
			if !ok || len(items) == 0 { return fmt.Errorf("INVALID_LIVE_POLICY: %s", rid) }
		}
		if refs, exists := binding["condition_id"]; exists {
			items, ok := refs.([]any)
			if !ok || len(items) != 1 {
				return fmt.Errorf("RULE_CONDITION_NAME_CARDINALITY_UNRESOLVED: %s", rid)
			}
			ref := stringValue(items[0])
			if !conditions[ref] { return fmt.Errorf("UNRESOLVED_CONDITION: %s/%s", rid, ref) }
			if conditionNames[ref] != stringValue(binding["condition_name"]) {
				return fmt.Errorf("CONDITION_NAME_MISMATCH: %s/%s", rid, ref)
			}
		}
	}
	return nil
}

// The previous main-profile component metadata materializer has no successor.
func (c *Catalog) MaterializeMain(v Validator, profile Document) (Document, error) {
	return nil, fmt.Errorf("RETIRED_COMPONENT_METADATA_PROJECTION")
}
