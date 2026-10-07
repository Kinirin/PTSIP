package machine

import (
	"fmt"
)

const supportScopeRecord = "developer/policy/registries/vpms-contract-materialization.json"
const activationScopeRecord = "developer/policy/registries/vpms-runtime-activation.json"
const implementationScopeRecord = "developer/policy/registries/vpms-api-implementation.json"

func init() {
	RegisterOperations("support-contract", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		switch command {
		case "preflight":
			return r.SupportRegistrationPreflight()
		case "verify":
			return r.VerifyContractRegistration(BoolOption(opts, "check-worktree"))
		case "inspect":
			return r.ProductContract(args[0], BoolOption(opts, "require-active"))
		case "validate-selection":
			payload, err := r.Read(opts["--input"])
			if err != nil {
				return nil, err
			}
			return r.ValidateSelectionDocument(opts["--kind"], payload)
		}
		return nil, fmt.Errorf("unregistered support-contract operation")
	})
	RegisterOperations("vpms-api", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		return r.VerifyAPIImplementation(BoolOption(opts, "check-worktree"))
	})
}
