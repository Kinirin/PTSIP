package authoring

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	legacy "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
)

const FamilySource = entry.IdentityVocabularyPath
const FamilySchema = legacy.FamilySchema
const CommonSchema = entry.CommonSchema
const RBACSchemaRoot = legacy.RBACSchemaRoot
const commonID = "urn:ptsip:candidate:contract:policy:rules:v2"

func stringValue(value any) string { text, _ := value.(string); return text }

// CheckContracts validates the registered v2 Family structures.
func (c *Catalog) CheckContracts(v Validator) error {
	for _, family := range c.Families {
		if _, err := v(c.Contracts[family], Document{}, c.Resources); err != nil {
			return fmt.Errorf("INVALID_FAMILY_STRUCTURE: %s: %w", family, err)
		}
	}
	return nil
}

// Load exposes the v2 schema catalog for explicit legacy contract checks.
// Unified candidate authoring resolves its own registered entry before this
// loader. Historical Rules/ data is never an authoring fallback.
func Load(root string) (*Catalog, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	boundary, err := legacy.ReadJSON(absolute, BoundaryPath)
	if err != nil {
		return nil, err
	}
	if boundary["domain"] != "policy" || object(boundary["namespaces"])["rules"] != "Policy_Rules/" {
		return nil, fmt.Errorf("POLICY_RULES_BOUNDARY_INVALID")
	}
	binding := object(boundary["rules_binding"])
	for key, expected := range map[string]string{
		"family_source": FamilySource, "directory": FamilyDirectory,
		"contract_schema": FamilySchema, "selection_schema": CommonSchema,
		"family_selection":       FamilyDirectory + "/Family_Selection.json",
		"responsibility_profile": ProfilePath, "responsibility_contract": DefinitionsPath,
	} {
		if binding[key] != expected {
			return nil, fmt.Errorf("RULES_BINDING_MISMATCH: %s", key)
		}
	}
	c := &Catalog{Catalog: &legacy.Catalog{Root: absolute, Contracts: map[string]Document{}}}
	vocabulary, err := legacy.ReadJSON(absolute, FamilySource)
	if err != nil {
		return nil, err
	}
	values, err := entry.IdentityValues(vocabulary, "root_family")
	if err != nil {
		return nil, err
	}
	registry, err := legacy.ReadJSON(absolute, ".ptsip/contracts/policy/candidate.index.json")
	if err != nil {
		return nil, err
	}
	if registry["active_registration"] != true || registry["active_policy_cutover"] != "STRUCTURE_V2_ACTIVE" {
		return nil, fmt.Errorf("V2_POLICY_STRUCTURE_NOT_ACTIVE")
	}
	known := map[string]bool{}
	add := func(file, id string) error {
		if err := entry.CheckPolicyContractPath(file); err != nil {
			return err
		}
		doc, err := legacy.ReadJSON(absolute, file)
		if err != nil {
			return err
		}
		if doc["$id"] != id {
			return fmt.Errorf("SCHEMA_ID_MISMATCH: %s", file)
		}
		if !known[id] {
			c.Resources = append(c.Resources, doc)
			known[id] = true
		}
		if id == "urn:ptsip:candidate:contract:policy:public-structure:v2" {
			c.Common = doc
		}
		return nil
	}
	artifacts, ok := registry["artifacts"].([]any)
	if !ok {
		return nil, fmt.Errorf("V2_SCHEMA_ARTIFACTS_MISSING")
	}
	for _, raw := range artifacts {
		item := object(raw)
		if item["kind"] == "SCHEMA" {
			file, id := stringValue(item["path"]), stringValue(item["schema_id"])
			if !strings.HasPrefix(file, ".ptsip/contracts/policy/") || id == "" {
				return nil, fmt.Errorf("V2_SCHEMA_OUTSIDE_POLICY_CONTRACT_ROOT")
			}
			if err := add(file, id); err != nil {
				return nil, err
			}
		}
	}
	for id, raw := range object(registry["external_resources"]) {
		item := object(raw)
		file := stringValue(item["path"])
		if !strings.HasPrefix(file, ".ptsip/contracts/policy/") && !strings.HasPrefix(file, RBACSchemaRoot) {
			return nil, fmt.Errorf("RETIRED_OR_UNREGISTERED_CONTRACT_ROOT: %s", file)
		}
		name, err := LocalFile(absolute, file)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
		if hex.EncodeToString(digest[:]) != stringValue(item["sha256"]) {
			return nil, fmt.Errorf("V2_EXTERNAL_RESOURCE_DIGEST_MISMATCH: %s", id)
		}
		if err := add(file, id); err != nil {
			return nil, err
		}
	}
	if c.Common == nil {
		return nil, fmt.Errorf("V2_PUBLIC_SCHEMA_UNRESOLVED")
	}
	selection, err := legacy.ReadJSON(absolute, FamilyDirectory+"/Family_Selection.json")
	if err != nil {
		return nil, err
	}
	if selection["active"] != true || selection["selection_basis"] != "DECLARED_POLICY_IDENTITY_ROOT_FAMILY" {
		return nil, fmt.Errorf("V2_FAMILY_SELECTION_INACTIVE")
	}
	routes := object(selection["family_routes"])
	for _, raw := range values {
		family := stringValue(raw)
		selected, err := entry.SelectPolicy(absolute, "PTSIP_DEVELOPER_POLICY", family)
		if err != nil {
			return nil, err
		}
		route := object(routes[family])
		ref := selected.ContractRefs["family_fields"]
		if route["rules_schema_ref"] != commonID || route["document_fields_ref"] != ref {
			return nil, fmt.Errorf("V2_FAMILY_RULE_SCHEMA_MISMATCH: %s", family)
		}
		c.Families = append(c.Families, family)
		c.Contracts[family] = Document{"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "urn:ptsip:policy:family-fields:" + family + ":v1", "$ref": ref}
	}
	if len(routes) != len(c.Families) {
		return nil, fmt.Errorf("V2_FAMILY_ROUTE_COUNT_MISMATCH")
	}
	return c, nil
}
