package contract_authoring

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

const unifiedPolicyBridge = `import json, sys
from pathlib import Path
request = json.loads(sys.stdin.buffer.read().decode('utf-8'))
sys.path.insert(0, str(Path(request['root']) / 'src'))
from ptsip.policy_contracts.unified import validate_document, restore_source
report = validate_document(request['root'], request['document'])
if report.get('status') == 'VALID' and request['operation'] == 'restore':
    report['source'] = restore_source(request['document'])
sys.stdout.buffer.write(json.dumps(report, ensure_ascii=False, allow_nan=False).encode('utf-8'))
`

func unifiedPolicyOperation(root string, doc Document, operation string) (Document, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := SelectPolicyDocument(absolute, doc); err != nil {
		return nil, err
	}
	if _, err := LocalFile(absolute, "src/ptsip/policy_contracts/unified.py"); err != nil {
		return nil, err
	}
	request, err := json.Marshal(Document{"root": absolute, "document": doc, "operation": operation})
	if err != nil {
		return nil, err
	}
	encoded, err := pythonSerialization(unifiedPolicyBridge, request)
	if err != nil {
		return nil, err
	}
	var report Document
	if err := json.Unmarshal(encoded, &report); err != nil {
		return nil, err
	}
	if report["status"] != "VALID" {
		return nil, fmt.Errorf("UNIFIED_POLICY_REJECTED: %v", report["problem"])
	}
	if report["execution_authorized"] != false || report["mutation_authorized"] != false {
		return nil, fmt.Errorf("UNIFIED_POLICY_AUTHORITY_BOUNDARY_CHANGED")
	}
	return report, nil
}

// CheckUnifiedDocument calls the shared candidate verifier. Structural validity
// does not approve semantics, lifecycle transitions, registration or execution.
func CheckUnifiedDocument(root string, doc Document) (Document, error) {
	return unifiedPolicyOperation(root, doc, "validate")
}

// RestoreUnifiedSource is an explicit read-only historical view for consumers
// testing original contracts. ReadDocument always returns the canonical document.
func RestoreUnifiedSource(root string, doc Document) (Document, error) {
	if !IsUnifiedPolicy(doc) {
		return clone(doc), nil
	}
	report, err := unifiedPolicyOperation(root, doc, "restore")
	if err != nil {
		return nil, err
	}
	source := object(report["source"])
	if source == nil {
		return nil, fmt.Errorf("UNIFIED_POLICY_SOURCE_VIEW_MISSING")
	}
	return source, nil
}

func clone(input Document) Document {
	raw, _ := json.Marshal(input)
	var result Document
	_ = json.Unmarshal(raw, &result)
	if result == nil {
		result = Document{}
	}
	return result
}

func (c *Catalog) selectedFamily(doc Document, requested string) (string, error) {
	identity, present := doc["Policy_Identity"]
	if present && object(identity) == nil {
		return "", fmt.Errorf("POLICY_IDENTITY_MUST_BE_OBJECT")
	}
	declared, declaredPresent := object(identity)["root_family"]
	if declaredPresent {
		value, ok := declared.(string)
		if !ok || value == "" {
			return "", fmt.Errorf("ROOT_FAMILY_MUST_BE_NONEMPTY_STRING")
		}
		if requested != "" && requested != value {
			return "", fmt.Errorf("ROOT_FAMILY_CONFLICT: %s/%s", requested, value)
		}
		requested = value
	}
	if requested != "" && c.Contracts[requested] == nil {
		return "", fmt.Errorf("UNKNOWN_ROOT_FAMILY: %s", requested)
	}
	return requested, nil
}

func (c *Catalog) CheckRules(v Validator, doc Document) error {
	if IsUnifiedPolicy(doc) {
		_, err := CheckUnifiedDocument(c.Root, doc)
		return err
	}
	family, err := c.selectedFamily(doc, "")
	if err != nil {
		return err
	}
	if family == "" {
		return fmt.Errorf("DECLARED_ROOT_FAMILY_REQUIRED")
	}
	if c.Common == nil {
		return fmt.Errorf("REGISTERED_POLICY_SCHEMA_REQUIRED")
	}
	selection, err := SelectPolicyDocument(c.Root, doc)
	if err != nil {
		return err
	}
	if c.Common["$id"] != selection.SchemaIDs["public"] ||
		c.Contracts[family]["$ref"] != selection.ContractRefs["family_fields"] {
		return fmt.Errorf("AUTHORING_SELECTION_MISMATCH")
	}
	// The registered public schema owns the complete document and delegates
	// Family field requirements to Policy_Rules. Authoring does not reinterpret
	// retired four-block rule bindings or claim semantic approval.
	return c.validate(v, c.Common, doc, "policy document")
}

func (c *Catalog) GenerateRules(v Validator, input Document, requested string) (Document, string, error) {
	doc := clone(input)
	// An unspecified empty request has no declaration to serialize. It must not
	// create a policy, a Family selection, or empty required containers.
	if len(doc) == 0 && requested == "" {
		return doc, "OMITTED", nil
	}
	family, err := c.selectedFamily(doc, requested)
	if err != nil {
		return nil, "", err
	}
	if family == "" {
		return nil, "", fmt.Errorf("DECLARED_ROOT_FAMILY_REQUIRED")
	}
	if err := c.CheckRules(v, doc); err != nil {
		return nil, "", err
	}
	return doc, "GENERATED", nil
}

func (c *Catalog) GenerateProfile(v Validator, input Document) (Document, error) {
	base, err := ReadDocument(c.Root, ProfilePath)
	if err != nil {
		return nil, err
	}
	doc := clone(base)
	// Metadata is retained from the editable profile. An omitted responsibility
	// list generates no mappings; it never copies or invents grants.
	doc["responsibility"] = []any{}
	doc["component_bindings"] = Document{}
	for key, value := range clone(input) {
		doc[key] = value
	}
	if err := c.CheckProfile(v, doc); err != nil {
		return nil, err
	}
	return doc, nil
}
