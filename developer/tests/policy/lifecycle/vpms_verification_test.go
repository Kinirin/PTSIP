package lifecycle_test

import (
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func currentVPMSCase() object {
	return object{
		"case_id":   "CURRENT-CASE-001",
		"target":    object{"kind": "TargetRef", "ref": "CURRENT_DEVELOPER_CATALOG"},
		"formula":   object{"kind": "FormulaRef", "ref": "CURRENT_REGISTERED_CASE_SCHEMA"},
		"variables": object{"kind": "VariablesRef", "ref": "CURRENT_CASE_INPUT"},
		"policy":    object{"kind": "PolicyRef", "ref": "MPD-ASSURE-0004#current_contract_verification"},
		"runner":    object{"kind": "RunnerRef", "ref": "GO_SCHEMA_CONFORMANCE"},
	}
}

func TestCurrentVPMSGoConformanceUsesNeutralContractWithoutLegacyBodies(t *testing.T) {
	repo := discoveryFixture(t)
	result, err := lifecycle.ValidateVPMSVerificationCase(repo, currentVPMSCase())
	if err != nil || result["outcome"] != "PASS" || result["case_id"] != "CURRENT-CASE-001" || result["normative_authority"] != false || result["runtime_activation"] != "FORBIDDEN" || len(result["diagnostics"].([]any)) != 0 {
		t.Fatal(result, err)
	}
}

func TestCurrentVPMSGoConformanceRejectsInvalidReferenceDomainsAndLegacyInputs(t *testing.T) {
	for _, defect := range []string{"missing_runner", "unknown_runner", "wrong_reference_type", "withdrawn_policy", "support_policy", "legacy_source", "extra_field", "empty_case_id"} {
		t.Run(defect, func(t *testing.T) {
			input := currentVPMSCase()
			switch defect {
			case "missing_runner":
				delete(input, "runner")
			case "unknown_runner":
				input["runner"].(object)["ref"] = "UNREGISTERED_RUNNER"
			case "wrong_reference_type":
				input["target"].(object)["kind"] = "PolicyRef"
			case "withdrawn_policy":
				input["policy"].(object)["ref"] = "MPD-VERI-0007#verification_protocol_scope"
			case "support_policy":
				input["policy"].(object)["ref"] = "SFP-INFO-0001#authority_semantics"
			case "legacy_source":
				input["legacy_source_ref"] = "developer/policy/legacy/MPD-VERI-0007.yaml"
			case "extra_field":
				input["purpose"] = "UNREGISTERED_PURPOSE"
			case "empty_case_id":
				input["case_id"] = ""
			}
			result, err := lifecycle.ValidateVPMSVerificationCase(discoveryFixture(t), input)
			if err != nil || result["outcome"] != "FAIL" || result["normative_authority"] != false || result["runtime_activation"] != "FORBIDDEN" {
				t.Fatal(result, err)
			}
			diagnostics := result["diagnostics"].([]any)
			if len(diagnostics) != 1 || diagnostics[0] != "VPMS-CASE-CONTRACT-INVALID" {
				t.Fatal(diagnostics)
			}
		})
	}
}

func TestCurrentVPMSGoConformanceRejectsContractAuthorityExpansion(t *testing.T) {
	for _, field := range []string{"normative_authority", "runtime_activation", "legacy_policy_body_read", "case_schema_ref"} {
		t.Run(field, func(t *testing.T) {
			repo := discoveryFixture(t)
			ref := "developer/policy/contracts/vpms-current-verification.v1.json"
			contract, err := repo.Read(ref)
			if err != nil {
				t.Fatal(err)
			}
			if field == "normative_authority" {
				contract[field] = true
			} else {
				contract[field] = "UNREGISTERED_OR_ENABLED"
			}
			testrepo.WriteJSON(t, repo, ref, contract)
			_, err = lifecycle.ValidateVPMSVerificationCase(repo, currentVPMSCase())
			lifecycleCode(t, err, "VPMS_VERIFICATION_BINDING_UNAVAILABLE")
		})
	}
}

func TestVPMSAllocationReservesWithdrawnIdentitiesFromCurrentPRARegistry(t *testing.T) {
	repo := isolationFixture(t)
	classAnalysis(t, repo, vpmsClass, "ASSURE", "MPD-ASSURE-0002", false)
	registry, err := repo.Read(lifecycle.PolicyAnalysisRegistry)
	if err != nil {
		t.Fatal(err)
	}
	registry["bindings"] = []any{object{"policy_id": "MPD-ASSURE-0007", "analysis_id": fixtureAnalysis, "group_id": "G01", "policy_class": vpmsClass, "family": "ASSURE"}}
	testrepo.Write(t, repo, lifecycle.PolicyAnalysisRegistry, registry)
	writeApproval(t, repo, "MPD-ASSURE-0008", "DRAFT")
	result, err := lifecycle.PreflightFamilyPolicy(repo, vpmsClass, "ASSURE", fixtureApproval, fixtureAnalysis, "G01")
	if err != nil || result["allocated_policy_id"] != "MPD-ASSURE-0008" {
		t.Fatal(result, err)
	}
	writeApproval(t, repo, "MPD-ASSURE-0007", "DRAFT")
	_, err = lifecycle.PreflightFamilyPolicy(repo, vpmsClass, "ASSURE", fixtureApproval, fixtureAnalysis, "G01")
	lifecycleCode(t, err, "REQUESTED_POLICY_ID_NOT_NEXT_AVAILABLE")
}

func TestVPMSCreationRejectsLegacyFamiliesAndSchemaIdentities(t *testing.T) {
	repo := isolationFixture(t)
	for _, family := range []string{"SPEC", "PLAN", "WORK", "VERI", "MIGR", "RELS"} {
		lifecycleCode(t, lifecycle.RequireFamilyClass(repo, vpmsClass, family), "LEGACY_FAMILY_NEW_ALLOCATION_FORBIDDEN")
		for _, class := range []string{vpmsClass, lifecycle.DeveloperClass} {
			_, err := lifecycle.ActiveFamilyIDs(repo, class, family)
			lifecycleCode(t, err, "INVALID_AUTHORITY_FAMILY_KEY")
		}
	}
	legacy := object{"schema_version": "developer-policy/v2", "policy_class": vpmsClass,
		"policy": object{"id": "MPD-VERI-0009", "status": "DRAFT", "version": "0.0", "title": "invalid legacy namespace"},
		"rules":  object{"fixture": object{"enabled": true}}}
	if err := repo.Validate("developer/policy/schemas/management-policy.schema.json", legacy); err == nil {
		t.Fatal("current VPMS schema accepted a legacy family identity")
	}
}
