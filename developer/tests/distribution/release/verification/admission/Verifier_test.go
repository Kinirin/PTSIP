package ctrl_admission_test

import (
	. "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/admission"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationAdmissionIsUnverifiedWithCurrentVerificationContract(t *testing.T) {
	root := testrepo.Root(t)
	report, code := Run(root)
	if code != 4 || report.(map[string]any)["publication_authorized"] != false {
		t.Fatalf("verification admitted publication: %v code=%d", report, code)
	}
	if err := Verify(root); err == nil {
		t.Fatal("temporary contract granted publication")
	}
	fixture := t.TempDir()
	const path = ".ptsip/contracts/release/Platform/Common/verification-boundary.json"
	raw, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(fixture, path)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, path), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if report, code := Run(fixture); code != 4 || report.(map[string]any)["status"] != "UNVERIFIED" {
		t.Fatalf("missing legacy files changed admission: %v %d", report, code)
	}
}
