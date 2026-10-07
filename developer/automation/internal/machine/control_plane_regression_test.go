package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/planning"
)

func migratedSourceValue(t *testing.T, r *Repository, sourceID, pointer string) any {
	t.Helper()
	plane := "developer/policy"
	field := "rules"
	if strings.HasPrefix(sourceID, "SFP-") {
		plane, field = "src/policy", "authority_semantics"
	}
	graph, err := r.Read(plane + "/registries/root-family-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		if source["source_policy_id"] != sourceID {
			continue
		}
		for _, raw := range List(source["units"]) {
			unit := Map(raw)
			if unit["source_pointer"] != pointer {
				continue
			}
			policy, err := r.Read(plane + "/" + Text(unit["policy_path"]))
			if err != nil {
				t.Fatal(err)
			}
			value, exists := Map(policy[field])[Text(unit["section"])]
			if !exists {
				t.Fatal("Root source unit missing", unit)
			}
			return value
		}
	}
	t.Fatal("unregistered source responsibility", sourceID, pointer)
	return nil
}

func TestDeveloperControlPlanesAndDependencyGateAreMachineValid(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	if failures := r.ValidateDeveloperPolicy(); len(failures) != 0 {
		t.Fatal(failures)
	}
	if failures, err := r.ProfileRegistryErrors(); err != nil || len(failures) != 0 {
		t.Fatal(failures, err)
	}
	if failures := r.ValidatePlanning(); len(failures) != 0 {
		t.Fatal(failures)
	}
	if hits, err := r.LegacyDependencyHits(); err != nil || len(hits) != 0 {
		t.Fatal(hits, err)
	}
}

func TestLayoutContainersAreNotGovernanceConstantsAndNestedSourcesStayValidated(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := r.Read("developer/policy/registries/governance-source-registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []any{[]any{"projection.go"}, Object{"module": "projection.go"}} {
		payload := Object{"target_layout": Object{"analysis": Object{"source": source}}}
		if failures := planning.GovernanceErrors(payload, registry, "responsibility-map.yaml"); len(failures) != 0 {
			t.Fatal(failures)
		}
	}
	payload := Object{"source": []any{Object{"approval_source": "DIRECT_PROJECT_OWNER_TEMPORARY_APPROVAL"}, Object{"source": "DIRECT_PROJECT_OWNER_INSTRUCTION"}}}
	failures := planning.GovernanceErrors(payload, registry, "fixture.yaml")
	if len(failures) != 2 || !strings.Contains(failures[0], "source[0].approval_source") || !strings.Contains(failures[1], "source[1].source") {
		t.Fatal(failures)
	}
}

func TestLegacyRemovalGateTracksExactE4CompletionWithoutMutation(t *testing.T) {
	for _, status := range []string{"COMPLETE", "VALIDATION_PENDING"} {
		t.Run(status, func(t *testing.T) {
			r := policyTestRepo(t)
			payload := Object{"migration_stages": Object{"P01_E_LEGACY_REMOVAL": Object{"preauthorized_action": "REMOVE_DECISIONS_DIRECTORY_FROM_ACTIVE_TREE", "confirmation_required": false}}, "p01_e_execution_plan": Object{"execution_order": []any{Object{"id": "P01_E4_MIGRATION_ONLY_RETIREMENT_AND_GATE_SIMPLIFICATION", "status": status}}}}
			policyTestWrite(t, r, "developer/planning/0.4.0/WU-02/WU-02-P01.yaml", payload)
			result, err := r.EvaluateLegacyRemoval()
			if err != nil || result["confirmation_required"] != false {
				t.Fatal(result, err)
			}
			if status == "COMPLETE" {
				if result["state"] != "AUTHORIZED" || result["action"] != "REMOVE_DECISIONS_DIRECTORY_FROM_ACTIVE_TREE" || len(Strings(result["blockers"])) != 0 {
					t.Fatal(result)
				}
			} else if result["state"] != "HOLD_NOT_AUTHORIZED" || result["action"] != nil || !Has(Strings(result["blockers"]), "P01_E4_VALIDATION_NOT_COMPLETE") {
				t.Fatal(result)
			}
		})
	}
}

func TestMigrationOnlyToolingAndEvidenceRemainRetired(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := r.Read("developer/automation/internal/machine/testdata/control_plane_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range Strings(vectors["retired_paths"]) {
		path, err := r.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("retired artifact reintroduced", ref, err)
		}
	}
}

func TestRootMigrationPreservesExactMaterializedRelationSetAndClassBoundary(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := r.Read("developer/automation/internal/machine/testdata/control_plane_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	expected, actual := []string{}, []string{}
	for _, raw := range List(vectors["relation_edges"]) {
		row := List(raw)
		expected = append(expected, fmt.Sprintf("%v|%v|%v|%v", row[0], row[1], row[2], row[3]))
	}
	for _, plane := range []string{"developer/policy", "src/policy"} {
		graph, err := r.Read(plane + "/registries/root-family-migration.json")
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range List(graph["sources"]) {
			source := Map(raw)
			id := Text(source["source_policy_id"])
			relations := Map(migratedSourceValue(t, r, id, "/relations"))
			for _, kind := range []string{"supersedes", "amends", "extends", "depends_on"} {
				for _, raw := range List(relations[kind]) {
					edge := Map(raw)
					target := Text(edge["policy"])
					if strings.HasPrefix(id, "SFP-") && strings.HasPrefix(target, "MPD-") {
						t.Fatal("Support inherited Developer authority", id, target)
					}
					actual = append(actual, fmt.Sprintf("%s|%s|%s|%v", id, kind, target, edge["scope"]))
				}
			}
		}
	}
	// VPMS is an independently admitted current class, outside the Developer Root
	// migration graph; its relation remains part of the preserved relation set.
	policy, err := r.Read("developer/policy/VERI/MPD-VERI-0007.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for kind, raw := range Map(policy["relations"]) {
		for _, raw := range List(raw) {
			edge := Map(raw)
			actual = append(actual, fmt.Sprintf("MPD-VERI-0007|%s|%s|%v", kind, Text(edge["policy"]), edge["scope"]))
		}
	}
	sort.Strings(expected)
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("materialized relation set changed\nactual=%v\nexpected=%v", actual, expected)
	}
}

func TestCurrentPolicyIndexesCoverCanonicalCorpusAndKeepHistoricalSourcesAuditOnly(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := r.LoadConsistentPolicyCorpus()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.policyCheckDiscovery(corpus, ""); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, raw := range List(corpus.Index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		if seen[id] {
			t.Fatal("duplicate identity", id)
		}
		seen[id] = true
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			if !strings.HasPrefix(Text(entry["path"]), "developer/policy/legacy/") {
				t.Fatal(entry)
			}
			continue
		}
		path, err := policyCanonicalPath(id)
		if err != nil || entry["path"] != path {
			t.Fatal(entry, err)
		}
	}
	if !reflect.DeepEqual(Strings(Map(Map(corpus.Subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"]), corpus.IDs) || corpus.Index["legacy_decisions_migration"] != nil {
		t.Fatal("subject/catalog projection mismatch")
	}
	index, err := r.Read("src/policy/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rows := List(index["policies"])
	if len(rows) < 24 {
		t.Fatal(index)
	}
	seen = map[string]bool{}
	for i, raw := range rows {
		entry := Map(raw)
		id := Text(entry["id"])
		if seen[id] {
			t.Fatal("duplicate Support identity", id)
		}
		seen[id] = true
		if i < 24 {
			if id != fmt.Sprintf("SFP-%04d", i+1) || entry["authority_role"] != "MIGRATION_SOURCE" {
				t.Fatal(entry)
			}
			status := "ACTIVE"
			if i == 3 {
				status = "DRAFT"
			}
			if i == 5 {
				status = "RETIRED"
			}
			if entry["status"] != status {
				t.Fatal(entry)
			}
			continue
		}
		payload, err := r.Read("src/policy/" + Text(entry["path"]))
		if err != nil || Map(payload["policy"])["id"] != id || Map(payload["policy"])["status"] != entry["status"] {
			t.Fatal(entry, err)
		}
		data, err := CanonicalJSON(payload)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"subject_binding", "authority_role", "repository_binding"} {
			if strings.Contains(string(data), `"`+forbidden+`"`) {
				t.Fatal("repository-specific authority wrapper", id, forbidden)
			}
		}
	}
}

func TestCanonicalPolicyPathsDistinguishBoundaryAndRejectUnregisteredFamilies(t *testing.T) {
	for _, test := range [][2]string{{"MPD-BOUND-0001", "developer/policy/MPD-BOUND-0001.yaml"}, {"MPD-NORM-0001", "developer/policy/NORM/MPD-NORM-0001.yaml"}, {"MPD-ARCH-0001", "developer/policy/ARCH/MPD-ARCH-0001.yaml"}, {"MPD-VERI-0007", "developer/policy/VERI/MPD-VERI-0007.yaml"}} {
		got, err := policyCanonicalPath(test[0])
		if err != nil || got != test[1] {
			t.Fatal(got, err)
		}
	}
	for _, id := range []string{"MPD-UNKNOWN-0001", "MPD-0001"} {
		if _, err := policyCanonicalPath(id); err == nil {
			t.Fatal("unregistered runtime path synthesized", id)
		}
	}
}

func TestPPImplementationStateAndSupportNamespaceRemainExplicit(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	state := Map(migratedSourceValue(t, r, "MPD-0011", "/rules/implementation_state"))
	requireAgentFields(t, state, Object{"operationalization_level": "L4_LOCAL_AUTO_REMEDIATION_L3_REMOTE_AND_RELEASE_VERIFY", "policy_record": "MATERIALIZED", "policy_index_registration": "MATERIALIZED", "policy_resolver_routing": "MATERIALIZED", "authority_plane_registration": "MATERIALIZED", "transition_reconciler": "MATERIALIZED_AND_HOOK_INVOKED", "h3_hook_activation": "MATERIALIZED_SHARED_INSTALLER", "remote_commit_verifier": "MATERIALIZED_PUSH_VERIFY_ONLY", "release_transition_verifier": "MATERIALIZED_EXACT_SHA_VERIFY_ONLY", "claim": "LOCAL_REMOTE_RELEASE_PP_AUTOMATION_ACTIVE"})
	namespace := Map(Map(migratedSourceValue(t, r, "MPD-SPEC-0001", "/rules/namespace"))["support_feature"])
	requireAgentFields(t, namespace, Object{"machine_policy_path": "src/policy/", "machine_policy_index": "src/policy/index.yaml", "machine_policy_pattern": "src/policy/SFP-*.yaml", "canonical_schema_path": "src/policy/schemas/ptsip-support-feature-policy.schema.json", "embedded_schema_path": "ptsip/support/schemas/ptsip-support-feature-policy.schema.json", "embedded_schema_path_role": "INSTALLED_DISTRIBUTION_PROJECTION", "distribution": "REQUIRED"})
	if !iwpPathExists(r, Text(namespace["canonical_schema_path"])) {
		t.Fatal("canonical Support schema missing")
	}
}

func TestFrozenSpecificationRegistryAndSchemaAliasesRemainExact(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	current, err := r.GitOutput("rev-parse", "HEAD:registry/ptsip-registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := r.GitOutput("rev-parse", "3c47816770d194ae42f98faedc911d980db0e62a:registry/ptsip-registry.yaml")
	if err != nil || current != frozen {
		t.Fatal("frozen Specification registry changed", current, frozen, err)
	}
	for ref, expected := range map[string]string{"schemas/ptsip-project-authority-record.schema.json": "ptsip-support-project-authority-record.schema.json", "schemas/ptsip-authority-eligibility-result.schema.json": "ptsip-support-authority-eligibility-result.schema.json"} {
		schema, err := r.Read(ref)
		if err != nil || schema["$ref"] != expected {
			t.Fatal(schema, err)
		}
	}
}

func TestOwnerAuthorizationAndSupportProjectionHaveSeparatePlanes(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	subject, err := r.Read("src/policy/registries/ptsip-support-authority-subject-registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if subject["current_repository_bindings"] != nil || len(Map(subject["subject_identity_schemes"])) != 1 || Map(subject["subject_identity_schemes"])["SUPPORT_POLICY_ID"] == nil {
		t.Fatal(subject)
	}
	support, err := r.Read("src/policy/registries/ptsip-support-authorization-registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	developer, err := r.Read("developer/policy/registries/authorization-transition-registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if support["authorization_provenance"] != nil || support["rules"] != nil || Map(developer["authorization_provenance"])["authority"] != "PROJECT_OWNER" || Map(developer["rules"])["P03G_PROJECT_AUTHORITY_RUNTIME"] == nil {
		t.Fatal("authorization authority crossed planes")
	}
	ready, err := r.AuthorizationReadiness()
	if err != nil {
		t.Fatal(err)
	}
	for predicate, value := range ready {
		if value != true {
			t.Fatal("readiness predicate failed", predicate, value)
		}
	}
	for _, scope := range []string{"PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME", "PROJECT_AUTHORITY_PROJECTION_RUNTIME", "PROJECT_AUTHORITY_RECORD_MATERIALIZATION", "AUTHORIZATION_READINESS_TRANSITION_ENGINE"} {
		result, err := r.AuthorizationTransition(scope, ready)
		if err != nil || result["state"] != "AUTHORIZED" {
			t.Fatal(result, err)
		}
	}
}

func TestOwnerAuthorizationRequiresExactValidatedSupportCorpus(t *testing.T) {
	for _, defect := range []string{"missing", "duplicate", "unknown", "empty"} {
		t.Run(defect, func(t *testing.T) {
			r := policyTestRepo(t)
			index, err := r.Read("src/policy/index.yaml")
			if err != nil {
				t.Fatal(err)
			}
			rows := List(index["policies"])
			switch defect {
			case "missing":
				rows = rows[:len(rows)-1]
			case "duplicate":
				rows = append(rows, policyClone(Map(rows[len(rows)-1])))
			case "unknown":
				Map(rows[len(rows)-1])["id"] = "SFP-9999"
			case "empty":
				rows = []any{}
			}
			index["policies"] = rows
			policyTestWrite(t, r, "src/policy/index.yaml", index)
			ready, err := r.AuthorizationReadiness()
			if err != nil {
				ready = Object{"CURRENT_SUPPORT_POLICY_CORPUS_VALID": false}
			}
			if ready["CURRENT_SUPPORT_POLICY_CORPUS_VALID"] != false {
				t.Fatal("invalid support corpus admitted", defect)
			}
			result, err := r.AuthorizationTransition("PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME", ready)
			if err != nil || result["state"] != "HOLD_NOT_AUTHORIZED" {
				t.Fatal(result, err)
			}
		})
	}
}

func TestProductGovernanceHasNoLocalRetiredPolicyTreeDependency(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	root, err := r.Path("src/ptsip/governance")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".py") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "decisions"+"/") {
			t.Fatal("product depends on local retired policy tree", entry.Name())
		}
	}
}
