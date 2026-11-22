package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGitHubAuthorityStoreReadsMarkerAndUpdatesExistingRef(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const repo = "/repos/owner/example"
	const branch = "ptsip-policy"
	const decision = "decisions/gdec-test.json"
	const ref = repo + "/git/ref/heads/" + branch
	head := strings.Repeat("a", 40)
	tree := strings.Repeat("b", 40)
	marker := strings.Repeat("c", 40)
	record := strings.Repeat("d", 40)
	newTree := strings.Repeat("e", 40)
	commit := strings.Repeat("f", 40)
	getHead := githubExchange{"GET", ref, "", `{"object":{"sha":"` + head + `"}}`, 0}
	getCommit := githubExchange{"GET", repo + "/git/commits/" + head, "", `{"tree":{"sha":"` + tree + `"}}`, 0}
	getTree := githubExchange{"GET", repo + "/git/trees/" + tree + "?recursive=1", "",
		`{"truncated":false,"tree":[{"path":"authority.json","mode":"100644","type":"blob","sha":"` + marker + `"},` +
			`{"path":"` + decision + `","mode":"100644","type":"blob","sha":"` + record + `"}]}`, 0}
	getBlob := func(sha, data string) githubExchange {
		return githubExchange{"GET", repo + "/git/blobs/" + sha, "",
			`{"encoding":"base64","content":"` + base64.StdEncoding.EncodeToString([]byte(data)) + `"}`, 0}
	}
	readMarker := getBlob(marker, `{"format":"ptsip-github-authority/v1","repository":"owner/example","ref":"refs/heads/ptsip-policy"}`)
	verifiedHead := []githubExchange{getHead, getCommit, getTree, readMarker}
	store := func(t *testing.T, exchanges []githubExchange) *GitHubAuthorityStore {
		t.Helper()
		client := scriptedGitHub(t, key, exchanges)
		t.Setenv("PTSIP_GITHUB_API_URL", client.baseURL)
		t.Setenv("GH_TOKEN", "test-installation-token")
		t.Setenv("GITHUB_TOKEN", "different-test-token")
		result, err := NewGitHubAuthorityStore("owner/example", branch)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	ctx := context.Background()
	t.Run("absent branch returns no state and cannot be bootstrapped", func(t *testing.T) {
		s := store(t, []githubExchange{{"GET", ref, "", `{}`, http.StatusNotFound}})
		if head, doc, err := s.ReadAuthority(ctx, decision); err != nil || head != "" || doc != nil {
			t.Fatalf("absent branch read: %q, %v, %v", head, doc, err)
		}
		if _, err := s.WriteAuthority(ctx, decision, Object{"status": "PENDING"}, "", "register"); err == nil {
			t.Fatal("missing expected head authorized a write")
		}
	})
	t.Run("verified marker permits exact decision read", func(t *testing.T) {
		s := store(t, append(append([]githubExchange{}, verifiedHead...), getBlob(record, `{"status":"PENDING"}`)))
		if actual, doc, err := s.ReadAuthority(ctx, decision); err != nil || actual != head || doc["status"] != "PENDING" {
			t.Fatalf("authority record: %q, %v, %v", actual, doc, err)
		}
	})
	t.Run("foreign authority marker stops writes", func(t *testing.T) {
		wrong := getBlob(marker, `{"format":"ptsip-github-authority/v1","repository":"another/example","ref":"refs/heads/ptsip-policy"}`)
		s := store(t, []githubExchange{getHead, getCommit, getTree, wrong})
		if _, err := s.WriteAuthority(ctx, decision, Object{"status": "PENDING"}, head, "register"); err == nil || !strings.Contains(err.Error(), "authority marker") {
			t.Fatalf("incompatible authority marker was accepted: %v", err)
		}
	})
	t.Run("incomplete tree fails closed", func(t *testing.T) {
		truncated := getTree
		truncated.response = strings.Replace(truncated.response, `"truncated":false`, `"truncated":true`, 1)
		s := store(t, []githubExchange{getHead, getCommit, truncated})
		if _, _, err := s.ReadAuthority(ctx, decision); err == nil || !strings.Contains(err.Error(), "tree is incomplete") {
			t.Fatalf("incomplete authority tree was accepted: %v", err)
		}
	})
	value := Object{"status": "PENDING", "purpose": "프로필 <test>"}
	content, _ := json.Marshal("{\n  \"purpose\": \"프로필 <test>\",\n  \"status\": \"PENDING\"\n}\n")
	mutation := []githubExchange{
		{"POST", repo + "/git/trees", `{"base_tree":"` + tree + `","tree":[{"path":"` + decision + `","mode":"100644","type":"blob","content":` + string(content) + `}]}`, `{"sha":"` + newTree + `"}`, http.StatusCreated},
		{"POST", repo + "/git/commits", `{"message":"register","parents":["` + head + `"],"tree":"` + newTree + `"}`, `{"sha":"` + commit + `"}`, http.StatusCreated},
		getHead,
	}
	for _, status := range []int{http.StatusOK, http.StatusConflict} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			patch := githubExchange{"PATCH", repo + "/git/refs/heads/" + branch, `{"sha":"` + commit + `","force":false}`, `{}`, status}
			exchanges := append(append(append([]githubExchange{}, verifiedHead...), mutation...), patch)
			s := store(t, exchanges)
			actual, err := s.WriteAuthority(ctx, decision, value, head, "register")
			if status == http.StatusOK && (err != nil || actual != commit) {
				t.Fatalf("authority write: %q, %v", actual, err)
			}
			if status == http.StatusConflict && (!errors.Is(err, ErrAuthorityConflict) || actual != "") {
				t.Fatalf("authority conflict: %q, %v", actual, err)
			}
		})
	}
}
