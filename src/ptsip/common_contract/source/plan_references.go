package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PlanPolicyReferenceError struct {
	Index     int
	Reference string
	Err       error
}

func (e *PlanPolicyReferenceError) Error() string {
	return fmt.Sprintf("policy_refs[%d] %q: %v", e.Index, e.Reference, e.Err)
}

// CheckPlanPolicyReferences resolves the structurally validated references
// against the registered Support Feature catalog. Checkout mode is explicit.
func CheckPlanPolicyReferences(productRoot string, checkout bool, instance any) *PlanPolicyReferenceError {
	refs := Array(Object(instance)["policy_refs"])
	if len(refs) == 0 {
		return nil
	}
	productRoot, rootErr := filepath.Abs(productRoot)
	// Canonicalize the selected product root before checking paths below it.
	if rootErr == nil {
		productRoot, rootErr = filepath.EvalSymlinks(productRoot)
	}
	if rootErr != nil {
		return &PlanPolicyReferenceError{Reference: String(refs[0]), Err: rootErr}
	}
	policyRoot := filepath.Join(productRoot, "support", "policy")
	if checkout {
		policyRoot = filepath.Join(filepath.Dir(productRoot), "policy")
	}
	index, indexErr := resource(policyRoot, "index.yaml")
	if indexErr == nil && (index["schema_version"] != "ptsip-support-feature-policy-index/v1" ||
		index["policy_class"] != "PTSIP_SUPPORT_FEATURE" || Array(index["policies"]) == nil) {
		indexErr = errors.New("invalid registered Support Feature catalog")
	}
	rulesByOwner := make(map[string]map[string]int)
	for i, raw := range refs {
		reference := String(raw)
		policyID, ruleID, found := strings.Cut(reference, "#")
		reject := func(err error) *PlanPolicyReferenceError {
			return &PlanPolicyReferenceError{Index: i, Reference: reference, Err: err}
		}
		if !found || policyID == "" || ruleID == "" || strings.Contains(ruleID, "#") {
			return reject(errors.New("expected Policy_ID#Rule_ID"))
		}
		if indexErr != nil {
			return reject(indexErr)
		}
		rules, loaded := rulesByOwner[policyID]
		if !loaded {
			doc, err := registeredPlanPolicy(policyRoot, Array(index["policies"]), policyID)
			if err != nil {
				return reject(err)
			}
			rules, err = declaredRuleCounts(doc)
			if err != nil {
				return reject(err)
			}
			rulesByOwner[policyID] = rules
		}
		if count := rules[ruleID]; count != 1 {
			return reject(fmt.Errorf("expected exactly one declaration of rule %q in policy %q, found %d", ruleID, policyID, count))
		}
	}
	return nil
}

// IsPolicyCheckout recognizes the native product's source layout. The installed
// Support catalog takes precedence when both layouts happen to be present.
func IsPolicyCheckout(productRoot string) bool {
	productRoot, err := filepath.Abs(productRoot)
	if err != nil {
		return false
	}
	if _, err := os.Lstat(filepath.Join(productRoot, "support", "policy", "index.yaml")); !os.IsNotExist(err) {
		return false
	}
	return filepath.Base(productRoot) == "ptsip" && filepath.Base(filepath.Dir(productRoot)) == "src"
}

func registeredPlanPolicy(root string, policies []any, policyID string) (Document, error) {
	var relative string
	count := 0
	for _, row := range policies {
		entry := Object(row)
		if entry["id"] == policyID {
			relative = String(entry["path"])
			count++
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("expected exactly one registration of policy %q, found %d", policyID, count)
	}
	doc, err := resource(root, relative)
	if err != nil {
		return nil, err
	}
	if Identity(doc)["id"] != policyID {
		return nil, fmt.Errorf("registered document does not declare policy %q", policyID)
	}
	return doc, nil
}

func declaredRuleCounts(doc Document) (map[string]int, error) {
	counts := make(map[string]int)
	// Product and unified policy documents both declare flat rules. The native
	// policy sources also have contract-local rule arrays identified by RuleUnits.
	rules := Object(doc["Policy_Rules"])
	if raw, exists := rules["rules"]; exists {
		items := Array(raw)
		if items == nil {
			return nil, errors.New("Policy_Rules.rules must be an array")
		}
		for _, rule := range items {
			if id := String(Object(rule)["id"]); id != "" {
				counts[id]++
			}
		}
	}
	units, err := RuleUnits(doc)
	if err != nil {
		return nil, err
	}
	for _, unit := range units {
		if id := String(Object(unit.Value)["id"]); unit.Kind == "RULE" && id != "" {
			counts[id]++
		}
	}
	return counts, nil
}
