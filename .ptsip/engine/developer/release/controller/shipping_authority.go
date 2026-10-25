package releasecontroller

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const activePublicationEvidenceOwner = "developer/policy/CTRL/release-admission.policy.json"
const publicationEvidenceApproval = "developer/policy/approvals/MPA-20261013-RELEASE-PRESENTATION-AND-DELETED-CARRIER-REISSUE.yaml"

type publicationEvidenceAuthority struct {
	Schema string `json:"schema_version"`
	Authority string `json:"authority_ref"`
	Approval string `json:"approval_ref"`
	VerificationRun string `json:"verification_run_id"`
	ReleaseID string `json:"release_id"`
	Lookback int `json:"verification_lookback_pages"`
}

// validatePublicationEvidenceAuthority never infers authority from a legacy
// family identifier. Publication evidence is subordinate to the active CTRL
// contract and may only bind the registered original-source resolver.
func validatePublicationEvidenceAuthority(e publicationEvidenceAuthority) error {
	if e.Schema != "ptsip-publication-evidence/v1" ||
		e.Authority != activePublicationEvidenceOwner ||
		e.Approval != publicationEvidenceApproval ||
		e.VerificationRun != "AUTOMATIC_NEWEST_SUCCESSFUL_EXACT_SOURCE_AND_SELECTION" ||
		e.ReleaseID != "AUTOMATIC_UNIQUE_SOURCE_TAG_THEN_RETURNED_NUMERIC_ID" ||
		e.Lookback != 3 {
		return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: publication evidence must bind active CTRL/ASSURE authority")
	}
	return nil
}

// AdmitPublicationEvidenceAuthority validates only the policy-derived
// evidence routing contract. AdmitVerification and AdmitShipping independently
// bind a specific source SHA, selected components, and original artifacts.
func AdmitPublicationEvidenceAuthority(root string) error {
	if err := AdmitCTRLPublicationPolicy(root); err != nil {
		return err
	}
	var e publicationEvidenceAuthority
	if err := ReadJSON(root, EvidenceContractPath, &e); err != nil {
		return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: evidence authority unavailable: %w", err)
	}
	if err := validatePublicationEvidenceAuthority(e); err != nil {
		return err
	}
	approval, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Approval)))
	if err != nil {
		return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: missing approved publication provenance: %w", err)
	}
	for _, marker := range [][]byte{
		[]byte("  decision: APPROVED"),
		[]byte("  decision_source: USER_EXPLICIT"),
		[]byte("  approval_id: MPA-20261013-RELEASE-PRESENTATION-AND-DELETED-CARRIER-REISSUE"),
	} {
		if !bytes.Contains(approval, marker) {
			return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: publication evidence approval incomplete")
		}
	}
	return nil
}
