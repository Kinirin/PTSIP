package releasetest

import (
	"fmt"
	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
)

// Kept as a rejecting CLI compatibility entry. Every Package requires the full
// source-bound Package Join; historical one-time verification waivers are retired.
func VerifyLightweightPackageInstall(root string, s release.Selection, input, platform string) (release.PackageReceipt, error) {
	return release.PackageReceipt{}, fmt.Errorf("LIGHTWEIGHT_VERIFICATION_RETIRED: full source-bound Package Join required")
}
