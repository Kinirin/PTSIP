// Package contract_authoring generates declarations from the selected local
// contracts. JSON Schema validation is supplied by the installed PTSIP engine.
package contract_authoring

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
const IdentityVocabularyPath = ".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json"
const IdentityVocabularyID = "urn:ptsip:identity:policy:v1"
const FamilySource = IdentityVocabularyPath
const FamilyDirectory = ".ptsip/contracts/policy/Policy_Rules"
const FamilySchema = ".ptsip/contracts/policy/Policy_Rules/Contract_Types.schema.json"
const CommonSchema = ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json"
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

// IdentityValues reads one named vocabulary from the unified identity contract.
// It performs no policy approval or name-based Family inference.
func IdentityValues(document Document, name string) ([]any, error) {
	if document["$id"] != IdentityVocabularyID {
		return nil, fmt.Errorf("IDENTITY_VOCABULARY_ID_MISMATCH")
	}
	if name != "policy_kind" && name != "root_family" {
		return nil, fmt.Errorf("UNKNOWN_IDENTITY_VOCABULARY: %s", name)
	}
	definition := object(object(document["$defs"])[name])
	values, ok := definition["enum"].([]any)
	if definition["type"] != "string" || !ok || len(values) == 0 {
		return nil, fmt.Errorf("EMPTY_IDENTITY_VOCABULARY: %s", name)
	}
	seen := map[string]bool{}
	for _, value := range values {
		text := stringValue(value)
		if text == "" || seen[text] {
			return nil, fmt.Errorf("INVALID_IDENTITY_VOCABULARY: %s", name)
		}
		seen[text] = true
	}
	return values, nil
}

// Load exposes the v2 schema catalog for explicit legacy contract checks.
// Unified candidate authoring resolves its own registered entry before this
// loader. Historical Rules/ data is never an authoring fallback.
func Load(root string) (*Catalog, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	boundary, err := readJSON(absolute, BoundaryPath)
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
	c, err := LoadResponsibilities(absolute)
	if err != nil {
		return nil, err
	}
	vocabulary, err := readJSON(absolute, FamilySource)
	if err != nil {
		return nil, err
	}
	values, err := IdentityValues(vocabulary, "root_family")
	if err != nil {
		return nil, err
	}
	registry, err := readJSON(absolute, ".ptsip/contracts/policy/candidate.index.json")
	if err != nil {
		return nil, err
	}
	if registry["active_registration"] != true || registry["active_policy_cutover"] != "STRUCTURE_V2_ACTIVE" {
		return nil, fmt.Errorf("V2_POLICY_STRUCTURE_NOT_ACTIVE")
	}
	known := map[string]bool{}
	for _, raw := range c.Resources {
		known[stringValue(object(raw)["$id"])] = true
	}
	add := func(file, id string) error {
		if err := CheckPolicyContractPath(file); err != nil {
			return err
		}
		doc, err := readJSON(absolute, file)
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
	selection, err := readJSON(absolute, FamilyDirectory+"/Family_Selection.json")
	if err != nil {
		return nil, err
	}
	if selection["active"] != true || selection["selection_basis"] != "DECLARED_POLICY_IDENTITY_ROOT_FAMILY" {
		return nil, fmt.Errorf("V2_FAMILY_SELECTION_INACTIVE")
	}
	routes := object(selection["family_routes"])
	for _, raw := range values {
		family := stringValue(raw)
		selected, err := SelectPolicy(absolute, "PTSIP_DEVELOPER_POLICY", family)
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
	c.Authorizations, err = readJSON(absolute, AuthorizationDataPath)
	if err != nil {
		return nil, err
	}
	c.AuthorizationDecision, err = readJSON(absolute, AuthorizationDecisionDataPath)
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
	for _, family := range c.Families {
		// Compile exact current Family profile references. Probe validity is not
		// policy fulfillment, approval or a generation template.
		if _, err := v(c.Contracts[family], Document{}, c.Resources); err != nil {
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
