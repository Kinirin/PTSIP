package main

import (
	"fmt"

	documentation "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/releasenote"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type closedDocumentationLoader struct{}

func (closedDocumentationLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("EXTERNAL_DOCUMENTATION_SCHEMA_FORBIDDEN: %s", url)
}

func documentationValidator(schema, instance any, resources []any) (bool, error) {
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

func validateTranslationRegistry(root string, r translationRegistry) error {
	value := map[string]any{"schema_version": r.SchemaVersion, "canonical": map[string]any{"language": r.Canonical.Language, "path": r.Canonical.Path}, "translations": []any{}}
	for _, target := range r.Translations {
		value["translations"] = append(value["translations"].([]any), map[string]any{"language": target.Language, "path": target.Path})
	}
	return documentation.ValidateRegistry(root, value, documentationValidator)
}
