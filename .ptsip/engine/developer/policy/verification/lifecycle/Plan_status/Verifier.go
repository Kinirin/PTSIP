package planstatus

import (
	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contractcheck"
)

// Check composes only the three declared Plan_status contracts. Planning
// creation and transition authorization require their own future contracts.
func Check(root string, instance any, validate authoring.Validator) error {
	resources, err := contractcheck.Load(root,
		contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_DRAFT.json", ID: "urn:ptsip:contract:constant:plan_status_draft:v1"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_ACTIVATE.json", ID: "urn:ptsip:contract:constant:plan_status_activate:v1"},
		contractcheck.Source{Path: ".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_INTERRUPTION.json", ID: "urn:ptsip:contract:constant:plan_status_interruption:v1"},
	)
	if err != nil {
		return err
	}
	branches := make([]any, 0, len(resources))
	for _, resource := range resources {
		branches = append(branches, map[string]any{"$ref": resource.(map[string]any)["$id"]})
	}
	schema := map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": branches}
	return contractcheck.Validate(validate, schema, instance, resources)
}
