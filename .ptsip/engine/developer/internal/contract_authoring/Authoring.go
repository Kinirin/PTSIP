package contract_authoring

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func clone(input Document) Document {
	raw, _ := json.Marshal(input)
	var result Document
	_ = json.Unmarshal(raw, &result)
	if result == nil {
		result = Document{}
	}
	return result
}

func (c *Catalog) selectedFamily(doc Document, requested string) (string, error) {
	identity, present := doc["Policy_Identity"]
	if present && object(identity) == nil {
		return "", fmt.Errorf("POLICY_IDENTITY_MUST_BE_OBJECT")
	}
	declared, declaredPresent := object(identity)["root_family"]
	if declaredPresent {
		value, ok := declared.(string)
		if !ok || value == "" {
			return "", fmt.Errorf("ROOT_FAMILY_MUST_BE_NONEMPTY_STRING")
		}
		if requested != "" && requested != value {
			return "", fmt.Errorf("ROOT_FAMILY_CONFLICT: %s/%s", requested, value)
		}
		requested = value
	}
	if requested != "" && c.Contracts[requested] == nil {
		return "", fmt.Errorf("UNKNOWN_ROOT_FAMILY: %s", requested)
	}
	return requested, nil
}

func (c *Catalog) CheckRules(v Validator, doc Document) error {
	family, err := c.selectedFamily(doc, "")
	if err != nil {
		return err
	}
	selection := Document{"$ref": commonID + "#/$defs/PolicyDocumentSelection"}
	if err := c.validate(v, selection, doc, "conditional policy rules"); err != nil {
		return err
	}
	if family != "" {
		if err := c.checkResponsibilityRefs(v, c.Contracts[family]["responsibility_refs"]); err != nil {
			return err
		}
	}
	return c.checkResponsibilityRefs(v, object(doc["Policy_Rules"])["Responsibility_Refs"])
}

// materialize creates only required object/array containers. Required scalar
// values, including approval state and rule identity, must come from the input.
// A family constant is the sole scalar supplied from the explicit selection.
func materialize(shape Document, value any, location, family string, missing *[]string) any {
	if shape["type"] == "object" {
		current := object(value)
		if current == nil {
			if value != nil {
				return value
			}
			current = Document{}
		}
		properties := object(shape["properties"])
		required := map[string]bool{}
		if entries, ok := shape["required"].([]any); ok {
			for _, entry := range entries {
				required[stringValue(entry)] = true
			}
		}
		for _, key := range keys(properties) {
			child, present := current[key]
			if !present && !required[key] {
				continue
			}
			childShape := object(properties[key])
			if !present && childShape["type"] != "object" && childShape["type"] != "array" {
				if key == "family" && childShape["const"] == family {
					current[key] = family
					continue
				}
				*missing = append(*missing, location+"."+key)
				continue
			}
			current[key] = materialize(childShape, child, location+"."+key, family, missing)
		}
		return current
	}
	if shape["type"] == "array" && value == nil {
		return []any{}
	}
	return value
}

func (c *Catalog) GenerateRules(v Validator, input Document, requested string) (Document, string, error) {
	doc := clone(input)
	family, err := c.selectedFamily(doc, requested)
	if err != nil {
		return nil, "", err
	}
	if family == "" {
		if err := c.CheckRules(v, doc); err != nil {
			return nil, "", err
		}
		return doc, "OMITTED", nil
	}
	identity := object(doc["Policy_Identity"])
	if identity == nil {
		identity = Document{}
		doc["Policy_Identity"] = identity
	}
	identity["root_family"] = family
	if present, ok := doc["Policy_Rules"]; ok && object(present) == nil {
		return nil, "", fmt.Errorf("POLICY_RULES_MUST_BE_OBJECT")
	}
	missing := []string{}
	doc["Policy_Rules"] = materialize(object(c.Contracts[family]["structure"]), doc["Policy_Rules"], "Policy_Rules", family, &missing)
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, "", fmt.Errorf("REQUIRED_INPUT_VALUES: %s", strings.Join(missing, ", "))
	}
	if err := c.CheckRules(v, doc); err != nil {
		return nil, "", err
	}
	return doc, "GENERATED", nil
}

func (c *Catalog) GenerateProfile(v Validator, input Document) (Document, error) {
	base, err := ReadDocument(c.Root, ProfilePath)
	if err != nil {
		return nil, err
	}
	doc := clone(base)
	// Metadata is retained from the editable profile. An omitted responsibility
	// list generates no mappings; it never copies or invents grants.
	doc["responsibility"] = []any{}
	delete(doc, "component_map")
	delete(doc, "component_bindings")
	for key, value := range clone(input) {
		doc[key] = value
	}
	if err := c.CheckProfile(v, doc); err != nil {
		return nil, err
	}
	return doc, nil
}
