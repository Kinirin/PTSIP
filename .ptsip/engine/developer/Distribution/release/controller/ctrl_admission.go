package releasecontroller

import boundary "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"

// The temporary verification boundary has no publication-admission authority.
func AdmitCTRLPublicationPolicy(root string) error { return boundary.RequirePublication(root) }
