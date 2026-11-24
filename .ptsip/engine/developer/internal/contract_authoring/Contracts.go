// Package contract_authoring generates declarations from the selected local
// contracts. JSON Schema validation is supplied by the installed PTSIP engine.
package contract_authoring

import (
	"encoding/json"
	"fmt"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Document = map[string]any
type Validator func(schema, instance any, resources []any) (bool, error)

const BoundaryPath = entry.BoundaryPath
const IdentityVocabularyPath = entry.IdentityVocabularyPath
const IdentityVocabularyID = entry.IdentityVocabularyID
const FamilySource = IdentityVocabularyPath
const FamilyDirectory = entry.FamilyDirectory
const FamilySchema = ".ptsip/contracts/policy/Policy_Rules/Contract_Types.schema.json"
const CommonSchema = entry.CommonSchema
const ProfilePath = ".ptsip/profiles/Responsibility-Authority-list.yaml"
const DefinitionsPath = ".ptsip/contracts/Responsibility-Authority/Responsibility-Authority.json"
const RBACSchemaRoot = ".ptsip/contracts/compile-sealed-rbac/"
const AuthorizationDataPath = RBACSchemaRoot + "Authority/Authorization_intention.json"
const AuthorizationSchemaPath = RBACSchemaRoot + "Authority/Authorization_intention.schema.json"
const AuthorizationDecisionDataPath = ".ptsip/contracts/Responsibility-Authority/Authorization_decision.json"
const AuthorizationDecisionSchemaPath = RBACSchemaRoot + "Authority/Authorization_decision.schema.json"

const commonID = "urn:ptsip:candidate:contract:policy:rules:v2"
const responsibilityID = "urn:ptsip:rbac:responsibility:v1"

type Catalog struct {
	Root                       string
	Families                   []string
	Contracts                  map[string]Document
	Resources                  []any
	Common                     Document
	ProfileShape               Document
	Definitions                Document
	DefinitionShape            Document
	AuthorizationShape         Document
	Authorizations             Document
	AuthorizationDecisionShape Document
	AuthorizationDecision      Document
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

func ReadJSON(root, relative string) (Document, error) {
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
		{RBACSchemaRoot + "Authority/Operational_Authority.schema.json", "urn:ptsip:rbac:operational-authority:v1", nil},
		{RBACSchemaRoot + "Responsibility/Responsibility.schema.json", responsibilityID, nil},
		{RBACSchemaRoot + "Responsibility/Responsibilities_List.schema.json", "urn:ptsip:rbac:responsibilities-list:v3", &c.ProfileShape},
		{RBACSchemaRoot + "Responsibility/Responsibility_Authority_Contract.schema.json", "urn:ptsip:rbac:responsibility-authority-contract:v1", &c.DefinitionShape},
		{RBACSchemaRoot + "lifecycle/status.json", "urn:ptsip:rbac:lifecycle:status:v1", nil},
		{RBACSchemaRoot + "lifecycle/keywords.json", "urn:ptsip:rbac:lifecycle:keywords:v1", nil},
		{RBACSchemaRoot + "lifecycle/index.json", "urn:ptsip:rbac:lifecycle:index:v1", nil},
		{AuthorizationSchemaPath, "urn:ptsip:rbac:authorization-intention:v1", &c.AuthorizationShape},
		{AuthorizationDecisionSchemaPath, "urn:ptsip:rbac:authorization-decision:v1", &c.AuthorizationDecisionShape},
	}
	for _, resource := range resources {
		doc, err := ReadJSON(absolute, resource.file)
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
	c.Definitions, err = ReadJSON(absolute, DefinitionsPath)
	if err != nil {
		return nil, err
	}
	c.Authorizations, err = ReadJSON(absolute, AuthorizationDataPath)
	if err != nil {
		return nil, err
	}
	c.AuthorizationDecision, err = ReadJSON(absolute, AuthorizationDecisionDataPath)
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
	if err := c.checkAuthorizationDecision(v); err != nil {
		return err
	}
	if err := c.checkAuthorizationRegistry(v); err != nil {
		return err
	}
	if err := c.validate(v, c.DefinitionShape, c.Definitions, DefinitionsPath); err != nil {
		return err
	}
	if _, _, err := c.definitionIDs(); err != nil {
		return err
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
	if err := c.checkAuthorizationDecision(v); err != nil {
		return err
	}
	if err := c.checkAuthorizationRegistry(v); err != nil {
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

// authorizationIDs returns only explicitly registered lookup codes.
// A prefix conveys a requested action class, never a grant.
func (c *Catalog) authorizationIDs() map[string]Document {
	known := map[string]Document{}
	for _, raw := range c.Authorizations["authorizations"].([]any) {
		row := object(raw)
		known[stringValue(row["ID"])] = row
	}
	return known
}

// checkAuthorizationDecision validates the project-owner approved fail-closed
// decision contract. ALLOW is a candidate, not runtime authority.
func (c *Catalog) checkAuthorizationDecision(v Validator) error {
	return c.validate(v, c.AuthorizationDecisionShape, c.AuthorizationDecision, AuthorizationDecisionDataPath)
}

func (c *Catalog) checkAuthorizationRegistry(v Validator) error {
	if err := c.validate(v, c.AuthorizationShape, c.Authorizations, AuthorizationDataPath); err != nil {
		return err
	}
	knownConditions := map[string]bool{}
	for _, raw := range c.Definitions["condition_list"].([]any) {
		knownConditions[stringValue(object(raw)["name"])] = true
	}
	seen := map[string]bool{}
	for _, raw := range c.Authorizations["authorizations"].([]any) {
		row := object(raw)
		id := stringValue(row["ID"])
		if id == "" || seen[id] {
			return fmt.Errorf("DUPLICATE_OR_EMPTY_AUTHORIZATION_ID: %s", id)
		}
		seen[id] = true
		if !knownConditions[stringValue(row["condition_name"])] {
			return fmt.Errorf("UNRESOLVED_AUTHORIZATION_CONDITION: %s", id)
		}
	}
	return nil
}

// Resolves the five required fields only for a registered, responsibility-linked
// authorization rule. A lookup is not an execution grant, even for ALLOW.
func (c *Catalog) LookupAuthorizationIntention(v Validator, id string) (Document, error) {
	if err := c.checkAuthorizationRegistry(v); err != nil {
		return nil, err
	}
	if err := c.validate(v, Document{"$ref": "urn:ptsip:rbac:lifecycle:index:v1#/$defs/AuthorizationID"}, id, "Authorization_ID"); err != nil {
		return nil, err
	}
	rule, exists := c.authorizationIDs()[id]
	if !exists {
		return nil, fmt.Errorf("UNRESOLVED_AUTHORIZATION_INTENTION: %s", id)
	}
	profile, err := ReadDocument(c.Root, ProfilePath)
	if err != nil {
		return nil, err
	}
	if err := c.CheckProfile(v, profile); err != nil {
		return nil, err
	}
	for rid, raw := range object(profile["component_bindings"]) {
		binding := object(raw)
		if stringValue(binding["Authorization_ID"]) == id {
			if stringValue(binding["condition_name"]) != stringValue(rule["condition_name"]) {
				return nil, fmt.Errorf("AUTHORIZATION_CONDITION_MISMATCH: %s", id)
			}
			return Document{
				"ID": id, "resource": rule["resource"], "action": rule["action"],
				"effect": rule["effect"], "condition_name": rule["condition_name"],
				"responsibility_id": rid, "execution_authorized": false,
			}, nil
		}
	}
	return nil, fmt.Errorf("UNBOUND_AUTHORIZATION_INTENTION: %s", id)
}

func keys(doc Document) []string {
	result := make([]string, 0, len(doc))
	for key := range doc {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// ValidatePolicySchema delegates v2 validation to the registered catalog resources.
func (c *Catalog) ValidatePolicySchema(v Validator, doc Document) error {
	return c.validate(v, c.Common, doc, "policy document")
}
