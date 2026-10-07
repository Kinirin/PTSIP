package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

type githubClient struct {
	repository string
	apiRoot    string
	token      string
	http       *http.Client
}

func newGitHubClient(repository, apiRoot string) (*githubClient, error) {
	if strings.Count(repository, "/") != 1 {
		return nil, machine.Fail("RELEASE_GITHUB_REPOSITORY_INVALID", "repository must be owner/name")
	}
	if apiRoot == "" {
		apiRoot = os.Getenv("GITHUB_API_URL")
	}
	if apiRoot == "" {
		apiRoot = "https://api.github.com"
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		return nil, machine.Fail("RELEASE_GITHUB_TOKEN_REQUIRED", "GITHUB_TOKEN or GH_TOKEN is required")
	}
	return &githubClient{
		repository: repository,
		apiRoot:    strings.TrimRight(apiRoot, "/"),
		token:      token,
		http:       &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *githubClient) request(method, path string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.apiRoot+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "PTSIP-release-automation")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return machine.Fail("RELEASE_GITHUB_API_UNREACHABLE", err.Error())
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return machine.Fail("RELEASE_GITHUB_API_ERROR", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(data)))
	}
	if target != nil && len(data) != 0 {
		if err := json.Unmarshal(data, target); err != nil {
			return err
		}
	}
	return nil
}

func RequireExactCIGate(repository, apiRoot, sourceSHA string) (machine.Object, error) {
	client, err := newGitHubClient(repository, apiRoot)
	if err != nil {
		return nil, err
	}
	var response struct {
		Statuses []struct {
			Context string `json:"context"`
			State   string `json:"state"`
		} `json:"statuses"`
	}
	if err := client.request("GET", "/repos/"+repository+"/commits/"+sourceSHA+"/status", nil, &response); err != nil {
		return nil, err
	}
	required := []string{"ci/tooling-test", "ci/pp-transition"}
	for _, context := range required {
		found := false
		for _, status := range response.Statuses {
			if status.Context == context && status.State == "success" {
				found = true
				break
			}
		}
		if !found {
			return nil, machine.Fail("RELEASE_GATE_UNSATISFIED", fmt.Sprintf("no successful %s status for %s", context, sourceSHA))
		}
	}
	return machine.Object{"status": "PASS", "source_sha": sourceSHA, "required_contexts": required}, nil
}

func CreateDraft(repository, apiRoot, sourceSHA, version, tag, note string, repo *machine.Repository) (machine.Object, error) {
	client, err := newGitHubClient(repository, apiRoot)
	if err != nil {
		return nil, err
	}
	notePath, err := repo.Path(note)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(notePath)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"tag_name":         tag,
		"target_commitish": sourceSHA,
		"name":             "PTSIP Tool " + version,
		"body":             string(body),
		"draft":            true,
		"prerelease":       strings.IndexFunc(version, func(r rune) bool { return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') }) >= 0,
	}
	var result struct {
		HTMLURL string `json:"html_url"`
	}
	if err := client.request("POST", "/repos/"+repository+"/releases", payload, &result); err != nil {
		return nil, err
	}
	return machine.Object{"status": "CREATED", "source_sha": sourceSHA, "tag": tag, "release_url": result.HTMLURL}, nil
}
