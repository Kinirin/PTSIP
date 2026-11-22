package contracts_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/app"
	"github.com/Kinirin/PTSIP/src/ptsip/clarification"
)

func TestGoAppControlPlaneAndExistingSQLite(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "existing decisions.sqlite3")
	old, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`
		CREATE TABLE decisions (
			id TEXT PRIMARY KEY, repository TEXT NOT NULL, branch TEXT NOT NULL, subject_revision TEXT NOT NULL,
			component_id TEXT NOT NULL, request_json TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'PENDING',
			answer_json TEXT, resolution_source TEXT, resolved_by TEXT,
			application_status TEXT NOT NULL DEFAULT 'NOT_APPLIED', applied_revision TEXT, issue_number INTEGER,
			issue_url TEXT, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO decisions (id,repository,branch,subject_revision,component_id,request_json,status,answer_json,resolution_source,resolved_by)
		VALUES ('past','owner/repo','main','before','component','{}','RESOLVED',
			'{"classification":"PRODUCT","purpose":"동작","shipped":true,"runtime_required":true,"executable":true,"lifecycle_owner":"PRODUCT"}',
			'GITHUB_ISSUE','original-owner');`)
	if err != nil {
		_ = old.Close()
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenStore(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service, err := app.NewService(store, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := app.NewHandler(service, service, "agent-token", "webhook-secret", "fixture-version")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client, err := app.NewClient(server.URL, "agent-token")
	if err != nil {
		t.Fatal(err)
	}
	dispatch := func(result app.Object, err error) app.Object {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	decision := dispatch(client.Decision(ctx, app.Object{"decision_id": "past"}))
	if row := decision["decision"].(map[string]any); row["profile_path"] != "ptsip.yaml" || row["resolved_by"] != "original-owner" {
		t.Fatalf("existing decision changed: %v", row)
	}
	answer := app.Object{"classification": "PRODUCT", "purpose": "동작", "shipped": true, "runtime_required": true, "executable": true}
	if retry := dispatch(client.Resolve(ctx, app.Object{"decision_id": "past", "answer": answer})); retry["accepted"] != true {
		t.Fatalf("compatible existing decision cannot be retried: %v", retry)
	}
	row, err := store.Find(ctx, "past")
	if err != nil || row.Answer["lifecycle_owner"] != "PRODUCT" || row.ResolvedBy == nil || *row.ResolvedBy != "original-owner" {
		t.Fatalf("persisted historical answer was overwritten: %+v %v", row, err)
	}

	gate := app.Object{"id": "first", "repository": "owner/repo", "branch": "main", "subject_revision": "new",
		"profile_path": `.ptsip\profiles\main.yaml`, "component_id": "new-component", "request": app.Object{}}
	if status := dispatch(client.Gate(ctx, gate))["status"]; status != "DECISION_REQUIRED" {
		t.Fatalf("initial gate status = %v", status)
	}
	gate["id"] = "second"
	dispatch(client.Gate(ctx, gate))
	if status := dispatch(client.Decision(ctx, app.Object{"decision_id": "first"}))["status"]; status != "STALE" {
		t.Fatalf("superseded decision status = %v", status)
	}
	dispatch(client.Resolve(ctx, app.Object{"decision_id": "second", "answer": answer}))
	if result := dispatch(client.Application(ctx, app.Object{"decision_id": "second", "status": "PROPOSAL_APPROVED"})); result["status"] != "PROPOSAL_APPROVED" {
		t.Fatalf("application outcome mismatch: %v", result)
	}
	gate["repository"] = "other/repo"
	if _, err := client.Gate(ctx, gate); err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("cross-repository decision id accepted: %v", err)
	}

	body := `{"action":"created"}`
	post := func(route, header, value, body string, status int) {
		t.Helper()
		request, err := http.NewRequest(http.MethodPost, server.URL+route, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set(header, value)
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != status {
			t.Fatalf("POST %s status = %d, want %d", route, response.StatusCode, status)
		}
	}
	mac := hmac.New(sha256.New, []byte("webhook-secret"))
	_, _ = mac.Write([]byte(body))
	post("/github/webhook", "X-Hub-Signature-256", "sha256=wrong", body, http.StatusUnauthorized)
	post("/github/webhook", "X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)), body, http.StatusOK)
	post("/v1/decision", "Authorization", "Bearer wrong", body, http.StatusUnauthorized)
	post("/v1/decision", "Authorization", "Bearer agent-token", `{"decision_id":"first","decision_id":"second"}`, http.StatusBadRequest)
}

type authorityFixture struct {
	head   string
	record app.Object
	race   bool
	writes int
}

func (f *authorityFixture) ReadAuthority(_ context.Context, _ string) (string, app.Object, error) {
	return f.head, f.record, nil
}

func (f *authorityFixture) WriteAuthority(_ context.Context, _ string, input app.Object, before, _ string) (string, error) {
	if f.head != before || f.race {
		f.race = false
		return "", app.ErrAuthorityConflict
	}
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	var record app.Object
	if err := json.Unmarshal(data, &record); err != nil {
		return "", err
	}
	f.head, f.record, f.writes = f.head+"1", record, f.writes+1
	return f.head, nil
}

func TestGoAppAuthorityAndMissingProfileWriter(t *testing.T) {
	ctx := context.Background()
	store := &authorityFixture{}
	client, err := app.NewCoordinator("owner/repository", store)
	if err != nil {
		t.Fatal(err)
	}
	scope := app.Object{"repository": "owner/repository", "request": app.Object{"include": []any{"src/run", "./src/run"}}}
	peek, err := client.Peek(ctx, scope)
	if err != nil || peek["status"] != "NO_AUTHORITY_DECISION" || store.writes != 0 {
		t.Fatalf("authority lookup should not initialize a branch: %v, %v", peek, err)
	}
	scope["id"], scope["component_id"], scope["branch"], scope["subject_revision"] = "clarification", "run", "main", "source"
	if _, err := client.Gate(ctx, scope); err == nil || store.writes != 0 {
		t.Fatalf("missing authority accepted a write: %v", err)
	}
	store.head, store.race = "head", true
	gate, err := client.Gate(ctx, scope)
	if err != nil || gate["status"] != "DECISION_REQUIRED" || store.writes != 1 {
		t.Fatalf("authority gate did not recover from contention: %v, %v", gate, err)
	}
	id := store.record["id"]
	answer := app.Object{"classification": "PRODUCT", "purpose": "Run", "shipped": true, "runtime_required": true, "executable": true}
	resolved, err := client.Resolve(ctx, app.Object{"decision_id": id, "answer": answer})
	if err != nil || resolved["accepted"] != true || store.writes != 2 {
		t.Fatalf("authority resolution failed: %v, %v", resolved, err)
	}
	replay, err := client.Resolve(ctx, app.Object{"decision_id": id, "answer": answer})
	if err != nil || replay["accepted"] != false || store.writes != 2 {
		t.Fatalf("winning authority record was overwritten: %v, %v", replay, err)
	}
	input := app.ReconciliationInput{Repository: "owner/repository", Branch: "main", Revision: "source", ProfilePath: "ptsip.yaml",
		Scopes: []app.ReconciliationScope{{ComponentID: "run", Request: scope["request"].(app.Object)}}}
	result, err := client.Reconcile(ctx, input, nil)
	if err != nil || result["status"] != "AUTHORITY_PROFILE_CONFLICT" || store.writes != 2 {
		t.Fatalf("decision granted an implicit Profile write: %v, %v", result, err)
	}
	failed := &snapshotFixture{}
	result, err = client.Reconcile(ctx, input, failed)
	if err != nil || result["status"] != "STALE_EVIDENCE" || failed.writes != 0 {
		t.Fatalf("stale snapshot allowed reconciliation: %v, %v", result, err)
	}
}

type snapshotFixture struct{ checks, writes int }

func (s *snapshotFixture) VerifySnapshot(context.Context) error {
	s.checks++
	if s.checks > 1 {
		return errors.New("reference changed after preparation")
	}
	return nil
}

func (s *snapshotFixture) PrepareProfile(context.Context, string, app.ReconciliationScope, app.Object) (app.LocalProfileProjection, error) {
	return app.LocalProfileProjection{Changed: true, Apply: func(context.Context) error {
		s.writes++
		return nil
	}}, nil
}

func (s *snapshotFixture) RefreshSnapshot(context.Context) error { return nil }

type appGitHubFixture struct{}

func (appGitHubFixture) RepositoryInstallation(context.Context, string) (int64, error) {
	return 1, nil
}
func (appGitHubFixture) CreateIssue(context.Context, string, int64, string, string) (app.Issue, error) {
	return app.Issue{Number: 2, URL: "https://example.test/issues/2"}, nil
}
func (appGitHubFixture) UpdateIssue(context.Context, string, int64, int64, string) error {
	return nil
}
func (appGitHubFixture) Comment(context.Context, string, int64, int64, string) error {
	return nil
}
func (appGitHubFixture) Permission(context.Context, string, int64, string) (string, error) {
	return "write", nil
}
func (appGitHubFixture) BranchHead(context.Context, string, int64, string) (string, error) {
	return "source", nil
}

func TestGoAppRejectsReboundWebhookScope(t *testing.T) {
	ctx := context.Background()
	store, err := app.OpenStore(ctx, filepath.Join(t.TempDir(), "webhook.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	applied := false
	service, err := app.NewService(store, appGitHubFixture{}, func(ctx context.Context, record app.Record, _ clarification.Answer) (app.PreparedApplication, error) {
		// Simulate a gate on a second process that shares this decision database.
		_, _, err := store.Gate(ctx, app.GateRequest{ID: record.ID, Repository: record.Repository,
			Branch: record.Branch, SubjectRevision: record.SubjectRevision, ProfilePath: record.ProfilePath,
			ComponentID: record.ComponentID, Request: app.Object{"include": []any{"src/moved"}}})
		if err != nil {
			return nil, err
		}
		return func(context.Context, string) (string, error) {
			applied = true
			return "updated", nil
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Gate(ctx, app.Object{"id": "bound", "repository": "owner/repo", "branch": "main",
		"subject_revision": "source", "component_id": "component", "request": app.Object{"include": []any{"src/original"}},
		"issue": app.Object{"title": "scope"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.HandleWebhook(ctx, "issue_comment", app.Object{"action": "created",
		"issue": app.Object{"number": json.Number("2")}, "repository": app.Object{"full_name": "owner/repo"},
		"sender": app.Object{"login": "owner"}, "comment": app.Object{"body": "format: ptsip-clarification-answer/v2\ndecision:\n  classification: PRODUCT\n  purpose: run\n  shipped: yes\n  runtime_required: yes\n  executable: yes\n"}})
	if err != nil || result["status"] != "STALE" || applied {
		t.Fatalf("changed request allowed Profile application: %v, %v, applied=%v", result, err, applied)
	}
}
