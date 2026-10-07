package machine

import "github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"

func (r *Repository) ValidateCurrentRootContracts(class string) (Object, error) {
	return lifecycle.ValidateCurrentRootContracts(r, class)
}
