// Package release exposes developer-only structural release checks.
package release

import boundary "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"

// RunAdmission checks the current temporary verification contract only.
// It does not load historical approvals or infer release admission.
func RunAdmission(root string) (any, int) { return boundary.Run(root) }
