package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Kinirin/PTSIP/src/ptsip/clarification"
)

type Issue struct {
	Number int64
	URL    string
}

type GitHub interface {
	RepositoryInstallation(context.Context, string) (int64, error)
	CreateIssue(context.Context, string, int64, string, string) (Issue, error)
	UpdateIssue(context.Context, string, int64, int64, string) error
	Comment(context.Context, string, int64, int64, string) error
	Permission(context.Context, string, int64, string) (string, error)
	BranchHead(context.Context, string, int64, string) (string, error)
}

// PreparedApplication commits only through the calling owner's approved write
// implementation. It must verify the expected parent before applying.
type PreparedApplication func(context.Context, string) (string, error)
type PrepareApplication func(context.Context, Record, clarification.Answer) (PreparedApplication, error)

type Service struct {
	store   *Store
	github  GitHub
	prepare PrepareApplication
	gate    sync.Mutex
}

func NewService(store *Store, github GitHub, prepare PrepareApplication) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("decision store is required")
	}
	return &Service{store: store, github: github, prepare: prepare}, nil
}

func text(payload Object, key string) string {
	value, _ := payload[key].(string)
	return value
}

func object(payload Object, key string) Object {
	value, _ := payload[key].(map[string]any)
	return value
}

func positiveID(payload Object, key string) int64 {
	value, ok := payload[key].(json.Number)
	if !ok {
		return 0
	}
	id, err := value.Int64()
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func canonicalAnswer(payload Object) (clarification.Answer, error) {
	if len(payload) != 5 {
		return clarification.Answer{}, badInput("canonical v2 answer must have exactly five fields")
	}
	answer := clarification.Answer{Classification: strings.ToUpper(strings.TrimSpace(text(payload, "classification"))), Purpose: strings.TrimSpace(text(payload, "purpose"))}
	if answer.Classification == "" || answer.Purpose == "" {
		return clarification.Answer{}, badInput("classification and purpose must be non-empty strings")
	}
	for _, entry := range []struct {
		key string
		dst *bool
	}{{"shipped", &answer.Shipped}, {"runtime_required", &answer.RuntimeRequired}, {"executable", &answer.Executable}} {
		value, ok := payload[entry.key].(bool)
		if !ok {
			return clarification.Answer{}, badInput(entry.key + " must be a boolean")
		}
		*entry.dst = value
	}
	return answer, nil
}

func storedAnswer(payload Object) (clarification.Answer, error) {
	if len(payload) != 6 {
		return canonicalAnswer(payload)
	}
	owner := strings.ToUpper(strings.TrimSpace(text(payload, "lifecycle_owner")))
	canonical := make(Object, 5)
	for key, item := range payload {
		if key != "lifecycle_owner" {
			canonical[key] = item
		}
	}
	answer, err := canonicalAnswer(canonical)
	if err != nil {
		return clarification.Answer{}, err
	}
	wantOwner := answer.Classification
	if wantOwner == "NEUTRAL_CONTRACT" {
		wantOwner = "INDEPENDENT"
	}
	if owner != wantOwner || !answer.Validate().Valid {
		return clarification.Answer{}, badInput("legacy lifecycle_owner conflicts with the clarification")
	}
	return answer, nil
}

func workflowStatus(record Record) string {
	switch {
	case record.Status == "PENDING":
		return "DECISION_REQUIRED"
	case record.Status == "RESOLVED" && record.ApplicationStatus != "APPLIED" &&
		record.ApplicationStatus != "LOCAL_APPLIED" && record.ApplicationStatus != "PROPOSAL_APPROVED":
		return "RESOLVED_APPLICATION_REQUIRED"
	default:
		return record.Status
	}
}

func (s *Service) installation(ctx context.Context, repo string) (int64, error) {
	id, err := s.store.Installation(ctx, repo)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}
	id, err = s.github.RepositoryInstallation(ctx, repo)
	if err != nil {
		return 0, err
	}
	return id, s.store.SetInstallation(ctx, repo, id)
}

func (s *Service) Gate(ctx context.Context, payload Object) (Object, error) {
	profile := "ptsip.yaml"
	if payload["profile_path"] != nil {
		profile = text(payload, "profile_path")
	}
	issue := object(payload, "issue")
	if s.github != nil && issue == nil {
		return nil, badInput("issue must be an object")
	}
	request := GateRequest{ID: text(payload, "id"), Repository: text(payload, "repository"),
		Branch: text(payload, "branch"), SubjectRevision: text(payload, "subject_revision"), ProfilePath: profile,
		ComponentID: text(payload, "component_id"), Request: object(payload, "request")}
	s.gate.Lock()
	defer s.gate.Unlock()
	record, stale, err := s.store.Gate(ctx, request)
	if err != nil {
		return nil, err
	}
	if s.github == nil {
		return Object{"status": workflowStatus(record), "decision": record}, nil
	}
	installation, err := s.installation(ctx, record.Repository)
	if err != nil {
		return nil, err
	}
	for _, old := range stale {
		if old.IssueNumber != nil {
			s.closeIssue(ctx, old, installation, "PTSIP marked this clarification as stale because a newer component decision request replaced it.")
		}
	}
	if record.Status == "PENDING" {
		if record.IssueNumber == nil {
			title := text(issue, "title")
			if title == "" {
				title = "[PTSIP clarification] " + record.ComponentID
			}
			created, err := s.github.CreateIssue(ctx, record.Repository, installation, title, text(issue, "body"))
			if err != nil {
				return nil, err
			}
			record, err = s.store.AttachIssue(ctx, record.ID, created.Number, created.URL)
			if err != nil {
				return nil, err
			}
		} else {
			_ = s.github.UpdateIssue(ctx, record.Repository, installation, *record.IssueNumber, "open")
		}
	}
	return Object{"status": workflowStatus(record), "decision": record}, nil
}

func (s *Service) Decision(ctx context.Context, payload Object) (Object, error) {
	record, err := s.find(ctx, text(payload, "decision_id"))
	if err != nil {
		return nil, err
	}
	return Object{"status": workflowStatus(record), "decision": record}, nil
}

func (s *Service) find(ctx context.Context, id string) (Record, error) {
	if id == "" {
		return Record{}, badInput("decision_id is required")
	}
	record, err := s.store.Find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, badInput("decision not found")
	}
	return record, err
}

func (s *Service) Resolve(ctx context.Context, payload Object) (Object, error) {
	id := text(payload, "decision_id")
	if id == "" {
		return nil, badInput("decision_id is required")
	}
	answer, err := canonicalAnswer(object(payload, "answer"))
	if err != nil {
		return nil, err
	}
	validation := answer.Validate()
	if !validation.Valid {
		return Object{"status": "CONFLICT", "validation": validation}, nil
	}
	actor := text(payload, "actor")
	if actor == "" {
		actor = "coding-agent-session"
	}
	record, accepted, err := s.store.Resolve(ctx, id, answer, "AGENT_CHAT", actor)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, badInput("decision not found")
	}
	if err != nil {
		return nil, err
	}
	status := "ALREADY_RESOLVED"
	if accepted {
		status = "RESOLVED"
	}
	return Object{"status": status, "accepted": accepted, "decision": record, "validation": validation}, nil
}

func (s *Service) Application(ctx context.Context, payload Object) (Object, error) {
	status := text(payload, "status")
	switch status {
	case "LOCAL_APPLIED", "PROPOSAL_APPROVED", "FAILED", "STALE":
	default:
		return nil, badInput("unsupported agent application status")
	}
	record, err := s.find(ctx, text(payload, "decision_id"))
	if err != nil {
		return nil, err
	}
	if record.Status != "RESOLVED" {
		return nil, badInput("application state can be changed only for a resolved decision")
	}
	var revision *string
	if current := text(payload, "applied_revision"); current != "" {
		revision = &current
	}
	record, err = s.store.Apply(ctx, record.ID, status, revision)
	if err != nil {
		return nil, err
	}
	if s.github != nil && record.IssueNumber != nil && (status == "LOCAL_APPLIED" || status == "PROPOSAL_APPROVED") {
		if installation, err := s.installation(ctx, record.Repository); err == nil {
			message := "The decision was applied by the active coding-agent workflow."
			if status == "PROPOSAL_APPROVED" {
				message = "The proposed component is approved for follow-up and has no active declaration."
			}
			s.closeIssue(ctx, record, installation, message)
		}
	}
	return Object{"status": status, "decision": record}, nil
}

func (s *Service) closeIssue(ctx context.Context, record Record, installation int64, message string) {
	if record.IssueNumber != nil {
		_ = s.github.Comment(ctx, record.Repository, installation, *record.IssueNumber, message)
		_ = s.github.UpdateIssue(ctx, record.Repository, installation, *record.IssueNumber, "closed")
	}
}

func (s *Service) HandleWebhook(ctx context.Context, event string, payload Object) (Object, error) {
	if s.github == nil {
		return Object{"status": "IGNORED"}, nil
	}
	id := positiveID(object(payload, "installation"), "id")
	repository := text(object(payload, "repository"), "full_name")
	if id > 0 {
		if repository != "" {
			if err := s.store.SetInstallation(ctx, repository, id); err != nil {
				return nil, err
			}
		}
		for _, key := range []string{"repositories", "repositories_added"} {
			repositories, _ := payload[key].([]any)
			for _, item := range repositories {
				metadata, _ := item.(map[string]any)
				if name := text(metadata, "full_name"); name != "" {
					if err := s.store.SetInstallation(ctx, name, id); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if event != "issue_comment" {
		return Object{"status": "ACCEPTED"}, nil
	}
	number := positiveID(object(payload, "issue"), "number")
	actor := text(object(payload, "sender"), "login")
	if text(payload, "action") != "created" || repository == "" || number == 0 || actor == "" {
		return Object{"status": "IGNORED"}, nil
	}
	return s.issueComment(ctx, repository, number, actor, text(object(payload, "comment"), "body"))
}

func (s *Service) issueComment(ctx context.Context, repository string, number int64, actor, body string) (Object, error) {
	s.gate.Lock()
	defer s.gate.Unlock()
	record, err := s.store.ByIssue(ctx, repository, number)
	if errors.Is(err, sql.ErrNoRows) {
		return Object{"status": "IGNORED_NOT_PTSIP"}, nil
	}
	if err != nil {
		return nil, err
	}
	if record.Status != "PENDING" {
		return Object{"status": "IGNORED_TERMINAL_DECISION", "decision": record}, nil
	}
	installation, err := s.installation(ctx, repository)
	if err != nil {
		return Object{"status": "IGNORED_NO_INSTALLATION"}, nil
	}
	permission, err := s.github.Permission(ctx, repository, installation, actor)
	if err != nil || (permission != "admin" && permission != "maintain" && permission != "write") {
		return Object{"status": "IGNORED_UNAUTHORIZED"}, nil
	}
	answer, err := clarification.ParseAnswer(body)
	if err != nil {
		answer, err = clarification.ParseHistoricalAnswer(body)
		if err != nil {
			return Object{"status": "IGNORED_UNSTRUCTURED", "error": err.Error()}, nil
		}
	}
	if validation := answer.Validate(); !validation.Valid {
		_ = s.github.Comment(ctx, repository, installation, number, "PTSIP could not accept this decision:\n\n- "+strings.Join(validation.Errors, "\n- "))
		return Object{"status": "CONFLICT", "validation": validation}, nil
	}
	if s.prepare == nil {
		return Object{"status": "CONFLICT", "error": "the approved Profile application backend is unavailable"}, nil
	}
	apply, err := s.prepare(ctx, record, answer)
	if err != nil || apply == nil {
		return Object{"status": "CONFLICT", "error": "the approved Profile application could not be prepared"}, nil
	}
	record, accepted, err := s.store.resolve(ctx, record.ID, answer, "GITHUB_ISSUE", actor, &record)
	if err != nil {
		return nil, err
	}
	if !accepted {
		if record.Status == "PENDING" {
			return Object{"status": "STALE", "decision": record}, nil
		}
		return Object{"status": "IGNORED_TERMINAL_DECISION", "decision": record}, nil
	}
	head, err := s.github.BranchHead(ctx, repository, installation, record.Branch)
	status := "FAILED"
	var revision *string
	if err == nil && head != record.SubjectRevision {
		status = "STALE"
	} else if err == nil {
		if applied, applyErr := apply(ctx, record.SubjectRevision); applyErr == nil && applied != "" {
			status, revision = "APPLIED", &applied
		} else if refreshed, refreshErr := s.github.BranchHead(ctx, repository, installation, record.Branch); refreshErr == nil && refreshed != record.SubjectRevision {
			status = "STALE"
		}
	}
	record, err = s.store.Apply(ctx, record.ID, status, revision)
	if err != nil {
		return nil, err
	}
	s.closeIssue(ctx, record, installation, "PTSIP decision accepted. Application status: "+status+".")
	return Object{"status": "RESOLVED", "decision": record}, nil
}
