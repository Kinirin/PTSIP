// Package applicability makes read-only, evidence-bound RA/RAC/AEC applicability
// assessments. It is not an authorization, policy migration, or Rego evaluator.
package applicability

import (
    "errors"
    "fmt"
    "path"
    "regexp"
    "sort"
    "strings"
)

const InputVersion = "ptsip-policy-applicability-input/v1"
const OutputVersion = "ptsip-policy-applicability-assessment/v1"

var sha40 = regexp.MustCompile(`^[a-f0-9]{40}$`)
var kinds = []string{"RA", "RAC", "AEC"}

type Reference struct {
    SourceRef string `json:"source_ref"`
    ItemRef string `json:"item_ref"`
    RevisionRef string `json:"revision_ref"`
}
type Binding struct {
    State string `json:"state"`
    BasisRefs []string `json:"basis_refs"`
    ContractRef *Reference `json:"contract_ref,omitempty"`
}
type Rule struct {
    ID string `json:"id"`
    Kind string `json:"kind"`
    SourceRef string `json:"source_ref"`
    ContractBindings map[string]Binding `json:"contract_bindings"`
}
type SourceItem struct {
    ID string `json:"id"`
    SourcePointer string `json:"source_pointer"`
}
type Policy struct {
    ID string `json:"policy_id"`
    Path string `json:"path"`
    Plane string `json:"plane"`
    Family string `json:"family"`
    CatalogStatus string `json:"catalog_status"`
    DocumentStatus string `json:"document_status"`
    Rules []Rule `json:"rules"`
    Items []SourceItem `json:"items"`
}
type Contract struct {
    SourceRef string `json:"source_ref"`
    ActivationState string `json:"activation_state"`
    DefinitionState string `json:"definition_state,omitempty"`
}
type Input struct {
    SchemaVersion string `json:"schema_version"`
    SourceHead string `json:"source_head"`
    Contracts map[string]Contract `json:"contracts"`
    Policies []Policy `json:"policies"`
}
type Decision struct {
    PolicyID string `json:"policy_id"`
    RuleID string `json:"rule_id"`
    ContractKind string `json:"contract_kind"`
    PolicyPath string `json:"policy_path"`
    SourcePointer string `json:"source_pointer"`
    State string `json:"state"`
    Reason string `json:"reason"`
    EvidenceRefs []string `json:"evidence_refs"`
    ContractRef *Reference `json:"contract_ref,omitempty"`
    EvaluationAuthorized bool `json:"evaluation_authorized"`
    ExecutionAuthorized bool `json:"execution_authorized"`
}
type Output struct {
    SchemaVersion string `json:"schema_version"`
    SourceHead string `json:"source_head"`
    Authority string `json:"authority"`
    AssessmentScope string `json:"assessment_scope"`
    PolicyCount int `json:"policy_count"`
    RuleCount int `json:"rule_count"`
    BindingCount int `json:"binding_count"`
    Counts map[string]int `json:"counts"`
    Decisions []Decision `json:"decisions"`
    EvaluationAuthorized bool `json:"evaluation_authorized"`
    ExecutionAuthorized bool `json:"execution_authorized"`
    PolicyMutationAuthorized bool `json:"policy_mutation_authorized"`
}

func safePath(s string) bool {
    if s == "" || strings.HasPrefix(s, "/") || strings.Contains(s, "\\") || path.Clean(s) != s {
        return false
    }
    for _, part := range strings.Split(s, "/") {
        if part == ".." || part == "." || part == "" {
            return false
        }
    }
    return true
}
func statusRoot(s string) string {
    if i := strings.IndexByte(s, '.'); i >= 0 { return s[:i] }
    return s
}
func effective(s string) bool { return s == "ACTIVE" || s == "APPROVED" }
func pointer(s string) bool { return s == "#" || s == "" || strings.HasPrefix(s, "/") }
func exactRef(ref Reference) bool {
    return safePath(ref.SourceRef) && pointer(ref.ItemRef) &&
        strings.HasPrefix(ref.RevisionRef, "git:") &&
        sha40.MatchString(strings.TrimPrefix(ref.RevisionRef, "git:"))
}

// Assess does not infer applicability from family names, field names, or an
// upstream policy_basis reference. Only explicit rule-local bindings are evidence.
func Assess(input Input) (Output, error) {
    out := Output{
        SchemaVersion: OutputVersion, SourceHead: input.SourceHead,
        Authority: "NONE", AssessmentScope: "PROVISIONAL_PER_RULE_CONTRACT_USE_ONLY",
        Counts: map[string]int{}, Decisions: []Decision{},
    }
    if input.SchemaVersion != InputVersion || !sha40.MatchString(input.SourceHead) {
        return out, errors.New("INPUT_VERSION_OR_REVISION_INVALID")
    }
    if len(input.Contracts) != len(kinds) {
        return out, errors.New("EXACT_CONTRACT_KIND_REGISTRY_REQUIRED")
    }
    for _, kind := range kinds {
        contract, ok := input.Contracts[kind]
        if !ok || !safePath(contract.SourceRef) || contract.ActivationState == "" {
            return out, fmt.Errorf("CONTRACT_REGISTRATION_INCOMPLETE: %s", kind)
        }
        if !strings.HasPrefix(contract.SourceRef, ".ptsip/contracts/compile-sealed-rbac/") {
            return out, fmt.Errorf("CONTRACT_OUTSIDE_CANONICAL_ROOT: %s", kind)
        }
    }
    seenPolicies := map[string]bool{}
    for _, policy := range input.Policies {
        if policy.ID == "" || seenPolicies[policy.ID] || !safePath(policy.Path) {
            return out, fmt.Errorf("POLICY_ID_OR_PATH_INVALID: %s", policy.ID)
        }
        seenPolicies[policy.ID] = true
        prefix := "developer/policy/"
        expectedPlane := "PTSIP_DEVELOPER_POLICY"
        if policy.Plane == "PTSIP_SUPPORT_FEATURE" {
            prefix, expectedPlane = "src/policy/", "PTSIP_SUPPORT_FEATURE"
        }
        if policy.Plane != expectedPlane || !strings.HasPrefix(policy.Path, prefix+policy.Family+"/") ||
            path.Base(policy.Path) != policy.ID+".yaml" {
            return out, fmt.Errorf("POLICY_SCOPE_MISMATCH: %s", policy.ID)
        }
        if statusRoot(policy.CatalogStatus) != statusRoot(policy.DocumentStatus) {
            return out, fmt.Errorf("POLICY_LIFECYCLE_CONFLICT: %s", policy.ID)
        }
        items := map[string]string{}
        for _, item := range policy.Items {
            if item.ID == "" || item.SourcePointer == "" || items[item.ID] != "" {
                return out, fmt.Errorf("DUPLICATE_OR_EMPTY_SOURCE_ITEM: %s", policy.ID)
            }
            items[item.ID] = item.SourcePointer
        }
        rules := map[string]bool{}
        for _, rule := range policy.Rules {
            if rule.ID == "" || rules[rule.ID] || items[rule.SourceRef] == "" ||
                (rule.Kind != "RULE" && rule.Kind != "REQUIREMENT" && rule.Kind != "SEMANTIC_UNIT") {
                return out, fmt.Errorf("RULE_ID_OR_SOURCE_INVALID: %s/%s", policy.ID, rule.ID)
            }
            rules[rule.ID] = true
            if len(rule.ContractBindings) != len(kinds) {
                return out, fmt.Errorf("RULE_CONTRACT_SET_INCOMPLETE: %s/%s", policy.ID, rule.ID)
            }
            out.RuleCount++
            for _, kind := range kinds {
                binding, ok := rule.ContractBindings[kind]
                if !ok || len(binding.BasisRefs) == 0 {
                    return out, fmt.Errorf("BINDING_MISSING: %s/%s/%s", policy.ID, rule.ID, kind)
                }
                evidence := make([]string, 0, len(binding.BasisRefs))
                ruleSourcePresent := false
                for _, ref := range binding.BasisRefs {
                    if items[ref] == "" {
                        return out, fmt.Errorf("BINDING_SOURCE_UNRESOLVED: %s/%s/%s", policy.ID, rule.ID, kind)
                    }
                    ruleSourcePresent = ruleSourcePresent || ref == rule.SourceRef
                    evidence = append(evidence, ref)
                }
                if !ruleSourcePresent {
                    return out, fmt.Errorf("BINDING_RULE_SOURCE_REQUIRED: %s/%s/%s", policy.ID, rule.ID, kind)
                }
                sort.Strings(evidence)
                d := Decision{
                    PolicyID: policy.ID, RuleID: rule.ID, ContractKind: kind,
                    PolicyPath: policy.Path, SourcePointer: items[rule.SourceRef],
                    EvidenceRefs: evidence, State: "UNRESOLVED", Reason: "NO_CONFIRMED_CONTRACT_CONSUMPTION",
                }
                if binding.ContractRef != nil {
                    if !exactRef(*binding.ContractRef) || binding.ContractRef.SourceRef != input.Contracts[kind].SourceRef {
                        return out, fmt.Errorf("UNREGISTERED_OR_UNPINNED_CONTRACT_SOURCE: %s/%s/%s", policy.ID, rule.ID, kind)
                    }
                    d.ContractRef = binding.ContractRef
                }
                switch binding.State {
                case "NOT_APPLICABLE":
                    if binding.ContractRef != nil {
                        return out, fmt.Errorf("NOT_APPLICABLE_CANNOT_REFERENCE_CONTRACT: %s/%s/%s", policy.ID, rule.ID, kind)
                    }
                    d.State, d.Reason = "NOT_APPLICABLE_CANDIDATE", "EXPLICIT_RULE_LOCAL_NOT_APPLICABLE"
                case "DECLARED":
                    if binding.ContractRef == nil {
                        return out, fmt.Errorf("DECLARED_CONTRACT_REFERENCE_REQUIRED: %s/%s/%s", policy.ID, rule.ID, kind)
                    }
                    d.State, d.Reason = "APPLICABLE_CANDIDATE", "EXPLICIT_RULE_LOCAL_DECLARATION"
                case "UNRESOLVED":
                    if binding.ContractRef != nil {
                        // A migrated definition reference is not proof of consumption.
                        d.Reason = "REFERENCE_EXISTS_BUT_CONSUMPTION_UNRESOLVED"
                    }
                default:
                    return out, fmt.Errorf("INVALID_BINDING_STATE: %s/%s/%s", policy.ID, rule.ID, kind)
                }
                if !effective(statusRoot(policy.DocumentStatus)) {
                    d.State, d.Reason = "INELIGIBLE_POLICY", "POLICY_LIFECYCLE_NOT_EFFECTIVE"
                } else if d.State == "APPLICABLE_CANDIDATE" && input.Contracts[kind].ActivationState != "REGISTERED" {
                    d.Reason = "DECLARED_BUT_CONTRACT_NOT_REGISTERED"
                } else if d.State == "APPLICABLE_CANDIDATE" {
                    // Registration alone is insufficient without effective evaluation
                    // source, matching owner revision, and an authority gate.
                    d.Reason = "DECLARED_REQUIRES_INDEPENDENT_EVALUATION_AUTHORITY"
                }
                out.Counts[d.State]++
                out.BindingCount++
                out.Decisions = append(out.Decisions, d)
            }
        }
    }
    out.PolicyCount = len(input.Policies)
    sort.Slice(out.Decisions, func(i, j int) bool {
        a, b := out.Decisions[i], out.Decisions[j]
        if a.PolicyPath != b.PolicyPath { return a.PolicyPath < b.PolicyPath }
        if a.RuleID != b.RuleID { return a.RuleID < b.RuleID }
        return a.ContractKind < b.ContractKind
    })
    return out, nil
}
