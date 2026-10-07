package testrepo

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func (r *Repository) LoadNeutralPolicyIndex() (Object, error) {
	payload, err := r.Read("developer/policy/index.yaml")
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/developer-policy-catalog.schema.json", payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (r *Repository) AtomicWrite(ref string, raw []byte, expected *string) error {
	path, err := r.Path(ref)
	if err != nil {
		return err
	}
	if expected != nil {
		prior, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(prior)) != *expected {
			return fmt.Errorf("STALE_REGISTRY_DIGEST")
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

func (r *Repository) WriteYAML(ref string, value any, expected *string) error {
	raw, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return r.AtomicWrite(ref, raw, expected)
}
