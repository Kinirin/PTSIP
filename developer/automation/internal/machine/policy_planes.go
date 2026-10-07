package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	policylifecycle "github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
)

func (r *Repository) policyValidateGovernanceSources(records map[string]Object) []string {
	errors := []string{}
	registry, err := r.Read("developer/policy/registries/governance-source-registry.yaml")
	if err != nil {
		return []string{err.Error()}
	}
	if err := r.Validate("developer/policy/schemas/governance-source-registry.schema.json", registry); err != nil {
		errors = append(errors, err.Error())
	}
	constants := Map(registry["constants"])
	if len(constants) != 3 || Map(constants["USER_EXPLICIT"])["may_create_official_authority"] != true || Map(constants["AGENT_INFERRED"])["may_create_official_authority"] != false || Map(constants["AUTOMATION_DERIVED"])["may_create_official_authority"] != false {
		errors = append(errors, "governance source constants must preserve USER_EXPLICIT sole authority")
	}
	if Map(constants["AGENT_INFERRED"])["provisional_resolution_only"] != true || Map(constants["AGENT_INFERRED"])["explicit_user_opt_in_required"] != true {
		errors = append(errors, "AGENT_INFERRED requires provisional-only explicit opt-in")
	}
	if Map(constants["AUTOMATION_DERIVED"])["may_propagate_existing_authority_only"] != true {
		errors = append(errors, "AUTOMATION_DERIVED may propagate existing authority only")
	}
	review, err := r.Read("developer/policy/source-application-review.yaml")
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		if err := r.Validate("developer/policy/schemas/source-application-review.schema.json", review); err != nil {
			errors = append(errors, err.Error())
		}
		for _, raw := range List(review["policy_query_list"]) {
			query := Map(raw)
			id := Text(query["policy_id"])
			record := records[id]
			if record == nil {
				errors = append(errors, "source review requires canonical Root policy: "+id)
				continue
			}
			identity := Map(record["policy"])
			if identity["status"] != query["expected_status"] {
				errors = append(errors, id+": source review status mismatch")
			}
			if identity["status"] != "ACTIVE" && query["review_role"] != "REVIEW_ONLY_NOT_AUTHORITY" {
				errors = append(errors, id+": inactive source review must be non-authority")
			}
			for _, section := range policyStrings(query["sections"]) {
				if _, found := Map(record["rules"])[section]; !found {
					errors = append(errors, id+": source review missing exact section "+section)
				}
			}
		}
		for _, raw := range List(review["artifact_review_list"]) {
			reference := Text(Map(raw)["path"])
			path, err := r.Path(reference)
			if err != nil {
				errors = append(errors, err.Error())
			} else if _, err := os.Stat(path); err != nil {
				errors = append(errors, "source review artifact missing: "+reference)
			}
		}
	}
	auth, err := r.Read("developer/policy/registries/authorization-transition-registry.yaml")
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		if Map(auth["authorization_provenance"])["authorization_source"] != "USER_EXPLICIT" {
			errors = append(errors, "authorization source must be USER_EXPLICIT")
		}
		semantics := Map(auth["transition_semantics"])
		if semantics["transition_source"] != "AUTOMATION_DERIVED" || semantics["may_create_official_authority"] != false {
			errors = append(errors, "transition source must be non-authority AUTOMATION_DERIVED")
		}
	}
	root, err := r.Path("developer/policy/approvals")
	if err != nil {
		return append(errors, err.Error())
	}
	paths, err := filepath.Glob(filepath.Join(root, "*.yaml"))
	if err != nil {
		return append(errors, err.Error())
	}
	for _, path := range paths {
		payload, err := r.Read(path)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if err := r.Validate(policyApprovalSchema, payload); err != nil {
			errors = append(errors, path+": "+err.Error())
		}
		if Map(payload["approval"])["decision_source"] != "USER_EXPLICIT" {
			errors = append(errors, path+": approval decision source must be USER_EXPLICIT")
		}
	}
	return errors
}

func (r *Repository) policyValidateAnalysisPlane(records map[string]Object) []string {
	errors := []string{}
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		return append(errors, err.Error())
	}
	if err := r.Validate(policyAnalysisRegistrySchema, registry); err != nil {
		errors = append(errors, err.Error())
	}

	recordIDs := []string{}
	recordRefs := []string{}
	recordByID := map[string]Object{}
	discoveryKeys := map[string]bool{}
	for _, raw := range List(registry["records"]) {
		record := Map(raw)
		id, reference := Text(record["analysis_id"]), Text(record["analysis_ref"])
		recordIDs = append(recordIDs, id)
		recordRefs = append(recordRefs, reference)
		if _, found := recordByID[id]; found {
			errors = append(errors, "analysis registry duplicate analysis_id "+id)
		}
		recordByID[id] = record
		expected := policyAnalysisRecordRoot + id + ".yaml"
		if reference != expected {
			errors = append(errors, id+": noncanonical analysis_ref "+reference)
		}
		key := Text(record["subject_type"]) + "|" + Text(record["subject_id"]) + "|" + Text(record["analysis_kind"])
		if discoveryKeys[key] {
			errors = append(errors, "analysis registry duplicate discovery key "+key)
		}
		discoveryKeys[key] = true
	}
	sortedIDs := append([]string{}, recordIDs...)
	sort.Strings(sortedIDs)
	if !reflect.DeepEqual(recordIDs, sortedIDs) {
		errors = append(errors, "analysis registry records must be in analysis ID order")
	}
	if !policyUnique(recordIDs) || !policyUnique(recordRefs) {
		errors = append(errors, "analysis registry record IDs and refs must be unique")
	}

	root, err := r.Path(strings.TrimSuffix(policyAnalysisRoot, "/"))
	if err != nil {
		return append(errors, err.Error())
	}
	legacy, err := filepath.Glob(filepath.Join(root, "PRA-*.yaml"))
	if err != nil {
		return append(errors, err.Error())
	}
	if len(legacy) > 0 {
		errors = append(errors, "root-level PRA files are forbidden; use registered records/PRA-NNNN.yaml")
	}
	recordRoot, err := r.Path(strings.TrimSuffix(policyAnalysisRecordRoot, "/"))
	if err != nil {
		return append(errors, err.Error())
	}
	paths, err := filepath.Glob(filepath.Join(recordRoot, "PRA-*.yaml"))
	if err != nil {
		return append(errors, err.Error())
	}
	discovered := []string{}
	for _, path := range paths {
		relative, err := r.Scope(path)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		discovered = append(discovered, relative)
	}
	sort.Strings(discovered)
	expectedRefs := append([]string{}, recordRefs...)
	sort.Strings(expectedRefs)
	if !reflect.DeepEqual(discovered, expectedRefs) {
		errors = append(errors, "analysis registry records must exactly cover canonical PRA files")
	}

	for _, id := range recordIDs {
		record := recordByID[id]
		reference := Text(record["analysis_ref"])
		payload, err := r.Read(reference)
		if err != nil {
			errors = append(errors, reference+": "+err.Error())
			continue
		}
		if err := r.Validate(policyAnalysisSchema, payload); err != nil {
			errors = append(errors, reference+": "+err.Error())
		}
		for _, message := range r.ValidateAnalysisSemantics(payload) {
			errors = append(errors, reference+": "+message)
		}
		if Text(Map(payload["analysis"])["analysis_id"]) != id {
			errors = append(errors, reference+": payload analysis_id mismatch")
		}
	}

	bound := []string{}
	expected := []string{}
	for id, record := range records {
		if record["policy_class"] == DeveloperClass {
			expected = append(expected, id)
		}
	}
	sort.Strings(expected)
	for _, raw := range List(registry["bindings"]) {
		binding := Map(raw)
		id := Text(binding["policy_id"])
		record := records[id]
		if record == nil {
			if rootID.MatchString(id) {
				errors = append(errors, "analysis registry unknown Root policy: "+id)
			}
			continue
		}
		if record["policy_class"] == DeveloperClass {
			bound = append(bound, id)
		}
		analysisID := Text(binding["analysis_id"])
		if _, found := recordByID[analysisID]; !found {
			errors = append(errors, id+": unknown analysis_id "+analysisID)
			continue
		}
		result, err := r.ValidateResponsibilityAnalysis(analysisID, false)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if result["analysis_id"] != binding["analysis_id"] || result["materialization_allowed"] != true {
			errors = append(errors, id+": analysis identity/allowance mismatch")
		}
		group := Object{}
		matches := 0
		for _, rawGroup := range List(result["materialization_groups"]) {
			candidate := Map(rawGroup)
			if candidate["group_id"] == binding["group_id"] {
				group = candidate
				matches++
			}
		}
		if matches != 1 {
			errors = append(errors, id+": materialization group unresolved")
			continue
		}
		match := policyReadableFamilyID.FindStringSubmatch(id)
		if match == nil || group["family"] != match[1] || binding["family"] != group["family"] {
			errors = append(errors, id+": analysis Family mismatch")
		}
		if group["policy_class"] != record["policy_class"] || binding["policy_class"] != record["policy_class"] {
			errors = append(errors, id+": analysis class boundary mismatch")
		}
	}
	if !reflect.DeepEqual(bound, expected) || !policyUnique(bound) {
		errors = append(errors, "analysis registry must cover every canonical Root policy exactly in policy ID order")
	}
	return errors
}

func (r *Repository) policyValidateRegistryPlanes(records map[string]Object) []string {
	errors := []string{}
	schemaRegistry, err := r.Read("developer/policy/registries/authority-schema-registry.yaml")
	if err != nil {
		return []string{err.Error()}
	}
	roleRegistry, err := r.Read("developer/policy/registries/authority-role-registry.yaml")
	if err != nil {
		return []string{err.Error()}
	}
	for label, payload := range map[string]Object{"schema": schemaRegistry, "role": roleRegistry} {
		if err := r.Validate("developer/policy/schemas/developer-governance-registry.schema.json", payload); err != nil {
			errors = append(errors, label+": "+err.Error())
		}
	}
	schemaIDs, roleIDs := []string{}, []string{}
	for _, raw := range List(schemaRegistry["entries"]) {
		schemaIDs = append(schemaIDs, Text(Map(raw)["policy_id"]))
	}
	for _, raw := range List(roleRegistry["policy_roles"]) {
		roleIDs = append(roleIDs, Text(Map(raw)["policy_id"]))
	}
	if !reflect.DeepEqual(schemaIDs, roleIDs) {
		errors = append(errors, "developer schema and role registries must cover identical exact owner identities")
	}
	vocabulary := Map(roleRegistry["effect_vocabulary"])
	tokens := policyStrings(vocabulary["tokens"])
	if !policyUnique(tokens) || fmt.Sprint(vocabulary["count"]) != fmt.Sprint(len(tokens)) {
		errors = append(errors, "developer effect vocabulary count/uniqueness mismatch")
	}
	semanticSchema, err := r.Read(Text(schemaRegistry["semantic_schema_document"]))
	if err != nil {
		return append(errors, err.Error())
	}
	for _, raw := range List(schemaRegistry["entries"]) {
		entry := Map(raw)
		id := Text(entry["policy_id"])
		record := records[id]
		if record == nil {
			errors = append(errors, "developer semantic registry requires canonical Root policy: "+id)
			continue
		}
		definition := Map(Map(semanticSchema["$defs"])[Text(entry["schema_definition"])])
		value := Map(record["rules"])["authority_semantics"]
		if section := Text(entry["section"]); section != "" {
			value = Map(record["rules"])[section]
		}
		if err := r.policyValidateInline(definition, value); err != nil {
			errors = append(errors, id+": "+err.Error())
		}
	}
	auth, err := r.Read("developer/policy/registries/authorization-transition-registry.yaml")
	if err != nil {
		errors = append(errors, err.Error())
	} else if err := r.Validate("developer/policy/schemas/developer-governance-registry.schema.json", auth); err != nil {
		errors = append(errors, err.Error())
	}
	return errors
}

func (r *Repository) policyValidateSupportPlane() []string {
	errors := []string{}
	index, err := r.Read("src/policy/index.yaml")
	if err != nil {
		return []string{err.Error()}
	}
	if err := r.Validate("src/policy/schemas/ptsip-support-feature-policy-index.schema.json", index); err != nil {
		errors = append(errors, err.Error())
	}
	records := map[string]Object{}
	allIDs := []string{}
	last := ""
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		allIDs = append(allIDs, id)
		if id <= last {
			errors = append(errors, "Support IDs must be unique and ordered")
		}
		last = id
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			continue
		}
		if !strings.HasPrefix(id, "SFP-") {
			errors = append(errors, "Support plane must not inherit Developer identity: "+id)
			continue
		}
		relative := Text(entry["path"])
		path, err := r.Scope("src/policy/" + relative)
		if err != nil || !strings.HasPrefix(path, "src/policy/") {
			errors = append(errors, "Support policy path escapes class plane")
			continue
		}
		record, err := r.Read(path)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		records[id] = record
		identity := Map(record["policy"])
		if record["policy_class"] != "PTSIP_SUPPORT_FEATURE" || identity["id"] != id || identity["status"] != entry["status"] {
			errors = append(errors, id+": Support class/status/identity mismatch")
		}
		if err := r.Validate("src/policy/schemas/ptsip-support-root-family-policy.schema.json", record); err != nil {
			errors = append(errors, err.Error())
		}
	}
	registries := []string{"ptsip-support-authority-schema-registry.yaml", "ptsip-support-authority-role-registry.yaml", "ptsip-support-authority-subject-registry.yaml", "ptsip-support-authorization-registry.yaml"}
	payloads := []Object{}
	for _, name := range registries {
		path := "src/policy/registries/" + name
		payload, err := r.Read(path)
		if err != nil {
			errors = append(errors, err.Error())
			payloads = append(payloads, Object{})
			continue
		}
		payloads = append(payloads, payload)
		if err := r.Validate("src/policy/schemas/ptsip-support-governance-registry.schema.json", payload); err != nil {
			errors = append(errors, path+": "+err.Error())
		}
		dataPath, _ := r.Path(path)
		data, _ := os.ReadFile(dataPath)
		if strings.Contains(string(data), "decisions/") {
			errors = append(errors, path+": registry must not depend on legacy decisions")
		}
	}
	if len(payloads) == 4 {
		schemaIDs, roleIDs := []string{}, []string{}
		for _, raw := range List(payloads[0]["entries"]) {
			schemaIDs = append(schemaIDs, Text(Map(raw)["policy_id"]))
		}
		for _, raw := range List(payloads[1]["policy_roles"]) {
			roleIDs = append(roleIDs, Text(Map(raw)["policy_id"]))
		}
		if !reflect.DeepEqual(schemaIDs, allIDs) || !reflect.DeepEqual(roleIDs, allIDs) {
			errors = append(errors, "Support schema and role registries must exactly cover Support catalog")
		}
		vocabulary := Map(payloads[1]["effect_vocabulary"])
		tokens := policyStrings(vocabulary["tokens"])
		if !policyUnique(tokens) || fmt.Sprint(vocabulary["count"]) != fmt.Sprint(len(tokens)) {
			errors = append(errors, "Support effect vocabulary count/uniqueness mismatch")
		}
		subject := payloads[2]
		schemes := Map(subject["subject_identity_schemes"])
		if len(schemes) != 1 || !reflect.DeepEqual(policyStrings(Map(schemes["SUPPORT_POLICY_ID"])["registered_values"]), allIDs) {
			errors = append(errors, "Support subject identities must use Support catalog only")
		}
		if _, found := subject["current_repository_bindings"]; found {
			errors = append(errors, "Support registry must not ship repository bindings")
		}
		for _, forbidden := range []string{"authorization_provenance", "rules", "held_scopes"} {
			if _, found := payloads[3][forbidden]; found {
				errors = append(errors, "Support authorization registry must not ship "+forbidden)
			}
		}
	}
	errors = append(errors, r.policyValidateMigrationUnits("src/policy", "PTSIP_SUPPORT_FEATURE", records)...)
	return errors
}

// Migration validation verifies direct Root ownership and complete accounting.
// Frozen source identities and paths are evidence, never runtime record inputs.
func (r *Repository) policyValidateMigrationUnits(root, class string, records map[string]Object) []string {
	index, err := r.Read(root + "/index.yaml")
	if err != nil {
		return []string{err.Error()}
	}
	reference := Text(index["migration_registry_ref"])
	if reference == "" {
		return []string{root + ": migration evidence registry is not admitted"}
	}
	graph, err := r.Read(root + "/" + reference)
	if err != nil {
		return []string{err.Error()}
	}
	if err := r.Validate(root+"/schemas/root-family-migration.schema.json", graph); err != nil {
		return []string{err.Error()}
	}
	if graph["policy_class"] != class {
		return []string{root + ": migration evidence class mismatch"}
	}
	errors := []string{}
	if root == "developer/policy" {
		if err := policylifecycle.CheckMigrationSourceCatalog(index, graph); err != nil {
			errors = append(errors, err.Error())
		}
	}
	units := map[string]bool{}
	sourceIDs := map[string]bool{}
	counts := map[string]int{}
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		sourceID := Text(source["source_policy_id"])
		if sourceIDs[sourceID] {
			errors = append(errors, root+": duplicate evidence source identity")
		}
		sourceIDs[sourceID] = true
		pointers := []string{}
		for _, rawUnit := range List(source["units"]) {
			unit := Map(rawUnit)
			id, section := Text(unit["policy_id"]), Text(unit["section"])
			record := records[id]
			key := id + ":" + section
			if units[key] {
				errors = append(errors, root+": duplicate canonical unit owner "+key)
			}
			units[key] = true
			counts[id]++
			if record == nil {
				errors = append(errors, root+": missing canonical Root owner "+id)
				continue
			}
			if record["policy_class"] != class || record["responsibility_family"] != unit["family"] || Map(record["policy"])["status"] != source["source_status"] {
				errors = append(errors, key+": migrated class/family/state mismatch")
			}
			field := "rules"
			if class == "PTSIP_SUPPORT_FEATURE" {
				field = "authority_semantics"
			}
			if _, found := Map(record[field])[section]; !found {
				errors = append(errors, key+": migrated responsibility section missing")
			}
			pointer := Text(unit["source_pointer"])
			for _, prior := range pointers {
				if pointer == prior || strings.HasPrefix(pointer, prior+"/") || strings.HasPrefix(prior, pointer+"/") {
					errors = append(errors, sourceID+": overlapping evidence responsibility pointers")
				}
			}
			pointers = append(pointers, pointer)
		}
	}
	for _, raw := range List(graph["materializations"]) {
		owner := Map(raw)
		id := Text(owner["policy_id"])
		record := records[id]
		if record == nil {
			errors = append(errors, root+": owner missing "+id)
			continue
		}
		if fmt.Sprint(owner["unit_count"]) != fmt.Sprint(counts[id]) {
			errors = append(errors, id+": materialization count mismatch")
		}
		if owner["definition_only"] == true && (counts[id] != 0 || Map(record["policy"])["status"] != "DRAFT") {
			errors = append(errors, id+": definition-only owner must remain DRAFT")
		}
		field := "rules"
		if class == "PTSIP_SUPPORT_FEATURE" {
			field = "authority_semantics"
		}
		for section := range Map(record[field]) {
			if strings.HasPrefix(section, "unit_") && !units[id+":"+section] {
				errors = append(errors, id+":"+section+": unit not covered by evidence")
			}
		}
	}
	for _, family := range policyRootFamilies {
		found := false
		for _, record := range records {
			if record["responsibility_family"] == family {
				found = true
			}
		}
		if !found {
			errors = append(errors, root+": Root Family missing "+family)
		}
	}
	return errors
}
