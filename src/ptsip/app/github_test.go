package app

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type githubExchange struct {
	method, path, request, response string
	status                          int
}

func verifyGitHubTestJWT(t *testing.T, key *rsa.PublicKey, authorization string) {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(authorization, "Bearer "), ".")
	if len(parts) != 3 || !strings.HasPrefix(authorization, "Bearer ") {
		t.Error("GitHub App request lacks a signed bearer JWT")
		return
	}
	encoding := base64.RawURLEncoding
	header, headerErr := encoding.DecodeString(parts[0])
	claims, claimsErr := encoding.DecodeString(parts[1])
	signature, signatureErr := encoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if headerErr != nil || claimsErr != nil || signatureErr != nil ||
		string(header) != `{"alg":"RS256","typ":"JWT"}` || rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
		t.Error("GitHub App JWT failed RS256 signature verification")
		return
	}
	var data struct {
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
		Issuer    string `json:"iss"`
	}
	now := time.Now().Unix()
	if err := json.Unmarshal(claims, &data); err != nil || data.Issuer != "12345" ||
		data.IssuedAt < now-65 || data.IssuedAt >= now || data.ExpiresAt-data.IssuedAt != 600 {
		t.Errorf("unexpected GitHub App JWT claims: %+v, %v", data, err)
	}
}

func scriptedGitHub(t *testing.T, key *rsa.PrivateKey, exchanges []githubExchange) *GitHubAppClient {
	t.Helper()
	var mu sync.Mutex
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if count >= len(exchanges) {
			t.Errorf("unexpected GitHub API call: %s %s", req.Method, req.URL.RequestURI())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		exchange := exchanges[count]
		count++
		if req.Method != exchange.method || req.URL.RequestURI() != exchange.path {
			t.Errorf("GitHub API call %d: got %s %s, want %s %s", count, req.Method, req.URL.RequestURI(), exchange.method, exchange.path)
		}
		if req.Header.Get("Accept") != "application/vnd.github+json" || req.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Error("missing GitHub media type or API version")
		}
		authorization := req.Header.Get("Authorization")
		if strings.HasSuffix(exchange.path, "/installation") || strings.HasSuffix(exchange.path, "/access_tokens") {
			verifyGitHubTestJWT(t, &key.PublicKey, authorization)
		} else if authorization != "Bearer test-installation-token" {
			t.Error("GitHub repository call lacked its installation token")
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
		}
		if exchange.request == "" {
			if len(body) != 0 {
				t.Error("expected bodyless GitHub request")
			}
		} else {
			var got, want any
			if json.Unmarshal(body, &got) != nil || json.Unmarshal([]byte(exchange.request), &want) != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("GitHub request %d: got %s, want %s", count, body, exchange.request)
			}
		}
		status := exchange.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, exchange.response)
	}))
	t.Cleanup(func() {
		server.Close()
		mu.Lock()
		defer mu.Unlock()
		if count != len(exchanges) {
			t.Errorf("made %d of %d GitHub API calls", count, len(exchanges))
		}
	})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	client, err := newGitHubAppClient("12345", keyPEM, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestGitHubAppIssueAndFileEndpoints(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const repo = "owner/example"
	const token = `{"token":"test-installation-token"}`
	const text = "title: 프로필\n"
	encoded := base64.StdEncoding.EncodeToString([]byte("\ufeff" + text))
	client := scriptedGitHub(t, key, []githubExchange{
		{"GET", "/repos/owner/example/installation", "", `{"id":7}`, 0},
		{"POST", "/app/installations/7/access_tokens", `{}`, token, http.StatusCreated},
		{"POST", "/repos/owner/example/issues", `{"title":"Review","body":"Description"}`, `{"number":12,"html_url":"https://example.test/issues/12"}`, http.StatusCreated},
		{"POST", "/repos/owner/example/issues/12/comments", `{"body":"Decision recorded"}`, `{}`, http.StatusCreated},
		{"PATCH", "/repos/owner/example/issues/12", `{"state":"closed"}`, `{}`, 0},
		{"GET", "/repos/owner/example/collaborators/reviewer/permission", "", `{"permission":"write"}`, 0},
		{"GET", "/repos/owner/example/contents/a%20folder/ptsip.yaml?ref=feature%2Fexample", "", `{"encoding":"base64","content":"` + encoded + `"}`, 0},
		{"GET", "/repos/owner/example/contents/missing.yaml?ref=main", "", `{"message":"Not Found"}`, http.StatusNotFound},
		{"POST", "/app/installations/8/access_tokens", `{}`, `{"message":"Not Found"}`, http.StatusNotFound},
	})
	ctx := context.Background()
	installation, err := client.RepositoryInstallation(ctx, repo)
	if err != nil || installation != 7 {
		t.Fatalf("installation: %d, %v", installation, err)
	}
	issue, err := client.CreateIssue(ctx, repo, installation, "Review", "Description")
	if err != nil || issue.Number != 12 || issue.URL != "https://example.test/issues/12" {
		t.Fatalf("issue: %+v, %v", issue, err)
	}
	if err := client.Comment(ctx, repo, installation, issue.Number, "Decision recorded"); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateIssue(ctx, repo, installation, issue.Number, "closed"); err != nil {
		t.Fatal(err)
	}
	if permission, err := client.Permission(ctx, repo, installation, "reviewer"); err != nil || permission != "write" {
		t.Fatalf("permission: %s, %v", permission, err)
	}
	if content, err := client.FileText(ctx, repo, installation, "a folder/ptsip.yaml", "feature/example"); err != nil || content == nil || *content != text {
		t.Fatalf("file result: %v, %v", content, err)
	}
	if content, err := client.FileText(ctx, repo, installation, "missing.yaml", "main"); err != nil || content != nil {
		t.Fatalf("missing file result: %v, %v", content, err)
	}
	var apiError *GitHubAPIError
	if _, err := client.FileText(ctx, repo, 8, "ptsip.yaml", "main"); !errors.As(err, &apiError) || apiError.StatusCode != http.StatusNotFound {
		t.Fatalf("installation error became a missing file: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := client.BranchHead(canceled, repo, installation, "main"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
}

func TestGitHubAppCommitChecksExpectedHeadAndNeverForces(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	parent := strings.Repeat("a", 40)
	baseTree := strings.Repeat("b", 40)
	newTree := strings.Repeat("c", 40)
	newCommit := strings.Repeat("d", 40)
	changed := strings.Repeat("e", 40)
	const repo = "/repos/owner/example"
	head := func(sha string) githubExchange {
		return githubExchange{"GET", repo + "/git/ref/heads/feature%2Fprofile", "", `{"object":{"sha":"` + sha + `"}}`, 0}
	}
	installation := githubExchange{"POST", "/app/installations/7/access_tokens", `{}`, `{"token":"test-installation-token"}`, http.StatusCreated}
	prepared := []githubExchange{installation, head(parent),
		{"GET", repo + "/git/commits/" + parent, "", `{"tree":{"sha":"` + baseTree + `"}}`, 0},
		{"POST", repo + "/git/trees", `{"base_tree":"` + baseTree + `","tree":[{"path":"ptsip.yaml","mode":"100644","type":"blob","content":"example"}]}`, `{"sha":"` + newTree + `"}`, http.StatusCreated},
		{"POST", repo + "/git/commits", `{"message":"Update Profile","tree":"` + newTree + `","parents":["` + parent + `"]}`, `{"sha":"` + newCommit + `"}`, http.StatusCreated},
	}
	patch := githubExchange{"PATCH", repo + "/git/refs/heads/feature%2Fprofile", `{"sha":"` + newCommit + `","force":false}`, `{}`, 0}
	for _, tc := range []struct {
		name      string
		exchanges []githubExchange
		wantError error
	}{
		{"current head", append(append([]githubExchange{}, prepared...), head(parent), patch), nil},
		{"stale before creating commit", []githubExchange{installation, head(changed)}, ErrGitHubHeadChanged},
		{"changed before advancing ref", append(append([]githubExchange{}, prepared...), head(changed)), ErrGitHubHeadChanged},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := scriptedGitHub(t, key, tc.exchanges)
			actual, err := client.CommitFileAtParent(context.Background(), "owner/example", 7, "feature/profile", parent, "ptsip.yaml", "example", "Update Profile")
			if !errors.Is(err, tc.wantError) || (tc.wantError == nil && actual != newCommit) || (tc.wantError != nil && actual != "") {
				t.Fatalf("commit result: %s, %v", actual, err)
			}
		})
	}
}
