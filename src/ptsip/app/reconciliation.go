package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type ReconciliationScope struct {
	ComponentID     string
	ClarificationID string
	Request         Object
	Issue           Object
}

type ReconciliationInput struct {
	Repository, Branch, Revision, ProfilePath string
	Scopes                                    []ReconciliationScope
}

type LocalProfileProjection struct {
	Changed bool
	Apply   func(context.Context) error
}

// LocalProfileReconciler must use an admitted snapshot and an approved Profile
// write path. The control plane handles decision evidence and sequencing.
type LocalProfileReconciler interface {
	VerifySnapshot(context.Context) error
	PrepareProfile(context.Context, string, ReconciliationScope, Object) (LocalProfileProjection, error)
	RefreshSnapshot(context.Context) error
}

// Reconcile checks GitHub-authoritative decisions against one selected Profile.
// A local declaration without a remote decision is left in the selected scope.
func (c *Coordinator) Reconcile(ctx context.Context, input ReconciliationInput, local LocalProfileReconciler) (Object, error) {
	if input.Repository != c.repository || input.Branch == "" || input.Revision == "" {
		return nil, badInput("reconciliation requires the selected GitHub repository, branch and revision")
	}
	profile, err := profilePath(input.ProfilePath)
	if err != nil {
		return nil, err
	}
	rows := make([]Object, 0, len(input.Scopes))
	finish := func(status string, exitCode int) Object {
		return Object{"status": status, "backend": "GITHUB", "profile_path": profile, "repository": input.Repository,
			"subject_revision": input.Revision, "decisions": rows, "exit_code": exitCode}
	}
	var profileConflict, requiresDecision, invalidDecision, resolved bool
	for _, scope := range input.Scopes {
		query := Object{"repository": input.Repository, "profile_path": profile, "request": scope.Request}
		row, err := c.Peek(ctx, query)
		if err != nil {
			return nil, err
		}
		if row["status"] == "NO_AUTHORITY_DECISION" && scope.ClarificationID != "" {
			query["id"], query["component_id"], query["issue"] = scope.ClarificationID, scope.ComponentID, scope.Issue
			query["branch"], query["subject_revision"] = input.Branch, input.Revision
			row, err = c.Gate(ctx, query)
			if err != nil {
				return nil, err
			}
		}
		switch row["status"] {
		case "NO_AUTHORITY_DECISION":
			row["status"] = "NO_DECISION_REQUIRED"
			row["reconciliation"] = Object{"status": "LOCAL_DECLARATION_ONLY"}
		case "DECISION_REQUIRED":
			requiresDecision = true
		case "RESOLVED":
			resolved = true
		case "RESOLVED_APPLICATION_REQUIRED":
			record := object(row, "decision")
			answer := object(record, "answer")
			canonical, err := storedAnswer(answer)
			if err != nil || !canonical.Validate().Valid {
				row["status"], row["message"] = "DECISION_ERROR", "resolved authority record has no valid answer"
				invalidDecision = true
				break
			}
			if !c.verifyScope(record, profile) || local == nil {
				row["status"], row["message"] = "AUTHORITY_PROFILE_CONFLICT", "approved Profile projection is unavailable for the selected scope"
				profileConflict = true
				break
			}
			authorityRevision := text(row, "authority_revision")
			decisionID := text(record, "id")
			authorityRecord, err := json.Marshal(record)
			if err != nil || authorityRevision == "" {
				row["status"], row["message"] = "DECISION_ERROR", "resolved decision authority could not be bound to the selected Profile"
				invalidDecision = true
				break
			}
			if err := local.VerifySnapshot(ctx); err != nil {
				result := finish("STALE_EVIDENCE", 8)
				result["message"] = err.Error()
				return result, nil
			}
			prepared, err := local.PrepareProfile(ctx, profile, scope, answer)
			if err != nil {
				row["status"], row["message"] = "AUTHORITY_PROFILE_CONFLICT", err.Error()
				profileConflict = true
				break
			}
			reconciliation := Object{"status": "CONSISTENT", "component_id": scope.ComponentID, "profile_path": profile}
			if prepared.Changed {
				if prepared.Apply == nil {
					return nil, fmt.Errorf("validated Profile projection has no approved application")
				}
				if err := local.VerifySnapshot(ctx); err != nil {
					result := finish("STALE_EVIDENCE", 8)
					result["message"] = err.Error()
					return result, nil
				}
				head, current, err := c.read(ctx, decisionID)
				currentRecord, encodeErr := json.Marshal(current)
				if err != nil || encodeErr != nil || head != authorityRevision || !bytes.Equal(currentRecord, authorityRecord) {
					result := finish("STALE_EVIDENCE", 8)
					result["message"] = "decision authority changed or could not be verified before Profile application"
					return result, nil
				}
				if err := prepared.Apply(ctx); err != nil {
					result := finish("STALE_EVIDENCE", 8)
					result["message"] = err.Error()
					return result, nil
				}
				applied, err := c.Application(ctx, Object{"decision_id": decisionID, "status": "LOCAL_APPLIED",
					"profile_path": profile, "applied_revision": input.Revision})
				if err != nil {
					return nil, err
				}
				reconciliation["status"], reconciliation["application"] = "LOCAL_APPLIED", applied
				if err := local.RefreshSnapshot(ctx); err != nil {
					return nil, fmt.Errorf("Profile was applied but the repository snapshot could not be refreshed: %w", err)
				}
			}
			row["status"], row["reconciliation"] = "RESOLVED", reconciliation
			resolved = true
		default:
			invalidDecision = true
		}
		rows = append(rows, row)
	}
	switch {
	case profileConflict:
		return finish("AUTHORITY_PROFILE_CONFLICT", 8), nil
	case requiresDecision:
		return finish("DECISION_REQUIRED", 7), nil
	case invalidDecision:
		return finish("DECISION_ERROR", 8), nil
	case resolved:
		return finish("RESOLVED", 0), nil
	default:
		return finish("NO_DECISION_REQUIRED", 0), nil
	}
}
