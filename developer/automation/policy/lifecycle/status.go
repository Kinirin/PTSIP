package lifecycle

import "reflect"

const PolicyStatusContract = "developer/contracts/lifecycle/policy_status.json"

// NormalizePolicyStatus consumes the GOV contract and returns a canonical state.
// The caller owns persistence; reads never mutate policy records or create approval.
func NormalizePolicyStatus(r Repository, state Object) (Object, error) {
	if err := r.Validate(PolicyStatusContract, state); err != nil {
		return nil, policyFailure("INVALID_POLICY_STATUS", err.Error())
	}
	contract, err := r.Read(PolicyStatusContract)
	if err != nil {
		return nil, err
	}
	rule := Map(contract["automatic_normalization"])
	if Map(contract["authority"])["primary_owner_family"] != "GOV" || rule["automatic"] != true || rule["semantic_equivalence"] != true || rule["creates_new_authority"] != false {
		return nil, policyFailure("INVALID_POLICY_STATUS_CONTRACT", "GOV-owned approval-preserving normalization is required")
	}
	result := policyClone(state)
	if reflect.DeepEqual(state, Map(rule["from"])) {
		result = policyClone(Map(rule["to"]))
	}
	if err := r.Validate(PolicyStatusContract, result); err != nil {
		return nil, policyFailure("INVALID_POLICY_STATUS_NORMALIZATION", err.Error())
	}
	return result, nil
}
