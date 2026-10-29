package contract_authoring

import (
	"fmt"
	"os"
	"path/filepath"
)

const MainProfilePath = ".ptsip/profiles/main.ptsip.yaml"
const LocalProfileCatalogPath = ".ptsip/profiles/index.json"
const mainProfileSchemaPath = "src/ptsip/specdata/ptsip-profile-pp-1.02.schema.json"

var componentMetadataFields = []string{"roles", "release_owner", "compatibility_owner", "runtime_required"}

// The default editable local list cannot drop all component assignments while
// its linked main source/catalog remains present. Unlinked authoring fixtures
// and generated candidates can still declare an empty responsibility list.
func (c *Catalog) CheckLocalProfile(v Validator, profile Document) error {
	if _, present := profile["component_bindings"]; !present {
		for _, relative := range []string{MainProfilePath, LocalProfileCatalogPath} {
			_, err := os.Lstat(filepath.Join(c.Root, filepath.FromSlash(relative)))
			if err == nil || !os.IsNotExist(err) {
				return fmt.Errorf("COMPONENT_BINDINGS_REQUIRED_FOR_LINKED_PROFILE")
			}
		}
	}
	return c.CheckProfile(v, profile)
}

// Component bindings preserve declared owner labels and roles. They are not
// operational authorities and never create responsibility execution conditions.
func (c *Catalog) componentProjection(profile Document) (Document, error) {
	raw, exists := profile["component_bindings"]
	if !exists {
		return nil, nil
	}
	bindings, ok := raw.([]any)
	if !ok || len(bindings) == 0 {
		return nil, fmt.Errorf("COMPONENT_BINDINGS_REQUIRED")
	}
	catalog, err := readJSON(c.Root, LocalProfileCatalogPath)
	if err != nil {
		return nil, err
	}
	if catalog["schema_version"] != "ptsip-local-profile-catalog/v1" || catalog["default_profile"] != "main" {
		return nil, fmt.Errorf("COMPONENT_BINDING_CATALOG_INVALID")
	}
	rows, ok := catalog["profiles"].([]any)
	if !ok {
		return nil, fmt.Errorf("COMPONENT_BINDING_CATALOG_INVALID")
	}
	selected := 0
	for _, raw := range rows {
		row := object(raw)
		if row["id"] == "main" {
			if row["resource"] != "main.ptsip.yaml" || row["responsibility_authority_ref"] != "responsibility-authority" {
				return nil, fmt.Errorf("COMPONENT_BINDING_CATALOG_ROUTE_INVALID")
			}
			selected++
		}
	}
	auxiliary := object(object(catalog["auxiliary_profiles"])["responsibility-authority"])
	if selected != 1 || auxiliary["resource"] != "Responsibility-Authority-list.yaml" || auxiliary["profile_role"] != "RESPONSIBILITY-AUTHORITY" || auxiliary["schema_ref"] != "../contracts/schemas/compile-sealed-rbac/Responsibilities_List.schema.json" || auxiliary["source_profile"] != "main.ptsip.yaml" {
		return nil, fmt.Errorf("COMPONENT_BINDING_CATALOG_ROUTE_INVALID")
	}
	main, err := ReadDocument(c.Root, MainProfilePath)
	if err != nil {
		return nil, err
	}
	if _, retired := object(main["ptsip"])["rule_profile"]; retired {
		return nil, fmt.Errorf("RETIRED_RULE_PROFILE_BINDING")
	}
	components, ok := main["components"].([]any)
	if !ok || len(components) == 0 {
		return nil, fmt.Errorf("MAIN_COMPONENTS_REQUIRED")
	}
	known := map[string]Document{}
	for _, raw := range components {
		row := object(raw)
		id := stringValue(row["id"])
		if id == "" || known[id] != nil {
			return nil, fmt.Errorf("MAIN_COMPONENT_DUPLICATE_OR_EMPTY: %s", id)
		}
		for _, field := range componentMetadataFields {
			if _, mixed := row[field]; mixed {
				return nil, fmt.Errorf("COMPONENT_METADATA_MUST_USE_RESPONSIBILITY_PROFILE: %s/%s", id, field)
			}
		}
		known[id] = row
	}
	seen := map[string]bool{}
	for _, raw := range bindings {
		binding := object(raw)
		id := stringValue(binding["component_id"])
		if known[id] == nil {
			return nil, fmt.Errorf("UNKNOWN_COMPONENT_BINDING: %s", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("DUPLICATE_COMPONENT_BINDING: %s", id)
		}
		seen[id] = true
		for _, field := range componentMetadataFields {
			if value, present := binding[field]; present {
				known[id][field] = value
			}
		}
	}
	for id := range known {
		if !seen[id] {
			return nil, fmt.Errorf("MISSING_COMPONENT_BINDING: %s", id)
		}
	}
	return main, nil
}

func (c *Catalog) checkComponentBindings(v Validator, profile Document) error {
	projection, err := c.componentProjection(profile)
	if err != nil || projection == nil {
		return err
	}
	schema, err := readJSON(c.Root, mainProfileSchemaPath)
	if err != nil {
		return err
	}
	return c.validate(v, schema, projection, "materialized local main profile")
}

// MaterializeMain explicitly reconstructs an existing pp.1.02-shaped effective
// record for inspection or downstream consumer validation. The projection has no
// separate authority and does not modify either authored source.
func (c *Catalog) MaterializeMain(v Validator, profile Document) (Document, error) {
	if err := c.CheckProfile(v, profile); err != nil {
		return nil, err
	}
	projection, err := c.componentProjection(profile)
	if err != nil {
		return nil, err
	}
	if projection == nil {
		return nil, fmt.Errorf("COMPONENT_BINDINGS_REQUIRED_FOR_MATERIALIZATION")
	}
	return projection, nil
}
