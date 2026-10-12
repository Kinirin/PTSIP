package release

import (
	"fmt"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

// Release execution remains unavailable until a separately admitted successor
// replaces this suspension. Editing a state projection cannot grant permission.
func requireReleaseExecution(repo *machine.Repository) error {
	state, err := repo.Read(".github/checks/release/release-availability.json")
	if err != nil {
		return fmt.Errorf("RELEASE_EXECUTION_BLOCKED: release state unavailable: %w", err)
	}
	if state["state"] != "BLOCKED" || state["verification_contract"] != nil || state["legacy_fallback"] != false {
		return fmt.Errorf("RELEASE_EXECUTION_BLOCKED: no successor release admission is implemented")
	}
	return fmt.Errorf("RELEASE_EXECUTION_BLOCKED: %s", machine.Text(state["reason"]))
}
