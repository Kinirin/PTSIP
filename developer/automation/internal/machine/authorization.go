package machine

import (
	"fmt"
	"os"
	"sort"
)

func (r *Repository) AuthorizationReadiness() (Object, error) {
	transition, err := r.Read("developer/policy/registries/authorization-transition-registry.yaml")
	if err != nil {
		return nil, err
	}
	index, err := r.Read("src/policy/index.yaml")
	if err != nil {
		return nil, err
	}
	schema, err := r.Read("src/policy/registries/ptsip-support-authority-schema-registry.yaml")
	if err != nil {
		return nil, err
	}
	role, err := r.Read("src/policy/registries/ptsip-support-authority-role-registry.yaml")
	if err != nil {
		return nil, err
	}
	subject, err := r.Read("src/policy/registries/ptsip-support-authority-subject-registry.yaml")
	if err != nil {
		return nil, err
	}
	valid := index["policy_class"] == "PTSIP_SUPPORT_FEATURE"
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			continue
		}
		record, err := r.Read("src/policy/" + Text(entry["path"]))
		if err != nil {
			return nil, err
		}
		if record["policy_class"] != "PTSIP_SUPPORT_FEATURE" || Map(record["policy"])["id"] != entry["id"] || Map(record["policy"])["status"] != entry["status"] {
			valid = false
		}
		if err := r.Validate("src/policy/schemas/ptsip-support-root-family-policy.schema.json", record); err != nil {
			valid = false
		}
	}
	vocabulary := Map(role["effect_vocabulary"])
	_, supportIDPresent := Map(subject["subject_identity_schemes"])["SUPPORT_POLICY_ID"]
	provenance := Map(transition["authorization_provenance"])
	count := len(List(vocabulary["tokens"]))
	countValid := false
	switch value := vocabulary["count"].(type) {
	case int:
		countValid = value == count
	case float64:
		countValid = value == float64(count)
	}
	return Object{
		"AUTHORITY_SCHEMA_REGISTRY_VALID":               len(List(schema["entries"])) > 0,
		"AUTHORITY_ROLE_REGISTRY_VALID":                 len(Map(role["policy_roles"])) > 0,
		"AUTHORITY_SUBJECT_REGISTRY_VALID":              supportIDPresent,
		"CURRENT_SUPPORT_POLICY_CORPUS_VALID":           valid && len(List(index["policies"])) > 0,
		"ROLE_EFFECT_VOCABULARY_VALID":                  countValid,
		"SUPPORT_POLICY_SUBJECT_CONTRACT_VALID":         subject["repository_binding_policy"] == "SOLVE_SUBJECT_PROVIDED_NO_BUILTIN_CURRENT_REPOSITORY",
		"PROJECT_AUTHORITY_RUNTIME_OWNER_PREAUTHORIZED": provenance["authority"] == "PROJECT_OWNER" && provenance["type"] == "GIT_COMMIT" && len(Text(provenance["revision"])) == 40,
	}, nil
}
func (r *Repository) AuthorizationTransition(scope string, readiness Object) (Object, error) {
	registry, err := r.Read("developer/policy/registries/authorization-transition-registry.yaml")
	if err != nil {
		return nil, err
	}
	rules := Map(registry["rules"])
	if rules == nil {
		return nil, Fail("AUTHORIZATION_TRANSITION_REGISTRY_INVALID", "rules must be mapping")
	}
	ids := []string{}
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		rule := Map(rules[id])
		if !Has(Strings(rule["target_scopes"]), scope) {
			continue
		}
		predicates := Map(rule["required_predicates"])
		if predicates == nil {
			return nil, Fail("AUTHORIZATION_TRANSITION_REGISTRY_INVALID", "predicates must be mapping")
		}
		failed := []string{}
		blockers := []string{}
		for name, expected := range predicates {
			if readiness[name] != expected {
				failed = append(failed, name)
			}
		}
		sort.Strings(failed)
		for _, name := range failed {
			blockers = append(blockers, "PREDICATE_NOT_SATISFIED:"+name)
		}
		state := rule["when_all_true"]
		if len(failed) > 0 {
			state = rule["otherwise"]
		}
		if !Has(Strings(registry["states"]), Text(state)) {
			return nil, Fail("AUTHORIZATION_STATE_UNREGISTERED", Text(state))
		}
		return Object{"scope": scope, "state": state, "rule_id": id, "failed_predicates": failed, "blockers": blockers}, nil
	}
	if hold := Map(Map(registry["held_scopes"])[scope]); hold != nil {
		state := hold["state"]
		if state == nil {
			state = "HOLD_NOT_AUTHORIZED"
		}
		return Object{"scope": scope, "state": state, "rule_id": nil, "failed_predicates": []string{}, "blockers": Strings(hold["blockers"])}, nil
	}
	return Object{"scope": scope, "state": "HOLD_NOT_AUTHORIZED", "rule_id": nil, "failed_predicates": []string{}, "blockers": []string{"NO_OWNER_PREAUTHORIZED_TRANSITION_RULE"}}, nil
}
func (r *Repository) SupportGroupState(name, desired string) error {
	refs, err := r.Read("developer/policy/contracts/go-support-audit-refs.v1.json")
	if err != nil {
		return err
	}
	group := List(Map(refs["state_groups"])[name])
	if len(group) == 0 {
		return Fail("SUPPORT_STATE_GROUP_UNREGISTERED", name)
	}
	index, err := r.Read("src/policy/index.yaml")
	if err != nil {
		return err
	}
	byID := map[string]Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		byID[Text(entry["id"])] = entry
	}
	for _, raw := range group {
		id := Text(raw)
		entry := byID[id]
		if entry == nil || entry["status"] != desired {
			return Fail("ACTIVATION_SUPPORT_STATE_MISMATCH", id)
		}
		record, err := r.Read("src/policy/" + Text(entry["path"]))
		if err != nil {
			return err
		}
		if record["policy_class"] != "PTSIP_SUPPORT_FEATURE" || Map(record["policy"])["status"] != desired {
			return Fail("ACTIVATION_SUPPORT_STATE_MISMATCH", id)
		}
	}
	return nil
}
func (r *Repository) ActivationPreflight() (Object, error) {
	record, err := r.ActivationRecord()
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, Fail("EXPLICIT_ACTIVATION_APPROVAL_REQUIRED", "no registered activation receipt")
	}
	head, err := r.GitOutput("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if head != record["base_head"] {
		return nil, Fail("ACTIVATION_BASE_HEAD_CHANGED", "HEAD differs from approved preimage")
	}
	catalog, err := r.Read("src/vpms/contracts/index.json")
	if err != nil {
		return nil, err
	}
	for _, id := range Map(catalog["entrypoints"]) {
		contract, err := r.ProductContract(Text(id), false)
		if err != nil {
			return nil, err
		}
		if contract["status"] != "APPROVED" || contract["runtime_enabled"] != false {
			return nil, Fail("ACTIVATION_PREIMAGE_NOT_APPROVED_INACTIVE", Text(id))
		}
	}
	if err := r.SupportGroupState("retired_selector", "ACTIVE"); err != nil {
		return nil, err
	}
	if err := r.SupportGroupState("integration", "DRAFT"); err != nil {
		return nil, err
	}
	return Object{"status": "READY", "base_head": head, "approved_target_count": len(List(record["mutation_targets"]))}, nil
}
func (r *Repository) VerifyActivation(check bool) (Object, error) {
	record, err := r.ActivationRecord()
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, Fail("EXPLICIT_ACTIVATION_APPROVAL_REQUIRED", "no registered activation receipt")
	}
	registered, err := r.VerifyContractRegistration(false)
	if err != nil {
		return nil, err
	}
	implementation, err := r.VerifyAPIImplementation(false)
	if err != nil {
		return nil, err
	}
	if err := r.SupportGroupState("retired_selector", "RETIRED"); err != nil {
		return nil, err
	}
	if err := r.SupportGroupState("integration", "ACTIVE"); err != nil {
		return nil, err
	}
	provider, err := r.SupportField("provider_binding")
	if err != nil {
		return nil, err
	}
	if fmt.Sprint(provider) != fmt.Sprint(record["provider_binding"]) {
		return nil, Fail("ACTIVATION_PROVIDER_BINDING_MISMATCH", "provider binding differs")
	}
	selector, err := r.Path("src/vpms/domain/selector.py")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(selector); !os.IsNotExist(err) {
		return nil, Fail("RETIRED_SOURCE_PRESENT", "src/vpms/domain/selector.py")
	}
	for _, relative := range []string{"src/vpms/__init__.py", "src/vpms/execution/runner.py"} {
		for _, name := range []string{"SelectionScope", "select_cases", "run_selected_cases"} {
			exists, err := PythonIdentifierPresent(r, relative, name)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, Fail("RETIRED_API_PRESENT", name)
			}
		}
	}
	for _, relative := range Strings(record["mutation_targets"]) {
		if relative == "src/vpms/domain/selector.py" {
			continue
		}
		path, err := r.Path(relative)
		if err != nil {
			return nil, err
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return nil, Fail("MISSING_ACTIVATION_TARGET", relative)
		}
	}
	if err := r.ExactAuditScope(record, Strings(record["mutation_targets"]), check); err != nil {
		return nil, err
	}
	return Object{"status": "ACTIVE_VERIFIED", "product_contract_count": registered["product_contract_count"], "runtime_enabled": registered["runtime_enabled"], "support_targets": record["support_targets"], "implementation_status": implementation["status"], "approved_target_count": len(List(record["mutation_targets"]))}, nil
}
func init() {
	RegisterOperations("authorization-transition", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		ready, err := r.AuthorizationReadiness()
		if err != nil {
			return nil, err
		}
		if command == "readiness" {
			return ready, nil
		}
		if opts["--input"] != "" {
			ready, err = r.Read(opts["--input"])
			if err != nil {
				return nil, err
			}
		}
		if command == "evaluate" {
			return r.AuthorizationTransition(opts["--scope"], ready)
		}
		results := []any{}
		for _, scope := range []string{"PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME", "PROJECT_AUTHORITY_PROJECTION_RUNTIME", "PROJECT_AUTHORITY_RECORD_MATERIALIZATION", "AUTHORIZATION_READINESS_TRANSITION_ENGINE"} {
			result, err := r.AuthorizationTransition(scope, ready)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return Object{"readiness": ready, "results": results}, nil
	})
	RegisterOperations("vpms-activation", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		if command == "preflight" {
			return r.ActivationPreflight()
		}
		return r.VerifyActivation(BoolOption(opts, "check-worktree"))
	})
}
