// Package contract_authoring generates declarations from the selected local
// contracts. JSON Schema validation is supplied by the installed PTSIP engine.
package contract_authoring

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Document = map[string]any
type Validator func(schema, instance any, resources []any) (bool, error)

const BoundaryPath = ".ptsip/contracts/policy/boundary.json"
const FamilySource = ".ptsip/contracts/policy/identity/root_family.json"
const FamilyDirectory = ".ptsip/contracts/policy/Rules"
const FamilySchema = ".ptsip/contracts/schemas/Policy/Policy_Rules/Family_Rules_Contract.schema.json"
const CommonSchema = ".ptsip/contracts/schemas/Policy/Policy_Rules/Public_Structure_Rules.schema.json"
const ProfilePath = ".ptsip/profiles/Responsibility-Authority-list.yaml"
const DefinitionsPath = ".ptsip/contracts/Responsibility-Authority/Responsibility-Authority.json"
const RBACSchemaRoot = ".ptsip/contracts/schemas/compile-sealed-rbac/"

const commonID = "urn:ptsip:policy:Public_Structure_Rules:v3"
const responsibilityID = "urn:ptsip:rbac:responsibility:v1"

type Catalog struct {
	Root            string
	Families        []string
	Contracts       map[string]Document
	Resources       []any
	ContractShape   Document
	Common          Document
	ProfileShape    Document
	Definitions     Document
	DefinitionShape Document
}

func object(value any) Document    { obj, _ := value.(map[string]any); return obj }
func stringValue(value any) string { valueString, _ := value.(string); return valueString }

// LocalFile accepts an existing regular file inside the repository. Every
// component is checked so a parent symlink cannot redirect a contract read.
func LocalFile(root, relative string) (string, error) {
	if relative == "" || path.IsAbs(relative) || path.Clean(relative) != relative ||
		strings.ContainsAny(relative, "\\:\x00") {
		return "", fmt.Errorf("INVALID_LOCAL_PATH: %s", relative)
	}
	current := root
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("INVALID_LOCAL_PATH: %s", relative)
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf("LOCAL_RESOURCE_UNAVAILABLE: %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("SYMLINK_RESOURCE: %s", relative)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("RESOURCE_PARENT_NOT_DIRECTORY: %s", relative)
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return "", fmt.Errorf("RESOURCE_NOT_REGULAR: %s", relative)
		}
	}
	return current, nil
}

func readJSON(root, relative string) (Document, error) {
	file, err := LocalFile(root, relative)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		return nil, fmt.Errorf("INVALID_CONTRACT_JSON: %s: %v", relative, err)
	}
	return doc, nil
}

func Load(root string) (*Catalog, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	boundary, err := readJSON(absolute, BoundaryPath)
	if err != nil {
		return nil, err
	}
	if boundary["domain"] != "policy" || object(boundary["namespaces"])["rules"] != "Rules/" {
		return nil, fmt.Errorf("POLICY_RULES_BOUNDARY_INVALID")
	}
	binding := object(boundary["rules_binding"])
	for key, expected := range map[string]string{
		"family_source": FamilySource, "directory": FamilyDirectory,
		"filename": "{root_family}_rules.json", "contract_schema": FamilySchema,
		"selection_schema": CommonSchema, "responsibility_profile": ProfilePath,
		"responsibility_contract": DefinitionsPath,
	} {
		if binding[key] != expected {
			return nil, fmt.Errorf("RULES_BINDING_MISMATCH: %s", key)
		}
	}
	c, err := LoadResponsibilities(absolute)
	if err != nil {
		return nil, err
	}
	vocabulary, err := readJSON(absolute, FamilySource)
	if err != nil {
		return nil, err
	}
	if vocabulary["$id"] != "urn:ptsip:identity:root-family:v1" {
		return nil, fmt.Errorf("FAMILY_VOCABULARY_ID_MISMATCH")
	}
	values, ok := vocabulary["enum"].([]any)
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("EMPTY_FAMILY_VOCABULARY")
	}
	seen := map[string]bool{}
	for _, raw := range values {
		family := stringValue(raw)
		if family == "" || seen[family] || strings.IndexFunc(family, func(r rune) bool { return r < 'A' || r > 'Z' }) >= 0 {
			return nil, fmt.Errorf("INVALID_FAMILY_VOCABULARY: %v", raw)
		}
		seen[family] = true
		c.Families = append(c.Families, family)
	}
	c.Resources = append(c.Resources, vocabulary)
	resources := []struct {
		file, id    string
		destination *Document
	}{
		{FamilySchema, "urn:ptsip:policy:family-rules-contract:v2", &c.ContractShape},
		{CommonSchema, commonID, &c.Common},
	}
	for _, resource := range resources {
		doc, err := readJSON(absolute, resource.file)
		if err != nil {
			return nil, err
		}
		if doc["$id"] != resource.id {
			return nil, fmt.Errorf("SCHEMA_ID_MISMATCH: %s", resource.file)
		}
		if resource.destination != nil {
			*resource.destination = doc
		}
		c.Resources = append(c.Resources, doc)
	}
	for _, family := range c.Families {
		file := FamilyDirectory + "/" + family + "_rules.json"
		doc, err := readJSON(absolute, file)
		if err != nil {
			return nil, err
		}
		if doc["$id"] != "urn:ptsip:contract:policy:rules:"+family+":v2" || doc["root_family"] != family ||
			object(doc["generation"])["default"] != "OMIT" || object(doc["generation"])["when_root_family"] != family {
			return nil, fmt.Errorf("FAMILY_CONTRACT_MISMATCH: %s", family)
		}
		c.Contracts[family] = doc
		// The data contract's $schema points to its authoring schema. Its
		// embedded structure is exposed as a Draft 2020-12 schema resource,
		// without interpreting that data-schema link as a schema dialect.
		c.Resources = append(c.Resources, Document{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$id":     doc["$id"], "structure": doc["structure"],
		})
	}
	return c, nil
}

// LoadResponsibilities has no policy-family or policy-lifecycle dependency.
// Profile rule bindings are scoped by explicit registered R-ID values.
func LoadResponsibilities(root string) (*Catalog, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	c := &Catalog{Root: absolute, Contracts: map[string]Document{}}
	resources := []struct {
		file, id    string
		destination *Document
	}{
		{RBACSchemaRoot + "Conditions_exercise/Condition.schema.json", "urn:ptsip:rbac:condition:v1", nil},
		{RBACSchemaRoot + "authority/Operational_Authority.schema.json", "urn:ptsip:rbac:operational-authority:v1", nil},
		{RBACSchemaRoot + "responsibility/Responsibility.schema.json", responsibilityID, nil},
		{RBACSchemaRoot + "Responsibilities_List.schema.json", "urn:ptsip:rbac:responsibilities-list:v2", &c.ProfileShape},
		{RBACSchemaRoot + "Responsibility_Authority_Contract.schema.json", "urn:ptsip:rbac:responsibility-authority-contract:v1", &c.DefinitionShape},
	}
	for _, resource := range resources {
		doc, err := readJSON(absolute, resource.file)
		if err != nil {
			return nil, err
		}
		if doc["$id"] != resource.id {
			return nil, fmt.Errorf("SCHEMA_ID_MISMATCH: %s", resource.file)
		}
		if resource.destination != nil {
			*resource.destination = doc
		}
		c.Resources = append(c.Resources, doc)
	}
	c.Definitions, err = readJSON(absolute, DefinitionsPath)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Catalog) validate(v Validator, schema, instance any, label string) error {
	resources := c.Resources
	if id := stringValue(object(schema)["$id"]); id != "" {
		resources = make([]any, 0, len(c.Resources))
		for _, resource := range c.Resources {
			if object(resource)["$id"] != id {
				resources = append(resources, resource)
			}
		}
	}
	valid, err := v(schema, instance, resources)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if !valid {
		return fmt.Errorf("SCHEMA_REJECTED: %s", label)
	}
	return nil
}

func (c *Catalog) CheckContracts(v Validator) error {
	if err := c.validate(v, c.DefinitionShape, c.Definitions, DefinitionsPath); err != nil {
		return err
	}
	if _, _, err := c.definitionIDs(); err != nil {
		return err
	}
	for _, family := range c.Families {
		contract := c.Contracts[family]
		if err := c.validate(v, c.ContractShape, contract, family+" rules contract"); err != nil {
			return err
		}
		// Compiling a structure rejects malformed schema keywords and unresolved
		// references even when the empty probe instance itself is invalid.
		if _, err := v(contract["structure"], Document{}, c.Resources); err != nil {
			return fmt.Errorf("INVALID_FAMILY_STRUCTURE: %s: %w", family, err)
		}
	}
	return nil
}

func ids(records any, label string) (map[string]bool, error) {
	items, ok := records.([]any)
	if !ok {
		return nil, fmt.Errorf("LIST_REQUIRED: %s", label)
	}
	seen := map[string]bool{}
	for _, raw := range items {
		id := stringValue(object(raw)["ID"])
		if id == "" || seen[id] {
			return nil, fmt.Errorf("DUPLICATE_OR_EMPTY_ID: %s/%s", label, id)
		}
		seen[id] = true
	}
	return seen, nil
}

func (c *Catalog) definitionIDs() (map[string]bool, map[string]bool, error) {
	conditions, err := ids(c.Definitions["condition_list"], "condition_list")
	if err != nil {
		return nil, nil, err
	}
	authorities, err := ids(c.Definitions["authority_list"], "authority_list")
	return conditions, authorities, err
}

func (c *Catalog) CheckProfile(v Validator, profile Document) error {
	if _, legacy := profile["component_map"]; legacy {
		return fmt.Errorf("RETIRED_COMPONENT_MAP_FIELD")
	}
	if err := c.validate(v, c.ProfileShape, profile, "responsibility-authority profile"); err != nil {
		return err
	}
	if err := c.validate(v, c.DefinitionShape, c.Definitions, DefinitionsPath); err != nil {
		return err
	}
	if _, err := ids(profile["responsibility"], "responsibility"); err != nil {
		return err
	}
	conditions, authorities, err := c.definitionIDs()
	if err != nil {
		return err
	}
	for _, raw := range profile["responsibility"].([]any) {
		responsibility := object(raw)
		for field, lookup := range map[string]map[string]bool{"condition_id": conditions, "Operational_Authority": authorities} {
			for _, ref := range responsibility[field].([]any) {
				if !lookup[stringValue(ref)] {
					return fmt.Errorf("UNRESOLVED_REFERENCE: %s/%s/%v", responsibility["ID"], field, ref)
				}
			}
		}
	}
	return c.checkComponentBindings(v, profile)
}

// checkRuleBindings resolves every rule token through the explicit R-ID registry.
// Resolving a rule does not activate it or grant an Operational Authority.
func (c *Catalog) checkRuleBindings(v Validator, doc Document) error {
	items, ok := object(doc["Policy_Rules"])["Definition"].([]any)
	if !ok || len(items) == 0 {
		return fmt.Errorf("POLICY_RULE_DEFINITIONS_REQUIRED")
	}
	profile, err := ReadDocument(c.Root, ProfilePath)
	if err != nil {
		return err
	}
	if err := c.CheckLocalProfile(v, profile); err != nil {
		return err
	}
	bindings := object(profile["component_bindings"])
	if bindings == nil {
		return fmt.Errorf("RULE_BINDINGS_REQUIRED")
	}
	seen := map[string]bool{}
	for _, raw := range items {
		definition := object(raw)
		id, rule := stringValue(definition["id"]), stringValue(definition["rule"])
		if id == "" || seen[id] {
			return fmt.Errorf("INVALID_OR_DUPLICATE_RULE_ID: %s", id)
		}
		seen[id] = true
		binding := object(bindings[id])
		if binding == nil {
			return fmt.Errorf("UNRESOLVED_RESPONSIBILITY: %s", id)
		}
		if stringValue(binding["rule"]) != rule {
			return fmt.Errorf("RULE_CONSTANT_MISMATCH: %s", id)
		}
		// The policy may select registered conditions, but must never author them.
		// Require an exact set match, including the absence of preconditions.
		registered := map[string]bool{}
		if raw, present := binding["condition_id"]; present {
			for _, ref := range raw.([]any) {
				registered[stringValue(ref)] = true
			}
		}
		declared := map[string]bool{}
		if raw, present := definition["preconditions"]; present {
			items, ok := raw.([]any)
			if !ok || len(items) == 0 {
				return fmt.Errorf("INVALID_RULE_PRECONDITIONS: %s", id)
			}
			for _, ref := range items {
				name := stringValue(ref)
				if name == "" || declared[name] {
					return fmt.Errorf("INVALID_RULE_PRECONDITIONS: %s/%s", id, name)
				}
				declared[name] = true
			}
		}
		if len(registered) != len(declared) {
			return fmt.Errorf("RULE_PRECONDITIONS_PROFILE_MISMATCH: %s", id)
		}
		for name := range declared {
			if !registered[name] {
				return fmt.Errorf("UNREGISTERED_RULE_PRECONDITION: %s/%s", id, name)
			}
		}
	}
	return nil
}

func keys(doc Document) []string {
	result := make([]string, 0, len(doc))
	for key := range doc {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
