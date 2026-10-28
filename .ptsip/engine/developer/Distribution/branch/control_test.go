package branch

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reply(payload any) *http.Response {
	raw, _ := json.Marshal(payload)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}
}
func TestBranchQueriesAndPaginationUseOnlyReadTransport(t *testing.T) {
	root := contractFixture(t)
	client, err := NewClient("Kinirin/PTSIP", "test-token", "https://example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client.HTTP = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != "GET" || request.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected transport: %s", request.Method)
		}
		if strings.HasSuffix(request.URL.Path, "/branches/main") {
			return reply(Object{"name": "main", "commit": Object{"sha": strings.Repeat("a", 40)}, "protected": true}), nil
		}
		items := []any{}
		count := 100
		if request.URL.Query().Get("page") == "2" {
			count = 1
		}
		for i := 0; i < count; i++ {
			items = append(items, Object{"name": "example", "commit": Object{"sha": strings.Repeat("a", 40)}})
		}
		return reply(items), nil
	})}
	result, err := Query(root, client, "inspect", "main")
	if err != nil || object(result)["branch"] != "main" {
		t.Fatalf("inspect: %v %v", result, err)
	}
	result, err = Query(root, client, "list", "")
	if err != nil || len(object(result)["branches"].([]any)) != 101 || calls != 3 {
		t.Fatalf("list: %v calls=%d", err, calls)
	}
}
func TestBranchMutationsAreStoppedBeforeAnyNetworkCall(t *testing.T) {
	root := contractFixture(t)
	client, err := NewClient("Kinirin/PTSIP", "token", "https://example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	client.HTTP = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("unadmitted network operation: %s", request.Method)
		return nil, nil
	})}
	for _, operation := range []func() (Object, error){func() (Object, error) { return Create(root, client, "dev/example", "dev/example", "main") }, func() (Object, error) { return Recreate(root, client, "dev/example", "dev/example", "main") }} {
		_, err := operation()
		if errorCode(t, err) != "BRANCH_MUTATION_ADMISSION_UNRESOLVED" {
			t.Fatal(err)
		}
	}
	for _, method := range []string{"POST", "PATCH", "DELETE"} {
		_, err := client.Request(method, "/repos/Kinirin/PTSIP/git/refs", Object{})
		if errorCode(t, err) != "BRANCH_MUTATION_ADMISSION_UNRESOLVED" {
			t.Fatal(err)
		}
	}
}
func TestBranchUnknownOperationsAndMalformedResponsesFailClosed(t *testing.T) {
	root := contractFixture(t)
	if _, err := Control(root, "invented", nil); errorCode(t, err) != "UNREGISTERED_BRANCH_COMMAND" {
		t.Fatal(err)
	}
	if _, err := NewClient("invalid", "", ""); errorCode(t, err) != "INVALID_REPOSITORY" {
		t.Fatal(err)
	}
	client, _ := NewClient("Kinirin/PTSIP", "", "https://example.invalid")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return reply(Object{}), nil })}
	if _, err := Query(root, client, "list", ""); errorCode(t, err) != "INVALID_GITHUB_RESPONSE" {
		t.Fatal(err)
	}
}
