package repository_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

const identityCandidate = "src/agent_contracts/contracts/context-evidence-identity.json"
const identityVectors = "src/agent_contracts/conformance/context-evidence-identity.json"
const identitySchema = "src/agent_contracts/schemas/candidate-context-evidence.schema.json"
const identityPlan = "developer/planning/0.3.8/0.3.8a3/WU-03/WU-03-P01.yaml"

func candidateDocument(t *testing.T, r *testrepo.Repository, ref string) testrepo.Object {
	t.Helper()
	value, err := r.Read(ref)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func identityCandidateFixture(t *testing.T) *testrepo.Repository {
	t.Helper()
	r := testrepo.Open(t.TempDir())
	testrepo.CopyTree(t, r, "developer/policy", "developer/planning", "developer/bindings", "src/policy")
	testrepo.CopyFiles(t, r, "pyproject.toml", identityCandidate, identityVectors, identitySchema)
	return r
}

func TestNativeContextEvidenceCandidate(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	r := identityCandidateFixture(t)
	semantics := testrepo.ReadJSON(t, r, "src/policy/schemas/ptsip-support-authority-semantics.schema.json")
	defs := semantics["$defs"].(map[string]any)
	common := defs["ROOT_FAMILY_V1"].(map[string]any)["properties"].(map[string]any)["family_definition"]
	specialized := defs["SFP_INFO_0004_V1"].(map[string]any)["properties"].(map[string]any)["family_definition"]
	commonJSON, _ := json.Marshal(common)
	specializedJSON, _ := json.Marshal(specialized)
	if string(commonJSON) != string(specializedJSON) {
		t.Fatal("source composition specialization changed the common kernel definition shape")
	}
	before := map[string]string{}
	for _, ref := range []string{identityCandidate, identityVectors, identitySchema, identityPlan, "src/policy/INFO/SFP-INFO-0007.yaml"} {
		path, _ := r.Path(ref)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[ref] = string(raw)
	}
	result, err, output := testrepo.CLI(t, binary, r.Root, "agent-context-migration", "verify", "--stage", "CONTEXT_EVIDENCE_CANDIDATE")
	if err != nil || result["status"] != "PASS" || result["vectors_checked"] != float64(30) {
		t.Fatalf("%v\n%s", err, output)
	}
	if result["assignment_performed"] != false || result["mutation_authorized"] != false || result["normative_authority"] != false {
		t.Fatal("candidate acquired runtime authority")
	}
	if len(result["activation_blockers"].([]any)) != 3 || len(result["projections"].([]any)) != 2 {
		t.Fatal("missing bounded candidate projection")
	}
	blocked, allowed := 0, 0
	for _, raw := range result["results"].([]any) {
		item := raw.(map[string]any)["result"].(map[string]any)
		if item["projected_action"] == "BLOCK" {
			blocked++
		} else {
			allowed++
		}
		if item["assignment_performed"] != false || item["mutation_authorized"] != false {
			t.Fatal("synthetic proof authorized a write")
		}
	}
	if blocked != 25 || allowed != 5 {
		t.Fatalf("proof gate coverage changed: blocked=%d allowed=%d", blocked, allowed)
	}
	for ref, want := range before {
		path, _ := r.Path(ref)
		raw, _ := os.ReadFile(path)
		if string(raw) != want {
			t.Fatalf("candidate verification wrote %s", ref)
		}
	}
	if _, err := os.Stat(filepath.Join(r.Root, ".ptsip")); !os.IsNotExist(err) {
		t.Fatal("candidate created a consumer runtime plane")
	}
}

func TestNativeContextEvidenceCandidateFailsClosed(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	for _, change := range []string{"activate", "four_roles", "weaken_history", "dangling_gate", "semantic_id", "wrong_proof_type", "vector_mismatch", "current_policy_change", "missing_policy_registration", "plan_unapproved"} {
		t.Run(change, func(t *testing.T) {
			r := identityCandidateFixture(t)
			module := testrepo.ReadJSON(t, r, identityCandidate)
			ref, value := identityCandidate, module
			switch change {
			case "activate":
				module["runtime_activation"] = "ACTIVE"
			case "four_roles":
				for _, raw := range module["primitives"].(map[string]any) {
					primitive := raw.(map[string]any)
					delete(primitive, "failure_semantics_ref")
					delete(primitive, "conformance_vectors_ref")
					break
				}
			case "weaken_history":
				module["gates"].(map[string]any)["never_assigned"].(map[string]any)["input_constraint"] = map[string]any{}
			case "dangling_gate":
				for _, raw := range module["operations"].(map[string]any) {
					raw.(map[string]any)["gate_refs"] = []any{"unregistered"}
					break
				}
			case "semantic_id", "wrong_proof_type", "vector_mismatch":
				ref, value = identityVectors, testrepo.ReadJSON(t, r, identityVectors)
				for _, raw := range value["operations"].(map[string]any) {
					vector := raw.([]any)[0].(map[string]any)
					if change == "semantic_id" {
						vector["input"].(map[string]any)["candidate_id"] = "provider-feature-20261010"
					}
					if change == "wrong_proof_type" {
						vector["input"].(map[string]any)["facts"].(map[string]any)["proof_revision_current"] = "true"
					}
					if change == "vector_mismatch" {
						vector["expected"].(map[string]any)["projected_action"] = "BLOCK"
					}
					break
				}
			case "current_policy_change":
				ref = "src/policy/INFO/SFP-INFO-0007.yaml"
				value = candidateDocument(t, r, ref)
				value["policy"].(map[string]any)["status"] = "ACTIVE"
			case "missing_policy_registration":
				ref = "src/policy/index.yaml"
				value = candidateDocument(t, r, ref)
				for _, raw := range value["policies"].([]any) {
					row := raw.(map[string]any)
					if row["id"] == "SFP-INFO-0007" {
						row["path"] = "INFO/unregistered.yaml"
					}
				}
			case "plan_unapproved":
				ref, value = identityPlan, candidateDocument(t, r, identityPlan)
				value["approval"].(map[string]any)["status"] = "PENDING"
			}
			testrepo.WriteJSON(t, r, ref, value)
			_, err, output := testrepo.CLI(t, binary, r.Root, "agent-context-migration", "verify", "--stage", "CONTEXT_EVIDENCE_CANDIDATE")
			if err == nil {
				t.Fatalf("invalid %s candidate accepted\n%s", change, output)
			}
			var unresolved map[string]any
			if err := json.Unmarshal([]byte(output), &unresolved); err != nil || unresolved["status"] != "UNRESOLVED" || unresolved["reason_code"] != "CANDIDATE_REQUIRED_CONTRACT_UNRESOLVED" {
				t.Fatalf("candidate failure lacks a typed unresolved result: %v\n%s", err, output)
			}
			if unresolved["assignment_performed"] != false || unresolved["mutation_authorized"] != false {
				t.Fatal("unresolved candidate acquired mutation authority")
			}
			if strings.Contains(output, "executable file not found") {
				t.Fatal("candidate depends on an external interpreter")
			}
		})
	}
}

// Successor to the legacy repository-scoped Context/Evidence policy test. The
// neutral candidate declares expectations; this Go test verifies their native
// execution and the current P4 handoff without resurrecting a retired proof name.
func TestContextEvidenceRepositoryScopePolicyConformance(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	r := identityCandidateFixture(t)
	module := testrepo.ReadJSON(t, r, identityCandidate)
	bindings := module["policy_bindings"].(map[string]any)
	for i := 0; i < 20; i++ {
		key := "repository_scope_conformance_" + strconv.Itoa(i)
		if bindings[key] == nil {
			t.Fatalf("missing legacy verification successor expectation: %s", key)
		}
	}
	result, err, output := testrepo.CLI(t, binary, r.Root, "agent-context-migration", "verify", "--stage", "CONTEXT_EVIDENCE_CANDIDATE")
	if err != nil || result["status"] != "PASS" {
		t.Fatalf("scope policy conformance: %v\n%s", err, output)
	}
	if result["assignment_performed"] != false {
		t.Fatal("policy verification assigned an identity")
	}
}
