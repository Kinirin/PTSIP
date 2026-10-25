package releasecontroller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicationEvidenceAuthorityActiveCTRL(t *testing.T) {
	root,err:=filepath.Abs("../../../../..")
	if err!=nil{t.Fatal(err)}
	if err:=AdmitPublicationEvidenceAuthority(root);err!=nil{
		t.Fatalf("current CTRL evidence authority unexpectedly blocked: %v",err)
	}
}

func TestPublicationEvidenceAuthorityRejectsRetiredResolver(t *testing.T) {
	root,err:=filepath.Abs("../../../../..")
	if err!=nil{t.Fatal(err)}
	var valid publicationEvidenceAuthority
	if err:=ReadJSON(root,EvidenceContractPath,&valid);err!=nil{t.Fatal(err)}
	if err:=validatePublicationEvidenceAuthority(valid);err!=nil{t.Fatal(err)}
	changed:=valid
	changed.Authority="developer/policy/CTRL/MPD-CTRL-0001.yaml#rules.release_orchestration"
	if err:=validatePublicationEvidenceAuthority(changed);err==nil{t.Fatal("retired authority permitted publication")}
	changed=valid
	changed.VerificationRun="PREVIOUS_SUCCESS_UNBOUND"
	if err:=validatePublicationEvidenceAuthority(changed);err==nil{t.Fatal("unbound evidence was accepted")}
	changed=valid
	changed.Lookback=0
	if err:=validatePublicationEvidenceAuthority(changed);err==nil{t.Fatal("unbounded or missing evidence lookup accepted")}
}

func TestAdmitShippingHasNoRetiredPolicyRuntimePath(t *testing.T) {
	root,err:=filepath.Abs("../../../../..")
	if err!=nil{t.Fatal(err)}
	data,err:=os.ReadFile(filepath.Join(root,".ptsip/engine/developer/release/controller/recovery.go"))
	if err!=nil{t.Fatal(err)}
	body:=string(data)
	if !strings.Contains(body,"AdmitPublicationEvidenceAuthority(root)") {
		t.Fatal("publication stage lacks current CTRL admission")
	}
	if strings.Contains(body,"policy-resolver") || strings.Contains(body,"developer/automation") {
		t.Fatal("retired developer policy control plane remains in shipping path")
	}
}
