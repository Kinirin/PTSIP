package branch

import "github.com/Kinirin/PTSIP/developer/automation/internal/machine"

func init() {
	machine.RegisterOperations("branch-control", func(repo *machine.Repository, command string, options map[string]string, args []string) (any, error) {
		return Control(repo, command, options)
	})
	machine.RegisterOperations("branch-guard", func(repo *machine.Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "validate":
			return ValidateCreation(
				repo,
				options["--candidate"],
				options["--approved-name"],
				options["--authorization-source"],
				options["--request-kind"],
				options["--creation-mechanism"],
			)
		case "classify-existing":
			return ClassifyExisting(repo, options["--branch"])
		case "profile-transition":
			return ProfileTransition(repo, options["--branch"], options["--from-profile"], options["--to-profile"])
		}
		return nil, machine.Fail("UNREGISTERED_BRANCH_COMMAND", command)
	})
}
