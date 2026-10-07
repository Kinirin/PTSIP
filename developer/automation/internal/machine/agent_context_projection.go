package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const contextSourceRef = ".ptsip/context/source/context.source.json"
const contextJSONRef = ".ptsip/context/context.json"
const contextJSONLRef = ".ptsip/context/context.jsonl"
const contextSchemaRef = ".ptsip/context/context.schema.json"

// ContextProjectionSchema is the original provider-neutral wire schema, generated independently in Go.
func ContextProjectionSchema() Object {
	ref := func(name string) Object { return Object{"$ref": "#/$defs/" + name} }
	object := func(required []string, properties Object) Object {
		return Object{"type": "object", "required": required, "properties": properties, "additionalProperties": false}
	}
	nonempty := Object{"type": "string", "minLength": 1}
	token := Object{"type": "string", "pattern": "^[A-Z][A-Z0-9_]*$"}
	stringArray := Object{"type": "array", "items": nonempty, "uniqueItems": true}
	memory := object([]string{"format", "id", "recorded_on", "type", "subject", "status"}, Object{"format": Object{"const": "ptsip-memory-event/v1"}, "id": Object{"type": "string", "pattern": "^PTSIP-MEM-[0-9]{6}$"}, "recorded_on": Object{"type": "string", "format": "date"}, "type": token, "subject": nonempty, "status": token, "refs": stringArray, "data": Object{"type": "object"}})
	formats := []string{"json", "jsonl", "schema-json"}
	prefix := []any{}
	for _, name := range formats {
		prefix = append(prefix, Object{"const": name})
	}
	providers := Object{"type": "array", "items": nonempty, "uniqueItems": true, "minItems": 1}
	policy := object([]string{"authority", "generated_only", "semantic_equivalence", "mandatory_formats", "provider_scope", "extension_policy"}, Object{"authority": Object{"const": "CANONICAL_SEMANTIC_MODEL"}, "generated_only": Object{"const": true}, "semantic_equivalence": Object{"const": "REQUIRED"}, "mandatory_formats": Object{"type": "array", "prefixItems": prefix, "items": false, "minItems": 3, "maxItems": 3}, "provider_scope": providers, "extension_policy": Object{"const": "EXTENSIBLE"}})
	binding := object([]string{"path", "sha256"}, Object{"path": Object{"const": contextSourceRef}, "sha256": Object{"type": "string", "pattern": "^[0-9a-f]{64}$"}})
	stateFields := []string{"format", "status", "project_profile", "specification", "tool", "planning", "agent_contract", "verification", "context_plane", "legacy_markdown_state"}
	stateProperties := Object{"format": Object{"const": "ptsip-project-state/v1"}, "status": nonempty}
	for _, name := range stateFields[2:] {
		stateProperties[name] = Object{"type": "object", "additionalProperties": true}
	}
	state := object(stateFields, stateProperties)
	state["additionalProperties"] = true
	bundle := object([]string{"format", "source_binding", "projection_policy", "state", "memory"}, Object{"format": Object{"const": "ptsip-context/v1"}, "source_binding": ref("source_binding"), "projection_policy": ref("projection_policy"), "state": ref("state"), "memory": Object{"type": "array", "items": ref("memory_event")}})
	manifest := object([]string{"format", "record_type", "source_binding", "projection_policy"}, Object{"format": Object{"const": "ptsip-context-record/v1"}, "record_type": Object{"const": "manifest"}, "source_binding": ref("source_binding"), "projection_policy": ref("projection_policy")})
	stateRecord := object([]string{"format", "record_type", "payload"}, Object{"format": Object{"const": "ptsip-context-record/v1"}, "record_type": Object{"const": "state"}, "payload": ref("state")})
	memoryRecord := object([]string{"format", "record_type", "ordinal", "payload"}, Object{"format": Object{"const": "ptsip-context-record/v1"}, "record_type": Object{"const": "memory"}, "ordinal": Object{"type": "integer", "minimum": 0}, "payload": ref("memory_event")})
	return Object{"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "https://github.com/Kinirin/PTSIP/.ptsip/context/context.schema.json", "title": "PTSIP Provider-Neutral Context Plane", "$defs": Object{"projection_policy": policy, "source_binding": binding, "state": state, "memory_event": memory, "context_bundle": bundle, "manifest_record": manifest, "state_record": stateRecord, "memory_record": memoryRecord, "context_record": Object{"oneOf": []any{ref("manifest_record"), ref("state_record"), ref("memory_record")}}}, "oneOf": []any{ref("context_bundle"), ref("context_record")}}
}
func agentPrettyJSON(value any) ([]byte, error) {
	data, err := CanonicalJSON(value)
	if err != nil {
		return nil, err
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, data, "", "  "); err != nil {
		return nil, err
	}
	return append(formatted.Bytes(), '\n'), nil
}
func contextValidateSource(source Object) error {
	if source["format"] != "ptsip-context-source/v1" {
		return fmt.Errorf("context source format must be ptsip-context-source/v1")
	}
	state := Map(source["state"])
	if state["format"] != "ptsip-project-state/v1" {
		return fmt.Errorf("state format must be ptsip-project-state/v1")
	}
	memory, ok := source["memory"].([]any)
	if !ok {
		return fmt.Errorf("context source memory must be an array")
	}
	ids := map[string]bool{}
	for i, raw := range memory {
		event := Map(raw)
		if event["format"] != "ptsip-memory-event/v1" {
			return fmt.Errorf("memory[%d] format must be ptsip-memory-event/v1", i)
		}
		id := Text(event["id"])
		if id == "" || ids[id] {
			return fmt.Errorf("memory event ids must be unique non-empty strings")
		}
		ids[id] = true
	}
	policy := Map(source["projection_policy"])
	if policy["authority"] != "CANONICAL_SEMANTIC_MODEL" || policy["generated_only"] != true || policy["semantic_equivalence"] != "REQUIRED" || policy["extension_policy"] != "EXTENSIBLE" {
		return fmt.Errorf("context projection policy is invalid")
	}
	if !agentEqualStrings(policy["mandatory_formats"], []string{"json", "jsonl", "schema-json"}) {
		return fmt.Errorf("mandatory_formats must be exactly json, jsonl, schema-json")
	}
	if len(List(policy["provider_scope"])) == 0 {
		return fmt.Errorf("provider_scope must be a non-empty array")
	}
	return nil
}
func DecodeContextJSONL(records []Object) (Object, error) {
	if len(records) < 2 || records[0]["record_type"] != "manifest" || records[1]["record_type"] != "state" {
		return nil, fmt.Errorf("context.jsonl must begin with manifest and state records")
	}
	memory := []any{}
	for i, record := range records[2:] {
		if record["record_type"] != "memory" || agentInt(record["ordinal"]) != i || Map(record["payload"]) == nil {
			return nil, fmt.Errorf("context.jsonl memory ordinals must be contiguous")
		}
		memory = append(memory, record["payload"])
	}
	return Object{"format": "ptsip-context/v1", "source_binding": records[0]["source_binding"], "projection_policy": records[0]["projection_policy"], "state": records[1]["payload"], "memory": memory}, nil
}
func RenderContextProjections(source Object) (map[string][]byte, error) {
	if err := contextValidateSource(source); err != nil {
		return nil, err
	}
	sourceBytes, err := agentPrettyJSON(source)
	if err != nil {
		return nil, err
	}
	binding := Object{"path": contextSourceRef, "sha256": SHA256(sourceBytes)}
	bundle := Object{"format": "ptsip-context/v1", "source_binding": binding, "projection_policy": source["projection_policy"], "state": source["state"], "memory": source["memory"]}
	records := []Object{{"format": "ptsip-context-record/v1", "record_type": "manifest", "source_binding": binding, "projection_policy": source["projection_policy"]}, {"format": "ptsip-context-record/v1", "record_type": "state", "payload": source["state"]}}
	for i, event := range List(source["memory"]) {
		records = append(records, Object{"format": "ptsip-context-record/v1", "record_type": "memory", "ordinal": i, "payload": event})
	}
	schema := ContextProjectionSchema()
	compiler := jsonschema.NewCompiler()
	compiler.UseRegexpEngine(compileSchemaRegexp)
	compiler.UseLoader(closedLoader{})
	if err := compiler.AddResource(Text(schema["$id"]), agentJSON(schema)); err != nil {
		return nil, err
	}
	compiled, err := compiler.Compile(Text(schema["$id"]))
	if err != nil {
		return nil, err
	}
	if err := compiled.Validate(agentJSON(bundle)); err != nil {
		return nil, fmt.Errorf("context projection schema validation failed: %w", err)
	}
	jsonl := []byte{}
	for _, record := range records {
		if err := compiled.Validate(agentJSON(record)); err != nil {
			return nil, err
		}
		line, err := CanonicalJSON(record)
		if err != nil {
			return nil, err
		}
		jsonl = append(jsonl, line...)
		jsonl = append(jsonl, '\n')
	}
	decoded, err := DecodeContextJSONL(records)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(agentJSON(decoded), agentJSON(bundle)) {
		return nil, fmt.Errorf("JSON and JSONL projections are not semantically equivalent")
	}
	bundleBytes, err := agentPrettyJSON(bundle)
	if err != nil {
		return nil, err
	}
	schemaBytes, err := agentPrettyJSON(schema)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{contextSourceRef: sourceBytes, contextJSONRef: bundleBytes, contextJSONLRef: jsonl, contextSchemaRef: schemaBytes}, nil
}

func ContextProjectionOperation(r *Repository, command, input string) (Object, error) {
	// Repository-local context capability selection precedes using its bounded machine plane.
	index, err := r.Read(".ptsip/index.json")
	if err != nil {
		return nil, err
	}
	namespace := Map(Map(index["namespaces"])["context"])
	if index["format"] != "ptsip-repository-index/v1" || namespace["status"] != "ACTIVE" || namespace["root"] != "context/" {
		return nil, fmt.Errorf("repository-local context namespace is not exactly registered")
	}
	source := contextSourceRef
	if command == "write" {
		if input == "" {
			return nil, fmt.Errorf("context write requires input")
		}
		source = input
	} else if command != "check" && command != "sync" {
		return nil, fmt.Errorf("unsupported context command %s", command)
	}
	payload, err := r.Read(source)
	if err != nil {
		return nil, err
	}
	files, err := RenderContextProjections(payload)
	if err != nil {
		return nil, err
	}
	drift := []string{}
	if command == "check" {
		for ref, expected := range files {
			path, err := r.Path(ref)
			if err != nil {
				return nil, err
			}
			actual, err := os.ReadFile(path)
			if err != nil {
				drift = append(drift, ref+": MISSING")
				continue
			}
			normalized := func(data []byte) []byte {
				return bytes.ReplaceAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))
			}
			if !bytes.Equal(normalized(actual), normalized(expected)) {
				drift = append(drift, ref+": DRIFT")
			}
		}
		if len(drift) > 0 {
			return nil, fmt.Errorf("context projection check failed: %s", strings.Join(drift, "; "))
		}
	} else {
		for ref, data := range files {
			if err := r.AtomicWrite(ref, data, nil); err != nil {
				return nil, err
			}
		}
	}
	return Object{"status": "PASS", "command": command, "source": contextSourceRef, "projections": []string{contextJSONRef, contextJSONLRef, contextSchemaRef}}, nil
}
