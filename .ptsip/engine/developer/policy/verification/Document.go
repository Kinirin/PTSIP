package schema_check

import (
	"fmt"
	"path/filepath"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	v2 "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/v2"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/identity"
	planstatus "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/lifecycle/Plan_status"
	policystatus "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/lifecycle/Policy_status"
)

func CheckDocument(root, selected string, document authoring.Document, validate authoring.Validator) error {
	component := func(name string) any {
		if value, present := document[name]; present {
			return value
		}
		return document
	}
	switch selected {
	case "policy-identity-check":
		return identity.Check(root, component("Policy_Identity"), validate)
	case "policy-status-check":
		return policystatus.Check(root, component("Policy_Status"), validate)
	case "plan-status-check":
		return planstatus.Check(root, component("Plan_Status"), validate)
	case "policy-check":
		return v2.Check(root, document, validate)
	default:
		return fmt.Errorf("UNREGISTERED_POLICY_VALIDATOR: %s", selected)
	}
}

func RunDocument(root, selected, input string) (any, int) {
	report := map[string]any{
		"status": "FAIL", "scope": "STRUCTURE_AND_DECLARED_REFERENCES",
		"validator": selected, "execution_authorized": false,
		"lifecycle_transition_authorized": false, "policy_conformance": "NOT_ASSERTED",
	}
	reject := func(err error) (any, int) { report["problem"] = err.Error(); return report, 2 }
	if input == "" {
		return reject(fmt.Errorf("EXPLICIT_INPUT_REQUIRED"))
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return reject(err)
	}
	document, err := authoring.ReadDocument(absolute, input)
	if err != nil {
		return reject(err)
	}
	validate, err := InstalledValidator()
	if err != nil {
		return reject(err)
	}
	if err := CheckDocument(absolute, selected, document, validate); err != nil {
		return reject(err)
	}
	report["status"] = "VALID"
	report["input"] = input
	return report, 0
}
