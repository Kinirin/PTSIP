package usercontracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Result struct {
	Valid                   bool             `json:"valid"`
	Diagnostics             []Issue          `json:"diagnostics"`
	UserContracts           []ContractResult `json:"user_contracts"`
	RAUsage                 []RAUsage        `json:"ra_usage"`
	ExecutionAuthorized     bool             `json:"execution_authorized"`
	PTSIPConformanceVerdict string           `json:"ptsip_conformance_verdict"`
}

func result(issues []Issue, contracts []ContractResult, usage []RAUsage) Result {
	if issues == nil {
		issues = []Issue{}
	}
	if contracts == nil {
		contracts = []ContractResult{}
	}
	if usage == nil {
		usage = []RAUsage{}
	}
	return Result{
		Valid: len(issues) == 0, Diagnostics: issues, UserContracts: contracts, RAUsage: usage,
		ExecutionAuthorized: true, PTSIPConformanceVerdict: "NOT_ASSERTED",
	}
}

func validateInput(operation string, input map[string]any) (string, error) {
	var field, required string
	allowed := map[string]bool{}
	switch operation {
	case "validation.policy.identity", "validation.policy.rules":
		field = "Policy_Identity"
		if operation == "validation.policy.rules" {
			field = "Policy_Rules"
		}
		required = "policy"
		allowed = map[string]bool{"policy": true, "project_root": true, "bootstrap_user_contracts": true}
	case "validation.rbac.ra", "validation.rbac.rac", "validation.rbac.aec":
		required = "document"
		allowed["document"] = true
	case "validation.rbac.references":
		required = "policy"
		allowed = map[string]bool{"policy": true, "ra": true, "rac": true, "aec": true}
	default:
		return "", fmt.Errorf("unknown operation %q", operation)
	}
	for name := range input {
		if !allowed[name] {
			return "", fmt.Errorf("undeclared operation input %q", name)
		}
	}
	document, ok := input[required].(map[string]any)
	if !ok {
		return "", fmt.Errorf("expected object field %q", required)
	}
	if field != "" {
		if _, exists := document[field]; !exists {
			return "", fmt.Errorf("policy requires %s", field)
		}
	}
	if value, exists := input["project_root"]; exists {
		if root, ok := value.(string); !ok || root == "" {
			return "", errors.New("project_root must be a nonempty string")
		}
	}
	if value, exists := input["bootstrap_user_contracts"]; exists {
		if _, ok := value.(bool); !ok {
			return "", errors.New("bootstrap_user_contracts must be a boolean")
		}
	}
	for _, key := range []string{"ra", "rac", "aec"} {
		if value, exists := input[key]; exists && object(value) == nil {
			return "", fmt.Errorf("%s must be an object", key)
		}
	}
	return field, nil
}

func (v *Validator) policyOperation(field string, input map[string]any) (Result, error) {
	document := object(input["policy"])
	policy := map[string]any{field: document[field]}
	prefixes := []string{field}
	if field == "Policy_Rules" {
		prefixes = append(prefixes, "Policy_Content.items[].id")
		if content, exists := document["Policy_Content"]; exists {
			policy["Policy_Content"] = content
		}
	}
	issues := v.StructureIssues("ptsip.policy.v4", policy)
	if len(issues) != 0 {
		return result(issues, nil, nil), nil
	}
	projectRoot, _ := input["project_root"].(string)
	bootstrap, _ := input["bootstrap_user_contracts"].(bool)
	if field == "Policy_Identity" && bootstrap {
		if err := BootstrapVersionContract(projectRoot); err != nil {
			return Result{}, err
		}
	}
	contracts, err := v.ValidateUserContracts(policy, projectRoot, bootstrap, prefixes)
	if err != nil {
		return Result{}, err
	}
	customVersion := false
	for _, contract := range contracts {
		isVersion := false
		for _, target := range contract.Targets {
			isVersion = isVersion || target == VersionTarget
		}
		if isVersion {
			customVersion = customVersion || contract.Status == "VALID"
		} else if contract.Status == "INVALID" || contract.Status == "UNSUPPORTED" || contract.Status == "UNBOUND" {
			issues = append(issues, contract.Issues...)
		}
	}
	if field == "Policy_Identity" {
		if version, exists := object(policy[field])["Version"]; exists && !customVersion && !v.OfficialVersionValid(version) {
			issues = append(issues, Issue{
				Code: "VERSION", Path: []any{field, "Version"},
				Message: "Version must match an official format or a configured user version contract",
			})
		}
	} else {
		references, _ := policyReferences(policy, nil, true)
		issues = append(issues, references...)
	}
	return result(issues, contracts, nil), nil
}

func referencePolicy(policy map[string]any) map[string]any {
	rawRules, present := policy["Policy_Rules"]
	if !present {
		return map[string]any{}
	}
	rules, ok := rawRules.(map[string]any)
	if !ok {
		return map[string]any{"Policy_Rules": rawRules}
	}
	rawEntries, exists := rules["rules"]
	if !exists {
		return map[string]any{}
	}
	entries, ok := rawEntries.([]any)
	if !ok {
		return map[string]any{"Policy_Rules": map[string]any{"rules": rawEntries}}
	}
	selected := make([]any, 0, len(entries))
	for _, entry := range entries {
		rule, ok := entry.(map[string]any)
		if !ok {
			selected = append(selected, entry)
			continue
		}
		projected := map[string]any{}
		if value, exists := rule["RA_id"]; exists {
			projected["RA_id"] = value
		}
		selected = append(selected, projected)
	}
	return map[string]any{"Policy_Rules": map[string]any{"rules": selected}}
}

func (v *Validator) EvaluateOperation(operation string, input map[string]any) (Result, error) {
	field, err := validateInput(operation, input)
	if err != nil {
		return Result{}, err
	}
	if field != "" {
		return v.policyOperation(field, input)
	}
	if operation == "validation.rbac.references" {
		policy := referencePolicy(object(input["policy"]))
		issues := v.StructureIssues("ptsip.policy.v4", policy)
		if len(issues) != 0 {
			return result(issues, nil, nil), nil
		}
		indexed, issues := v.rbacIndex(object(input["ra"]), object(input["rac"]), object(input["aec"]))
		references, usage := policyReferences(policy, indexed, false)
		return result(append(issues, references...), nil, usage), nil
	}
	kind := map[string]string{
		"validation.rbac.ra": "RA", "validation.rbac.rac": "RAC", "validation.rbac.aec": "AEC",
	}[operation]
	document := object(input["document"])
	issues := v.StructureIssues(kind+"_list", document)
	if len(issues) == 0 {
		path := []any{kind + "_list"}
		if kind == "RA" {
			path = append(path, "entries")
		}
		_, duplicates := indexRecords(listRecords(kind, document), kind+"_id", path)
		issues = append(issues, duplicates...)
	}
	return result(issues, nil, nil), nil
}

func RunWorker(root string, input io.Reader, output io.Writer) error {
	const limit = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(input, limit+1))
	if err != nil {
		return err
	}
	if len(raw) > limit {
		return errors.New("user-contract worker request exceeds 1 MiB")
	}
	decoded, err := ParseJSON(raw)
	if err != nil {
		return err
	}
	request := object(decoded)
	operation, ok := request["operation_id"].(string)
	if !ok || len(request) != 2 {
		return errors.New("worker request requires exactly operation_id and input")
	}
	data, ok := request["input"].(map[string]any)
	if !ok {
		return errors.New("worker input must be an object")
	}
	validator, err := New(root)
	if err != nil {
		return err
	}
	evaluated, err := validator.EvaluateOperation(operation, data)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(evaluated)
}
