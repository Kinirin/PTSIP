package ctrl_admission

import boundary "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"

func Verify(root string) error { return boundary.RequirePublication(root) }
func Run(root string) (any, int) {
	report := map[string]any{"status": "UNVERIFIED", "scope": "PUBLICATION_ADMISSION", "publication_authorized": false, "per_release_evidence": "NOT_ASSERTED", "contract_ref": boundary.Path}
	if _, err := boundary.Load(root); err != nil {
		report["status"] = "FAIL"
		report["problem"] = err.Error()
		return report, 2
	}
	report["problem"] = "SUCCESSOR_AUTHORITY_UNRESOLVED"
	return report, 4
}
