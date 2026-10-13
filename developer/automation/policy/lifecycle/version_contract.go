package lifecycle

const PolicyVersionContract = "developer/contracts/lifecycle/policy_version.json"

// ValidatePolicyLifecycleVersion joins the independent GOV status and version
// contracts without reading archived policy sources or inferring either value.
func ValidatePolicyLifecycleVersion(r Repository, identity Object) error {
	contract, err := r.Read(PolicyVersionContract)
	if err != nil {
		return err
	}
	if Map(contract["authority"])["primary_owner_family"] != "GOV" || contract["status_contract_ref"] != PolicyStatusContract || contract["definition_state"] != "DEFINED" {
		return policyFailure("INVALID_POLICY_VERSION_CONTRACT", "defined GOV version and status contracts are required")
	}
	state := Object{"status": identity["status"]}
	if sub, present := identity["substatus"]; present {
		state["substatus"] = sub
	}
	if err := r.Validate(PolicyStatusContract, state); err != nil {
		return policyFailure("INVALID_POLICY_STATUS", err.Error())
	}
	if err := r.Validate(PolicyVersionContract, Object{"status": identity["status"], "version": identity["version"]}); err != nil {
		return policyFailure("INVALID_POLICY_VERSION", err.Error())
	}
	return nil
}
