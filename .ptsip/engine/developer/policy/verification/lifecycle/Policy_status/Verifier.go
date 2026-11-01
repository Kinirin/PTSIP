package policystatus

import (
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
)

// Check validates declared combined status constants. It does not admit a
// lifecycle transition or interpret a description as execution permission.
func Check(root string, instance any, validate authoring.Validator) error {
	resources, err := contractcheck.Load(root,
		contractcheck.Source{Path: ".ptsip/contracts/policy/Policy_lifecycle/Public_Policy_Status.schema.json", ID: "urn:ptsip:policy:Status:v2"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/Policy_lifecycle/status/Policy_status.json", ID: "urn:ptsip:candidate:contract:policy:status:v1"},
	)
	if err != nil {
		return err
	}
	return contractcheck.Validate(validate, resources[0].(map[string]any), instance, resources)
}
