package machine

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type branchRoundTripFunc func(*http.Request) (*http.Response, error)

func (f branchRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func branchHTTPResponse(status int, payload any) *http.Response {
	body := ""
	if payload != nil {
		encoded, _ := json.Marshal(payload)
		body = string(encoded)
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type branchHTTPState struct {
	refs           map[string]string
	behindBy       float64
	failNextCreate bool
	deleteCalls    int
	createCalls    int
}

func (s *branchHTTPState) transport(t *testing.T) http.RoundTripper {
	t.Helper()
	return branchRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		const repositoryPrefix = "/repos/Kinirin/PTSIP"
		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, repositoryPrefix+"/git/ref/heads/"):
			ref := strings.TrimPrefix(request.URL.Path, repositoryPrefix+"/git/ref/heads/")
			sha, ok := s.refs[ref]
			if !ok {
				return branchHTTPResponse(http.StatusNotFound, Object{"message": "not found"}), nil
			}
			return branchHTTPResponse(http.StatusOK, Object{"object": Object{"sha": sha}}), nil

		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, repositoryPrefix+"/compare/"):
			return branchHTTPResponse(http.StatusOK, Object{"behind_by": s.behindBy, "ahead_by": float64(0)}), nil

		case request.Method == http.MethodDelete && strings.HasPrefix(request.URL.Path, repositoryPrefix+"/git/refs/heads/"):
			ref := strings.TrimPrefix(request.URL.Path, repositoryPrefix+"/git/refs/heads/")
			s.deleteCalls++
			delete(s.refs, ref)
			return branchHTTPResponse(http.StatusNoContent, nil), nil

		case request.Method == http.MethodPost && request.URL.Path == repositoryPrefix+"/git/refs":
			if request.Header.Get("Authorization") == "" {
				t.Fatal("authenticated branch mutation omitted Authorization header")
			}
			var payload Object
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create-ref payload: %v", err)
			}
			s.createCalls++
			if s.failNextCreate {
				s.failNextCreate = false
				return branchHTTPResponse(http.StatusInternalServerError, Object{"message": "simulated failure"}), nil
			}
			ref := strings.TrimPrefix(Text(payload["ref"]), "refs/heads/")
			sha := Text(payload["sha"])
			s.refs[ref] = sha
			return branchHTTPResponse(http.StatusCreated, Object{"ref": "refs/heads/" + ref, "object": Object{"sha": sha}}), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL.String())
			return nil, nil
		}
	})
}

func branchControlTestClient(t *testing.T, state *branchHTTPState) *BranchClient {
	t.Helper()
	client, err := NewBranchClient("Kinirin/PTSIP", "test-token", "https://example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	client.HTTP = &http.Client{Transport: state.transport(t)}
	return client
}

func TestBranchControlRegisteredCommandsArePolicyBound(t *testing.T) {
	repo := branchGuardTestRepo(t)
	raw, err := repo.BranchControl("commands", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	result := Map(raw)
	commands := Strings(result["commands"])
	for _, required := range []string{"COMMANDS", "LIST", "INSPECT", "CREATE", "RECREATE"} {
		if !Has(commands, required) {
			t.Fatalf("registered branch command %s missing from %#v", required, commands)
		}
	}
	if result["unregistered_operation"] != "FAIL_CLOSED" {
		t.Fatalf("unexpected command fallback: %#v", result)
	}
	if _, err := repo.BranchControl("invented", map[string]string{}); branchGuardErrorCode(t, err) != "UNREGISTERED_BRANCH_COMMAND" {
		t.Fatalf("unregistered command admitted: %v", err)
	}
}

func TestBranchControlClientRepositoryShapeFailsClosed(t *testing.T) {
	if _, err := NewBranchClient("not-a-repository", "token", ""); branchGuardErrorCode(t, err) != "INVALID_REPOSITORY" {
		t.Fatalf("invalid repository shape admitted: %v", err)
	}
}

func TestBranchControlCreateUsesExactApprovedNameAndBaseSHA(t *testing.T) {
	repo := branchGuardTestRepo(t)
	baseSHA := strings.Repeat("b", 40)
	state := &branchHTTPState{refs: map[string]string{"main": baseSHA}}
	client := branchControlTestClient(t, state)

	result, err := repo.CreateBranch(client, "dev/0.10.0", "dev/0.10.0", "main")
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "CREATED" ||
		result["branch"] != "dev/0.10.0" ||
		result["base_sha"] != baseSHA ||
		result["creation_mechanism"] != "GITHUB_CREATE_BRANCH_API" {
		t.Fatalf("unexpected create result: %#v", result)
	}
	if state.refs["dev/0.10.0"] != baseSHA || state.createCalls != 1 {
		t.Fatalf("branch was not created at exact base SHA: %#v", state)
	}

	if _, err := repo.CreateBranch(client, "dev/0.10.1", "dev/0.10.0", "main"); branchGuardErrorCode(t, err) != "BRANCH_NAME_NOT_EXACTLY_APPROVED" {
		t.Fatalf("unapproved branch name reached mutation path: %v", err)
	}
	if state.createCalls != 1 {
		t.Fatalf("failed guard performed API mutation: %d", state.createCalls)
	}
}

func TestBranchControlRecreateRequiresFullContainment(t *testing.T) {
	repo := branchGuardTestRepo(t)
	state := &branchHTTPState{
		refs: map[string]string{
			"main":      strings.Repeat("b", 40),
			"dev/0.3.8": strings.Repeat("a", 40),
		},
		behindBy: 1,
	}
	client := branchControlTestClient(t, state)

	_, err := repo.RecreateBranch(client, "dev/0.3.8", "dev/0.3.8", "main")
	if branchGuardErrorCode(t, err) != "BRANCH_HAS_UNMERGED_COMMITS" {
		t.Fatalf("non-contained branch produced wrong failure: %v", err)
	}
	if state.deleteCalls != 0 || state.createCalls != 0 {
		t.Fatalf("containment failure mutated remote state: %#v", state)
	}
}

func TestBranchControlRecreateMovesExactRefToCurrentBase(t *testing.T) {
	repo := branchGuardTestRepo(t)
	oldSHA := strings.Repeat("a", 40)
	baseSHA := strings.Repeat("b", 40)
	state := &branchHTTPState{
		refs: map[string]string{
			"main":      baseSHA,
			"dev/0.3.8": oldSHA,
		},
		behindBy: 0,
	}
	client := branchControlTestClient(t, state)

	result, err := repo.RecreateBranch(client, "dev/0.3.8", "dev/0.3.8", "main")
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "RECREATED" ||
		result["old_sha"] != oldSHA ||
		result["new_sha"] != baseSHA ||
		result["safety"] != "OLD_BRANCH_FULLY_CONTAINED_IN_BASE" {
		t.Fatalf("unexpected recreate result: %#v", result)
	}
	if state.deleteCalls != 1 || state.createCalls != 1 || state.refs["dev/0.3.8"] != baseSHA {
		t.Fatalf("unexpected recreate mutation sequence: %#v", state)
	}
}

func TestBranchControlRecreateRollsBackOriginalRef(t *testing.T) {
	repo := branchGuardTestRepo(t)
	oldSHA := strings.Repeat("a", 40)
	baseSHA := strings.Repeat("b", 40)
	state := &branchHTTPState{
		refs: map[string]string{
			"main":      baseSHA,
			"dev/0.3.8": oldSHA,
		},
		behindBy:       0,
		failNextCreate: true,
	}
	client := branchControlTestClient(t, state)

	_, err := repo.RecreateBranch(client, "dev/0.3.8", "dev/0.3.8", "main")
	if branchGuardErrorCode(t, err) != "RECREATE_FAILED_ROLLED_BACK" {
		t.Fatalf("unexpected rollback failure: %v", err)
	}
	if state.deleteCalls != 1 || state.createCalls != 2 || state.refs["dev/0.3.8"] != oldSHA {
		t.Fatalf("original branch ref was not restored: %#v", state)
	}
}

func TestBranchControlRecreateRejectsSameBaseBeforeMutation(t *testing.T) {
	repo := branchGuardTestRepo(t)
	state := &branchHTTPState{refs: map[string]string{"dev/0.3.8": strings.Repeat("a", 40)}}
	client := branchControlTestClient(t, state)

	_, err := repo.RecreateBranch(client, "dev/0.3.8", "dev/0.3.8", "dev/0.3.8")
	if branchGuardErrorCode(t, err) != "RECREATE_BASE_EQUALS_BRANCH" {
		t.Fatalf("same branch/base produced wrong failure: %v", err)
	}
	if state.deleteCalls != 0 || state.createCalls != 0 {
		t.Fatalf("same branch/base performed mutation: %#v", state)
	}
}
