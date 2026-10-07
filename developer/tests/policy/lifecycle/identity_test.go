package lifecycle_test

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const fixtureApproval = "developer/policy/approvals/MPA-test.yaml"
const fixtureAnalysis = "PRA-9999"

func identityFixture(t *testing.T) *testrepo.Repository {
	t.Helper()
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyCatalogContracts(t, repo)
	testrepo.CopyFiles(t, repo, "developer/policy/schemas/root-family-policy.schema.json", "developer/policy/schemas/policy-approval-provenance.schema.json", "developer/policy/registries/root-family-entry-registry.json", "developer/policy/contracts/go-policy-resolver.v1.yaml", "developer/policy/analysis/schemas/policy-responsibility-analysis.schema.json", "developer/policy/analysis/schemas/policy-materialization-analysis-registry.schema.json", "developer/policy/registries/authority-subject-registry.yaml")
	testrepo.CopyFiles(t, repo, "developer/policy/CNTR/MPD-CNTR-0004.yaml", "developer/policy/policy-resolver-bindings/registry.yaml", "developer/policy/policy-resolver-bindings/bindings.jsonl", "developer/policy/schemas/policy-resolver-binding.schema.json")
	index := object{"schema_version": "developer-policy-catalog/v1", "artifact_class": "DEVELOPER_POLICY_CATALOG", "policies": []any{object{"id": "MPD-CNTR-0004", "path": "developer/policy/CNTR/MPD-CNTR-0004.yaml", "status": "ACTIVE", "policy_class": lifecycle.DeveloperClass}}}
	for _, test := range []struct{ id, status, version string }{{"MPD-NORM-0012", "DRAFT", "0.0"}, {"MPD-NORM-0013", "ACTIVE", "2.0"}} {
		path, err := lifecycle.CanonicalPath(test.id)
		if err != nil {
			t.Fatal(err)
		}
		index["policies"] = append(index["policies"].([]any), object{"id": test.id, "path": path, "status": test.status, "policy_class": lifecycle.DeveloperClass})
		payload := rootFixturePolicy(t, test.id, test.version, test.status)
		testrepo.Write(t, repo, path, payload)
	}
	testrepo.Write(t, repo, lifecycle.PolicyIndex, index)
	subject, err := repo.Read(lifecycle.PolicySubjectRegistry)
	if err != nil {
		t.Fatal(err)
	}
	subject["subject_identity_schemes"].(object)["MANAGEMENT_POLICY_ID"].(object)["registered_values"] = []any{"MPD-CNTR-0004", "MPD-NORM-0012", "MPD-NORM-0013"}
	testrepo.Write(t, repo, lifecycle.PolicySubjectRegistry, subject)
	analysis := object{"schema_version": "developer-policy-responsibility-analysis/v3", "artifact_class": "PTSIP_POLICY_RESPONSIBILITY_ANALYSIS", "analysis": object{
		"analysis_id": fixtureAnalysis, "source_ref": "test-fixture", "responsibilities": []any{object{"responsibility_id": "R01", "statement": "One new normative scope", "authority_relation": "OWN", "authority_subject": "FIXTURE_SUBJECT", "lifecycle_scope": "FIXTURE_LIFECYCLE", "cohesion_key": "FIXTURE", "policy_class": lifecycle.DeveloperClass, "family": "NORM", "referenced_policy_class": nil, "referenced_family": nil,
			"existing_authority_lookup": object{"searched_policy_class": lifecycle.DeveloperClass, "searched_family": "NORM", "searched_policy_ids": []any{"MPD-NORM-0013"}, "lookup_outcome": "MATCHES_FOUND", "candidate_comparisons": []any{object{"policy_id": "MPD-NORM-0013", "section": "fixture", "scope_relation": "DISTINCT_SCOPE", "collision_class": "DISTINCT_SCOPED_AUTHORITY", "resolution_action": "COEXIST_SCOPED"}}}, "materialization_action": "CREATE_NEW_POLICY", "target_group_id": "G01"}},
		"decision": object{"owned_authority_family_set": []any{object{"policy_class": lifecycle.DeveloperClass, "family": "NORM"}}, "split_required": false, "materialization_allowed": true, "materialization_groups": []any{object{"group_id": "G01", "policy_class": lifecycle.DeveloperClass, "family": "NORM", "cohesion_key": "FIXTURE", "responsibility_ids": []any{"R01"}, "cohesion_rationale": "Single fixture scope"}}},
	}}
	analysisRef := lifecycle.PolicyAnalysisRecordRoot + fixtureAnalysis + ".yaml"
	testrepo.Write(t, repo, analysisRef, analysis)
	testrepo.Write(t, repo, lifecycle.PolicyAnalysisRegistry, object{"schema_version": "developer-policy-materialization-analysis-registry/v3", "artifact_class": "PTSIP_POLICY_MATERIALIZATION_ANALYSIS_REGISTRY", "records": []any{object{"analysis_id": fixtureAnalysis, "analysis_ref": analysisRef, "subject_type": "POLICY", "subject_id": "MPD-NORM-0014", "analysis_kind": "TEST_ROOT_MATERIALIZATION", "recorded_at": "2026-10-07"}}, "bindings": []any{}})
	return repo
}

func rootFixturePolicy(t *testing.T, id, version, status string) object {
	t.Helper()
	payload, err := testrepo.Open(testrepo.Root(t)).Read("developer/policy/NORM/MPD-NORM-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	payload["policy"] = object{"id": id, "version": version, "title": "Fixture", "status": status}
	payload["rules"] = object{"fixture": object{"enabled": true}}
	if status == "DRAFT" {
		delete(payload, "transition")
	}
	return payload
}

func writeApproval(t *testing.T, repo *testrepo.Repository, id, target string) {
	t.Helper()
	testrepo.Write(t, repo, fixtureApproval, object{"schema_version": "ptsip-policy-approval-provenance/v1", "policy_class": lifecycle.DeveloperClass, "approval": object{"approval_id": "MPA-test", "decision": "APPROVED", "source_kind": "PROJECT_OWNER_DIRECT_INSTRUCTION", "decision_source": "USER_EXPLICIT", "source_reference": "test-fixture", "approval_scope": "TEMPORARY_DIRECTION_AND_IMPLEMENTATION", "target_status": target, "implementation_authorized": true, "policy_content_review_scope": "DIRECTION", "requested_policy_id": id, "recorded_at": "2026-10-07T16:00:00+09:00"}})
}

func lifecycleCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !strings.HasPrefix(err.Error(), code) {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestInspectDraftDoesNotGrantOperationalResolution(t *testing.T) {
	repo := identityFixture(t)
	result, err := lifecycle.InspectPolicy(repo, "MPD-NORM-0012")
	if err != nil || result["status"] != "FOUND" || result["policy_status"] != "DRAFT" || result["index_status"] != "DRAFT" || result["operationally_resolvable"] != false {
		t.Fatalf("%#v %v", result, err)
	}
}

func TestInspectFailsClosedWhenAnyIndexedPolicyFileIsMissing(t *testing.T) {
	repo := identityFixture(t)
	path, err := repo.Path("developer/policy/NORM/MPD-NORM-0012.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_, err = lifecycle.InspectPolicy(repo, "MPD-NORM-0013")
	lifecycleCode(t, err, "POLICY_FILE_NOT_FOUND")
	if !strings.Contains(err.Error(), "MPD-NORM-0012") {
		t.Fatal(err)
	}
}

func TestNewPolicyPreflightRejectsAlreadyAllocatedRequestedIdentity(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0013", "DRAFT")
	_, err := lifecycle.PreflightNewPolicy(repo, fixtureApproval, fixtureAnalysis, "G01")
	lifecycleCode(t, err, "REQUESTED_POLICY_ID_NOT_NEXT_AVAILABLE")
}

func TestNewPolicyPreflightAllocatesExactFamilyAndPreservesApprovalBoundary(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0014", "DRAFT")
	result, err := lifecycle.PreflightNewPolicy(repo, fixtureApproval, fixtureAnalysis, "G01")
	if err != nil || result["allocated_policy_id"] != "MPD-NORM-0014" || result["approval_scope"] != "TEMPORARY_DIRECTION_AND_IMPLEMENTATION" || result["target_status"] != "DRAFT" || result["implementation_authorized"] != true {
		t.Fatalf("%#v %v", result, err)
	}
}

func TestNewPolicyPreflightRequiresExplicitApprovalTargetStatus(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0014", "DRAFT")
	payload, err := repo.Read(fixtureApproval)
	if err != nil {
		t.Fatal(err)
	}
	delete(payload["approval"].(object), "target_status")
	testrepo.Write(t, repo, fixtureApproval, payload)
	_, err = lifecycle.PreflightNewPolicy(repo, fixtureApproval, fixtureAnalysis, "G01")
	lifecycleCode(t, err, "INVALID_APPROVAL_PROVENANCE")
}

func TestRegistrationUpdatesCatalogAndSubjectOnlyAfterExactPolicyFile(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0014", "DRAFT")
	path := "developer/policy/NORM/MPD-NORM-0014.yaml"
	indexPath, _ := repo.Path(lifecycle.PolicyIndex)
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.RegisterPolicy(repo, fixtureApproval, fixtureAnalysis, "G01", path); err == nil {
		t.Fatal("missing policy file registered")
	}
	after, err := os.ReadFile(indexPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed registration mutated catalog", err)
	}
	testrepo.Write(t, repo, path, rootFixturePolicy(t, "MPD-NORM-0014", "0.0", "DRAFT"))
	result, err := lifecycle.RegisterPolicy(repo, fixtureApproval, fixtureAnalysis, "G01", path)
	if err != nil || result["status"] != "REGISTERED" {
		t.Fatalf("%#v %v", result, err)
	}
	index, err := repo.Read(lifecycle.PolicyIndex)
	if err != nil {
		t.Fatal(err)
	}
	ids := []any{}
	for _, raw := range index["policies"].([]any) {
		row := raw.(object)
		ids = append(ids, row["id"])
		if row["id"] == "MPD-NORM-0014" && row["status"] != "DRAFT" {
			t.Fatal(row)
		}
	}
	want := []any{"MPD-CNTR-0004", "MPD-NORM-0012", "MPD-NORM-0013", "MPD-NORM-0014"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatal(ids)
	}
	subject, err := repo.Read(lifecycle.PolicySubjectRegistry)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(subject["subject_identity_schemes"].(object)["MANAGEMENT_POLICY_ID"].(object)["registered_values"], want) {
		t.Fatal(subject)
	}
}

func TestStatusPreflightRequiresExactExistingPolicyAndApprovalIdentity(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0013", "DRAFT")
	result, err := lifecycle.StatusPreflight(repo, "MPD-NORM-0013", fixtureApproval)
	want := object{"status": "READY", "policy_id": "MPD-NORM-0013", "current_status": "ACTIVE", "target_status": "DRAFT", "approval_id": "MPA-test", "implementation_authorized": true}
	if err != nil || !reflect.DeepEqual(result, want) {
		t.Fatalf("%#v %v", result, err)
	}
	_, err = lifecycle.StatusPreflight(repo, "MPD-NORM-0012", fixtureApproval)
	lifecycleCode(t, err, "APPROVAL_POLICY_ID_MISMATCH")
}

func TestNewPolicyPreflightRequiresDraftInitialStatus(t *testing.T) {
	repo := identityFixture(t)
	writeApproval(t, repo, "MPD-NORM-0014", "ACTIVE")
	_, err := lifecycle.PreflightNewPolicy(repo, fixtureApproval, fixtureAnalysis, "G01")
	lifecycleCode(t, err, "NEW_POLICY_MUST_START_DRAFT")
}
