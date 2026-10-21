package lifecycle

const PolicyStatusContract = ".ptsip/contracts/schemas/Policy_Status/policy_status_constant.schema.json"

// NormalizePolicyStatus is a validated identity operation only.
// It never converts DRAFT.Pass into approval or performs policy transitions.
func NormalizePolicyStatus(r Repository, state Object) (Object, error) {
	if err := r.Validate(PolicyStatusContract, state); err != nil {
		return nil, policyFailure("INVALID_POLICY_STATUS", err.Error())
	}
	return policyClone(state), nil
}
