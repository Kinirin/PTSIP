package source

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	index               = ".ptsip/contracts/policy/unified/index.json"
	schemaPath          = ".ptsip/contracts/policy/unified/Unified_Policy.schema.json"
	schemaID            = "urn:ptsip:candidate:policy:unified:v3"
	evaluationRegistry  = ".ptsip/contracts/policy/unified/evaluation-registry.json"
	evaluationInterface = ".ptsip/contracts/policy/unified/Evaluation_Interface.json"
	authoringBasis      = ".ptsip/contracts/policy/unified/authoring-decision.json"
)

var gitRevision = regexp.MustCompile(`^git:[0-9a-f]{40}$`)

// Verifier is an explicit repository-tooling boundary for the existing unified
// candidate. Customer URN lookup never uses its repository resources.
type Verifier struct {
	root   string
	schema *jsonschema.Schema
}

func NewVerifier(root string) (*Verifier, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	entry, err := resource(root, ".ptsip/index.json")
	if err != nil {
		return nil, err
	}
	namespace := Object(Object(entry["namespaces"])["contracts"])
	if namespace["status"] != "ACTIVE" || namespace["index"] != "contracts/index.json" {
		return nil, fmt.Errorf("CONTRACT_NAMESPACE_UNRESOLVED")
	}
	catalog, err := resource(root, ".ptsip/contracts/index.json")
	if err != nil {
		return nil, err
	}
	boundary, err := resource(root, ".ptsip/contracts/"+String(catalog["policy"]))
	if err != nil {
		return nil, err
	}
	if boundary["domain"] != "policy" || Object(boundary["unified_binding"])["index"] != index {
		return nil, fmt.Errorf("UNIFIED_POLICY_ROUTE_UNRESOLVED")
	}
	reg, err := resource(root, index)
	if err != nil {
		return nil, err
	}
	if !has(reg, Document{
		"schema_version": "ptsip-unified-policy-registration/v1", "document_schema_version": Version,
		"status": "CANDIDATE", "schema_ref": schemaPath, "schema_id": schemaID,
		"evaluation_registry_ref": evaluationRegistry, "evaluation_interface_ref": evaluationInterface,
		"authoring_basis_ref": authoringBasis, "execution_authorized": false, "publication_authorized": false,
		"structural_authoring_registered": true, "normative_activation": false,
		"policy_kinds": []any{"PTSIP_DEVELOPER_POLICY", "PTSIP_SUPPORT_FEATURE"},
	}) {
		return nil, fmt.Errorf("UNIFIED_POLICY_REGISTRATION_MISMATCH")
	}
	if _, err := resource(root, authoringBasis); err != nil {
		return nil, err
	}
	iface, err := resource(root, evaluationInterface)
	if err != nil {
		return nil, err
	}
	if !has(iface, Document{"status": "CANDIDATE", "registered_engine": false, "runtime_activation_authorized": false, "execution_authorized": false}) {
		return nil, fmt.Errorf("CANDIDATE_EVALUATION_INTERFACE_CHANGED")
	}
	evaluation, err := resource(root, evaluationRegistry)
	if err != nil {
		return nil, err
	}
	if !has(evaluation, Document{"status": "UNREGISTERED", "entries": []any{}, "execution_authorized": false, "evaluation_authorized": false}) {
		return nil, fmt.Errorf("CANDIDATE_EVALUATION_REGISTRATION_CHANGED")
	}
	schema, err := resource(root, schemaPath)
	if err != nil {
		return nil, err
	}
	if schema["$id"] != schemaID {
		return nil, fmt.Errorf("UNIFIED_POLICY_SCHEMA_ID_MISMATCH")
	}
	if !localReferences(schema) {
		return nil, fmt.Errorf("UNREGISTERED_UNIFIED_SCHEMA_REFERENCE")
	}
	compiled, err := validator.Compile(resolver.Graph{Root: schemaID, Docs: map[string]map[string]any{schemaID: schema}})
	if err != nil {
		return nil, err
	}
	return &Verifier{root: root, schema: compiled}, nil
}

func has(document, expected Document) bool {
	for key, value := range expected {
		if !reflect.DeepEqual(document[key], value) {
			return false
		}
	}
	return true
}

func localReferences(value any) bool {
	switch node := value.(type) {
	case map[string]any:
		if ref, exists := node["$ref"]; exists && !strings.HasPrefix(String(ref), "#/") {
			return false
		}
		if _, exists := node["$dynamicRef"]; exists {
			return false
		}
		for _, child := range node {
			if !localReferences(child) {
				return false
			}
		}
	case []any:
		for _, child := range node {
			if !localReferences(child) {
				return false
			}
		}
	}
	return true
}

func resource(root, relative string) (Document, error) {
	if !Relative(relative) {
		return nil, fmt.Errorf("INVALID_LOCAL_RESOURCE_PATH: %s", relative)
	}
	current := root
	for _, part := range strings.Split(relative, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("POLICY_RESOURCE_UNAVAILABLE: %s", relative)
		}
	}
	resolved, err := filepath.EvalSymlinks(current)
	if err != nil || !strings.EqualFold(resolved, current) {
		return nil, fmt.Errorf("ESCAPING_POLICY_RESOURCE: %s", relative)
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	return Decode(raw)
}

func (v *Verifier) Check(doc Document) (Document, Document, error) {
	if err := v.schema.Validate(doc); err != nil {
		return nil, nil, fmt.Errorf("UNIFIED_SCHEMA_REJECTED: %w", err)
	}
	original, err := Restore(doc)
	if err != nil {
		return nil, nil, err
	}
	id := Object(doc["Policy_Identity"])
	if !reflect.DeepEqual(id, Identity(original)) {
		return nil, nil, fmt.Errorf("SOURCE_POLICY_IDENTITY_MISMATCH")
	}
	status := Status(original)
	lifecycle := Object(doc["Policy_Lifecycle"])
	if lifecycle["status"] != status {
		return nil, nil, fmt.Errorf("SOURCE_POLICY_STATUS_MISMATCH")
	}
	content := Object(doc["Policy_Content"])
	file := Object(content["source_document"])
	location := String(file["path"])
	root := "src/policy/"
	if id["policy_kind"] == "PTSIP_DEVELOPER_POLICY" {
		root = "developer/policy/"
	}
	parts := strings.Split(location, "/")
	if !Relative(location) || !strings.HasPrefix(location, root) || len(parts) < 4 || parts[2] != id["root_family"] {
		return nil, nil, fmt.Errorf("POLICY_SOURCE_OUTSIDE_DECLARED_PLANE_OR_FAMILY")
	}
	items := map[string]Document{}
	for _, raw := range Array(content["items"]) {
		item := Object(raw)
		items[String(item["id"])] = item
	}
	value := items[String(lifecycle["source_ref"])]["value"]
	if !reflect.DeepEqual(Status(value), status) && !reflect.DeepEqual(value, status) {
		return nil, nil, fmt.Errorf("LIFECYCLE_SOURCE_ITEM_MISMATCH")
	}
	for _, key := range []string{"Policy_Authority", "Policy_Evaluation", "Policy_Relationships"} {
		if !knownRefs(items, Array(Object(doc[key])["source_refs"])) {
			return nil, nil, fmt.Errorf("UNRESOLVED_POLICY_SOURCE_ITEM: %s", key)
		}
	}
	rules := Array(Object(doc["Policy_Rules"])["rules"])
	declared, err := v.checkRules(id, items, rules, original)
	if err != nil {
		return nil, nil, err
	}
	return Document{
		"status": "VALID", "scope": "STRUCTURE_SOURCE_INTEGRITY_AND_DECLARED_REFERENCES",
		"policy_conformance": "NOT_ASSERTED", "policy_id": id["id"], "source_sha256": file["sha256"],
		"source_items": len(items), "rule_count": len(rules), "mutation_authorized": false,
		"evaluation_authorized": false, "execution_authorized": false, "publication_authorized": false,
		"evaluation_registration": Document{"status": "UNREGISTERED", "entry_count": 0, "declared_bindings": declared},
	}, original, nil
}

func knownRefs(items map[string]Document, refs []any) bool {
	for _, ref := range refs {
		if items[String(ref)] == nil {
			return false
		}
	}
	return true
}

func (v *Verifier) checkRules(id Document, items map[string]Document, rules []any, original Document) (int, error) {
	units, err := RuleUnits(original)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	declared := 0
	for _, raw := range rules {
		rule := Object(raw)
		ruleID := String(rule["id"])
		if seen[ruleID] {
			return 0, fmt.Errorf("DUPLICATE_POLICY_RULE_ID: %s", ruleID)
		}
		seen[ruleID] = true
		item := items[String(rule["source_ref"])]
		address := String(item["source_pointer"])
		unit, exists := units[address]
		if !exists || item == nil {
			return 0, fmt.Errorf("UNREGISTERED_OR_DUPLICATE_RULE_SOURCE: %s", address)
		}
		delete(units, address)
		if rule["kind"] != unit.Kind || !reflect.DeepEqual(item["value"], unit.Value) {
			return 0, fmt.Errorf("POLICY_RULE_SOURCE_UNIT_MISMATCH: %s", address)
		}
		expected, scheme := String(id["id"])+"#"+address, "SOURCE_COORDINATE"
		for _, key := range []string{"id", "ID", "requirement_id"} {
			if raw, exists := Object(unit.Value)[key]; exists {
				expected, scheme = String(raw), "DECLARED"
				if expected == "" {
					return 0, fmt.Errorf("INVALID_SOURCE_RULE_ID: %s", address)
				}
				break
			}
		}
		if ruleID != expected || rule["identity_scheme"] != scheme {
			return 0, fmt.Errorf("POLICY_RULE_ID_SOURCE_MISMATCH: %s", address)
		}
		bindings := Object(rule["contract_bindings"])
		for _, kind := range []string{"RA", "RAC", "AEC"} {
			binding := Object(bindings[kind])
			if !knownRefs(items, Array(binding["basis_refs"])) {
				return 0, fmt.Errorf("UNRESOLVED_POLICY_SOURCE_ITEM: %s", kind)
			}
			if ref, exists := binding["contract_ref"]; exists {
				if err := v.gitReference(Object(ref)); err != nil {
					return 0, err
				}
			}
			if binding["state"] == "DECLARED" {
				declared++
			}
		}
	}
	if len(units) != 0 {
		return 0, fmt.Errorf("INCOMPLETE_POLICY_RULE_COVERAGE")
	}
	return declared, nil
}

func (v *Verifier) gitReference(ref Document) error {
	location := String(ref["source_ref"])
	if !Relative(location) {
		return fmt.Errorf("INVALID_LOCAL_RESOURCE_PATH: %s", location)
	}
	lower := strings.ToLower(location)
	if strings.HasPrefix(lower, ".ptsip/contracts/policy/rules/") || strings.HasPrefix(lower, ".ptsip/contracts/schemas/") || path.Base(lower) == "responsibilities_list.schema.json" {
		return fmt.Errorf("RETIRED_CONTRACT_REFERENCE_FORBIDDEN")
	}
	revision := String(ref["revision_ref"])
	if !gitRevision.MatchString(revision) {
		return fmt.Errorf("EXACT_GIT_REFERENCE_REQUIRED")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "git", "show", "--no-textconv", revision[4:]+":"+location)
	command.Dir = v.root
	raw, err := command.Output()
	if err != nil || len(raw) > MaxBytes {
		return fmt.Errorf("PINNED_CONTRACT_REFERENCE_UNAVAILABLE")
	}
	document, err := Decode(raw)
	if err != nil {
		return err
	}
	pointer := String(ref["item_ref"])
	if pointer == "#" {
		pointer = ""
	}
	value, exists := resolver.Pointer(document, pointer)
	if !exists || Object(value) == nil {
		return fmt.Errorf("DECLARED_CONTRACT_REFERENCE_MUST_BE_OBJECT")
	}
	return nil
}
