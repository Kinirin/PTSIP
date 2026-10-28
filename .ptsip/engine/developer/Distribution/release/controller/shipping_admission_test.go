package releasecontroller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicationEvidenceAuthorityBlocksWithdrawnPolicy(t *testing.T) {
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	if err = AdmitPublicationEvidenceAuthority(root); err == nil {
		t.Fatal("withdrawn CTRL policy admitted publication evidence")
	}
}

func TestAdmitShippingHasNoRetiredPolicyRuntimePath(t *testing.T) {
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".ptsip/engine/developer/Distribution/release/controller/recovery.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "AdmitPublicationEvidenceAuthority(root)") {
		t.Fatal("publication stage lacks current CTRL admission")
	}
	if strings.Contains(body, "policy-resolver") || strings.Contains(body, "developer/automation") {
		t.Fatal("retired developer policy control plane remains in shipping path")
	}
}
