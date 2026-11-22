package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/clarification"
	_ "modernc.org/sqlite"
)

type Record struct {
	ID                string  `json:"id"`
	Repository        string  `json:"repository"`
	Branch            string  `json:"branch"`
	SubjectRevision   string  `json:"subject_revision"`
	ProfilePath       string  `json:"profile_path"`
	ComponentID       string  `json:"component_id"`
	Request           Object  `json:"request"`
	Status            string  `json:"status"`
	Answer            Object  `json:"answer"`
	ResolutionSource  *string `json:"resolution_source"`
	ResolvedBy        *string `json:"resolved_by"`
	ApplicationStatus string  `json:"application_status"`
	AppliedRevision   *string `json:"applied_revision"`
	IssueNumber       *int64  `json:"issue_number"`
	IssueURL          *string `json:"issue_url"`
}

type GateRequest struct {
	ID, Repository, Branch, SubjectRevision, ProfilePath, ComponentID string
	Request                                                           Object
}

type Store struct{ db *sql.DB }

const decisionColumns = "id, repository, branch, subject_revision, profile_path, component_id, request_json, status, answer_json, resolution_source, resolved_by, application_status, applied_revision, issue_number, issue_url"

// OpenStore opens existing PTSIP SQLite files in place and retains their rows.
func OpenStore(ctx context.Context, filename string) (*Store, error) {
	if filename == "" {
		return nil, badInput("decision database path is required")
	}
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(absolute)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	dsn := (&url.URL{Scheme: "file", Path: uriPath, RawQuery: "_pragma=busy_timeout(30000)&_txlock=immediate"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{db}
	if err := store.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) initialize(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS installations (repository TEXT PRIMARY KEY, installation_id INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS decisions (
		    id TEXT PRIMARY KEY, repository TEXT NOT NULL, branch TEXT NOT NULL, subject_revision TEXT NOT NULL,
		    profile_path TEXT NOT NULL DEFAULT 'ptsip.yaml', component_id TEXT NOT NULL, request_json TEXT NOT NULL,
		    status TEXT NOT NULL DEFAULT 'PENDING', answer_json TEXT, resolution_source TEXT, resolved_by TEXT,
		    application_status TEXT NOT NULL DEFAULT 'NOT_APPLIED', applied_revision TEXT, issue_number INTEGER,
		    issue_url TEXT, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_decision_issue ON decisions(repository, issue_number);
		CREATE INDEX IF NOT EXISTS idx_decision_component ON decisions(repository, component_id, status);`)
	if err != nil {
		return err
	}
	var hasProfile int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('decisions') WHERE name = 'profile_path'").Scan(&hasProfile); err != nil {
		return err
	}
	if hasProfile == 0 {
		if _, err := tx.ExecContext(ctx, "ALTER TABLE decisions ADD COLUMN profile_path TEXT NOT NULL DEFAULT 'ptsip.yaml'"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type rowScanner interface{ Scan(...any) error }

func scanRecord(row rowScanner) (Record, error) {
	var record Record
	var request string
	var answer *string
	if err := row.Scan(&record.ID, &record.Repository, &record.Branch, &record.SubjectRevision, &record.ProfilePath,
		&record.ComponentID, &request, &record.Status, &answer, &record.ResolutionSource, &record.ResolvedBy,
		&record.ApplicationStatus, &record.AppliedRevision, &record.IssueNumber, &record.IssueURL); err != nil {
		return Record{}, err
	}
	var err error
	record.ProfilePath, err = profilePath(record.ProfilePath)
	if err != nil {
		return Record{}, err
	}
	record.Request, err = decodeObject([]byte(request))
	if err != nil {
		return Record{}, fmt.Errorf("decode decision %q request: %w", record.ID, err)
	}
	if answer != nil {
		record.Answer, err = decodeObject([]byte(*answer))
		if err != nil {
			return Record{}, fmt.Errorf("decode decision %q answer: %w", record.ID, err)
		}
	}
	return record, nil
}

func profilePath(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if strings.HasPrefix(raw, "/") || strings.ContainsAny(raw, ":\x00\r\n") {
		return "", badInput("profile_path must be repository-relative")
	}
	parts := make([]string, 0)
	for _, part := range strings.Split(raw, "/") {
		switch part {
		case "", ".":
		case "..":
			return "", badInput("profile_path must not escape the repository")
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", badInput("profile_path must name a file inside the repository")
	}
	return strings.Join(parts, "/"), nil
}

func (s *Store) Gate(ctx context.Context, request GateRequest) (Record, []Record, error) {
	if request.ID == "" || request.Repository == "" || request.Branch == "" || request.SubjectRevision == "" ||
		request.ComponentID == "" || request.Request == nil {
		return Record{}, nil, badInput("gate requires id, repository, branch, subject_revision, component_id and request")
	}
	profile, err := profilePath(request.ProfilePath)
	if err != nil {
		return Record{}, nil, err
	}
	raw, err := json.Marshal(request.Request)
	if err != nil {
		return Record{}, nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Record{}, nil, err
	}
	defer tx.Rollback()
	ctxScope := []any{request.Repository, request.ComponentID, request.ID}
	existing, err := findDecision(ctx, tx, request.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Record{}, nil, err
	}
	if err == nil && (existing.Repository != request.Repository || existing.ComponentID != request.ComponentID) {
		return Record{}, nil, badInput("decision id already belongs to another repository or component")
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+decisionColumns+" FROM decisions WHERE repository=? AND component_id=? AND id<>? AND status='PENDING' ORDER BY id", ctxScope...)
	if err != nil {
		return Record{}, nil, err
	}
	var stale []Record
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			_ = rows.Close()
			return Record{}, nil, err
		}
		stale = append(stale, record)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return Record{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE decisions SET status='STALE', updated_at=CURRENT_TIMESTAMP WHERE repository=? AND component_id=? AND id<>? AND status='PENDING'", ctxScope...); err != nil {
		return Record{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO decisions(id, repository, branch, subject_revision, profile_path, component_id, request_json)
		VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET branch=excluded.branch,
		subject_revision=excluded.subject_revision, profile_path=excluded.profile_path, request_json=excluded.request_json,
		updated_at=CURRENT_TIMESTAMP WHERE status='PENDING' OR (status='RESOLVED' AND application_status IN ('NOT_APPLIED','FAILED','STALE'))`,
		request.ID, request.Repository, request.Branch, request.SubjectRevision, profile, request.ComponentID, string(raw)); err != nil {
		return Record{}, nil, err
	}
	record, err := findDecision(ctx, tx, request.ID)
	if err != nil {
		return Record{}, nil, err
	}
	return record, stale, tx.Commit()
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func findDecision(ctx context.Context, db rowQuerier, id string) (Record, error) {
	return scanRecord(db.QueryRowContext(ctx, "SELECT "+decisionColumns+" FROM decisions WHERE id=?", id))
}

func (s *Store) Find(ctx context.Context, id string) (Record, error) {
	return findDecision(ctx, s.db, id)
}

func (s *Store) ByIssue(ctx context.Context, repo string, number int64) (Record, error) {
	return scanRecord(s.db.QueryRowContext(ctx, "SELECT "+decisionColumns+
		" FROM decisions WHERE repository=? AND issue_number=? ORDER BY created_at DESC, id LIMIT 1", repo, number))
}

func (s *Store) SetInstallation(ctx context.Context, repo string, id int64) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO installations VALUES (?, ?) ON CONFLICT(repository) DO UPDATE SET installation_id=excluded.installation_id", repo, id)
	return err
}

func (s *Store) Installation(ctx context.Context, repo string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, "SELECT installation_id FROM installations WHERE repository=?", repo).Scan(&id)
	return id, err
}

func (s *Store) AttachIssue(ctx context.Context, id string, number int64, url string) (Record, error) {
	return scanRecord(s.db.QueryRowContext(ctx, "UPDATE decisions SET issue_number=?, issue_url=?, updated_at=CURRENT_TIMESTAMP"+
		" WHERE id=? AND status='PENDING' AND issue_number IS NULL RETURNING "+decisionColumns, number, url, id))
}

func (s *Store) Resolve(ctx context.Context, id string, answer clarification.Answer, source, actor string) (Record, bool, error) {
	return s.resolve(ctx, id, answer, source, actor, nil)
}

func (s *Store) resolve(ctx context.Context, id string, answer clarification.Answer, source, actor string, observed *Record) (Record, bool, error) {
	raw, err := json.Marshal(answer)
	if err != nil {
		return Record{}, false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Record{}, false, err
	}
	defer tx.Rollback()
	if observed != nil {
		current, err := findDecision(ctx, tx, id)
		if err != nil {
			return Record{}, false, err
		}
		if current.Branch != observed.Branch || current.SubjectRevision != observed.SubjectRevision ||
			current.ProfilePath != observed.ProfilePath || !reflect.DeepEqual(current.Request, observed.Request) {
			return current, false, tx.Commit()
		}
	}
	record, err := scanRecord(tx.QueryRowContext(ctx, "UPDATE decisions SET status='RESOLVED', answer_json=?, resolution_source=?, resolved_by=?, updated_at=CURRENT_TIMESTAMP"+
		" WHERE id=? AND status='PENDING' RETURNING "+decisionColumns, string(raw), source, actor, id))
	accepted := err == nil
	if errors.Is(err, sql.ErrNoRows) {
		record, err = findDecision(ctx, tx, id)
		if err == nil && source == "AGENT_CHAT" && record.Status == "RESOLVED" &&
			(record.ApplicationStatus == "NOT_APPLIED" || record.ApplicationStatus == "FAILED" || record.ApplicationStatus == "STALE") {
			existing, parseErr := storedAnswer(record.Answer)
			accepted = parseErr == nil && existing == answer
		}
	}
	if err != nil {
		return Record{}, false, err
	}
	return record, accepted, tx.Commit()
}

func (s *Store) Apply(ctx context.Context, id, state string, revision *string) (Record, error) {
	return scanRecord(s.db.QueryRowContext(ctx, "UPDATE decisions SET application_status=?, applied_revision=?, updated_at=CURRENT_TIMESTAMP"+
		" WHERE id=? AND status='RESOLVED' RETURNING "+decisionColumns, state, revision, id))
}
