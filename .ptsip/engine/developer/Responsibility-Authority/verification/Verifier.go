package responsibilityauthority

import authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"

// Run is the developer authoring and reference-validation interface. Conditions
// are declarations; this interface does not evaluate or activate a permission.
func Run(root, operation, input, output string, validate authoring.Validator) (any, int) {
	return authoring.Run(root, "responsibility-authority", operation, "", input, output, validate)
}
