package ctrl_admission

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationAdmissionIsUnverifiedWithCurrentVerificationContract(t *testing.T) {
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
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
