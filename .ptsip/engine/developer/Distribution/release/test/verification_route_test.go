package releasetest

import (
	"os"
	"path/filepath"
	"testing"
)

func testOnlyRouteFixture(t *testing.T) (string, TestContract) {
	t.Helper()
	root := releaseBoundaryFixture(t)
	path := filepath.Join(root, ".ptsip", "contracts", "release")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	admission := `{
      "schema_version":"ptsip-release-admission-direction-projection/v1",
      "normative_policy_status":"NOT_ACTIVE_NORMATIVE_POLICY",
      "old_root_policy_fallback":"FORBIDDEN",
      "verification_module":".ptsip/engine/developer/Distribution/release/",
      "publication_authorized":false,
      "verification_boundary_ref":".ptsip/contracts/release/verification-boundary.json"
    }`
	if err := os.WriteFile(filepath.Join(path, "admission.json"), []byte(admission), 0644); err != nil {
		t.Fatal(err)
	}
	no := false
	return root, TestContract{
		Schema:       "ptsip-release-test/v1",
		Workflow:     ".github/workflows/tooling-test.yml",
		ModeRegistry: ".github/test_modes.yaml",
		TestRoot:     ".ptsip/engine/developer/Distribution/release/test/",
		Production:   ".ptsip/contracts/release/composition-contract.json",
		Publication:  &no,
		Boundary:     ".ptsip/contracts/release/verification-boundary.json",
		Required:     []string{"windows-x64", "linux-x64"},
		Runners:      map[string]string{"windows-x64": "windows-2025", "linux-x64": "ubuntu-24.04"},
	}
}

func TestReadOnlyVerificationRouteRequiresRegisteredContracts(t *testing.T) {
	root, c := testOnlyRouteFixture(t)
	ref := ".ptsip/contracts/release/verification-contract.json"
	if err := verifyNonNormativeTestRoute(root, ref, c); err != nil {
		t.Fatal(err)
	}
	if err := verifyNonNormativeTestRoute(root, ".github/checks/release/test/verification-contract.json", c); err == nil {
		t.Fatal("retired release test controller contract admitted")
	}
	c.Workflow = ".github/workflows/tooling-release.yml"
	if err := verifyNonNormativeTestRoute(root, ref, c); err == nil {
		t.Fatal("publication workflow admitted as verification")
	}
}

func TestReadOnlyVerificationRouteFailsClosedOnPublicationOrMissingEvidence(t *testing.T) {
	root, c := testOnlyRouteFixture(t)
	ref := ".ptsip/contracts/release/verification-contract.json"
	yes := true
	c.Publication = &yes
	if err := verifyNonNormativeTestRoute(root, ref, c); err == nil {
		t.Fatal("publication operation admitted in test-only mode")
	}
	no := false
	c.Publication = &no
	path := filepath.Join(root, ".ptsip", "contracts", "release", "admission.json")
	for _, mutated := range []string{
		`{"schema_version":"ptsip-release-admission-direction-projection/v1","normative_policy_status":"ACTIVE","old_root_policy_fallback":"FORBIDDEN","verification_module":".ptsip/engine/developer/Distribution/release/","publication_authorized":false}`,
		`{"schema_version":"ptsip-release-admission-direction-projection/v1","normative_policy_status":"NOT_ACTIVE_NORMATIVE_POLICY","old_root_policy_fallback":"FORBIDDEN","verification_module":".ptsip/engine/developer/Distribution/release/","publication_authorized":true}`,
		`{"schema_version":"ptsip-release-admission-direction-projection/v1","normative_policy_status":"NOT_ACTIVE_NORMATIVE_POLICY","old_root_policy_fallback":"ALLOWED","verification_module":".ptsip/engine/developer/Distribution/release/","publication_authorized":false}`,
		`{}`,
	} {
		if err := os.WriteFile(path, []byte(mutated), 0644); err != nil {
			t.Fatal(err)
		}
		if err := verifyNonNormativeTestRoute(root, ref, c); err == nil {
			t.Fatalf("unadmitted release contract passed: %s", mutated)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := verifyNonNormativeTestRoute(root, ref, c); err == nil {
		t.Fatal("missing admission contract accepted")
	}
}

func TestDeclaredVerificationSelectionNeverAssertsPolicyOrPublishing(t *testing.T) {
	root := repoRoot(t)
	v, err := Plan(root, "release-verification", "package", "")
	if err != nil {
		t.Fatal(err)
	}
	if v.PolicyAuthority != "NOT_ASSERTED" || v.PublicationAuthorized ||
		v.VerificationScope != "SOURCE_BOUND_TEST_EVIDENCE_ONLY" {
		t.Fatal("test execution was misrepresented as release policy authority")
	}
	// Every selected component and host requires full verification without waivers.
	platforms := len(v.Selection.Platforms)
	if len(v.Matrix.Include) != len(v.Selection.Components)*platforms ||
		len(v.Join.Include) != platforms || len(v.Lightweight.Include) != 0 {
		t.Fatalf("missing host/component verification matrix: joins=%d lightweight=%d components=%d", len(v.Join.Include), len(v.Lightweight.Include), len(v.Matrix.Include))
	}
	seen := map[string]bool{}
	for _, entry := range v.Join.Include {
		if entry.Component != "package" || entry.Kind != "package" || seen[entry.Platform] {
			t.Fatalf("invalid or duplicate Package host verification: %+v", entry)
		}
		seen[entry.Platform] = true
	}
	for _, platform := range v.Selection.Platforms {
		if !seen[platform] {
			t.Fatalf("missing required Package host verification: %s", platform)
		}
	}
}
