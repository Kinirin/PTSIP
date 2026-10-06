package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func planningFixture(t *testing.T, includePlanning bool) *Repository {
	t.Helper()
	source, err := Open("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	target := &Repository{Root: t.TempDir()}
	paths := []string{"developer/policy", "developer/bindings"}
	if includePlanning {
		paths = append(paths, "developer/planning")
	}
	for _, ref := range paths {
		err = filepath.WalkDir(filepath.Join(source.Root, filepath.FromSlash(ref)), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "legacy" {
					return filepath.SkipDir
				}
				return nil
			}
			relative, _ := filepath.Rel(source.Root, path)
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return target.AtomicWrite(filepath.ToSlash(relative), raw, nil)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return target
}
func emptyBindingRegistry() Object {
	return Object{"schema_version": "ptsip-policy-plan-bindings/v2", "registry_role": "POLICY_PLAN_BINDING", "schema_ref": BindingSchemaPath, "bindings": []any{}}
}
func testWriteYAML(t *testing.T, r *Repository, ref string, payload Object) {
	t.Helper()
	if err := r.WriteYAML(ref, payload, nil); err != nil {
		t.Fatal(err)
	}
}
func TestBindingNativeExactIdentityAndCAS(t *testing.T) {
	r := planningFixture(t, false)
	testWriteYAML(t, r, BindingRegistryPath, emptyBindingRegistry())
	created, err := r.CreateBinding("MPD-INTENT-0001")
	if err != nil {
		t.Fatal(err)
	}
	id := Text(Map(created["binding"])["binding_id"])
	identity := Object{"resolved_plan_id": "PLN.MIGR.READ.A7k2Q9mX", "plan_file_id": "PLANFILE.MAIN.H7sP2kQ9mXa4", "version": "1.0", "revision": "Rev.0001"}
	ref := "developer/planning/example.yaml"
	testWriteYAML(t, r, ref, Object{"plan_identity": identity})
	values := planningClone(identity)
	values["policy_ref"] = "MPD-INTENT-0001"
	values["binding_id"] = id
	values["plan_ref"] = ref
	linked, err := r.LinkPlan(values)
	if err != nil || linked["status"] != "LINKED" {
		t.Fatalf("%v %v", linked, err)
	}
	linked, err = r.LinkPlan(values)
	if err != nil || linked["changed"] != false {
		t.Fatal("idempotent link", linked, err)
	}
	resolution, err := r.ResolveBindings(Object{"policy_ref": "MPD-INTENT-0001", "plan_file_id": identity["plan_file_id"]})
	if err != nil || len(List(resolution["bindings"])) != 1 {
		t.Fatal(resolution, err)
	}
	resolution, err = r.ResolveBindings(Object{"binding_id": "PPB-9999"})
	if err != nil || resolution["status"] != "UNBOUND" {
		t.Fatal(resolution, err)
	}
	if _, err = r.ResolveBindings(Object{}); err == nil {
		t.Fatal("empty query accepted")
	}
	snapshot, err := r.LoadBindingRegistry(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.CreateBinding("MPD-ASSURE-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReplaceBindingRegistry(snapshot.Payload, snapshot.Digest, r.ValidateBindingRegistry); err == nil || !strings.Contains(err.Error(), "STALE") {
		t.Fatalf("stale CAS accepted %v", err)
	}
	if _, err = r.CreateBinding("MPD-0013"); err == nil {
		t.Fatal("old policy alias accepted")
	}
}
func TestBindingNativeMovementAndAmbiguity(t *testing.T) {
	r := planningFixture(t, false)
	ref := "developer/planning/old.yaml"
	identity := Object{"resolved_plan_id": "PLN.MIGR.READ.A7k2Q9mX", "plan_file_id": "PLANFILE.MAIN.H7sP2kQ9mXa4", "version": "1.0", "revision": "Rev.0001"}
	testWriteYAML(t, r, ref, Object{"plan_identity": identity})
	binding := Object{"binding_id": "PPB-0001", "policy_ref": "MPD-INTENT-0001", "planning_state": "CREATED", "resolved_plan_id": identity["resolved_plan_id"], "plan_file_id": identity["plan_file_id"], "version": "1.0", "revision": "Rev.0001", "plan_ref": ref}
	registry := emptyBindingRegistry()
	registry["bindings"] = []any{binding}
	testWriteYAML(t, r, BindingRegistryPath, registry)
	path, _ := r.Path(ref)
	newRef := "developer/planning/new.yaml"
	newPath, _ := r.Path(newRef)
	if err := os.Rename(path, newPath); err != nil {
		t.Fatal(err)
	}
	tracking, err := r.TrackPlanRef("PPB-0001", false)
	if err != nil || tracking["status"] != "RECONCILE_REQUIRED" {
		t.Fatal(tracking, err)
	}
	tracking, err = r.TrackPlanRef("PPB-0001", true)
	if err != nil || tracking["status"] != "RECONCILED" {
		t.Fatal(tracking, err)
	}
	tracking, err = r.TrackPlanRef("PPB-0001", true)
	if err != nil || tracking["status"] != "CURRENT" {
		t.Fatal(tracking, err)
	}
	testWriteYAML(t, r, "developer/planning/duplicate.yaml", Object{"plan_identity": identity})
	if _, err = r.TrackPlanRef("PPB-0001", false); err == nil || !strings.Contains(err.Error(), "AMBIGUOUS") {
		t.Fatal("duplicate file identity accepted", err)
	}
	bad := planningClone(binding)
	bad["binding_id"] = "PPB-0002"
	bad["plan_ref"] = "../outside.yaml"
	registry["bindings"] = []any{binding, bad}
	if failures := r.ValidateBindingRegistry(registry); len(failures) == 0 {
		t.Fatal("escaped plan reference accepted")
	}
}
func readyExtension() Object {
	return Object{"schema_version": "ptsip-plan-extension/v1", "plan_version": "0.4.0", "extension": Object{"id": "WU-02-P01", "parent": "WU-02", "lifecycle": Object{"status": "ACTIVE"}}, "implementation_authorization": Object{"status": "AUTHORIZED", "authorization_source": "USER_EXPLICIT"}, "current_known_blockers": []any{}, "migration_stages": Object{"A": Object{"status": "COMPLETE"}, "B": Object{"status": "COMPLETE"}}, "x_execution_plan": Object{"execution_order": []any{Object{"id": "FIN", "status": "COMPLETE", "validation": Object{"status": "PASS"}}}}}
}
func TestPlanningExtensionMachineReadinessAndProvenance(t *testing.T) {
	if !ExtensionMachineReady(readyExtension()) {
		t.Fatal("ready extension rejected")
	}
	for _, name := range []string{"blocker", "validation", "stage", "records", "schema", "authorization"} {
		t.Run(name, func(t *testing.T) {
			payload := readyExtension()
			switch name {
			case "blocker":
				payload["current_known_blockers"] = []any{"WAIT"}
			case "validation":
				Map(Map(List(Map(payload["x_execution_plan"])["execution_order"])[0])["validation"])["status"] = "PENDING"
			case "stage":
				Map(Map(payload["migration_stages"])["A"])["status"] = "ACTIVE"
			case "records":
				delete(payload, "migration_stages")
				delete(payload, "x_execution_plan")
			case "schema":
				payload["schema_version"] = "unknown"
			case "authorization":
				Map(payload["implementation_authorization"])["status"] = "PENDING"
			}
			if ExtensionMachineReady(payload) {
				t.Fatal("incomplete extension auto-closed")
			}
		})
	}
	parent := Object{"work_unit": Object{"id": "WU-02"}, "extensions": []any{Object{"id": "WU-02-P01", "path": "developer/planning/ext.yaml", "status": "COMPLETE"}}}
	if failures := ExtensionParentConsistency(parent, readyExtension(), "WU-02-P01", "developer/planning/ext.yaml"); len(failures) != 1 {
		t.Fatal(failures)
	}
}
func TestPlanningStagePromotionAndGateSelection(t *testing.T) {
	payload := Object{"execution_plan": Object{"execution_order": []any{Object{"id": "A", "status": "IMPLEMENTED_VALIDATION_PENDING", "validation": Object{"status": "PENDING"}}, Object{"id": "B", "status": "BLOCKED"}}}, "control": Object{"item": Object{"status": "OLD"}}, "blockers": []any{"REMOVE", "KEEP"}}
	automatic := Object{"next_stage": Object{"id": "B", "from_status": "BLOCKED", "to_status": "ACTIVE"}, "document_updates": Object{"mapping_scalars": []any{Object{"section": "control", "key": "item", "field": "status", "from_value": "OLD", "to_value": "NEW"}}, "list_removals": []any{Object{"section": "blockers", "value": "REMOVE"}}}}
	if err := planningPromotePayload(payload, "A", automatic); err != nil {
		t.Fatal(err)
	}
	stage, _ := planningFindStage(payload, "A")
	if stage["status"] != "COMPLETE" || Map(stage["validation"])["status"] != "PASS" {
		t.Fatal(stage)
	}
	next, _ := planningFindStage(payload, "B")
	if next["status"] != "ACTIVE" || Map(Map(payload["control"])["item"])["status"] != "NEW" || !reflect.DeepEqual(payload["blockers"], []any{"KEEP"}) {
		t.Fatal(payload)
	}
	if err := planningPromotePayload(payload, "A", automatic); err == nil {
		t.Fatal("stale stage transition accepted")
	}
	r := &Repository{Root: t.TempDir()}
	wu := func(id, status string, depends []any) Object {
		return Object{"id": id, "path": "developer/planning/" + id + ".yaml", "lifecycle": Object{"status": status}, "depends_on": depends}
	}
	index := Object{"plan": Object{"current_gate": "WU-01"}, "work_units": []any{wu("WU-01", "COMPLETE", []any{}), wu("WU-02", "ACTIVE", []any{"WU-01"}), wu("WU-03", "ACTIVE", []any{"WU-02"})}}
	root := Object{"entry_routing": Object{"dependency_bearing_convergence": Object{"id": "WU-03"}}}
	gate, _, err := r.SelectPlanningGate(index, root, map[string]Object{}, "")
	if err != nil || gate != "WU-02" {
		t.Fatal(gate, err)
	}
	Map(index["plan"])["current_gate"] = "WU-03"
	gate, _, err = r.SelectPlanningGate(index, root, map[string]Object{}, "")
	if err != nil || gate != "WU-03" {
		t.Fatal("active gate preserved", gate, err)
	}
	Map(index["plan"])["current_gate"] = "UNKNOWN"
	if _, _, err = r.SelectPlanningGate(index, root, map[string]Object{}, ""); err == nil {
		t.Fatal("unknown gate accepted")
	}
}
func TestPlanningCanonicalCorpusNativeValidation(t *testing.T) {
	r := planningFixture(t, true)
	failures := r.ValidatePlanning()
	if len(failures) > 0 {
		t.Fatal(strings.Join(failures, "\n"))
	}
	root, err := r.Read(PlanningRootIndex)
	if err != nil {
		t.Fatal(err)
	}
	entry := Map(List(root["plans"])[0])
	routing := Map(entry["entry_routing"])
	routes := List(routing["branch_entrypoints"])
	routing["branch_entrypoints"] = append(routes, planningClone(Map(routes[0])))
	testWriteYAML(t, r, PlanningRootIndex, root)
	if len(r.ValidatePlanning()) == 0 {
		t.Fatal("duplicate routing accepted")
	}
}

type ppMemorySnapshot struct {
	label string
	files map[string][]byte
}

func (s ppMemorySnapshot) Label() string                         { return s.label }
func (s ppMemorySnapshot) ReadBytes(path string) ([]byte, error) { return s.files[path], nil }
func (s ppMemorySnapshot) Resources() ([]string, error) {
	out := []string{}
	for path := range s.files {
		if strings.HasPrefix(path, "profiles/") && !strings.Contains(strings.TrimPrefix(path, "profiles/"), "/") && strings.HasSuffix(path, ".ptsip.yaml") {
			out = append(out, strings.TrimPrefix(path, "profiles/"))
		}
	}
	sort.Strings(out)
	return out, nil
}
func ppTestFiles(t *testing.T) map[string][]byte {
	t.Helper()
	profile := []byte("# retain comment\nptsip:\n  version: 'pp.1.02' # identity\nresponsibility_map:\n  mode: explicit\ncontract:\n  value: original\n")
	schema := Object{"$id": "old", "title": "old", "type": "object", "properties": Object{"ptsip": Object{"properties": Object{"version": Object{"const": "pp.1.02", "description": "identity"}}}}}
	schemaRaw, _ := json.Marshal(schema)
	registry := Object{"schema_version": "ptsip-project-profile-contract-registry/v1", "authority": "PTSIP_PROJECT_PROFILE_CONTRACT_IDENTITY", "current": "pp.1.02", "contracts": []any{Object{"version": "pp.1.02", "lifecycle": "CURRENT", "operations": []any{"IDENTIFY", "VALIDATE", "ANALYZE", "CREATE_TARGET"}, "schema": ppSchemaPath("pp.1.02"), "baseline": ppHistory("pp.1.02")}}, "transitions": []any{}}
	regRaw, _ := yaml.Marshal(registry)
	catalogRaw, _ := yaml.Marshal(Object{"schema_version": "ptsip-public-profile-catalog/v1", "authority": "PTSIP_PUBLIC_PROFILE_CATALOG", "root": "profiles", "profiles": []any{Object{"id": "sample", "resource": "sample.ptsip.yaml", "contract": "pp.1.02", "responsibility_mode": "explicit"}}})
	return map[string][]byte{PPCatalog: catalogRaw, PPRegistry: regRaw, PPEmbeddedRegistry: regRaw, "profiles/sample.ptsip.yaml": profile, ppHistory("pp.1.02") + "/sample.ptsip.yaml": profile, ppSchemaPath("pp.1.02"): schemaRaw, ppEmbeddedSchemaPath("pp.1.02"): schemaRaw}
}
func ppCopyFiles(files map[string][]byte) map[string][]byte {
	copy := map[string][]byte{}
	for key, raw := range files {
		copy[key] = append([]byte{}, raw...)
	}
	return copy
}
func TestPPAuthorityDeltaClassifications(t *testing.T) {
	baseFiles := ppTestFiles(t)
	base, err := LoadPPAuthority(ppMemorySnapshot{"base", baseFiles})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"unchanged", "format", "content", "membership", "resource", "schema", "manual", "wrong-next", "removed", "baseline"} {
		t.Run(name, func(t *testing.T) {
			files := ppCopyFiles(baseFiles)
			expected := "NO_T2_AUTHORITY_DELTA"
			valid, triggered := true, false
			left := base
			switch name {
			case "format":
				files["profiles/sample.ptsip.yaml"] = append([]byte("# extra\n"), files["profiles/sample.ptsip.yaml"]...)
			case "content":
				files["profiles/sample.ptsip.yaml"] = bytes.ReplaceAll(files["profiles/sample.ptsip.yaml"], []byte("original"), []byte("new"))
				expected = "T2_AUTHORITY_DELTA"
				triggered = true
			case "membership":
				catalog, _ := ppYAML(files[PPCatalog], "catalog", true)
				catalog["profiles"] = []any{}
				files[PPCatalog], _ = yaml.Marshal(catalog)
				expected = "T2_AUTHORITY_DELTA"
				triggered = true
			case "resource":
				catalog, _ := ppYAML(files[PPCatalog], "catalog", true)
				Map(List(catalog["profiles"])[0])["resource"] = "other.ptsip.yaml"
				files[PPCatalog], _ = yaml.Marshal(catalog)
				expected = "T2_AUTHORITY_DELTA"
				triggered = true
			case "schema":
				var schema Object
				json.Unmarshal(files[ppSchemaPath("pp.1.02")], &schema)
				schema["additionalProperties"] = false
				files[ppSchemaPath("pp.1.02")], _ = json.Marshal(schema)
				expected = "T2_AUTHORITY_DELTA"
				triggered = true
			case "manual", "wrong-next":
				registry, _ := ppYAML(files[PPRegistry], "registry", true)
				registry["current"] = "pp.1.03"
				Map(List(registry["contracts"])[0])["version"] = "pp.1.03"
				files[PPRegistry], _ = yaml.Marshal(registry)
				expected = "MANUAL_PP_TRANSITION_WITHOUT_AUTHORITY"
				valid = false
				if name == "wrong-next" {
					registry["current"] = "pp.1.04"
					Map(List(registry["contracts"])[0])["version"] = "pp.1.04"
					files[PPRegistry], _ = yaml.Marshal(registry)
					files["profiles/sample.ptsip.yaml"] = bytes.ReplaceAll(files["profiles/sample.ptsip.yaml"], []byte("original"), []byte("new"))
					expected = "PP_TRANSITION_IDENTITY_MISMATCH"
					triggered = true
				}
			case "removed":
				delete(files, PPCatalog)
				expected = "AUTHORITY_PLANE_REMOVED"
				valid = false
				triggered = true
			case "baseline":
				old := ppCopyFiles(baseFiles)
				delete(old, PPCatalog)
				delete(old, PPRegistry)
				left, _ = LoadPPAuthority(ppMemorySnapshot{"old", old})
				expected = "BASELINE_MATERIALIZATION_EXISTING_DISTRIBUTION"
			}
			right, err := LoadPPAuthority(ppMemorySnapshot{name, files})
			if err != nil {
				t.Fatal(err)
			}
			oldSchema := baseFiles[ppSchemaPath("pp.1.02")]
			if name == "schema" {
				oldSchema = files[ppSchemaPath("pp.1.02")]
			}
			result, err := EvaluatePPDelta(left, right, oldSchema)
			if err != nil || result.Classification != expected || result.Valid != valid || result.Triggered != triggered {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
	if _, err = ppNext("pp.1.2"); err == nil {
		t.Fatal("noncanonical PP accepted")
	}
	if _, err = ppNext("pp.01.02"); err == nil {
		t.Fatal("noncanonical major accepted")
	}
}
func TestPPGitAtomicTransitionAndRollbackProtection(t *testing.T) {
	r := planningFixture(t, false)
	files := ppTestFiles(t)
	for path, raw := range files {
		if err := r.AtomicWrite(path, raw, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.invalid"}, {"config", "user.name", "PTSIP Test"}, {"config", "core.hooksPath", "disabled-hooks"}, {"add", "."}, {"commit", "-m", "baseline"}} {
		if _, err := ppGit(r.Root, args...); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := r.ComparePP("HEAD", "", true)
	if before.Classification != "NO_T2_AUTHORITY_DELTA" {
		t.Fatal(before)
	}
	profile := bytes.ReplaceAll(files["profiles/sample.ptsip.yaml"], []byte("original"), []byte("new"))
	if err := r.AtomicWrite("profiles/sample.ptsip.yaml", profile, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ppGit(r.Root, "add", "--", "profiles/sample.ptsip.yaml"); err != nil {
		t.Fatal(err)
	}
	plan, err := r.BuildPPTransition("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "RECONCILE" || plan.Target != "pp.1.03" {
		t.Fatal(plan)
	}
	if !bytes.Contains(plan.Outputs["profiles/sample.ptsip.yaml"], []byte("version: 'pp.1.03' # identity")) {
		t.Fatal("format or comment lost")
	}
	if _, err = r.ApplyPPTransition(plan); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReconcilePP(true); err != nil {
		t.Fatal("idempotent reconcile", err)
	}
	result, err := r.VerifyPPPreCommit()
	if err != nil || result["status"] != "PASS" {
		t.Fatal(result, err)
	}
	if _, err = ppGit(r.Root, "commit", "-m", "T2"); err != nil {
		t.Fatal(err)
	}
	commit, err := r.VerifyPPCommit("HEAD")
	if err != nil || commit["triggered"] != true {
		t.Fatal(commit, err)
	}
	head, _ := ppGit(r.Root, "rev-parse", "HEAD")
	release, err := r.VerifyPPRelease(strings.TrimSpace(string(head)))
	if err != nil || release["status"] != "PASS" {
		t.Fatal(release, err)
	}
	if _, err = r.VerifyPPRelease("HEAD^"); err == nil {
		t.Fatal("nonexact release accepted")
	}
	rangeResult, err := r.VerifyPPRange("HEAD^", "HEAD")
	if err != nil || rangeResult["verified_commit_count"] != 1 {
		t.Fatal(rangeResult, err)
	}
	if err = r.AtomicWrite(ppHistory("pp.1.02")+"/sample.ptsip.yaml", []byte("tamper"), nil); err != nil {
		t.Fatal(err)
	}
	ppGit(r.Root, "add", "--", "profiles/history")
	if _, err = r.BuildPPTransition("HEAD"); err == nil {
		t.Fatal("history mutation accepted")
	}
}
func TestPPMergeParentConflictAndSnapshotFailures(t *testing.T) {
	files := ppTestFiles(t)
	base, _ := LoadPPAuthority(ppMemorySnapshot{"left", files})
	changed := ppCopyFiles(files)
	changed["profiles/sample.ptsip.yaml"] = bytes.ReplaceAll(changed["profiles/sample.ptsip.yaml"], []byte("original"), []byte("new"))
	right, _ := LoadPPAuthority(ppMemorySnapshot{"right", changed})
	if err := PPComparableParents([]*PPAuthorityState{base, right}); err == nil {
		t.Fatal("divergent merge parent accepted")
	}
	for _, name := range []string{"embedded-registry", "schema", "baseline", "mode", "version", "coverage"} {
		t.Run(name, func(t *testing.T) {
			candidate := ppCopyFiles(files)
			switch name {
			case "embedded-registry":
				candidate[PPEmbeddedRegistry] = []byte("invalid")
			case "schema":
				candidate[ppEmbeddedSchemaPath("pp.1.02")] = []byte("{}")
			case "baseline":
				candidate[ppHistory("pp.1.02")+"/sample.ptsip.yaml"] = []byte("tamper")
			case "mode":
				candidate["profiles/sample.ptsip.yaml"] = bytes.ReplaceAll(candidate["profiles/sample.ptsip.yaml"], []byte("explicit"), []byte("OTHER"))
				candidate[ppHistory("pp.1.02")+"/sample.ptsip.yaml"] = candidate["profiles/sample.ptsip.yaml"]
			case "version":
				candidate["profiles/sample.ptsip.yaml"] = bytes.ReplaceAll(candidate["profiles/sample.ptsip.yaml"], []byte("pp.1.02"), []byte("pp.1.03"))
				candidate[ppHistory("pp.1.02")+"/sample.ptsip.yaml"] = candidate["profiles/sample.ptsip.yaml"]
			case "coverage":
				candidate["profiles/extra.ptsip.yaml"] = files["profiles/sample.ptsip.yaml"]
			}
			if _, err := ValidatePPSnapshot(ppMemorySnapshot{name, candidate}); err == nil {
				t.Fatal("invalid projection accepted")
			}
		})
	}
}
func TestPPCanonicalRuntimeProductParity(t *testing.T) {
	r, err := Open("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	state, err := ValidatePPSnapshot(PPGitSnapshot{Root: r.Root, Worktree: true})
	if err != nil {
		t.Fatal(err)
	}
	python := filepath.Join(r.Root, ".venv", "Scripts", "python.exe")
	if _, err = os.Stat(python); err != nil {
		python = "python"
	}
	script := "import json; from ptsip.project_profile_contracts import current_runtime_project_profile_contract; from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION; from ptsip.profile_compatibility import current_project_profile_target; r=current_runtime_project_profile_contract(); t=current_project_profile_target(); print(json.dumps(dict(current=CURRENT_PROJECT_PROFILE_VERSION,runtime=r.version,schema=r.schema_resource,operations=sorted(r.operations),target=t.contract.canonical,target_schema=t.schema_resource)))"
	cmd := exec.Command(python, "-c", script)
	cmd.Dir = r.Root
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(r.Root, "src"))
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("product runtime regression: %s %v", raw, err)
	}
	var actual Object
	if err = json.Unmarshal(raw, &actual); err != nil {
		t.Fatal(err)
	}
	schema := filepath.Base(state.Contracts[state.Current])
	if actual["current"] != state.Current || actual["runtime"] != state.Current || actual["target"] != state.Current || actual["schema"] != schema || actual["target_schema"] != schema {
		t.Fatalf("product projection drift: %+v versus %s", actual, state.Current)
	}
	registry, err := r.Read(PPRegistry)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range List(registry["contracts"]) {
		row := Map(raw)
		if row["version"] == state.Current {
			expected := Strings(row["operations"])
			sort.Strings(expected)
			if !planningEqual(actual["operations"], expected) {
				t.Fatal("runtime operation coverage mismatch")
			}
		}
	}
	_ = fmt.Sprint(actual)
}

func TestPlanningNativeStageTextAndMutationRollback(t *testing.T) {
	r := planningFixture(t, true)
	testWriteYAML(t, r, PlanningRootIndex, Object{"schema_version": "ptsip-developer-planning-root/v1", "policy_class": DeveloperClass, "plans": []any{}})
	source := "# preserve this evidence\nexecution_order:\n  - id: STAGE_A   \n    status: IMPLEMENTED_VALIDATION_PENDING   \n    validation:   \n      status: PENDING   \n    automatic_completion:\n      from_status: IMPLEMENTED_VALIDATION_PENDING\n      to_status: COMPLETE\n      required_checks:\n        - PLANNING_VALIDATION\n  - id: STAGE_B\n    status: BLOCKED\n"
	ref := "developer/planning/stage.yaml"
	if err := r.AtomicWrite(ref, []byte(source), nil); err != nil {
		t.Fatal(err)
	}
	result, err := r.FinalizePlanningStage(ref, "STAGE_A")
	if err != nil || result["promoted"] != true || len(Strings(result["failures"])) > 0 {
		t.Fatal(result, err)
	}
	path, _ := r.Path(ref)
	actual, _ := os.ReadFile(path)
	if !bytes.Contains(actual, []byte("# preserve this evidence")) || !bytes.Contains(actual, []byte("validation:   \n      status: PASS\n    automatic_completion:")) || !bytes.Contains(actual, []byte("  - id: STAGE_B\n    status: BLOCKED\n")) {
		t.Fatal("text structure changed", string(actual))
	}
	result, err = r.FinalizePlanningStage(ref, "STAGE_A")
	if err != nil || result["promoted"] != false || len(Strings(result["failures"])) > 0 {
		t.Fatal("retry not idempotent", result, err)
	}
	before := append([]byte{}, actual...)
	payload, _ := r.Read(ref)
	err = r.planningWriteTransaction(map[string]Object{ref: payload}, func() []string { return []string{"verification rejected"} })
	if err == nil {
		t.Fatal("rejected validation accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("rollback changed original bytes")
	}
	digest := SHA256(before)
	if err = r.AtomicWrite(ref, []byte("foreign: edit\n"), nil); err != nil {
		t.Fatal(err)
	}
	if err = r.planningWriteTransaction(map[string]Object{ref: payload}, nil, map[string]string{ref: digest}); err == nil {
		t.Fatal("stale planning snapshot accepted")
	}
	after, _ = os.ReadFile(path)
	if string(after) != "foreign: edit\n" {
		t.Fatal("stale rejection overwrote foreign change")
	}
}

func TestPlanningNativeStageDocumentSynchronization(t *testing.T) {
	source := "migration_stages:\n  P01_E:\n    status: IMPLEMENTED_VALIDATION_PENDING\n    artifacts: PENDING\ncurrent_known_blockers:\n  - BLOCKER\nexecution_order:\n    - id: STAGE_A\n      status: IMPLEMENTED_VALIDATION_PENDING\n      validation:\n        status: PENDING\n    - id: STAGE_B\n      status: BLOCKED\n"
	automatic := Object{"next_stage": Object{"id": "STAGE_B", "from_status": "BLOCKED", "to_status": "READY"}, "document_updates": Object{"mapping_scalars": []any{Object{"section": "migration_stages", "key": "P01_E", "field": "status", "from_value": "IMPLEMENTED_VALIDATION_PENDING", "to_value": "COMPLETE"}, Object{"section": "migration_stages", "key": "P01_E", "field": "artifacts", "from_value": "PENDING", "to_value": "RETIRED"}}, "list_removals": []any{Object{"section": "current_known_blockers", "value": "BLOCKER"}}}}
	text, err := PlanningPromoteStageText(source, "STAGE_A", automatic)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := ppYAML([]byte(text), "promoted", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(List(payload["current_known_blockers"])) != 0 || !strings.Contains(text, "current_known_blockers: []") || Map(Map(payload["migration_stages"])["P01_E"])["artifacts"] != "RETIRED" {
		t.Fatal(text)
	}
	if !strings.Contains(text, "status: PASS\n    - id: STAGE_B\n      status: READY") {
		t.Fatal(text)
	}
	if _, err = PlanningPromoteStageText(source, "MISSING", automatic); err == nil {
		t.Fatal("unknown stage accepted")
	}
	invalid := planningClone(automatic)
	Map(invalid["document_updates"])["mapping_scalars"] = "malformed"
	if _, err = PlanningPromoteStageText(source, "STAGE_A", invalid); err == nil {
		t.Fatal("invalid update list accepted")
	}
}

func TestPlanningNativeWorkUnitAuthorityGuards(t *testing.T) {
	approval := Object{"status": "APPROVED", "approval_source": "USER_EXPLICIT", "inherited_from": []any{}}
	authorization := Object{"status": "AUTHORIZED", "authorization_source": "USER_EXPLICIT"}
	wu := Object{"id": "WU-01", "lifecycle": Object{"status": "ACTIVE"}, "approval": approval, "implementation_authorization": authorization, "depends_on": []any{}}
	for _, name := range []string{"approval-source", "authorization-source", "provenance", "dependencies", "completion-evidence"} {
		t.Run(name, func(t *testing.T) {
			copy := planningClone(wu)
			doc := Object{"work_unit": copy}
			switch name {
			case "approval-source":
				Map(copy["approval"])["approval_source"] = "MODEL_INFERENCE"
			case "authorization-source":
				Map(copy["implementation_authorization"])["authorization_source"] = "MODEL_INFERENCE"
			case "provenance":
				delete(Map(copy["approval"]), "inherited_from")
			case "dependencies":
				copy["depends_on"] = "unknown"
			case "completion-evidence":
				Map(copy["lifecycle"])["status"] = "COMPLETE"
			}
			if err := planningCopyState(Object{"work_units": []any{Object{"id": "WU-01"}}}, map[string]Object{"WU-01": doc}); err == nil {
				t.Fatal("invalid authority state accepted")
			}
		})
	}
}
