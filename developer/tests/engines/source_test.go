package engines_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func entry(t *testing.T, kind string) object {
	t.Helper()
	for _, raw := range entries(t) {
		e := raw.(object)
		if e["kind"] == kind {
			return e
		}
	}
	t.Fatal("unregistered engine", kind)
	return nil
}

func native(t *testing.T, kind string) string {
	t.Helper()
	e := entry(t, kind)
	cmd := exec.Command("cargo", "build", "--locked", "--manifest-path", e["cargo_manifest_ref"].(string))
	cmd.Dir = testrepo.Root(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native build required: %v\n%s", err, out)
	}
	binding := document(t, e["execution_binding_ref"].(string))["bindings"].([]any)[0].(object)
	binary := filepath.Join(testrepo.Root(t), filepath.Dir(filepath.FromSlash(e["cargo_manifest_ref"].(string))), "target", "debug", binding["entrypoint"].(string))
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	return binary
}

func invoke(t *testing.T, binary, kind, operation string, input object, pass bool) object {
	t.Helper()
	e := entry(t, kind)
	c := document(t, e["contract_ref"].(string))
	request := object{"engine_id": e["engine_id"], "engine_version": e["version"], "contract_id": c["contract_id"], "contract_version": c["version"], "operation_id": operation, "input": input}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary)
	cmd.Stdin = strings.NewReader(string(encoded))
	out, err := cmd.Output()
	var result object
	if decode := json.Unmarshal(out, &result); decode != nil {
		t.Fatalf("structured native result unavailable: %v %v %s", err, decode, out)
	}
	if (err == nil) != pass || (result["status"] == "PASS") != pass {
		t.Fatalf("expected PASS=%v: %v %s", pass, err, out)
	}
	if result["normative_authority"] != false || result["execution_authorized"] != false {
		t.Fatal("verification created authority", result)
	}
	if pass {
		return result["output"].(object)
	}
	return result
}

func writeJSON(t *testing.T, root, ref string, value any) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(ref))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	bytes = append(bytes, '\n')
	if err := os.WriteFile(p, bytes, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}

func copyRegisteredSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	idx := document(t, "src/ptsip/engines/index.json")
	refs := []string{"src/ptsip/engines/index.json", idx["architecture_owner"].(string), idx["contract_owner"].(string)}
	federation := idx["federation"].(object)
	refs = append(refs, federation["static_registry_ref"].(string), federation["execution_registry_ref"].(string), idx["transport_contract_ref"].(string))
	for _, raw := range idx["engine_schema_refs"].(object) {
		refs = append(refs, raw.(string))
	}
	for _, raw := range entries(t) {
		e := raw.(object)
		for _, key := range []string{"manifest_ref", "contract_ref", "execution_binding_ref"} {
			refs = append(refs, e[key].(string))
		}
		for _, rawExtension := range document(t, e["manifest_ref"].(string))["extensions"].([]any) {
			refs = append(refs, rawExtension.(object)["path"].(string))
		}
	}
	for _, ref := range refs {
		bytes, err := os.ReadFile(filepath.Join(testrepo.Root(t), filepath.FromSlash(ref)))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, filepath.FromSlash(ref))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, bytes, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func storageInput(t *testing.T, e, c object) object {
	t.Helper()
	root := t.TempDir()
	contractRef := "storage-contract.json"
	contractHash := writeJSON(t, root, contractRef, c)
	provider := object{"id": "provider-contract", "version": "1.0.0", "provider_id": "consumer-provider", "provider_owner": "CONSUMER_OWNER", "status": "ACTIVE", "operations": []any{object{"operation_id": "get"}}}
	providerHash := writeJSON(t, root, "provider.json", provider)
	evidence := object{"id": "provider-compatibility", "version": "1.0.0", "provider_id": "consumer-provider", "storage_engine_id": e["engine_id"], "provider_operation_id": "get", "storage_operation_id": "read", "outcome": "PASS", "source_refs": []any{object{"id": provider["id"], "version": provider["version"], "path": "provider.json", "sha256": providerHash}, object{"id": c["contract_id"], "version": c["version"], "path": contractRef, "sha256": contractHash}}}
	evidenceHash := writeJSON(t, root, "compatibility.json", evidence)
	registry := object{"schema_version": "ptsip-storage-provider-registry/v1", "engine_id": e["engine_id"], "providers": []any{object{"provider_id": "consumer-provider", "provider_owner": "CONSUMER_OWNER", "operation_id": "read", "provider_operation_id": "get", "storage_capability_id": c["contract_id"], "storage_contract_version": c["version"], "source_ref": object{"id": provider["id"], "version": provider["version"], "path": "provider.json", "sha256": providerHash}, "compatibility_ref": object{"id": evidence["id"], "version": evidence["version"], "path": "compatibility.json", "sha256": evidenceHash}}}}
	hash := writeJSON(t, root, "registry.json", registry)
	return object{"root": root, "provider_registry_ref": "registry.json", "expected_registry_sha256": hash, "capability_id": c["contract_id"], "operation_id": "read", "contract_version": c["version"]}
}

func TestSourceBoundDiscoveryRejectsStalenessAndPreservesIndependentSelection(t *testing.T) {
	binary := native(t, "DISCOVERY")
	root := copyRegisteredSource(t)
	input := object{"root": root, "index_ref": "src/ptsip/engines/index.json", "expected_revision": nil}
	proof := invoke(t, binary, "DISCOVERY", "discovery.verify-registry", input, true)
	if len(proof["registered_engines"].([]any)) != 7 || proof["derived_authority"] != false {
		t.Fatal("registered source closure missing", proof)
	}
	context := entry(t, "CONTEXT")
	contract := document(t, context["contract_ref"].(string))
	selected := object{"root": root, "index_ref": input["index_ref"], "expected_revision": nil, "engine_id": context["engine_id"], "capability_id": contract["contract_id"], "version": context["version"], "contract_id": contract["contract_id"]}
	first := invoke(t, binary, "DISCOVERY", "discovery.resolve", selected, true)
	selected["expected_revision"] = first["source_revision"]
	// Corrupt an unrelated engine's source. Current Context selection has a proven independent closure.
	unrelated := entry(t, "PLANNING")
	writeJSON(t, root, unrelated["manifest_ref"].(string), object{"bad": "source"})
	invoke(t, binary, "DISCOVERY", "discovery.resolve", selected, true)
	invoke(t, binary, "DISCOVERY", "discovery.verify-registry", input, false)
	path := filepath.Join(root, filepath.FromSlash(context["contract_ref"].(string)))
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(original, []byte(" ")...), 0644); err != nil {
		t.Fatal(err)
	}
	invoke(t, binary, "DISCOVERY", "discovery.resolve", selected, false)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	invoke(t, binary, "DISCOVERY", "discovery.resolve", selected, true)
}

func TestStorageMediationRequiresBothCurrentOwnerSources(t *testing.T) {
	binary := native(t, "STORAGE")
	e := entry(t, "STORAGE")
	c := document(t, e["contract_ref"].(string))
	input := storageInput(t, e, c)
	output := invoke(t, binary, "STORAGE", "storage.mediate", input, true)
	if output["provider_owner"] != "CONSUMER_OWNER" || output["provider_invoked"] != false {
		t.Fatal("provider ownership or invocation boundary changed")
	}
	root := input["root"].(string)
	fixture := testrepo.Open(root)
	proof, err := fixture.Read("compatibility.json")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := fixture.Read("registry.json")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := fixture.Read("provider.json")
	if err != nil {
		t.Fatal(err)
	}
	provider["unrelated_revision"] = "same-id-and-version-different-source"
	refs := proof["source_refs"].([]any)
	refs[0].(object)["path"] = "other-provider.json"
	refs[0].(object)["sha256"] = writeJSON(t, root, "other-provider.json", provider)
	registry["providers"].([]any)[0].(object)["compatibility_ref"].(object)["sha256"] = writeJSON(t, root, "compatibility.json", proof)
	input["expected_registry_sha256"] = writeJSON(t, root, "registry.json", registry)
	invoke(t, binary, "STORAGE", "storage.mediate", input, false)
	writeJSON(t, root, "provider.json", object{"id": "forged-provider", "version": "1.0.0"})
	invoke(t, binary, "STORAGE", "storage.mediate", input, false)
}

func TestTypedImpactGraphUsesCurrentTwoSidedEvidenceAndScopedClosure(t *testing.T) {
	binary := native(t, "PLANNING")
	root := t.TempDir()
	var nodes []any
	hashes := map[string]string{}
	for _, id := range []string{"a", "b", "c", "independent"} {
		hashes[id] = writeJSON(t, root, id+".json", object{"id": id, "version": "1", "value": "current"})
		nodes = append(nodes, object{"id": id, "version": "1", "path": id + ".json", "sha256": hashes[id]})
	}
	evidence := func(p, c, producer, compatibility string) object {
		id := p + "-" + c + "-" + producer
		path := id + ".json"
		hash := writeJSON(t, root, path, object{"id": id, "version": "1", "provider": p, "consumer": c, "producer_id": producer, "provider_sha256": hashes[p], "consumer_sha256": hashes[c], "outcome": "PASS", "compatibility": compatibility})
		return object{"id": id, "version": "1", "path": path, "sha256": hash}
	}
	ab := object{"provider": "a", "consumer": "b", "relation": "DEPENDENCY", "provider_evidence": evidence("a", "b", "a", "COMPATIBLE"), "consumer_evidence": evidence("a", "b", "b", "COMPATIBLE")}
	bc := object{"provider": "b", "consumer": "c", "relation": "DEPENDENCY", "provider_evidence": evidence("b", "c", "b", "COMPATIBLE"), "consumer_evidence": evidence("b", "c", "c", "COMPATIBLE")}
	graph := object{"nodes": nodes, "edges": []any{ab, bc}}
	input := object{"root": root, "graph_ref": "graph.json", "expected_graph_sha256": writeJSON(t, root, "graph.json", graph), "changed_id": "a"}
	out := invoke(t, binary, "PLANNING", "planning.impact", input, true)
	if out["status"] != "VERIFIED_COMPATIBLE_UNAFFECTED" || len(out["affected_ids"].([]any)) != 1 {
		t.Fatal("current independent compatibility proof ignored", out)
	}
	ab["provider_evidence"] = evidence("a", "b", "a", "AFFECTED")
	ab["consumer_evidence"] = evidence("a", "b", "b", "AFFECTED")
	input["expected_graph_sha256"] = writeJSON(t, root, "graph.json", graph)
	out = invoke(t, binary, "PLANNING", "planning.impact", input, true)
	if out["status"] != "VERIFIED_AFFECTED" || len(out["affected_ids"].([]any)) != 2 {
		t.Fatal("proven downstream compatibility did not bound propagation", out)
	}
	ab["provider_evidence"].(object)["sha256"] = strings.Repeat("0", 64)
	input["expected_graph_sha256"] = writeJSON(t, root, "graph.json", graph)
	out = invoke(t, binary, "PLANNING", "planning.impact", input, true)
	if out["status"] != "UNCERTAIN_UNVERIFIED" || out["verification_eligible"] != false {
		t.Fatal("stale evidence issued eligibility", out)
	}
	graph["edges"] = append(graph["edges"].([]any), object{"provider": "c", "consumer": "a", "relation": "DEPENDENCY"})
	input["expected_graph_sha256"] = writeJSON(t, root, "graph.json", graph)
	invoke(t, binary, "PLANNING", "planning.impact", input, false)
}

func TestVersionedHandoffMappingClassifiesScopeWithoutReplacingPublicAPI(t *testing.T) {
	binary := native(t, "VALIDATION")
	root := t.TempDir()
	refs := []string{"src/vpms/contracts/index.json", "src/vpms/contracts/protocol.json", "src/vpms/contracts/selection.json", "src/vpms/contracts/execution-composition.json", "src/ptsip/engines/contracts/vpms-scope-mapping.json"}
	for _, ref := range refs {
		b, err := os.ReadFile(filepath.Join(testrepo.Root(t), filepath.FromSlash(ref)))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, filepath.FromSlash(ref))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	mappingRef := refs[4]
	input := object{"root": root, "mapping_ref": mappingRef, "expected_mapping_sha256": digest(t, mappingRef)}
	out := invoke(t, binary, "VALIDATION", "validation.verify-handoff", input, true)
	if len(out["covered_source_ids"].([]any)) != 3 || len(out["cardinalities"].([]any)) != 3 || out["public_api_equivalence"] != false || out["cutover_authorized"] != false {
		t.Fatal("current relation scope or public boundary invalid", out)
	}
	mapping := document(t, mappingRef)
	mapping["relations"].([]any)[0].(object)["public_api_equivalence"] = true
	input["expected_mapping_sha256"] = writeJSON(t, root, mappingRef, mapping)
	invoke(t, binary, "VALIDATION", "validation.verify-handoff", input, false)
}

func TestIndependentValidationCompositionHasTypedClosure(t *testing.T) {
	binary := native(t, "VALIDATION")
	a := object{"check_id": "a", "operation_id": "validation.schema", "schema": object{"type": "integer"}, "instance": 1, "depends_on": []any{}}
	b := object{"check_id": "b", "operation_id": "validation.typed-io", "schema": object{"type": "string"}, "instance": 1, "depends_on": []any{"a"}}
	input := object{"checks": []any{b, a}}
	out := invoke(t, binary, "VALIDATION", "validation.compose", input, true)
	if out["valid_all"] != false || out["checks"].([]any)[0].(object)["check_id"] != "a" || out["ptsip_conformance_verdict"] != "NOT_ASSERTED" {
		t.Fatal("independent validation or normative judgment boundary changed", out)
	}
	a["depends_on"] = []any{"b"}
	invoke(t, binary, "VALIDATION", "validation.compose", input, false)
}

func TestInstallationSelectionNeedsAllSevenCurrentHostArtifacts(t *testing.T) {
	binary := native(t, "PLANNING")
	root := copyRegisteredSource(t)
	planRef := "plan.json"
	plan := document(t, "src/ptsip/engines/contracts/installation-plan.json")
	hostTarget := "x86_64-unknown-linux-gnu"
	if runtime.GOOS == "windows" {
		hostTarget = "x86_64-pc-windows-msvc"
	}
	if runtime.GOOS == "darwin" {
		hostTarget = "aarch64-apple-darwin"
	}
	input := object{"root": root, "plan_ref": planRef, "expected_plan_sha256": writeJSON(t, root, planRef, plan), "host_target": hostTarget, "evidence_refs": []any{}}
	out := invoke(t, binary, "PLANNING", "planning.select-installation", input, true)
	if out["ready"] != false || len(out["unverified_packages"].([]any)) != 7 || out["installation_authorized"] != false {
		t.Fatal("missing host proof silently enabled installation", out)
	}
	var reports []any
	for _, raw := range entries(t) {
		e := raw.(object)
		kind := e["kind"].(string)
		artifact, err := os.ReadFile(native(t, kind))
		if err != nil {
			t.Fatal(err)
		}
		artifactRef := "artifacts/" + strings.ToLower(kind) + ".bin"
		artifactPath := filepath.Join(root, filepath.FromSlash(artifactRef))
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(artifactPath, artifact, 0644); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(artifact)
		report := object{"id": "report-" + kind, "version": "1", "engine_id": e["engine_id"], "implementation_version": e["version"], "host_target": input["host_target"], "outcome": "PASS", "verification_scope": "NATIVE_TEST_BUILD_PACKAGE_STDIO_CONFORMANCE", "manifest_ref": object{"path": e["manifest_ref"], "sha256": digest(t, e["manifest_ref"].(string))}, "artifact_ref": object{"path": artifactRef, "sha256": hex.EncodeToString(hash[:])}}
		ref := "reports/" + strings.ToLower(kind) + ".json"
		reports = append(reports, object{"id": report["id"], "version": "1", "path": ref, "sha256": writeJSON(t, root, ref, report)})
	}
	input["evidence_refs"] = reports
	out = invoke(t, binary, "PLANNING", "planning.select-installation", input, true)
	if out["ready"] != true || len(out["selected_packages"].([]any)) != 7 || out["external_installation_performed"] != false {
		t.Fatal("current native artifact closure not usable as a plan", out)
	}
	input["evidence_refs"] = reports[:6]
	out = invoke(t, binary, "PLANNING", "planning.select-installation", input, true)
	if out["ready"] != false {
		t.Fatal("partial package availability claimed all seven")
	}
}

func TestExecutionEffectObservationCannotUseStaleOrUnlinkedEvidence(t *testing.T) {
	binary := native(t, "EXECUTION")
	root := t.TempDir()
	observation := object{"id": "observation", "version": "1", "attempt_id": "attempt", "observation": "VERIFIED_NO_EFFECT"}
	ref := object{"id": "observation", "version": "1", "path": "observation.json", "sha256": writeJSON(t, root, "observation.json", observation)}
	record := object{"evidence_id": "effect", "attempt_id": "attempt", "observation": "VERIFIED_NO_EFFECT", "source_ref": ref}
	input := object{"root": root, "attempt": object{"attempt_id": "attempt", "status": "FAILED", "cause": "RESOURCE_EXHAUSTION", "effect_evidence_refs": []any{ref}}, "effects": []any{record}}
	out := invoke(t, binary, "EXECUTION", "execution.assess-attempt", input, true)
	if out["effect_status"] != "VERIFIED_NO_EFFECT" || out["attempt_status"] != "FAILED" {
		t.Fatal("source observation and attempt result were conflated", out)
	}
	writeJSON(t, root, "observation.json", object{"id": "observation", "version": "1", "attempt_id": "attempt", "observation": "OBSERVED_EFFECT"})
	out = invoke(t, binary, "EXECUTION", "execution.assess-attempt", input, true)
	if out["effect_status"] != "UNVERIFIED" {
		t.Fatal("stale observation claimed effect absence", out)
	}
	input["attempt"].(object)["effect_evidence_refs"] = []any{}
	invoke(t, binary, "EXECUTION", "execution.assess-attempt", input, false)
}

func TestContextProjectionPlanPreservesOwnersAndBlocksImplicitConsumerCutover(t *testing.T) {
	binary := native(t, "VALIDATION")
	root := copyRegisteredSource(t)
	ref := "handoff.json"
	plan := document(t, "src/ptsip/engines/contracts/context-projection-handoff.json")
	input := object{"root": root, "mapping_ref": ref, "expected_mapping_sha256": writeJSON(t, root, ref, plan)}
	out := invoke(t, binary, "VALIDATION", "validation.verify-context-handoff", input, true)
	if out["source_semantics_owner"] != "CONTEXT" || out["output_transformation_owner"] != "PROJECTION" || out["public_api_changed"] != false {
		t.Fatal("owner boundary or public preservation changed", out)
	}
	plan["consumer_cutovers"] = []any{object{"consumer_id": "unapproved"}}
	input["expected_mapping_sha256"] = writeJSON(t, root, ref, plan)
	invoke(t, binary, "VALIDATION", "validation.verify-context-handoff", input, false)
}

func TestPreservedVPMSCallerPlanResolvesCurrentBindingsAndVerificationRoutes(t *testing.T) {
	plan := document(t, "src/ptsip/engines/contracts/vpms-caller-transition-plan.json")
	if plan["public_api_retirement"] != false || plan["cutover_performed"] != false || plan["unmapped_or_ambiguous"] != "BLOCK_CUTOVER" {
		t.Fatal("preserved API plan admitted an implicit cutover")
	}
	profile := document(t, plan["profile_ref"].(string))
	components := map[string]object{}
	for _, raw := range profile["components"].([]any) {
		component := raw.(object)
		components[component["id"].(string)] = component
	}
	modes := map[string]object{}
	for _, raw := range document(t, plan["mode_registry_ref"].(string))["modes"].([]any) {
		mode := raw.(object)
		modes[mode["id"].(string)] = mode
	}
	contract := document(t, "src/ptsip/engines/validation/contract.json")
	catalog := document(t, plan["current_source_catalog_ref"].(string))["contracts"].(object)
	bindings := map[string]object{}
	versions := map[string]float64{}
	for id, raw := range catalog {
		registration := raw.(object)
		if registration["status"] != "ACTIVE" {
			t.Fatal("caller plan source is not active", id)
		}
		source := document(t, "src/vpms/contracts/"+registration["path"].(string))
		versions[id] = source["revision"].(float64)
		planned, ok := source["planned_bindings"].([]any)
		if !ok {
			planned = []any{source["planned_bindings"]}
		}
		for _, binding := range planned {
			b := binding.(object)
			bindings[id+":"+b["api"].(string)] = b
		}
	}
	for _, raw := range plan["caller_contract_relations"].([]any) {
		row := raw.(object)
		id := row["source_contract_id"].(string)
		key := id + ":" + row["api"].(string)
		binding, ok := bindings[key]
		version, err := strconv.ParseFloat(row["source_contract_version"].(string), 64)
		if !ok || err != nil || version != versions[id] || row["implementation_ref"] != binding["source"] || row["current_availability"] != binding["availability"] {
			t.Fatal("missing, duplicate or stale current API binding", row)
		}
		delete(bindings, key)
		if row["consumer_cutover_authorized"] != false || row["new_validation_relation"] != "VALIDATION_COVERAGE_ONLY_NOT_RUNTIME_EQUIVALENCE" {
			t.Fatal("coverage silently replaced public behavior", row)
		}
		wantTransition := "PRESERVE_CURRENT_PUBLIC_API"
		if binding["availability"] == "RETIRED" {
			wantTransition = "PRESERVE_RETIRED_STATE"
		}
		if row["transition"] != wantTransition {
			t.Fatal("API state transition changed", row)
		}
		candidate := row["candidate_validation_contract"].(object)
		if candidate["id"] != contract["contract_id"] || candidate["version"] != contract["version"] || candidate["relation"] != row["new_validation_relation"] {
			t.Fatal("candidate capability identity or scope drift", row)
		}
		for _, rawRoute := range row["verification_routes"].([]any) {
			route := rawRoute.(object)
			mode, exists := modes[route["test_mode_id"].(string)]
			if !exists || mode["component_ref"] != route["component_ref"] {
				t.Fatal("unmapped verification mode", route)
			}
			if _, exists := components[route["verifies_component_ref"].(string)]; !exists {
				t.Fatal("unregistered profile implementation/contract component", route)
			}
			found := false
			for _, rawRelationship := range profile["relationships"].([]any) {
				relation := rawRelationship.(object)
				if relation["type"] == "VERIFIES" && relation["from"] == route["component_ref"] && relation["to"] == route["verifies_component_ref"] {
					found = true
				}
			}
			if !found {
				t.Fatal("verification responsibility has no explicit profile relationship", route)
			}
		}
	}
	if len(bindings) != 0 {
		t.Fatal("public API omitted from transition plan", bindings)
	}
}
