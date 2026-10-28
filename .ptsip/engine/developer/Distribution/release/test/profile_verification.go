package releasetest

import profileverify "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/profile"

func VerifyProjectProfiles(root, expectedVersion string) error {
	return profileverify.VerifyProjectProfiles(root, expectedVersion)
}
