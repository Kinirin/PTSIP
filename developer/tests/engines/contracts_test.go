package engines_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

type object = map[string]any

func document(t *testing.T, ref string) object {
	t.Helper()
	d, err := testrepo.Open(testrepo.Root(t)).Read(ref)
	if err != nil {
		t.Fatal(ref, err)
	}
	return d
}

func entries(t *testing.T) []any {
	t.Helper()
	return document(t, "src/ptsip/engines/index.json")["official_engine_entries"].([]any)
}

func schema(t *testing.T, name string, value any) error {
	t.Helper()
	return testrepo.Open(testrepo.Root(t)).Validate("src/ptsip/engines/contracts/schemas/"+name+".schema.json", value)
}

func digest(t *testing.T, ref string) string {
	t.Helper()
	r := testrepo.Open(testrepo.Root(t))
	p, err := r.Path(ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(ref, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func TestActivePolicyAndPlanReconcileDeferredEngineBoundary(t *testing.T) {
	index := document(t, "src/policy/index.yaml")
	for _, id := range []string{"SFP-ARCH-0004", "SFP-CNTR-0005"} {
		found := false
		for _, raw := range index["policies"].([]any) {
			entry := raw.(object)
			if entry["id"] != id {
				continue
			}
			found = true
			policy := document(t, "src/policy/"+entry["path"].(string))
			if entry["status"] != "ACTIVE" || policy["policy"].(object)["status"] != "ACTIVE" {
				t.Fatal("activation source/index mismatch", id)
			}
			unit := policy["authority_semantics"].(object)["unit_ptsip_engine_draft"].(object)
			if unit["policy_activation"].(object)["approval_source"] != "USER_EXPLICIT" {
				t.Fatal("test result cannot create activation approval")
			}
			if unit["implementation_admission"].(object)["legacy_python_implementation"] != "FORBIDDEN" {
				t.Fatal("legacy reuse boundary lost")
			}
		}
		if !found {
			t.Fatal("unregistered current policy", id)
		}
	}
	base := document(t, "src/policy/ARCH/SFP-ARCH-0001.yaml")["authority_semantics"].(object)["engine_plane_boundary"].(object)
	if base["runtime_activation"] != "CURRENT_REGISTERED_CONTRACT_AND_GOV_CTRL_AUTHORIZATION_REQUIRED" {
		t.Fatal("runtime guard not reconciled")
	}
	if base["executable_source_copy_into_repository_engine_root"] != "FORBIDDEN" {
		t.Fatal("consumer source boundary changed")
	}
	wu := document(t, "developer/planning/0.3.8/WU-02/WU-02.yaml")["work_unit"].(object)
	if wu["approval"].(object)["approval_source"] != "USER_EXPLICIT" || wu["implementation_authorization"].(object)["status"] != "AUTHORIZED" {
		t.Fatal("plan authority must be explicit")
	}
}

func TestRegisteredManifestContractBindingAndIndependentOwnerClosure(t *testing.T) {
	all := entries(t)
	if len(all) != 7 {
		t.Fatal("exact seven official engines required")
	}
	seen := map[string]bool{}
	for _, raw := range all {
		e := raw.(object)
		id := e["engine_id"].(string)
		if seen[id] {
			t.Fatal("duplicate engine identity", id)
		}
		seen[id] = true
		manifest := document(t, e["manifest_ref"].(string))
		if err := schema(t, "core-manifest", manifest); err != nil {
			t.Fatal(err)
		}
		if manifest["engine_id"] != id || manifest["origin"] != "OFFICIAL" {
			t.Fatal("explicit identity/origin mismatch")
		}
		contract := document(t, e["contract_ref"].(string))
		if err := schema(t, "capability-contract", contract); err != nil {
			t.Fatal(err)
		}
		if contract["engine_id"] != id || contract["semantic_owner"] != e["kind"] || contract["execution_permission_from_declaration"] != false {
			t.Fatal("contract owner/permission boundary")
		}
		for _, value := range manifest["capabilities"].([]any) {
			ref := value.(object)
			if ref["id"] != contract["contract_id"] || ref["sha256"] != digest(t, ref["path"].(string)) {
				t.Fatal("stale or wrong manifest contract binding")
			}
		}
		for _, value := range contract["operations"].([]any) {
			op := value.(object)
			for _, field := range []string{"typed_input", "typed_output", "preconditions", "postconditions", "side_effects", "declared_permission_requirements", "failure_semantics", "verification_evidence", "compatibility", "machine_semantics"} {
				if op[field] == nil {
					t.Fatal("incomplete independent operation contract", field)
				}
			}
		}
		binding := document(t, e["execution_binding_ref"].(string))["bindings"].([]any)[0].(object)
		if err := schema(t, "execution-binding", binding); err != nil {
			t.Fatal(err)
		}
		if binding["engine_id"] != id || binding["capability_id"] != contract["contract_id"] || binding["contract_version"] != contract["version"] {
			t.Fatal("entry binding is not exact")
		}
		transport := binding["transport_ref"].(object)
		if transport["sha256"] != digest(t, transport["path"].(string)) {
			t.Fatal("transport reference stale")
		}
		projectionRef := filepath.ToSlash(filepath.Join(filepath.Dir(e["manifest_ref"].(string)), "transport.json"))
		if transport["sha256"] != digest(t, projectionRef) {
			t.Fatal("package-local neutral transport projection diverges from its canonical owner")
		}
		cargoPath := filepath.Join(testrepo.Root(t), filepath.FromSlash(e["cargo_manifest_ref"].(string)))
		cargo, err := os.ReadFile(cargoPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"path =", "pyo3", "rust-cpython"} {
			if strings.Contains(string(cargo), forbidden) {
				t.Fatal("engine implementation isolation violated", forbidden)
			}
		}
		if len(manifest["dependencies"].([]any)) != 0 {
			t.Fatal("these independent initial capabilities have no undeclared engine dependency")
		}
	}
	state := document(t, "src/ptsip/engines/registries/derived-snapshot.json")
	if err := schema(t, "derived-state", state); err != nil {
		t.Fatal(err)
	}
	if state["authority"] != false || state["reuse_eligible"] != false {
		t.Fatal("derived state must not issue current eligibility")
	}
	for _, raw := range state["source_refs"].([]any) {
		ref := raw.(object)
		if ref["sha256"] != digest(t, ref["path"].(string)) {
			t.Fatal("snapshot cannot be reused after source change", ref["path"])
		}
	}
}

func TestDataContractsRejectAuthorityExpansionAndConflateNeitherStopNorEffect(t *testing.T) {
	ref := object{"id": "id", "version": "1", "path": "registered.json", "sha256": strings.Repeat("a", 64)}
	vectors := []struct {
		name  string
		input object
	}{
		{"derived-state", object{"state_id": "s", "source_refs": []any{ref}, "snapshot_revision": "r", "authority": false, "write_permission": false, "reuse_eligible": false}},
		{"attempt-result", object{"attempt_id": "a", "status": "FAILED", "cause": "RESOURCE_EXHAUSTION", "effect_evidence_refs": []any{}}},
		{"effect-evidence", object{"evidence_id": "e", "attempt_id": "a", "observation": "UNKNOWN", "source_ref": ref}},
		{"stop-ack", object{"correlation_id": "s", "status": "ACCEPTED", "cause": "cancel"}},
		{"stop-result", object{"correlation_id": "s", "attempt_id": "a", "status": "PENDING", "effect_evidence_refs": []any{}}},
		{"impact-result", object{"changed_id": "c", "affected_ids": []any{"a"}, "status": "UNCERTAIN_UNVERIFIED", "verification_eligible": false, "execution_authorized": false}},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			if err := schema(t, v.name, v.input); err != nil {
				t.Fatal(err)
			}
			v.input["grant_write"] = true
			if err := schema(t, v.name, v.input); err == nil {
				t.Fatal("unknown authority field accepted")
			}
		})
	}
	state := vectors[0].input
	delete(state, "grant_write")
	state["authority"] = true
	if err := schema(t, "derived-state", state); err == nil {
		t.Fatal("state cannot become authority")
	}
	impact := vectors[5].input
	delete(impact, "grant_write")
	impact["verification_eligible"] = true
	if err := schema(t, "impact-result", impact); err == nil {
		t.Fatal("uncertain compatibility cannot grant verification eligibility")
	}
	impact["verification_eligible"] = false
	impact["execution_authorized"] = true
	if err := schema(t, "impact-result", impact); err == nil {
		t.Fatal("impact judgment cannot grant execution")
	}
}

func TestAllSixteenWorkItemsAndCurrentPublicHandoffsHaveExplicitRoutes(t *testing.T) {
	bindings := document(t, "src/ptsip/engines/contracts/work-item-bindings.json")["items"].(object)
	if len(bindings) != 16 {
		t.Fatal("sixteen approved items must remain visible")
	}
	wu := document(t, "developer/planning/0.3.8/WU-02/WU-02.yaml")["work_unit"].(object)
	for _, raw := range wu["work_items"].([]any) {
		item := raw.(object)
		id := item["id"].(string)
		binding, ok := bindings[id].(object)
		if !ok {
			t.Fatal("unmapped item", id)
		}
		for _, raw := range binding["deliverable_refs"].([]any) {
			document(t, raw.(string))
		}
	}
	handoff := document(t, "src/ptsip/engines/contracts/vpms-handoff.json")
	current := document(t, "src/vpms/contracts/index.json")["contracts"].(object)
	if len(handoff["relations"].([]any)) != 3 {
		t.Fatal("current VPMS identities missing")
	}
	for _, raw := range handoff["relations"].([]any) {
		relation := raw.(object)
		source := relation["source_contract"].(object)
		if current[source["id"].(string)] == nil || source["status"] != "ACTIVE" || relation["eligible_for_cutover"] != false {
			t.Fatal("unverified API cutover or legacy reconstruction")
		}
	}
	if handoff["legacy_retirement"] != false || handoff["generic_validation_implies_vpms_protocol_equivalence"] != false {
		t.Fatal("generic validation is not public VPMS equivalence")
	}
}

func TestNativeIndependentRustEnginePackages(t *testing.T) {
	cargo, err := exec.LookPath("cargo")
	if err != nil {
		t.Fatal("required Rust toolchain unavailable; verification cannot be skipped", err)
	}
	for _, raw := range entries(t) {
		e := raw.(object)
		t.Run(strings.ToLower(e["kind"].(string)), func(t *testing.T) {
			for _, args := range [][]string{{"test", "--locked"}, {"build", "--locked"}, {"package", "--list", "--allow-dirty", "--locked"}, {"package", "--allow-dirty", "--locked"}} {
				args = append(args, "--manifest-path", e["cargo_manifest_ref"].(string))
				cmd := exec.Command(cargo, args...)
				cmd.Dir = testrepo.Root(t)
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("%v: %v\n%s", args, err, output)
				}
				if args[0] == "package" && args[1] == "--list" && (strings.Contains(string(output), ".py\n") || strings.Contains(string(output), "target/")) {
					t.Fatal("source package includes foreign implementation or build output")
				}
			}
			binding := document(t, e["execution_binding_ref"].(string))["bindings"].([]any)[0].(object)
			binary := filepath.Join(testrepo.Root(t), filepath.Dir(filepath.FromSlash(e["cargo_manifest_ref"].(string))), "target", "debug", binding["entrypoint"].(string))
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			cmd := exec.Command(binary, "--manifest")
			out, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			var compiled object
			if err := json.Unmarshal(out, &compiled); err != nil {
				t.Fatal(err)
			}
			if compiled["engine_id"] != e["engine_id"] || compiled["implementation_version"] != e["version"] {
				t.Fatal("compiled identity does not match registered binding")
			}
			contract := document(t, e["contract_ref"].(string))
			inputs := map[string]object{
				"CONTEXT":    {"context_id": "c", "content": object{"x": 1}, "source_revision": "r"},
				"STORAGE":    {"capability_id": "c", "operation_id": "read", "contract_version": "1", "providers": []any{object{"capability_id": "c", "provider_id": "p", "operation_id": "read", "provider_operation_id": "get", "contract_version": "1", "compatible": true, "provider_owner": "consumer"}}},
				"VALIDATION": {"schema": object{"type": "integer"}, "instance": 1},
				"DISCOVERY":  {"engine_id": "e", "capability_id": "c", "version": "1", "contract_id": "k", "registrations": []any{object{"engine_id": "e", "capability_id": "c", "version": "1", "contract_id": "k", "entrypoint": "exe", "origin": "CONSUMER", "current": true, "eligible": true}}},
				"PLANNING":   {"nodes": []any{object{"id": "a", "dependencies": []any{}}}},
				"EXECUTION":  {"attempt": object{"attempt_id": "a", "status": "FAILED", "cause": "RESOURCE_EXHAUSTION", "effect_evidence_refs": []any{}}, "effects": []any{}},
				"PROJECTION": {"context_id": "c", "content": object{"x": 1}, "pointers": []any{"/x"}},
			}
			if e["kind"] == "STORAGE" {
				inputs["STORAGE"] = storageInput(t, e, contract)
			}
			if e["kind"] == "DISCOVERY" {
				inputs["DISCOVERY"] = object{"root": testrepo.Root(t), "index_ref": "src/ptsip/engines/index.json", "expected_revision": nil, "engine_id": e["engine_id"], "capability_id": contract["contract_id"], "version": e["version"], "contract_id": contract["contract_id"]}
			}
			inputs["EXECUTION"]["root"] = testrepo.Root(t)
			request := object{"engine_id": e["engine_id"], "engine_version": e["version"], "contract_id": contract["contract_id"], "contract_version": contract["version"], "operation_id": e["operations"].([]any)[0], "input": inputs[e["kind"].(string)]}
			encoded, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			cmd = exec.Command(binary)
			cmd.Stdin = strings.NewReader(string(encoded))
			out, err = cmd.Output()
			if err != nil {
				t.Fatal("registered positive invocation failed", err, string(out))
			}
			var positive object
			if err := json.Unmarshal(out, &positive); err != nil {
				t.Fatal(err)
			}
			if positive["status"] != "PASS" || positive["normative_authority"] != false || positive["execution_authorized"] != false {
				t.Fatal("native positive result changed authority boundary")
			}
			cmd = exec.Command(binary)
			cmd.Stdin = strings.NewReader(`{"grant_write":true}`)
			out, err = cmd.Output()
			if err == nil {
				t.Fatal("invalid native request succeeded")
			}
			var result object
			if err := json.Unmarshal(out, &result); err != nil {
				t.Fatal("failure result must be structured", err)
			}
			if result["status"] != "FAIL" || result["execution_authorized"] != false || result["normative_authority"] != false {
				t.Fatal("native result expands authority", fmt.Sprint(result))
			}
		})
	}
}
