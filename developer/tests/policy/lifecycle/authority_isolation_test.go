package lifecycle_test

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const vpmsClass = "VPMS_DEVELOPER_POLICY"

func isolationFixture(t *testing.T) *testrepo.Repository {
	t.Helper()
	repo := identityFixture(t)
	testrepo.CopyFiles(t, repo, "developer/policy/schemas/management-policy.schema.json")
	testrepo.Write(t, repo, "developer/policy/ASSURE/MPD-ASSURE-0002.yaml", object{"schema_version": "developer-policy/v2", "policy_class": vpmsClass, "policy": object{"id": "MPD-ASSURE-0002", "status": "APPROVED", "substatus": "Default", "version": "2.0", "title": "VPMS fixture"}, "rules": object{"fixture": object{"enabled": true}}})
	index, err := repo.Read(lifecycle.PolicyIndex)
	if err != nil {
		t.Fatal(err)
	}
	index["policies"] = append(index["policies"].([]any), object{"id": "MPD-ASSURE-0002", "policy_class": vpmsClass, "path": "developer/policy/ASSURE/MPD-ASSURE-0002.yaml", "status": "APPROVED", "substatus": "Default"})
	sort.Slice(index["policies"].([]any), func(i, j int) bool {
		rows := index["policies"].([]any)
		return rows[i].(object)["id"].(string) < rows[j].(object)["id"].(string)
	})
	testrepo.Write(t, repo, lifecycle.PolicyIndex, index)
	subject, err := repo.Read(lifecycle.PolicySubjectRegistry)
	if err != nil {
		t.Fatal(err)
	}
	ids := []any{}
	for _, raw := range index["policies"].([]any) {
		ids = append(ids, raw.(object)["id"])
	}
	subject["subject_identity_schemes"].(object)["MANAGEMENT_POLICY_ID"].(object)["registered_values"] = ids
	testrepo.Write(t, repo, lifecycle.PolicySubjectRegistry, subject)
	return repo
}

func classAnalysis(t *testing.T, repo *testrepo.Repository, class, family, policyID string, collision bool) object {
	t.Helper()
	payload := responsibilityPayload(t, repo)
	row, decision := responsibilityParts(payload)
	row["policy_class"], row["family"] = class, family
	comparison := object{"policy_id": policyID, "section": "fixture", "scope_relation": "DISTINCT_SCOPE", "collision_class": "DISTINCT_SCOPED_AUTHORITY", "resolution_action": "COEXIST_SCOPED"}
	row["existing_authority_lookup"] = object{"searched_policy_class": class, "searched_family": family, "searched_policy_ids": []any{policyID}, "lookup_outcome": "MATCHES_FOUND", "candidate_comparisons": []any{comparison}}
	decision["owned_authority_family_set"] = []any{object{"policy_class": class, "family": family}}
	group := decision["materialization_groups"].([]any)[0].(object)
	group["policy_class"], group["family"] = class, family
	if collision {
		comparison["scope_relation"], comparison["collision_class"], comparison["resolution_action"] = "SAME_SCOPE", "EXACT_DUPLICATE", "REFERENCE_EXISTING"
		row["materialization_action"], row["target_group_id"] = "USE_EXISTING_AUTHORITY", nil
		decision["materialization_groups"] = []any{}
	}
	testrepo.Write(t, repo, lifecycle.PolicyAnalysisRecordRoot+fixtureAnalysis+".yaml", payload)
	return payload
}

func TestAuthorityFamilyLookupIsIsolatedByClassAndExcludesMigrationSources(t *testing.T) {
	repo := isolationFixture(t)
	for _, test := range []struct {
		class, family string
		expected      []string
	}{{lifecycle.DeveloperClass, "NORM", []string{"MPD-NORM-0013"}}, {vpmsClass, "ASSURE", []string{"MPD-ASSURE-0002"}}, {lifecycle.DeveloperClass, "ASSURE", []string{}}} {
		t.Run(test.class+"/"+test.family, func(t *testing.T) {
			got, err := lifecycle.ActiveFamilyIDs(repo, test.class, test.family)
			if err != nil || !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("%v %v", got, err)
			}
		})
	}
}

func TestAuthorityFamilyCompositeKeyCannotBeMissingOrUnknown(t *testing.T) {
	repo := isolationFixture(t)
	for _, test := range [][2]string{{"", "ASSURE"}, {"UNKNOWN", "ASSURE"}, {lifecycle.DeveloperClass, "BOUND"}} {
		_, err := lifecycle.ActiveFamilyIDs(repo, test[0], test[1])
		lifecycleCode(t, err, "INVALID_AUTHORITY_FAMILY_KEY")
	}
}

func TestAuthorityProjectionMetadataDriftFailsBeforeDomainFiltering(t *testing.T) {
	for _, test := range []struct{ field, value string }{{"policy_class", vpmsClass}, {"status", "DRAFT"}} {
		t.Run(test.field, func(t *testing.T) {
			repo := isolationFixture(t)
			index, err := repo.Read(lifecycle.PolicyIndex)
			if err != nil {
				t.Fatal(err)
			}
			for _, raw := range index["policies"].([]any) {
				row := raw.(object)
				if row["id"] == "MPD-NORM-0013" {
					row[test.field] = test.value
				}
			}
			testrepo.Write(t, repo, lifecycle.PolicyIndex, index)
			_, err = lifecycle.ActiveFamilyIDs(repo, lifecycle.DeveloperClass, "NORM")
			lifecycleCode(t, err, "AUTHORITY_METADATA_MISMATCH")
		})
	}
}

func TestCrossClassComparisonCannotEnterCollisionDomain(t *testing.T) {
	repo := isolationFixture(t)
	classAnalysis(t, repo, vpmsClass, "ASSURE", "MPD-NORM-0013", false)
	_, err := lifecycle.ValidateResponsibilityAnalysis(repo, fixtureAnalysis, true)
	lifecycleCode(t, err, "RESPONSIBILITY_ANALYSIS_BLOCKED")
	if !strings.Contains(err.Error(), "must exactly cover") {
		t.Fatal(err)
	}
}

func TestSameClassFamilyCollisionUsesExistingAuthority(t *testing.T) {
	for _, test := range [][3]string{{lifecycle.DeveloperClass, "NORM", "MPD-NORM-0013"}, {vpmsClass, "ASSURE", "MPD-ASSURE-0002"}} {
		t.Run(test[0], func(t *testing.T) {
			repo := isolationFixture(t)
			classAnalysis(t, repo, test[0], test[1], test[2], true)
			result, err := lifecycle.ValidateResponsibilityAnalysis(repo, fixtureAnalysis, true)
			if err != nil || result["status"] != "PASS" {
				t.Fatalf("%#v %v", result, err)
			}
		})
	}
}

func TestAuthorityClassesAreExplicitInOwnerLookupAndGroup(t *testing.T) {
	for _, test := range []string{"missing_owner_class", "lookup_class", "mixed_group_classes"} {
		t.Run(test, func(t *testing.T) {
			repo := isolationFixture(t)
			payload := responsibilityPayload(t, repo)
			row, decision := responsibilityParts(payload)
			expected := "explicit policy_class"
			switch test {
			case "missing_owner_class":
				delete(row, "policy_class")
			case "lookup_class":
				row["existing_authority_lookup"].(object)["searched_policy_class"] = vpmsClass
				expected = "searched_policy_class"
			case "mixed_group_classes":
				other := lifecycle.Clone(row)
				other["responsibility_id"], other["policy_class"] = "R02", vpmsClass
				other["existing_authority_lookup"].(object)["searched_policy_class"] = vpmsClass
				payload["analysis"].(object)["responsibilities"] = append(payload["analysis"].(object)["responsibilities"].([]any), other)
				decision["owned_authority_family_set"] = append(decision["owned_authority_family_set"].([]any), object{"policy_class": vpmsClass, "family": "NORM"})
				decision["split_required"] = true
				group := decision["materialization_groups"].([]any)[0].(object)
				group["responsibility_ids"] = []any{"R01", "R02"}
				expected = "policy_class mismatch"
			}
			requireResponsibilityError(t, lifecycle.ValidateAnalysisSemantics(repo, payload), expected)
		})
	}
}

func TestRequestedMaterializationClassMustMatchAnalysisGroup(t *testing.T) {
	repo := isolationFixture(t)
	_, _, err := lifecycle.AnalysisGroup(repo, vpmsClass, "NORM", fixtureAnalysis, "G01")
	lifecycleCode(t, err, "MATERIALIZATION_GROUP_CLASS_MISMATCH")
}

func TestFamilyAllocationIgnoresOtherNamespacesAndPreservesSharedSequence(t *testing.T) {
	for _, test := range []struct {
		family   string
		ids      []string
		expected string
	}{{"NORM", []string{"MPD-NORM-0001", "MPD-BOUND-9999", "MPD-VERI-9999"}, "MPD-NORM-0002"}, {"ASSURE", []string{"MPD-VERI-0001", "MPD-ASSURE-0002", "MPD-BOUND-9999"}, "MPD-ASSURE-0003"}} {
		got, err := lifecycle.NextFamilyID(test.ids, test.family)
		if err != nil || got != test.expected {
			t.Fatal(got, err)
		}
	}
}

func TestVPMSMaterializationRequiresCurrentNamespaceAdmission(t *testing.T) {
	repo := isolationFixture(t)
	ref := "developer/policy/registries/developer-policy-catalog-contracts.json"
	record, err := repo.Read(ref)
	if err != nil {
		t.Fatal(err)
	}
	record["vpms_family_allocation"].(object)["materialization_authorized"] = false
	testrepo.WriteJSON(t, repo, ref, record)
	lifecycleCode(t, lifecycle.RequireFamilyClass(repo, vpmsClass, "ASSURE"), "VPMS_CLASS_MATERIALIZATION_NOT_ENABLED")
}

func TestCurrentPRARegistryPreservesCreationProvenanceAndHistoricalDigestRecords(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	registry, err := repo.Read(lifecycle.PolicyAnalysisRegistry)
	if err != nil {
		t.Fatal(err)
	}
	// The owner confirmed the opaque PRA registry as the current entry. Earlier
	// path-bound digests remain immutable historical records, not live lookups.
	checked := 0
	for _, raw := range registry["records"].([]any) {
		row := raw.(object)
		id := row["analysis_id"].(string)
		if id > "PRA-0008" {
			continue
		}
		checked++
		resolved, err := lifecycle.ResolveAnalysisRecord(repo, id, "", "", "")
		if err != nil || !reflect.DeepEqual(resolved, row) {
			t.Fatalf("%#v %v", resolved, err)
		}
		result, err := lifecycle.ValidateResponsibilityAnalysis(repo, id, false)
		if err != nil || result["status"] != "PASS" {
			t.Fatalf("%s: %#v %v", id, result, err)
		}
	}
	if checked != 8 {
		t.Fatalf("expected eight registered creation records, checked %d", checked)
	}
	contract, err := repo.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	digests := contract["application_execution"].(object)["preserved_analysis_semantic_digests"].(object)
	if len(digests) != 8 {
		t.Fatal(digests)
	}
	pattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for _, digest := range digests {
		if !pattern.MatchString(digest.(string)) {
			t.Fatal("invalid historical digest", digest)
		}
	}
}

func TestCurrentVPMSPolicyIsRegisteredAsDraftWithoutRuntimeAuthority(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	inspected, err := lifecycle.InspectPolicy(repo, "MPD-ASSURE-0004")
	if err != nil || inspected["policy_status"] != "DRAFT" || inspected["policy_version"] != "0.0" || inspected["operationally_resolvable"] != false || inspected["subject_identity_registered"] != true {
		t.Fatal(inspected, err)
	}
	for _, class := range []string{vpmsClass, lifecycle.DeveloperClass} {
		ids, err := lifecycle.ActiveFamilyIDs(repo, class, "ASSURE")
		if err != nil || (class == vpmsClass && len(ids) != 0) || lifecycle.Contains(ids, "MPD-ASSURE-0004") {
			t.Fatal("candidate or withdrawn policy became active authority", ids, err)
		}
	}
}

func TestCurrentVPMSAnalysisUsesRegisteredClassAndCurrentLookup(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	record, err := lifecycle.ResolveAnalysisRecord(repo, "", "POLICY", "MPD-ASSURE-0004", "VPMS_CURRENT_CONTRACT_SUCCESSOR")
	if err != nil || record["analysis_id"] != "PRA-0018" || record["analysis_ref"] != "developer/policy/analysis/records/PRA-0018.yaml" {
		t.Fatal(record, err)
	}
	result, err := lifecycle.ValidateResponsibilityAnalysis(repo, "PRA-0018", true)
	if err != nil || result["status"] != "PASS" || !reflect.DeepEqual(result["owned_authority_family_set"], []any{object{"policy_class": vpmsClass, "family": "ASSURE"}}) || !reflect.DeepEqual(result["materialization_groups"].([]any)[0].(object)["responsibility_ids"], []any{"R01", "R02"}) {
		t.Fatal(result, err)
	}
}

func TestM8OpeningRetainsCompletedPreM8VerificationProvenance(t *testing.T) {
	contract, err := testrepo.Open(testrepo.Root(t)).Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	execution := contract["application_execution"].(object)
	evidence := execution["m1_m7_verification"].(object)
	if execution["status"] != "COMPLETE" || execution["m1_m7_verified"] != true || execution["vpms_class_materialization_enabled"] != true || evidence["snapshot_kind"] != "WORKING_TREE_ON_BASE_HEAD" || evidence["base_head"] != execution["base_head"] || evidence["return_code"] != 0 || evidence["failed"] != 0 || evidence["passed"] != 518 || evidence["skipped"] != 2 || !reflect.DeepEqual(evidence["test_modes"], []any{"repository-architecture", "ptsip-contract"}) {
		t.Fatal(execution)
	}
}

func TestClassAwareVPMSRegistrationChecksExactSourceClass(t *testing.T) {
	for _, candidateClass := range []string{vpmsClass, lifecycle.DeveloperClass} {
		t.Run(candidateClass, func(t *testing.T) {
			repo := isolationFixture(t)
			classAnalysis(t, repo, vpmsClass, "ASSURE", "MPD-ASSURE-0002", false)
			writeApproval(t, repo, "MPD-ASSURE-0003", "DRAFT")
			result, err := lifecycle.PreflightFamilyPolicy(repo, vpmsClass, "ASSURE", fixtureApproval, fixtureAnalysis, "G01")
			if err != nil || result["allocated_policy_id"] != "MPD-ASSURE-0003" || result["policy_class"] != vpmsClass {
				t.Fatalf("%#v %v", result, err)
			}
			path := "developer/policy/ASSURE/MPD-ASSURE-0003.yaml"
			testrepo.Write(t, repo, path, object{"schema_version": "developer-policy/v2", "policy_class": candidateClass, "policy": object{"id": "MPD-ASSURE-0003", "version": "0.0", "title": "VPMS fixture", "status": "DRAFT", "substatus": "Default"}, "rules": object{"fixture": object{"enabled": true}}})
			result, err = lifecycle.RegisterFamilyPolicy(repo, vpmsClass, "ASSURE", fixtureApproval, fixtureAnalysis, "G01", path)
			if candidateClass != vpmsClass {
				lifecycleCode(t, err, "POLICY_CLASS_MISMATCH")
				index, readErr := repo.Read(lifecycle.PolicyIndex)
				if readErr != nil {
					t.Fatal(readErr)
				}
				for _, raw := range index["policies"].([]any) {
					if raw.(object)["id"] == "MPD-ASSURE-0003" {
						t.Fatal("wrong class registration mutated catalog")
					}
				}
			} else if err != nil || result["status"] != "REGISTERED" {
				t.Fatalf("%#v %v", result, err)
			}
		})
	}
}
