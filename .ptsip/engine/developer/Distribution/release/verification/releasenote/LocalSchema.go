package releasenote

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type closedDocumentationLoader struct{}

func (closedDocumentationLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("EXTERNAL_DOCUMENTATION_SCHEMA_FORBIDDEN: %s", url)
}

// ValidateLocal validates a release document using only the supplied schemas.
func ValidateLocal(schema, instance any, resources []any) (bool, error) {
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(closedDocumentationLoader{})
	for _, resource := range resources {
		document, ok := resource.(map[string]any)
		if !ok {
			return false, fmt.Errorf("DOCUMENTATION_RESOURCE_OBJECT_REQUIRED")
		}
		id, _ := document["$id"].(string)
		if id == "" {
			return false, fmt.Errorf("DOCUMENTATION_RESOURCE_ID_REQUIRED")
		}
		if err := compiler.AddResource(id, document); err != nil {
			return false, err
		}
	}
	const id = "urn:ptsip:docs:validation-request:v1"
	if err := compiler.AddResource(id, schema); err != nil {
		return false, err
	}
	compiled, err := compiler.Compile(id)
	if err != nil {
		return false, err
	}
	if err := compiled.Validate(instance); err != nil {
		return false, nil
	}
	return true, nil
}
