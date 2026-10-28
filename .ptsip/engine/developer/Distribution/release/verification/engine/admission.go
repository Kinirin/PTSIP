package enginewheel

import boundary "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"

const VerificationContract = boundary.Path

// Admit permits local verification artifacts only. Every source/run/digest and
// platform check remains in its original verifier; publication has a separate gate.
func Admit(root string) error { return boundary.CheckOperation(root, "ENGINE_LOCAL_VERIFICATION") }
