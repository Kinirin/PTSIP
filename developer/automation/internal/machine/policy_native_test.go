package machine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func policyTestRepo(t *testing.T) *Repository {
	t.Helper()
	source, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	for _, relative := range []string{"developer/policy", "developer/planning", "developer/bindings", "src/policy", "developer/automation", "developer/tests", ".github", "docs/releasenote"} {
		root := filepath.Join(source.Root, filepath.FromSlash(relative))
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && (entry.Name() == "legacy" || entry.Name() == "__pycache__" || entry.Name() == ".cache") {
				return filepath.SkipDir
			}
			suffix, _ := filepath.Rel(source.Root, path)
			target := filepath.Join(destination, suffix)
			if entry.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(destination, "pyproject.toml"), []byte("[project]\nname='ptsip'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo, err := Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}
func policyTestWrite(t *testing.T, r *Repository, path string, value Object) {
	t.Helper()
	var err error
	if filepath.Ext(path) == ".json" {
		err = r.WriteJSON(path, value, nil)
	} else {
		err = r.WriteYAML(path, value, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
}
func policyTestAnalysis(family string) Object {
	return Object{"schema_version": "developer-policy-responsibility-analysis/v3", "artifact_class": "PTSIP_POLICY_RESPONSIBILITY_ANALYSIS",
		"analysis": Object{"analysis_id": "PRA-9999", "source_ref": "test-fixture",
			"responsibilities": []any{Object{"responsibility_id": "R01", "statement": "Fixture scoped authority", "authority_relation": "OWN", "authority_subject": "FIXTURE_SUBJECT", "lifecycle_scope": "FIXTURE_LIFECYCLE", "cohesion_key": "FIXTURE", "policy_class": DeveloperClass, "family": family, "referenced_policy_class": nil, "referenced_family": nil,
				"existing_authority_lookup": Object{"searched_policy_class": DeveloperClass, "searched_family": family, "searched_policy_ids": []any{}, "lookup_outcome": "NO_MATCH", "candidate_comparisons": []any{}}, "materialization_action": "CREATE_NEW_POLICY", "target_group_id": "G01"}},
			"decision": Object{"owned_authority_family_set": []any{Object{"policy_class": DeveloperClass, "family": family}}, "split_required": false, "materialization_allowed": true,
				"materialization_groups": []any{Object{"group_id": "G01", "policy_class": DeveloperClass, "family": family, "cohesion_key": "FIXTURE", "responsibility_ids": []any{"R01"}, "cohesion_rationale": "Single fixture scope"}}}}}
}
func policyTestApproval(id string) Object {
	return Object{"schema_version": "ptsip-policy-approval-provenance/v1", "policy_class": DeveloperClass, "approval": Object{"approval_id": "MPA-GO-NATIVE-TEST", "decision": "APPROVED", "source_kind": "PROJECT_OWNER_REPOSITORY_RECORD", "decision_source": "USER_EXPLICIT", "approval_scope": "DRAFT_CREATION", "target_status": "DRAFT", "implementation_authorized": false, "policy_content_review_scope": "FULL", "recorded_at": "2026-10-06", "requested_policy_id": id}}
}
func policyTestMaterialization(t *testing.T, r *Repository) (string, string, string, Object) {
	t.Helper()
	family := "SUPPLY"
	state, err := r.LoadConsistentPolicyCorpus()
	if err != nil {
		t.Fatal(err)
	}
	id, err := policyNextFamilyID(state.IDs, family)
	if err != nil {
		t.Fatal(err)
	}
	analysis := policyTestAnalysis(family)
	active, err := r.ActiveFamilyIDs(DeveloperClass, family)
	if err != nil {
		t.Fatal(err)
	}
	lookup := Map(Map(List(Map(analysis["analysis"])["responsibilities"])[0])["existing_authority_lookup"])
	if len(active) > 0 {
		values := []any{}
		comparisons := []any{}
		for _, owner := range active {
			values = append(values, owner)
			comparisons = append(comparisons, Object{"policy_id": owner, "section": "family_definition", "scope_relation": "DISTINCT_SCOPE", "collision_class": "DISTINCT_SCOPED_AUTHORITY", "resolution_action": "COEXIST_SCOPED"})
		}
		lookup["searched_policy_ids"] = values
		lookup["candidate_comparisons"] = comparisons
		lookup["lookup_outcome"] = "MATCHES_FOUND"
	}
	analysisID := "PRA-9999"
	analysisRef := policyAnalysisRecordRoot + analysisID + ".yaml"
	approvalRef := "developer/policy/approvals/MPA-GO-NATIVE-TEST.yaml"
	policyTestWrite(t, r, analysisRef, analysis)
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		t.Fatal(err)
	}
	registry["records"] = append(List(registry["records"]), Object{
		"analysis_id":   analysisID,
		"analysis_ref":  analysisRef,
		"subject_type":  "POLICY",
		"subject_id":    id,
		"analysis_kind": "TEST_NATIVE_MATERIALIZATION",
		"recorded_at":   "2026-10-06",
	})
	policyTestWrite(t, r, policyAnalysisRegistry, registry)
	policyTestWrite(t, r, approvalRef, policyTestApproval(id))
	template, err := r.Read("developer/policy/" + family + "/MPD-" + family + "-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	identity := Map(template["policy"])
	identity["id"] = id
	identity["version"] = "0.0"
	identity["status"] = "DRAFT"
	identity["title"] = "Native Go materialization fixture"
	template["rules"] = Object{"family_definition": Map(template["rules"])["family_definition"]}
	template["relations"] = Object{"supersedes": []any{}, "amends": []any{}, "extends": []any{}, "depends_on": []any{}}
	delete(template, "transition")
	return id, approvalRef, analysisID, template
}

func TestPolicyAnalysisCommandAdmissionUsesRegistryIdentity(t *testing.T) {
	r := policyTestRepo(t)
	if err := r.AdmitCommand(
		[]string{"policy-responsibility", "validate"},
		map[string]string{"--analysis-id": "PRA-0014"},
	); err != nil {
		t.Fatalf("registered PRA identity rejected: %v", err)
	}
	if err := r.AdmitCommand(
		[]string{"policy-responsibility", "validate"},
		map[string]string{"--analysis-ref": "developer/policy/analysis/records/PRA-0014.yaml"},
	); err == nil {
		t.Fatal("direct PRA path input was admitted")
	}

	contract, err := r.Read("developer/policy/contracts/go-automation-cutover.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(contract["runtime_commands"]) {
		command := Map(raw)
		prefix := Strings(command["command"])
		if len(prefix) != 2 || (prefix[0] != "policy-lifecycle" && prefix[0] != "policy-responsibility") {
			continue
		}
		for _, field := range []string{"required_options", "optional_options"} {
			for _, option := range Strings(command[field]) {
				if option == "analysis_ref" {
					t.Fatalf("%s %s still registers analysis_ref", prefix[0], prefix[1])
				}
			}
		}
		if prefix[0] == "policy-lifecycle" && (prefix[1] == "preflight" || prefix[1] == "register" || prefix[1] == "family-preflight" || prefix[1] == "family-register") {
			if !Has(Strings(command["required_options"]), "analysis_id") {
				t.Fatalf("%s %s does not require analysis_id", prefix[0], prefix[1])
			}
		}
	}
}

func TestPolicyVersionNativeTransitions(t *testing.T) {
	cases := []struct {
		version, status, change, target, next string
		valid                                 bool
	}{
		{"0.0", "DRAFT", "DRAFT_NORMATIVE", "", "0.1", true}, {"0.9", "DRAFT", "LIFECYCLE_TRANSITION", "APPROVED", "1.9", true}, {"1.9", "APPROVED", "LIFECYCLE_TRANSITION", "ACTIVE", "2.9", true}, {"2.8", "ACTIVE", "COMPATIBLE_NORMATIVE", "", "2.9", true}, {"2.8", "ACTIVE", "INCOMPATIBLE_NORMATIVE", "", "3.0", true}, {"2.8", "ACTIVE", "LIFECYCLE_TRANSITION", "RETIRED", "2.8", true}, {"3.2", "ACTIVE", "LIFECYCLE_TRANSITION", "SUPERSEDED", "3.2", true}, {"1.3", "APPROVED", "NON_NORMATIVE", "", "1.3", true},
		{"01.0", "DRAFT", "NON_NORMATIVE", "", "", false}, {"0.0", "DRAFT", "LIFECYCLE_TRANSITION", "ACTIVE", "", false}, {"2.0", "ACTIVE", "LIFECYCLE_TRANSITION", "APPROVED", "", false}, {"1.1", "APPROVED", "COMPATIBLE_NORMATIVE", "", "", false}, {"2.0", "ACTIVE", "DRAFT_NORMATIVE", "", "", false}, {"0.0", "DRAFT", "NON_NORMATIVE", "ACTIVE", "", false}, {"2.0", "ACTIVE", "UNREGISTERED", "", "", false},
	}
	for _, test := range cases {
		t.Run(test.status+"_"+test.change+"_"+test.target+"_"+test.version, func(t *testing.T) {
			result, err := ResolvePolicyVersionTransition(test.version, test.status, test.change, test.target)
			if test.valid {
				if err != nil || result["next_version"] != test.next {
					t.Fatalf("result=%v err=%v", result, err)
				}
			} else if err == nil {
				t.Fatalf("invalid transition admitted: %v", result)
			}
		})
	}
}
func TestPolicyResponsibilityCollisionAccounting(t *testing.T) {
	r := policyTestRepo(t)
	for collision, resolutions := range policyCollisionResolutions {
		for _, resolution := range resolutions {
			t.Run(collision+"_"+resolution, func(t *testing.T) {
				payload := policyTestAnalysis("SUPPLY")
				analysis := Map(payload["analysis"])
				unit := Map(List(analysis["responsibilities"])[0])
				decision := Map(analysis["decision"])
				lookup := Map(unit["existing_authority_lookup"])
				scope := "SAME_SCOPE"
				if collision == "DISTINCT_SCOPED_AUTHORITY" {
					scope = "DISTINCT_SCOPE"
				}
				lookup["searched_policy_ids"] = []any{"MPD-SUPPLY-0001"}
				lookup["lookup_outcome"] = "MATCHES_FOUND"
				lookup["candidate_comparisons"] = []any{Object{"policy_id": "MPD-SUPPLY-0001", "section": "family_definition", "scope_relation": scope, "collision_class": collision, "resolution_action": resolution}}
				blocker := collision == "PARTIAL_OVERLAP" || collision == "CONFLICT"
				create := resolution == "CREATE_NEW_SIBLING_POLICY" || resolution == "COEXIST_SCOPED"
				if blocker {
					unit["materialization_action"] = "BLOCKED"
					unit["target_group_id"] = nil
					decision["materialization_groups"] = []any{}
					decision["materialization_allowed"] = false
					decision["split_required"] = collision == "PARTIAL_OVERLAP"
				} else if !create {
					unit["materialization_action"] = "USE_EXISTING_AUTHORITY"
					unit["target_group_id"] = nil
					decision["materialization_groups"] = []any{}
				}
				if errors := r.ValidateAnalysisSemantics(payload); len(errors) > 0 {
					t.Fatal(errors)
				}
				Map(List(lookup["candidate_comparisons"])[0])["resolution_action"] = "UNREGISTERED_ACTION"
				if errors := r.ValidateAnalysisSemantics(payload); len(errors) == 0 {
					t.Fatal("unregistered collision resolution admitted")
				}
			})
		}
	}
	malformed := policyTestAnalysis("SUPPLY")
	Map(List(Map(malformed["analysis"])["responsibilities"])[0])["referenced_policy_class"] = "PTSIP_SUPPORT_FEATURE"
	if errors := r.ValidateAnalysisSemantics(malformed); len(errors) == 0 {
		t.Fatal("foreign class inheritance admitted")
	}
}
func TestPolicyNativeRegistrationWithoutLegacy(t *testing.T) {
	r := policyTestRepo(t)
	if _, err := os.Stat(filepath.Join(r.Root, "developer/policy/legacy")); !os.IsNotExist(err) {
		t.Fatal("fixture accidentally contains legacy policy files")
	}
	id, approval, analysisID, record := policyTestMaterialization(t, r)
	ready, err := r.PreflightNewPolicy(approval, analysisID, "G01")
	if err != nil || ready["allocated_policy_id"] != id {
		t.Fatalf("preflight=%v err=%v", ready, err)
	}
	path, _ := policyCanonicalPath(id)
	policyTestWrite(t, r, path, record)
	registered, err := r.RegisterPolicy(approval, analysisID, "G01", path)
	if err != nil || registered["status"] != "REGISTERED" {
		t.Fatalf("registered=%v err=%v", registered, err)
	}
	inspected, err := r.InspectPolicy(id)
	if err != nil || inspected["policy_status"] != "DRAFT" || inspected["policy_version"] != "0.0" {
		t.Fatalf("inspection=%v err=%v", inspected, err)
	}
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, raw := range List(registry["bindings"]) {
		if Map(raw)["policy_id"] == id {
			found = true
		}
	}
	if !found {
		t.Fatal("analysis binding was not committed with identity registration")
	}
	if _, err := NewResolver(r); err != nil {
		t.Fatal(err)
	}
}
func TestPolicyRegistrationRejectsExtraCandidateAndPreservesIndex(t *testing.T) {
	r := policyTestRepo(t)
	id, approval, analysisID, record := policyTestMaterialization(t, r)
	path, _ := policyCanonicalPath(id)
	policyTestWrite(t, r, path, record)
	extra := policyClone(record)
	Map(extra["policy"])["id"] = "MPD-SUPPLY-9999"
	policyTestWrite(t, r, "developer/policy/SUPPLY/MPD-SUPPLY-9999.yaml", extra)
	original, _ := os.ReadFile(filepath.Join(r.Root, filepath.FromSlash(policyIndex)))
	if _, err := r.RegisterPolicy(approval, analysisID, "G01", path); err == nil {
		t.Fatal("multiple unregistered candidates admitted")
	}
	after, _ := os.ReadFile(filepath.Join(r.Root, filepath.FromSlash(policyIndex)))
	if !reflect.DeepEqual(original, after) {
		t.Fatal("failed registration changed index")
	}
}
func TestPolicyAnalysisFreshnessAndApprovalBoundaries(t *testing.T) {
	r := policyTestRepo(t)
	id, approvalRef, analysisID, _ := policyTestMaterialization(t, r)
	approval, err := r.Read(approvalRef)
	if err != nil {
		t.Fatal(err)
	}
	Map(approval["approval"])["target_status"] = "ACTIVE"
	policyTestWrite(t, r, approvalRef, approval)
	if _, err := r.PreflightNewPolicy(approvalRef, analysisID, "G01"); err == nil {
		t.Fatal("ACTIVE initial policy admitted")
	}
	Map(approval["approval"])["target_status"] = "DRAFT"
	Map(approval["approval"])["requested_policy_id"] = "MPD-0051"
	policyTestWrite(t, r, approvalRef, approval)
	if _, err := r.PreflightNewPolicy(approvalRef, analysisID, "G01"); err == nil {
		t.Fatal("legacy identity allocation revived")
	}
	if _, err := r.policyApproval("../approval.yaml"); err == nil {
		t.Fatal("approval outside repository admitted")
	}
	if _, err := r.ValidateResponsibilityAnalysis(policyAnalysisRegistry, true); err == nil {
		t.Fatal("analysis registry admitted as analysis")
	}
	analysisRef := policyAnalysisRecordRoot + analysisID + ".yaml"
	payload, err := r.Read(analysisRef)
	if err != nil {
		t.Fatal(err)
	}
	Map(Map(List(Map(payload["analysis"])["responsibilities"])[0])["existing_authority_lookup"])["searched_policy_ids"] = []any{id}
	policyTestWrite(t, r, analysisRef, payload)
	if _, err := r.ValidateResponsibilityAnalysis(analysisID, true); err == nil {
		t.Fatal("stale/non-active lookup admitted")
	}
}
func TestPolicyAnalysisBackfillPreservesMeaning(t *testing.T) {
	r := policyTestRepo(t)
	original := policyTestAnalysis("SUPPLY")
	legacy, err := LegacyAnalysisProjection(original)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := PolicySemanticDigest(legacy)
	updated, err := BackfillLegacyAnalysis(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if errors := r.ValidateAnalysisSemantics(updated); len(errors) > 0 {
		t.Fatal(errors)
	}
	recovered, err := LegacyAnalysisProjection(updated)
	if err != nil || !reflect.DeepEqual(recovered, legacy) {
		t.Fatal("backfill changed prior analysis")
	}
	recoveredDigest, _ := PolicySemanticDigest(recovered)
	if digest != recoveredDigest {
		t.Fatal("semantic digest changed")
	}
	if _, err := BackfillLegacyAnalysis(updated); err == nil {
		t.Fatal("non-legacy analysis backfilled twice")
	}
}
func TestPolicyTransitionDependencyValidation(t *testing.T) {
	valid := Object{"policy": Object{"status": "ACTIVE"}, "transition": Object{"state": "COMPLETE", "requirements": []any{Object{"id": "A", "state": "SATISFIED", "next_action": Object{"after": []any{}}}, Object{"id": "B", "state": "SATISFIED", "next_action": Object{"after": []any{"A"}}}}}}
	if errors := ValidatePolicyTransitionSemantics("MPD-CTRL-0001", valid); len(errors) > 0 {
		t.Fatal(errors)
	}
	for _, kind := range []string{"cycle", "unknown", "unsatisfied", "duplicate", "state"} {
		t.Run(kind, func(t *testing.T) {
			payload := policyClone(valid)
			requirements := List(Map(payload["transition"])["requirements"])
			switch kind {
			case "cycle":
				Map(Map(requirements[0])["next_action"])["after"] = []any{"B"}
			case "unknown":
				Map(Map(requirements[1])["next_action"])["after"] = []any{"MISSING"}
			case "unsatisfied":
				Map(requirements[0])["state"] = "PENDING"
			case "duplicate":
				Map(requirements[1])["id"] = "A"
			case "state":
				Map(payload["transition"])["state"] = "READY"
			}
			if errors := ValidatePolicyTransitionSemantics("MPD-CTRL-0001", payload); len(errors) == 0 {
				t.Fatal("invalid transition history admitted")
			}
		})
	}
}
func TestCurrentPolicyValidationDoesNotRequireHistoricalUnitAccounting(t *testing.T) {
	r := policyTestRepo(t)
	if _, err := r.ValidateCurrentRootContracts(DeveloperClass); err != nil {
		t.Fatal(err)
	}
	current, err := r.Read("developer/policy/INFO/MPD-INFO-0003.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if Map(current["policy"])["status"] != "APPROVED" {
		t.Fatal("source DRAFT still constrains current approval")
	}
	delete(current, "authority_subject")
	policyTestWrite(t, r, "developer/policy/INFO/MPD-INFO-0003.yaml", current)
	if _, err := r.ValidateCurrentRootContracts(DeveloperClass); err == nil {
		t.Fatal("invalid current schema admitted")
	}
}

func TestPolicyPlanConsistencyNativeRegistry(t *testing.T) {
	r := policyTestRepo(t)
	payload, err := r.Read(BindingRegistryPath)
	if err != nil {
		t.Fatal(err)
	}
	rows := List(payload["bindings"])
	if len(rows) == 0 {
		t.Skip("no registered binding fixtures")
	}
	fixture := policyClone(Map(rows[0]))
	fixture["binding_id"] = "PPB-9999"
	fixture["policy_ref"] = "MPD-REAL-0005"
	fixture["planning_state"] = "NOT_CREATED"
	for _, field := range []string{"resolved_plan_id", "plan_file_id", "version", "revision", "plan_ref"} {
		delete(fixture, field)
	}
	payload["bindings"] = []any{fixture}
	policyTestWrite(t, r, BindingRegistryPath, payload)
	result, err := r.VerifyPolicyPlanConsistency()
	if err != nil || result["status"] != "PASS" || result["checked_binding_count"] != 1 {
		t.Fatalf("result=%v err=%v", result, err)
	}
	fixture["policy_ref"] = "MPD-0010"
	policyTestWrite(t, r, BindingRegistryPath, payload)
	result, err = r.VerifyPolicyPlanConsistency()
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "FAIL" {
		t.Fatal("legacy policy execution authority admitted")
	}
}
func TestPolicyCatalogClosedSchemaAndInventory(t *testing.T) {
	r := policyTestRepo(t)
	if _, err := r.LoadNeutralPolicyIndex(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveNeutralCatalogContract("UNREGISTERED"); err == nil {
		t.Fatal("unregistered contract admitted")
	}
	state, err := r.LoadConsistentPolicyCorpus()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.policyCheckDiscovery(state, ""); err != nil {
		t.Fatal(err)
	}
	fake := Object{"$id": "urn:unregistered:test", "$ref": "https://example.invalid/schema"}
	if err := r.policyValidateInline(fake, Object{}); err == nil {
		t.Fatal("external schema retrieval admitted")
	}
	errors := r.ValidateDeveloperPolicy()
	for _, message := range errors {
		if strings.Contains(message, "legacy") && strings.Contains(message, "does not exist") {
			t.Fatalf("validator relied on legacy policy body: %s", message)
		}
	}
}

func TestPolicyNeutralCatalogNativeMigration(t *testing.T) {
	r := policyTestRepo(t)
	index, err := r.Read(policyIndex)
	if err != nil {
		t.Fatal(err)
	}
	entries := []any{}
	ids := []any{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		if entry["policy_class"] == DeveloperClass && entry["authority_role"] != "MIGRATION_SOURCE" {
			delete(entry, "policy_class")
			entries = append(entries, entry)
			ids = append(ids, entry["id"])
		}
	}
	index["policies"] = entries
	index["schema_version"] = "ptsip-developer-policy-index/v1"
	delete(index, "artifact_class")
	index["policy_class"] = DeveloperClass
	subject, err := r.Read(policySubjectRegistry)
	if err != nil {
		t.Fatal(err)
	}
	subject["schema_version"] = "ptsip-developer-authority-subject-registry/v1"
	delete(subject, "artifact_class")
	subject["policy_class"] = DeveloperClass
	Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"] = ids
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		t.Fatal(err)
	}
	registry["bindings"] = []any{}
	policyTestWrite(t, r, policyIndex, index)
	policyTestWrite(t, r, policySubjectRegistry, subject)
	policyTestWrite(t, r, policyAnalysisRegistry, registry)
	prepared, err := r.MigrateAuthorityFamilyCatalog(false)
	if err != nil || prepared["status"] != "READY" {
		t.Fatalf("prepared=%v err=%v", prepared, err)
	}
	stillLegacy, err := r.Read(policyIndex)
	if err != nil || stillLegacy["schema_version"] != "ptsip-developer-policy-index/v1" {
		t.Fatal("read-only migration mutated source")
	}
	applied, err := r.MigrateAuthorityFamilyCatalog(true)
	if err != nil || applied["status"] != "APPLIED" {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	canonical, err := r.LoadNeutralPolicyIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(List(canonical["policies"])) != len(entries) {
		t.Fatal("migration changed catalog membership")
	}
	if _, err := r.MigrateAuthorityFamilyCatalog(true); err == nil {
		t.Fatal("migration replay was admitted")
	}
}
