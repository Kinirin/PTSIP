package check_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/authoring"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func repository(t *testing.T) string {
	t.Helper()
	return testrepo.Root(t)
}

func native(t *testing.T) authoring.Validator {
	t.Helper()
	validate, err := verification.InstalledValidator()
	if err != nil {
		t.Skipf("installed PTSIP validation unavailable: %v", err)
	}
	return validate
}

func isolated(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Join(repository(t), filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		filename := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestIdentityHasNoLifecycleRulesOrLegacyDependency(t *testing.T) {
	root := isolated(t,
		".ptsip/contracts/policy/Policy_Identity/PTSIP_Policy_Identity.schema.json",
		".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json",
	)
	validate := native(t)
	document := authoring.Document{"id": "EXAMPLE", "policy_kind": "PTSIP_DEVELOPER_POLICY", "root_family": "GOV"}
	if err := check.CheckDocument(root, "policy-identity-check", document, validate); err != nil {
		t.Fatal(err)
	}
	document["root_family"] = "UNKNOWN"
	if err := check.CheckDocument(root, "policy-identity-check", document, validate); err == nil {
		t.Fatal("unregistered family accepted")
	}
	if err := os.Remove(filepath.Join(root, ".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json")); err != nil {
		t.Fatal(err)
	}
	if err := check.CheckDocument(root, "policy-identity-check", document, validate); err == nil {
		t.Fatal("missing current source fell back to another authority")
	}
}

func TestLifecycleChecksOnlyDeclaredStatusVocabulary(t *testing.T) {
	root := isolated(t,
		".ptsip/contracts/policy/Policy_lifecycle/Public_Policy_Status.schema.json",
		".ptsip/contracts/policy/Policy_lifecycle/status/Policy_status.json",
		"src/ptsip/schemas/plan/plan-lifecycle.schema.json",
	)
	validate := native(t)
	for _, status := range []string{"DRAFT.Default", "APPROVED.Applying", "RETIRED.Waiting"} {
		document := authoring.Document{"Policy_Status_Schema_ver": "2", "status": status, "contract": "urn:ptsip:contract:policy:gov:lifecycle:governance"}
		if err := check.CheckDocument(root, "policy-status-check", document, validate); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
	}
	for _, status := range []any{"APPROVED.Unknown", map[string]any{"status": "APPROVED", "substatus": "Applying"}} {
		document := authoring.Document{"Policy_Status_Schema_ver": "2", "status": status, "contract": "urn:ptsip:contract:policy:gov:lifecycle:governance"}
		if err := check.CheckDocument(root, "policy-status-check", document, validate); err == nil {
			t.Fatalf("invalid status accepted: %v", status)
		}
	}
	plan := testrepo.Open(root)
	const planLifecycle = "src/ptsip/schemas/plan/plan-lifecycle.schema.json"
	for _, status := range []string{"Draft.Proposed", "Approval.In_progress", "Halt.Paused"} {
		if err := plan.Validate(planLifecycle, authoring.Document{"status": status}); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
	}
	for _, status := range []string{"DRAFT", "ACTIVATE", "INTERRUPTION", "APPROVED"} {
		if err := plan.Validate(planLifecycle, authoring.Document{"status": status}); err == nil {
			t.Fatalf("obsolete or policy status interpreted as current Plan status: %s", status)
		}
		if err := check.CheckDocument(root, "plan-status-check", authoring.Document{"status": status}, validate); err == nil || !strings.Contains(err.Error(), "Plan_status_DRAFT.json") {
			t.Fatalf("retired Plan status route selected a fallback: %v", err)
		}
	}
}

func TestV2PolicyChecksSharedStructureAndRejectsMissingRules(t *testing.T) {
	root, validate := repository(t), native(t)
	document, err := authoring.ReadDocument(root, "developer/policy/ARCH/MPD-ARCH-0001.yaml")
	if err != nil {
		t.Fatal(err)
	}
	document, err = check.RestoreUnifiedSource(root, document)
	if err != nil {
		t.Fatal(err)
	}
	if err := check.CheckDocument(root, "policy-check", document, validate); err != nil {
		t.Fatal(err)
	}
	delete(document, "Policy_Rules")
	if err := check.CheckDocument(root, "policy-check", document, validate); err == nil {
		t.Fatal("missing v2 Rules accepted for selected ARCH family")
	}
}
