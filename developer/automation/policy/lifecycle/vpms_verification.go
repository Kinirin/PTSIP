package lifecycle

import "github.com/Kinirin/PTSIP/developer/automation/policy/binding"

// ValidateVPMSVerificationCase consumes the registered neutral candidate contract.
// It performs Go conformance checks only and cannot activate a policy or runner.
func ValidateVPMSVerificationCase(r Repository, input Object) (Object, error) {
	catalog, err := r.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/developer-policy-catalog-contracts.schema.json", catalog); err != nil {
		return nil, err
	}
	route := Map(catalog["vpms_verification_candidate"])
	if route == nil {
		return nil, policyFailure("VPMS_VERIFICATION_BINDING_UNAVAILABLE", "candidate route is not registered")
	}
	resolver, err := binding.NewResolver(r)
	if err != nil {
		return nil, err
	}
	owner, err := resolver.Policy(Text(route["policy_ref"]))
	if err != nil {
		return nil, err
	}
	contract, err := r.Read(Text(route["contract_ref"]))
	if err != nil {
		return nil, err
	}
	rules := Map(owner["rules"])
	caseRule := Map(rules["current_contract_verification"])
	resultRule := Map(rules["candidate_result_contract"])
	if owner["policy_class"] != route["policy_class"] || Map(owner["policy"])["status"] != "DRAFT" || contract["policy_ref"] != route["policy_ref"] || contract["policy_class"] != route["policy_class"] || contract["status"] != "DRAFT" || contract["normative_authority"] != false || contract["runtime_activation"] != "FORBIDDEN" || contract["legacy_policy_body_read"] != "FORBIDDEN" || contract["legacy_source_reconstruction"] != "FORBIDDEN" || caseRule["contract_ref"] != route["contract_ref"] || caseRule["case_schema_ref"] != contract["case_schema_ref"] || resultRule["result_schema_ref"] != contract["result_schema_ref"] {
		return nil, policyFailure("VPMS_VERIFICATION_BINDING_UNAVAILABLE", "current candidate owner and contract disagree")
	}
	resultContract := Map(contract["result_contract"])
	outcome := resultContract["valid_case"]
	diagnostics := []any{}
	if err := r.Validate(Text(contract["case_schema_ref"]), input); err != nil {
		outcome = resultContract["invalid_case"]
		diagnostics = append(diagnostics, Map(resultContract["diagnostics"])["invalid_case"])
	}
	result := Object{"case_id": Text(input["case_id"]), "outcome": outcome, "diagnostics": diagnostics, "normative_authority": contract["normative_authority"], "runtime_activation": contract["runtime_activation"]}
	if err := r.Validate(Text(contract["result_schema_ref"]), result); err != nil {
		return nil, err
	}
	return result, nil
}
