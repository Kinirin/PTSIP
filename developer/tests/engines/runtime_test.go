package engines_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func sourceRef(t *testing.T, root, path string, value object) object {
	t.Helper()
	return object{"id": value["id"], "version": value["version"], "path": path, "sha256": writeJSON(t, root, path, value)}
}

func fileHash(t *testing.T, root, path string) string {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

func copyNative(t *testing.T, root, binary, name string) object {
	t.Helper()
	bytes, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	path := "artifacts/" + name
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bytes, 0755); err != nil {
		t.Fatal(err)
	}
	return object{"path": path, "sha256": fileHash(t, root, path)}
}

func registeredInvocation(t *testing.T, root, kind, id, operation string, input object) object {
	t.Helper()
	e := entry(t, kind)
	c := document(t, e["contract_ref"].(string))
	binding := document(t, e["execution_binding_ref"].(string))["bindings"].([]any)[0].(object)
	name := binding["entrypoint"].(string)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	artifact := copyNative(t, root, native(t, kind), name)
	stored := object{"id": id + "-request", "version": "1.0.0", "request": object{"engine_id": e["engine_id"], "engine_version": e["version"], "contract_id": c["contract_id"], "contract_version": c["version"], "operation_id": operation, "input": input}}
	return object{"id": id, "version": "1.0.0", "engine_id": e["engine_id"], "engine_version": e["version"], "capability_id": c["contract_id"], "contract_version": c["version"], "operation_id": operation, "manifest_ref": object{"id": e["engine_id"], "version": e["version"], "path": e["manifest_ref"], "sha256": digest(t, e["manifest_ref"].(string))}, "contract_ref": object{"id": c["contract_id"], "version": c["version"], "path": e["contract_ref"], "sha256": digest(t, e["contract_ref"].(string))}, "binding_ref": object{"path": e["execution_binding_ref"], "sha256": digest(t, e["execution_binding_ref"].(string))}, "artifact_ref": artifact, "request_ref": sourceRef(t, root, "requests/"+id+".json", stored)}
}

func admittedAuthority(t *testing.T, root, kind, operation, invocationID string, requestHash, artifactHash any) object {
	t.Helper()
	e := entry(t, kind)
	c := document(t, e["contract_ref"].(string))
	profile := object{"schema_version": "ptsip-engine-owner-admission/v1", "repository_id": "isolated-native-verification", "owner_id": "fixture-project-owner", "admission_source": "EXPLICIT_CALLER_PROJECT_AUTHORITY", "authority_policy_refs": []any{"SFP-GOV-0001", "SFP-CTRL-0001"}, "runtime_enabled": true, "actors": []any{"fixture-operator"}, "operation_subjects": []any{object{"engine_id": e["engine_id"], "capability_id": c["contract_id"], "contract_version": c["version"], "operation_id": operation}}}
	profileRef := object{"path": "owner-admission.json", "sha256": writeJSON(t, root, "owner-admission.json", profile)}
	authority := object{"schema_version": "ptsip-engine-authority-consumption/v1", "id": "isolated-owner-grant", "version": "1.0.0", "repository_id": profile["repository_id"], "owner_id": profile["owner_id"], "approval_source": "USER_EXPLICIT", "status": "APPROVED", "profile_ref": profileRef, "actor_id": "fixture-operator", "engine_id": e["engine_id"], "capability_id": c["contract_id"], "contract_version": c["version"], "operation_id": operation, "invocation_id": invocationID, "request_sha256": requestHash, "artifact_sha256": artifactHash, "not_before_unix_ms": time.Now().Add(-time.Second).UnixMilli(), "expires_unix_ms": time.Now().Add(time.Minute).UnixMilli(), "required_remote_sources": []any{}, "grant_is_engine_issued": false}
	return object{"authority_ref": sourceRef(t, root, "authority.json", authority), "profile_ref": profileRef, "actor_id": "fixture-operator", "attempt_id": "native-attempt", "budget_ms": 10000}
}

func merge(left, right object) object {
	result := object{}
	for k, v := range left {
		result[k] = v
	}
	for k, v := range right {
		result[k] = v
	}
	return result
}

func TestNativeContextProjectionCurrentSourceAndOutputPipeline(t *testing.T) {
	root := copyRegisteredSource(t)
	contextBinary := native(t, "CONTEXT")
	projectionBinary := native(t, "PROJECTION")
	source := object{"schema_version": "ptsip-native-context-source/v1", "id": "context-source", "version": "1.0.0", "content": object{"count": 7, "a/b": object{"~label": "한글"}}, "constraints": object{"type": "object", "required": []any{"count"}}}
	ref := sourceRef(t, root, "context-source.json", source)
	snapshot := invoke(t, contextBinary, "CONTEXT", "context.load", object{"root": root, "source_ref": ref}, true)
	snapshotRef := object{"path": "snapshot.json", "sha256": writeJSON(t, root, "snapshot.json", snapshot)}
	definition := object{"id": "projection-definition", "version": "1.0.0", "semantic_owner": "PROJECTION", "format": "JSON_OBJECT", "fields": []any{object{"name": "number", "pointer": "/count"}, object{"name": "label", "pointer": "/a~1b/~0label"}}, "output_schema": object{"type": "object", "required": []any{"number", "label"}}}
	definitionRef := sourceRef(t, root, "projection.json", definition)
	input := object{"root": root, "snapshot_ref": snapshotRef, "definition_ref": definitionRef}
	output := invoke(t, projectionBinary, "PROJECTION", "projection.render", input, true)
	if output["content"].(object)["label"] != "한글" || output["content"].(object)["number"] != float64(7) {
		t.Fatal("source values changed", output)
	}
	definition["format"] = "TEXT_LINES"
	definition["output_schema"] = object{"type": "string"}
	input["definition_ref"] = sourceRef(t, root, "projection.json", definition)
	output = invoke(t, projectionBinary, "PROJECTION", "projection.render", input, true)
	if output["content"] != "number:7\nlabel:\"한글\"\n" {
		t.Fatal("declared projection format drift", output)
	}
	snapshot["content"].(object)["count"] = 99
	input["snapshot_ref"] = object{"path": "snapshot.json", "sha256": writeJSON(t, root, "snapshot.json", snapshot)}
	invoke(t, projectionBinary, "PROJECTION", "projection.render", input, false)
	bad := []byte(`{"schema_version":"ptsip-native-context-source/v1","id":"context-source","id":"forged","version":"1.0.0","content":{},"constraints":{}}`)
	if err := os.WriteFile(filepath.Join(root, "context-source.json"), bad, 0600); err != nil {
		t.Fatal(err)
	}
	ref["sha256"] = fileHash(t, root, "context-source.json")
	invoke(t, contextBinary, "CONTEXT", "context.load", object{"root": root, "source_ref": ref}, false)
	ref["path"] = "../context-source.json"
	invoke(t, contextBinary, "CONTEXT", "context.load", object{"root": root, "source_ref": ref}, false)
}

func TestNativeValidationCatalogSelectionAndAllSelectedAdapterPreflight(t *testing.T) {
	root := t.TempDir()
	binary := native(t, "VALIDATION")
	schema := object{"type": "integer"}
	adapter := object{"id": "adapter", "version": "1.0.0", "invocation_id": "native-check", "input_schema": schema, "output_schema": schema}
	adapterRef := sourceRef(t, root, "adapter.json", adapter)
	var cases []any
	for _, id := range []string{"a", "b"} {
		cases = append(cases, sourceRef(t, root, id+".json", object{"id": id, "version": "1.0.0", "input": 3, "input_schema": schema, "output_schema": schema, "adapter_ref": adapterRef, "tags": []any{"native"}}))
	}
	selectorRef := sourceRef(t, root, "selector.json", object{"id": "ordered", "version": "1.0.0", "case_ids": []any{"b", "a"}})
	catalog := object{"schema_version": "ptsip-native-verification-catalog/v1", "id": "catalog", "version": "1.0.0", "cases": cases, "selectors": []any{selectorRef}}
	ref := sourceRef(t, root, "catalog.json", catalog)
	invoke(t, binary, "VALIDATION", "validation.prepare-cases", object{"root": root, "catalog_ref": ref}, true)
	input := object{"root": root, "catalog_ref": ref, "selection": object{"case_ids": []any{}, "selector_id": "ordered"}}
	output := invoke(t, binary, "VALIDATION", "validation.select-cases", input, true)
	if output["selected_ids"].([]any)[0] != "b" {
		t.Fatal("explicit selector order lost", output)
	}
	output = invoke(t, binary, "VALIDATION", "validation.prepare-composition", input, true)
	if len(output["prepared_adapters"].([]any)) != 2 || output["execution_authorized"] != false {
		t.Fatal("preflight or permission boundary invalid", output)
	}
	adapter["input_schema"] = object{"type": "string"}
	newRef := sourceRef(t, root, "adapter.json", adapter)
	for i, id := range []string{"a", "b"} {
		cases[i] = sourceRef(t, root, id+".json", object{"id": id, "version": "1.0.0", "input": 3, "input_schema": schema, "output_schema": schema, "adapter_ref": newRef, "tags": []any{"native"}})
	}
	input["catalog_ref"] = sourceRef(t, root, "catalog.json", catalog)
	invoke(t, binary, "VALIDATION", "validation.prepare-composition", input, false)
	input["selection"] = object{"case_ids": []any{"missing"}, "selector_id": nil}
	invoke(t, binary, "VALIDATION", "validation.select-cases", input, false)
}

func TestNativeDiscoveryPlanningAndActualEngineInvocation(t *testing.T) {
	root := copyRegisteredSource(t)
	contextRef := sourceRef(t, root, "content.json", object{"schema_version": "ptsip-native-context-source/v1", "id": "native-context", "version": "1.0.0", "content": object{"ready": true}, "constraints": object{"type": "object"}})
	invocation := registeredInvocation(t, root, "CONTEXT", "load-context", "context.load", object{"root": root, "source_ref": contextRef})
	registry := object{"schema_version": "ptsip-native-invocation-registry/v1", "id": "registry", "version": "1.0.0", "owner_id": "fixture-project-owner", "index_ref": "src/ptsip/engines/index.json", "invocations": []any{invocation}}
	registryRef := sourceRef(t, root, "invocations.json", registry)
	input := object{"root": root, "registry_ref": registryRef, "invocation_id": "load-context"}
	invoke(t, native(t, "DISCOVERY"), "DISCOVERY", "discovery.verify-invocation", input, true)
	plan := object{"id": "plan", "version": "1.0.0", "steps": []any{object{"id": "load", "dependencies": []any{}, "invocation_id": "load-context"}}}
	planRef := sourceRef(t, root, "plan.json", plan)
	invoke(t, native(t, "PLANNING"), "PLANNING", "planning.prepare", object{"root": root, "registry_ref": registryRef, "plan_ref": planRef}, true)
	input = merge(input, admittedAuthority(t, root, "EXECUTION", "execution.invoke", "load-context", invocation["request_ref"].(object)["sha256"], invocation["artifact_ref"].(object)["sha256"]))
	binary := native(t, "EXECUTION")
	output := invoke(t, binary, "EXECUTION", "execution.invoke", input, true)
	if output["attempt_status"] != "SUCCEEDED" || output["invoked"] != true || output["effect_status"] != "UNKNOWN" || output["normative_permission_created"] != false {
		t.Fatal("native invocation receipt invalid", output)
	}
	input["actor_id"] = "unadmitted-actor"
	invoke(t, binary, "EXECUTION", "execution.invoke", input, false)
	input["actor_id"] = "fixture-operator"
	artifactPath := invocation["artifact_ref"].(object)["path"].(string)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(artifactPath)), []byte("stale artifact"), 0755); err != nil {
		t.Fatal(err)
	}
	invoke(t, binary, "EXECUTION", "execution.invoke", input, false)
}

func storageRuntimeInput(t *testing.T, delay int, fail bool, size int) (object, string) {
	t.Helper()
	e := entry(t, "STORAGE")
	c := document(t, e["contract_ref"].(string))
	input := storageInput(t, e, c)
	root := input["root"].(string)
	providerSource := filepath.Join(testrepo.Root(t), "developer", "tests", "engines", "testdata", "provider", "main.go")
	name := "native-consumer-provider"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(root, name)
	cmd := exec.Command("go", "build", "-o", binary, providerSource)
	cmd.Dir = testrepo.Root(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	artifactRef := copyNative(t, root, binary, name)
	fixture := testrepo.Open(root)
	provider, err := fixture.Read("provider.json")
	if err != nil {
		t.Fatal(err)
	}
	outputSchema := object{"type": "object", "required": []any{"stored"}, "properties": object{"stored": object{"type": "integer"}}}
	provider["operations"] = []any{object{"operation_id": "get", "input_schema": object{}, "output_schema": object{}}, object{"operation_id": "put", "input_schema": object{"type": "object"}, "output_schema": outputSchema}}
	provider["native_binding"] = object{"transport": "PROVIDER_JSON_STDIO_V1", "arguments": []any{}, "environment": object{}, "artifact_ref": artifactRef}
	providerRef := sourceRef(t, root, "provider.json", provider)
	proof, err := fixture.Read("compatibility.json")
	if err != nil {
		t.Fatal(err)
	}
	proof["provider_operation_id"] = "put"
	proof["source_refs"].([]any)[0] = providerRef
	proofRef := sourceRef(t, root, "compatibility.json", proof)
	registry, err := fixture.Read("registry.json")
	if err != nil {
		t.Fatal(err)
	}
	registration := registry["providers"].([]any)[0].(object)
	registration["provider_operation_id"] = "put"
	registration["source_ref"] = providerRef
	registration["compatibility_ref"] = proofRef
	input["expected_registry_sha256"] = writeJSON(t, root, "registry.json", registry)
	request := object{"id": "provider-request", "version": "1.0.0", "operation_id": "put", "input": object{"target": "effect.json", "value": 7, "delay_ms": delay, "fail": fail, "output_size": size, "invalid_response": false}}
	requestRef := sourceRef(t, root, "provider-request.json", request)
	input["request_ref"] = requestRef
	input = merge(input, admittedAuthority(t, root, "STORAGE", "storage.invoke", "consumer-provider", requestRef["sha256"], artifactRef["sha256"]))
	return input, root
}

func TestNativeStorageActualProviderEffectAndAuthorityDenial(t *testing.T) {
	binary := native(t, "STORAGE")
	input, root := storageRuntimeInput(t, 0, false, 0)
	output := invoke(t, binary, "STORAGE", "storage.invoke", input, true)
	if output["attempt_status"] != "SUCCEEDED" || output["effect_status"] != "UNKNOWN" {
		t.Fatal(output)
	}
	bytes, err := os.ReadFile(filepath.Join(root, "effect.json"))
	if err != nil || string(bytes) != "7" {
		t.Fatal("provider owned effect missing", err, string(bytes))
	}
	denied, deniedRoot := storageRuntimeInput(t, 0, false, 0)
	denied["actor_id"] = "not-admitted"
	invoke(t, binary, "STORAGE", "storage.invoke", denied, false)
	if _, err := os.Stat(filepath.Join(deniedRoot, "effect.json")); !os.IsNotExist(err) {
		t.Fatal("unauthorized provider was invoked", err)
	}
	remote, remoteRoot := storageRuntimeInput(t, 0, false, 0)
	grant, err := testrepo.Open(remoteRoot).Read("authority.json")
	if err != nil {
		t.Fatal(err)
	}
	grant["required_remote_sources"] = []any{"unread-remote-authority"}
	remote["authority_ref"] = sourceRef(t, remoteRoot, "authority.json", grant)
	invoke(t, binary, "STORAGE", "storage.invoke", remote, false)
	for _, condition := range []string{"engine-issued", "expired"} {
		blocked, blockedRoot := storageRuntimeInput(t, 0, false, 0)
		grant, err := testrepo.Open(blockedRoot).Read("authority.json")
		if err != nil {
			t.Fatal(err)
		}
		if condition == "engine-issued" {
			grant["grant_is_engine_issued"] = true
		} else {
			grant["expires_unix_ms"] = time.Now().Add(-time.Second).UnixMilli()
		}
		blocked["authority_ref"] = sourceRef(t, blockedRoot, "authority.json", grant)
		invoke(t, binary, "STORAGE", "storage.invoke", blocked, false)
		if _, err := os.Stat(filepath.Join(blockedRoot, "effect.json")); !os.IsNotExist(err) {
			t.Fatal("inadmissible authority allowed provider effect", condition, err)
		}
	}
}

func TestNativeStorageTimeoutRevocationAndFailedAttemptPreserveUnknownEffects(t *testing.T) {
	binary := native(t, "STORAGE")
	for _, mode := range []string{"timeout", "revoke", "failure", "output-limit"} {
		t.Run(mode, func(t *testing.T) {
			delay := 0
			fail := mode == "failure"
			size := 0
			if mode == "timeout" || mode == "revoke" {
				delay = 2000
			}
			if mode == "output-limit" {
				size = 1_100_000
			}
			input, root := storageRuntimeInput(t, delay, fail, size)
			if mode == "timeout" {
				input["budget_ms"] = 1000
			}
			var revoked chan error
			if mode == "revoke" {
				revoked = make(chan error, 1)
				go func() {
					deadline := time.Now().Add(5 * time.Second)
					for time.Now().Before(deadline) {
						if _, err := os.Stat(filepath.Join(root, "effect.json")); err == nil {
							bytes, err := os.ReadFile(filepath.Join(root, "authority.json"))
							if err != nil {
								revoked <- err
								return
							}
							var grant object
							err = json.Unmarshal(bytes, &grant)
							if err == nil {
								grant["status"] = "REVOKED"
								bytes, err = json.Marshal(grant)
							}
							if err == nil {
								err = os.WriteFile(filepath.Join(root, "authority.json"), bytes, 0600)
							}
							revoked <- err
							return
						}
						time.Sleep(5 * time.Millisecond)
					}
					revoked <- os.ErrDeadlineExceeded
				}()
			}
			output := invoke(t, binary, "STORAGE", "storage.invoke", input, true)
			want := map[string]string{"timeout": "TIMEOUT", "revoke": "AUTHORITY_REVOKED", "failure": "PROVIDER_FAILED", "output-limit": "OUTPUT_LIMIT"}[mode]
			if output["attempt_status"] != "FAILED" || output["cause"] != want || output["effect_status"] != "UNKNOWN" {
				t.Fatal("failed attempt inferred absence of effects", output)
			}
			if mode == "timeout" || mode == "revoke" {
				if output["stop_ack"].(object)["status"] != "ACCEPTED" || output["stop_result"].(object)["status"] != "STOP_CONFIRMED" || output["descendant_stop"] != "NOT_ASSERTED" {
					t.Fatal("direct child stop evidence invalid", output)
				}
			}
			if revoked != nil {
				if err := <-revoked; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "effect.json")); err != nil {
				t.Fatal("effect before failure lost", err)
			}
		})
	}
}

func TestNativeRuntimeInputCannotSelfIssueAuthority(t *testing.T) {
	for _, kind := range []string{"EXECUTION", "STORAGE"} {
		binary := native(t, kind)
		invoke(t, binary, kind, strings.ToLower(kind)+".invoke", object{"grant_write": true, "execution_authorized": true}, false)
	}
}
