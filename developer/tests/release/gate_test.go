package release_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestReleaseGateRequiresSuccessfulExactSHAContexts(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	sourceSHA := "0123456789abcdef0123456789abcdef01234567"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/ptsip/commits/"+sourceSHA+"/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"statuses": []map[string]string{
				{"context": "ci/tooling-test", "state": "success"},
				{"context": "ci/pp-transition", "state": "success"},
			},
		})
	}))
	defer server.Close()

	result, err, output := runReleaseCommand(
		t,
		binary,
		testrepo.Root(t),
		map[string]string{"GITHUB_TOKEN": "fixture-token"},
		"release", "gate",
		"--github-repository", "acme/ptsip",
		"--github-api-url", server.URL,
		"--source-sha", sourceSHA,
	)
	if err != nil || result["status"] != "PASS" || result["source_sha"] != sourceSHA {
		t.Fatalf("exact-SHA release gate rejected: %v\n%s\n%#v", err, output, result)
	}
}

func TestReleaseGateFailsWhenRequiredContextIsMissing(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	sourceSHA := "0123456789abcdef0123456789abcdef01234567"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"statuses": []map[string]string{
				{"context": "ci/tooling-test", "state": "success"},
			},
		})
	}))
	defer server.Close()

	_, err, output := runReleaseCommand(
		t,
		binary,
		testrepo.Root(t),
		map[string]string{"GITHUB_TOKEN": "fixture-token"},
		"release", "gate",
		"--github-repository", "acme/ptsip",
		"--github-api-url", server.URL,
		"--source-sha", sourceSHA,
	)
	if err == nil || !strings.Contains(output, "RELEASE_GATE_UNSATISFIED") {
		t.Fatalf("missing exact-SHA status was not rejected: %v\n%s", err, output)
	}
}

func TestReleaseGateReconfirmsMainAfterVerification(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	root, _, verifiedSHA := newReleaseFixture(t, "1.2.3a1")

	result, err, output := runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "reconfirm",
		"--source-sha", verifiedSHA,
	)
	if err != nil || result["status"] != "PASS" {
		t.Fatalf("unchanged main was not reconfirmed: %v\n%s\n%#v", err, output, result)
	}

	writeReleaseFile(t, root, "next.txt", "moved\n")
	releaseGit(t, root, "add", "next.txt")
	releaseGit(t, root, "commit", "-m", "move main")
	releaseGit(t, root, "push", "origin", "main")

	_, err, output = runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "reconfirm",
		"--source-sha", verifiedSHA,
	)
	if err == nil || !strings.Contains(output, "RELEASE_MAIN_MOVED") {
		t.Fatalf("moved main was not rejected: %v\n%s", err, output)
	}
}
