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

const publicPolicyStatusPath = ".ptsip/contracts/schemas/Policy/Policy_Status/Public_Policy_Status.schema.json"
const policyVersionPath = ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json"

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

// verifySharedStatusLifecycleRefs validates the fixed reference graph and the
// authoritative string enums in the three lifecycle source schemas. It returns
// the exact resources for the installed native VALIDATION engine to resolve.
// No in-memory enum projection or alternative schema validator is created.
func verifySharedStatusLifecycleRefs(root string, schema map[string]any) ([]any, error) {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SHARED_STATUS_PROPERTIES_MISSING")
	}
	status, ok := props["status"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SHARED_STATUS_STATUS_MISSING")
	}
	if len(status) != 1 {
		return nil, fmt.Errorf("PUBLIC_STATUS_REFERENCE_SHAPE_INVALID")
	}
	declarations, ok := status["oneOf"].([]any)
	if !ok || len(declarations) != 3 {
		return nil, fmt.Errorf("LIFECYCLE_REFERENCE_COUNT_INVALID")
	}
	sources := []struct{ path, id, prefix string }{
		{".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json", "urn:ptsip:contract:constant:policy_status_approved:v2", "APPROVED."},
		{".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json", "urn:ptsip:contract:constant:policy_status_draft:v2", "DRAFT."},
		{".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json", "urn:ptsip:contract:constant:policy_status_retired:v2", "RETIRED."},
	}
	seen := map[string]bool{}
	resources := make([]any, 0, len(sources))
	for i, source := range sources {
		ref, ok := declarations[i].(map[string]any)
		if !ok || len(ref) != 1 || ref["$ref"] != source.id+"#/properties/status" {
			return nil, fmt.Errorf("LIFECYCLE_REFERENCE_BINDING_MISMATCH: %s", source.path)
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source.path)))
		if err != nil {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_UNAVAILABLE: %s: %w", source.path, err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_JSON_INVALID: %s: %w", source.path, err)
		}
		if doc["$schema"] != "https://json-schema.org/draft/2020-12/schema" ||
			doc["$id"] != source.id || doc["schema_version"] != "ptsip-lifecycle-status-constant/v2" ||
			doc["type"] != "object" {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_ID_OR_VERSION_MISMATCH: %s", source.path)
		}
		required, ok := doc["required"].([]any)
		if !ok {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_REQUIRED_INVALID: %s", source.path)
		}
		hasStatus := false
		for _, v := range required {
			if v == "status" {
				hasStatus = true
			}
			if v == "substatus" {
				return nil, fmt.Errorf("LIFECYCLE_SOURCE_LEGACY_SUBSTATUS: %s", source.path)
			}
		}
		if !hasStatus {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_STATUS_NOT_REQUIRED: %s", source.path)
		}
		sourceProps, ok := doc["properties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_PROPERTIES_MISSING: %s", source.path)
		}
		if _, legacy := sourceProps["substatus"]; legacy {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_LEGACY_SUBSTATUS: %s", source.path)
		}
		primary, ok := sourceProps["status"].(map[string]any)
		if !ok || primary["type"] != "string" {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_STATUS_TYPE_INVALID: %s", source.path)
		}
		values, ok := primary["enum"].([]any)
		if !ok || len(values) == 0 {
			return nil, fmt.Errorf("LIFECYCLE_SOURCE_ENUM_EMPTY: %s", source.path)
		}
		for _, v := range values {
			value, ok := v.(string)
			if !ok || !strings.HasPrefix(value, source.prefix) || len(value) <= len(source.prefix) {
				return nil, fmt.Errorf("LIFECYCLE_SOURCE_ENUM_INVALID: %s", source.path)
			}
			if seen[value] {
				return nil, fmt.Errorf("LIFECYCLE_SOURCE_ENUM_DUPLICATE: %s", value)
			}
			seen[value] = true
		}
		resources = append(resources, doc)
	}
	return resources, nil
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
