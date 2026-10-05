package machine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

const DeveloperClass = "PTSIP_DEVELOPER_POLICY"
const ResolverContract = "developer/policy/contracts/go-policy-resolver.v1.yaml"

var rootID = regexp.MustCompile(`^MPD-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD)-[0-9]{4}$`)
var operations = []string{"MODIFY", "PLAN", "READ", "RELEASE", "VERIFY"}

type Resolver struct {
	Repo     *Repository
	Contract Object
	Index    map[string]Object
	Bindings map[string]Object
	ID       string
}

func NewResolver(repo *Repository) (*Resolver, error) {
	contract, err := repo.Read(ResolverContract)
	if err != nil {
		return nil, err
	}
	config := Map(contract["resolver"])
	if contract["policy_class"] != DeveloperClass || config["runtime_classification"] != "DEVELOPER_CONTROL_PLANE_ONLY" {
		return nil, fmt.Errorf("invalid developer resolver contract class")
	}
	lookup := Map(contract["lookup"])
	required := map[string]string{
		"scope_resolution": "EXACT_ANCESTOR_PATH", "descendant_inheritance": "EXPLICIT_SCOPE_CONTROL",
		"exact_scope_binding": "ALWAYS_ELIGIBLE", "ancestor_binding_requirement": "INHERIT_TO_DESCENDANTS_TRUE",
		"non_inheritable_ancestor": "SKIP_AND_CONTINUE", "no_eligible_binding": "FAIL_CLOSED",
		"operation_resolution": "EXACT_CLOSED_VOCABULARY", "semantic_similarity": "FORBIDDEN",
		"repository_document_scan": "FORBIDDEN", "git_history_lookup": "FORBIDDEN", "natural_language_fallback": "FORBIDDEN",
		"unresolved_required_binding": "FAIL_CLOSED",
	}
	for name, expected := range required {
		if lookup[name] != expected {
			return nil, fmt.Errorf("lookup.%s must be %s", name, expected)
		}
	}
	authority := Map(contract["authority"])
	if authority["canonical_policy_records_remain_authority"] != true || authority["resolver_projection_is_authority"] != false || authority["resolver_may_create_or_rank_policy_semantics"] != false || authority["product_runtime_dependency"] != "FORBIDDEN" {
		return nil, fmt.Errorf("invalid resolver authority boundary")
	}
	if config["task_context_ref"] != nil {
		return nil, fmt.Errorf("unregistered Go task context adapter; fail closed")
	}
	if config["policy_index_ref"] != "developer/policy/index.yaml" {
		return nil, fmt.Errorf("noncanonical developer catalog")
	}
	catalog, err := repo.Read(Text(config["policy_index_ref"]))
	if err != nil {
		return nil, err
	}
	if err := repo.Validate("developer/policy/schemas/developer-policy-catalog.schema.json", catalog); err != nil {
		return nil, err
	}
	index := map[string]Object{}
	last := ""
	for _, value := range List(catalog["policies"]) {
		entry := Map(value)
		id := Text(entry["id"])
		if id <= last {
			return nil, fmt.Errorf("catalog IDs must be unique and ordered")
		}
		last = id
		index[id] = entry
	}
	resolver := &Resolver{Repo: repo, Contract: contract, Index: index, Bindings: map[string]Object{}, ID: Text(config["id"])}
	if contract["schema_version"] == "ptsip-policy-resolver-contract/v3" {
		ownerID := Text(config["owner_policy_ref"])
		owner, err := resolver.Policy(ownerID)
		if err != nil {
			return nil, err
		}
		section := Map(Map(owner["rules"])[Text(config["owner_section"])])
		if Map(owner["policy"])["status"] != "ACTIVE" || section["resolver_contract_ref"] != ResolverContract || section["policy_reference_model"] != "EXACT_ROOT_POLICY_ID_AND_SECTION" || section["legacy_policy_id_fallback"] != "FORBIDDEN" {
			return nil, fmt.Errorf("resolver owner contract not admitted")
		}
	} else {
		return nil, fmt.Errorf("Go resolver requires admitted direct Root contract v3")
	}
	registry, err := repo.Read(Text(config["binding_registry_ref"]))
	if err != nil {
		return nil, err
	}
	if len(registry) != 6 || registry["schema_version"] != "ptsip-policy-resolver-binding-registry/v1" || registry["policy_class"] != DeveloperClass || registry["resolver_id"] != resolver.ID {
		return nil, fmt.Errorf("invalid binding registry")
	}
	vocabulary := []string{}
	for _, item := range List(registry["operation_vocabulary"]) {
		vocabulary = append(vocabulary, Text(item))
	}
	sort.Strings(vocabulary)
	if strings.Join(vocabulary, ",") != strings.Join(operations, ",") {
		return nil, fmt.Errorf("invalid closed operation vocabulary")
	}
	file, err := repo.Path(Text(registry["binding_records_ref"]))
	if err != nil {
		return nil, err
	}
	opened, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	scanner := bufio.NewScanner(opened)
	scanner.Buffer(make([]byte, 4096), 16*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var binding Object
		if err := json.Unmarshal(scanner.Bytes(), &binding); err != nil {
			return nil, fmt.Errorf("binding line %d: %w", line, err)
		}
		if err := repo.Validate(Text(registry["binding_record_schema_ref"]), binding); err != nil {
			return nil, fmt.Errorf("binding line %d: %w", line, err)
		}
		scope := Text(binding["scope"])
		normalized, err := repo.Scope(scope)
		if err != nil || normalized != scope {
			return nil, fmt.Errorf("binding scope must be canonical: %s", scope)
		}
		if _, exists := resolver.Bindings[scope]; exists {
			return nil, fmt.Errorf("duplicate binding scope: %s", scope)
		}
		resolver.Bindings[scope] = binding
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if line == 0 {
		return nil, fmt.Errorf("empty binding source")
	}
	return resolver, nil
}

// Policy reads only the indexed canonical record. Migration source identities are rejected.
func (r *Resolver) Policy(id string) (Object, error) {
	entry, ok := r.Index[id]
	if !ok {
		return nil, fmt.Errorf("unknown developer policy identity: %s", id)
	}
	if entry["authority_role"] == "MIGRATION_SOURCE" {
		return nil, fmt.Errorf("legacy policy identity is audit-only: %s", id)
	}
	if entry["policy_class"] == DeveloperClass {
		match := rootID.FindStringSubmatch(id)
		if match == nil {
			return nil, fmt.Errorf("developer policy requires Root Family identity: %s", id)
		}
		expected := "developer/policy/" + match[1] + "/" + id + ".yaml"
		if entry["path"] != expected {
			return nil, fmt.Errorf("%s: canonical Root path mismatch", id)
		}
	}
	record, err := r.Repo.Read(Text(entry["path"]))
	if err != nil {
		return nil, err
	}
	identity := Map(record["policy"])
	if identity["id"] != id || identity["status"] != entry["status"] || record["policy_class"] != entry["policy_class"] {
		return nil, fmt.Errorf("%s: catalog metadata mismatch", id)
	}
	if record["policy_class"] == DeveloperClass {
		match := rootID.FindStringSubmatch(id)
		if record["responsibility_family"] != match[1] {
			return nil, fmt.Errorf("%s: family mismatch", id)
		}
		if err := r.Repo.Validate("developer/policy/schemas/root-family-policy.schema.json", record); err != nil {
			return nil, err
		}
	}
	return record, nil
}

func (r *Resolver) reference(value any) (Object, error) {
	reference := Map(value)
	id := Text(reference["policy_id"])
	record, err := r.Policy(id)
	if err != nil {
		return nil, err
	}
	if Map(record["policy"])["status"] != "ACTIVE" {
		return nil, fmt.Errorf("binding references non-active policy %s", id)
	}
	rules := Map(record["rules"])
	sections := List(reference["sections"])
	if len(sections) == 0 {
		return nil, fmt.Errorf("%s: empty sections", id)
	}
	seen := map[string]bool{}
	for _, item := range sections {
		section := Text(item)
		if _, exists := rules[section]; !exists || seen[section] {
			return nil, fmt.Errorf("%s: invalid or duplicate section %q", id, section)
		}
		seen[section] = true
	}
	entry := r.Index[id]
	return Object{"policy_id": id, "path": entry["path"], "status": entry["status"], "sections": sections}, nil
}

func (r *Resolver) Resolve(scope, operation string) (Object, error) {
	normalized, err := r.Repo.Scope(scope)
	if err != nil {
		return nil, err
	}
	operation = strings.ToUpper(strings.TrimSpace(operation))
	found := false
	for _, candidate := range operations {
		if candidate == operation {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("unsupported policy operation: %s", operation)
	}
	current := normalized
	for {
		binding, exists := r.Bindings[current]
		if exists && (current == normalized || binding["inherit_to_descendants"] == true) {
			refs := List(binding["default_refs"])
			if selected, ok := Map(binding["operations"])[operation]; ok {
				refs = List(selected)
			}
			if len(refs) == 0 {
				return nil, fmt.Errorf("no policy refs for %s:%s", current, operation)
			}
			resolved := []any{}
			seen := map[string]bool{}
			for _, ref := range refs {
				value, err := r.reference(ref)
				if err != nil {
					return nil, err
				}
				id := Text(value["policy_id"])
				if seen[id] {
					return nil, fmt.Errorf("duplicate policy %s", id)
				}
				seen[id] = true
				resolved = append(resolved, value)
			}
			return Object{"schema_version": "ptsip-policy-resolution/v1", "resolver_id": r.ID,
				"scope": normalized, "operation": operation, "binding_scope": current, "policies": resolved,
				"authority": "CANONICAL_POLICY_RECORDS", "projection_authority": false}, nil
		}
		if current == "." {
			return nil, fmt.Errorf("no eligible policy binding for %q", normalized)
		}
		current = path.Dir(current)
	}
}

func (r *Resolver) Get(id, section string) (Object, error) {
	record, err := r.Policy(id)
	if err != nil {
		return nil, err
	}
	var value any = record
	var fragment any
	if section != "" {
		selected, exists := Map(record["rules"])[section]
		if !exists {
			return nil, fmt.Errorf("%s: unknown rule section %q", id, section)
		}
		value = selected
		fragment = "rules." + section
	}
	return Object{"schema_version": "ptsip-policy-get/v1", "policy_id": id, "canonical_path": r.Index[id]["path"], "fragment": fragment, "record": value}, nil
}

func (r *Resolver) Explain(id string) (Object, error) {
	record, err := r.Policy(id)
	if err != nil {
		return nil, err
	}
	sections := []string{}
	for section := range Map(record["rules"]) {
		sections = append(sections, section)
	}
	sort.Strings(sections)
	identity := Map(record["policy"])
	return Object{"schema_version": "ptsip-policy-explain/v1", "policy_id": id, "title": identity["title"], "status": identity["status"], "canonical_path": r.Index[id]["path"], "rule_sections": sections}, nil
}

func (r *Resolver) ValidateBindings() error {
	for scope, binding := range r.Bindings {
		lists := []any{binding["default_refs"]}
		for _, refs := range Map(binding["operations"]) {
			lists = append(lists, refs)
		}
		for _, list := range lists {
			seen := map[string]bool{}
			for _, ref := range List(list) {
				value, err := r.reference(ref)
				if err != nil {
					return fmt.Errorf("%s: %w", scope, err)
				}
				id := Text(value["policy_id"])
				if seen[id] {
					return fmt.Errorf("%s: duplicate policy %s", scope, id)
				}
				seen[id] = true
			}
		}
	}
	return nil
}

func (r *Resolver) Rule(id string) (Object, error) {
	config := Map(r.Contract["resolver"])
	source, err := r.Repo.Read(Text(config["normative_rule_registry_ref"]))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var found Object
	for _, value := range List(Map(source["ptsip_registry"])["rules"]) {
		rule := Map(value)
		ruleID := Text(rule["id"])
		if ruleID == "" || seen[ruleID] {
			return nil, fmt.Errorf("invalid or duplicate normative rule identity")
		}
		seen[ruleID] = true
		if ruleID == id {
			found = rule
		}
	}
	if found == nil {
		return nil, fmt.Errorf("unknown normative rule identity: %s", id)
	}
	return Object{"schema_version": "ptsip-normative-rule-projection/v2", "rule_id": id, "canonical_source": config["normative_rule_registry_ref"], "registry_record": found, "projection_authority": false}, nil
}
