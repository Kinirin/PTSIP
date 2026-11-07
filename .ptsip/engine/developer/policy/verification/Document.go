package schema_check

import (
	"fmt"
	"path/filepath"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/identity"
	planstatus "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/lifecycle/Plan_status"
	policystatus "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/lifecycle/Policy_status"
	v2 "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/v2"
)

func CheckDocument(root, selected string, document authoring.Document, validate authoring.Validator) error {
	if authoring.IsUnifiedPolicy(document) && (selected == "policy-check" || selected == "policy-identity-check" || selected == "policy-status-check") {
		_, err := authoring.CheckUnifiedDocument(root, document)
		return err
	}
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
		return policystatus.Check(root, component("Policy_Lifecycle"), validate)
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
	if selected == "policy-check" {
		selection, err := authoring.SelectPolicyDocument(absolute, document)
		if err != nil {
			return reject(err)
		}
		report["selection"] = selection
	}
	if authoring.IsUnifiedPolicy(document) && selected != "plan-status-check" {
		candidate, err := authoring.CheckUnifiedDocument(absolute, document)
		if err != nil {
			return reject(err)
		}
		for key, value := range candidate {
			report[key] = value
		}
		report["structural_state"] = "CANDIDATE"
		report["normative_approval"] = false
		report["input"] = input
		return report, 0
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
