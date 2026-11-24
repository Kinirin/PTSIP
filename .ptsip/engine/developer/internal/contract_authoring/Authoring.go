package contract_authoring

import "encoding/json"

func clone(input Document) Document {
	raw, _ := json.Marshal(input)
	var result Document
	_ = json.Unmarshal(raw, &result)
	if result == nil {
		result = Document{}
	}
	return result
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
	doc["component_bindings"] = Document{}
	for key, value := range clone(input) {
		doc[key] = value
	}
	if err := c.CheckProfile(v, doc); err != nil {
		return nil, err
	}
	return doc, nil
}
