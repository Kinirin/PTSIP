package release

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicationNeedsObservedExactSourceAndAllRequiredJobs(t *testing.T) {
	sha := strings.Repeat("a", 40)
	wrongSource := false
	jobPassed := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/jobs") {
			conclusion := "success"
			if !jobPassed {
				conclusion = "failure"
			}
			fmt.Fprintf(w, `{"jobs":[{"name":"build-context-linux-x64","status":"completed","conclusion":%q}]}`, conclusion)
			return
		}
		source := sha
		if wrongSource {
			source = strings.Repeat("b", 40)
		}
		fmt.Fprintf(w, `{"head_sha":%q,"path":".github/workflows/tooling-release.yml","event":"workflow_dispatch"}`, source)
	}))
	defer server.Close()
	client := &githubClient{repository: "example/repo", apiRoot: server.URL, token: "fixture", http: server.Client()}
	s := Selection{SourceSHA: sha}
	c := Component{ID: "context"}
	receipts := []Receipt{{RunID: "123", Platform: "linux-x64"}}
	if err := verifyBuildEvidence(client, s, c, receipts, "123"); err != nil {
		t.Fatal(err)
	}
	wrongSource = true
	if err := verifyBuildEvidence(client, s, c, receipts, "123"); err == nil {
		t.Fatal("other source SHA supplied publication authority")
	}
	wrongSource = false
	jobPassed = false
	if err := verifyBuildEvidence(client, s, c, receipts, "123"); err == nil {
		t.Fatal("failed component build supplied publication authority")
	}
	jobPassed = true
	receipts[0].RunID = "122"
	if err := verifyBuildEvidence(client, s, c, receipts, "123"); err == nil {
		t.Fatal("old run receipt supplied publication authority")
	}
	if err := verifyBuildEvidence(client, s, c, receipts, ""); err == nil {
		t.Fatal("local self-claimed PASS supplied publication authority")
	}
}
