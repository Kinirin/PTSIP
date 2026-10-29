package contract_authoring

import "fmt"

// CheckLocalProfile validates only explicit R-ID rule bindings and their references.
// The retired component-to-owner projection is intentionally not reintroduced.
func (c *Catalog) CheckLocalProfile(v Validator, profile Document) error {
	return c.CheckProfile(v, profile)
}

func (c *Catalog) checkComponentBindings(v Validator, profile Document) error {
	bindings := object(profile["component_bindings"])
	if bindings == nil {
		return fmt.Errorf("RULE_BINDINGS_REQUIRED")
	}
	conditions, _, err := c.definitionIDs()
	if err != nil {
		return err
	}
	for id, raw := range bindings {
		binding := object(raw)
		if binding == nil {
			return fmt.Errorf("INVALID_RULE_BINDING: %s", id)
		}
		if _, ok := binding["Rule_state"]; !ok {
			return fmt.Errorf("RULE_STATE_REQUIRED: %s", id)
		}
		if refs, exists := binding["condition_id"]; exists {
			items, ok := refs.([]any)
			if !ok {
				return fmt.Errorf("INVALID_RULE_CONDITIONS: %s", id)
			}
			for _, ref := range items {
				if !conditions[stringValue(ref)] {
					return fmt.Errorf("UNRESOLVED_CONDITION: %s/%v", id, ref)
				}
			}
		}
	}
	return nil
}

// The previous main-profile component metadata materializer has no successor.
func (c *Catalog) MaterializeMain(v Validator, profile Document) (Document, error) {
	return nil, fmt.Errorf("RETIRED_COMPONENT_METADATA_PROJECTION")
}
