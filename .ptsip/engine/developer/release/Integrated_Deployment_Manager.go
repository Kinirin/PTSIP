// Package release checks whether the repository's integrated deployment manager
// document can be read as JSON. It does not certify a release or interpret policy.
package release

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

const managerPath = ".github/release/Integrated_Deployment_Manager.json"

// Report contains a machine-readable readiness result. DOCUMENT_PARSEABLE is
// not a normative release-verification PASS.
type Report struct {
	Status              string `json:"status"`
	Path                string `json:"path"`
	Reason              string `json:"reason,omitempty"`
	Message             string `json:"message,omitempty"`
	ReleaseVerification string `json:"release_verification"`
}

func blocked(reason, message string) (any, int) {
	return Report{Status: "BLOCKED", Path: managerPath, Reason: reason,
		Message: message, ReleaseVerification: "NOT_ASSERTED"}, 2
}

// Run checks document availability and JSON syntax without deriving authority
// or approving deployment. Root is the repository root, not the Go module root.
func Run(root string) (any, int) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(managerPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return blocked("MANAGER_NOT_FOUND", "deployment manager document is missing")
		}
		return blocked("MANAGER_READ_ERROR", err.Error())
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return blocked("MANAGER_EMPTY", "deployment manager document is empty")
	}
	if !json.Valid(trimmed) {
		return blocked("MANAGER_INVALID_JSON", "deployment manager document is not valid JSON")
	}
	return Report{Status: "DOCUMENT_PARSEABLE", Path: managerPath,
		Message:             "JSON syntax only; release and policy conformance not verified",
		ReleaseVerification: "NOT_ASSERTED"}, 0
}
