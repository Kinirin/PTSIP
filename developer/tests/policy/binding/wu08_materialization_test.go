package binding_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const wu08Plan = "developer/planning/0.3.8/0.3.8a3/index.yaml"
const wu08Document = "developer/planning/0.3.8/0.3.8a3/WU-08/WU-08.yaml"

func equalWU08JSON(t *testing.T, left, right any) bool {
	t.Helper()
	leftJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatal(err)
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(leftJSON, rightJSON)
}

func TestWU08MutationGateUsesActiveRootOwnerAndPreservesExplicitApprovalEvidence(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	result, err := lifecycle.InspectPolicy(repo, "MPD-CTRL-0001")
	if err != nil || result["policy_status"] != "ACTIVE" || result["index_status"] != "ACTIVE" || result["operationally_resolvable"] != true {
		t.Fatal(result, err)
	}
	if len(contractRule(t, repo, "MPD-CTRL-0001", "unit_mpd_work_0004_b413e855fd78")) == 0 {
		t.Fatal("canonical mutation authorization section missing")
	}
	for _, target := range []string{"APPROVED", "ACTIVE"} {
		ref := "developer/policy/approvals/MPA-20261004-WU08-MUTATION-GATE-" + target + ".yaml"
		record, err := repo.Read(ref)
		if err != nil {
			t.Fatal(err)
		}
		approval := record["approval"].(object)
		if approval["target_status"] != target || approval["decision_source"] != "USER_EXPLICIT" {
			t.Fatal("source approval provenance changed", approval)
		}
	}
}

func TestWU08MaterializedRootRelationsResolveByExactBindingIdentity(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	store := binding.NewStore(repo)
	plan, err := repo.Read(wu08Plan)
	if err != nil {
		t.Fatal(err)
	}
	identity := plan["plan_identity"].(object)
	registry, err := store.LoadBindingRegistry(true)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, raw := range registry.Payload["bindings"].([]any) {
		row := raw.(object)
		if row["plan_ref"] != wu08Plan {
			continue
		}
		checked++
		resolved, err := store.ResolveBindings(object{"binding_id": row["binding_id"], "policy_ref": row["policy_ref"], "plan_ref": wu08Plan})
		if err != nil || resolved["status"] != "BOUND" || len(resolved["bindings"].([]any)) != 1 {
			t.Fatal(resolved, err)
		}
		for _, field := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
			if row[field] != identity[field] {
				t.Fatal("formal plan identity changed", field, row)
			}
		}
		if row["planning_state"] != "CREATED" || len(row["policy_sections"].([]any)) == 0 {
			t.Fatal("Root relationship lost exact section ownership", row)
		}
	}
	if checked != 14 {
		t.Fatal("WU08 relationship coverage changed", checked)
	}
	reconciled, err := store.ReconcileBindings(false)
	if err != nil || reconciled["status"] != "CURRENT" || reconciled["changed"] != false || reconciled["applied"] != false {
		t.Fatal(reconciled, err)
	}
	report, err := store.VerifyConsistency()
	if err != nil || report["status"] != "PASS" || report["checked_binding_count"] != report["binding_count"] {
		t.Fatal(report, err)
	}
}

func TestWU08ExactScopeRoutingAndDigestCandidatesPreserveApprovedSemantics(t *testing.T) {
	resolver := rootResolver(t)
	for _, operation := range []string{"READ", "PLAN", "MODIFY", "VERIFY"} {
		result, err := resolver.Resolve(wu08Document, operation)
		if err != nil || result["binding_scope"] != wu08Document || !reflect.DeepEqual(result["policies"], expectedMigratedRefs(t, wu08Document, operation)) {
			t.Fatal("WU08 scope changed", operation, result, err)
		}
	}
	repo := testrepo.Open(testrepo.Root(t))
	canonical := contractRule(t, repo, "MPD-INFO-0001", "unit_mpd_spec_0022_9826b3cb2a95")
	projection := testrepo.ReadJSON(t, repo, "src/agent_contracts/digests/projection-v1.json")
	digest := testrepo.ReadJSON(t, repo, "src/agent_contracts/digests/policy-v1.json")
	expected := canonical["normative_semantic_projection"].(object)
	for _, candidate := range []object{projection, digest["projection_policy"].(object)} {
		for _, field := range []string{"field_governance", "collection_semantics", "binding_scope"} {
			if !equalWU08JSON(t, candidate[field], expected[field]) {
				t.Fatal("digest candidate diverged from canonical policy", field)
			}
		}
		for _, kind := range []string{"exact_inclusion_list", "exact_exclusion_list"} {
			if len(candidate["field_governance"].(object)[kind].(object)["exact_role_bindings"].([]any)) == 0 {
				t.Fatal("exact digest role bindings lost", kind)
			}
		}
	}
	if !equalWU08JSON(t, digest["activation_requirements"], canonical["canonical_digest_scheme_selection"].(object)["activation_requirements"]) || digest["active_scheme"] != "UNRESOLVED" || projection["normative_authority"] != false || digest["normative_authority"] != false || testrepo.ReadJSON(t, repo, "src/agent_contracts/index.json")["canonical_digest"] != "UNRESOLVED" {
		t.Fatal("unresolved digest scheme was activated or approval requirements changed")
	}
	required := contractRule(t, repo, "MPD-ASSURE-0001", "unit_mpd_veri_0004_f699ce5ccfa4")
	entry := required["repository_entry"].(object)
	for field, want := range map[string]bool{"source_ownership_required_before_mode_selection": true, "adding_analysis_inputs_alone_creates_source_ownership": false, "agent_may_infer_source_classification_or_verification_relationship": false, "unmapped_path_may_use_full_verification_fallback": false} {
		if entry[field] != want {
			t.Fatal("verification ownership boundary changed", field, entry)
		}
	}
	resolved, err := resolver.Resolve("src/agent_contracts/candidate.py", "VERIFY")
	if err != nil || resolved["binding_scope"] != "src/agent_contracts" || !reflect.DeepEqual(resolved["policies"], expectedMigratedRefs(t, "src/agent_contracts", "VERIFY")) {
		t.Fatal(resolved, err)
	}
}
