// Package release retains the developer CLI's old call boundary only. All
// selection, verification and publication use the admitted successor modules.
package release

import "fmt"

func CLI(args []string) (any, error) {
	return nil, fmt.Errorf("RELEASE_EXECUTION_BLOCKED: legacy release commands are retired; release publication and retired CLI entry remain unavailable; use approved developer validator in .ptsip/engine/developer")
}
