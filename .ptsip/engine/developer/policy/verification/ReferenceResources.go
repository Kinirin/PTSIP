// Approved Policy_Status/Policy_version resources for native PTSIP VALIDATION.
// This adapter does not resolve $ref itself, infer schema authority, or depend
// on developer/automation's retired Go validator code.
package schema_check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type StatusProbe struct {
	Status                    string `json:"status"`
	SchemaPath                string `json:"schema_path"`
	PositiveValid             bool   `json:"positive_valid"`
	MissingVersionRejected    bool   `json:"missing_version_rejected"`
	LifecycleVocabularyLinked bool   `json:"lifecycle_vocabulary_linked"`
	LegacyNestedRejected      bool   `json:"legacy_nested_rejected"`
	InvalidStatusRejected     bool   `json:"invalid_status_rejected"`
	Problem                   string `json:"problem,omitempty"`
}

const publicPolicyStatusPath = ".ptsip/contracts/policy/Policy_lifecycle/Public_Policy_Status.schema.json"
const policyVersionPath = ".ptsip/contracts/policy/Policy_lifecycle/version/Policy_version.json"
const policyStatusVocabularyPath = ".ptsip/contracts/policy/Policy_lifecycle/status/Policy_status.json"
const policyStatusVocabularyID = "urn:ptsip:candidate:contract:policy:status:v1"

// approvedResource reads exactly a file already in the approved-schema set.
// It does not search arbitrary paths or fetch URIs from the network.
func approvedResource(root, relative string) (map[string]any, error) {
	allowed := false
	for _, path := range approvedSchemaPaths {
		if path == relative {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("UNAPPROVED_SCHEMA_RESOURCE: %s", relative)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return nil, fmt.Errorf("APPROVED_SCHEMA_RESOURCE_UNAVAILABLE: %s: %w", relative, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("INVALID_APPROVED_SCHEMA_RESOURCE: %s: %w", relative, err)
	}
	id, ok := doc["$id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("APPROVED_SCHEMA_RESOURCE_ID_REQUIRED: %s", relative)
	}
	return doc, nil
}

// verifySharedStatusLifecycleRefs checks the one registered vocabulary source.
// The installed VALIDATION engine resolves the original schema, not an enum projection.
func verifySharedStatusLifecycleRefs(root string, schema map[string]any) ([]any, error) {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SHARED_STATUS_PROPERTIES_MISSING")
	}
	status, ok := props["status"].(map[string]any)
	if !ok || len(status) != 1 || status["$ref"] != policyStatusVocabularyID {
		return nil, fmt.Errorf("LIFECYCLE_REFERENCE_BINDING_MISMATCH")
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(policyStatusVocabularyPath)))
	if err != nil {
		return nil, fmt.Errorf("LIFECYCLE_SOURCE_UNAVAILABLE: %w", err)
	}
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, fmt.Errorf("LIFECYCLE_SOURCE_JSON_INVALID: %w", err)
	}
	if properties, ok := source["properties"].(map[string]any); ok {
		if _, legacy := properties["substatus"]; legacy {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_LEGACY_SUBSTATUS")
		}
	}
	if source["$schema"] != "https://json-schema.org/draft/2020-12/schema" ||
		source["$id"] != policyStatusVocabularyID || source["type"] != "string" {
		return nil, fmt.Errorf("LIFECYCLE_SOURCE_ID_OR_TYPE_MISMATCH")
	}
	values, ok := source["enum"].([]any)
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("LIFECYCLE_SOURCE_ENUM_EMPTY")
	}
	families := map[string]bool{"DRAFT": false, "APPROVED": false, "RETIRED": false}
	seen := map[string]bool{}
	for _, item := range values {
		value, ok := item.(string)
		family, suffix, found := strings.Cut(value, ".")
		if _, known := families[family]; !ok || !found || suffix == "" || !known {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_ENUM_INVALID")
		}
		if seen[value] {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_ENUM_DUPLICATE: %s", value)
		}
		seen[value], families[family] = true, true
	}
	for family, present := range families {
		if !present {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_FAMILY_MISSING: %s", family)
		}
	}
	return []any{source}, nil
}

// probeApprovedFlatPolicyStatus checks the current shared status record shape
// through the installed VALIDATION engine. It never grants policy conformance.
func probeApprovedFlatPolicyStatus(root string, e Engine) StatusProbe {
	report := StatusProbe{Status: "UNRESOLVED", SchemaPath: publicPolicyStatusPath}
	fail := func(err error) StatusProbe { report.Problem = err.Error(); return report }
	schema, err := approvedResource(root, publicPolicyStatusPath)
	if err != nil {
		return fail(err)
	}
	if schema["$id"] != "urn:ptsip:policy:Status:v2" {
		return fail(fmt.Errorf("SHARED_STATUS_SCHEMA_ID_MISMATCH"))
	}
	resources, err := verifySharedStatusLifecycleRefs(root, schema)
	if err != nil {
		return fail(err)
	}
	report.LifecycleVocabularyLinked = true
	positive := map[string]any{
		"Policy_Status_Schema_ver": "2",
		"status":                   "APPROVED.Applying",
		"contract":                 "urn:ptsip:contract:policy:gov:responsibility:orchestration",
	}
	valid, err := nativeValidate(e, schema, positive, resources)
	if err != nil {
		return fail(err)
	}
	// Every source family must resolve and validate through the native engine.
	for _, value := range []string{"DRAFT.Default", "RETIRED.Waiting"} {
		instance := map[string]any{
			"Policy_Status_Schema_ver": "2",
			"status":                   value,
			"contract":                 "urn:ptsip:contract:policy:gov:responsibility:orchestration",
		}
		familyValid, familyErr := nativeValidate(e, schema, instance, resources)
		if familyErr != nil {
			return fail(familyErr)
		}
		valid = valid && familyValid
	}
	report.PositiveValid = valid

	missingVersion := map[string]any{
		"status":   "APPROVED.Applying",
		"contract": "urn:ptsip:contract:policy:gov:responsibility:orchestration",
	}
	accepted, err := nativeValidate(e, schema, missingVersion, resources)
	if err != nil {
		return fail(err)
	}
	report.MissingVersionRejected = !accepted

	legacy := map[string]any{
		"Schema": map[string]any{"Policy_Status_version": map[string]any{"version": "1"}},
		"constant": map[string]any{
			"status":    "APPROVED",
			"substatus": "Applying",
			"contract":  "urn:ptsip:contract:policy:gov:responsibility:orchestration",
		},
	}
	accepted, err = nativeValidate(e, schema, legacy, resources)
	if err != nil {
		return fail(err)
	}
	report.LegacyNestedRejected = !accepted

	invalidStatus := map[string]any{
		"Policy_Status_Schema_ver": "2",
		"status":                   "APPROVED.NotDeclared",
		"contract":                 "urn:ptsip:contract:policy:gov:responsibility:orchestration",
	}
	accepted, err = nativeValidate(e, schema, invalidStatus, resources)
	if err != nil {
		return fail(err)
	}
	report.InvalidStatusRejected = !accepted

	if !report.PositiveValid || !report.MissingVersionRejected ||
		!report.LegacyNestedRejected || !report.InvalidStatusRejected {
		return fail(fmt.Errorf("SHARED_FLAT_STATUS_SCHEMA_UNEXPECTED_VERDICT"))
	}
	report.Status = "PASS"
	return report
}
