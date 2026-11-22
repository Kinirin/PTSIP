package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"
)

const githubAuthorityFormat = "ptsip-github-authority/v1"
const githubAuthorityMarker = "authority.json"

var githubAuthorityRecordPath = regexp.MustCompile(`^decisions/[A-Za-z0-9._-]+\.json$`)

// GitHubAuthorityStore reads and updates an already configured authority ref.
type GitHubAuthorityStore struct {
	repository, branch, repoAPI, hostname, token string
	client                                       *GitHubAppClient
}

var _ AuthorityStore = (*GitHubAuthorityStore)(nil)

// NewGitHubAuthorityStore uses configured user credentials or authenticated gh.
// The caller supplies the existing authority branch explicitly.
func NewGitHubAuthorityStore(repository, branch string) (*GitHubAuthorityStore, error) {
	repoAPI, err := repositoryAPIPath(repository)
	if err != nil {
		return nil, err
	}
	if _, err := githubBranchPath(branch); err != nil {
		return nil, err
	}
	apiURL := os.Getenv("PTSIP_GITHUB_API_URL")
	if apiURL == "" {
		apiURL = os.Getenv("GITHUB_API_URL")
	}
	client, err := newGitHubAppClient("", nil, apiURL, nil)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(client.baseURL)
	if err != nil {
		return nil, err
	}
	hostname := base.Hostname()
	if hostname == "api.github.com" {
		hostname = "github.com"
	}
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return &GitHubAuthorityStore{
		repository: repository, branch: branch, repoAPI: repoAPI, hostname: hostname, token: token, client: client,
	}, nil
}

func (s *GitHubAuthorityStore) requests(ctx context.Context) (githubRepoRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	token := s.token
	if token == "" {
		output, err := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", s.hostname).Output()
		if err != nil || len(bytes.TrimSpace(output)) == 0 {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("GitHub authority requires GH_TOKEN, GITHUB_TOKEN or an authenticated GitHub CLI")
		}
		token = strings.TrimSpace(string(output))
	}
	return func(ctx context.Context, method, path string, payload, target any) error {
		return s.client.request(ctx, method, s.repoAPI+path, token, payload, target)
	}, nil
}

func (s *GitHubAuthorityStore) head(ctx context.Context, request githubRepoRequest) (string, error) {
	head, err := githubBranchHead(ctx, request, s.branch)
	var apiError *GitHubAPIError
	if errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound {
		return "", nil
	}
	return head, err
}

type authorityTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Mode string `json:"mode"`
	SHA  string `json:"sha"`
}

func (s *GitHubAuthorityStore) tree(ctx context.Context, request githubRepoRequest, head string) (string, []authorityTreeEntry, error) {
	sha, err := githubCommitTree(ctx, request, head)
	if err != nil {
		return "", nil, err
	}
	var data struct {
		Entries   []authorityTreeEntry `json:"tree"`
		Truncated bool                 `json:"truncated"`
	}
	if err := request(ctx, http.MethodGet, "/git/trees/"+sha+"?recursive=1", nil, &data); err != nil {
		return "", nil, err
	}
	if data.Truncated || data.Entries == nil {
		return "", nil, fmt.Errorf("GitHub authority tree is incomplete")
	}
	marker, err := readAuthorityBlob(ctx, request, data.Entries, githubAuthorityMarker)
	if err != nil {
		return "", nil, err
	}
	if marker["format"] != githubAuthorityFormat || marker["repository"] != s.repository || marker["ref"] != "refs/heads/"+s.branch {
		return "", nil, fmt.Errorf("existing GitHub ref has no compatible authority marker")
	}
	return sha, data.Entries, nil
}

func authorityBlob(entries []authorityTreeEntry, path string) (string, error) {
	var sha string
	for _, entry := range entries {
		if entry.Path != path {
			continue
		}
		if sha != "" || entry.Type != "blob" || (entry.Mode != "100644" && entry.Mode != "100755") || !gitObjectID(entry.SHA) {
			return "", fmt.Errorf("authority path must resolve to one regular Git blob: %s", path)
		}
		sha = entry.SHA
	}
	return sha, nil
}

func readAuthorityBlob(ctx context.Context, request githubRepoRequest, entries []authorityTreeEntry, path string) (Object, error) {
	sha, err := authorityBlob(entries, path)
	if err != nil || sha == "" {
		return nil, err
	}
	var blob struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := request(ctx, http.MethodGet, "/git/blobs/"+sha, nil, &blob); err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.Strict().DecodeString(blob.Content)
	if err != nil || blob.Encoding != "base64" || !utf8.Valid(data) {
		return nil, fmt.Errorf("GitHub authority document %s is not base64-encoded UTF-8", path)
	}
	return decodeObject(data)
}

func (s *GitHubAuthorityStore) ReadAuthority(ctx context.Context, path string) (string, Object, error) {
	if !githubAuthorityRecordPath.MatchString(path) {
		return "", nil, fmt.Errorf("invalid GitHub authority decision path")
	}
	request, err := s.requests(ctx)
	if err != nil {
		return "", nil, err
	}
	head, err := s.head(ctx, request)
	if err != nil || head == "" {
		return "", nil, err
	}
	_, entries, err := s.tree(ctx, request, head)
	if err != nil {
		return "", nil, err
	}
	record, err := readAuthorityBlob(ctx, request, entries, path)
	return head, record, err
}

func (s *GitHubAuthorityStore) WriteAuthority(ctx context.Context, path string, record Object, expectedHead, message string) (string, error) {
	if !githubAuthorityRecordPath.MatchString(path) || record == nil || message == "" {
		return "", fmt.Errorf("authority decision path, record and commit message are required")
	}
	if !gitObjectID(expectedHead) {
		return "", fmt.Errorf("an existing GitHub authority head is required")
	}
	request, err := s.requests(ctx)
	if err != nil {
		return "", err
	}
	checkHead := func() error {
		head, err := s.head(ctx, request)
		if err != nil {
			return err
		}
		if head != expectedHead {
			return ErrAuthorityConflict
		}
		return nil
	}
	if err := checkHead(); err != nil {
		return "", err
	}
	tree, entries, err := s.tree(ctx, request, expectedHead)
	if err != nil {
		return "", err
	}
	if _, err := authorityBlob(entries, path); err != nil {
		return "", err
	}
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(record); err != nil {
		return "", err
	}
	commit, err := githubFileCommit(ctx, request, expectedHead, tree, path, content.String(), message)
	if err != nil {
		return "", err
	}
	if err := checkHead(); err != nil {
		return "", err
	}
	if err := githubAdvanceBranch(ctx, request, s.branch, commit); err != nil {
		var apiError *GitHubAPIError
		if errors.As(err, &apiError) && (apiError.StatusCode == http.StatusConflict || apiError.StatusCode == http.StatusUnprocessableEntity) {
			return "", fmt.Errorf("%w: %v", ErrAuthorityConflict, err)
		}
		return "", err
	}
	return commit, nil
}
