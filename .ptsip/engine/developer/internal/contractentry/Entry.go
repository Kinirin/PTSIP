// Package contractentry exposes the shared registered developer-policy selector.
// Selection grants neither policy approval nor execution authority.
package contractentry

import authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"

type Selection = authoring.PolicySelection

func Resolve(root, target string) (Selection, error) {
	return authoring.ResolvePolicyTarget(root, target)
}
