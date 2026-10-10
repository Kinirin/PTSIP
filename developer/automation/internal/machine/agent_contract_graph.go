package machine

import (
	"fmt"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// candidateValue resolves JSON pointers only; labels and source text never select semantics.
func candidateValue(document any, pointer string) (any, error) {
	if pointer == "" {
		return document, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid candidate pointer %q", pointer)
	}
	for _, raw := range strings.Split(pointer[1:], "/") {
		key := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		if object := Map(document); object != nil {
			var ok bool
			document, ok = object[key]
			if !ok {
				return nil, fmt.Errorf("unresolved candidate pointer %q", pointer)
			}
		} else if array := List(document); array != nil {
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(array) {
				return nil, fmt.Errorf("unresolved candidate array pointer %q", pointer)
			}
			document = array[index]
		} else {
			return nil, fmt.Errorf("unresolved candidate pointer %q", pointer)
		}
	}
	return document, nil
}

func candidateSchema(value Object, identity string) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(closedLoader{})
	if err := compiler.AddResource(identity, value); err != nil {
		return nil, err
	}
	return compiler.Compile(identity)
}

// VerifyContextEvidenceCandidate interprets the approved neutral P0/P1 candidate.
// JSON Schema owns gates and output shape. Synthetic proof facts never become
// runtime proof, policy activation, record retention duties, or an ID assignment.
func VerifyContextEvidenceCandidate(r *Repository) (Object, error) {
	result, err := verifyContextEvidenceCandidate(r)
	if err != nil {
		return Object{"status": "UNRESOLVED", "stage": "CONTEXT_EVIDENCE_CANDIDATE", "reason_code": "CANDIDATE_REQUIRED_CONTRACT_UNRESOLVED", "detail": err.Error(), "normative_authority": false, "mutation_authorized": false, "assignment_performed": false}, nil
	}
	return result, nil
}

func verifyContextEvidenceCandidate(r *Repository) (Object, error) {
	const base = "src/agent_contracts/"
	module, err := r.Read(base + "contracts/context-evidence-identity.json")
	if err != nil {
		return nil, err
	}
	definition, err := r.Read(base + "schemas/candidate-context-evidence.schema.json")
	if err != nil {
		return nil, err
	}
	schema, err := candidateSchema(definition, "urn:ptsip:candidate:module")
	if err != nil {
		return nil, err
	}
	if err = schema.Validate(module); err != nil {
		return nil, err
	}
	planRef := Text(module["plan_ref"])
	plan, err := r.Read(planRef)
	if err != nil {
		return nil, err
	}
	if Map(plan["approval"])["status"] != "APPROVED" || (Map(plan["implementation_authorization"])["status"] != "AUTHORIZED" && Map(plan["implementation_authorization"])["status"] != "COMPLETE") {
		return nil, fmt.Errorf("candidate plan implementation is not authorized")
	}
	parent, err := r.Read("developer/planning/0.3.8/0.3.8a3/WU-03/WU-03.yaml")
	if err != nil {
		return nil, err
	}
	if failures := ExtensionParentConsistency(parent, plan, Text(Map(plan["extension"])["id"]), planRef); len(failures) > 0 {
		return nil, fmt.Errorf("candidate plan registration: %v", failures)
	}
	bindings, err := r.ResolveBindings(Object{"plan_ref": planRef})
	if err != nil {
		return nil, err
	}
	if bindings["status"] != "BOUND" {
		return nil, fmt.Errorf("candidate plan has no exact external binding")
	}
	boundRows := List(bindings["bindings"])
	if len(boundRows) != 1 {
		return nil, fmt.Errorf("ambiguous candidate plan binding")
	}
	for _, field := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
		if Map(boundRows[0])[field] != Map(plan["plan_identity"])[field] {
			return nil, fmt.Errorf("candidate plan binding identity drift: %s", field)
		}
	}
	inventory := Map(module["inventory"])
	index, err := r.Read(Text(inventory["policy_index_ref"]))
	if err != nil {
		return nil, err
	}
	registered := map[string]Object{}
	for _, raw := range List(index["policies"]) {
		row := Map(raw)
		if registered[Text(row["id"])] != nil {
			return nil, fmt.Errorf("ambiguous Support policy registration: %s", row["id"])
		}
		registered[Text(row["id"])] = row
	}
	sources := map[string]Object{}
	for identity, raw := range Map(inventory["policy_revisions"]) {
		revision, row := Map(raw), registered[identity]
		if row == nil || "src/policy/"+Text(row["path"]) != revision["path"] || row["status"] != revision["status"] || row["authority_role"] != "CANONICAL_AUTHORITY" {
			return nil, fmt.Errorf("unresolved exact candidate policy registration: %s", identity)
		}
		value, err := r.Read(Text(revision["path"]))
		if err != nil {
			return nil, err
		}
		bytes, err := CanonicalJSON(value)
		if err != nil {
			return nil, err
		}
		if Map(value["policy"])["id"] != identity || Map(value["policy"])["status"] != revision["status"] || SHA256(bytes) != revision["validation_fingerprint"] {
			return nil, fmt.Errorf("policy revalidation required: %s", identity)
		}
		sources[identity] = value
	}
	for _, raw := range Map(module["policy_bindings"]) {
		binding := Map(raw)
		value, err := candidateValue(sources[Text(binding["policy_id"])], Text(binding["json_pointer"]))
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(value, binding["expected"]) {
			return nil, fmt.Errorf("candidate policy binding drift: %s", binding["json_pointer"])
		}
	}
	vectorRef := Text(module["conformance_ref"])
	if !agentSafeRef(vectorRef) {
		return nil, fmt.Errorf("unsafe candidate vector reference")
	}
	vectors, err := r.Read(base + vectorRef)
	if err != nil {
		return nil, err
	}
	vectorSchema, err := candidateSchema(Map(Map(definition["$defs"])["conformance"]), "urn:ptsip:candidate:vectors")
	if err != nil {
		return nil, err
	}
	if err = vectorSchema.Validate(vectors); err != nil {
		return nil, err
	}
	inputSchema, err := candidateSchema(Map(module["input_schema"]), "urn:ptsip:candidate:input")
	if err != nil {
		return nil, err
	}
	outputSchema, err := candidateSchema(Map(module["output_schema"]), "urn:ptsip:candidate:output")
	if err != nil {
		return nil, err
	}
	gateSchemas := map[string]*jsonschema.Schema{}
	for identity, raw := range Map(module["gates"]) {
		gate := Map(raw)
		gateSchemas[identity], err = candidateSchema(Map(gate["input_constraint"]), "urn:ptsip:candidate:gate:"+identity)
		if err != nil {
			return nil, err
		}
		if _, err = candidateValue(module, strings.TrimPrefix(Text(gate["failure_semantics_ref"]), "#")); err != nil {
			return nil, err
		}
		for _, binding := range Strings(gate["policy_binding_refs"]) {
			if Map(module["policy_bindings"])[binding] == nil {
				return nil, fmt.Errorf("unresolved gate policy binding")
			}
		}
	}
	primitives, operations, cases := Map(module["primitives"]), Map(module["operations"]), Map(vectors["operations"])
	if len(primitives) != len(operations) || len(cases) != len(primitives) {
		return nil, fmt.Errorf("candidate primitive/vector coverage mismatch")
	}
	identities := []string{}
	for identity := range primitives {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	results, projections := []any{}, []any{}
	for _, identity := range identities {
		primitive, operation := Map(primitives[identity]), Map(operations[identity])
		if operation == nil || len(List(cases[identity])) == 0 || primitive["primitive_id"] != identity {
			return nil, fmt.Errorf("unresolved exact candidate identity: %s", identity)
		}
		for _, role := range []string{"input_schema_ref", "output_schema_ref", "operation_semantics_ref", "failure_semantics_ref", "conformance_vectors_ref"} {
			parts := strings.SplitN(Text(primitive[role]), "#", 2)
			if len(parts) != 2 || (parts[0] != "" && parts[0] != vectorRef) {
				return nil, fmt.Errorf("unregistered primitive reference")
			}
			var document any = module
			if parts[0] == vectorRef {
				document = vectors
			}
			if _, err = candidateValue(document, parts[1]); err != nil {
				return nil, err
			}
		}
		seen := map[string]bool{}
		for _, raw := range List(cases[identity]) {
			vector := Map(raw)
			name := Text(vector["id"])
			if name == "" || seen[name] {
				return nil, fmt.Errorf("invalid or duplicate candidate vector")
			}
			seen[name] = true
			for _, binding := range Strings(vector["policy_binding_refs"]) {
				if Map(module["policy_bindings"])[binding] == nil {
					return nil, fmt.Errorf("unresolved vector policy binding %s", binding)
				}
			}
			request := Map(vector["input"])
			if err = inputSchema.Validate(request); err != nil {
				return nil, err
			}
			failed := []string{}
			for _, gate := range Strings(operation["gate_refs"]) {
				checker := gateSchemas[gate]
				if checker == nil {
					return nil, fmt.Errorf("unresolved candidate gate %s", gate)
				}
				if checker.Validate(request) != nil {
					failed = append(failed, gate)
				}
			}
			sort.Strings(failed)
			failedValues := []any{}
			for _, gate := range failed {
				failedValues = append(failedValues, gate)
			}
			action, reference := operation["projected_action"], request[Text(operation["reference_id_field"])]
			if len(failed) > 0 {
				failure := Map(Map(Map(module["operation_failures"])[identity])["on_gate_failure"])
				action, reference = failure["projected_action"], failure["reference_id"]
			}
			result := Object{"status": "CANDIDATE_EVALUATED", "projected_action": action, "reference_id": reference, "failed_gates": failedValues, "normative_authority": false, "mutation_authorized": false, "assignment_performed": false}
			result[Text(operation["scope_output_field"])] = request[Text(operation["scope_input_field"])]
			if err = outputSchema.Validate(result); err != nil {
				return nil, err
			}
			expected := Object{"projected_action": action, "reference_id": reference, "failed_gates": failed, Text(operation["scope_output_field"]): request[Text(operation["scope_input_field"])]}
			left, _ := CanonicalJSON(expected)
			right, _ := CanonicalJSON(vector["expected"])
			if string(left) != string(right) {
				return nil, fmt.Errorf("candidate conformance failed: %s", name)
			}
			results = append(results, Object{"primitive_id": identity, "vector_id": name, "result": result})
		}
		projections = append(projections, Object{"primitive": primitive, "operation": operation, "input_schema": module["input_schema"], "output_schema": module["output_schema"], "failure_semantics": Map(module["operation_failures"])[identity]})
	}
	return Object{"status": "PASS", "stage": module["verification_stage"], "module_id": module["module_id"], "policy_revisions": inventory["policy_revisions"], "projections": projections, "results": results, "vectors_checked": len(results), "activation_blockers": module["activation_blockers"], "normative_authority": false, "mutation_authorized": false, "assignment_performed": false}, nil
}

type agentContractGraph struct {
	repo           *Repository
	index, binding Object
	byKind         map[string]map[string]Object
	refs           map[string]map[string]string
	rules          map[string]Object
	ruleOwners     map[string]string
	counts         Object
}

func agentSafeRef(ref string) bool {
	return ref != "" && !path.IsAbs(ref) && !strings.Contains(ref, "\\") && !strings.Contains(ref, ":") && path.Clean(ref) == ref && ref != ".." && !strings.HasPrefix(ref, "../")
}
func agentAllStrings(value any) []string {
	result := []string{}
	switch v := value.(type) {
	case string:
		result = append(result, v)
	case map[string]any:
		for _, child := range v {
			result = append(result, agentAllStrings(child)...)
		}
	case []any:
		for _, child := range v {
			result = append(result, agentAllStrings(child)...)
		}
	}
	return result
}
func agentRejectMarkdown(payload Object, ref string) error {
	for _, text := range agentAllStrings(payload) {
		lower := strings.ToLower(text)
		if strings.HasSuffix(lower, ".md") || strings.Contains(lower, ".md#") {
			return fmt.Errorf("Markdown normative dependency is forbidden in %s: %s", ref, text)
		}
	}
	return nil
}
func loadAgentContractGraph(r *Repository) (*agentContractGraph, error) {
	const base = "src/agent_contracts"
	index, err := r.Read(path.Join(base, "index.yaml"))
	if err != nil {
		return nil, err
	}
	schemas := Map(index["schemas"])
	if schemas == nil {
		return nil, fmt.Errorf("Agent Contract schemas missing")
	}
	if !agentSafeRef(Text(schemas["index"])) {
		return nil, fmt.Errorf("unsafe index schema ref")
	}
	if err := r.Validate(path.Join(base, Text(schemas["index"])), index); err != nil {
		return nil, err
	}
	if err := agentRejectMarkdown(index, "index.yaml"); err != nil {
		return nil, err
	}
	graph := &agentContractGraph{repo: r, index: index, byKind: map[string]map[string]Object{}, refs: map[string]map[string]string{}, rules: map[string]Object{}, ruleOwners: map[string]string{}, counts: Object{}}
	identities := map[string]bool{}
	config := map[string][2]string{"specs": {"spec", "id"}, "operations": {"operation", "operation_id"}, "actions": {"action", "action_id"}, "conditions": {"condition", "condition_id"}, "gates": {"gate", "gate_id"}, "io_schemas": {"io", "x-ptsip-io-id"}, "vocabularies": {"vocabulary", "vocabulary_id"}}
	for _, kind := range []string{"specs", "operations", "actions", "conditions", "gates", "io_schemas", "vocabularies"} {
		settings := config[kind]
		graph.byKind[kind] = map[string]Object{}
		graph.refs[kind] = map[string]string{}
		entries, ok := index[kind].([]any)
		if !ok {
			return nil, fmt.Errorf("index %s must be a list", kind)
		}
		schemaRef := Text(schemas[settings[0]])
		if !agentSafeRef(schemaRef) {
			return nil, fmt.Errorf("unsafe %s schema ref", kind)
		}
		for _, raw := range entries {
			entry := Map(raw)
			id, ref := Text(entry["id"]), Text(entry["ref"])
			if id == "" || identities[id] || !agentSafeRef(ref) {
				return nil, fmt.Errorf("duplicate or unsafe contract %s", id)
			}
			identities[id] = true
			payload, err := r.Read(path.Join(base, ref))
			if err != nil {
				return nil, err
			}
			if err := r.Validate(path.Join(base, schemaRef), payload); err != nil {
				return nil, fmt.Errorf("%s: %w", ref, err)
			}
			if err := agentRejectMarkdown(payload, ref); err != nil {
				return nil, err
			}
			if payload[settings[1]] != id {
				return nil, fmt.Errorf("%s index identity mismatch", ref)
			}
			graph.byKind[kind][id] = payload
			graph.refs[kind][id] = ref
		}
		graph.counts[kind] = len(graph.byKind[kind])
	}
	for id, spec := range graph.byKind["specs"] {
		for _, raw := range List(spec["rules"]) {
			rule := Map(raw)
			ruleID := Text(rule["rule_id"])
			if ruleID == "" || identities[ruleID] || graph.rules[ruleID] != nil {
				return nil, fmt.Errorf("duplicate or colliding rule id %s", ruleID)
			}
			graph.rules[ruleID] = rule
			graph.ruleOwners[ruleID] = id
		}
	}
	bindingRef := Text(Map(index["binding"])["current_ref"])
	if !agentSafeRef(bindingRef) {
		return nil, fmt.Errorf("unsafe binding ref")
	}
	binding, err := r.Read(path.Join(base, bindingRef))
	if err != nil {
		return nil, err
	}
	if err := r.Validate(path.Join(base, Text(schemas["binding"])), binding); err != nil {
		return nil, err
	}
	if err := agentRejectMarkdown(binding, bindingRef); err != nil {
		return nil, err
	}
	graph.binding = binding
	graph.counts["rules"] = len(graph.rules)
	graph.counts["bindings"] = 1
	return graph, nil
}
func (g *agentContractGraph) payload(kind, id string) (Object, error) {
	value := g.byKind[kind][id]
	if value == nil {
		return nil, fmt.Errorf("unresolved %s reference: %s", kind, id)
	}
	return value, nil
}
func agentSchemaPointer(schema Object, pointer string) (Object, error) {
	if pointer == "" {
		return schema, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid JSON pointer %s", pointer)
	}
	current := schema
	for _, raw := range strings.Split(pointer[1:], "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		current = Map(Map(current["properties"])[token])
		if current == nil {
			return nil, fmt.Errorf("JSON pointer %s is not declared", pointer)
		}
	}
	return current, nil
}
func agentSchemaCompatible(source, target Object) bool {
	if source["type"] != nil && target["type"] != nil && !reflect.DeepEqual(source["type"], target["type"]) {
		return false
	}
	sourceEnum, targetEnum := List(source["enum"]), List(target["enum"])
	if sourceEnum != nil && targetEnum != nil {
		for _, value := range sourceEnum {
			hit := false
			for _, allowed := range targetEnum {
				hit = hit || reflect.DeepEqual(value, allowed)
			}
			if !hit {
				return false
			}
		}
	}
	return true
}

var agentDataRef = regexp.MustCompile(`^\$(input|step:([A-Za-z0-9_-]+))#(.*)$`)

func agentConditionChildren(condition Object) []string {
	evaluation := Map(condition["evaluation"])
	switch evaluation["type"] {
	case "ALL_OF", "ANY_OF":
		return Strings(evaluation["condition_refs"])
	case "NOT":
		return []string{Text(evaluation["condition_ref"])}
	}
	return []string{}
}
func agentAcyclic(graph map[string][]string, entry string) error {
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("reference cycle at %s", id)
		}
		children, exists := graph[id]
		if !exists {
			return fmt.Errorf("unresolved graph node %s", id)
		}
		visiting[id] = true
		for _, child := range children {
			if err := visit(child); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	if entry == "" {
		for id := range graph {
			if err := visit(id); err != nil {
				return err
			}
		}
	} else {
		if err := visit(entry); err != nil {
			return err
		}
		if len(visited) != len(graph) {
			return fmt.Errorf("operation contains unreachable steps")
		}
	}
	return nil
}

func (g *agentContractGraph) validate() error {
	outcomes := map[string]bool{}
	for _, raw := range List(g.byKind["vocabularies"]["PTSIP-VOCAB-OUTCOMES"]["entries"]) {
		outcomes[Text(Map(raw)["id"])] = true
	}
	requireOutcome := func(value any) error {
		if !outcomes[Text(value)] {
			return fmt.Errorf("undefined outcome %v", value)
		}
		return nil
	}
	conditions := g.byKind["conditions"]
	conditionGraph := map[string][]string{}
	for id, condition := range conditions {
		input, err := g.payload("io_schemas", Text(condition["input_schema_ref"]))
		if err != nil {
			return err
		}
		evaluation := Map(condition["evaluation"])
		if evaluation["type"] == "FIELD_COMPARE" {
			fragment, err := agentSchemaPointer(input, Text(evaluation["pointer"]))
			if err != nil {
				return err
			}
			compiler := jsonschema.NewCompiler()
			compiler.UseLoader(closedLoader{})
			if err := compiler.AddResource("urn:ptsip:agent:fragment", fragment); err != nil {
				return err
			}
			compiled, err := compiler.Compile("urn:ptsip:agent:fragment")
			if err != nil {
				return err
			}
			if err := compiled.Validate(evaluation["expected"]); err != nil {
				return fmt.Errorf("condition %s expected value fails schema: %w", id, err)
			}
		}
		children := agentConditionChildren(condition)
		for _, childID := range children {
			child := conditions[childID]
			if child == nil || child["input_schema_ref"] != condition["input_schema_ref"] {
				return fmt.Errorf("condition %s child schema mismatch", id)
			}
		}
		conditionGraph[id] = children
	}
	if err := agentAcyclic(conditionGraph, ""); err != nil {
		return err
	}
	checks := func(items any, schema string, allowed []string) error {
		for _, raw := range List(items) {
			check := Map(raw)
			condition := conditions[Text(check["condition_ref"])]
			if condition == nil || condition["input_schema_ref"] != schema {
				return fmt.Errorf("condition schema reference mismatch")
			}
			for _, key := range []string{"on_false", "on_unknown"} {
				if err := requireOutcome(check[key]); err != nil {
					return err
				}
				if allowed != nil && !Has(allowed, Text(check[key])) {
					return fmt.Errorf("terminal outcome not declared")
				}
			}
		}
		return nil
	}
	for id, action := range g.byKind["actions"] {
		for _, field := range []string{"input_schema_ref", "output_schema_ref"} {
			if _, err := g.payload("io_schemas", Text(action[field])); err != nil {
				return err
			}
		}
		failures := Strings(action["failure_outcomes"])
		for _, outcome := range failures {
			if err := requireOutcome(outcome); err != nil {
				return err
			}
		}
		if err := checks(action["preconditions"], Text(action["input_schema_ref"]), failures); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		if err := checks(action["postconditions"], Text(action["output_schema_ref"]), failures); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
	}
	for _, gate := range g.byKind["gates"] {
		if _, err := g.payload("io_schemas", Text(gate["input_schema_ref"])); err != nil {
			return err
		}
		if err := checks(gate["requirements"], Text(gate["input_schema_ref"]), nil); err != nil {
			return err
		}
	}
	for id, operation := range g.byKind["operations"] {
		if err := g.validateOperation(operation, outcomes, checks); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
	}
	current := Map(g.index["contract_set"])["status"] == "CURRENT"
	if current && g.binding["status"] != "CURRENT" {
		return fmt.Errorf("CURRENT Agent Contract set requires CURRENT binding")
	}
	activeFields := map[string]string{"active_specs": "specs", "active_operations": "operations", "active_actions": "actions", "active_conditions": "conditions", "active_gates": "gates", "active_io_schemas": "io_schemas", "active_vocabularies": "vocabularies"}
	for field, kind := range activeFields {
		ids := Strings(g.binding[field])
		if len(UniqueStrings(ids)) != len(g.byKind[kind]) {
			return fmt.Errorf("binding %s mismatch", field)
		}
		for id, payload := range g.byKind[kind] {
			if !Has(ids, id) {
				return fmt.Errorf("binding %s missing %s", field, id)
			}
			if current && kind != "io_schemas" && payload["status"] != "CURRENT" {
				return fmt.Errorf("CURRENT set contains non-CURRENT %s", id)
			}
		}
	}
	for _, config := range [][3]string{{"action_bindings", "action_id", "actions"}, {"condition_evaluator_bindings", "condition_id", "conditions"}, {"operation_entrypoints", "operation_id", "operations"}} {
		seen := map[string]bool{}
		for _, raw := range List(g.binding[config[0]]) {
			item := Map(raw)
			id := Text(item[config[1]])
			if seen[id] || g.byKind[config[2]][id] == nil {
				return fmt.Errorf("duplicate or inactive implementation binding %s", id)
			}
			seen[id] = true
			if config[0] != "operation_entrypoints" {
				if config[0] == "condition_evaluator_bindings" && Map(conditions[id]["evaluation"])["type"] != "BOUND_EVALUATOR" {
					return fmt.Errorf("unexpected evaluator binding")
				}
				if err := validateAgentCallableDeclaration(g.repo, Text(item["python_callable"])); err != nil {
					return err
				}
			}
		}
		if config[0] == "action_bindings" && len(seen) != len(g.byKind["actions"]) {
			return fmt.Errorf("every active action must have one binding")
		}
		if config[0] == "condition_evaluator_bindings" {
			for id, condition := range conditions {
				if (Map(condition["evaluation"])["type"] == "BOUND_EVALUATOR") != seen[id] {
					return fmt.Errorf("bound evaluator coverage mismatch %s", id)
				}
			}
		}
	}
	for _, raw := range List(g.binding["external_machine_contracts"]) {
		contract := Map(raw)
		packageName, resource := Text(contract["package"]), Text(contract["resource"])
		if packageName != "ptsip" || !agentSafeRef(resource) {
			return fmt.Errorf("unregistered external machine contract resource")
		}
		payload, err := g.repo.Read(path.Join("src/ptsip", resource))
		if err != nil {
			return err
		}
		if contract["contract_id"] == "PTSIP_PROJECT_PROFILE_CONTRACT_IDENTITY" && payload["current"] != contract["version"] {
			return fmt.Errorf("Project Profile current identity mismatch")
		}
	}
	return nil
}

func validateAgentCallableDeclaration(r *Repository, locator string) error {
	parts := strings.SplitN(locator, ":", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "ptsip.") {
		return fmt.Errorf("invalid registered callable locator %s", locator)
	}
	moduleName := parts[0]
	file, err := resolveAgentPythonModule(r, moduleName)
	if err != nil {
		return err
	}
	attributes := strings.Split(parts[1], ".")
	selector := Object{"kind": "PYTHON_FUNCTION", "name": parts[1]}
	if len(attributes) == 2 {
		selector = Object{"kind": "PYTHON_METHOD", "class": attributes[0], "method": attributes[1]}
	} else if len(attributes) != 1 {
		return fmt.Errorf("unsupported exact callable attribute path %s", locator)
	}
	_, err = ValidateImplementationRef(r, Object{"path": file, "selector": selector})
	if err == nil {
		return nil
	}
	if len(attributes) != 1 {
		return err
	}
	return resolveAgentImportedCallable(r, moduleName, attributes[0], map[string]bool{})
}

func resolveAgentPythonModule(r *Repository, module string) (string, error) {
	if !strings.HasPrefix(module, "ptsip.") {
		return "", fmt.Errorf("Python callable package not registered: %s", module)
	}
	ref := path.Join("src", strings.ReplaceAll(module, ".", "/"))
	file, packageRef := ref+".py", path.Join(ref, "__init__.py")
	fileExists, packageExists := iwpPathExists(r, file), iwpPathExists(r, packageRef)
	if fileExists && packageExists {
		return "", fmt.Errorf("ambiguous Python module source: %s", module)
	}
	if packageExists {
		return packageRef, nil
	}
	if fileExists {
		return file, nil
	}
	return "", fmt.Errorf("Python module source missing: %s", module)
}
func resolveAgentImportedCallable(r *Repository, module, name string, seen map[string]bool) error {
	identity := module + ":" + name
	if seen[identity] {
		return fmt.Errorf("Python callable import cycle: %s", identity)
	}
	seen[identity] = true
	file, err := resolveAgentPythonModule(r, module)
	if err != nil {
		return err
	}
	source, err := agentText(r, file)
	if err != nil {
		return err
	}
	tree, lang, err := pythonTree([]byte(source))
	if err != nil {
		return err
	}
	defer tree.Release()
	root := tree.RootNode()
	declarations := pythonNamedDeclarations(root, lang, []byte(source), "function_definition", name)
	if len(declarations) == 1 {
		return nil
	}
	if len(declarations) > 1 {
		return fmt.Errorf("duplicate Python callable declaration: %s", identity)
	}
	matches := [][2]string{}
	for i := 0; i < root.NamedChildCount(); i++ {
		node := root.NamedChild(i)
		if node.Type(lang) != "import_from_statement" {
			continue
		}
		moduleNode := node.ChildByFieldName("module_name", lang)
		if moduleNode == nil {
			continue
		}
		importModule := moduleNode.Text([]byte(source))
		for j := 0; j < node.ChildCount(); j++ {
			if node.FieldNameForChild(j, lang) != "name" {
				continue
			}
			item := node.Child(j)
			original, alias := item.Text([]byte(source)), item.Text([]byte(source))
			if item.Type(lang) == "aliased_import" {
				originalNode := item.ChildByFieldName("name", lang)
				aliasNode := item.ChildByFieldName("alias", lang)
				if originalNode == nil || aliasNode == nil {
					continue
				}
				original, alias = originalNode.Text([]byte(source)), aliasNode.Text([]byte(source))
			}
			if alias == name {
				matches = append(matches, [2]string{importModule, original})
			}
		}
	}
	if len(matches) != 1 {
		return fmt.Errorf("callable %s cannot resolve exactly as declaration or explicit import", identity)
	}
	targetModule, targetName := matches[0][0], matches[0][1]
	if strings.HasPrefix(targetModule, ".") {
		dots := len(targetModule) - len(strings.TrimLeft(targetModule, "."))
		base := strings.Split(module, ".")
		if !strings.HasSuffix(file, "/__init__.py") {
			base = base[:len(base)-1]
		}
		remove := dots - 1
		if remove >= len(base) {
			return fmt.Errorf("Python relative import escapes registered package")
		}
		base = base[:len(base)-remove]
		suffix := strings.TrimLeft(targetModule, ".")
		targetModule = strings.Join(base, ".")
		if suffix != "" {
			targetModule += "." + suffix
		}
	}
	return resolveAgentImportedCallable(r, targetModule, targetName, seen)
}

func (g *agentContractGraph) validateOperation(operation Object, outcomes map[string]bool, checks func(any, string, []string) error) error {
	inputID, outputID := Text(operation["input_schema_ref"]), Text(operation["output_schema_ref"])
	input, err := g.payload("io_schemas", inputID)
	if err != nil {
		return err
	}
	output, err := g.payload("io_schemas", outputID)
	if err != nil {
		return err
	}
	allowed := Strings(operation["outcomes"])
	for _, id := range allowed {
		if !outcomes[id] {
			return fmt.Errorf("undefined operation outcome %s", id)
		}
	}
	fragment, err := agentSchemaPointer(output, "/outcome")
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(iwpSortedUnique(Strings(fragment["enum"])), iwpSortedUnique(allowed)) {
		return fmt.Errorf("operation outcomes do not match output enum")
	}
	for _, id := range Strings(operation["vocabulary_refs"]) {
		if _, err := g.payload("vocabularies", id); err != nil {
			return err
		}
	}
	for _, id := range Strings(operation["rule_refs"]) {
		if g.rules[id] == nil {
			return fmt.Errorf("unresolved operation rule %s", id)
		}
	}
	if err := checks(operation["preconditions"], inputID, allowed); err != nil {
		return err
	}
	steps := List(operation["steps"])
	byID := map[string]Object{}
	position := map[string]int{}
	graph := map[string][]string{}
	for i, raw := range steps {
		step := Map(raw)
		id := Text(step["step_id"])
		if byID[id] != nil {
			return fmt.Errorf("duplicate step id %s", id)
		}
		byID[id] = step
		position[id] = i
		graph[id] = []string{}
	}
	entry := Text(operation["entry_step"])
	if byID[entry] == nil {
		return fmt.Errorf("entry_step missing")
	}
	sourceSchema := func(ref, current string, allowSame bool) (Object, error) {
		match := agentDataRef.FindStringSubmatch(ref)
		if match == nil {
			return nil, fmt.Errorf("invalid data source ref %s", ref)
		}
		schema := input
		if match[1] != "input" {
			id := match[2]
			step := byID[id]
			if step == nil || step["type"] != "ACTION" || position[id] > position[current] || position[id] == position[current] && !allowSame {
				return nil, fmt.Errorf("non-prior action data source %s", id)
			}
			action, err := g.payload("actions", Text(step["action_ref"]))
			if err != nil {
				return nil, err
			}
			schema, err = g.payload("io_schemas", Text(action["output_schema_ref"]))
			if err != nil {
				return nil, err
			}
		}
		return agentSchemaPointer(schema, match[3])
	}
	bindings := func(raw any, schemaID, current string, allowSame, implicitOutcome bool) error {
		values := Map(raw)
		if values == nil {
			return fmt.Errorf("step input bindings must be a map")
		}
		schema, err := g.payload("io_schemas", schemaID)
		if err != nil {
			return err
		}
		for _, name := range Strings(schema["required"]) {
			if implicitOutcome && name == "outcome" {
				continue
			}
			token := strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
			if _, ok := values["/"+token]; !ok {
				return fmt.Errorf("step %s does not bind required field %s", current, name)
			}
		}
		for pointer, value := range values {
			target, err := agentSchemaPointer(schema, pointer)
			if err != nil {
				return err
			}
			source, err := sourceSchema(Text(value), current, allowSame)
			if err != nil {
				return err
			}
			if !agentSchemaCompatible(source, target) {
				return fmt.Errorf("incompatible binding %s -> %s", value, pointer)
			}
		}
		return nil
	}
	transition := func(raw any, current string) error {
		next := Map(raw)
		if next == nil {
			return fmt.Errorf("transition missing")
		}
		if target := Text(next["step_ref"]); target != "" {
			if byID[target] == nil {
				return fmt.Errorf("unknown next step %s", target)
			}
			graph[current] = append(graph[current], target)
			return nil
		}
		if outcome := Text(next["outcome"]); outcome != "" {
			if !Has(allowed, outcome) {
				return fmt.Errorf("undeclared terminal outcome %s", outcome)
			}
			if _, exists := next["output_bindings"]; exists {
				return bindings(next["output_bindings"], outputID, current, true, true)
			}
			return nil
		}
		reference := Text(next["outcome_from"])
		match := agentDataRef.FindStringSubmatch(reference)
		if match == nil || match[1] == "input" {
			return fmt.Errorf("outcome_from must read action step")
		}
		fragment, err := sourceSchema(reference, current, true)
		if err != nil {
			return err
		}
		values := Strings(fragment["enum"])
		if len(values) == 0 {
			return fmt.Errorf("dynamic outcome source must expose enum")
		}
		for _, value := range values {
			if !Has(allowed, value) {
				return fmt.Errorf("dynamic source can emit undeclared outcome %s", value)
			}
		}
		return nil
	}
	for _, raw := range steps {
		step := Map(raw)
		id := Text(step["step_id"])
		if step["type"] == "ACTION" {
			action, err := g.payload("actions", Text(step["action_ref"]))
			if err != nil {
				return err
			}
			if err := bindings(step["input_bindings"], Text(action["input_schema_ref"]), id, false, false); err != nil {
				return err
			}
			if action["effect"] == "MUTATE" {
				gateID := Text(step["mutation_gate_ref"])
				gate, err := g.payload("gates", gateID)
				if err != nil {
					return err
				}
				if err := bindings(step["mutation_gate_input_bindings"], Text(gate["input_schema_ref"]), id, false, false); err != nil {
					return err
				}
				for _, raw := range List(gate["requirements"]) {
					requirement := Map(raw)
					if !Has(allowed, Text(requirement["on_false"])) || !Has(allowed, Text(requirement["on_unknown"])) {
						return fmt.Errorf("gate denial outcomes not declared")
					}
				}
			} else if step["mutation_gate_ref"] != nil || step["mutation_gate_input_bindings"] != nil {
				return fmt.Errorf("READ_ONLY action must not have mutation gate")
			}
			for _, outcome := range Strings(action["failure_outcomes"]) {
				if !Has(allowed, outcome) {
					return fmt.Errorf("action failure outcome not declared")
				}
			}
			if err := transition(step["on_success"], id); err != nil {
				return err
			}
		} else {
			condition, err := g.payload("conditions", Text(step["condition_ref"]))
			if err != nil {
				return err
			}
			if err := bindings(step["input_bindings"], Text(condition["input_schema_ref"]), id, false, false); err != nil {
				return err
			}
			for _, branch := range []string{"TRUE", "FALSE", "UNKNOWN"} {
				if err := transition(Map(step["branches"])[branch], id); err != nil {
					return err
				}
			}
		}
	}
	return agentAcyclic(graph, entry)
}

func (g *agentContractGraph) resolveOperation(id string) (Object, error) {
	operation, err := g.payload("operations", id)
	if err != nil {
		return nil, err
	}
	selectedRules := []any{}
	specs := Object{}
	specOrder := []string{}
	for _, id := range Strings(operation["rule_refs"]) {
		rule := g.rules[id]
		if rule == nil {
			return nil, fmt.Errorf("unresolved rule %s", id)
		}
		selectedRules = append(selectedRules, rule)
		owner := g.ruleOwners[id]
		if specs[owner] == nil {
			specs[owner] = Object{"id": owner, "rule_ids": []any{}}
			specOrder = append(specOrder, owner)
		}
		entry := Map(specs[owner])
		entry["rule_ids"] = append(List(entry["rule_ids"]), id)
	}
	actionIDs, conditionIDs, gateIDs := []string{}, []string{}, []string{}
	for _, raw := range List(operation["preconditions"]) {
		if id := Text(Map(raw)["condition_ref"]); id != "" {
			conditionIDs = append(conditionIDs, id)
		}
	}
	for _, raw := range List(operation["steps"]) {
		step := Map(raw)
		if step["type"] == "ACTION" {
			actionIDs = append(actionIDs, Text(step["action_ref"]))
			if id := Text(step["mutation_gate_ref"]); id != "" {
				gateIDs = append(gateIDs, id)
			}
		} else if step["type"] == "CONDITION" {
			conditionIDs = append(conditionIDs, Text(step["condition_ref"]))
		}
	}
	actions, gates, conditions, io, vocab := Object{}, Object{}, Object{}, Object{}, Object{}
	ioIDs := []string{Text(operation["input_schema_ref"]), Text(operation["output_schema_ref"])}
	for _, id := range iwpSortedUnique(actionIDs) {
		payload, err := g.payload("actions", id)
		if err != nil {
			return nil, err
		}
		actions[id] = payload
		ioIDs = append(ioIDs, Text(payload["input_schema_ref"]), Text(payload["output_schema_ref"]))
		for _, field := range []string{"preconditions", "postconditions"} {
			for _, raw := range List(payload[field]) {
				conditionIDs = append(conditionIDs, Text(Map(raw)["condition_ref"]))
			}
		}
	}
	for _, id := range iwpSortedUnique(gateIDs) {
		payload, err := g.payload("gates", id)
		if err != nil {
			return nil, err
		}
		gates[id] = payload
		ioIDs = append(ioIDs, Text(payload["input_schema_ref"]))
		for _, raw := range List(payload["requirements"]) {
			conditionIDs = append(conditionIDs, Text(Map(raw)["condition_ref"]))
		}
	}
	sort.Strings(conditionIDs)
	for len(conditionIDs) > 0 {
		id := conditionIDs[0]
		conditionIDs = conditionIDs[1:]
		if conditions[id] != nil {
			continue
		}
		payload, err := g.payload("conditions", id)
		if err != nil {
			return nil, err
		}
		conditions[id] = payload
		ioIDs = append(ioIDs, Text(payload["input_schema_ref"]))
		conditionIDs = append(conditionIDs, agentConditionChildren(payload)...)
	}
	for _, id := range iwpSortedUnique(ioIDs) {
		payload, err := g.payload("io_schemas", id)
		if err != nil {
			return nil, err
		}
		io[id] = payload
	}
	for _, id := range Strings(operation["vocabulary_refs"]) {
		payload, err := g.payload("vocabularies", id)
		if err != nil {
			return nil, err
		}
		vocab[id] = payload
	}
	if len(vocab) == 0 {
		return nil, fmt.Errorf("operation vocabulary_refs must be non-empty")
	}
	selectedSpecs := []any{}
	for _, id := range specOrder {
		selectedSpecs = append(selectedSpecs, specs[id])
	}
	selectedBindings := Object{"actions": []any{}, "conditions": []any{}, "entrypoints": []any{}}
	for _, config := range [][3]string{{"actions", "action_bindings", "action_id"}, {"conditions", "condition_evaluator_bindings", "condition_id"}, {"entrypoints", "operation_entrypoints", "operation_id"}} {
		for _, raw := range List(g.binding[config[1]]) {
			item := Map(raw)
			bindingID := Text(item[config[2]])
			selected := false
			switch config[0] {
			case "actions":
				selected = actions[bindingID] != nil
			case "conditions":
				selected = conditions[bindingID] != nil
			case "entrypoints":
				selected = bindingID == id
			}
			if selected {
				selectedBindings[config[0]] = append(List(selectedBindings[config[0]]), item)
			}
		}
	}
	return Object{"schema_version": "ptsip-agent-operation-resolution/v1", "operation_id": id, "operation": operation, "selected_specs": selectedSpecs, "rules": selectedRules, "actions": actions, "conditions": conditions, "gates": gates, "io_schemas": io, "vocabularies": vocab, "implementation_bindings": selectedBindings, "markdown_dependency": false}, nil
}
func ValidateAgentContractPlane(r *Repository) (Object, error) {
	graph, err := loadAgentContractGraph(r)
	if err != nil {
		return nil, err
	}
	if err := graph.validate(); err != nil {
		return nil, err
	}
	return graph.counts, nil
}
