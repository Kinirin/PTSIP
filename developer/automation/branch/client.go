package branch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

type Client struct {
	Repository string
	Token      string
	APIRoot    string
	HTTP       *http.Client
}

func NewClient(repository, token, apiRoot string) (*Client, error) {
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(repository) {
		return nil, machine.Fail("INVALID_REPOSITORY", "repository must be in owner/name form")
	}
	if apiRoot == "" {
		apiRoot = "https://api.github.com"
	}
	return &Client{
		Repository: repository,
		Token: token,
		APIRoot: strings.TrimRight(apiRoot, "/"),
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) Request(method, path string, payload any) (any, error) {
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
		return nil, machine.Fail("GITHUB_API_UNREACHABLE", err.Error())
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 32*1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, machine.Fail("GITHUB_API_ERROR", fmt.Sprintf("HTTP %d: %s", response.StatusCode, body))
	}
	if len(body) == 0 {
		return machine.Object{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func branchRef(ref string) string {
	return url.PathEscape(strings.TrimPrefix(ref, "refs/heads/"))
}

func (c *Client) ResolveSHA(ref string) (string, error) {
	value, err := c.Request("GET", "/repos/"+c.Repository+"/git/ref/heads/"+branchRef(ref), nil)
	if err != nil {
		return "", err
	}
	sha := machine.Text(machine.Map(machine.Map(value)["object"])["sha"])
	if sha == "" {
		return "", machine.Fail("INVALID_GITHUB_RESPONSE", "ref response has no object.sha")
	}
	return sha, nil
}

func (c *Client) CreateAtSHA(branch, sha string) (machine.Object, error) {
	if c.Token == "" {
		return nil, machine.Fail("GITHUB_TOKEN_REQUIRED", "branch creation requires GITHUB_TOKEN or GH_TOKEN")
	}
	value, err := c.Request("POST", "/repos/"+c.Repository+"/git/refs", machine.Object{
		"ref": "refs/heads/" + branch,
		"sha": sha,
	})
	if err != nil {
		return nil, err
	}
	if machine.Map(value) == nil {
		return nil, machine.Fail("INVALID_GITHUB_RESPONSE", "create-ref response is not an object")
	}
	return machine.Map(value), nil
}
