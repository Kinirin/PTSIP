package main

import (
	documentation "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/releasenote"
)

func validateTranslationRegistry(root string, r translationRegistry) error {
	value := map[string]any{"schema_version": r.SchemaVersion, "canonical": map[string]any{"language": r.Canonical.Language, "path": r.Canonical.Path}, "translations": []any{}}
	for _, target := range r.Translations {
		value["translations"] = append(value["translations"].([]any), map[string]any{"language": target.Language, "path": target.Path})
	}
	return documentation.ValidateRegistry(root, value, documentation.ValidateLocal)
}
