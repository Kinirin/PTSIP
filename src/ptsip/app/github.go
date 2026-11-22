package app

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const githubResponseLimit = 8 << 20

var ErrGitHubHeadChanged = errors.New("GitHub branch head changed before update")

type GitHubAPIError struct {
	Method     string
	Path       string
	StatusCode int
}

func (e *GitHubAPIError) Error() string {
	return fmt.Sprintf("GitHub API %s %s failed: HTTP %d", e.Method, e.Path, e.StatusCode)
}

type githubToken struct {
	value     string
	expiresAt time.Time
}

type GitHubAppClient struct {
	appID      string
	privateKey *rsa.PrivateKey
	baseURL    string
	http       *http.Client
	mu         sync.Mutex
	tokens     map[int64]githubToken
}

var _ GitHub = (*GitHubAppClient)(nil)

// NewGitHubAppClient uses the GitHub App environment of the control plane.
func NewGitHubAppClient() (*GitHubAppClient, error) {
	key := []byte(os.Getenv("PTSIP_GITHUB_PRIVATE_KEY"))
	if file := os.Getenv("PTSIP_GITHUB_PRIVATE_KEY_PATH"); len(key) == 0 && file != "" {
		if strings.HasPrefix(file, "~/") || strings.HasPrefix(file, `~\`) {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			file = filepath.Join(home, file[2:])
		}
		var err error
		key, err = os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read GitHub App private key: %w", err)
		}
	}
	return newGitHubAppClient(os.Getenv("PTSIP_GITHUB_APP_ID"), key, os.Getenv("PTSIP_GITHUB_API_URL"), nil)
}

func newGitHubAppClient(appID string, key []byte, apiURL string, client *http.Client) (*GitHubAppClient, error) {
	if apiURL == "" {
		apiURL = "https://api.github.com"
	}
	base, err := url.Parse(apiURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" ||
		base.User != nil || base.ForceQuery || base.RawQuery != "" || base.Fragment != "" || base.Opaque != "" {
		return nil, fmt.Errorf("invalid GitHub API base URL")
	}
	var privateKey *rsa.PrivateKey
	if len(key) > 0 {
		privateKey, err = parseGitHubKey(key)
		if err != nil {
			return nil, err
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	transport := *client
	transport.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &GitHubAppClient{
		appID: appID, privateKey: privateKey, baseURL: strings.TrimRight(base.String(), "/"),
		http: &transport, tokens: make(map[int64]githubToken),
	}, nil
}

func parseGitHubKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("GitHub App private key must be PEM-encoded RSA")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return privateKey, nil
		}
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if privateKey, ok := parsed.(*rsa.PrivateKey); err == nil && ok {
			return privateKey, nil
		}
	}
	return nil, fmt.Errorf("GitHub App private key must contain a valid PKCS#1 or PKCS#8 RSA key")
}

func (c *GitHubAppClient) appJWT(now time.Time) (string, error) {
	if c.appID == "" || c.privateKey == nil {
		return "", fmt.Errorf("GitHub App credentials are not configured")
	}
	claims, err := json.Marshal(struct {
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
		Issuer    string `json:"iss"`
	}{now.Add(-time.Minute).Unix(), now.Add(9 * time.Minute).Unix(), c.appID})
	if err != nil {
		return "", err
	}
	base := base64.RawURLEncoding
	unsigned := base.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + base.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign GitHub App token: %w", err)
	}
	return unsigned + "." + base.EncodeToString(signature), nil
}

func (c *GitHubAppClient) request(ctx context.Context, method, path, token string, payload, target any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ptsip-github-app/0.3.1")
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &GitHubAPIError{Method: method, Path: request.URL.EscapedPath(), StatusCode: response.StatusCode}
	}
	if target == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(response.Body, githubResponseLimit))
		return err
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, githubResponseLimit+1))
	if err != nil {
		return err
	}
	if len(data) > githubResponseLimit {
		return fmt.Errorf("GitHub API response is too large")
	}
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return fmt.Errorf("GitHub API returned a non-object response")
	}
	return json.Unmarshal(data, target)
}

func repositoryAPIPath(repository string) (string, error) {
	owner, name, found := strings.Cut(repository, "/")
	if !found || strings.Contains(name, "/") || !githubPathSegment(owner) || !githubPathSegment(name) {
		return "", fmt.Errorf("repository must use owner/repository form")
	}
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name), nil
}

func githubPathSegment(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "\\\x00\r\n")
}

func (c *GitHubAppClient) RepositoryInstallation(ctx context.Context, repository string) (int64, error) {
	repo, err := repositoryAPIPath(repository)
	if err != nil {
		return 0, err
	}
	jwt, err := c.appJWT(time.Now())
	if err != nil {
		return 0, err
	}
	var data struct {
		ID int64 `json:"id"`
	}
	if err := c.request(ctx, http.MethodGet, repo+"/installation", jwt, nil, &data); err != nil {
		return 0, err
	}
	if data.ID <= 0 {
		return 0, fmt.Errorf("GitHub did not return a repository installation id")
	}
	return data.ID, nil
}

func (c *GitHubAppClient) installationToken(ctx context.Context, id int64) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("invalid GitHub installation id")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	now := time.Now()
	c.mu.Lock()
	cached := c.tokens[id]
	c.mu.Unlock()
	if cached.expiresAt.After(now.Add(time.Minute)) {
		return cached.value, nil
	}
	jwt, err := c.appJWT(now)
	if err != nil {
		return "", err
	}
	var data struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := c.request(ctx, http.MethodPost, fmt.Sprintf("/app/installations/%d/access_tokens", id), jwt, struct{}{}, &data); err != nil {
		return "", err
	}
	expires := now.Add(50 * time.Minute)
	if !data.ExpiresAt.IsZero() && data.ExpiresAt.Before(expires) {
		expires = data.ExpiresAt
	}
	if data.Token == "" || !expires.After(time.Now()) {
		return "", fmt.Errorf("GitHub did not return a usable installation token")
	}
	c.mu.Lock()
	if !c.tokens[id].expiresAt.After(expires) {
		c.tokens[id] = githubToken{value: data.Token, expiresAt: expires}
	}
	c.mu.Unlock()
	return data.Token, nil
}

func (c *GitHubAppClient) repoRequest(ctx context.Context, method, repository string, installationID int64, suffix string, payload, target any) error {
	repo, err := repositoryAPIPath(repository)
	if err != nil {
		return err
	}
	token, err := c.installationToken(ctx, installationID)
	if err != nil {
		return err
	}
	return c.request(ctx, method, repo+suffix, token, payload, target)
}

func (c *GitHubAppClient) CreateIssue(ctx context.Context, repository string, installationID int64, title, body string) (Issue, error) {
	var data struct {
		Number int64  `json:"number"`
		URL    string `json:"html_url"`
	}
	if err := c.repoRequest(ctx, http.MethodPost, repository, installationID, "/issues", map[string]string{"title": title, "body": body}, &data); err != nil {
		return Issue{}, err
	}
	if data.Number <= 0 || data.URL == "" {
		return Issue{}, fmt.Errorf("GitHub did not return an issue number and URL")
	}
	return Issue{Number: data.Number, URL: data.URL}, nil
}

func (c *GitHubAppClient) UpdateIssue(ctx context.Context, repository string, installationID, issueNumber int64, state string) error {
	if issueNumber <= 0 || (state != "open" && state != "closed") {
		return fmt.Errorf("invalid GitHub issue number or state")
	}
	return c.repoRequest(ctx, http.MethodPatch, repository, installationID, fmt.Sprintf("/issues/%d", issueNumber), map[string]string{"state": state}, nil)
}

func (c *GitHubAppClient) Comment(ctx context.Context, repository string, installationID, issueNumber int64, body string) error {
	if issueNumber <= 0 {
		return fmt.Errorf("invalid GitHub issue number")
	}
	return c.repoRequest(ctx, http.MethodPost, repository, installationID, fmt.Sprintf("/issues/%d/comments", issueNumber), map[string]string{"body": body}, nil)
}

func (c *GitHubAppClient) Permission(ctx context.Context, repository string, installationID int64, user string) (string, error) {
	if !githubPathSegment(user) || strings.Contains(user, "/") {
		return "", fmt.Errorf("invalid GitHub collaborator name")
	}
	var data struct {
		Permission string `json:"permission"`
	}
	path := "/collaborators/" + url.PathEscape(user) + "/permission"
	if err := c.repoRequest(ctx, http.MethodGet, repository, installationID, path, nil, &data); err != nil {
		return "", err
	}
	if data.Permission == "" {
		return "none", nil
	}
	return data.Permission, nil
}

func githubBranchPath(branch string) (string, error) {
	for _, segment := range strings.Split(branch, "/") {
		if !githubPathSegment(segment) {
			return "", fmt.Errorf("invalid GitHub branch name")
		}
	}
	return url.PathEscape(branch), nil
}

func gitObjectID(sha string) bool {
	if len(sha) != 40 && len(sha) != 64 {
		return false
	}
	_, err := hex.DecodeString(sha)
	return err == nil
}

type githubObjectRef struct {
	SHA string `json:"sha"`
}

type githubRepoRequest func(context.Context, string, string, any, any) error

func (c *GitHubAppClient) repositoryRequests(repository string, installationID int64) githubRepoRequest {
	return func(ctx context.Context, method, path string, payload, target any) error {
		return c.repoRequest(ctx, method, repository, installationID, path, payload, target)
	}
}

func (c *GitHubAppClient) BranchHead(ctx context.Context, repository string, installationID int64, branch string) (string, error) {
	return githubBranchHead(ctx, c.repositoryRequests(repository, installationID), branch)
}

func githubBranchHead(ctx context.Context, request githubRepoRequest, branch string) (string, error) {
	encoded, err := githubBranchPath(branch)
	if err != nil {
		return "", err
	}
	var data struct {
		Object githubObjectRef `json:"object"`
	}
	if err := request(ctx, http.MethodGet, "/git/ref/heads/"+encoded, nil, &data); err != nil {
		return "", err
	}
	if !gitObjectID(data.Object.SHA) {
		return "", fmt.Errorf("GitHub branch ref did not contain a SHA")
	}
	return data.Object.SHA, nil
}

// FileText reads a single repository file at the supplied ref. A missing file
// returns nil, matching the GitHub App's existing reconciliation contract.
func (c *GitHubAppClient) FileText(ctx context.Context, repository string, installationID int64, path, ref string) (*string, error) {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if !githubPathSegment(part) {
			return nil, fmt.Errorf("file path must be repository-relative")
		}
		parts[i] = url.PathEscape(part)
	}
	var data struct {
		Content  *string `json:"content"`
		Encoding string  `json:"encoding"`
	}
	suffix := "/contents/" + strings.Join(parts, "/") + "?" + url.Values{"ref": {ref}}.Encode()
	if err := c.repoRequest(ctx, http.MethodGet, repository, installationID, suffix, nil, &data); err != nil {
		var apiErr *GitHubAPIError
		if errors.As(err, &apiErr) && apiErr.Method == http.MethodGet && apiErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	if data.Content == nil {
		return nil, nil
	}
	if data.Encoding != "" && data.Encoding != "base64" {
		return nil, fmt.Errorf("GitHub file content has an unsupported encoding")
	}
	decoded, err := base64.StdEncoding.DecodeString(*data.Content)
	if err != nil || !utf8.Valid(decoded) {
		return nil, fmt.Errorf("GitHub file content is not base64-encoded UTF-8")
	}
	content := strings.TrimPrefix(string(decoded), "\ufeff")
	return &content, nil
}

// CommitFileAtParent advances an existing branch with a single-file commit.
// The final ref update uses GitHub's non-forced fast-forward check.
func (c *GitHubAppClient) CommitFileAtParent(ctx context.Context, repository string, installationID int64, branch, parentSHA, path, content, message string) (string, error) {
	if !gitObjectID(parentSHA) || message == "" {
		return "", fmt.Errorf("expected parent SHA and commit message are required")
	}
	for _, part := range strings.Split(path, "/") {
		if !githubPathSegment(part) {
			return "", fmt.Errorf("file path must be repository-relative")
		}
	}
	checkHead := func() error {
		head, err := c.BranchHead(ctx, repository, installationID, branch)
		if err != nil {
			return err
		}
		if head != parentSHA {
			return ErrGitHubHeadChanged
		}
		return nil
	}
	if err := checkHead(); err != nil {
		return "", err
	}
	request := c.repositoryRequests(repository, installationID)
	baseTree, err := githubCommitTree(ctx, request, parentSHA)
	if err != nil {
		return "", err
	}
	commit, err := githubFileCommit(ctx, request, parentSHA, baseTree, path, content, message)
	if err != nil {
		return "", err
	}
	if err := checkHead(); err != nil {
		return "", err
	}
	if err := githubAdvanceBranch(ctx, request, branch, commit); err != nil {
		return "", err
	}
	return commit, nil
}

func githubCommitTree(ctx context.Context, request githubRepoRequest, parentSHA string) (string, error) {
	var parent struct {
		Tree githubObjectRef `json:"tree"`
	}
	if err := request(ctx, http.MethodGet, "/git/commits/"+parentSHA, nil, &parent); err != nil {
		return "", err
	}
	if !gitObjectID(parent.Tree.SHA) {
		return "", fmt.Errorf("parent commit did not contain a tree SHA")
	}
	return parent.Tree.SHA, nil
}

func githubFileCommit(ctx context.Context, request githubRepoRequest, parentSHA, baseTree, path, content, message string) (string, error) {
	var tree githubObjectRef
	if err := request(ctx, http.MethodPost, "/git/trees", map[string]any{
		"base_tree": baseTree,
		"tree":      []map[string]string{{"path": path, "mode": "100644", "type": "blob", "content": content}},
	}, &tree); err != nil {
		return "", err
	}
	if !gitObjectID(tree.SHA) {
		return "", fmt.Errorf("GitHub did not return a created tree SHA")
	}
	var commit githubObjectRef
	if err := request(ctx, http.MethodPost, "/git/commits", map[string]any{
		"message": message, "tree": tree.SHA, "parents": []string{parentSHA},
	}, &commit); err != nil {
		return "", err
	}
	if !gitObjectID(commit.SHA) {
		return "", fmt.Errorf("GitHub did not return a created commit SHA")
	}
	return commit.SHA, nil
}

func githubAdvanceBranch(ctx context.Context, request githubRepoRequest, branch, commit string) error {
	encoded, err := githubBranchPath(branch)
	if err != nil {
		return err
	}
	return request(ctx, http.MethodPatch, "/git/refs/heads/"+encoded, map[string]any{"sha": commit, "force": false}, nil)
}
