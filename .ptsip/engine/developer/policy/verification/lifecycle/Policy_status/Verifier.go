package policystatus

import (
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
)

// Check validates declared combined status constants. It does not admit a
// lifecycle transition or interpret a description as execution permission.
func Check(root string, instance any, validate authoring.Validator) error {
	resources, err := contractcheck.Load(root,
		contractcheck.Source{Path: ".ptsip/contracts/schemas/Policy/Policy_Status/Public_Policy_Status.schema.json", ID: "urn:ptsip:policy:Status:v2"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json", ID: "urn:ptsip:contract:constant:policy_status_approved:v2"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json", ID: "urn:ptsip:contract:constant:policy_status_draft:v2"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json", ID: "urn:ptsip:contract:constant:policy_status_retired:v2"},
	)
	if err != nil {
		return err
	}
	return contractcheck.Validate(validate, resources[0].(map[string]any), instance, resources)
}
