package update

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

// The repository owner controls the committed plan and the execution checkout.
// Git is the existing authority boundary, not a new authentication service.
// Uncommitted plan edits and request-supplied approval files are not authority.
const DelegatedPlanPath = "developer/planning/0.3.8/go-policy-source-digest-sync.plan.json"

type delegatedPlan struct {
	PlanID                   string `json:"plan_id"`
	Status                   string `json:"status"`
	ImplementationAuthorized bool   `json:"implementation_authorized"`
	Approval struct {
		Status                   string            `json:"status"`
		PolicyMutationAuthorized bool              `json:"policy_mutation_authorized"`
		AuthorizedRequests       []json.RawMessage `json:"authorized_update_requests"`
	} `json:"approval"`
}

// DelegatedAuthorizer consumes exact requests recorded under the existing
// owner's WU-15 delegation in the committed plan. The agent may record those
// requests within its approved scope; no repeated owner approval is required.
// A plan flag or an approval_ref alone never authorizes a delta.
func DelegatedAuthorizer(root string) (Authorizer, error) {
	root, err := canonicalRepositoryRoot(root)
	if err != nil {
		return nil, err
	}
	top, err := gitApproval(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	canonical, err := canonicalRepositoryRoot(strings.TrimSpace(string(top)))
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root, canonical)
	if err != nil || relative != "." {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_REPOSITORY_MISMATCH")
	}
	raw, err := gitApproval(root, "show", "--no-textconv", "HEAD:"+DelegatedPlanPath)
	if err != nil {
		return nil, err
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_PLAN_INVALID: %w", err)
	}
	var plan delegatedPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_PLAN_INVALID: %w", err)
	}
	if plan.PlanID != "DEV-038-GO-UNIFIED-POLICY-SOURCE-DIGEST-SYNC" ||
		(plan.Status != "Approval.Scheduled_implementation" && plan.Status != "Approval.In_progress") ||
		!plan.ImplementationAuthorized || plan.Approval.Status != "APPROVED" || !plan.Approval.PolicyMutationAuthorized {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_DELEGATION_UNAVAILABLE")
	}
	requests := make(map[string]Request, len(plan.Approval.AuthorizedRequests))
	for _, raw := range plan.Approval.AuthorizedRequests {
		var request Request
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&request); err != nil || strings.TrimSpace(request.ApprovalRef) == "" ||
			!sha256Pattern.MatchString(request.BaselineSHA256) || len(request.Values) == 0 ||
			!source.Relative(request.Target) ||
			(!strings.HasPrefix(request.Target, "developer/policy/") && !strings.HasPrefix(request.Target, "src/policy/")) {
			return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_REQUEST_INVALID")
		}
		if _, exists := requests[request.ApprovalRef]; exists {
			return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_REFERENCE_AMBIGUOUS")
		}
		requests[request.ApprovalRef] = request
	}
	return func(ctx context.Context, actual Approval) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		request, exists := requests[actual.Reference]
		if !exists {
			return fmt.Errorf("POLICY_UPDATE_APPROVAL_EVIDENCE_REQUIRED")
		}
		if actual.Root != root || actual.Target != request.Target || actual.BaselineSHA256 != request.BaselineSHA256 {
			return fmt.Errorf("POLICY_UPDATE_APPROVAL_BINDING_MISMATCH")
		}
		// Derive the complete binding again from the independently recorded
		// request; the callback cannot approve different values or identities.
		expected, err := prepare(root, request)
		if err != nil {
			return err
		}
		if !sameApproval(expected.approval, actual) {
			return fmt.Errorf("POLICY_UPDATE_APPROVAL_BINDING_MISMATCH")
		}
		return nil
	}, nil
}

func sameApproval(left, right Approval) bool {
	a, err := json.Marshal(left)
	if err != nil {
		return false
	}
	b, err := json.Marshal(right)
	return err == nil && bytes.Equal(a, b)
}

func gitApproval(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"--no-replace-objects", "-C", root}, args...)...)
	raw, err := cmd.Output()
	if err != nil || len(raw) > source.MaxBytes {
		return nil, fmt.Errorf("POLICY_UPDATE_APPROVAL_PLAN_UNAVAILABLE")
	}
	return raw, nil
}
