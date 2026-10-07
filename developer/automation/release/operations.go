package release

import "github.com/Kinirin/PTSIP/developer/automation/internal/machine"

func init() {
	machine.RegisterOperations("release", func(repo *machine.Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "prepare":
			return Prepare(repo, options["--dispatched-sha"], options["--dispatched-ref"])
		case "gate":
			return RequireExactCIGate(options["--github-repository"], options["--github-api-url"], options["--source-sha"])
		case "reconfirm":
			return ReconfirmMain(repo, options["--source-sha"])
		case "draft":
			return CreateDraft(
				options["--github-repository"],
				options["--github-api-url"],
				options["--source-sha"],
				options["--version"],
				options["--tag"],
				options["--note"],
				repo,
			)
		case "tag-verify":
			return VerifyTag(repo, options["--release-tag"])
		case "build":
			return Build(repo)
		case "metadata-check":
			return VerifyDistributionMetadata(repo)
		default:
			return nil, machine.Fail("UNREGISTERED_RELEASE_COMMAND", command)
		}
	})
}
