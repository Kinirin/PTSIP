package schema_check

// The owner-approved verification selector must never promote retired paths
// or a frozen migration graph into the current policy verification gate.
import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

type verificationExclusionContract struct {
	SchemaVersion     string `json:"schema_version"`
	Domain            string `json:"domain"`
	AuthoritySource   string `json:"authority_source"`
	PolicyOwner       string `json:"policy_owner"`
	ExcludedPathRules []struct {
		Match string `json:"match"`
		Path  string `json:"path"`
	} `json:"excluded_path_rules"`
	AppliesTo                                                     []string `json:"applies_to"`
	HistoricalMigrationReconstructionRequiredForCurrentPolicyPass bool     `json:"historical_migration_reconstruction_required_for_current_policy_pass"`
	OnReferencedRetiredSource                                     string   `json:"on_referenced_retired_source"`
	OnUnsupportedSuccessorValidation                              string   `json:"on_unsupported_successor_validation"`
}

func TestPolicyVerificationExclusions(t *testing.T) {
	root := routingRepositoryRoot(t)
	body, err := os.ReadFile(filepath.Join(root, ".ptsip", "contracts", "policy", "routing", "verification_exclusions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract verificationExclusionContract
	if err := json.Unmarshal(body, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.SchemaVersion != "ptsip-developer-policy-verification-scope/v1" ||
		contract.Domain != "PTSIP_DEVELOPER_POLICY" ||
		contract.AuthoritySource != "USER_EXPLICIT" ||
		contract.PolicyOwner != "MPD-GOV-0006" ||
		contract.HistoricalMigrationReconstructionRequiredForCurrentPolicyPass ||
		contract.OnReferencedRetiredSource != "EXCLUDE_FROM_CURRENT_POLICY_VERIFICATION_NO_FALLBACK" ||
		contract.OnUnsupportedSuccessorValidation != "REPORT_UNVERIFIED_NOT_PASS" {
		t.Fatal("retirement scope does not preserve approved verification authority")
	}
	expected := map[string]string{
		"developer/policy/registries/":                    "PREFIX",
		"developer/policy/schemas/":                       "PREFIX",
		"developer/policy/policy-resolver-bindings/":      "PREFIX",
		"developer/policy/policy-resolver-contract.yaml":  "EXACT",
		"developer/policy/MPD-BOUND-0001.yaml":            "EXACT",
		"developer/policy/source-application-review.yaml": "EXACT",
		"developer/policy/approvals/":                     "PREFIX",
		"developer/policy/analysis/":                      "PREFIX",
		"developer/policy/legacy/":                        "PREFIX",
		"developer/bindings/":                             "PREFIX",
		"developer/automation/":                           "PREFIX",
		"developer/verification/":                         "PREFIX",
		".github/release/":                                "PREFIX",
		".github/scripts/":                                "PREFIX",
		".github/checks/":                                 "PREFIX",
		"developer/state/":                                "PREFIX",
		"reference/":                                      "PREFIX",
		"registry/":                                       "PREFIX",
		"schemas/":                                        "PREFIX",
		"planning/":                                       "PREFIX",
	}
	if len(contract.ExcludedPathRules) != len(expected) {
		t.Fatalf("retired verification route count changed: %d", len(contract.ExcludedPathRules))
	}
	seen := map[string]bool{}
	for _, rule := range contract.ExcludedPathRules {
		if seen[rule.Path] || expected[rule.Path] != rule.Match ||
			strings.HasPrefix(rule.Path, "/") || strings.Contains(rule.Path, "\\") ||
			strings.Contains(rule.Path, "..") || path.Clean(rule.Path) != strings.TrimSuffix(rule.Path, "/") {
			t.Fatalf("unapproved or duplicate retired path rule: %s (%s)", rule.Path, rule.Match)
		}
		seen[rule.Path] = true
	}
	for p := range expected {
		if !seen[p] {
			t.Fatalf("missing retired verification exclusion: %s", p)
		}
	}

	requiredPurposes := map[string]bool{
		"CURRENT_POLICY_AUTHORITY_DISCOVERY":                true,
		"CURRENT_POLICY_DOCUMENT_OR_SCHEMA_VALIDATION":      true,
		"MANDATORY_POLICY_VERIFICATION_AND_ADMISSION_GATES": true,
	}
	if len(contract.AppliesTo) != len(requiredPurposes) {
		t.Fatal("wrong exclusion purposes")
	}
	for _, purpose := range contract.AppliesTo {
		if !requiredPurposes[purpose] {
			t.Fatalf("invalid exclusion purpose: %s", purpose)
		}
		delete(requiredPurposes, purpose)
	}
	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "policy-cleanup-verification.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(workflow), "TestHistoricalSourceEvidencePreservesArchiveAndRouting") ||
		strings.Contains(string(workflow), "developer/policy/legacy/**") ||
		strings.Contains(string(workflow), "developer/policy/*/*.yaml") ||
		strings.Contains(string(workflow), "go -C developer/automation") ||
		strings.Contains(string(workflow), "go -C developer/tests") {
		t.Fatal("retired migration source must not be a required policy-cleanup gate or trigger")
	}
	modes, err := os.ReadFile(filepath.Join(root, ".github", "test_modes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(modes), "- developer/tests/rootfamily") {
		t.Fatal("historical root-family audit reselected as current repository verification")
	}
}
