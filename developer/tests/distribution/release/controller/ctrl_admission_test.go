package releasecontroller_test

import (
	. "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCTRLPublicationGateDoesNotLoadLegacyAuthority(t *testing.T) {
	source := testrepo.Root(t)
	root := t.TempDir()
	const contract = ".ptsip/contracts/release/Platform/Common/verification-boundary.json"
	if err := AdmitCTRLPublicationPolicy(root); err == nil {
		t.Fatal("missing verification contract admitted publication")
	}
	raw, err := os.ReadFile(filepath.Join(source, contract))
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, contract)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err := AdmitCTRLPublicationPolicy(root); err == nil || !strings.Contains(err.Error(), "SUCCESSOR_AUTHORITY_UNRESOLVED") {
		t.Fatalf("temporary structure became publication approval: %v", err)
	}
	forged := strings.Replace(string(raw), `"publication_authorized": false`, `"publication_authorized": true`, 1)
	if err := os.WriteFile(filename, []byte(forged), 0644); err != nil {
		t.Fatal(err)
	}
	if err := AdmitCTRLPublicationPolicy(root); err == nil {
		t.Fatal("edited completion flag admitted publication")
	}
}
