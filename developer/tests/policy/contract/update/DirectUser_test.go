package update_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/check"
	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/update"
)

func buildDeveloperCLI(t *testing.T) string {
	t.Helper()
	module, err := filepath.Abs("../../../../../.ptsip/engine/developer")
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "ptsip-developer")
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cmd := exec.Command("go", "-C", module, "build", "-mod=readonly", "-o", name, ".")
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building developer CLI: %v\n%s", err, raw)
	}
	return name
}

type cliReport struct {
	Status              string          `json:"status"`
	Problem             string          `json:"problem"`
	Approval            update.Approval `json:"approval"`
	Result              update.Result   `json:"result"`
	ExecutionAuthorized bool            `json:"execution_authorized"`
	NormativeApproval   bool            `json:"normative_approval"`
}

func runDeveloperCLI(t *testing.T, binary, root string, options ...string) (cliReport, int) {
	t.Helper()
	args := append([]string{"--validator", "policy-rules", "--operation", "update", "--root", root, "--input", "update-request.json"}, options...)
	cmd := exec.Command(binary, args...)
	raw, err := cmd.Output()
	var exit int
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			exit = failure.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	var report cliReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode CLI report (exit %d): %v\n%s", exit, err, raw)
	}
	if report.ExecutionAuthorized || report.NormativeApproval {
		t.Fatalf("policy update issued unrelated authority: %+v", report)
	}
	return report, exit
}

func marshal(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func gitFixture(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestDelegatedApprovalThroughDeveloperCLI(t *testing.T) {
	binary := buildDeveloperCLI(t)
	for _, plane := range []string{"developer", "support"} {
		t.Run(plane, func(t *testing.T) {
			root, target, raw := policyFixture(t, plane)
			req := request(target, raw, "SRC-0003", `{"message":"Delegated fixture","limit":3}`)
			put(t, root, "update-request.json", marshal(t, req))
			gitFixture(t, root, "init", "-q")
			gitFixture(t, root, "config", "user.name", "PTSIP fixture")
			gitFixture(t, root, "config", "user.email", "fixture@example.invalid")
			gitFixture(t, root, "config", "commit.gpgsign", "false")
			writePlan := func(allowed bool, requests ...update.Request) {
				put(t, root, update.DelegatedPlanPath, marshal(t, map[string]any{
					"plan_id": "DEV-038-GO-UNIFIED-POLICY-SOURCE-DIGEST-SYNC",
					"status": "Approval.Scheduled_implementation",
					"implementation_authorized": true,
					"approval": map[string]any{"status": "APPROVED", "policy_mutation_authorized": allowed,
						"authorized_update_requests": requests},
				}))
			}
			commitPlan := func() {
				gitFixture(t, root, "add", "--", update.DelegatedPlanPath)
				gitFixture(t, root, "commit", "-qm", "fixture owner approval")
			}
			requireReject := func(problem string) {
				t.Helper()
				report, code := runDeveloperCLI(t, binary, root)
				if code != 2 || !strings.Contains(report.Problem, problem) || !bytes.Equal(raw, read(t, root, target)) {
					t.Fatalf("expected %s and original bytes, got %d: %+v", problem, code, report)
				}
			}
			preview, code := runDeveloperCLI(t, binary, root, "--dry-run")
			if code != 0 || preview.Status != "PREPARED_NOT_AUTHORIZED" ||
				preview.Approval.Target != target || preview.Approval.RequestSHA256 == "" ||
				preview.Approval.ProposedSHA256 == req.BaselineSHA256 || !bytes.Equal(raw, read(t, root, target)) {
				t.Fatalf("non-mutating preview failed: %+v", preview)
			}
			requireReject("POLICY_UPDATE_APPROVAL_PLAN_UNAVAILABLE")
			writePlan(true)
			commitPlan()
			requireReject("POLICY_UPDATE_APPROVAL_EVIDENCE_REQUIRED")
			writePlan(true, req)
			// A worktree-only approval is not authority.
			requireReject("POLICY_UPDATE_APPROVAL_EVIDENCE_REQUIRED")
			writePlan(false, req)
			commitPlan()
			requireReject("POLICY_UPDATE_APPROVAL_DELEGATION_UNAVAILABLE")
			writePlan(true, req, req)
			commitPlan()
			requireReject("POLICY_UPDATE_APPROVAL_REFERENCE_AMBIGUOUS")
			writePlan(true, req)
			commitPlan()
			changed := request(target, raw, "SRC-0003", `{"message":"Different request","limit":4}`)
			put(t, root, "update-request.json", marshal(t, changed))
			requireReject("POLICY_UPDATE_APPROVAL_BINDING_MISMATCH")
			changed = req
			changed.ApprovalRef = "unrecorded"
			put(t, root, "update-request.json", marshal(t, changed))
			requireReject("POLICY_UPDATE_APPROVAL_EVIDENCE_REQUIRED")
			put(t, root, "update-request.json", marshal(t, req))

			report, code := runDeveloperCLI(t, binary, root)
			updated := read(t, root, target)
			if code != 0 || report.Status != "UPDATED" || bytes.Equal(raw, updated) ||
				report.Result.DigestAfter != preview.Approval.SourceAfter {
				t.Fatalf("valid delegated request was not applied: %+v", report)
			}
			if _, err := check.CheckUnifiedDocument(root, document(t, updated)); err != nil {
				t.Fatal(err)
			}
			report, code = runDeveloperCLI(t, binary, root)
			if code != 2 || !strings.Contains(report.Problem, "STALE_POLICY_UPDATE_BASELINE") ||
				!bytes.Equal(updated, read(t, root, target)) {
				t.Fatalf("stale baseline changed the file: %+v", report)
			}
			// Restoring the exact baseline is permitted by the same committed
			// delegation; no separate replay store is required.
			put(t, root, target, raw)
			report, code = runDeveloperCLI(t, binary, root)
			if code != 0 || report.Status != "UPDATED" || !bytes.Equal(updated, read(t, root, target)) {
				t.Fatalf("delegated retry failed: %+v", report)
			}
		})
	}
}

func TestDelegatedOperationalPolicyThroughDeveloperCLI(t *testing.T) {
	repository, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	committed := func(path string) []byte {
		t.Helper()
		cmd := exec.Command("git", "-C", repository, "show", "--no-textconv", "HEAD:"+path)
		raw, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	plan := committed(update.DelegatedPlanPath)
	var record struct {
		Approval struct {
			Requests []update.Request `json:"authorized_update_requests"`
		} `json:"approval"`
	}
	if err := json.Unmarshal(plan, &record); err != nil {
		t.Fatal(err)
	}
	var req update.Request
	for _, candidate := range record.Approval.Requests {
		if candidate.ApprovalRef == "WU15-OPERATING-VERIFICATION-DUPLICATE-RULE-CLEANUP-20261127" {
			req = candidate
			break
		}
	}
	if req.Target == "" {
		t.Fatal("recorded operational verification request missing")
	}
	// Use the exact original policy revision even after the canonical update
	// is committed; the evidence record is intentionally reproducible.
	raw := committed(req.Target)
	root, _, _ := policyFixture(t, "developer")
	put(t, root, req.Target, raw)
	put(t, root, update.DelegatedPlanPath, plan)
	put(t, root, "update-request.json", marshal(t, req))
	gitFixture(t, root, "init", "-q")
	gitFixture(t, root, "config", "user.name", "PTSIP operating verification")
	gitFixture(t, root, "config", "user.email", "fixture@example.invalid")
	gitFixture(t, root, "config", "commit.gpgsign", "false")
	gitFixture(t, root, "add", "--", update.DelegatedPlanPath)
	gitFixture(t, root, "commit", "-qm", "exact committed delegated request")
	binary := buildDeveloperCLI(t)
	preview, code := runDeveloperCLI(t, binary, root, "--dry-run")
	if code != 0 || preview.Status != "PREPARED_NOT_AUTHORIZED" {
		t.Fatalf("operating preview: %+v", preview)
	}
	report, code := runDeveloperCLI(t, binary, root)
	updated := read(t, root, req.Target)
	if code != 0 || report.Status != "UPDATED" || bytes.Equal(raw, updated) ||
		report.Result.DigestAfter != preview.Approval.SourceAfter {
		t.Fatalf("operating policy update: %+v", report)
	}
	if _, err := check.CheckUnifiedDocument(root, document(t, updated)); err != nil {
		t.Fatal(err)
	}
	t.Logf("operating policy %s: %s -> %s", req.Target, report.Result.DigestBefore, report.Result.DigestAfter)
}
