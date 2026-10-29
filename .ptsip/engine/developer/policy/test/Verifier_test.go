package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

func repository(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
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
		".ptsip/contracts/schemas/Policy/Policy_Identity/PTSIP_Policy_Identity.schema.json",
		".ptsip/contracts/policy/identity/policy_kind.json",
		".ptsip/contracts/policy/identity/root_family.json",
	)
	validate := native(t)
	document := authoring.Document{"id": "EXAMPLE", "policy_kind": "PTSIP_DEVELOPER_POLICY", "root_family": "GOV"}
	if err := verification.CheckDocument(root, "policy-identity-check", document, validate); err != nil {
		t.Fatal(err)
	}
	document["root_family"] = "UNKNOWN"
	if err := verification.CheckDocument(root, "policy-identity-check", document, validate); err == nil {
		t.Fatal("unregistered family accepted")
	}
	if err := os.Remove(filepath.Join(root, ".ptsip/contracts/policy/identity/root_family.json")); err != nil {
		t.Fatal(err)
	}
	if err := verification.CheckDocument(root, "policy-identity-check", document, validate); err == nil {
		t.Fatal("missing current source fell back to another authority")
	}
}

func TestLifecycleChecksOnlyDeclaredStatusVocabulary(t *testing.T) {
	root := isolated(t,
		".ptsip/contracts/schemas/Policy/Policy_Status/Public_Policy_Status.schema.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json",
		".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_DRAFT.json",
		".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_ACTIVATE.json",
		".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_INTERRUPTION.json",
	)
	validate := native(t)
	for _, status := range []string{"DRAFT.Default", "APPROVED.Applying", "RETIRED.Waiting"} {
		document := authoring.Document{"Policy_Status_Schema_ver": "2", "status": status, "contract": "urn:ptsip:contract:policy:gov:lifecycle:governance"}
		if err := verification.CheckDocument(root, "policy-status-check", document, validate); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
	}
	for _, status := range []any{"APPROVED.Unknown", map[string]any{"status": "APPROVED", "substatus": "Applying"}} {
		document := authoring.Document{"Policy_Status_Schema_ver": "2", "status": status, "contract": "urn:ptsip:contract:policy:gov:lifecycle:governance"}
		if err := verification.CheckDocument(root, "policy-status-check", document, validate); err == nil {
			t.Fatalf("invalid status accepted: %v", status)
		}
	}
	for _, status := range []string{"DRAFT", "ACTIVATE", "INTERRUPTION"} {
		if err := verification.CheckDocument(root, "plan-status-check", authoring.Document{"status": status}, validate); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
	}
	if err := verification.CheckDocument(root, "plan-status-check", authoring.Document{"status": "APPROVED"}, validate); err == nil {
		t.Fatal("policy status interpreted as plan status")
	}
}

func TestCurrentPolicyChecksSharedStructureAndConditionalRules(t *testing.T) {
	root, validate := repository(t), native(t)
	document, err := authoring.ReadDocument(root, "developer/policy/GOV/MPD-GOV-0006.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := verification.CheckDocument(root, "policy-check", document, validate); err != nil {
		t.Fatal(err)
	}
	delete(document, "Policy_Rules")
	if err := verification.CheckDocument(root, "policy-check", document, validate); err == nil {
		t.Fatal("missing Rules accepted for selected GOV family")
	}
}
