package boundary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

const platformRoot = ".ptsip/contracts/release/Platform/"

type platformRoute struct {
	Platform   string            `json:"platform"`
	Root       string            `json:"root"`
	Contracts  map[string]string `json:"contracts"`
	Validation string            `json:"validation_ref"`
}

// These routes describe storage and structural validation only. They do not
// authorize publication on either service or recover retired admission input.
func expectedPlatforms() map[string]platformRoute {
	files := map[string]map[string]string{
		"Common": {
			"admission":                   "admission.json",
			"composition":                 "composition-contract.json",
			"migration_inventory":         "migration-inventory.json",
			"package_installation":        "package-installation-contract.json",
			"release":                     "release-contract.json",
			"release_contract_validation": "release-contract_validation.json",
			"release_documentation":       "releasenote.json",
			"release_validation":          "release_validation.json",
			"verification_boundary":       "verification-boundary.json",
			"verification":                "verification-contract.json",
			"version_changes":             "versions_different.json",
			"profile_routing":             "profile/index.json",
			"profile_snapshot":            "profile/project-profile-contracts.yaml",
		},
		"GitHub": {
			"publication_evidence": "publication-evidence.json",
			"release_presentation": "release-presentation.json",
		},
		"PyPI": {"publisher_routing": "pypi-publishers.json"},
	}
	routes := make(map[string]platformRoute, len(files))
	for name, members := range files {
		root := platformRoot + name + "/"
		contracts := make(map[string]string, len(members))
		for id, member := range members {
			contracts[id] = root + member
		}
		routes[name] = platformRoute{Platform: name, Root: root, Contracts: contracts, Validation: root + name + "_validation.json"}
	}
	return routes
}

// The schema representation supports only the owned routing schema structure.
// Checking its fixed declaration is not JSON Schema evaluation; consumers may
// evaluate the schemas using a standard validator independently.
type routeSchemaProperty struct {
	Type       string                         `json:"type,omitempty"`
	Const      string                         `json:"const,omitempty"`
	Required   []string                       `json:"required,omitempty"`
	Additional *bool                          `json:"additionalProperties,omitempty"`
	Properties map[string]routeSchemaProperty `json:"properties,omitempty"`
}

type routeSchema struct {
	Schema     string                         `json:"$schema"`
	ID         string                         `json:"$id"`
	Title      string                         `json:"title"`
	Type       string                         `json:"type"`
	Required   []string                       `json:"required"`
	Additional *bool                          `json:"additionalProperties"`
	Properties map[string]routeSchemaProperty `json:"properties"`
}

func expectedRouteSchema(route platformRoute) routeSchema {
	no := false
	keys := make([]string, 0, len(route.Contracts))
	properties := make(map[string]routeSchemaProperty, len(route.Contracts))
	for id, path := range route.Contracts {
		keys = append(keys, id)
		properties[id] = routeSchemaProperty{Const: path}
	}
	sort.Strings(keys)
	return routeSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		ID:     "urn:ptsip:release:platform:" + strings.ToLower(route.Platform) + "-routing:v1",
		Title:  "PTSIP " + route.Platform + " Release Contract Routing",
		Type:   "object", Required: []string{"platform", "root", "contracts", "validation_ref"}, Additional: &no,
		Properties: map[string]routeSchemaProperty{
			"platform":       {Const: route.Platform},
			"root":           {Const: route.Root},
			"contracts":      {Type: "object", Required: keys, Additional: &no, Properties: properties},
			"validation_ref": {Const: route.Validation},
		},
	}
}

func strictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("RELEASE_PLATFORM_TRAILING_JSON")
	}
	return nil
}

func checkPlatforms(root string) error {
	filename, err := authoring.LocalFile(root, ".ptsip/contracts/release/index.json")
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	var catalog struct {
		Dimension string          `json:"platform_dimension"`
		Platforms json.RawMessage `json:"platforms"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return err
	}
	var routes []platformRoute
	if err := strictJSON(catalog.Platforms, &routes); err != nil {
		return fmt.Errorf("RELEASE_PLATFORM_ROUTING_INVALID: %w", err)
	}
	expected := expectedPlatforms()
	if catalog.Dimension != "DISTRIBUTION_SERVICE" || len(routes) != len(expected) {
		return fmt.Errorf("RELEASE_PLATFORM_DIMENSION_OR_INVENTORY_INVALID")
	}
	seen := map[string]bool{}
	allowedFiles := map[string]bool{}
	for _, route := range routes {
		want, exists := expected[route.Platform]
		if !exists || seen[route.Platform] || route.Root != want.Root || route.Validation != want.Validation || !maps.Equal(route.Contracts, want.Contracts) {
			return fmt.Errorf("RELEASE_PLATFORM_ROUTING_INVALID: %s", route.Platform)
		}
		seen[route.Platform] = true
		for _, path := range route.Contracts {
			if _, err := authoring.LocalFile(root, path); err != nil {
				return fmt.Errorf("RELEASE_PLATFORM_RESOURCE_INVALID: %s: %w", path, err)
			}
			allowedFiles[path] = true
		}
		filename, err := authoring.LocalFile(root, route.Validation)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		var schema routeSchema
		if err := strictJSON(raw, &schema); err != nil {
			return fmt.Errorf("RELEASE_PLATFORM_SCHEMA_INVALID: %s: %w", route.Platform, err)
		}
		if !reflect.DeepEqual(schema, expectedRouteSchema(want)) {
			return fmt.Errorf("RELEASE_PLATFORM_SCHEMA_BINDING_INVALID: %s", route.Platform)
		}
		allowedFiles[route.Validation] = true
	}
	return filepath.WalkDir(filepath.Join(root, filepath.FromSlash(platformRoot)), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative == strings.TrimSuffix(platformRoot, "/") {
				return nil
			}
			if relative == platformRoot+"Common/profile" {
				return nil
			}
			for name := range expected {
				if relative == platformRoot+name {
					return nil
				}
			}
		} else if allowedFiles[relative] {
			return nil
		}
		return fmt.Errorf("RELEASE_PLATFORM_UNREGISTERED_RESOURCE: %s", relative)
	})
}
