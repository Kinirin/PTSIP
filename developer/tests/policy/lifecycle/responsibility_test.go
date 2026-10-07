package lifecycle_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func responsibilityPayload(t *testing.T, repo *testrepo.Repository) object {
	t.Helper()
	payload, err := repo.Read(lifecycle.PolicyAnalysisRecordRoot + fixtureAnalysis + ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func responsibilityParts(payload object) (object, object) {
	analysis := payload["analysis"].(object)
	return analysis["responsibilities"].([]any)[0].(object), analysis["decision"].(object)
}

func requireResponsibilityError(t *testing.T, errors []string, expected string) {
	t.Helper()
	for _, message := range errors {
		if strings.Contains(message, expected) {
			return
		}
	}
	t.Fatalf("expected %q, got %v", expected, errors)
}

func TestMultipleOwnedFamiliesRequireExplicitSplit(t *testing.T) {
	repo := identityFixture(t)
	payload := responsibilityPayload(t, repo)
	first, decision := responsibilityParts(payload)
	second := lifecycle.Clone(first)
	second["responsibility_id"], second["family"], second["target_group_id"] = "R02", "GOV", "G02"
	second["existing_authority_lookup"] = object{"searched_policy_class": lifecycle.DeveloperClass, "searched_family": "GOV", "searched_policy_ids": []any{}, "lookup_outcome": "NO_MATCH", "candidate_comparisons": []any{}}
	payload["analysis"].(object)["responsibilities"] = []any{first, second}
	decision["owned_authority_family_set"] = []any{object{"policy_class": lifecycle.DeveloperClass, "family": "NORM"}, object{"policy_class": lifecycle.DeveloperClass, "family": "GOV"}}
	groups := decision["materialization_groups"].([]any)
	secondGroup := lifecycle.Clone(groups[0].(object))
	secondGroup["group_id"], secondGroup["family"], secondGroup["responsibility_ids"] = "G02", "GOV", []any{"R02"}
	decision["materialization_groups"] = append(groups, secondGroup)
	requireResponsibilityError(t, lifecycle.ValidateAnalysisSemantics(repo, payload), "decision.split_required must be true")
	decision["split_required"] = true
	if errors := lifecycle.ValidateAnalysisSemantics(repo, payload); len(errors) != 0 {
		t.Fatal(errors)
	}
}

func TestExactDuplicateReferencesExistingAuthorityWithoutNewGroup(t *testing.T) {
	repo := identityFixture(t)
	payload := responsibilityPayload(t, repo)
	row, decision := responsibilityParts(payload)
	comparison := row["existing_authority_lookup"].(object)["candidate_comparisons"].([]any)[0].(object)
	comparison["scope_relation"], comparison["collision_class"], comparison["resolution_action"] = "SAME_SCOPE", "EXACT_DUPLICATE", "REFERENCE_EXISTING"
	row["materialization_action"], row["target_group_id"] = "USE_EXISTING_AUTHORITY", nil
	decision["materialization_groups"] = []any{}
	if errors := lifecycle.ValidateAnalysisSemantics(repo, payload); len(errors) != 0 {
		t.Fatal(errors)
	}
}

func TestConflictCannotBeBypassedByCreatingSiblingPolicy(t *testing.T) {
	repo := identityFixture(t)
	payload := responsibilityPayload(t, repo)
	row, _ := responsibilityParts(payload)
	comparison := row["existing_authority_lookup"].(object)["candidate_comparisons"].([]any)[0].(object)
	comparison["scope_relation"], comparison["collision_class"], comparison["resolution_action"] = "SAME_SCOPE", "CONFLICT", "CREATE_NEW_SIBLING_POLICY"
	requireResponsibilityError(t, lifecycle.ValidateAnalysisSemantics(repo, payload), "CONFLICT does not allow CREATE_NEW_SIBLING_POLICY")
}

func TestAnalysisRegistryResolvesExactSubjectKeyWithoutFilenameInference(t *testing.T) {
	repo := identityFixture(t)
	result, err := lifecycle.ResolveAnalysisRecord(repo, "", "POLICY", "MPD-NORM-0014", "TEST_ROOT_MATERIALIZATION")
	if err != nil || result["analysis_id"] != fixtureAnalysis || result["analysis_ref"] != lifecycle.PolicyAnalysisRecordRoot+fixtureAnalysis+".yaml" {
		t.Fatalf("%#v %v", result, err)
	}
}

func TestLegacyFamilyCannotAllocateNewDeveloperAuthority(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0014", "DRAFT")
	_, err := lifecycle.PreflightFamilyPolicy(repo, lifecycle.DeveloperClass, "SPEC", fixtureApproval, fixtureAnalysis, "G01")
	lifecycleCode(t, err, "LEGACY_FAMILY_NEW_ALLOCATION_FORBIDDEN")
}

func TestRootFamilyRegistrationAtomicallyBindsResponsibilityAnalysis(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0014", "DRAFT")
	preflight, err := lifecycle.PreflightFamilyPolicy(repo, lifecycle.DeveloperClass, "NORM", fixtureApproval, fixtureAnalysis, "G01")
	if err != nil || preflight["allocated_policy_id"] != "MPD-NORM-0014" {
		t.Fatalf("%#v %v", preflight, err)
	}
	path := "developer/policy/NORM/MPD-NORM-0014.yaml"
	testrepo.Write(t, repo, path, rootFixturePolicy(t, "MPD-NORM-0014", "0.0", "DRAFT"))
	result, err := lifecycle.RegisterFamilyPolicy(repo, lifecycle.DeveloperClass, "NORM", fixtureApproval, fixtureAnalysis, "G01", path)
	if err != nil || result["status"] != "REGISTERED" || result["analysis_id"] != fixtureAnalysis {
		t.Fatalf("%#v %v", result, err)
	}
	registry, err := repo.Read(lifecycle.PolicyAnalysisRegistry)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{object{"policy_id": "MPD-NORM-0014", "policy_class": lifecycle.DeveloperClass, "family": "NORM", "analysis_id": fixtureAnalysis, "group_id": "G01"}}
	if !reflect.DeepEqual(registry["bindings"], want) {
		t.Fatal(registry["bindings"])
	}
}
