package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

var ErrAuthorityConflict = errors.New("decision authority revision changed")
var decisionID = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var repositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// AuthorityStore binds reads and compare-and-set writes to one admitted
// repository authority. Reads never initialize an external authority.
type AuthorityStore interface {
	ReadAuthority(ctx context.Context, path string) (head string, record Object, err error)
	WriteAuthority(ctx context.Context, path string, record Object, expectedHead, message string) (string, error)
}

type Coordinator struct {
	repository string
	store      AuthorityStore
}

func NewCoordinator(repository string, store AuthorityStore) (*Coordinator, error) {
	if !repositoryName.MatchString(repository) || store == nil {
		return nil, badInput("repository and decision authority store are required")
	}
	return &Coordinator{repository, store}, nil
}

func profileFor(payload Object) (string, error) {
	if payload["profile_path"] == nil {
		return "ptsip.yaml", nil
	}
	return profilePath(text(payload, "profile_path"))
}

func scopeID(repo, profile string, request Object) (string, error) {
	var selectors []string
	include, _ := request["include"].([]any)
	for _, item := range include {
		raw, ok := item.(string)
		if !ok {
			return "", badInput("include selectors must be strings")
		}
		value := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
		for strings.HasPrefix(value, "./") {
			value = value[2:]
		}
		value = strings.Trim(value, "/")
		if value != "" {
			selectors = append(selectors, value)
		}
	}
	if len(selectors) == 0 {
		return "", badInput("GitHub-coordinated decision request requires include selectors")
	}
	slices.Sort(selectors)
	selectors = slices.Compact(selectors)
	identity := repo + "\x00" + strings.Join(selectors, "\x00")
	if profile != "ptsip.yaml" {
		identity += "\x00profile\x00" + profile
	}
	hash := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("gdec-%x", hash[:10]), nil
}

func (c *Coordinator) scope(payload Object) (string, string, Object, error) {
	if text(payload, "repository") != c.repository {
		return "", "", nil, badInput("decision repository does not match GitHub authority repository")
	}
	request := object(payload, "request")
	if request == nil {
		return "", "", nil, badInput("decision request must be an object")
	}
	profile, err := profileFor(payload)
	if err != nil {
		return "", "", nil, err
	}
	id, err := scopeID(c.repository, profile, request)
	return id, profile, request, err
}

func authorityResult(head string, record Object, status string) Object {
	if status == "" {
		switch text(record, "status") {
		case "PENDING":
			status = "DECISION_REQUIRED"
		case "RESOLVED":
			status = "RESOLVED_APPLICATION_REQUIRED"
			if text(object(record, "request"), "origin") == "EXPLICIT_PROPOSED_COMPONENT" {
				status = "RESOLVED"
			}
		default:
			status = text(record, "status")
		}
	}
	var revision any
	if head != "" {
		revision = head
	}
	return Object{"backend": "GITHUB", "status": status, "authority_revision": revision, "decision": record}
}

func (c *Coordinator) verifyScope(record Object, profile string) bool {
	recordProfile, err := profileFor(record)
	return err == nil && text(record, "repository") == c.repository && recordProfile == profile
}

func (c *Coordinator) Peek(ctx context.Context, payload Object) (Object, error) {
	id, profile, _, err := c.scope(payload)
	if err != nil {
		return nil, err
	}
	head, record, err := c.store.ReadAuthority(ctx, "decisions/"+id+".json")
	if err != nil {
		return nil, err
	}
	status := ""
	if record == nil {
		status = "NO_AUTHORITY_DECISION"
	} else if !c.verifyScope(record, profile) || text(record, "id") != id {
		status = "CONFLICT"
	}
	result := authorityResult(head, record, status)
	result["decision_id"] = id
	return result, nil
}

func (c *Coordinator) Gate(ctx context.Context, payload Object) (Object, error) {
	id, profile, request, err := c.scope(payload)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"id", "branch", "subject_revision", "component_id"} {
		if text(payload, name) == "" {
			return nil, badInput("gate requires " + name)
		}
	}
	path := "decisions/" + id + ".json"
	for range 5 {
		head, record, err := c.store.ReadAuthority(ctx, path)
		if err != nil {
			return nil, err
		}
		if head == "" {
			return nil, fmt.Errorf("GitHub decision authority has no admitted head")
		}
		if record != nil {
			status := ""
			if !c.verifyScope(record, profile) || text(record, "id") != id {
				status = "CONFLICT"
			}
			return authorityResult(head, record, status), nil
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		record = Object{"id": id, "clarification_id": text(payload, "id"), "repository": c.repository,
			"branch": text(payload, "branch"), "subject_revision": text(payload, "subject_revision"), "profile_path": profile,
			"component_id": text(payload, "component_id"), "request": request, "status": "PENDING", "answer": nil,
			"resolution_source": nil, "resolved_by": nil, "issue_number": nil, "issue_url": nil, "created_at": now, "updated_at": now}
		next, err := c.store.WriteAuthority(ctx, path, record, head, "ptsip: register decision "+id)
		if errors.Is(err, ErrAuthorityConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return authorityResult(next, record, "DECISION_REQUIRED"), nil
	}
	return nil, ErrAuthorityConflict
}

func (c *Coordinator) read(ctx context.Context, id string) (string, Object, error) {
	if !decisionID.MatchString(id) || id == "." || id == ".." {
		return "", nil, badInput("invalid GitHub decision id")
	}
	head, record, err := c.store.ReadAuthority(ctx, "decisions/"+id+".json")
	if err != nil {
		return "", nil, err
	}
	if head == "" || record == nil || text(record, "id") != id || text(record, "repository") != c.repository {
		return "", nil, badInput("GitHub-coordinated decision does not exist for this repository")
	}
	return head, record, nil
}

func (c *Coordinator) Decision(ctx context.Context, payload Object) (Object, error) {
	head, record, err := c.read(ctx, text(payload, "decision_id"))
	if err != nil {
		return nil, err
	}
	return authorityResult(head, record, ""), nil
}

func (c *Coordinator) Resolve(ctx context.Context, payload Object) (Object, error) {
	id := text(payload, "decision_id")
	answer, err := canonicalAnswer(object(payload, "answer"))
	if err != nil {
		return nil, err
	}
	validation := answer.Validate()
	if !validation.Valid {
		return Object{"backend": "GITHUB", "status": "CONFLICT", "accepted": false, "validation": validation}, nil
	}
	source := text(payload, "source")
	if source == "" {
		source = "AGENT_CHAT"
	}
	if source != "AGENT_CHAT" && source != "PROJECT_ADOPTION" {
		return nil, badInput("unsupported GitHub authority resolution source")
	}
	actor := text(payload, "actor")
	if actor == "" {
		actor = "coding-agent-session"
	}
	for range 5 {
		head, record, err := c.read(ctx, id)
		if err != nil {
			return nil, err
		}
		status := text(record, "status")
		if status != "PENDING" {
			if status == "RESOLVED" {
				status = "ALREADY_RESOLVED"
			} else if status == "" {
				status = "DECISION_ERROR"
			}
			result := authorityResult(head, record, status)
			result["accepted"], result["validation"] = false, validation
			return result, nil
		}
		resolved := make(Object, len(record))
		for key, value := range record {
			resolved[key] = value
		}
		resolved["status"], resolved["answer"], resolved["resolution_source"], resolved["resolved_by"] = "RESOLVED", answer, source, actor
		resolved["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		next, err := c.store.WriteAuthority(ctx, "decisions/"+id+".json", resolved, head, "ptsip: resolve decision "+id)
		if errors.Is(err, ErrAuthorityConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result := authorityResult(next, resolved, "RESOLVED")
		result["accepted"], result["validation"] = true, validation
		return result, nil
	}
	return nil, ErrAuthorityConflict
}

func (c *Coordinator) Application(_ context.Context, payload Object) (Object, error) {
	id, status := text(payload, "decision_id"), text(payload, "status")
	if !decisionID.MatchString(id) || id == "." || id == ".." {
		return nil, badInput("invalid GitHub decision id")
	}
	switch status {
	case "LOCAL_APPLIED", "FAILED", "STALE", "PROPOSAL_APPROVED":
	default:
		return nil, badInput("unsupported agent application status")
	}
	var profile, revision any
	if payload["profile_path"] != nil {
		path, err := profileFor(payload)
		if err != nil {
			return nil, err
		}
		profile = path
	}
	if current := text(payload, "applied_revision"); current != "" {
		revision = current
	}
	return Object{"backend": "GITHUB", "scope": "LOCAL_PROJECTION", "status": status, "decision_id": id,
		"profile_path": profile, "applied_revision": revision}, nil
}
