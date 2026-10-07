package branch

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func Create(repo *machine.Repository, client *Client, branch, approved, base string) (machine.Object, error) {
	if _, err := ValidateCreation(repo, branch, approved, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API"); err != nil {
		return nil, err
	}
	sha, err := client.ResolveSHA(base)
	if err != nil {
		return nil, err
	}
	created, err := client.CreateAtSHA(branch, sha)
	if err != nil {
		return nil, err
	}
	return machine.Object{
		"status": "CREATED",
		"repository": client.Repository,
		"branch": branch,
		"base_ref": base,
		"base_sha": sha,
		"creation_mechanism": "GITHUB_CREATE_BRANCH_API",
		"ref": created["ref"],
	}, nil
}

func Recreate(repo *machine.Repository, client *Client, branch, approved, base string) (machine.Object, error) {
	if _, err := ValidateCreation(repo, branch, approved, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API"); err != nil {
		return nil, err
	}
	if branch == base {
		return nil, machine.Fail("RECREATE_BASE_EQUALS_BRANCH", "recreate base must differ from the branch")
	}
	old, err := client.ResolveSHA(branch)
	if err != nil {
		return nil, err
	}
	comparison, err := client.Request("GET", "/repos/"+client.Repository+"/compare/"+url.PathEscape(branch)+"..."+url.PathEscape(base), nil)
	if err != nil {
		return nil, err
	}
	if machine.Map(comparison) == nil || machine.Map(comparison)["behind_by"] != float64(0) {
		return nil, machine.Fail("BRANCH_HAS_UNMERGED_COMMITS", "branch is not fully contained in the base")
	}
	if client.Token == "" {
		return nil, machine.Fail("GITHUB_TOKEN_REQUIRED", "branch deletion requires an authenticated client")
	}
	if _, err := client.Request("DELETE", "/repos/"+client.Repository+"/git/refs/heads/"+branchRef(branch), nil); err != nil {
		return nil, err
	}
	recreate := func() (string, error) {
		selected, err := client.ResolveSHA(base)
		if err != nil {
			return "", err
		}
		if _, err := client.CreateAtSHA(branch, selected); err != nil {
			return "", err
		}
		actual, err := client.ResolveSHA(branch)
		if err != nil {
			return "", err
		}
		finalBase, err := client.ResolveSHA(base)
		if err != nil {
			return "", err
		}
		if selected != actual || finalBase != selected {
			return "", machine.Fail("RECREATE_BASE_MOVED_OR_REF_MISMATCH", "base moved or recreated ref differs")
		}
		return actual, nil
	}
	newSHA, err := recreate()
	if err != nil {
		if _, restoreErr := client.CreateAtSHA(branch, old); restoreErr != nil {
			return nil, machine.Fail("RECREATE_FAILED_ROLLBACK_FAILED", fmt.Sprintf("recreate failed: %v; rollback failed: %v", err, restoreErr))
		}
		return nil, machine.Fail("RECREATE_FAILED_ROLLED_BACK", fmt.Sprintf("%v; original ref restored", err))
	}
	return machine.Object{
		"status": "RECREATED",
		"repository": client.Repository,
		"branch": branch,
		"base_ref": base,
		"old_sha": old,
		"new_sha": newSHA,
		"creation_mechanism": "GITHUB_CREATE_BRANCH_API",
		"safety": "OLD_BRANCH_FULLY_CONTAINED_IN_BASE",
	}, nil
}

func Control(repo *machine.Repository, command string, options map[string]string) (any, error) {
	mechanism, err := rootSection(repo, "MPD-CNTR-0001", "unit_mpd_0014_2636c6da29a8")
	if err != nil {
		return nil, err
	}
	if !machine.Has(machine.Strings(machine.Map(mechanism)["registered_commands"]), strings.ToUpper(command)) {
		return nil, machine.Fail("UNREGISTERED_BRANCH_COMMAND", command)
	}
	if command == "commands" {
		return machine.Object{
			"status": "REGISTERED_COMMANDS",
			"commands": machine.Map(mechanism)["registered_commands"],
			"unregistered_operation": "FAIL_CLOSED",
			"canonical_entrypoint": "developer/automation/cmd/ptsip-dev",
		}, nil
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	client, err := NewClient(options["--github-repository"], token, "")
	if err != nil {
		return nil, err
	}
	branch := options["--branch"]
	switch command {
	case "create":
		return Create(repo, client, branch, options["--approved-name"], options["--base-ref"])
	case "recreate":
		return Recreate(repo, client, branch, options["--approved-name"], options["--base-ref"])
	case "inspect":
		value, err := client.Request("GET", "/repos/"+client.Repository+"/branches/"+branchRef(branch), nil)
		if err != nil {
			return nil, err
		}
		item := machine.Map(value)
		if item == nil {
			return nil, machine.Fail("INVALID_GITHUB_RESPONSE", "branch response is not an object")
		}
		return machine.Object{
			"status": "OK",
			"repository": client.Repository,
			"branch": item["name"],
			"sha": machine.Map(item["commit"])["sha"],
			"protected": item["protected"],
		}, nil
	case "list":
		branches := []any{}
		for page := 1; ; page++ {
			value, err := client.Request("GET", fmt.Sprintf("/repos/%s/branches?per_page=100&page=%d", client.Repository, page), nil)
			if err != nil {
				return nil, err
			}
			if machine.List(value) == nil {
				return nil, machine.Fail("INVALID_GITHUB_RESPONSE", "branch list response is not a list")
			}
			for _, raw := range machine.List(value) {
				if item := machine.Map(raw); item != nil {
					branches = append(branches, machine.Object{
						"name": item["name"],
						"sha": machine.Map(item["commit"])["sha"],
					})
				}
			}
			if len(machine.List(value)) < 100 {
				return machine.Object{
					"status": "OK",
					"repository": client.Repository,
					"branches": branches,
				}, nil
			}
		}
	}
	return nil, machine.Fail("UNREGISTERED_BRANCH_COMMAND", command)
}
