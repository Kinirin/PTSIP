package machine

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestAgentNativeContextProjectionParity(t *testing.T) {
	r, err := Open("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := r.Read(contextSourceRef)
	if err != nil {
		t.Fatal(err)
	}
	files, err := RenderContextProjections(source)
	if err != nil {
		t.Fatal(err)
	}
	for ref, expected := range files {
		file, err := r.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		actual = bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n"))
		if !bytes.Equal(actual, expected) {
			t.Fatalf("native projection differs from canonical wire bytes: %s", ref)
		}
	}
	if _, err := ContextProjectionOperation(r, "check", ""); err != nil {
		t.Fatal(err)
	}
}
func TestAgentNativeContextProjectionFailClosed(t *testing.T) {
	r, err := Open("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := r.Read(contextSourceRef)
	if err != nil {
		t.Fatal(err)
	}
	copySource := func() Object { return Map(agentJSON(source)) }
	cases := map[string]func(Object){"mandatory-format-order": func(payload Object) {
		Map(payload["projection_policy"])["mandatory_formats"] = []any{"jsonl", "json", "schema-json"}
	}, "duplicate-memory": func(payload Object) {
		events := List(payload["memory"])
		if len(events) == 0 {
			events = []any{Object{"format": "ptsip-memory-event/v1", "id": "PTSIP-MEM-000001", "recorded_on": "2026-10-06", "type": "TEST", "subject": "Test", "status": "ACTIVE"}}
		}
		payload["memory"] = append(events, events[0])
	}, "unregistered-memory-field": func(payload Object) {
		payload["memory"] = []any{Object{"format": "ptsip-memory-event/v1", "id": "PTSIP-MEM-000001", "recorded_on": "2026-10-06", "type": "TEST", "subject": "Test", "status": "ACTIVE", "unknown": true}}
	}, "nonsemantic-source": func(payload Object) { Map(payload["projection_policy"])["authority"] = "AGENT_CACHE" }}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := copySource()
			mutate(candidate)
			if _, err := RenderContextProjections(candidate); err == nil {
				t.Fatal("invalid context source admitted")
			}
		})
	}
	temp := &Repository{Root: t.TempDir()}
	_ = temp.WriteJSON(".ptsip/index.json", Object{"format": "ptsip-repository-index/v1", "namespaces": Object{"context": Object{"status": "ACTIVE", "root": "context/"}}}, nil)
	_ = temp.WriteJSON("input.json", source, nil)
	if _, err := ContextProjectionOperation(temp, "write", "input.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ContextProjectionOperation(temp, "check", ""); err != nil {
		t.Fatal(err)
	}
	_ = temp.AtomicWrite(contextJSONLRef, []byte("{}\n"), nil)
	if _, err := ContextProjectionOperation(temp, "check", ""); err == nil {
		t.Fatal("JSONL drift accepted")
	}
	if _, err := ContextProjectionOperation(temp, "sync", ""); err != nil {
		t.Fatal(err)
	}
}
func TestAgentContractGraphAndMigrationNative(t *testing.T) {
	r, err := Open("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	counts, err := ValidateAgentContractPlane(r)
	if err != nil {
		t.Fatal(err)
	}
	if agentInt(counts["operations"]) != 5 || agentInt(counts["actions"]) != 18 {
		t.Fatal(counts)
	}
	result, err := VerifyAgentContextMigration(r, "AUTO")
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "PASS" {
		t.Fatalf("migration result %#v", result)
	}
	graph, err := loadAgentContractGraph(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range agentRepresentativeOperations {
		result, err := graph.resolveOperation(id)
		if err != nil {
			t.Fatal(err)
		}
		if len(List(result["rules"])) == 0 || len(Map(result["actions"])) == 0 {
			t.Fatal("bounded operation graph omitted dependencies")
		}
	}
	if _, err := graph.resolveOperation("INVENTED_OPERATION"); err == nil {
		t.Fatal("invented operation admitted")
	}
	condition := graph.byKind["conditions"]["PTSIP-COND-ADOPT-SNAPSHOT-IDENTIFIED-001"]
	condition["evaluation"] = Object{"type": "ALL_OF", "condition_refs": []any{"PTSIP-COND-ADOPT-SNAPSHOT-IDENTIFIED-001"}}
	if err := graph.validate(); err == nil {
		t.Fatal("condition cycle accepted")
	}
}
func TestAgentImplementationReferencesBounded(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	_ = r.AtomicWrite("src/ptsip/bound.py", []byte("def function():\n    pass\n"), nil)
	_ = r.WriteYAML("src/ptsip/agent_contracts/index.yaml", Object{"operations": []any{Object{"ref": "operations/test.yaml"}}}, nil)
	_ = r.WriteYAML("src/ptsip/agent_contracts/operations/test.yaml", Object{"operation_id": "TEST", "implementation_refs": []any{"src/ptsip/bound.py"}}, nil)
	result := VerifyAgentOperationImplementationRefs(r)
	if result["status"] != "PASS" {
		t.Fatal(result)
	}
	_ = r.WriteYAML("src/ptsip/agent_contracts/operations/test.yaml", Object{"operation_id": "TEST", "implementation_refs": []any{"../outside.py"}}, nil)
	if VerifyAgentOperationImplementationRefs(r)["status"] != "FAIL" {
		t.Fatal("path escape admitted")
	}
	_ = r.WriteYAML("src/ptsip/agent_contracts/operations/unindexed.yaml", Object{"operation_id": "UNKNOWN"}, nil)
	if VerifyAgentOperationImplementationRefs(r)["status"] != "FAIL" {
		t.Fatal("unindexed operation admitted")
	}
}
func TestAgentSchemaGraphGuards(t *testing.T) {
	source := Object{"type": "string", "enum": []any{"A", "B"}}
	target := Object{"type": "string", "enum": []any{"A"}}
	if agentSchemaCompatible(source, target) {
		t.Fatal("broader source enum accepted")
	}
	if !agentSchemaCompatible(target, source) {
		t.Fatal("narrower source enum rejected")
	}
	schema := Object{"properties": Object{"value": target}}
	fragment, err := agentSchemaPointer(schema, "/value")
	if err != nil || !reflect.DeepEqual(fragment, target) {
		t.Fatal(fragment, err)
	}
	if _, err := agentSchemaPointer(schema, "/missing"); err == nil {
		t.Fatal("unregistered field accepted")
	}
	if err := agentAcyclic(map[string][]string{"a": {"b"}, "b": {"a"}}, "a"); err == nil {
		t.Fatal("step cycle admitted")
	}
	if err := agentAcyclic(map[string][]string{"a": {}, "b": {}}, "a"); err == nil {
		t.Fatal("unreachable step admitted")
	}
}
