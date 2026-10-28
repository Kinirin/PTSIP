package branch

import (
	"fmt"
	"net/http"
	"os"
)

func Create(root string, client *Client, name, approved, base string) (Object, error) {
	if _, err := ValidateCreation(root, name, approved, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API"); err != nil {
		return nil, err
	}
	return nil, Fail("BRANCH_MUTATION_ADMISSION_UNRESOLVED", "request structure is valid; successor branch creation authority is unresolved")
}
func Recreate(root string, client *Client, name, approved, base string) (Object, error) {
	if _, err := ValidateCreation(root, name, approved, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API"); err != nil {
		return nil, err
	}
	return nil, Fail("BRANCH_MUTATION_ADMISSION_UNRESOLVED", "branch deletion and recreation require a successor authority contract")
}

func Control(root, command string, options map[string]string) (any, error) {
	contract, err := Load(root)
	if err != nil {
		return nil, err
	}
	registered := false
	for _, name := range contract.Commands["branch-control"] {
		if name == command {
			registered = true
		}
	}
	if !registered {
		return nil, Fail("UNREGISTERED_BRANCH_COMMAND", command)
	}
	if command == "commands" {
		return Object{"status": "REGISTERED_COMMANDS", "commands": contract.Commands, "canonical_entrypoint": ".ptsip/engine/developer/Distribution/branch/cmd/ptsip-branch", "creation_admission": "UNRESOLVED", "execution_authorized": false}, nil
	}
	if command == "create" {
		return Create(root, nil, options["--branch"], options["--approved-name"], options["--base-ref"])
	}
	if command == "recreate" {
		return Recreate(root, nil, options["--branch"], options["--approved-name"], options["--base-ref"])
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	client, err := NewClient(options["--github-repository"], token, "")
	if err != nil {
		return nil, err
	}
	return Query(root, client, command, options["--branch"])
}

func Query(root string, client *Client, command, name string) (any, error) {
	if _, err := Load(root); err != nil {
		return nil, err
	}
	switch command {
	case "inspect":
		if err := CheckName(root, name); err != nil {
			return nil, err
		}
		value, err := client.Request(http.MethodGet, "/repos/"+client.Repository+"/branches/"+branchRef(name), nil)
		if err != nil {
			return nil, err
		}
		item := object(value)
		sha := text(object(item["commit"])["sha"])
		if text(item["name"]) != name || sha == "" {
			return nil, Fail("INVALID_GITHUB_RESPONSE", "branch identity missing or differs")
		}
		return Object{"status": "OK", "repository": client.Repository, "branch": name, "sha": sha, "protected": item["protected"], "execution_authorized": false}, nil
	case "list":
		branches := []any{}
		for page := 1; page <= 1000; page++ {
			value, err := client.Request(http.MethodGet, fmt.Sprintf("/repos/%s/branches?per_page=100&page=%d", client.Repository, page), nil)
			if err != nil {
				return nil, err
			}
			items, ok := value.([]any)
			if !ok {
				return nil, Fail("INVALID_GITHUB_RESPONSE", "branch list must be array")
			}
			for _, value := range items {
				item := object(value)
				if text(item["name"]) == "" || text(object(item["commit"])["sha"]) == "" {
					return nil, Fail("INVALID_GITHUB_RESPONSE", "missing branch name or SHA")
				}
				branches = append(branches, item)
			}
			if len(items) < 100 {
				return Object{"status": "OK", "repository": client.Repository, "branches": branches, "execution_authorized": false}, nil
			}
		}
		return nil, Fail("GITHUB_PAGINATION_LIMIT", "branch list exceeded page limit")
	}
	return nil, Fail("UNREGISTERED_BRANCH_COMMAND", command)
}
