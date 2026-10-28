package branch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	Repository, Token, APIRoot string
	HTTP                       *http.Client
}

func NewClient(repository, token, apiRoot string) (*Client, error) {
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(repository) {
		return nil, Fail("INVALID_REPOSITORY", "repository must be owner/name")
	}
	if apiRoot == "" {
		apiRoot = "https://api.github.com"
	}
	parsed, err := url.Parse(apiRoot)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, Fail("INVALID_API_ROOT", "HTTPS API root required")
	}
	return &Client{Repository: repository, Token: token, APIRoot: strings.TrimRight(apiRoot, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}, nil
}

// Request is read-only during the temporary transition. No token or forged
// completion flag can enable POST, PATCH or DELETE through this client.
func (c *Client) Request(method, path string, payload any) (any, error) {
	if method != http.MethodGet || payload != nil {
		return nil, Fail("BRANCH_MUTATION_ADMISSION_UNRESOLVED", "branch mutation has no successor authority contract")
	}
	if !strings.HasPrefix(path, "/repos/"+c.Repository+"/") || strings.ContainsAny(path, "\r\n") {
		return nil, Fail("INVALID_GITHUB_PATH", path)
	}
	request, err := http.NewRequest(method, c.APIRoot+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "PTSIP-branch-control")
	if c.Token != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return nil, Fail("GITHUB_API_UNREACHABLE", err.Error())
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 32*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 32*1024*1024 {
		return nil, Fail("GITHUB_RESPONSE_TOO_LARGE", "response exceeds limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, Fail("GITHUB_API_ERROR", fmt.Sprintf("HTTP %d: %s", response.StatusCode, body))
	}
	var result any
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func branchRef(name string) string { return url.PathEscape(strings.TrimPrefix(name, "refs/heads/")) }
func (c *Client) ResolveSHA(name string) (string, error) {
	value, err := c.Request("GET", "/repos/"+c.Repository+"/git/ref/heads/"+branchRef(name), nil)
	if err != nil {
		return "", err
	}
	sha := text(object(object(value)["object"])["sha"])
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(sha) {
		return "", Fail("INVALID_GITHUB_RESPONSE", "expected full object.sha")
	}
	return sha, nil
}
