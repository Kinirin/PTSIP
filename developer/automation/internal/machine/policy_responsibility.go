package machine

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const policyAnalysisRoot = "developer/policy/analysis/"
const policyAnalysisRecordRoot = "developer/policy/analysis/records/"
const policyAnalysisRegistry = "developer/policy/analysis/registry.yaml"
const policyAnalysisSchema = "developer/policy/analysis/schemas/policy-responsibility-analysis.schema.json"
const policyAnalysisRegistrySchema = "developer/policy/analysis/schemas/policy-materialization-analysis-registry.schema.json"

var policyAnalysisID = regexp.MustCompile(`^PRA-[0-9]{4}package machine

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

)

var policyRootFamilies = []string{"NORM", "GOV", "INTENT", "ARCH", "INFO", "CNTR", "RISK", "SUPPLY", "REAL", "ASSURE", "CTRL", "CHANGE", "OPS", "RECORD"}
var policyLegacyFamilies = []string{"SPEC", "PLAN", "WORK", "VERI", "MIGR", "RELS"}
var policyCollisionResolutions = map[string][]string{
	"EXACT_DUPLICATE": {"REFERENCE_EXISTING"}, "SEMANTIC_EQUIVALENT": {"REFERENCE_EXISTING"},
	"EXISTING_SUBSUMES_CANDIDATE": {"DROP_REDUNDANT_CANDIDATE"},
	"CANDIDATE_EXTENDS_EXISTING":  {"MERGE_INTO_EXISTING", "CREATE_NEW_SIBLING_POLICY"},
	"PARTIAL_OVERLAP":             {"SPLIT_RESPONSIBILITY_REQUIRED"}, "DISTINCT_SCOPED_AUTHORITY": {"COEXIST_SCOPED"},
	"CONFLICT": {"FAIL_CLOSED_OWNER_DECISION_REQUIRED"},
}

type PolicyError = OperationError

func policyFailure(code, detail string) error { return &PolicyError{code, detail} }
func policyContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
func policyStrings(value any) []string {
	result := []string{}
	for _, raw := range List(value) {
		result = append(result, Text(raw))
	}
	return result
}
func policyUnique(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func policySetsEqual(a, b map[string]bool) bool { return reflect.DeepEqual(a, b) }
func policyClone[T any](value T) T {
	// The canonical values have string keys, so copying recursively preserves numeric types.
	return policyCloneValue(value).(T)
}
func policyCloneValue(value any) any {
	switch raw := value.(type) {
	case map[string]any:
		out := Object{}
		for key, child := range raw {
			out[key] = policyCloneValue(child)
		}
		return out
	case []any:
		out := make([]any, len(raw))
		for i, child := range raw {
			out[i] = policyCloneValue(child)
		}
		return out
	default:
		return value
	}
}
func (r *Repository) policyClasses() ([]string, error) {
	contract, err := r.Read("developer/policy/registries/developer-policy-catalog-contracts.json")
	if err != nil {
		return nil, err
	}
	return policyStrings(Map(Map(contract["$defs"])["developer_policy_class"])["enum"]), nil
}
func (r *Repository) ActiveFamilyIDs(class, family string) ([]string, error) {
	classes, err := r.policyClasses()
	if err != nil {
		return nil, err
	}
	families := append(append([]string{}, policyRootFamilies...), policyLegacyFamilies...)
	if !policyContains(classes, class) || !policyContains(families, family) {
		return nil, policyFailure("INVALID_AUTHORITY_FAMILY_KEY", "registered class and Family are required")
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for id, entry := range resolver.Index {
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			continue
		}
		if entry["policy_class"] != class || entry["status"] != "ACTIVE" {
			continue
		}
		parts := strings.Split(id, "-")
		if len(parts) != 3 || parts[1] != family {
			continue
		}
		if _, err := resolver.Policy(id); err != nil {
			return nil, policyFailure("AUTHORITY_METADATA_MISMATCH", err.Error())
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

// ValidateAnalysisSemantics checks the responsibility accounting, collision decisions,
// class boundaries and materialization grouping. It does not admit new authority.
func (r *Repository) ValidateAnalysisSemantics(payload Object) []string {
	classes, err := r.policyClasses()
	if err != nil {
		return []string{err.Error()}
	}
	families := append(append([]string{}, policyRootFamilies...), policyLegacyFamilies...)
	analysis := Map(payload["analysis"])
	if analysis == nil {
		return []string{"analysis must be a mapping"}
	}
	responsibilities, ok := analysis["responsibilities"].([]any)
	decision := Map(analysis["decision"])
	if !ok || decision == nil {
		return []string{"analysis responsibilities/decision must be present"}
	}
	errors := []string{}
	byID := map[string]Object{}
	ownedKeys := map[string]bool{}
	createIDs := map[string]bool{}
	blockedIDs := map[string]bool{}
	partial := false
	add := func(id, message string) { errors = append(errors, id+": "+message) }
	for _, raw := range responsibilities {
		unit := Map(raw)
		if unit == nil {
			errors = append(errors, "responsibility entry must be a mapping")
			continue
		}
		id, ok := unit["responsibility_id"].(string)
		if !ok {
			errors = append(errors, "responsibility_id must be a string")
			continue
		}
		if _, found := byID[id]; found {
			add(id, "duplicate responsibility_id")
			continue
		}
		byID[id] = unit
		relation := Text(unit["authority_relation"])
		family := Text(unit["family"])
		class := Text(unit["policy_class"])
		action := Text(unit["materialization_action"])
		if relation == "OWN" {
			if !policyContains(families, family) || !policyContains(classes, class) {
				add(id, "OWN responsibility requires explicit policy_class and one Family")
				continue
			}
			ownedKeys[class+":"+family] = true
			if unit["referenced_family"] != nil || unit["referenced_policy_class"] != nil {
				add(id, "OWN responsibility must not use referenced authority identity")
			}
			lookup := Map(unit["existing_authority_lookup"])
			if lookup == nil {
				add(id, "OWN responsibility requires existing_authority_lookup")
				continue
			}
			if lookup["searched_family"] != family {
				add(id, "lookup searched_family must equal owned Family")
			}
			if lookup["searched_policy_class"] != class {
				add(id, "lookup searched_policy_class must equal owned policy_class")
			}
			searched, valid := lookup["searched_policy_ids"].([]any)
			if !valid || !policyUnique(policyStrings(searched)) {
				add(id, "searched_policy_ids must be a unique list")
				searched = []any{}
			}
			comparisons, valid := lookup["candidate_comparisons"].([]any)
			if !valid {
				add(id, "candidate_comparisons must be a list")
				comparisons = []any{}
			}
			compared := map[string]bool{}
			blocker, newResolution := false, false
			for _, candidate := range comparisons {
				comparison := Map(candidate)
				if comparison == nil {
					add(id, "comparison must be a mapping")
					continue
				}
				target, valid := comparison["policy_id"].(string)
				if !valid {
					add(id, "comparison policy_id must be a string")
					continue
				}
				if compared[target] {
					add(id, "duplicate comparison for "+target)
				}
				compared[target] = true
				if !policyContains(policyStrings(searched), target) {
					add(id, "compared policy "+target+" was not searched")
				}
				collision := Text(comparison["collision_class"])
				resolution := Text(comparison["resolution_action"])
				scope := Text(comparison["scope_relation"])
				if !policyContains(policyCollisionResolutions[collision], resolution) {
					add(id, collision+" does not allow "+resolution)
				}
				if collision == "DISTINCT_SCOPED_AUTHORITY" && scope != "DISTINCT_SCOPE" {
					add(id, "DISTINCT_SCOPED_AUTHORITY requires DISTINCT_SCOPE")
				} else if collision != "DISTINCT_SCOPED_AUTHORITY" && len(policyCollisionResolutions[collision]) > 0 && scope != "SAME_SCOPE" {
					add(id, collision+" requires SAME_SCOPE")
				}
				if collision == "PARTIAL_OVERLAP" || collision == "CONFLICT" {
					blocker = true
					blockedIDs[id] = true
				}
				if collision == "PARTIAL_OVERLAP" {
					partial = true
				}
				if resolution == "CREATE_NEW_SIBLING_POLICY" || resolution == "COEXIST_SCOPED" {
					newResolution = true
				}
			}
			outcome := "NO_MATCH"
			if len(comparisons) > 0 {
				outcome = "MATCHES_FOUND"
			}
			if lookup["lookup_outcome"] != outcome {
				add(id, "lookup_outcome must be "+outcome)
			}
			expected := "USE_EXISTING_AUTHORITY"
			if blocker {
				expected = "BLOCKED"
			} else if len(comparisons) == 0 || newResolution {
				expected = "CREATE_NEW_POLICY"
			}
			if action != expected {
				add(id, "materialization_action must be "+expected)
			}
			if expected == "CREATE_NEW_POLICY" {
				createIDs[id] = true
				if _, valid := unit["target_group_id"].(string); !valid {
					add(id, "CREATE_NEW_POLICY requires target_group_id")
				}
			} else if unit["target_group_id"] != nil {
				add(id, "only CREATE_NEW_POLICY may declare target_group_id")
			}
		} else {
			if unit["policy_class"] != nil {
				add(id, "non-OWN responsibility must not own a policy_class")
			}
			referencedFamily, referencedClass := Text(unit["referenced_family"]), Text(unit["referenced_policy_class"])
			if (unit["referenced_family"] == nil) != (unit["referenced_policy_class"] == nil) || (unit["referenced_family"] != nil && (!policyContains(families, referencedFamily) || !policyContains(classes, referencedClass))) {
				add(id, "referenced authority requires an explicit registered policy_class and Family pair")
			}
			if !policyContains([]string{"OWN", "REFERENCE", "CONSUME", "VERIFY", "TRANSFORM", "EXECUTE"}, relation) {
				add(id, "unknown authority_relation "+relation)
			}
			if unit["family"] != nil {
				add(id, "non-OWN responsibility must not own a Family")
			}
			if unit["existing_authority_lookup"] != nil {
				add(id, "non-OWN responsibility must not perform owner authority lookup")
			}
			if action != "USE_EXISTING_AUTHORITY" {
				add(id, "non-OWN responsibility must USE_EXISTING_AUTHORITY")
			}
			if unit["target_group_id"] != nil {
				add(id, "non-OWN responsibility must not target a new group")
			}
		}
	}
	expectedKeys := []any{}
	for _, class := range classes {
		for _, family := range families {
			if ownedKeys[class+":"+family] {
				expectedKeys = append(expectedKeys, Object{"policy_class": class, "family": family})
			}
		}
	}
	if !reflect.DeepEqual(decision["owned_authority_family_set"], expectedKeys) {
		errors = append(errors, "decision.owned_authority_family_set must equal canonical owned authority Family set")
	}
	groups, valid := decision["materialization_groups"].([]any)
	if !valid {
		errors = append(errors, "decision.materialization_groups must be a list")
		groups = []any{}
	}
	groupIDs := map[string]bool{}
	grouped := map[string]bool{}
	for _, raw := range groups {
		group := Map(raw)
		if group == nil {
			errors = append(errors, "materialization group must be a mapping")
			continue
		}
		id, valid := group["group_id"].(string)
		if !valid {
			errors = append(errors, "materialization group_id must be a string")
			continue
		}
		if groupIDs[id] {
			add(id, "duplicate materialization group")
		}
		groupIDs[id] = true
		family, class := Text(group["family"]), Text(group["policy_class"])
		if !policyContains(families, family) || !policyContains(classes, class) {
			add(id, "materialization group requires explicit policy_class and Family")
		}
		members, valid := group["responsibility_ids"].([]any)
		if !valid || len(members) == 0 {
			add(id, "responsibility_ids must be non-empty")
			continue
		}
		if !policyUnique(policyStrings(members)) {
			add(id, "responsibility_ids must be unique")
		}
		for _, member := range members {
			memberID := Text(member)
			unit, found := byID[memberID]
			if !found {
				add(id, "unknown responsibility "+memberID)
				continue
			}
			if grouped[memberID] {
				add(memberID, "responsibility appears in multiple groups")
			}
			grouped[memberID] = true
			if unit["materialization_action"] != "CREATE_NEW_POLICY" {
				add(id, memberID+" is not a new-policy responsibility")
			}
			for _, field := range []string{"target_group_id", "family", "policy_class", "cohesion_key"} {
				expected := group[field]
				if field == "target_group_id" {
					expected = id
				}
				if unit[field] != expected {
					add(id, memberID+" "+field+" mismatch")
				}
			}
		}
	}
	if !policySetsEqual(grouped, createIDs) {
		errors = append(errors, "materialization_groups must cover every and only CREATE_NEW_POLICY responsibility")
	}
	split := len(expectedKeys) > 1 || len(groups) > 1 || partial
	if decision["split_required"] != split {
		errors = append(errors, fmt.Sprintf("decision.split_required must be %t", split))
	}
	allowed := len(blockedIDs) == 0
	if decision["materialization_allowed"] != allowed {
		errors = append(errors, fmt.Sprintf("decision.materialization_allowed must be %t", allowed))
	}
	return errors
}

func (r *Repository) analysisRecords() ([]Object, error) {
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		return nil, policyFailure("INVALID_ANALYSIS_REGISTRY", err.Error())
	}
	if err := r.Validate(policyAnalysisRegistrySchema, registry); err != nil {
		return nil, policyFailure("INVALID_ANALYSIS_REGISTRY", err.Error())
	}
	records := []Object{}
	for _, raw := range List(registry["records"]) {
		record := Map(raw)
		if record == nil {
			return nil, policyFailure("INVALID_ANALYSIS_REGISTRY", "record must be a mapping")
		}
		records = append(records, record)
	}
	return records, nil
}

func (r *Repository) ResolveAnalysisRecord(analysisID, subjectType, subjectID, analysisKind string) (Object, error) {
	records, err := r.analysisRecords()
	if err != nil {
		return nil, err
	}
	subjectMode := subjectType != "" || subjectID != "" || analysisKind != ""
	if analysisID != "" && subjectMode {
		return nil, policyFailure("ANALYSIS_LOOKUP_KEY_CONFLICT", "analysis_id and subject lookup keys are mutually exclusive")
	}
	matches := []Object{}
	if analysisID != "" {
		if !policyAnalysisID.MatchString(analysisID) {
			return nil, policyFailure("INVALID_ANALYSIS_ID", analysisID)
		}
		for _, record := range records {
			if record["analysis_id"] == analysisID {
				matches = append(matches, record)
			}
		}
	} else {
		if subjectType == "" || subjectID == "" || analysisKind == "" {
			return nil, policyFailure("ANALYSIS_LOOKUP_KEY_REQUIRED", "use analysis_id or subject_type + subject_id + analysis_kind")
		}
		for _, record := range records {
			if record["subject_type"] == subjectType && record["subject_id"] == subjectID && record["analysis_kind"] == analysisKind {
				matches = append(matches, record)
			}
		}
	}
	if len(matches) == 0 {
		return nil, policyFailure("RESPONSIBILITY_ANALYSIS_NOT_FOUND", "analysis registry has no exact matching record")
	}
	if len(matches) != 1 {
		return nil, policyFailure("RESPONSIBILITY_ANALYSIS_AMBIGUOUS", "analysis registry lookup must resolve exactly one record")
	}
	record := matches[0]
	id, reference := Text(record["analysis_id"]), Text(record["analysis_ref"])
	expected := policyAnalysisRecordRoot + id + ".yaml"
	if reference != expected {
		return nil, policyFailure("NONCANONICAL_ANALYSIS_REF", id+": expected "+expected+", got "+reference)
	}
	if _, err := r.Read(reference); err != nil {
		return nil, policyFailure("RESPONSIBILITY_ANALYSIS_NOT_FOUND", reference)
	}
	return record, nil
}

func (r *Repository) ValidateResponsibilityAnalysis(analysisID string, currentLookup bool) (Object, error) {
	record, err := r.ResolveAnalysisRecord(analysisID, "", "", "")
	if err != nil {
		return nil, err
	}
	relative := Text(record["analysis_ref"])
	payload, err := r.Read(relative)
	if err != nil {
		return nil, policyFailure("RESPONSIBILITY_ANALYSIS_NOT_FOUND", err.Error())
	}
	analysis := Map(payload["analysis"])
	if Text(analysis["analysis_id"]) != analysisID {
		return nil, policyFailure("ANALYSIS_ID_MISMATCH", relative)
	}
	errors := r.ValidateAnalysisSemantics(payload)
	if err := r.Validate(policyAnalysisSchema, payload); err != nil {
		errors = append(errors, err.Error())
	}
	if currentLookup {
		cache := map[string][]string{}
		resolver, err := NewResolver(r)
		if err != nil {
			return nil, err
		}
		for _, raw := range List(analysis["responsibilities"]) {
			unit := Map(raw)
			if unit["authority_relation"] != "OWN" {
				continue
			}
			id := Text(unit["responsibility_id"])
			family, class := Text(unit["family"]), Text(unit["policy_class"])
			lookup := Map(unit["existing_authority_lookup"])
			if lookup == nil {
				continue
			}
			key := class + ":" + family
			expected, ok := cache[key]
			if !ok {
				expected, err = r.ActiveFamilyIDs(class, family)
				if err != nil {
					errors = append(errors, err.Error())
					continue
				}
				cache[key] = expected
			}
			if !reflect.DeepEqual(policyStrings(lookup["searched_policy_ids"]), expected) {
				errors = append(errors, id+": searched_policy_ids must exactly cover current ACTIVE "+key+" policies")
			}
			for _, candidate := range List(lookup["candidate_comparisons"]) {
				comparison := Map(candidate)
				target := Text(comparison["policy_id"])
				if !policyContains(expected, target) {
					errors = append(errors, id+": comparison target is not current ACTIVE authority: "+target)
					continue
				}
				targetRecord, err := resolver.Policy(target)
				if err != nil {
					errors = append(errors, err.Error())
					continue
				}
				if section, ok := comparison["section"].(string); ok {
					if _, exists := Map(targetRecord["rules"])[section]; !exists {
						errors = append(errors, id+": compared section missing: "+target+":"+section)
					}
				}
			}
		}
	}
	if len(errors) > 0 {
		return nil, policyFailure("RESPONSIBILITY_ANALYSIS_BLOCKED", strings.Join(errors, "; "))
	}
	decision := Map(analysis["decision"])
	return Object{"status": "PASS", "analysis_id": analysisID, "analysis_ref": relative, "owned_authority_family_set": decision["owned_authority_family_set"], "split_required": decision["split_required"], "materialization_allowed": decision["materialization_allowed"], "materialization_groups": decision["materialization_groups"]}, nil
}
