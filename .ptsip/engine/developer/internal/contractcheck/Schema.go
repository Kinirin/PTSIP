// Package contractcheck loads explicit current contracts without registry or
// policy-resolver discovery. Schema evaluation belongs to PTSIP VALIDATION.
package contractcheck

import (
	"encoding/json"
	"fmt"
	"os"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

type Source struct{ Path, ID string }

func Load(root string, sources ...Source) ([]any, error) {
	resources := make([]any, 0, len(sources))
	seen := map[string]bool{}
	for _, source := range sources {
		filename, err := authoring.LocalFile(root, source.Path)
		if err != nil {
			return nil, err
		}
		raw, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var document map[string]any
		if err := json.Unmarshal(raw, &document); err != nil {
			return nil, fmt.Errorf("INVALID_CONTRACT_JSON: %s: %w", source.Path, err)
		}
		id, _ := document["$id"].(string)
		if id == "" || (source.ID != "" && id != source.ID) || seen[id] {
			return nil, fmt.Errorf("CONTRACT_ID_MISMATCH_OR_DUPLICATE: %s", source.Path)
		}
		seen[id] = true
		resources = append(resources, document)
	}
	return resources, nil
}

func Validate(validate authoring.Validator, schema map[string]any, instance any, resources []any) error {
	filtered := make([]any, 0, len(resources))
	for _, resource := range resources {
		document, ok := resource.(map[string]any)
		if !ok {
			return fmt.Errorf("SCHEMA_RESOURCE_OBJECT_REQUIRED")
		}
		if schema["$id"] == nil || document["$id"] != schema["$id"] {
			filtered = append(filtered, resource)
		}
	}
	valid, err := validate(schema, instance, filtered)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("SCHEMA_REJECTED: %v", schema["$id"])
	}
	return nil
}
