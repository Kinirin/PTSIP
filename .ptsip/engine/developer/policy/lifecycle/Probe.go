// Package lifecycle provides a provisional proof for the approved
// Policy_Status_version structure. It is not a final policy validator.
package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	schema_check "github.com/Kinirin/PTSIP/.ptsip/engine/developer/schema_check"
)

const schemaPath = "developer/contracts/schemas/Policy_Status/Policy_Status_version.schema.json"

type CaseResult struct {
	Name          string `json:"name"`
	ExpectedValid bool   `json:"expected_valid"`
	ActualValid   bool   `json:"actual_valid"`
	Matches       bool   `json:"matches"`
}

type Report struct {
	Status            string       `json:"status"`
	Scope             string       `json:"scope"`
	EngineOperation   string       `json:"engine_operation"`
	Schema            string       `json:"schema"`
	Cases             []CaseResult `json:"cases"`
	Failed            int          `json:"failed"`
	PolicyConformance string       `json:"policy_conformance"`
	Completion        string       `json:"completion"`
	Undecided         []string     `json:"undecided"`
	Error             string       `json:"error,omitempty"`
}

type example struct {
	name     string
	document map[string]any
	valid    bool
}

func examples() []example {
	const good = "urn:ptsip:contract:policy:arch:identity:status"
	return []example{
		{"approved-shape", map[string]any{"contract": good, "version": "1.0"}, true},
		{"version-syntax-not-decided", map[string]any{"contract": good, "version": "v1.0-preview"}, true},
		{"missing-contract", map[string]any{"version": "1.0"}, false},
		{"missing-version", map[string]any{"contract": good}, false},
		{"numeric-version", map[string]any{"contract": good, "version": 1}, false},
		{"numeric-contract", map[string]any{"contract": 42, "version": "1.0"}, false},
		{"uppercase-root-family", map[string]any{"contract": "urn:ptsip:contract:policy:ARCH:identity:status", "version": "1.0"}, false},
		{"unregistered-root-family", map[string]any{"contract": "urn:ptsip:contract:policy:other:identity:status", "version": "1.0"}, false},
		{"missing-subdomain", map[string]any{"contract": "urn:ptsip:contract:policy:arch:status", "version": "1.0"}, false},
		{"uppercase-contract-name", map[string]any{"contract": "urn:ptsip:contract:policy:arch:identity:Status", "version": "1.0"}, false},
		{"underscore-subdomain", map[string]any{"contract": "urn:ptsip:contract:policy:arch:policy_identity:status", "version": "1.0"}, false},
		{"wrong-contract-kind", map[string]any{"contract": "urn:ptsip:contract:plan:arch:identity:status", "version": "1.0"}, false},
	}
}

func newReport() Report {
	return Report{
		Status: "UNRESOLVED", Scope: "APPROVED_POLICY_STATUS_VERSION_FIELDS_ONLY",
		EngineOperation: "validation.schema", Schema: schemaPath,
		Cases: []CaseResult{}, PolicyConformance: "NOT_ASSERTED",
		Completion: "PROVISIONAL_NOT_FINAL",
		Undecided: []string{
			"version-pattern-and-comparison", "urn-namespace-final-identifier",
			"root-family-subdomain-vocabulary", "urn-hyphen-boundaries",
			"unknown-properties-policy", "urn-registration-uniqueness-integration",
		},
	}
}

// Run delegates instance checks to the installed official engine. No legacy
// lifecycle validator or fallback algorithm is used.
func Run(root string) (Report, int) {
	return runWith(root, schema_check.ValidateInstance)
}

func runWith(root string, validate func(any, any) (bool, error)) (Report, int) {
	result := newReport()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(schemaPath)))
	if err != nil {
		result.Error = fmt.Sprintf("schema unavailable: %v", err)
		return result, 2
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		result.Error = fmt.Sprintf("schema is not JSON: %v", err)
		return result, 2
	}
	for _, candidate := range examples() {
		actual, err := validate(schema, candidate.document)
		if err != nil {
			result.Error = fmt.Sprintf("installed engine could not verify %s: %v", candidate.name, err)
			return result, 2
		}
		same := actual == candidate.valid
		result.Cases = append(result.Cases, CaseResult{
			Name: candidate.name, ExpectedValid: candidate.valid,
			ActualValid: actual, Matches: same,
		})
		if !same {
			result.Failed++
		}
	}
	if result.Failed != 0 {
		result.Status = "FAIL"
		return result, 1
	}
	result.Status = "PROVISIONAL_CASES_PASS"
	return result, 0
}
