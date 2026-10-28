package schema_check

import (
	"fmt"
	"path/filepath"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
	rules "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification/Rules"
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
		catalog, err := rules.Load(root, validate)
		if err != nil {
			return err
		}
		resources, err := contractcheck.Load(root,
			contractcheck.Source{Path: ".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json", ID: "urn:ptsip:policy:public-structure:v7"},
			contractcheck.Source{Path: ".ptsip/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json", ID: "urn:ptsip:policy:identity:v2"},
			contractcheck.Source{Path: ".ptsip/contracts/policy/identity/policy_kind.json", ID: "urn:ptsip:identity:policy-kind:v1"},
			contractcheck.Source{Path: ".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json", ID: "urn:ptsip:policy:Status:v2"},
			contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json", ID: "urn:ptsip:policy:Status-version:v1"},
			contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json", ID: "urn:ptsip:contract:constant:policy_status_approved:v2"},
			contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json", ID: "urn:ptsip:contract:constant:policy_status_draft:v2"},
			contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json", ID: "urn:ptsip:contract:constant:policy_status_retired:v2"},
		)
		if err != nil {
			return err
		}
		resources = append(resources, catalog.Resources...)
		if err := contractcheck.Validate(validate, resources[0].(map[string]any), document, resources); err != nil {
			return err
		}
		return catalog.CheckRules(validate, document)
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
