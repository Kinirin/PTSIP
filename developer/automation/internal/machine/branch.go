package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func (r *Repository) RootSection(id, section string) (any, error) {
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	result, err := resolver.Get(id, section)
	if err != nil {
		return nil, err
	}
	return result["record"], nil
}

func (r *Repository) ValidateBranchCreation(candidate, approved, authorization, request, mechanism string) (Object, error) {
	allowed, err := r.RootSection("MPD-CNTR-0001", "unit_mpd_0014_2636c6da29a8")
	if err != nil {
		return nil, err
	}
	source, err := r.RootSection("MPD-GOV-0001", "unit_mpd_0014_f261c3ba8099")
	if err != nil {
		return nil, err
	}
	classes, err := r.RootSection("MPD-GOV-0001", "unit_mpd_0014_d04f6a56972a")
	if err != nil {
		return nil, err
	}
	development := Map(Map(classes)["DEVELOPMENT_VERSION"])
	if mechanism != Text(Map(allowed)["allowed"]) {
		return nil, Fail("UNAUTHORIZED_BRANCH_CREATION_MECHANISM", "branch creation must use the admitted GitHub create-ref API")
	}
	if authorization != Text(source) {
		return nil, Fail("BRANCH_CREATION_REQUIRES_USER_EXPLICIT", "branch creation requires the exact explicit authorization source")
	}
	if request != Text(development["request_kind"]) {
		return nil, Fail("UNREGISTERED_BRANCH_REQUEST_KIND", request)
	}
	if approved == "" {
		return nil, Fail("APPROVED_BRANCH_NAME_REQUIRED", "exact approved name is required")
	}
	if candidate != approved {
		return nil, Fail("BRANCH_NAME_NOT_EXACTLY_APPROVED", candidate+" differs from the exact approved name")
	}
	pattern, err := regexp.Compile(Text(development["exact_name_pattern"]))
	if err != nil {
		return nil, err
	}
	if !pattern.MatchString(candidate) {
		return nil, Fail("UNAUTHORIZED_BRANCH_NAME", "candidate does not match the registered development branch pattern")
	}
	return Object{"status": "AUTHORIZED", "branch_name": candidate, "branch_class": "DEVELOPMENT_VERSION", "authorization_source": authorization, "request_kind": request, "creation_mechanism": mechanism}, nil
}

func (r *Repository) BranchProfileTransition(branch, previous, next string) (Object, error) {
	classes, err := r.RootSection("MPD-GOV-0001", "unit_mpd_0014_d04f6a56972a")
	if err != nil {
		return nil, err
	}
	pattern, err := regexp.Compile(Text(Map(Map(classes)["DEVELOPMENT_VERSION"])["exact_name_pattern"]))
	if err != nil {
		return nil, err
	}
	if !pattern.MatchString(branch) {
		return nil, Fail("INVALID_DEVELOPMENT_BRANCH", branch)
	}
	independence, err := r.RootSection("MPD-CHANGE-0001", "unit_mpd_0014_103ee75460a0")
	if err != nil {
		return nil, err
	}
	profilePattern, err := regexp.Compile(Text(Map(independence)["project_profile_identity_pattern"]))
	if err != nil {
		return nil, err
	}
	if !profilePattern.MatchString(previous) || !profilePattern.MatchString(next) {
		return nil, Fail("INVALID_PROJECT_PROFILE_IDENTITY", "invalid Project Profile identity")
	}
	return Object{"status": "NO_BRANCH_IDENTITY_CHANGE", "branch_name": branch, "previous_profile": previous, "next_profile": next, "branch_change_required": false, "branch_creation_authorized": false}, nil
}

func (r *Repository) ClassifyBranch(branch string) (Object, error) {
	classes, err := r.RootSection("MPD-GOV-0001", "unit_mpd_0014_d04f6a56972a")
	if err != nil {
		return nil, err
	}
	pattern, err := regexp.Compile(Text(Map(Map(classes)["DEVELOPMENT_VERSION"])["exact_name_pattern"]))
	if err != nil {
		return nil, err
	}
	state := "UNREGISTERED_SHAPE"
	if pattern.MatchString(branch) {
		state = "AUTHORIZED_DEVELOPMENT_VERSION"
	} else {
		retention, err := r.RootSection("MPD-CHANGE-0001", "unit_mpd_0014_5278c1ada455")
		if err != nil {
			return nil, err
		}
		for _, raw := range List(Map(retention)["grandfathered_patterns"]) {
			grandfathered, err := regexp.Compile(Text(raw))
			if err != nil {
				return nil, err
			}
			if grandfathered.MatchString(branch) {
				state = "GRANDFATHERED_RETENTION"
			}
		}
	}
	return Object{"branch_name": branch, "classification": state}, nil
}

type BranchClient struct {
	Repository, Token, APIRoot string
	HTTP                       *http.Client
}

func NewBranchClient(repository, token, apiRoot string) (*BranchClient, error) {
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(repository) {
		return nil, Fail("INVALID_REPOSITORY", "repository must be in owner/name form")
	}
	if apiRoot == "" {
		apiRoot = "https://api.github.com"
	}
	return &BranchClient{Repository: repository, Token: token, APIRoot: strings.TrimRight(apiRoot, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}, nil
}
func (c *BranchClient) Request(method, path string, payload any) (any, error) {
	var data io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		data = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, c.APIRoot+path, data)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "PTSIP-branch-control")
	if c.Token != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return nil, Fail("GITHUB_API_UNREACHABLE", err.Error())
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 32*1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, Fail("GITHUB_API_ERROR", fmt.Sprintf("HTTP %d: %s", response.StatusCode, body))
	}
	if len(body) == 0 {
		return Object{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	return value, nil
}
func branchRef(ref string) string { return url.PathEscape(strings.TrimPrefix(ref, "refs/heads/")) }
func (c *BranchClient) ResolveSHA(ref string) (string, error) {
	value, err := c.Request("GET", "/repos/"+c.Repository+"/git/ref/heads/"+branchRef(ref), nil)
	if err != nil {
		return "", err
	}
	sha := Text(Map(Map(value)["object"])["sha"])
	if sha == "" {
		return "", Fail("INVALID_GITHUB_RESPONSE", "ref response has no object.sha")
	}
	return sha, nil
}
func (c *BranchClient) CreateAtSHA(branch, sha string) (Object, error) {
	if c.Token == "" {
		return nil, Fail("GITHUB_TOKEN_REQUIRED", "branch creation requires GITHUB_TOKEN or GH_TOKEN")
	}
	value, err := c.Request("POST", "/repos/"+c.Repository+"/git/refs", Object{"ref": "refs/heads/" + branch, "sha": sha})
	if err != nil {
		return nil, err
	}
	if Map(value) == nil {
		return nil, Fail("INVALID_GITHUB_RESPONSE", "create-ref response is not an object")
	}
	return Map(value), nil
}
func (r *Repository) CreateBranch(c *BranchClient, branch, approved, base string) (Object, error) {
	if _, err := r.ValidateBranchCreation(branch, approved, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API"); err != nil {
		return nil, err
	}
	sha, err := c.ResolveSHA(base)
	if err != nil {
		return nil, err
	}
	created, err := c.CreateAtSHA(branch, sha)
	if err != nil {
		return nil, err
	}
	return Object{"status": "CREATED", "repository": c.Repository, "branch": branch, "base_ref": base, "base_sha": sha, "creation_mechanism": "GITHUB_CREATE_BRANCH_API", "ref": created["ref"]}, nil
}
func (r *Repository) RecreateBranch(c *BranchClient, branch, approved, base string) (Object, error) {
	if _, err := r.ValidateBranchCreation(branch, approved, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API"); err != nil {
		return nil, err
	}
	if branch == base {
		return nil, Fail("RECREATE_BASE_EQUALS_BRANCH", "recreate base must differ from the branch")
	}
	old, err := c.ResolveSHA(branch)
	if err != nil {
		return nil, err
	}
	comparison, err := c.Request("GET", "/repos/"+c.Repository+"/compare/"+url.PathEscape(branch)+"..."+url.PathEscape(base), nil)
	if err != nil {
		return nil, err
	}
	if Map(comparison) == nil || Map(comparison)["behind_by"] != float64(0) {
		return nil, Fail("BRANCH_HAS_UNMERGED_COMMITS", "branch is not fully contained in the base")
	}
	if c.Token == "" {
		return nil, Fail("GITHUB_TOKEN_REQUIRED", "branch deletion requires an authenticated client")
	}
	if _, err := c.Request("DELETE", "/repos/"+c.Repository+"/git/refs/heads/"+branchRef(branch), nil); err != nil {
		return nil, err
	}
	recreate := func() (string, error) {
		selected, err := c.ResolveSHA(base)
		if err != nil {
			return "", err
		}
		if _, err := c.CreateAtSHA(branch, selected); err != nil {
			return "", err
		}
		actual, err := c.ResolveSHA(branch)
		if err != nil {
			return "", err
		}
		finalBase, err := c.ResolveSHA(base)
		if err != nil {
			return "", err
		}
		if selected != actual || finalBase != selected {
			return "", Fail("RECREATE_BASE_MOVED_OR_REF_MISMATCH", "base moved or recreated ref differs")
		}
		return actual, nil
	}
	newSHA, err := recreate()
	if err != nil {
		if _, restoreErr := c.CreateAtSHA(branch, old); restoreErr != nil {
			return nil, Fail("RECREATE_FAILED_ROLLBACK_FAILED", fmt.Sprintf("recreate failed: %v; rollback failed: %v", err, restoreErr))
		}
		return nil, Fail("RECREATE_FAILED_ROLLED_BACK", fmt.Sprintf("%v; original ref restored", err))
	}
	return Object{"status": "RECREATED", "repository": c.Repository, "branch": branch, "base_ref": base, "old_sha": old, "new_sha": newSHA, "creation_mechanism": "GITHUB_CREATE_BRANCH_API", "safety": "OLD_BRANCH_FULLY_CONTAINED_IN_BASE"}, nil
}
func (r *Repository) BranchControl(command string, options map[string]string) (any, error) {
	mechanism, err := r.RootSection("MPD-CNTR-0001", "unit_mpd_0014_2636c6da29a8")
	if err != nil {
		return nil, err
	}
	if !Has(Strings(Map(mechanism)["registered_commands"]), strings.ToUpper(command)) {
		return nil, Fail("UNREGISTERED_BRANCH_COMMAND", command)
	}
	if command == "commands" {
		return Object{"status": "REGISTERED_COMMANDS", "commands": Map(mechanism)["registered_commands"], "unregistered_operation": "FAIL_CLOSED", "canonical_entrypoint": "developer/automation/cmd/ptsip-dev"}, nil
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	client, err := NewBranchClient(options["--github-repository"], token, "")
	if err != nil {
		return nil, err
	}
	branch := options["--branch"]
	switch command {
	case "create":
		return r.CreateBranch(client, branch, options["--approved-name"], options["--base-ref"])
	case "recreate":
		return r.RecreateBranch(client, branch, options["--approved-name"], options["--base-ref"])
	case "inspect":
		value, err := client.Request("GET", "/repos/"+client.Repository+"/branches/"+branchRef(branch), nil)
		if err != nil {
			return nil, err
		}
		item := Map(value)
		if item == nil {
			return nil, Fail("INVALID_GITHUB_RESPONSE", "branch response is not an object")
		}
		return Object{"status": "OK", "repository": client.Repository, "branch": item["name"], "sha": Map(item["commit"])["sha"], "protected": item["protected"]}, nil
	case "list":
		branches := []any{}
		for page := 1; ; page++ {
			value, err := client.Request("GET", fmt.Sprintf("/repos/%s/branches?per_page=100&page=%d", client.Repository, page), nil)
			if err != nil {
				return nil, err
			}
			if List(value) == nil {
				return nil, Fail("INVALID_GITHUB_RESPONSE", "branch list response is not a list")
			}
			for _, raw := range List(value) {
				if item := Map(raw); item != nil {
					branches = append(branches, Object{"name": item["name"], "sha": Map(item["commit"])["sha"]})
				}
			}
			if len(List(value)) < 100 {
				return Object{"status": "OK", "repository": client.Repository, "branches": branches}, nil
			}
		}
	}
	return nil, Fail("UNREGISTERED_BRANCH_COMMAND", command)
}
func init() {
	RegisterOperations("branch-control", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		return r.BranchControl(command, options)
	})
	RegisterOperations("branch-guard", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "validate":
			return r.ValidateBranchCreation(options["--candidate"], options["--approved-name"], options["--authorization-source"], options["--request-kind"], options["--creation-mechanism"])
		case "classify-existing":
			return r.ClassifyBranch(options["--branch"])
		case "profile-transition":
			return r.BranchProfileTransition(options["--branch"], options["--from-profile"], options["--to-profile"])
		}
		return nil, Fail("UNREGISTERED_BRANCH_COMMAND", command)
	})
}
