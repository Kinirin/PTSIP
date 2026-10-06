package machine

import (
	"fmt"
	policylifecycle "github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const policyCatalogContracts = "developer/policy/registries/developer-policy-catalog-contracts.json"

func (r *Repository) ResolveNeutralCatalogContract(id string) (Object, error) {
	if _, err := r.Compiler(); err != nil {
		return nil, err
	}
	record, err := r.Read(policyCatalogContracts)
	if err != nil {
		return nil, err
	}
	contract := Map(Map(record["contracts"])[id])
	if contract == nil {
		return nil, policyFailure("UNKNOWN_NEUTRAL_CATALOG_CONTRACT", id)
	}
	result := policyClone(contract)
	result["contract_id"] = id
	return result, nil
}
func (r *Repository) LoadNeutralPolicyIndex() (Object, error) {
	contracts, err := r.Read(policyCatalogContracts)
	if err != nil {
		return nil, err
	}
	contract := Map(Map(contracts["contracts"])[Text(Map(contracts["entrypoints"])["index"])])
	if contract == nil {
		return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", "unregistered index contract")
	}
	index, err := r.Read(Text(contract["canonical_path"]))
	if err != nil {
		return nil, err
	}
	if err := r.Validate(Text(contract["schema_ref"]), index); err != nil {
		return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", err.Error())
	}
	last := ""
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		if id <= last {
			return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", "IDs must be globally unique and ordered")
		}
		last = id
		// Migration source entries are provenance only; their bodies never enter this loader.
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			if entry["policy_class"] != DeveloperClass || entry["path"] != "developer/policy/legacy/"+id+".yaml" {
				return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", "invalid audit-only source metadata")
			}
			continue
		}
		expected, err := policyCanonicalPath(id)
		if err != nil {
			return nil, err
		}
		if entry["path"] != expected {
			return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", id+" canonical path mismatch")
		}
		boundary := policyBoundaryID.MatchString(id)
		if boundary != (entry["policy_class"] == "PTSIP_BOUND_POLICY") {
			return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", id+" class/namespace mismatch")
		}
		if entry["policy_class"] == DeveloperClass && !rootID.MatchString(id) {
			return nil, policyFailure("INVALID_NEUTRAL_POLICY_INDEX", id+" requires Root Family identity")
		}
	}
	return index, nil
}

func (r *Repository) ValidateNeutralCatalogContractRegistration() []string {
	errors := []string{}
	if _, err := r.Compiler(); err != nil {
		return []string{err.Error()}
	}
	record, err := r.Read(policyCatalogContracts)
	if err != nil {
		return []string{err.Error()}
	}
	contracts, entrypoints, definitions := Map(record["contracts"]), Map(record["entrypoints"]), Map(record["$defs"])
	registered := map[string]bool{}
	for _, id := range entrypoints {
		registered[Text(id)] = true
	}
	if len(registered) != len(contracts) {
		errors = append(errors, "neutral catalog entrypoints must cover each exact registered contract")
	}
	for id := range contracts {
		if !registered[id] {
			errors = append(errors, "unreachable neutral catalog contract: "+id)
		}
	}
	for role, definition := range map[string]string{"index": "catalog", "subject": "subject"} {
		id := Text(entrypoints[role])
		contract := Map(contracts[id])
		if id != Text(Map(definitions[definition+"_schema_version"])["const"]) {
			errors = append(errors, "neutral catalog "+role+" schema identity mismatch")
		}
		if contract["artifact_class"] != Map(definitions[definition+"_artifact_class"])["const"] {
			errors = append(errors, "neutral catalog "+role+" artifact identity mismatch")
		}
		for _, key := range []string{"schema_ref", "canonical_path"} {
			path, err := r.Path(Text(contract[key]))
			if err != nil {
				errors = append(errors, err.Error())
				continue
			}
			if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
				errors = append(errors, "neutral catalog unresolved repository reference: "+Text(contract[key]))
			}
		}
		schema, err := r.Read(Text(contract["schema_ref"]))
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		for _, field := range []string{"schema_version", "artifact_class"} {
			definition := Map(Map(schema["properties"])[field])
			value := contract[field]
			if field == "schema_version" {
				value = id
			}
			if err := r.policyValidateInline(definition, value); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}
	scope := Map(record["change_scope"])
	targets := List(scope["materialization_targets"])
	paths := []string{}
	for _, raw := range targets {
		paths = append(paths, Text(Map(raw)["path"]))
	}
	if !policyUnique(paths) {
		errors = append(errors, "neutral catalog materialization targets must be unique")
	}
	for _, raw := range append(append([]any{}, targets...), List(scope["deferred_application_targets"])...) {
		target := Map(raw)
		reference := Text(target["path"])
		absolute, err := r.Path(reference)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		content, err := os.ReadFile(absolute)
		if err != nil {
			errors = append(errors, "neutral catalog unresolved scope target: "+reference)
			continue
		}
		// Declared selectors are checked against sources and are never executed.
		for _, name := range policyStrings(target["go_functions"]) {
			pattern := regexp.MustCompile("(?m)^func\\s+(?:\\([^)]*\\)\\s+)?" + regexp.QuoteMeta(name) + "\\s*\\(")
			if !pattern.Match(content) {
				errors = append(errors, reference+": Go selector missing: "+name)
			}
		}
		for _, name := range policyStrings(target["python_functions"]) {
			pattern := regexp.MustCompile("(?m)^def\\s+" + regexp.QuoteMeta(name) + "\\s*\\(")
			if !pattern.Match(content) {
				errors = append(errors, reference+": historical selector missing: "+name)
			}
		}
	}
	for _, path := range paths {
		for _, prefix := range policyStrings(scope["forbidden_write_roots"]) {
			if strings.HasPrefix(path, prefix) {
				errors = append(errors, "neutral catalog forbidden target: "+path)
			}
		}
	}
	for _, path := range policyStrings(scope["verification_test_paths"]) {
		if !policyContains(paths, path) {
			errors = append(errors, "neutral catalog verification paths must be registered materialization targets")
		}
	}
	for _, reference := range policyStrings(scope["preserved_contracts"]) {
		absolute, err := r.Path(reference)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if _, err := os.Stat(absolute); err != nil {
			errors = append(errors, "neutral catalog preserved contract missing: "+reference)
		}
	}
	gate, execution := Map(record["application_gate"]), Map(record["application_execution"])
	if gate["catalog_payload_migration_authorized"] == true || gate["m2_m8_implementation_authorized"] == true || gate["vpms_policy_materialization_authorized"] == true {
		if execution["decision_source"] != "USER_EXPLICIT" {
			errors = append(errors, "neutral catalog application requires explicit execution provenance")
		}
	}
	if execution["vpms_class_materialization_enabled"] == true && (execution["m1_m7_verified"] != true || gate["vpms_policy_materialization_authorized"] != true) {
		errors = append(errors, "VPMS materialization requires M1-M7 verification and explicit authorization")
	}
	if reference := Text(gate["existing_policy_materialization_schema_ref"]); reference != "" {
		path, err := r.Path(reference)
		if err != nil {
			errors = append(errors, err.Error())
		} else if _, err := os.Stat(path); err != nil {
			errors = append(errors, "neutral catalog materialization gate unresolved")
		}
	}
	return errors
}
func (r *Repository) policyValidateInline(schema Object, value any) error {
	if schema == nil {
		return fmt.Errorf("missing schema definition")
	}
	compiler, err := r.Compiler()
	if err != nil {
		return err
	}
	data, err := CanonicalJSON(schema)
	if err != nil {
		return err
	}
	id := "urn:ptsip:go:policy-validation:" + SHA256(data)
	if compiled := r.schemas[id]; compiled != nil {
		return compiled.Validate(value)
	}
	inline := policyClone(schema)
	inline["$id"] = id
	if err := compiler.AddResource(id, inline); err != nil {
		return err
	}
	compiled, err := compiler.Compile(id)
	if err != nil {
		return err
	}
	r.schemas[id] = compiled
	return compiled.Validate(value)
}
func (r *Repository) ValidateNeutralCatalogSnapshot(catalog, subject Object, records map[string]Object, sourceIndex, sourceSubject Object) []string {
	errors := r.ValidateNeutralCatalogContractRegistration()
	if len(errors) > 0 {
		return errors
	}
	contracts, err := r.Read(policyCatalogContracts)
	if err != nil {
		return []string{err.Error()}
	}
	for role, payload := range map[string]Object{"index": catalog, "subject": subject} {
		contract := Map(Map(contracts["contracts"])[Text(Map(contracts["entrypoints"])[role])])
		if err := r.Validate(Text(contract["schema_ref"]), payload); err != nil {
			errors = append(errors, "neutral catalog "+role+": "+err.Error())
		}
	}
	if len(errors) > 0 {
		return errors
	}
	ids := []string{}
	sourceIDs := []string{}
	sourcePaths := map[string]string{}
	for _, raw := range List(sourceIndex["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		sourceIDs = append(sourceIDs, id)
		sourcePaths[id] = Text(entry["path"])
	}
	for _, raw := range List(catalog["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		ids = append(ids, id)
		source := records[id]
		identity := Map(source["policy"])
		expected, err := policyCanonicalPath(id)
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			expected = "developer/policy/legacy/" + id + ".yaml"
			err = nil
		}
		if err != nil || entry["path"] != expected || sourcePaths[id] != Text(entry["path"]) {
			errors = append(errors, id+": neutral catalog canonical path mismatch")
		}
		if policyBoundaryID.MatchString(id) != (entry["policy_class"] == "PTSIP_BOUND_POLICY") {
			errors = append(errors, id+": neutral catalog boundary class/namespace mismatch")
		}
		if source == nil || identity["id"] != id {
			errors = append(errors, id+": unresolved source policy")
			continue
		}
		if entry["policy_class"] != source["policy_class"] {
			errors = append(errors, id+": class projection mismatch")
		}
		if entry["status"] != identity["status"] {
			errors = append(errors, id+": status projection mismatch")
		}
	}
	if !policyUnique(ids) || !sort.StringsAreSorted(ids) {
		errors = append(errors, "neutral catalog IDs must be unique and ordered")
	}
	if !reflect.DeepEqual(ids, sourceIDs) || len(records) != len(ids) {
		errors = append(errors, "neutral catalog must preserve exact indexed source membership")
	}
	if !reflect.DeepEqual(policyStrings(Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"]), ids) {
		errors = append(errors, "neutral subject IDs must exactly project catalog membership")
	}
	for _, field := range policyStrings(Map(contracts["invariants"])["preserved_subject_fields"]) {
		if !reflect.DeepEqual(subject[field], sourceSubject[field]) {
			errors = append(errors, "neutral subject source semantics changed: "+field)
		}
	}
	return errors
}

func ValidatePolicyVersionSemantics(id string, payload Object) []string {
	return policylifecycle.ValidatePolicyVersionSemantics(id, payload)
}
func ValidatePolicyTransitionSemantics(id string, payload Object) []string {
	transition := Map(payload["transition"])
	if transition == nil {
		return []string{}
	}
	errors := []string{}
	requirements := List(transition["requirements"])
	states := map[string]string{}
	dependencies := map[string][]string{}
	order := []string{}
	for _, raw := range requirements {
		unit := Map(raw)
		name := Text(unit["id"])
		if name == "" {
			continue
		}
		if _, found := states[name]; found {
			errors = append(errors, id+": transition requirement IDs must be unique")
		}
		states[name] = Text(unit["state"])
		order = append(order, name)
	}
	for _, raw := range requirements {
		unit := Map(raw)
		name := Text(unit["id"])
		after := policyStrings(Map(unit["next_action"])["after"])
		dependencies[name] = after
		for _, dependency := range after {
			if _, found := states[dependency]; !found {
				errors = append(errors, id+": "+name+" references unknown requirement "+dependency)
			}
			if dependency == name {
				errors = append(errors, id+": "+name+" must not depend on itself")
			}
			if policyContains([]string{"IN_PROGRESS", "FAILED", "SATISFIED"}, states[name]) && states[dependency] != "SATISFIED" {
				errors = append(errors, id+": "+name+" cannot progress before dependency is SATISFIED: "+dependency)
			}
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string, []string)
	visit = func(name string, lineage []string) {
		if visited[name] {
			return
		}
		if visiting[name] {
			errors = append(errors, id+": transition dependency cycle: "+strings.Join(append(lineage, name), " -> "))
			return
		}
		visiting[name] = true
		for _, dependency := range dependencies[name] {
			if _, found := states[dependency]; found {
				visit(dependency, append(append([]string{}, lineage...), name))
			}
		}
		delete(visiting, name)
		visited[name] = true
	}
	for _, name := range order {
		visit(name, nil)
	}
	satisfied := len(requirements) > 0
	for _, raw := range requirements {
		if Map(raw)["state"] != "SATISFIED" {
			satisfied = false
		}
	}
	status := Map(payload["policy"])["status"]
	if status == "APPROVED" {
		expected := "PENDING"
		if satisfied {
			expected = "READY"
		}
		if transition["state"] != expected {
			errors = append(errors, id+": APPROVED transition state must be "+expected)
		}
	} else if status == "ACTIVE" {
		if !satisfied {
			errors = append(errors, id+": ACTIVE transition history requires all requirements SATISFIED")
		}
		if transition["state"] != "COMPLETE" {
			errors = append(errors, id+": ACTIVE transition history must be COMPLETE")
		}
	}
	return errors
}

func (r *Repository) ValidateDeveloperPolicy() []string {
	errors := []string{}
	add := func(label string, err error) {
		if err != nil {
			errors = append(errors, label+": "+err.Error())
		}
	}
	registry, err := r.Read("developer/policy/registries/root-family-entry-registry.json")
	add("Root Family registry", err)
	if err == nil {
		add("Root Family registry", r.Validate("developer/policy/schemas/root-family-entry-registry.schema.json", registry))
		for class, raw := range Map(registry["policy_classes"]) {
			_, err := r.Read(Text(Map(raw)["schema_ref"]))
			add(class+" schema_ref", err)
		}
	}
	index, err := r.LoadNeutralPolicyIndex()
	if err != nil {
		return append(errors, err.Error())
	}
	records := map[string]Object{}
	canonicalIDs := []string{}
	allIDs := []string{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		allIDs = append(allIDs, id)
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			continue
		}
		canonicalIDs = append(canonicalIDs, id)
		reference := Text(entry["path"])
		payload, err := r.Read(reference)
		if err != nil {
			add(reference, err)
			continue
		}
		records[id] = payload
		schema := "developer/policy/schemas/management-policy.schema.json"
		if entry["policy_class"] == DeveloperClass {
			schema = "developer/policy/schemas/root-family-policy.schema.json"
		}
		add(reference, r.Validate(schema, payload))
		identity := Map(payload["policy"])
		if identity["id"] != id || identity["status"] != entry["status"] || payload["policy_class"] != entry["policy_class"] {
			errors = append(errors, reference+": class/status/identity mismatch")
		}
		errors = append(errors, ValidatePolicyVersionSemantics(id, payload)...)
		errors = append(errors, ValidatePolicyTransitionSemantics(id, payload)...)
	}
	discovered, err := r.policyDiscoveredIDs()
	add("canonical inventory", err)
	if err == nil && !reflect.DeepEqual(canonicalIDs, discovered) {
		errors = append(errors, "developer policy index must cover canonical MPD files exactly")
	}
	subject, err := r.Read(policySubjectRegistry)
	add(policySubjectRegistry, err)
	if err == nil {
		add(policySubjectRegistry, r.Validate("developer/policy/schemas/developer-policy-subject-catalog.schema.json", subject))
		if !reflect.DeepEqual(policyStrings(Map(Map(subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])["registered_values"]), allIDs) {
			errors = append(errors, "developer subject identities must exactly project catalog membership")
		}
	}
	errors = append(errors, r.policyValidateGovernanceSources(records)...)
	errors = append(errors, r.policyValidateAnalysisPlane(records)...)
	errors = append(errors, r.policyValidateRegistryPlanes(records)...)
	errors = append(errors, r.policyValidateSupportPlane()...)
	errors = append(errors, r.policyValidateMigrationUnits("developer/policy", DeveloperClass, records)...)
	errors = append(errors, r.ValidateNeutralCatalogContractRegistration()...)
	resolver, err := NewResolver(r)
	if err == nil {
		add("resolver bindings", resolver.ValidateBindings())
	} else {
		add("resolver", err)
	}
	sort.Strings(errors)
	return errors
}
