// Package release retains the developer CLI's old call boundary only. All
// selection, verification and publication use the admitted successor modules.
package release

import "fmt"

func CLI(args []string) (any, error) {
	return nil, fmt.Errorf("RELEASE_EXECUTION_BLOCKED: legacy release commands are retired; use .github/checks/release/run_release.ps1 and the registered tooling-test/tooling-release workflows")
}
