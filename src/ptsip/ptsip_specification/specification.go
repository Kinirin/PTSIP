// Package specification reads the declared versions and registered product resources.
package specification

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

type Schema struct {
	ID      string `json:"$id"`
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"schema_sha256"`
}

type Specification struct {
	root     string
	versions versionsource.Versions
	schemas  map[string]Schema
}

// OpenSource opens a checkout rooted at the directory containing src/VERSION.
func OpenSource(root string) (*Specification, error) {
	root, err := canonicalRoot(root)
	if err != nil {
		return nil, err
	}
	versions, err := versionsource.Read(root)
	if err != nil {
		return nil, err
	}
	return open(filepath.Join(root, "src", "ptsip"), versions)
}

// OpenInstalled opens the product directory that contains the installed VERSION.
func OpenInstalled(root string) (*Specification, error) {
	root, err := canonicalRoot(root)
	if err != nil {
		return nil, err
	}
	raw, err := asset(root, "VERSION")
	if err != nil {
		return nil, err
	}
	versions, err := versionsource.Parse(raw)
	if err != nil {
		return nil, err
	}
	return open(root, versions)
}

func open(product string, versions versionsource.Versions) (*Specification, error) {
	product, err := canonicalRoot(product)
	if err != nil {
		return nil, err
	}
	catalog, err := asset(product, "schemas/config.json")
	if err != nil {
		return nil, err
	}
	manifest, err := asset(product, "common_contract/registry/manifest.json")
	if err != nil {
		return nil, err
	}
	if err := versions.VerifyOwners(catalog, manifest); err != nil {
		return nil, err
	}
	var registered struct {
		Policy map[string]Schema `json:"Policy"`
		Plan   map[string]Schema `json:"plan"`
		Store  struct {
			Profile map[string]Schema `json:"profile"`
			RBAC    map[string]Schema `json:"Responsibility-Authority"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(catalog, &registered); err != nil {
		return nil, err
	}
	var common struct {
		Contracts []struct {
			ID       string `json:"urn"`
			Resource string `json:"resource"`
			SHA256   string `json:"sha256"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(manifest, &common); err != nil {
		return nil, err
	}
	schemas := make(map[string]Schema, len(versions.Schemas))
	for _, group := range []map[string]Schema{registered.Policy, registered.Plan, registered.Store.Profile, registered.Store.RBAC} {
		for key, entry := range group {
			if _, current := versions.Schemas[key]; !current {
				continue
			}
			if _, duplicate := schemas[key]; duplicate || entry.ID == "" || entry.Path == "" {
				return nil, fmt.Errorf("invalid or ambiguous registered schema %s", key)
			}
			for _, pin := range common.Contracts {
				if pin.ID == entry.ID && pin.Resource == "schemas/"+entry.Path {
					if len(pin.SHA256) != 64 || (entry.SHA256 != "" && entry.SHA256 != pin.SHA256) {
						return nil, fmt.Errorf("invalid schema pin for %s", key)
					}
					entry.SHA256 = pin.SHA256
				}
			}
			schemas[key] = entry
		}
	}
	if len(schemas) != len(versions.Schemas) {
		return nil, errors.New("current declared schema is missing from the registry")
	}
	return &Specification{root: product, versions: versions, schemas: schemas}, nil
}

func (s *Specification) Versions() versionsource.Versions {
	versions := s.versions
	versions.Engines = maps.Clone(versions.Engines)
	versions.Schemas = maps.Clone(versions.Schemas)
	return versions
}

func (s *Specification) VerifyRegisteredSchemas() error {
	for _, key := range slices.Sorted(maps.Keys(s.schemas)) {
		if _, _, err := s.RegisteredSchema(key); err != nil {
			return err
		}
	}
	return nil
}

// RegisteredSchema verifies the published resource identity and any registered
// digest before returning the bytes and its observed SHA256.
func (s *Specification) RegisteredSchema(key string) (Schema, []byte, error) {
	entry, ok := s.schemas[key]
	if !ok {
		return Schema{}, nil, fmt.Errorf("unknown declared schema %s", key)
	}
	if !source.Relative(entry.Path) {
		return Schema{}, nil, fmt.Errorf("invalid registered schema path for %s", key)
	}
	raw, err := asset(s.root, "schemas/"+entry.Path)
	if err != nil {
		return Schema{}, nil, err
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return Schema{}, nil, err
	}
	doc := source.Object(value)
	declared := doc["version"]
	if key == versionsource.ProfileKey {
		properties := source.Object(doc["properties"])
		identity := source.Object(source.Object(properties["ptsip"])["properties"])
		declared = source.Object(identity["version"])["const"]
	}
	if doc["$id"] != entry.ID || declared != entry.Version {
		return Schema{}, nil, fmt.Errorf("resource identity or version differs for %s", key)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	if entry.SHA256 != "" && digest != entry.SHA256 {
		return Schema{}, nil, fmt.Errorf("resource SHA256 differs for %s", key)
	}
	entry.SHA256 = digest
	return entry, raw, nil
}

func canonicalRoot(root string) (string, error) {
	if root == "" {
		return "", errors.New("explicit product or repository root required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func asset(root, relative string) ([]byte, error) {
	file := filepath.Join(root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(file)
	if err != nil {
		return nil, err
	}
	if resolved != file {
		return nil, fmt.Errorf("registered product asset is an alias: %s", relative)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("registered product asset is not a file: %s", relative)
	}
	return os.ReadFile(resolved)
}
