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

func TestContractFirstPlanScopesImplementationAndPublication(t *testing.T) {
	const planRoot = "src/planning/0.3.8/contract-first-migration/"
	plan := document(t, planRoot+"implementation-runbook.plan.json")
	scope := plan["execution_scope"].(object)
	approved := map[string]bool{}
	for _, raw := range scope["approved_work_units"].([]any) {
		id := raw.(string)
		if approved[id] {
			t.Fatal("duplicate implementation work unit", id)
		}
		approved[id] = true
	}
	index := document(t, planRoot+"index.json")
	for _, raw := range index["responsibility_partitions"].([]any) {
		unit := raw.(object)
		id := unit["work_unit_id"].(string)
		if unit["implementation_authorized"] != approved[id] {
			t.Fatal("implementation approval does not match the runbook", id)
		}
		delete(approved, id)
	}
	if len(approved) != 0 || scope["build_or_publish_release"] != "NOT_AUTHORIZED" {
		t.Fatal("unresolved approved work units or unauthorized publication", approved)
	}
	if scope["already_applied_not_reexecute"].([]any)[0] != "WU-13" {
		t.Fatal("Planning retirement must be recorded as historical work")
	}
}

func TestRegisteredManifestContractBindingAndIndependentOwnerClosure(t *testing.T) {
	all := entries(t)
	plan := document(t, "src/planning/0.3.8/contract-first-migration/scope-baseline.plan.json")
	for _, raw := range plan["confirmed_engine_responsibilities"].(object)["current_migration_engine_kinds"].([]any) {
		entry(t, raw.(string))
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
	state := document(t, "src/ptsip/engines/contracts/engines_registries/derived-snapshot.json")
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

func TestAllSixteenWorkItemsHaveResolvableDeliverablesOrExplicitGaps(t *testing.T) {
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
	for _, id := range []string{
		"VALIDATION_VPMS_PROTOCOL_SELECTION_COMPOSITION_AND_RUNTIME_CONTRACT_BRIDGE",
		"VPMS_PUBLIC_API_AND_TEST_MODE_TRANSITION_PLAN",
	} {
		item := bindings[id].(object)
		if len(item["unresolved_deliverables"].([]any)) == 0 {
			t.Fatal("missing VPMS sources must have explicit unresolved evidence", id)
		}
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
				"VALIDATION": {"schema": object{"type": "integer"}, "instance": 1},
				"EXECUTION":  {"root": t.TempDir(), "attempt": object{"attempt_id": "a", "status": "FAILED", "cause": "RESOURCE_EXHAUSTION", "effect_evidence_refs": []any{}}, "effects": []any{}},
				"PROJECTION": {"context_id": "c", "content": object{"x": 1}, "pointers": []any{"/x"}},
			}
			operation := e["operations"].([]any)[0]
			if e["kind"] == "STORAGE" {
				inputs["STORAGE"] = storageInput(t, e, contract)
			}
			if e["kind"] == "DISCOVERY" {
				root := t.TempDir()
				plan := sourceRef(t, root, "plan.json", object{"id": "example", "version": "1"})
				catalog := sourceRef(t, root, "catalog.json", object{"id": "catalog", "version": "1", "schema_version": "ptsip-plan-discovery-catalog/v1", "plans": []any{plan}})
				inputs["DISCOVERY"] = object{"root": root, "catalog_ref": catalog, "plan_id": "example"}
				operation = "discovery.planning"
			}
			request := object{"engine_id": e["engine_id"], "engine_version": e["version"], "contract_id": contract["contract_id"], "contract_version": contract["version"], "operation_id": operation, "input": inputs[e["kind"].(string)]}
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
