package lifecycle

const PolicyStatusContract = "developer/contracts/lifecycle/policy_status.json"

// NormalizePolicyStatus remains callable for legacy clients but is a validated
// identity operation. Contract behavior never creates policy approval.
func NormalizePolicyStatus(r Repository, state Object) (Object, error) {
	if err := r.Validate(PolicyStatusContract, state); err != nil {
		return nil, policyFailure("INVALID_POLICY_STATUS", err.Error())
	}
	contract, err := r.Read(PolicyStatusContract)
	if err != nil {
		return nil, err
	}
	if Map(contract["authority"])["primary_owner_family"] != "GOV" || contract["automatic_normalization"] != nil {
		return nil, policyFailure("INVALID_POLICY_STATUS_CONTRACT", "GOV status contract must not normalize approval")
	}
	return policyClone(state), nil
}
