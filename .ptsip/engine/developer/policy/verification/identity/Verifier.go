package identity

import (
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
)

func Check(root string, instance any, validate authoring.Validator) error {
	resources, err := contractcheck.Load(root,
		contractcheck.Source{Path: ".ptsip/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json", ID: "urn:ptsip:policy:identity:v2"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/identity/policy_kind.json", ID: "urn:ptsip:identity:policy-kind:v1"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/identity/root_family.json", ID: "urn:ptsip:identity:root-family:v1"},
	)
	if err != nil {
		return err
	}
	return contractcheck.Validate(validate, resources[0].(map[string]any), instance, resources)
}
